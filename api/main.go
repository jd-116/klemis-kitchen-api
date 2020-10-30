package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jd-116/klemis-kitchen-api/util"
)

// Starts the main API and waits for termination signals.
// This function blocks.
func main() {
	apiPort, err := util.GetIntEnv("server port", "SERVER_PORT")
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
