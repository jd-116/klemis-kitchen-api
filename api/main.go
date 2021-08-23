package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jd-116/klemis-kitchen-api/env"
	"github.com/joho/godotenv"
)

// Starts the main API and waits for termination signals.
// This function blocks.
func main() {
	// Load the .env file if it is specified
	envPath := flag.String("env", "", "path to .env file")
	flag.Parse()
	if envPath != nil && *envPath != "" {
		err := godotenv.Load(*envPath)
		if err != nil {
			log.Fatal("error loading .env file")
		} else {
			log.Printf("loaded environment variables from %s file\n", *envPath)
		}
	}

	apiPort, err := env.GetIntEnv("server port", "SERVER_PORT")
	if err != nil {
		log.Fatal(err)
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
	server, err := NewAPIServer()
	if err != nil {
		log.Fatal(err)
	}

	// Connect to handlers
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()
	err = server.Connect(connectCtx)
	if err != nil {
		log.Fatal(err)
	}

	// Disconnect automatically
	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer disconnectCancel()
		err := server.Disconnect(disconnectCtx)
		if err != nil {
			log.Println(err)
		}
	}()

	server.Serve(serverCtx, apiPort)
}
