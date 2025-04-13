package s3

import (
	"context"
	"testing"
	"time"

	"github.com/luongquochai/s3-performance-test/internal/config"
)

func TestGeneratePresignedURL(t *testing.T) {
	// Skip this test when running in CI or without AWS credentials
	t.Skip("Skipping test that requires AWS credentials")

	// Create a test config
	cfg := &config.Config{
		AWS: config.AWSConfig{
			Region: "us-west-2",
			Bucket: "test-bucket",
		},
	}

	// Create a mock client for testing
	client := &Client{
		bucket: cfg.AWS.Bucket,
		region: cfg.AWS.Region,
	}

	// Test generating a presigned URL
	_, err := client.GeneratePresignedURL(context.Background(), "test-key", 3600*time.Second)
	if err == nil {
		// We expect an error since we don't have a real S3 client
		t.Error("Expected error with mock client, but got none")
	}
}

func TestGenerateMultiplePresignedURLs(t *testing.T) {
	// Skip this test when running in CI or without AWS credentials
	t.Skip("Skipping test that requires AWS credentials")

	// Create a test config
	cfg := &config.Config{
		AWS: config.AWSConfig{
			Region: "us-west-2",
			Bucket: "test-bucket",
		},
	}

	// Create a mock client for testing
	client := &Client{
		bucket: cfg.AWS.Bucket,
		region: cfg.AWS.Region,
	}

	// Test generating multiple presigned URLs
	_, err := client.GenerateMultiplePresignedURLs(context.Background(), "test", 5, 3600*time.Second)
	if err == nil {
		// We expect an error since we don't have a real S3 client
		t.Error("Expected error with mock client, but got none")
	}
}
