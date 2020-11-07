package upload

import (
	"context"
	"io"
)

// Provider represents a S3 provider implementation
type Provider interface {
	MaxBytes() int64
	Upload(ctx context.Context, part io.Reader, ext string, mime string) (string, error)
}
