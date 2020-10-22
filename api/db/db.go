package db

import (
	"context"

	"github.com/jd-116/klemis-kitchen-api/types"
)

// Provider represents a database provider implementation
type Provider interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error

	AnnouncementProvider
	ProductMetadataProvider
	LocationProvider
}

// AnnouncementProvider provides CRUD operations for type.Announcement structs
type AnnouncementProvider interface {
	GetAnnouncement(ctx context.Context, id string) (*types.Announcement, error)
	GetAllAnnouncements(ctx context.Context) ([]types.Announcement, error)
	CreateAnnouncement(ctx context.Context, announcement types.Announcement) error
	DeleteAnnouncement(ctx context.Context, id string) error
	UpdateAnnouncement(ctx context.Context, id string, update map[string]interface{}) (*types.Announcement, error)
}

// ProductMetadataProvider provides CRUD operations for type.ProductMetadata structs
type ProductMetadataProvider interface {
	GetProduct(ctx context.Context, id string) (*types.ProductMetadata, error)
	GetAllProducts(ctx context.Context) ([]types.ProductMetadata, error)
	CreateProduct(ctx context.Context, product types.ProductMetadata) error
	DeleteProduct(ctx context.Context, id string) error
	UpdateProduct(ctx context.Context, id string, update map[string]interface{}) (*types.ProductMetadata, error)
}

// LocationProvider provides CRUD operations for type.Location structs
type LocationProvider interface {
	GetLocation(ctx context.Context, id string) (*types.Location, error)
	GetAllLocations(ctx context.Context) ([]types.Location, error)
	CreateLocation(ctx context.Context, location types.Location) error
	DeleteLocation(ctx context.Context, id string) error
	UpdateLocation(ctx context.Context, id string, update map[string]interface{}) (*types.Location, error)
}
