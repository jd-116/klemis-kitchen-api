package db

import (
	"context"

	"github.com/jd-116/klemis-kitchen-api/types"
)

// Represents a database provider implementation
type Provider interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error

	AnnouncementProvider
}

// Provides CRUD operations for type.Announcement structs
type AnnouncementProvider interface {
	GetAnnouncement(ctx context.Context, id string) (*types.Announcement, error)
	GetAllAnnouncements(ctx context.Context) ([]types.Announcement, error)
	CreateAnnouncement(ctx context.Context, announcement types.Announcement) error
	DeleteAnnouncement(ctx context.Context, id string) error
	UpdateAnnouncement(ctx context.Context, id string, update map[string]interface{}) (*types.Announcement, error)
}
