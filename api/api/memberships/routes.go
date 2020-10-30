package memberships

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi"

	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

// Routes creates a new Chi router with all of the routes for the membership resource,
// at the root level
func Routes(database db.Provider) *chi.Mux {
	router := chi.NewRouter()
	router.Get("/", GetAll(database))
	router.Get("/{username}", GetSingle(database))
	router.Post("/", Create(database))
	router.Delete("/{username}", Delete(database))
	router.Patch("/{username}", Update(database))
	return router
}

// GetAll gets all memberships from the database
func GetAll(membershipProvider db.MembershipProvider) http.HandlerFunc {
	// Use a closure to inject the database provider
	return func(w http.ResponseWriter, r *http.Request) {
		memberships, err := membershipProvider.GetAllMemberships(r.Context())
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the list in a JSON object
		jsonResponse, err := json.Marshal(map[string]interface{}{
			"memberships": memberships,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

// GetSingle gets a single membership from the database by its username
func GetSingle(membershipProvider db.MembershipProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := chi.URLParam(r, "username")
		if username == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		membership, err := membershipProvider.GetMembership(r.Context(), username)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the single membership as the top-level JSON
		jsonResponse, err := json.Marshal(membership)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

// Create creates a new membership in the database
func Create(membershipProvider db.MembershipProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var membership types.Membership
		err := json.NewDecoder(r.Body).Decode(&membership)
		if err != nil {
			util.Error(w, err)
			return
		}

		membership.Username = strings.TrimSpace(membership.Username)
		if membership.Username == "" {
			util.ErrorWithCode(w, errors.New("membership Username cannot be empty"),
				http.StatusBadRequest)
			return
		}

		err = membershipProvider.CreateMembership(r.Context(), membership)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the single membership as the top-level JSON
		jsonResponse, err := json.Marshal(membership)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(jsonResponse)
	}
}

// Delete deletes a membership in the database
func Delete(membershipProvider db.MembershipProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := chi.URLParam(r, "username")
		if username == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		err := membershipProvider.DeleteMembership(r.Context(), username)
		if err != nil {
			util.Error(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// Update updates a membership in the database
func Update(membershipProvider db.MembershipProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := chi.URLParam(r, "username")
		if username == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		partial := make(map[string]interface{})
		err := json.NewDecoder(r.Body).Decode(&partial)
		if err != nil {
			util.Error(w, err)
			return
		}

		updated, err := membershipProvider.UpdateMembership(r.Context(), username, partial)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the updated membership as the top-level JSON
		jsonResponse, err := json.Marshal(updated)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}
