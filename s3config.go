package virefs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Provider identifies an S3-compatible storage provider.
// Each provider may have different defaults (e.g. path-style addressing).
type Provider int

const (
	// ProviderAWS is standard AWS S3 (virtual-hosted-style, default region us-east-1).
	ProviderAWS Provider = iota
	// ProviderMinIO requires path-style addressing.
	ProviderMinIO
	// ProviderR2 is Cloudflare R2, which requires path-style addressing
	// and region "auto".
	ProviderR2
)

// S3Config holds the parameters needed to construct an S3 client and,
// optionally, an ObjectFS in one step.
type S3Config struct {
	// Region is the AWS region (e.g. "us-east-1").
	// For R2 this defaults to "auto" if left empty.
	Region string

	// Endpoint is the custom S3-compatible endpoint URL.
	// Required for MinIO, R2, and other non-AWS providers.
	// Leave empty for standard AWS S3.
	Endpoint string

	// Bucket is the target bucket name.
	// Used by NewObjectFSFromConfig; ignored by NewS3Client.
	Bucket string

	// AccessKey and SecretKey provide static credentials.
	// When both are empty, the SDK's default credential chain is used
	// (env vars, shared config, IAM role, etc.).
	AccessKey string
	SecretKey string

	// Provider selects a provider preset that configures known quirks.
	// Default is ProviderAWS.
	Provider Provider

	// UsePathStyle forces path-style addressing (e.g. http://endpoint/bucket/key).
	// Automatically set to true for MinIO and R2 providers; set explicitly
	// to override the provider default.
	UsePathStyle *bool
}

// NewS3Client creates an *s3.Client from the given S3Config.
// Provider-specific quirks (path style, default region) are applied
// automatically. The caller's S3Config is not modified.
//
// For ProviderMinIO, request checksum calculation is set to
// aws.RequestChecksumCalculationWhenRequired to avoid compatibility issues with
// aws-chunked uploads on large objects.
func NewS3Client(ctx context.Context, cfg *S3Config) (*s3.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("virefs: S3Config must not be nil")
	}
	resolved := *cfg
	applyProviderDefaults(&resolved)

	var loadOpts []func(*config.LoadOptions) error

	if resolved.Region != "" {
		loadOpts = append(loadOpts, config.WithRegion(resolved.Region))
	}

	if resolved.AccessKey != "" && resolved.SecretKey != "" {
		loadOpts = append(loadOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(resolved.AccessKey, resolved.SecretKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("virefs: load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if resolved.Endpoint != "" {
			o.BaseEndpoint = aws.String(resolved.Endpoint)
		}
		if resolved.UsePathStyle != nil && *resolved.UsePathStyle {
			o.UsePathStyle = true
		}
		if resolved.Provider == ProviderMinIO {
			o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		}
	})

	return client, nil
}

// NewObjectFSFromConfig creates an ObjectFS (with a presign client) in
// one step from an S3Config. Any additional ObjectOption values are
// applied after the presign client is injected.
//
//	fs, err := virefs.NewObjectFSFromConfig(ctx, &virefs.S3Config{
//	    Provider:  virefs.ProviderMinIO,
//	    Endpoint:  "http://localhost:9000",
//	    Region:    "us-east-1",
//	    AccessKey: "minioadmin",
//	    SecretKey: "minioadmin",
//	    Bucket:    "my-bucket",
//	}, virefs.WithPrefix("uploads/"))
func NewObjectFSFromConfig(ctx context.Context, cfg *S3Config, opts ...ObjectOption) (*ObjectFS, error) {
	if cfg == nil {
		return nil, fmt.Errorf("virefs: S3Config must not be nil")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("virefs: S3Config.Bucket must not be empty")
	}

	client, err := NewS3Client(ctx, cfg)
	if err != nil {
		return nil, err
	}

	presignClient := s3.NewPresignClient(client)

	allOpts := make([]ObjectOption, 0, len(opts)+1)
	allOpts = append(allOpts, WithPresignClient(presignClient))
	allOpts = append(allOpts, opts...)

	return NewObjectFS(client, cfg.Bucket, allOpts...), nil
}

func applyProviderDefaults(cfg *S3Config) {
	switch cfg.Provider {
	case ProviderMinIO:
		if cfg.UsePathStyle == nil {
			cfg.UsePathStyle = aws.Bool(true)
		}
	case ProviderR2:
		if cfg.UsePathStyle == nil {
			cfg.UsePathStyle = aws.Bool(true)
		}
		if cfg.Region == "" {
			cfg.Region = "auto"
		}
	case ProviderAWS:
		if cfg.Region == "" {
			cfg.Region = "us-east-1"
		}
	}
}
