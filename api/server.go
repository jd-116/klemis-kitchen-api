package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"

	"github.com/jd-116/klemis-kitchen-api/api/announcements"
	apiAuth "github.com/jd-116/klemis-kitchen-api/api/auth"
	"github.com/jd-116/klemis-kitchen-api/api/locations"
	"github.com/jd-116/klemis-kitchen-api/api/memberships"
	apiProducts "github.com/jd-116/klemis-kitchen-api/api/products"
	apiUpload "github.com/jd-116/klemis-kitchen-api/api/upload"
	"github.com/jd-116/klemis-kitchen-api/auth"
	"github.com/jd-116/klemis-kitchen-api/cas"
	"github.com/jd-116/klemis-kitchen-api/db/mongo"
	"github.com/jd-116/klemis-kitchen-api/products/transact"
	"github.com/jd-116/klemis-kitchen-api/upload/s3"
)

// APIServer is a struct that bundles together the various server-wide
// resources used at runtime that each have
// a lifecycle of initialization, connection, and disconnection
type APIServer struct {
	itemProvider   *transact.Provider
	dbProvider     *mongo.Provider
	casProvider    *cas.Provider
	jwtManager     *auth.JWTManager
	uploadProvider *s3.Provider
	logger         zerolog.Logger
}

// NewAPIServer initializes the struct and all constituent components
func NewAPIServer(logger zerolog.Logger) (*APIServer, error) {
	// Initialize the Transact scraper
	itemProvider, err := transact.NewProvider(logger)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize Transact scraper")
	}

	// Initialize the MongoDB handler
	dbProvider, err := mongo.NewProvider()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize MongoDB handler")
	}

	// Initialize the CAS provider
	casProvider, err := cas.NewProvider()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize CAS provider")
	}

	// Initialize the JWT manager
	jwtManager, err := auth.NewJWTManager()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize JWT manager")
	}

	// Initialize the S3 handler
	uploadProvider, err := s3.NewProvider()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize S3 handler")
	}

	return &APIServer{
		itemProvider:   itemProvider,
		dbProvider:     dbProvider,
		casProvider:    casProvider,
		jwtManager:     jwtManager,
		uploadProvider: uploadProvider,
		logger:         logger,
	}, nil
}

// Connect initializes the struct and all constituent components
func (a *APIServer) Connect(ctx context.Context) error {
	// Start the Transact scraper goroutines
	a.logger.Info().Msg("initializing Transact API connector")
	err := a.itemProvider.Connect(ctx)
	if err != nil {
		return errors.Wrap(err, "could not authenticate with the Transact API")
	}
	a.logger.
		Info().
		Str("transact_version", a.itemProvider.Scraper.ClientVersion).
		Msg("successfully authenticated with the Transact API")

	// Connect to the MongoDB database
	a.logger.Info().Msg("initializing MongoDB database provider")
	err = a.dbProvider.Connect(ctx)
	if err != nil {
		return errors.Wrap(err, "could not disconnect to the database")
	}
	a.logger.Info().Msg("successfully connected to and pinged the database")

	return nil
}

// Disconnect initializes the struct and all constituent components
func (a *APIServer) Disconnect(ctx context.Context) error {
	err := a.dbProvider.Disconnect(ctx)
	if err != nil {
		return errors.Wrap(err, "could not disconnect from the database")
	}
	a.logger.Info().Msg("disconnected from the database")

	err = a.itemProvider.Disconnect(ctx)
	if err != nil {
		return errors.Wrap(err, "could not disconnect from the Transact API")
	}
	a.logger.Info().Msg("disconnected from the Transact API")

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
			a.logger.Fatal().Err(err).Msg("an error occurred when serving the HTTP server")
		}
	}()
	a.logger.Info().Int("port", port).Msg("API server started")

	<-ctx.Done()
	a.logger.Info().Msg("API server stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		a.logger.Fatal().Err(err).Msg("API server shutdown failed")
	}
	a.logger.Info().Msg("API server exited properly")
}

func (a *APIServer) routes() *chi.Mux {
	// Approach from:
	// https://itnext.io/structuring-a-production-grade-rest-api-in-golang-c0229b3feedc
	// https://itnext.io/how-i-pass-around-shared-resources-databases-configuration-etc-within-golang-projects-b27af4d8e8a
	router := chi.NewRouter()
	router.Use(
		middleware.Recoverer,      // Recover from panics without crashing the server
		hlog.NewHandler(a.logger), // Attach a logger to each request
		hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
			// Log API request calls once they complete:
			hlog.FromRequest(r).Info().
				Str("method", r.Method).
				Stringer("url", r.URL).
				Int("status", status).
				Int("bytes_out", size).
				Dur("duration", duration).
				Str("ip", r.RemoteAddr).
				Str("user_agent", r.Header.Get("User-Agent")).
				Msg("handled HTTP request")
		}),
		hlog.RequestIDHandler("req_id", "X-Request-Id"), // Attach a unique request ID to each incoming request
		middleware.RedirectSlashes,                      // Redirect slashes to no slash URL versions
		render.SetContentType(render.ContentTypeJSON),   // Set content-type headers to application/json
		middleware.Compress(5),                          // Compress results, mostly gzipping assets and json
		middleware.NoCache,                              // Prevent clients from caching the results
		a.corsMiddleware(),                              // Create cors middleware from go-chi/cors
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
			r.Mount("/upload", apiUpload.Routes(a.uploadProvider))
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
