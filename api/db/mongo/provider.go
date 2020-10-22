package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/jd-116/klemis-kitchen-api/util"
)

const (
	duplicateError = 11000
)

// Provider implements the Provider interface for a MongoDB connection
type Provider struct {
	connectionURI string
	databaseName  string
	client        *mongo.Client
}

// NewProvider creates a new provider and loads values in from the environment
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

	connectionURI := fmt.Sprintf("mongodb+srv://%s:%s@%s.qkdgq.mongodb.net/%s?retryWrites=true&w=majority",
		dbHost, dbPwd, dbCluster, dbName)
	return &Provider{
		connectionURI: connectionURI,
		databaseName:  dbName,
		client:        nil,
	}, nil
}

// Connect connects to the MongoDB server and creates any indices as necessary
func (p *Provider) Connect(ctx context.Context) error {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(p.connectionURI))
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

// Disconnect terminates the connection to the MongODB server
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

	_, err := p.announcements().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"id": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	_, err = p.products().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"id": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	_, err = p.locations().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"id": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) announcements() *mongo.Collection {
	return p.client.Database(p.databaseName).Collection("announcements")
}

func (p *Provider) products() *mongo.Collection {
	return p.client.Database(p.databaseName).Collection("productMetadata")
}

func (p *Provider) locations() *mongo.Collection {
	return p.client.Database(p.databaseName).Collection("locations")
}

// GetAnnouncement gets a single announcement given its ID
func (p *Provider) GetAnnouncement(ctx context.Context, id string) (*types.Announcement, error) {
	collection := p.announcements()
	result := collection.FindOne(ctx, bson.D{{Key: "id", Value: id}})
	if result.Err() == mongo.ErrNoDocuments {
		return nil, db.NewNotFoundError(id)
	}

	var announcement types.Announcement
	err := result.Decode(&announcement)
	if err != nil {
		return nil, err
	}

	return &announcement, nil
}

