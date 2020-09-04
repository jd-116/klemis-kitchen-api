package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/jd-116/klemis-kitchen-api/util"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// type Provider struct{}

type Provider struct {
	Data string
}

// func NewProvider() (*Provider, error) {
// 	dbPort, err := util.GetIntEnv("MongoDB port", "MONGO_DB_PORT")
// 	if err != nil {
// 		return nil, err
// 	}

// 	dbHost, err := util.GetEnv("MongoDB host", "MONGO_DB_HOST")
// 	if err != nil {
// 		log.Println(err)
// 		return nil, err
// 	}
// }

func DBConnect() {
	dbhost, err := util.GetEnv("database host name", "MONGO_DB_HOST")
	dbpwd, err := util.GetEnv("database host name", "MONGO_DB_PWD")
	dbca, err := util.GetEnv("database host name", "MONGO_DB_CA")
	uri := fmt.Sprintf("mongodb+srv://%s:%s@%s.qkdgq.mongodb.net/inventory?retryWrites=true&w=majority", dbhost, dbpwd, dbca)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// Ping the primary
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected and pinged.")

	// collection := client.Database("inventory").Collection("provider")

	// err = client.Disconnect(context.TODO())

	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println("Connection to MongoDB closed.")
}
