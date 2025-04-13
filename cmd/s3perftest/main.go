package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/luongquochai/s3-performance-test/internal/api"
	"github.com/luongquochai/s3-performance-test/internal/config"
	"github.com/luongquochai/s3-performance-test/internal/generator"
	"github.com/luongquochai/s3-performance-test/internal/s3"
	"github.com/luongquochai/s3-performance-test/internal/scheduler"
	"github.com/spf13/cobra"
)

var (
	configPath string
	verbose    bool
	cfg        *config.Config
)

// URLEntry represents a URL entry for K6 tests
type URLEntry struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

// TestData holds data for K6 tests
type TestData struct {
	URLs []URLEntry `json:"urls"`
}

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "s3perftest",
		Short: "S3 Performance Test Tool",
		Long:  "A tool for testing AWS S3 upload performance using presigned URLs",
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config/config.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add commands
	rootCmd.AddCommand(newRunCommand())
	rootCmd.AddCommand(newGenerateURLsCommand())
	rootCmd.AddCommand(newScheduleCommand())
	rootCmd.AddCommand(newServeCommand())
	rootCmd.AddCommand(newTestAPICommand())

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// loadConfig loads the configuration from file
func loadConfig() (*config.Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	var err error
	cfg, err = config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return cfg, nil
}

// newRunCommand creates the 'run' command for running performance tests
func newRunCommand() *cobra.Command {
	var (
		filePattern string
		fileSize    int64
		fileCount   int
		duration    string
		concurrent  int
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a performance test",
		Long:  "Run a performance test against AWS S3 using presigned URLs",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// Override config with command-line flags if provided
			if cmd.Flags().Changed("file-pattern") {
				cfg.Test.FilePattern = filePattern
			}
			if cmd.Flags().Changed("file-size") {
				cfg.Test.FileSize = fileSize
			}
			if cmd.Flags().Changed("file-count") {
				cfg.Test.FileCount = fileCount
			}
			if cmd.Flags().Changed("duration") {
				parsedDuration, err := time.ParseDuration(duration)
				if err != nil {
					return fmt.Errorf("invalid duration format: %w", err)
				}
				cfg.Test.Duration = parsedDuration
			}
			if cmd.Flags().Changed("concurrent") {
				cfg.Test.ConcurrentUsers = concurrent
			}

			// Create a context that can be canceled
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle interrupt signals
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-signalChan
				log.Println("Received interrupt signal, shutting down...")
				cancel()
			}()

			// Run the test
			return runPerformanceTest(ctx, cfg)
		},
	}

	// Add command-specific flags
	cmd.Flags().StringVar(&filePattern, "file-pattern", "random", "Pattern for test files (random, zeros, repeating)")
	cmd.Flags().Int64Var(&fileSize, "file-size", 1048576, "Size of test files in bytes")
	cmd.Flags().IntVar(&fileCount, "file-count", 100, "Number of test files to generate")
	cmd.Flags().StringVar(&duration, "duration", "5m", "Test duration (e.g., '5m', '30s')")
	cmd.Flags().IntVar(&concurrent, "concurrent", 10, "Number of concurrent users")

	return cmd
}

// newGenerateURLsCommand creates the 'generate-urls' command for generating presigned URLs
func newGenerateURLsCommand() *cobra.Command {
	var (
		count  int
		expiry int64
		output string
	)

	cmd := &cobra.Command{
		Use:   "generate-urls",
		Short: "Generate presigned URLs",
		Long:  "Generate presigned URLs for S3 uploads",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// Create a context
			ctx := context.Background()

			// Create S3 client
			s3Client, err := s3.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to create S3 client: %w", err)
			}

			// Check if the bucket exists
			exists, err := s3Client.BucketExists(ctx)
			if err != nil {
				return fmt.Errorf("failed to check if bucket exists: %w", err)
			}
			if !exists {
				return fmt.Errorf("bucket %s does not exist", cfg.AWS.Bucket)
			}

			// Generate presigned URLs
			log.Printf("Generating %d presigned URLs with expiry of %d seconds", count, expiry)
			urls, err := s3Client.GenerateMultiplePresignedURLs(ctx, "test", count, time.Duration(expiry)*time.Second)
			if err != nil {
				return fmt.Errorf("failed to generate presigned URLs: %w", err)
			}

			// Convert to URL entries
			urlEntries := make([]URLEntry, 0, len(urls))
			for key, url := range urls {
				urlEntries = append(urlEntries, URLEntry{
					URL: url,
					Key: key,
				})
			}

			// Create test data
			testData := TestData{
				URLs: urlEntries,
			}

			// Marshal to JSON
			jsonData, err := json.MarshalIndent(testData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal URLs to JSON: %w", err)
			}

			// Write to file or stdout
			if output == "-" {
				fmt.Println(string(jsonData))
			} else {
				if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
					return fmt.Errorf("failed to create output directory: %w", err)
				}
				if err := os.WriteFile(output, jsonData, 0644); err != nil {
					return fmt.Errorf("failed to write URLs to file: %w", err)
				}
				log.Printf("Wrote %d presigned URLs to %s", count, output)
			}

			return nil
		},
	}

	// Add command-specific flags
	cmd.Flags().IntVar(&count, "count", 100, "Number of presigned URLs to generate")
	cmd.Flags().Int64Var(&expiry, "expiry", 3600, "Expiry time in seconds")
	cmd.Flags().StringVar(&output, "output", "k6/scripts/test-data.json", "Output file (use '-' for stdout)")

	return cmd
}

