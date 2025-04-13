# S3 Performance Test Tool

A robust Go-based utility for testing AWS S3 upload performance and reliability using presigned URLs.

## 🌟 Features

- 🚀 Generate **presigned S3 URLs** for secure temporary upload access
- 📤 Upload files of configurable sizes (optimized for files under 1MB)
- 🕒 **Schedule tests** to run hourly, daily, or weekly via cron
- 📊 Run **concurrent performance tests with K6**
- 🗂 Automatically generate **random test files** with configurable sizes and content patterns
- 📈 Collect comprehensive metrics including latency, throughput, and error rates
- 📝 Detailed logging with configurable log levels
- ⚙️ Flexible configuration via environment variables or YAML
- 🌐 **API server** for generating presigned URLs on demand

## 📋 Prerequisites

- Go 1.18+
- AWS Account with S3 access
- K6 for performance testing (`brew install k6` on macOS)
- AWS credentials configured

## 🚀 Installation

```bash
# Clone the repository
git clone https://github.com/luongquochai/s3-performance-test
cd s3-performance-test

# Install dependencies
go mod tidy

# Build the binary
go build -o s3perftest ./cmd/s3perftest
```

## ⚙️ Configuration

Create a `config.yaml` file in the `config` directory or use environment variables:

```yaml
aws:
  region: us-west-2
  bucket: your-s3-bucket-name
  
test:
  file_size: 1048576  # 1MB in bytes
  file_count: 100
  concurrent_users: 10
  duration: 5m
  file_pattern: random  # random, zeros, or repeating
  
schedule:
  enabled: false
  cron: "0 * * * *"  # Run hourly
  
logging:
  level: info
  file: logs/s3perftest.log
  
api:
  port: 8080
  host: "0.0.0.0"
  enable_cors: true
  allow_origins: "*"
```

## 📊 Usage

### Run a direct performance test (file-based)

This approach generates files and presigned URLs, then runs the test in one step:

```bash
./s3perftest run --config=config/config.yaml
```

### Start the API server

Run the API server that provides presigned URLs on demand:

```bash
./s3perftest serve
```

The API server exposes the following endpoints:
- `GET /health` - Health check endpoint
- `GET /api/v1/presigned-url` - Get a single presigned URL
  - Query parameters: `expiry` (in seconds), `key` (optional)
- `GET /api/v1/presigned-urls` - Get multiple presigned URLs
  - Query parameters: `count`, `expiry` (in seconds)

### Run tests against the API server

Run performance tests against the running API server:

```bash
./s3perftest test-api --api-url=http://localhost:8080
```

### Generate presigned URLs only

```bash
./s3perftest generate-urls --count=10 --expiry=3600
```

### Schedule recurring tests

```bash
./s3perftest schedule --config=config/config.yaml
```

## 🧪 Key Components

1. **S3 Client**: Handles generation of presigned URLs and S3 operations
2. **File Generator**: Creates random test files of specified sizes
3. **K6 Integration**: Runs concurrent performance tests
4. **Scheduler**: Manages cron-based test scheduling
5. **Config Manager**: Handles configuration from files and environment variables
6. **API Server**: Provides HTTP endpoints for generating presigned URLs

## 📝 License

MIT

## 🔗 Related Resources

- [AWS S3 Documentation](https://docs.aws.amazon.com/s3/)
- [K6 Documentation](https://k6.io/docs/)

---
