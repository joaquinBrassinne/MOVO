package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Environment variables understood by LoadStorage.
const (
	envS3Endpoint     = "S3_ENDPOINT"
	envS3Region       = "S3_REGION"
	envS3Bucket       = "S3_BUCKET"
	envS3AccessKey    = "S3_ACCESS_KEY"
	envS3SecretKey    = "S3_SECRET_KEY"
	envS3PathStyle    = "S3_PATH_STYLE"
	envS3PresignTTL   = "S3_PRESIGN_TTL"
	envS3EnsureBucket = "S3_ENSURE_BUCKET"
)

const (
	defaultS3Region     = "us-east-1"
	defaultS3PresignTTL = 10 * time.Minute
)

// Storage holds the settings of the S3-compatible object storage. Only the API
// needs them, so they load apart from Config, like Auth.
type Storage struct {
	// Endpoint is the server URL. Empty means AWS S3 itself.
	Endpoint string
	// Region is the signing region; R2 and local servers accept any value.
	Region string
	// Bucket holds every object. It is required.
	Bucket string
	// AccessKey and SecretKey are the credentials. Both are required.
	AccessKey string
	SecretKey string
	// PathStyle addresses buckets as host/bucket instead of bucket.host. R2, B2
	// and local servers need it.
	PathStyle bool
	// PresignTTL is how long a presigned URL stays valid.
	PresignTTL time.Duration
	// EnsureBucket creates the bucket at startup when it does not exist. It is for
	// the local development server only: production buckets come with the service.
	EnsureBucket bool
}

// LoadStorage builds a Storage from get, normally os.Getenv.
func LoadStorage(get func(string) string) (Storage, error) {
	if get == nil {
		return Storage{}, errors.New("config: environment lookup must not be nil")
	}

	required := map[string]string{}
	for _, name := range []string{envS3Bucket, envS3AccessKey, envS3SecretKey} {
		value := strings.TrimSpace(get(name))
		if value == "" {
			return Storage{}, fmt.Errorf("config: %s is required and must not be empty", name)
		}
		required[name] = value
	}

	endpoint := strings.TrimSpace(get(envS3Endpoint))
	if endpoint != "" {
		if parsed, err := url.Parse(endpoint); err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return Storage{}, fmt.Errorf("config: %s %q must be an http(s) URL", envS3Endpoint, endpoint)
		}
	}

	pathStyle, err := optionalBool(get, envS3PathStyle)
	if err != nil {
		return Storage{}, err
	}
	ensureBucket, err := optionalBool(get, envS3EnsureBucket)
	if err != nil {
		return Storage{}, err
	}

	ttl, err := parsePositiveDuration(envS3PresignTTL, get(envS3PresignTTL), defaultS3PresignTTL)
	if err != nil {
		return Storage{}, err
	}

	return Storage{
		Endpoint:   endpoint,
		Region:     valueOrDefault(strings.TrimSpace(get(envS3Region)), defaultS3Region),
		Bucket:     required[envS3Bucket],
		AccessKey:  required[envS3AccessKey],
		SecretKey:  required[envS3SecretKey],
		PathStyle:  pathStyle,
		PresignTTL: ttl,

		EnsureBucket: ensureBucket,
	}, nil
}

// optionalBool reads a true/false variable; empty means false.
func optionalBool(get func(string) string, name string) (bool, error) {
	raw := strings.TrimSpace(get(name))
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("config: %s %q must be true or false", name, raw)
	}
	return value, nil
}
