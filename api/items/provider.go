package items

import "context"

// Represents an items provider implementation
type Provider interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error

	// TODO add provided fields
}
