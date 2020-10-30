package announcements

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

// Routes creates a new Chi router with all of the routes for the announcement resource,
// at the root level
func Routes(database db.Provider) *chi.Mux {
	router := chi.NewRouter()
	router.Get("/", GetAll(database))
	router.Get("/{id}", GetSingle(database))
	router.Post("/", Create(database))
	router.Delete("/{id}", Delete(database))
	router.Patch("/", Update(database))
	return router
}

// GetAll gets all announcements from the database
func GetAll(database db.Provider) http.HandlerFunc {
	// Use a closure to inject the database provider
	return func(w http.ResponseWriter, r *http.Request) {
		announcements, err := database.GetAllAnnouncements(r.Context())
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the list in a JSON object
		jsonResponse, err := json.Marshal(map[string]interface{}{
			"announcements": announcements,
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

// GetSingle gets a single announcement from the database by its ID
func GetSingle(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		announcement, err := database.GetAnnouncement(r.Context(), id)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the single announcement as the top-level JSON
		jsonResponse, err := json.Marshal(announcement)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

// Create creates a new announcement in the database
func Create(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var announcement types.Announcement
		err := json.NewDecoder(r.Body).Decode(&announcement)
		if err != nil {
			util.Error(w, err)
			return
		}

		announcement.ID = strings.TrimSpace(announcement.ID)
		if announcement.ID == "" {
			util.ErrorWithCode(w, errors.New("announcement ID cannot be empty"),
				http.StatusBadRequest)
			return
		}

		err = database.CreateAnnouncement(r.Context(), announcement)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the single announcement as the top-level JSON
		jsonResponse, err := json.Marshal(announcement)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(jsonResponse)
	}
}

// Delete deletes a announcement in the database
func Delete(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		err := database.DeleteAnnouncement(r.Context(), id)
		if err != nil {
			util.Error(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// Update updates a announcement in the database
func Update(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
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

		updated, err := database.UpdateAnnouncement(r.Context(), id, partial)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the updated announcement as the top-level JSON
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
