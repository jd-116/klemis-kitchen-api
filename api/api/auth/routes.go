package auth

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"

	"github.com/jd-116/klemis-kitchen-api/auth"
	"github.com/jd-116/klemis-kitchen-api/cas"
	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

// FlowContinuationCookieName is the name of the cookie attached to the auth flow
const FlowContinuationCookieName = "FlowContinuation"

// Routes creates a new Chi router with all of the routes for the auth flow
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

	// Create the flow continuation nonce map
	pollInterval := 2 * time.Minute
	maxTTL := 10 * time.Minute
	flowContinuation := NewNonceMap(pollInterval, int64(maxTTL/time.Second))

	// Create the auth code nonce map
	pollInterval = 3 * time.Minute
	maxTTL = 5 * time.Minute
	authCodes := NewNonceMap(pollInterval, int64(maxTTL/time.Second))

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

	// Try to see if the continuation cookies should be secure
	var tokenExpirationHours *int64 = nil
	if value, ok := os.LookupEnv("AUTH_JWT_TOKEN_EXPIRES_AFTER"); ok {
		valueInt, err := strconv.Atoi(value)
		if err == nil {
			valueInt64 := int64(valueInt)
			tokenExpirationHours = &valueInt64
		}
	}

	router := chi.NewRouter()

	// Public routes
	router.Group(func(r chi.Router) {
		r.Get("/login", Login(casProvider, flowContinuation, authCodes, cookieDomain,
			secureContinuationCookies, isRedirectURIValid, database, jwtManager,
			tokenExpirationHours))
		r.Post("/token-exchange", TokenExchange(authCodes, jwtManager))
	})

	// Protect the /session route and validate JWTs
	router.Group(func(r chi.Router) {
		// Seek, verify and validate JWT tokens,
		// sending appropriate status codes upon failure.
		r.Use(jwtManager.Authenticated())

		r.Get("/session", Session(jwtManager))
	})

	return router
}

// Login handles the GT SSO login flow via the CAS protocol v2
func Login(casProvider *cas.Provider, flowContinuation *NonceMap,
	authCodes *NonceMap, cookieDomain string, secureContinuationCookies bool,
	isRedirectURIValid func(string) bool, membershipProvider db.MembershipProvider,
	jwtManager *auth.JWTManager, tokenExpirationHours *int64) http.HandlerFunc {

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

			// Generate the flow continuation nonce
			flowContinuationNonce, err := flowContinuation.Provision(redirectURI)
			if err != nil {
				util.Error(w, err)
				return
			}

			// Include the flow continuation cookie
			ttl := 5 * time.Minute
			expire := time.Now().Add(ttl)
			cookie := http.Cookie{
				Name:     FlowContinuationCookieName,
				Value:    flowContinuationNonce,
				Secure:   secureContinuationCookies,
				HttpOnly: true,
				Path:     "/",
				Domain:   cookieDomain,
				// Cookie needs to be Lax so it is send when CAS redirects
				SameSite: http.SameSiteLaxMode,
				Expires:  expire,
			}
			http.SetCookie(w, &cookie)

			// Get the URL to redirect to GT SSO
			err = casProvider.Redirect(w, r)
			if err != nil {
				util.Error(w, err)
				return
			}
		} else {
			// First, make sure the flow came from us originally
			// by looking for the flow continuation cookie
			flowContinuationCookie, err := r.Cookie(FlowContinuationCookieName)
			if err != nil {
				util.ErrorWithCode(w, errors.New("request doesn't contain flow continuation nonce"),
					http.StatusForbidden)
				return
			}

			// Extract the original redirect URI from the flow continuation nonce
			redirectURIRaw, ok := flowContinuation.Use(flowContinuationCookie.Value)
			if !ok {
				util.ErrorWithCode(w, errors.New("request had unknown flow continuation nonce"),
					http.StatusForbidden)
				return
			}
			redirectURI, ok := redirectURIRaw.(string)
			if !ok {
				util.ErrorWithCode(w, errors.New("request had invalid flow continuation nonce value"),
					http.StatusForbidden)
				return
			}

			// This is the second part of the CAS flow,
			// send a request to the CAS server to validate the ticket
			identity, err := casProvider.ServiceValidate(r, ticket)
			if err != nil {
				util.Error(w, err)
				return
			}

			username := identity.Username
			firstName := identity.FirstName
			lastName := identity.LastName
			log.Printf("handling authentication for '%s' at the end of CAS flow\n", username)

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
			session := types.Session{
				Username:     username,
				FirstName:    firstName,
				LastName:     lastName,
				IssuedAt:     time.Now(),
				ExpiresAfter: tokenExpirationHours,
			}
			token := jwtManager.IssueJWT(session, permissions)

			// Create the code nonce that can be exchanged for the JWT later
			authCode, err := authCodes.Provision(token)
			if err != nil {
				util.Error(w, err)
				return
			}

			// Remove the flow continuation cookie
			removeCookie(w, FlowContinuationCookieName)

			// Finally, redirect them to the redirect URI with a code
			err = terminalRedirect(w, r, redirectURI, "code", authCode)
			if err != nil {
				util.Error(w, err)
				return
			}
		}
	}
}

// TokenExchangeResponse bundles together the token, the session, and the permissions
type TokenExchangeResponse struct {
	Token       string            `json:"token"`
	Session     types.Session     `json:"session"`
	Permissions types.Permissions `json:"permissions"`
}

// TokenExchange handles converting a code at the end of an auth flow
// into a normal JWT
func TokenExchange(authCodes *NonceMap, jwtManager *auth.JWTManager) func(w http.ResponseWriter, r *http.Request) {
	// Use a closure to inject dependencies
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the auth code from the body of the request
		authCodeBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Look for the auth code in the map
		rawToken, ok := authCodes.Use(string(authCodeBytes))
		if !ok {
			util.ErrorWithCode(w, errors.New("request had unknown auth code"),
				http.StatusForbidden)
			return
		}
		token, ok := rawToken.(*jwt.Token)
		if !ok {
			util.ErrorWithCode(w, errors.New("request had invalid flow continuation nonce value"),
				http.StatusForbidden)
			return
		}

		// Sign the JWT and return it to the user
		signed, err := jwtManager.SignToken(token)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Create the response object and send it to the user
		claims := token.Claims.(*auth.Claims)
		responseData := TokenExchangeResponse{
			Token:       signed,
			Session:     *claims.Session(),
			Permissions: claims.Permissions,
		}
		jsonResponse, err := json.Marshal(responseData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

// SessionResponse bundles together the session and the permissions
type SessionResponse struct {
	Session     types.Session     `json:"session"`
	Permissions types.Permissions `json:"permissions"`
}

// Session returns the inner data of the user's session by reading their JWT
// and validating it
func Session(jwtManager *auth.JWTManager) func(w http.ResponseWriter, r *http.Request) {
	// Use a closure to inject dependencies
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the claims from the token
		_, claims, err := auth.FromContext(r.Context())
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusUnauthorized)
		}

		// Create the response object and send it to the user
		responseData := SessionResponse{
			Session:     *claims.Session(),
			Permissions: claims.Permissions,
		}
		jsonResponse, err := json.Marshal(responseData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

// terminalRedirect is a utility function used at the end of the auth flow
// to send a single key-value pair to the original initiator
func terminalRedirect(w http.ResponseWriter, r *http.Request,
	baseURI string, key string, value string) error {

	u, err := url.ParseRequestURI(baseURI)
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

	http.Redirect(w, r, u.String(), http.StatusFound)
	return nil
}

// removeCookie sets a cookie with an empty value
func removeCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "storage",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
}
