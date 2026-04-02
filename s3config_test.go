package virefs

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestApplyProviderDefaults_AWS(t *testing.T) {
	cfg := S3Config{Provider: ProviderAWS}
	applyProviderDefaults(&cfg)
	if cfg.Region != "us-east-1" {
		t.Fatalf("AWS default region = %q, want %q", cfg.Region, "us-east-1")
	}
	if cfg.UsePathStyle != nil {
		t.Fatal("AWS should not force path style")
	}
}

func TestApplyProviderDefaults_AWS_RegionPreserved(t *testing.T) {
	cfg := S3Config{Provider: ProviderAWS, Region: "eu-west-1"}
	applyProviderDefaults(&cfg)
	if cfg.Region != "eu-west-1" {
		t.Fatalf("region = %q, want %q", cfg.Region, "eu-west-1")
	}
}

func TestApplyProviderDefaults_MinIO(t *testing.T) {
	cfg := S3Config{Provider: ProviderMinIO}
	applyProviderDefaults(&cfg)
	if cfg.UsePathStyle == nil || !*cfg.UsePathStyle {
		t.Fatal("MinIO should default to path style")
	}
}

func TestApplyProviderDefaults_MinIO_PathStyleOverride(t *testing.T) {
	cfg := S3Config{
		Provider:     ProviderMinIO,
		UsePathStyle: aws.Bool(false),
	}
	applyProviderDefaults(&cfg)
	if *cfg.UsePathStyle != false {
		t.Fatal("explicit UsePathStyle=false should be preserved")
	}
}

func TestApplyProviderDefaults_R2(t *testing.T) {
	cfg := S3Config{Provider: ProviderR2}
	applyProviderDefaults(&cfg)
	if cfg.Region != "auto" {
		t.Fatalf("R2 default region = %q, want %q", cfg.Region, "auto")
	}
	if cfg.UsePathStyle == nil || !*cfg.UsePathStyle {
		t.Fatal("R2 should default to path style")
	}
}

func TestApplyProviderDefaults_R2_RegionPreserved(t *testing.T) {
	cfg := S3Config{Provider: ProviderR2, Region: "wnam"}
	applyProviderDefaults(&cfg)
	if cfg.Region != "wnam" {
		t.Fatalf("region = %q, want %q", cfg.Region, "wnam")
	}
}

func TestS3Config_Validation(t *testing.T) {
	_, err := NewObjectFSFromConfig(t.Context(), &S3Config{})
	if err == nil {
		t.Fatal("NewObjectFSFromConfig with empty Bucket should fail")
	}
}

func TestNewS3Client_MinIO_RequestChecksumWhenRequired(t *testing.T) {
	client, err := NewS3Client(t.Context(), &S3Config{
		Provider:  ProviderMinIO,
		Region:    "us-east-1",
		Endpoint:  "http://localhost:9000",
		AccessKey: "x",
		SecretKey: "y",
	})
	if err != nil {
		t.Fatalf("NewS3Client failed: %v", err)
	}
	if got := client.Options().RequestChecksumCalculation; got != aws.RequestChecksumCalculationWhenRequired {
		t.Fatalf("RequestChecksumCalculation = %v, want %v", got, aws.RequestChecksumCalculationWhenRequired)
	}
}
