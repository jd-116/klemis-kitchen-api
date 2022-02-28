package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/jd-116/klemis-kitchen-api/db"
	"github.com/jd-116/klemis-kitchen-api/env"
	"github.com/jd-116/klemis-kitchen-api/types"
	"github.com/rs/zerolog"
)

const (
	duplicateError = 11000
)

// Provider implements the Provider interface for a MongoDB connection
type Provider struct {
	logger        zerolog.Logger
	connectionURI string
	databaseName  string
	clusterName   string
	client        *mongo.Client
}

// NewProvider creates a new provider and loads values in from the environment
func NewProvider(logger zerolog.Logger) (*Provider, error) {
	username, err := env.GetEnv("MongoDB username", "MONGO_DB_USERNAME")
	if err != nil {
		return nil, err
	}

	password, err := env.GetEnv("MongoDB password", "MONGO_DB_PASSWORD")
	if err != nil {
		return nil, err
	}

	clusterName, err := env.GetEnv("MongoDB cluster name ", "MONGO_DB_CLUSTER_NAME")
	if err != nil {
		return nil, err
	}

	databaseName, err := env.GetEnv("MongoDB database name ", "MONGO_DB_DATABASE_NAME")
	if err != nil {
		return nil, err
	}

	connectionURI := fmt.Sprintf("mongodb+srv://%s:%s@%s.qkdgq.mongodb.net/%s?retryWrites=true&w=majority",
		username, password, clusterName, databaseName)
	return &Provider{
		logger:        logger,
		connectionURI: connectionURI,
		databaseName:  databaseName,
		clusterName:   clusterName,
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

// Disconnect terminates the connection to the MongoDB server
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
	p.logger.
		Info().
		Str("database_name", p.databaseName).
		Str("cluster_name", p.clusterName).
		Msg("initializing the MongoDB database")

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

	_, err = p.memberships().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"username": 1},
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

func (p *Provider) memberships() *mongo.Collection {
	return p.client.Database(p.databaseName).Collection("memberships")
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

// GetMembership gets a single membership given its username
func (p *Provider) GetMembership(ctx context.Context, username string) (*types.Membership, error) {
	collection := p.memberships()
	result := collection.FindOne(ctx, bson.D{{Key: "username", Value: username}})
	if result.Err() == mongo.ErrNoDocuments {
		return nil, db.NewNotFoundError(username)
	}

	var membership types.Membership
	err := result.Decode(&membership)
	if err != nil {
		return nil, err
	}

	return &membership, nil
}

// GetAllAnnouncements gets a slice of all announcements in the database
func (p *Provider) GetAllAnnouncements(ctx context.Context) ([]types.Announcement, error) {
	collection := p.announcements()

	// Sort the announcements by their timestamp (descending)
	options := options.Find()
	options.SetSort(bson.D{{Key: "timestamp", Value: -1}})
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

// GetAllMemberships gets a slice of all memberships in the database
func (p *Provider) GetAllMemberships(ctx context.Context) ([]types.Membership, error) {
	collection := p.memberships()

	options := options.Find()
	options.SetSort(bson.D{{Key: "username", Value: 1}})
	cursor, err := collection.Find(ctx, bson.D{}, options)
	if err != nil {
		return nil, err
	}

	var memberships []types.Membership
	err = cursor.All(ctx, &memberships)
	if err != nil {
		return nil, err
	}

	// Return non-nil slice so JSON serialization is nice
	if memberships == nil {
		return []types.Membership{}, nil
	}

	return memberships, nil
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

// CreateLocation attempts to insert a new membership into the database
func (p *Provider) CreateMembership(ctx context.Context, membership types.Membership) error {
	collection := p.memberships()
	_, err := collection.InsertOne(ctx, membership)
	if err != nil {
		// Handle known cases (such as when the membership was duplicate)
		if writeException, ok := err.(mongo.WriteException); ok && isDuplicate(writeException) {
			return db.NewDuplicateIDError(membership.Username)
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

// DeleteLocation deletes an existing location by its username
func (p *Provider) DeleteMembership(ctx context.Context, username string) error {
	collection := p.memberships()
	result, err := collection.DeleteOne(ctx, bson.D{{Key: "username", Value: username}})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return db.NewNotFoundError(username)
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

// UpdateLocation updates an existing membership by its username
// and a partial document containing new fields that override current ones
func (p *Provider) UpdateMembership(ctx context.Context, username string, update map[string]interface{}) (*types.Membership, error) {
	// Construct the patch query from the map
	updateDocument := bson.D{}
	for key, value := range update {
		updateDocument = append(updateDocument, bson.E{Key: key, Value: value})
	}

	collection := p.memberships()
	filter := bson.D{{Key: "username", Value: username}}
	updateQuery := bson.D{{Key: "$set", Value: updateDocument}}
	var updatedMembership types.Membership
	err := collection.FindOneAndUpdate(ctx, filter, updateQuery).Decode(&updatedMembership)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, db.NewNotFoundError(username)
		}
	}

	return &updatedMembership, nil
}
