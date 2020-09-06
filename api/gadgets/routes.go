package gadgets

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

func Routes(database db.Provider) *chi.Mux {
	router := chi.NewRouter()
	router.Get("/", GetAll(database))
	router.Get("/{id}", GetSingle(database))
	router.Post("/", Create(database))
	router.Delete("/{id}", Delete(database))
	router.Patch("/", Update(database))
	return router
}

// Gets all gadgets from the database
func GetAll(database db.Provider) http.HandlerFunc {
	// Use a closure to inject the database provider
	return func(w http.ResponseWriter, r *http.Request) {
		gadgets, err := database.GetAllGadgets(r.Context())
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the list in a JSON object
		jsonResponse, err := json.Marshal(map[string]interface{}{
			"announcements": gadgets,
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

// Gets a single gadget from the database by its ID
func GetSingle(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		gadget, err := database.GetGadget(r.Context(), id)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the single gadget as the top-level JSON
		jsonResponse, err := json.Marshal(gadget)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonResponse)
	}
}

// Creates a new gadget in the database
func Create(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var gadget types.Gadget
		err := json.NewDecoder(r.Body).Decode(&gadget)
		if err != nil {
			util.Error(w, err)
			return
		}

		err = database.CreateGadget(r.Context(), gadget)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the single gadget as the top-level JSON
		jsonResponse, err := json.Marshal(gadget)
		if err != nil {
			util.ErrorWithCode(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(jsonResponse)
	}
}

// Deletes a gadget in the database
func Delete(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			util.ErrorWithCode(w, errors.New("the URL parameter is empty"),
				http.StatusBadRequest)
			return
		}

		err := database.DeleteGadget(r.Context(), id)
		if err != nil {
			util.Error(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// Updates a gadget in the database
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

		updated, err := database.UpdateGadget(r.Context(), id, partial)
		if err != nil {
			util.Error(w, err)
			return
		}

		// Return the updated gadget as the top-level JSON
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
