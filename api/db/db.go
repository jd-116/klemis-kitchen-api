package db

import (
	"context"

	"github.com/jd-116/klemis-kitchen-api/types"
)

// Represents a database provider implementation
type Provider interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error

	GadgetProvider
}

// Provides CRUD operations for type.Gadget structs
type GadgetProvider interface {
	GetGadget(ctx context.Context, id string) (*types.Gadget, error)
	GetAllGadgets(ctx context.Context) ([]types.Gadget, error)
	CreateGadget(ctx context.Context, gadget types.Gadget) error
	DeleteGadget(ctx context.Context, id string) error
	UpdateGadget(ctx context.Context, id string, update map[string]interface{}) (*types.Gadget, error)
}
