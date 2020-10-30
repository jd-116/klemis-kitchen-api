package auth

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/jd-116/klemis-kitchen-api/cas"
	"github.com/jd-116/klemis-kitchen-api/util"
)

func Routes(casProvider *cas.Provider) *chi.Mux {
	router := chi.NewRouter()
	router.Get("/login", Login(casProvider))
	return router
}

// Handles the GT SSO login flow via the CAS protocol v2
func Login(casProvider *cas.Provider) http.HandlerFunc {
	// Use a closure to inject dependencies
	return func(w http.ResponseWriter, r *http.Request) {
		// First, see if this is at the return of the CAS flow,
		// which will have a "ticket" query parameter if it is
		ticket := r.URL.Query().Get("ticket")

		if ticket == "" {
			// This is the first part of the CAS flow,
			// send them to the GT SSO flow

			// Get the URL to redirect to GT SSO
			err := casProvider.Redirect(w, r)
			if err != nil {
				util.Error(w, err)
			}

		} else {
			// This is the second part of the CAS flow,
			// send a request to the CAS server to validate the ticket
			result, err := casProvider.ServiceValidate(r, ticket)
			if err != nil {
				util.Error(w, err)
			}

			// TODO implement consuming the response
			fmt.Printf("got response with: %+v", result)
			fmt.Printf("                 : %#v", result)
		}
	}
}
