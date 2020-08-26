package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jd-116/klemis-kitchen-api/util"
)

// Starts the main API and waits for termination signals.
// This function blocks.
func main() {
	apiPort, err := util.GetIntEnv("server port", "SERVER_PORT")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	serverCtx, cancel := context.WithCancel(context.Background())

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Propagate termination signals to the cancellation of the server context
	go func() {
		<-done
		cancel()
	}()

	ServeAPI(serverCtx, apiPort)
	log.Println("exiting")
}
