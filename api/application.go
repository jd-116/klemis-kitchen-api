package main

import (
	"context"
	"flag"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jd-116/klemis-kitchen-api/env"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Starts the main API and waits for termination signals.
// This function blocks.
func main() {
	envPath := flag.String("env", "", "path to .env file")
	logFormat := flag.String("log-format", "console", "log format (one of 'json', 'console')")
	flag.Parse()

	// Set up structured logging
	zerolog.TimeFieldFormat = time.RFC3339Nano
	var logger zerolog.Logger
	switch *logFormat {
	case "console":
		output := zerolog.ConsoleWriter{Out: os.Stdout}
		logger = zerolog.New(output).With().Timestamp().Logger()
	case "json":
		logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	default:
		log.Fatal().Str("log_format", *logFormat).Msg("unknown log format given")
	}
	stdlog.SetFlags(0)
	stdlog.SetOutput(logger)

	// Load the .env file if it is specified
	if envPath != nil && *envPath != "" {
		err := godotenv.Load(*envPath)
		if err != nil {
			logger.Fatal().Err(err).Str("env_path", *envPath).Msg("error loading .env file")
		} else {
			logger.Info().Str("env_path", *envPath).Msg("loaded environment variables from file")
		}
	}

	apiPort, err := env.GetIntEnv("server port", "PORT")
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load PORT from env")
	}

	serverCtx, cancel := context.WithCancel(context.Background())

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Propagate termination signals to the cancellation of the server context
	go func() {
		<-done
		cancel()
	}()

	// Initialize the API server object
	server, err := NewAPIServer(logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not initialize API server object")
	}

	// Connect to handlers
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()
	err = server.Connect(connectCtx)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not connect to downstream services")
	}

	// Disconnect automatically
	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer disconnectCancel()
		err := server.Disconnect(disconnectCtx)
		if err != nil {
			logger.Error().Err(err).Msg("error disconnecting from downstream services")
		}
	}()

	server.Serve(serverCtx, apiPort)
}
