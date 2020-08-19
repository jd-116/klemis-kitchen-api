package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/jd-116/klemis-kitchen-api/src/api"
)

// Gets the port from the environment
func getPort() int {
	port, exists := os.LookupEnv("SERVER_PORT")
	if exists {
		intPort, err := strconv.Atoi(port)
		if err == nil {
			return intPort
		}
	}

	return 8080
}

// Starts the main API and waits for termination signals.
// This function blocks.
func main() {
	port := getPort()
	serverCtx, cancel := context.WithCancel(context.Background())

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Propagate termination signals to the cancellation of the server context
	go func() {
		<-done
		cancel()
	}()

	api.Serve(serverCtx, port)
	log.Println("exiting")
}
