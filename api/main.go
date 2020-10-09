package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jd-116/klemis-kitchen-api/db/mongo"
	"github.com/jd-116/klemis-kitchen-api/products/transact"
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

	// Initialize the Transact scraper & start the goroutines to scrape
	itemProvider, err := transact.NewProvider()
	log.Println("initializing Transact API connector")
	if err != nil {
		log.Fatal(err)
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()
	err = itemProvider.Connect(connectCtx)
	if err != nil {
		log.Println("could not authenticate with the Transact API")
		log.Fatal(err)
	} else {
		log.Printf("successfully authenticated with the Transact API version %s\n", itemProvider.Scraper.ClientVersion)
	}

	// Initialize the DB handler & connect to the MongoDB database
	dbProvider, err := mongo.NewProvider()
	log.Println("initializing MongoDB database provider")
	if err != nil {
		log.Fatal(err)
	}
	connectCtx, connectCancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()
	err = dbProvider.Connect(connectCtx)
	if err != nil {
		log.Println("could not disconnect to the database")
		log.Fatal(err)
	} else {
		log.Println("successfully connected to and pinged the database")
	}

	// Disconnect automatically
	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer disconnectCancel()
		err := dbProvider.Disconnect(disconnectCtx)
		if err != nil {
			log.Println("could not disconnect from the database")
			log.Println(err)
		} else {
			log.Println("disconnected from the database")
		}

		disconnectItemsCtx, disconnectItemsCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer disconnectItemsCancel()
		err = itemProvider.Disconnect(disconnectItemsCtx)
		if err != nil {
			log.Println("could not disconnect from the Transact API")
			log.Println(err)
		} else {
			log.Println("disconnected from the Transact API")
		}
	}()

	ServeAPI(serverCtx, apiPort, dbProvider, itemProvider)
}
