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
	GetGadget(id string) (*types.Gadget, error)
	GetAllGadgets() ([]types.Gadget, error)
	CreateGadget(gadget types.Gadget) error
	DeleteGadget(id string) error
	UpdateGadget(gadget types.Gadget) error
}
