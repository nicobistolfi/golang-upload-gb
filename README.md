# Upload GB

A simple HTTP file upload service built with Go and Gin, with performance logging and benchmarking.

## Overview

This service provides a single endpoint (`/upload`) that accepts file uploads via HTTP POST requests and saves them to a specified destination on the server filesystem. It includes detailed performance metrics for each upload.

## Features

- File uploading to any path on the server (requires appropriate permissions)
- Automatic directory creation if the destination directory doesn't exist
- Streaming uploads to minimize memory usage
- Performance logging with detailed metrics
- Benchmarking with results saved to benchmark.txt

## Getting Started

### Prerequisites

- Go 1.24 or higher
- Docker (optional, for containerized deployment)

### Running Locally

```bash
# Run the application directly
make run

# OR build and run the binary
make build
./upload-gb
```

### Using Docker

```bash
# Build Docker image
make docker-build

# Run in Docker
make docker-run
```

## API Usage

### Upload Endpoint

**Endpoint:** `POST /upload`

**Query Parameters:**
- `dest`: (Required) Destination path where the file should be saved

**Form Data:**
- `file`: (Required) The file to upload

**Response:**
```json
{
  "status": "success",
  "destination": "/path/to/saved/file",
  "size": 12345,
  "duration_ms": 42,
  "transfer_rate": "10.5 MB/s",
  "memory_used_mb": "8.45 MB",
  "cpu_usage": "12.3%"
}
```

### Example Usage

Upload a file using curl:

```bash
curl -X POST \
  "http://localhost:8080/upload?dest=/tmp/myupload.txt" \
  -F "file=@/path/to/local/file.txt"
```

For benchmarking with larger files, you can use:

```bash
# Create a 100MB test file
dd if=/dev/urandom of=testfile bs=1M count=100

# Upload the file and time the operation
time curl -X POST \
  "http://localhost:8080/upload?dest=/tmp/benchmark-100mb.dat" \
  -F "file=@testfile"
```

## Performance Metrics

The application logs detailed performance metrics for each upload:

- File size
- Upload duration (both total and just the file copy operation)
- Transfer rate in MB/s
- Memory usage (peak and used)
- CPU usage (estimated percentage)
- Number of goroutines

All uploads are automatically benchmarked and their results are appended to a `benchmark.txt` file with the following format:

```
[2025-04-10T15:34:12Z] File: example.jpg, Size: 5242880 bytes, Duration: 235.42ms, Transfer Rate: 21.25 MB/s, Memory Used: 8.45 MB, CPU Usage: 12.3%, Goroutines: 15
```

This allows tracking performance over time and across different types of files. The memory metrics help identify potential memory leaks or inefficient resource usage, while CPU metrics and goroutine count provide insights into processing efficiency.

## Development

### Project Structure

- `main.go` - Main application entry point and HTTP server with logging
- `Makefile` - Build and run commands
- `Dockerfile` - Container definition for Docker deployment
- `benchmark.txt` - Automatically generated benchmark results

### Makefile Commands

- `make run` - Run the application locally
- `make build` - Build the binary
- `make docker-build` - Build Docker image
- `make docker-run` - Run in Docker container
- `make clean` - Clean build artifacts
