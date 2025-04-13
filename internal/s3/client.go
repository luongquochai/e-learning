package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/luongquochai/s3-performance-test/internal/config"
)

// Client represents an S3 client for interacting with AWS S3
type Client struct {
	s3Client      *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	region        string
}

// NewClient creates a new S3 client
func NewClient(cfg *config.Config) (*Client, error) {
	// Load AWS configuration
	var awsConfig aws.Config
	var err error

	// Set up options with region
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWS.Region),
	}

	// If custom endpoint is provided (for S3-compatible services like MinIO)
	if cfg.AWS.Endpoint != "" {
		opts = append(opts, awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               cfg.AWS.Endpoint,
						SigningRegion:     cfg.AWS.Region,
						HostnameImmutable: true,
					}, nil
				},
			),
		))
	}

	// If access keys are provided in the config, use them
	if cfg.AWS.AccessKey != "" && cfg.AWS.SecretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AWS.AccessKey,
				cfg.AWS.SecretKey,
				"",
			),
		))
	}

	// Load the AWS configuration
	awsConfig, err = awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create the S3 client
	s3Client := s3.NewFromConfig(awsConfig)
	presignClient := s3.NewPresignClient(s3Client)

	return &Client{
		s3Client:      s3Client,
		presignClient: presignClient,
		bucket:        cfg.AWS.Bucket,
		region:        cfg.AWS.Region,
	}, nil
}

// BucketExists checks if the configured bucket exists
func (c *Client) BucketExists(ctx context.Context) (bool, error) {
	_, err := c.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// GeneratePresignedURL generates a presigned URL for S3 object upload (PUT)
func (c *Client) GeneratePresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	presignedReq, err := c.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedReq.URL, nil
}

// GenerateMultiplePresignedURLs generates multiple presigned URLs at once
func (c *Client) GenerateMultiplePresignedURLs(ctx context.Context, keyPrefix string, count int, expires time.Duration) (map[string]string, error) {
	results := make(map[string]string, count)
	timeNow := time.Now().UTC().Format("20060102-150405")

	for i := 0; i < count; i++ {
		key := fmt.Sprintf("%s/%s-%03d.dat", keyPrefix, timeNow, i)
		url, err := c.GeneratePresignedURL(ctx, key, expires)
		if err != nil {
			return nil, fmt.Errorf("failed to generate presigned URL for key %s: %w", key, err)
		}
		results[key] = url
	}

	return results, nil
}

// DeleteObject deletes an object from the S3 bucket
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", key, err)
	}
	return nil
}

// ListObjects lists objects in the S3 bucket with the given prefix
func (c *Client) ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(c.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(maxKeys),
	}

	resp, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects with prefix %s: %w", prefix, err)
	}

	keys := make([]string, 0, len(resp.Contents))
	for _, obj := range resp.Contents {
		keys = append(keys, *obj.Key)
	}

	return keys, nil
}
