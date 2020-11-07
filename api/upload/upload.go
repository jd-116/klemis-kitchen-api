package upload

import (
	"context"
	"mime/multipart"
)

// Provider represents a S3 provider implementation
type Provider interface {
	MaxBytes() int64
	UploadFormMultipart(ctx context.Context, part *multipart.Part, ext string) (string, error)
}
