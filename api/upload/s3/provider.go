package s3

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/segmentio/ksuid"

	"github.com/jd-116/klemis-kitchen-api/env"
)

// Provider implements an upload provider against the S3 API
type Provider struct {
	maxBytes int64
	session  *session.Session
	uploader *s3manager.Uploader
	bucket   string
}

// NewProvider creates a new instance of a Provider
// and parses environment variables
func NewProvider() (*Provider, error) {
	maxBytes, err := env.GetBytesEnv("max upload file size", "UPLOAD_MAX_SIZE")
	if err != nil {
		return nil, err
	}

	// Parse the S3 credentials from the environment
	awsRegion, err := env.GetEnv("upload AWS region", "UPLOAD_AWS_REGION")
	if err != nil {
		return nil, err
	}
	awsAccessKeyID, err := env.GetEnv("upload AWS access key ID", "UPLOAD_AWS_ACCESS_KEY_ID")
	if err != nil {
		return nil, err
	}
	awsSecretAccessKey, err := env.GetEnv("upload AWS secret access key", "UPLOAD_AWS_SECRET_ACCESS_KEY")
	if err != nil {
		return nil, err
	}

	// Initialize the session
	session, err := session.NewSession(&aws.Config{
		Region:      &awsRegion,
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})
	if err != nil {
		return nil, err
	}

	// Parse the uploader data from the environment
	uploadPartSize, err := env.GetBytesEnv("upload part size", "UPLOAD_PART_SIZE")
	if err != nil {
		return nil, err
	}

	// Initialize the uploader
	uploader := s3manager.NewUploader(session, func(u *s3manager.Uploader) {
		u.PartSize = int64(uploadPartSize.Bytes())
		u.LeavePartsOnError = false
	})

	// Get the bucket name from the environment
	s3Bucket, err := env.GetEnv("upload S3 bucket", "UPLOAD_S3_BUCKET")
	if err != nil {
		return nil, err
	}

	return &Provider{
		maxBytes: int64(maxBytes.Bytes()),
		session:  session,
		uploader: uploader,
		bucket:   s3Bucket,
	}, nil
}

// MaxBytes gets the max number of bytes that can be uploaded at once
func (p *Provider) MaxBytes() int64 {
	return p.maxBytes
}

// UploadFormMultipart uploads an image to S3
// that is sent via a multipart request body,
// returning the URL of the file once uploaded
func (p *Provider) UploadFormMultipart(ctx context.Context, part *multipart.Part, ext string) (string, error) {
	// Generate the filename using a random ID
	fileID, err := ksuid.NewRandom()
	if err != nil {
		return "", err
	}
	fileName := fmt.Sprintf("%s.%s", fileID, strings.TrimPrefix(ext, "."))
	log.Printf("uploading file '%s'\n", fileName)

	// Upload the file to S3
	result, err := p.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(fileName),
		Body:   part,
	})
	if err != nil {
		return "", err
	}

	// Return the URL of the object once uploaded
	return result.Location, nil
}