// newScheduleCommand creates the 'schedule' command for scheduling tests
func newScheduleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Schedule recurring tests",
		Long:  "Schedule recurring performance tests using a cron expression",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// Check if scheduling is enabled
			if !cfg.Schedule.Enabled {
				return fmt.Errorf("scheduling is not enabled in configuration")
			}

			// Create a context that can be canceled
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create a scheduler
			scheduler := scheduler.NewScheduler(cfg, func(ctx context.Context) error {
				return runPerformanceTest(ctx, cfg)
			})

			// Start the scheduler
			if err := scheduler.Start(ctx); err != nil {
				return fmt.Errorf("failed to start scheduler: %w", err)
			}

			// Print next run time
			nextRun, err := scheduler.NextRun()
			if err != nil {
				log.Printf("Warning: Failed to get next run time: %v", err)
			} else {
				log.Printf("Next test will run at: %s", nextRun.Format(time.RFC3339))
			}

			// Handle interrupt signals
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
			log.Println("Scheduler is running. Press Ctrl+C to stop.")
			<-signalChan
			log.Println("Received interrupt signal, shutting down...")

			// Stop the scheduler
			scheduler.Stop()
			return nil
		},
	}

	return cmd
}

// newServeCommand creates the 'serve' command for starting the API server
func newServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the API server",
		Long:  "Start the API server for generating presigned URLs and running performance tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// Create S3 client
			s3Client, err := s3.NewClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to create S3 client: %w", err)
			}

			// Check if the bucket exists
			exists, err := s3Client.BucketExists(context.Background())
			if err != nil {
				return fmt.Errorf("failed to check if bucket exists: %w", err)
			}
			if !exists {
				return fmt.Errorf("bucket %s does not exist", cfg.AWS.Bucket)
			}

			// Create and initialize API server
			apiServer := api.NewServer(cfg, s3Client)

			// Channel to listen for shutdown signals
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

			// Start the server in a goroutine
			go func() {
				log.Printf("Starting API server on %s:%d", cfg.API.Host, cfg.API.Port)
				if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
					log.Fatalf("API server failed: %v", err)
				}
			}()

			// Wait for shutdown signal
			<-quit
			log.Println("Shutting down API server...")

			// Create a deadline for server shutdown
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Attempt graceful shutdown
			if err := apiServer.Shutdown(ctx); err != nil {
				return fmt.Errorf("server shutdown failed: %w", err)
			}

			log.Println("Server shutdown complete")
			return nil
		},
	}

	return cmd
}

// newTestAPICommand creates the 'test-api' command for running performance tests against the API
func newTestAPICommand() *cobra.Command {
	var (
		apiURL     string
		duration   string
		concurrent int
		fileSize   int64
	)

	cmd := &cobra.Command{
		Use:   "test-api",
		Short: "Test the API performance",
		Long:  "Run a performance test against the API server for generating presigned URLs",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			// Override config with command-line flags if provided
			if cmd.Flags().Changed("duration") {
				parsedDuration, err := time.ParseDuration(duration)
				if err != nil {
					return fmt.Errorf("invalid duration format: %w", err)
				}
				cfg.Test.Duration = parsedDuration
			}
			if cmd.Flags().Changed("concurrent") {
				cfg.Test.ConcurrentUsers = concurrent
			}
			if cmd.Flags().Changed("file-size") {
				cfg.Test.FileSize = fileSize
			}

			// Use the provided API URL or build one from config
			apiBaseURL := apiURL
			if apiBaseURL == "" {
				apiBaseURL = fmt.Sprintf("http://%s:%d", cfg.API.Host, cfg.API.Port)
			}

			// Convert duration to seconds for K6
			durationSeconds := int(cfg.Test.Duration.Seconds())

			// Create a temporary directory for test data
			tempDir, err := os.MkdirTemp("", "s3perftest")
			if err != nil {
				return fmt.Errorf("failed to create temp directory: %w", err)
			}
			defer os.RemoveAll(tempDir)

			// Build the K6 command with API test script
			k6Cmd := fmt.Sprintf("k6 run "+
				"--env API_BASE_URL=%s "+
				"--env CONCURRENT_USERS=%d "+
				"--env TEST_DURATION_SECONDS=%d "+
				"--env FILE_SIZE=%d "+
				"--env VERBOSE=%t "+
				"k6/scripts/api-upload-test.js",
				apiBaseURL, cfg.Test.ConcurrentUsers, durationSeconds, cfg.Test.FileSize, verbose)

			// Execute the K6 command
			log.Printf("Running API performance test with %d concurrent users for %s...", cfg.Test.ConcurrentUsers, cfg.Test.Duration)
			log.Printf("API Base URL: %s", apiBaseURL)
			log.Printf("Executing: %s", k6Cmd)

			execCmd := exec.Command("sh", "-c", k6Cmd)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr

			if err := execCmd.Start(); err != nil {
				return fmt.Errorf("failed to start K6 test: %w", err)
			}

			// Wait for the command to finish
			if err := execCmd.Wait(); err != nil {
				return fmt.Errorf("K6 test failed: %w", err)
			}

			log.Println("API performance test completed successfully")
			return nil
		},
	}

	// Add command-specific flags
	cmd.Flags().StringVar(&apiURL, "api-url", "", "Base URL of the API server (default: http://host:port from config)")
	cmd.Flags().StringVar(&duration, "duration", "5m", "Test duration (e.g., '5m', '30s')")
	cmd.Flags().IntVar(&concurrent, "concurrent", 10, "Number of concurrent users")
	cmd.Flags().Int64Var(&fileSize, "file-size", 1048576, "Size of test files in bytes")

	return cmd
}

