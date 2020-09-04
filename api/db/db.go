package db

import "context"

// Represents a database provider implementation
type Provider interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
}
