package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	AWS      AWSConfig      `yaml:"aws"`
	Test     TestConfig     `yaml:"test"`
	Schedule ScheduleConfig `yaml:"schedule"`
	Logging  LoggingConfig  `yaml:"logging"`
	API      APIConfig      `yaml:"api"`
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	Region    string `yaml:"region"`
	Bucket    string `yaml:"bucket"`
	Endpoint  string `yaml:"endpoint,omitempty"` // For S3-compatible services
	AccessKey string `yaml:"access_key,omitempty"`
	SecretKey string `yaml:"secret_key,omitempty"`
}

// TestConfig holds test-specific configuration
type TestConfig struct {
	FileSize        int64         `yaml:"file_size"`        // Size in bytes
	FileCount       int           `yaml:"file_count"`       // Number of files to generate
	ConcurrentUsers int           `yaml:"concurrent_users"` // Number of concurrent users in tests
	Duration        time.Duration `yaml:"duration"`         // Test duration as string (e.g., "5m")
	FilePattern     string        `yaml:"file_pattern"`     // Pattern for file content (random, zeros, repeating)
}

// ScheduleConfig holds scheduler-specific configuration
type ScheduleConfig struct {
	Enabled bool   `yaml:"enabled"`
	Cron    string `yaml:"cron"` // Cron expression (e.g., "0 * * * *")
}

// LoggingConfig holds logging-specific configuration
type LoggingConfig struct {
	Level string `yaml:"level"` // debug, info, warn, error
	File  string `yaml:"file"`  // Log file path
}

// APIConfig holds API server configuration
type APIConfig struct {
	Port         int    `yaml:"port"`          // Port to listen on
	Host         string `yaml:"host"`          // Host to bind to
	EnableCORS   bool   `yaml:"enable_cors"`   // Whether to enable CORS
	AllowOrigins string `yaml:"allow_origins"` // Comma-separated list of allowed origins for CORS
}

// LoadConfig loads the configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	// Default configuration
	config := &Config{
		AWS: AWSConfig{
			Region: "us-west-2",
		},
		Test: TestConfig{
			FileSize:        1048576, // 1MB
			FileCount:       100,
			ConcurrentUsers: 10,
			Duration:        5 * time.Minute,
			FilePattern:     "random",
		},
		Schedule: ScheduleConfig{
			Enabled: false,
			Cron:    "0 * * * *", // Hourly
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  "logs/s3perftest.log",
		},
		API: APIConfig{
			Port:         8080,
			Host:         "0.0.0.0",
			EnableCORS:   true,
			AllowOrigins: "*",
		},
	}

	// Load from config file if provided
	if configPath != "" {
		if err := loadFromFile(configPath, config); err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	applyEnvironmentVariables(config)

	// Validate the configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(configPath string, config *Config) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("error parsing config file: %w", err)
	}

	return nil
}

// applyEnvironmentVariables overrides config with environment variables
func applyEnvironmentVariables(config *Config) {
	// AWS Config
	if region := os.Getenv("AWS_REGION"); region != "" {
		config.AWS.Region = region
	}
	if bucket := os.Getenv("AWS_S3_BUCKET"); bucket != "" {
		config.AWS.Bucket = bucket
	}
	if endpoint := os.Getenv("AWS_S3_ENDPOINT"); endpoint != "" {
		config.AWS.Endpoint = endpoint
	}
	if accessKey := os.Getenv("AWS_ACCESS_KEY_ID"); accessKey != "" {
		config.AWS.AccessKey = accessKey
	}
	if secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY"); secretKey != "" {
		config.AWS.SecretKey = secretKey
	}

	// Test Config
	if fileSize := os.Getenv("TEST_FILE_SIZE"); fileSize != "" {
		if size, err := strconv.ParseInt(fileSize, 10, 64); err == nil {
			config.Test.FileSize = size
		}
	}
	if fileCount := os.Getenv("TEST_FILE_COUNT"); fileCount != "" {
		if count, err := strconv.Atoi(fileCount); err == nil {
			config.Test.FileCount = count
		}
	}
	if concurrentUsers := os.Getenv("TEST_CONCURRENT_USERS"); concurrentUsers != "" {
		if users, err := strconv.Atoi(concurrentUsers); err == nil {
			config.Test.ConcurrentUsers = users
		}
	}
	if duration := os.Getenv("TEST_DURATION"); duration != "" {
		if d, err := time.ParseDuration(duration); err == nil {
			config.Test.Duration = d
		}
	}
	if filePattern := os.Getenv("TEST_FILE_PATTERN"); filePattern != "" {
		config.Test.FilePattern = filePattern
	}

	// Schedule Config
	if enabled := os.Getenv("SCHEDULE_ENABLED"); enabled != "" {
		config.Schedule.Enabled = strings.ToLower(enabled) == "true"
	}
	if cron := os.Getenv("SCHEDULE_CRON"); cron != "" {
		config.Schedule.Cron = cron
	}

	// Logging Config
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if file := os.Getenv("LOG_FILE"); file != "" {
		config.Logging.File = file
	}

	// API Config
	if port := os.Getenv("API_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.API.Port = p
		}
	}
	if host := os.Getenv("API_HOST"); host != "" {
		config.API.Host = host
	}
	if cors := os.Getenv("API_ENABLE_CORS"); cors != "" {
		config.API.EnableCORS = strings.ToLower(cors) == "true"
	}
	if origins := os.Getenv("API_ALLOW_ORIGINS"); origins != "" {
		config.API.AllowOrigins = origins
	}
}

// validateConfig ensures the configuration is valid
func validateConfig(config *Config) error {
	if config.AWS.Bucket == "" {
		return fmt.Errorf("AWS S3 bucket name is required")
	}
	if config.AWS.Region == "" {
		return fmt.Errorf("AWS region is required")
	}
	if config.Test.FileSize <= 0 {
		return fmt.Errorf("file size must be greater than 0")
	}
	if config.Test.FileCount <= 0 {
		return fmt.Errorf("file count must be greater than 0")
	}
	if config.Test.ConcurrentUsers <= 0 {
		return fmt.Errorf("concurrent users must be greater than 0")
	}
	if config.Test.Duration <= 0 {
		return fmt.Errorf("test duration must be greater than 0")
	}
	if config.Schedule.Enabled && config.Schedule.Cron == "" {
		return fmt.Errorf("cron expression is required when scheduling is enabled")
	}

	// Validate log level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[strings.ToLower(config.Logging.Level)] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	// Validate file pattern
	validPatterns := map[string]bool{"random": true, "zeros": true, "repeating": true}
	if !validPatterns[strings.ToLower(config.Test.FilePattern)] {
		return fmt.Errorf("invalid file pattern: %s", config.Test.FilePattern)
	}

	// Validate API port
	if config.API.Port <= 0 || config.API.Port > 65535 {
		return fmt.Errorf("invalid API port: %d", config.API.Port)
	}

	return nil
}
