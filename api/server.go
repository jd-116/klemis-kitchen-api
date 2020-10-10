package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"github.com/jd-116/klemis-kitchen-api/api/announcements"
	"github.com/jd-116/klemis-kitchen-api/api/products" apiProducts
	"github.com/jd-116/klemis-kitchen-api/api/locations"
	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/products"
)

func Routes(database db.Provider, products products.Provider) *chi.Mux {
	// Approach from:
	// https://itnext.io/structuring-a-production-grade-rest-api-in-golang-c0229b3feedc
	// https://itnext.io/how-i-pass-around-shared-resources-databases-configuration-etc-within-golang-projects-b27af4d8e8a
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON), // Set content-type headers to application/json
		middleware.Logger,          // Log API request calls
		middleware.Compress(5),     // Compress results, mostly gzipping assets and json
		middleware.RedirectSlashes, // Redirect slashes to no slash URL versions
		middleware.Recoverer,       // Recover from panics without crashing the server
	)

	// ==============================
	// Add all routes to the API here
	// ==============================
	router.Route("/api/v1", func(r chi.Router) {
		// Can be used for health checks
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(204)
		})

		r.Mount("/announcements", announcements.Routes(database))
		r.Mount("/products", apiProducts.Routes(database, products))
		r.Mount("/locations", locations.Routes(database, products))
	})

	return router
}

// Runs the main API server until it's cancelled for some reason,
// in which case it attempts to gracefully shutdown.
// This function blocks.
func ServeAPI(ctx context.Context, port int, database db.Provider, products products.Provider) {
	router := Routes(database, products)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Printf("API server started; serving on port %d\n", port)

	<-ctx.Done()
	log.Println("API server stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("API server shutdown failed: %+v", err)
	}
	log.Println("API server exited properly")
}
