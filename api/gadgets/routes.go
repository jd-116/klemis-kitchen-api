package gadgets

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/jd-116/klemis-kitchen-api/db"
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
		w.WriteHeader(200)
		fmt.Fprintln(w, "here are all of them :)")
	}
}

// Gets a single gadget from the database by its ID
func GetSingle(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		w.WriteHeader(200)
		fmt.Fprintf(w, "Here is the one with id '%s'\n", id)
	}
}

// Creates a new gadget in the database
func Create(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "ok im creating it")
	}
}

// Deletes a gadget in the database
func Delete(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		w.WriteHeader(200)
		fmt.Fprintf(w, "Ok I'm deleting the one with id '%s'\n", id)
	}
}

// Updates a gadget in the database
func Update(database db.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "ok im deleting it")
	}
}
