package aws

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	transport "github.com/aws/smithy-go/endpoints"
)

type Config struct {
	AccessKey string
	SecretKey string
	Region    string
	Bucket    string

	// Optional: used when connecting to S3-compatible endpoints like Cloudflare R2, MinIO, etc.
	Endpoint     string // e.g., "https://<account_id>.r2.cloudflarestorage.com"
	UsePathStyle bool   // true for R2 and other non-AWS S3 services
}

func (a Config) Validate() error {
	if a.AccessKey == "" || a.SecretKey == "" || a.Bucket == "" {
		return fmt.Errorf("AWS credentials are incomplete")
	}
	// Region is not required for R2, so skip enforcing it if endpoint is provided
	if a.Region == "" && a.Endpoint == "" {
		return fmt.Errorf("missing region or custom endpoint")
	}
	return nil
}

type FileInfo struct {
	Size         int64
	LastModified time.Time
	ETag         string
	ContentType  string
}

// customEndpointResolver implements s3.EndpointResolverV2
type customEndpointResolver struct {
	rawURL string
}

func (r *customEndpointResolver) ResolveEndpoint(ctx context.Context, params s3.EndpointParameters) (transport.Endpoint, error) {
	parsed, err := url.Parse(r.rawURL)
	if err != nil {
		return transport.Endpoint{}, err
	}

	return transport.Endpoint{
		URI:     *parsed,
		Headers: http.Header{
			// You can set special headers if needed (e.g., for custom auth)
			// but usually, this can be left empty for S3-compatible storage like R2
		},
		Properties: smithy.Properties{},
	}, nil
}
