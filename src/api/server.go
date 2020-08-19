package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Runs the main API server until it's cancelled for some reason,
// in which case it attempts to gracefully shutdown.
// This function blocks.
func Serve(ctx context.Context, port int) {
	router := mux.NewRouter().PathPrefix("/api/v1").Subrouter()
	router.HandleFunc("/health", healthEndpoint).Methods("GET")

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Printf("server started; serving on port %d\n", port)

	<-ctx.Done()
	log.Print("server stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %+v", err)
	}
	log.Print("server exited properly")
}

// Can be used for health checks
func healthEndpoint(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(204)
}