// GetProduct gets a single product metadata object given its ID
func (p *Provider) GetProduct(ctx context.Context, id string) (*types.ProductMetadata, error) {
	collection := p.products()
	result := collection.FindOne(ctx, bson.D{{Key: "id", Value: id}})
	if result.Err() == mongo.ErrNoDocuments {
		return nil, db.NewNotFoundError(id)
	}

	var product types.ProductMetadata
	err := result.Decode(&product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

// GetLocation gets a single location given its ID
func (p *Provider) GetLocation(ctx context.Context, id string) (*types.Location, error) {
	collection := p.locations()
	result := collection.FindOne(ctx, bson.D{{Key: "id", Value: id}})
	if result.Err() == mongo.ErrNoDocuments {
		return nil, db.NewNotFoundError(id)
	}

	var location types.Location
	err := result.Decode(&location)
	if err != nil {
		return nil, err
	}

	return &location, nil
}

// GetAllAnnouncements gets a slice of all announcements in the database
func (p *Provider) GetAllAnnouncements(ctx context.Context) ([]types.Announcement, error) {
	collection := p.announcements()

	options := options.Find()
	options.SetSort(bson.D{{Key: "id", Value: 1}})
	cursor, err := collection.Find(ctx, bson.D{}, options)
	if err != nil {
		return nil, err
	}

	var announcements []types.Announcement
	err = cursor.All(ctx, &announcements)
	if err != nil {
		return nil, err
	}

	// Return non-nil slice so JSON serialization is nice
	if announcements == nil {
		return []types.Announcement{}, nil
	}

	return announcements, nil
}

// GetAllProducts gets a slice of all product metadata objects in the database
func (p *Provider) GetAllProducts(ctx context.Context) ([]types.ProductMetadata, error) {
	collection := p.products()

	options := options.Find()
	options.SetSort(bson.D{{Key: "id", Value: 1}})
	cursor, err := collection.Find(ctx, bson.D{}, options)
	if err != nil {
		return nil, err
	}

	var products []types.ProductMetadata
	err = cursor.All(ctx, &products)
	if err != nil {
		return nil, err
	}

	// Return non-nil slice so JSON serialization is nice
	if products == nil {
		return []types.ProductMetadata{}, nil
	}

	return products, nil
}

// GetAllLocations gets a slice of all locations in the database
func (p *Provider) GetAllLocations(ctx context.Context) ([]types.Location, error) {
	collection := p.locations()

	options := options.Find()
	options.SetSort(bson.D{{Key: "id", Value: 1}})
	cursor, err := collection.Find(ctx, bson.D{}, options)
	if err != nil {
		return nil, err
	}

	var locations []types.Location
	err = cursor.All(ctx, &locations)
	if err != nil {
		return nil, err
	}

	// Return non-nil slice so JSON serialization is nice
	if locations == nil {
		return []types.Location{}, nil
	}

	return locations, nil
}

// CreateAnnouncement attempts to insert a new announcement into the database
func (p *Provider) CreateAnnouncement(ctx context.Context, announcement types.Announcement) error {
	collection := p.announcements()
	_, err := collection.InsertOne(ctx, announcement)
	if err != nil {
		// Handle known cases (such as when the announcement was duplicate)
		if writeException, ok := err.(mongo.WriteException); ok && isDuplicate(writeException) {
			return db.NewDuplicateIDError(announcement.ID)
		}

		return err
	}

	return nil
}

// CreateProduct attempts to insert a new product metadata object into the database
func (p *Provider) CreateProduct(ctx context.Context, product types.ProductMetadata) error {
	collection := p.products()
	_, err := collection.InsertOne(ctx, product)
	if err != nil {
		// Handle known cases (such as when the product was duplicate)
		if writeException, ok := err.(mongo.WriteException); ok && isDuplicate(writeException) {
			return db.NewDuplicateIDError(product.ID)
		}

		return err
	}

	return nil
}

// CreateLocation attempts to insert a new location into the database
func (p *Provider) CreateLocation(ctx context.Context, location types.Location) error {
	collection := p.locations()
	_, err := collection.InsertOne(ctx, location)
	if err != nil {
		// Handle known cases (such as when the location was duplicate)
		if writeException, ok := err.(mongo.WriteException); ok && isDuplicate(writeException) {
			return db.NewDuplicateIDError(location.ID)
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

// DeleteAnnouncement deletes an existing announcement by its ID
func (p *Provider) DeleteAnnouncement(ctx context.Context, id string) error {
	collection := p.announcements()
	result, err := collection.DeleteOne(ctx, bson.D{{Key: "id", Value: id}})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return db.NewNotFoundError(id)
	}

	return nil
}

// DeleteProduct deletes an existing product metadata object by its ID
func (p *Provider) DeleteProduct(ctx context.Context, id string) error {
	collection := p.products()
	result, err := collection.DeleteOne(ctx, bson.D{{Key: "id", Value: id}})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return db.NewNotFoundError(id)
	}

	return nil
}

// DeleteLocation deletes an existing location by its ID
func (p *Provider) DeleteLocation(ctx context.Context, id string) error {
	collection := p.locations()
	result, err := collection.DeleteOne(ctx, bson.D{{Key: "id", Value: id}})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return db.NewNotFoundError(id)
	}

	return nil
}

// UpdateAnnouncement updates an existing announcement by its ID
// and a partial document containing new fields that override current ones
func (p *Provider) UpdateAnnouncement(ctx context.Context, id string, update map[string]interface{}) (*types.Announcement, error) {
	// Construct the patch query from the map
	updateDocument := bson.D{}
	for key, value := range update {
		updateDocument = append(updateDocument, bson.E{Key: key, Value: value})
	}

	collection := p.announcements()
	filter := bson.D{{Key: "id", Value: id}}
	updateQuery := bson.D{{Key: "$set", Value: updateDocument}}
	var updatedAnnouncement types.Announcement
	err := collection.FindOneAndUpdate(ctx, filter, updateQuery).Decode(&updatedAnnouncement)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, db.NewNotFoundError(id)
		}
	}

	return &updatedAnnouncement, nil
}

// UpdateProduct updates an existing product metadata object by its ID
// and a partial document containing new fields that override current ones
func (p *Provider) UpdateProduct(ctx context.Context, id string, update map[string]interface{}) (*types.ProductMetadata, error) {
	// Construct the patch query from the map
	updateDocument := bson.D{}
	for key, value := range update {
		updateDocument = append(updateDocument, bson.E{Key: key, Value: value})
	}

	collection := p.products()
	filter := bson.D{{Key: "id", Value: id}}
	updateQuery := bson.D{{Key: "$set", Value: updateDocument}}
	var updatedProduct types.ProductMetadata
	err := collection.FindOneAndUpdate(ctx, filter, updateQuery).Decode(&updatedProduct)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, db.NewNotFoundError(id)
		}
	}

	return &updatedProduct, nil
}

// UpdateLocation updates an existing location by its ID
// and a partial document containing new fields that override current ones
func (p *Provider) UpdateLocation(ctx context.Context, id string, update map[string]interface{}) (*types.Location, error) {
	// Construct the patch query from the map
	updateDocument := bson.D{}
	for key, value := range update {
		updateDocument = append(updateDocument, bson.E{Key: key, Value: value})
	}

	collection := p.locations()
	filter := bson.D{{Key: "id", Value: id}}
	updateQuery := bson.D{{Key: "$set", Value: updateDocument}}
	var updatedLocation types.Location
	err := collection.FindOneAndUpdate(ctx, filter, updateQuery).Decode(&updatedLocation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, db.NewNotFoundError(id)
		}
	}

	return &updatedLocation, nil
}
