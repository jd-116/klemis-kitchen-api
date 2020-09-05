package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	duplicateError = 11000
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

	// Initialize any collections/indices
	err = p.initialize(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) Disconnect(ctx context.Context) error {
	err := p.client.Disconnect(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Create anything needed for the database,
// like indices
func (p *Provider) initialize(ctx context.Context) error {
	log.Println("initializing the MongoDB database")

	_, err := p.gadgets().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"id": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) gadgets() *mongo.Collection {
	return p.client.Database(p.databaseName).Collection("gadgets")
}

func (p *Provider) GetGadget(ctx context.Context, id string) (*types.Gadget, error) {
	collection := p.gadgets()
	result := collection.FindOne(ctx, bson.D{{"id", id}})
	if result.Err() == mongo.ErrNoDocuments {
		return nil, db.NewNotFoundError(id)
	}

	var gadget types.Gadget
	err := result.Decode(&gadget)
	if err != nil {
		return nil, err
	}

	return &gadget, nil
}

func (p *Provider) GetAllGadgets(ctx context.Context) ([]types.Gadget, error) {
	collection := p.gadgets()
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}

	var gadgets []types.Gadget
	err = cursor.All(ctx, &gadgets)
	if err != nil {
		return nil, err
	}

	// Return non-nil slice so JSON serialization is nice
	if gadgets == nil {
		return []types.Gadget{}, nil
	}

	return gadgets, nil
}

func (p *Provider) CreateGadget(ctx context.Context, gadget types.Gadget) error {
	collection := p.gadgets()
	_, err := collection.InsertOne(ctx, gadget)
	if err != nil {
		// Handle known cases (such as when the gadget was duplicate)
		if writeException, ok := err.(mongo.WriteException); ok && isDuplicate(writeException) {
			return db.NewDuplicateIDError(gadget.ID)
		}

		return err
	}

	return nil
}

// Detects if the given write exception is caused by (in part)
// by a duplicate key error
func isDuplicate(writeException mongo.WriteException) bool {
	for _, writeError := range writeException.WriteErrors {
		if writeError.Code == duplicateError {
			return true
		}
	}

	return false
}

func (p *Provider) DeleteGadget(ctx context.Context, id string) error {
	collection := p.gadgets()
	result, err := collection.DeleteOne(ctx, bson.D{{"id", id}})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return db.NewNotFoundError(id)
	}

	return nil
}

func (p *Provider) UpdateGadget(ctx context.Context, id string, update map[string]interface{}) (*types.Gadget, error) {
	// Construct the patch query from the map
	updateDocument := bson.D{}
	for key, value := range update {
		updateDocument = append(updateDocument, bson.E{key, value})
	}

	collection := p.gadgets()
	filter := bson.D{{"id", id}}
	updateQuery := bson.D{{"$set", updateDocument}}
	var updatedGadget types.Gadget
	err := collection.FindOneAndUpdate(ctx, filter, updateQuery).Decode(&updatedGadget)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, db.NewNotFoundError(id)
		}
	}

	return &updatedGadget, nil
}