// runPerformanceTest runs a performance test
func runPerformanceTest(ctx context.Context, cfg *config.Config) error {
	log.Println("Starting performance test...")

	// Create S3 client
	s3Client, err := s3.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Check if the bucket exists
	exists, err := s3Client.BucketExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %s does not exist", cfg.AWS.Bucket)
	}

	// Create file generator with the specified file pattern
	var filePattern generator.FilePattern
	switch cfg.Test.FilePattern {
	case "random":
		filePattern = generator.RandomPattern
	case "zeros":
		filePattern = generator.ZeroPattern
	case "repeating":
		filePattern = generator.RepeatingPattern
	default:
		return fmt.Errorf("unsupported file pattern: %s", cfg.Test.FilePattern)
	}

	fileGen, err := generator.NewFileGenerator(cfg.Test.FileSize, cfg.Test.FileCount, filePattern)
	if err != nil {
		return fmt.Errorf("failed to create file generator: %w", err)
	}
	defer fileGen.CleanupFiles()

	// Generate test files
	log.Printf("Generating %d test files of %d bytes each...", cfg.Test.FileCount, cfg.Test.FileSize)
	filePaths, err := fileGen.GenerateFiles()
	if err != nil {
		return fmt.Errorf("failed to generate test files: %w", err)
	}
	log.Printf("Generated %d test files", len(filePaths))

	// Generate presigned URLs
	log.Println("Generating presigned URLs...")
	urls, err := s3Client.GenerateMultiplePresignedURLs(ctx, "test", cfg.Test.FileCount, 1*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to generate presigned URLs: %w", err)
	}
	log.Printf("Generated %d presigned URLs", len(urls))

	// Convert to URL entries for K6
	urlEntries := make([]URLEntry, 0, len(urls))
	for key, url := range urls {
		urlEntries = append(urlEntries, URLEntry{
			URL: url,
			Key: key,
		})
	}

	// Create test data
	testData := TestData{
		URLs: urlEntries,
	}

	// Create a temporary directory for test data
	tempDir, err := os.MkdirTemp("", "s3perftest")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write test data to file
	testDataPath := filepath.Join(tempDir, "test-data.json")
	jsonData, err := json.MarshalIndent(testData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal test data to JSON: %w", err)
	}
	if err := os.WriteFile(testDataPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write test data to file: %w", err)
	}

	// Run K6 test
	log.Printf("Running K6 performance test with %d concurrent users for %s...", cfg.Test.ConcurrentUsers, cfg.Test.Duration)

	// Convert duration to seconds for K6
	durationSeconds := int(cfg.Test.Duration.Seconds())

	// Build the K6 command
	k6Cmd := fmt.Sprintf("k6 run "+
		"--env TEST_DATA_FILE=%s "+
		"--env CONCURRENT_USERS=%d "+
		"--env TEST_DURATION_SECONDS=%d "+
		"--env FILE_SIZE=%d "+
		"--env VERBOSE=%t "+
		"k6/scripts/upload-test.js",
		testDataPath, cfg.Test.ConcurrentUsers, durationSeconds, cfg.Test.FileSize, verbose)

	// Execute the K6 command
	log.Printf("Executing: %s", k6Cmd)
	cmd := exec.Command("sh", "-c", k6Cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start K6 test: %w", err)
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("K6 test failed: %w", err)
	}

	log.Println("Performance test completed successfully")
	return nil
}
