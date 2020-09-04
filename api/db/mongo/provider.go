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

type Provider struct {
	connectionUri string
	databaseName  string
	client        *mongo.Client
}

// Creates a new provider and loads values in from the environment
func NewProvider() (*Provider, error) {
	dbHost, err := util.GetEnv("database host name", "MONGO_DB_HOST")
	if err != nil {
		return nil, err
	}

	dbPwd, err := util.GetEnv("database password", "MONGO_DB_PWD")
	if err != nil {
		return nil, err
	}

	dbCluster, err := util.GetEnv("database cluster name ", "MONGO_DB_CLUSTER")
	if err != nil {
		return nil, err
	}

	dbName, err := util.GetEnv("database name ", "MONGO_DB_NAME")
	if err != nil {
		return nil, err
	}

	connectionUri := fmt.Sprintf("mongodb+srv://%s:%s@%s.qkdgq.mongodb.net/%s?retryWrites=true&w=majority",
		dbHost, dbPwd, dbCluster, dbName)
	return &Provider{
		connectionUri: connectionUri,
		databaseName:  dbName,
		client:        nil,
	}, nil
}

func (p *Provider) Connect(ctx context.Context) error {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(p.connectionUri))
	if err != nil {
		return err
	}

	// Ping the primary
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		return err
	}

	p.client = client
	return nil
}

func (p *Provider) Disconnect(ctx context.Context) error {
	err := p.client.Disconnect(ctx)
	if err != nil {
		return err
	}

	return nil
}
