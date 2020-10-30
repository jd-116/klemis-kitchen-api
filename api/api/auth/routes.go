package auth

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/segmentio/ksuid"

	"github.com/jd-116/klemis-kitchen-api/auth"
	"github.com/jd-116/klemis-kitchen-api/cas"
	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/util"
)

// The name of the cookie attached to the auth flow
const FlowContinuationCookieName = "FlowContinuation"

func Routes(casProvider *cas.Provider, database db.Provider, jwtManager *auth.JWTManager) *chi.Mux {
	// Try to get the domain env variable if it is set
	cookieDomain := strings.TrimSpace(os.Getenv("API_SERVER_DOMAIN"))

	// Try to see if the continuation cookies should be secure
	secureContinuationCookies := false
	if value, ok := os.LookupEnv("AUTH_SECURE_CONTINUATION"); ok {
		if strings.TrimSpace(value) == "1" {
			secureContinuationCookies = true
		}
	}

	// Create the flow continuation map
	pollInterval := 2 * time.Minute
	maxTTL := 10 * time.Minute
	flowContinuation := NewFlowContinuationMap(pollInterval, int64(maxTTL/time.Second))

	// Determine if the redirect URIs are valid using lambda
	validRedirectURIPrefixes := []string{}
	if value, ok := os.LookupEnv("AUTH_REDIRECT_URI_PREFIXES"); ok {
		value = strings.TrimSpace(value)
		if value != "" {
			validRedirectURIPrefixes = strings.Split(value, "|")
		}
	}
	isRedirectURIValid := func(uri string) bool {
		// If no prefixes supplied, then accept any
		if len(validRedirectURIPrefixes) == 0 {
			return true
		}

		// Attempt to find a matching prefix
		for _, prefix := range validRedirectURIPrefixes {
			if strings.HasPrefix(uri, prefix) {
				return true
			}
		}

		return false
	}

	router := chi.NewRouter()
	router.Get("/login", Login(casProvider, flowContinuation, cookieDomain,
		secureContinuationCookies, isRedirectURIValid, database, jwtManager))
	return router
}

// Handles the GT SSO login flow via the CAS protocol v2
func Login(casProvider *cas.Provider, flowContinuation *FlowContinuationMap,
	cookieDomain string, secureContinuationCookies bool,
	isRedirectURIValid func(string) bool,
	membershipProvider db.MembershipProvider,
	jwtManager *auth.JWTManager) http.HandlerFunc {

	// Use a closure to inject dependencies
	return func(w http.ResponseWriter, r *http.Request) {
		// First, see if this is at the return of the CAS flow,
		// which will have a "ticket" query parameter if it is
		ticket := r.URL.Query().Get("ticket")

		if ticket == "" {
			// This is the first part of the CAS flow,
			// send them to the GT SSO flow

			// Make sure the redirect URI is provided
			redirectURI := strings.TrimSpace(r.URL.Query().Get("redirect_uri"))
			if redirectURI == "" {
				util.ErrorWithCode(w, errors.New("redirect_uri is required"),
					http.StatusBadRequest)
				return
			}

			// Make sure the redirect URI is valid
			if !isRedirectURIValid(redirectURI) {
				util.ErrorWithCode(w, errors.New("redirect_uri is not valid"),
					http.StatusBadRequest)
				return
			}

			// Generate the flow continuation ID
			flowContinuationId, err := ksuid.NewRandom()
			if err != nil {
				util.Error(w, err)
			}
			flowContinuationIdStr := flowContinuationId.String()
			flowContinuation.Put(flowContinuationIdStr, redirectURI)

			// Remove the redirect URI parameter from the URL
			query := r.URL.Query()
			query.Del("redirect_uri")
			r.URL.RawQuery = query.Encode()

			// Get the URL to redirect to GT SSO
			err = casProvider.Redirect(w, r)
			if err != nil {
				util.Error(w, err)
			}

			// Include the flow continuation cookie
			ttl := 5 * time.Minute
			expire := time.Now().Add(ttl)
			cookie := http.Cookie{
				Name:     FlowContinuationCookieName,
				Value:    flowContinuationIdStr,
				Secure:   secureContinuationCookies,
				HttpOnly: true,
				Path:     "/",
				Domain:   cookieDomain,
				SameSite: http.SameSiteStrictMode,
				Expires:  expire,
			}
			log.Println(cookie.String())
			http.SetCookie(w, &cookie)
		} else {
			// First, make sure the flow came from us originally
			// by looking for the flow continuation cookie
			flowContinuationCookie, err := r.Cookie(FlowContinuationCookieName)
			if err != nil {
				util.ErrorWithCode(w, errors.New("request doesn't come at the end of authentication flow"),
					http.StatusForbidden)
				return
			}

			// Extract the original redirect URI
			redirectURI, ok := flowContinuation.Get(flowContinuationCookie.Value)
			if !ok {
				util.ErrorWithCode(w, errors.New("request doesn't come at the end of authentication flow"),
					http.StatusForbidden)
				return
			}

			// This is the second part of the CAS flow,
			// send a request to the CAS server to validate the ticket
			result, err := casProvider.ServiceValidate(r, ticket)
			if err != nil {
				util.Error(w, err)
			}

			log.Printf("handling authentication for '%s' at the end of CAS flow\n", result.User)
			username := result.User
			first_name := "Fix"
			last_name := "Me"

			// Determine whether the user is a member or not,
			// and if so, what level of access they have
			membership, err := membershipProvider.GetMembership(r.Context(), username)
			if err != nil {
				// Not a member, redirect to the redirect URI with an error status
				err = terminalRedirect(w, r, redirectURI, "failure", "1")
				if err != nil {
					util.Error(w, err)
				}

				return
			}

			// They are a member, so construct a JWT from their membership
			permissions := membership.Permissions()
			token, err := jwtManager.IssueJWT(username, permissions, first_name, last_name)
			if err != nil {
				util.Error(w, err)
				return
			}

			// Finally, redirect them to the redirect URI with the token
			err = terminalRedirect(w, r, redirectURI, "token", token)
			if err != nil {
				util.Error(w, err)
			}
		}
	}
}

// terminalRedirect is a utility function used at the end of the auth flow
// to send a single key-value pair to the original initiator
func terminalRedirect(w http.ResponseWriter, r *http.Request,
	baseUri string, key string, value string) error {

	u, err := url.ParseRequestURI(baseUri)
	if err != nil {
		return err
	}

	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return err
	}

	// Add the key-value pair to the existing query
	q.Add(key, value)
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusMovedTemporarily)
	return nil
}
