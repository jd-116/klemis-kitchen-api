package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"

	"github.com/jd-116/klemis-kitchen-api/api/announcements"
	apiAuth "github.com/jd-116/klemis-kitchen-api/api/auth"
	"github.com/jd-116/klemis-kitchen-api/api/locations"
	"github.com/jd-116/klemis-kitchen-api/api/memberships"
	apiProducts "github.com/jd-116/klemis-kitchen-api/api/products"
	"github.com/jd-116/klemis-kitchen-api/auth"
	"github.com/jd-116/klemis-kitchen-api/cas"
	"github.com/jd-116/klemis-kitchen-api/db/mongo"
	"github.com/jd-116/klemis-kitchen-api/products/transact"
)

// APIServer is a struct that bundles together the various server-wide
// resources used at runtime that each have
// a lifecycle of initialization, connection, and disconnection
type APIServer struct {
	itemProvider *transact.Provider
	dbProvider   *mongo.Provider
	casProvider  *cas.Provider
	jwtManager   *auth.JWTManager
}

// NewAPIServer initializes the struct and all constituent components
func NewAPIServer() (*APIServer, error) {
	// Initialize the Transact scraper
	itemProvider, err := transact.NewProvider()
	if err != nil {
		return nil, err
	}

	// Initialize the MongoDB handler
	dbProvider, err := mongo.NewProvider()
	if err != nil {
		return nil, err
	}

	// Initialize the CAS provider
	casProvider, err := cas.NewProvider()
	if err != nil {
		return nil, err
	}

	// Initialize the JWT manager
	jwtManager, err := auth.NewJWTManager()
	if err != nil {
		return nil, err
	}

	return &APIServer{
		itemProvider,
		dbProvider,
		casProvider,
		jwtManager,
	}, nil
}

// Connect initializes the struct and all constituent components
func (a *APIServer) Connect(ctx context.Context) error {
	// Start the Transact scraper goroutines
	log.Println("initializing Transact API connector")
	err := a.itemProvider.Connect(ctx)
	if err != nil {
		log.Println("could not authenticate with the Transact API")
		return err
	}
	log.Printf("successfully authenticated with the Transact API version %s\n",
		a.itemProvider.Scraper.ClientVersion)

	// Connect to the MongoDB database
	log.Println("initializing MongoDB database provider")
	err = a.dbProvider.Connect(ctx)
	if err != nil {
		log.Println("could not disconnect to the database")
		return err
	}
	log.Println("successfully connected to and pinged the database")

	return nil
}

// Disconnect initializes the struct and all constituent components
func (a *APIServer) Disconnect(ctx context.Context) error {
	err := a.dbProvider.Disconnect(ctx)
	if err != nil {
		log.Println("could not disconnect from the database")
		return err
	}
	log.Println("disconnected from the database")

	err = a.itemProvider.Disconnect(ctx)
	if err != nil {
		log.Println("could not disconnect from the Transact API")
		return err
	}
	log.Println("disconnected from the Transact API")

	return nil
}

// Serve runs the main API server until it's cancelled for some reason,
// in which case it attempts to gracefully shutdown.
// This function blocks.
func (a *APIServer) Serve(ctx context.Context, port int) {
	router := a.routes()
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

func (a *APIServer) routes() *chi.Mux {
	// Approach from:
	// https://itnext.io/structuring-a-production-grade-rest-api-in-golang-c0229b3feedc
	// https://itnext.io/how-i-pass-around-shared-resources-databases-configuration-etc-within-golang-projects-b27af4d8e8a
	router := chi.NewRouter()
	router.Use(
		middleware.Recoverer,                          // Recover from panics without crashing the server
		middleware.Logger,                             // Log API request calls
		middleware.RedirectSlashes,                    // Redirect slashes to no slash URL versions
		render.SetContentType(render.ContentTypeJSON), // Set content-type headers to application/json
		middleware.Compress(5),                        // Compress results, mostly gzipping assets and json
		middleware.NoCache,                            // Prevent clients from caching the results
		a.corsMiddleware(),                            // Create cors middleware from go-chi/cors
	)

	// ==============================
	// Add all routes to the API here
	// ==============================
	router.Route("/v1", func(r chi.Router) {
		// Public routes
		r.Group(func(r chi.Router) {
			// Can be used for health checks
			r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(204)
			})

			r.Mount("/auth", apiAuth.Routes(a.casProvider, a.dbProvider, a.jwtManager))
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			// Seek, verify and validate JWT tokens,
			// sending appropriate status codes upon failure.
			// Note that this does not perform *authorization* checks involving perms;
			// if needed, use auth.AdminAuthenticator to use Permissions.AdminAccess
			r.Use(a.jwtManager.Authenticated())

			r.Mount("/announcements", announcements.Routes(a.dbProvider))
			r.Mount("/products", apiProducts.Routes(a.dbProvider, a.itemProvider))
			r.Mount("/locations", locations.Routes(a.dbProvider, a.itemProvider))
			r.Mount("/memberships", memberships.Routes(a.dbProvider))
		})
	})

	return router
}

func (a *APIServer) corsMiddleware() func(http.Handler) http.Handler {
	// See if the CORS_ALLOWED_ORIGINS environment variable was set
	allowedOrigins := "*"
	if value, ok := os.LookupEnv("CORS_ALLOWED_ORIGINS"); ok {
		allowedOrigins = value
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigins},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           300,
	})
}
