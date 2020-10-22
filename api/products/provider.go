package products

import "context"

// Provider represents a Transact API provider
type Provider interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error

	PartialProductProvider
}

// PartialProductProvider represents a partial products provider implementation
type PartialProductProvider interface {
	GetAllLocations() ([]string, error)
	GetAllProducts(location string) ([]PartialProduct, error)
	GetProduct(location string, id string) (*PartialProduct, error)
}

// PartialProduct represents a partial product that has been retrieved from the Transact API
type PartialProduct struct {
	Name   string
	ID     string
	Amount int
}
