package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/joaquinBrassinne/MOVO/internal/platform/config"
)

// S3 implements Storage over the S3 API. Endpoint, region and path-style
// addressing come from configuration, so the same code serves RustFS in
// development and R2, B2 or AWS S3 in production.
type S3 struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
	ttl     func(*s3.PresignOptions)
}

// NewS3 builds the adapter from the storage configuration. It makes no network
// call; use EnsureBucket for that.
func NewS3(cfg config.Storage) *S3 {
	client := s3.New(s3.Options{
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		BaseEndpoint: endpointOrNil(cfg.Endpoint),
		UsePathStyle: cfg.PathStyle,
		// Compatible servers do not all implement the optional checksum headers
		// newer SDK versions add by default.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})

	return &S3{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  cfg.Bucket,
		ttl:     func(o *s3.PresignOptions) { o.Expires = cfg.PresignTTL },
	}
}

func endpointOrNil(endpoint string) *string {
	if endpoint == "" {
		return nil
	}
	return aws.String(endpoint)
}

// EnsureBucket creates the bucket when it does not exist. It is meant for the
// local development server; production buckets are provisioned with the service.
func (s *S3) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.bucket})
	if err == nil {
		return nil
	}

	if _, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &s.bucket}); err != nil {
		var exists *types.BucketAlreadyOwnedByYou
		if errors.As(err, &exists) {
			return nil
		}
		return fmt.Errorf("storage: create bucket %q: %w", s.bucket, err)
	}
	return nil
}

// PresignUpload validates req and signs a PUT whose content type and length are
// part of the signature: a body of another type or size is rejected by the server.
func (s *S3) PresignUpload(ctx context.Context, req UploadRequest) (PresignedUpload, error) {
	if err := ValidateImageUpload(req); err != nil {
		return PresignedUpload{}, err
	}

	signed, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &req.Key,
		ContentType:   &req.ContentType,
		ContentLength: aws.Int64(req.Size),
	}, s.ttl)
	if err != nil {
		return PresignedUpload{}, fmt.Errorf("storage: presign upload: %w", err)
	}

	return PresignedUpload{
		URL:     signed.URL,
		Method:  http.MethodPut,
		Headers: map[string]string{"Content-Type": req.ContentType},
	}, nil
}

// PresignDownload signs a GET for the object.
func (s *S3) PresignDownload(ctx context.Context, key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}

	signed, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key}, s.ttl)
	if err != nil {
		return "", fmt.Errorf("storage: presign download: %w", err)
	}
	return signed.URL, nil
}

// Delete removes the object. S3 already answers success for a missing key, and a
// NoSuchKey from a compatible server is treated the same way.
func (s *S3) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key}); err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey" {
			return nil
		}
		return fmt.Errorf("storage: delete object: %w", err)
	}
	return nil
}
