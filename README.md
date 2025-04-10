# upload-gb

A simple HTTP file upload service built with Go and Gin.

## Overview

This service provides a single endpoint (`/upload`) that accepts file uploads via HTTP POST requests and saves them to a specified destination on the server filesystem.

## Features

- File uploading to any path on the server (requires appropriate permissions)
- Automatic directory creation if the destination directory doesn't exist
- Streaming uploads to minimize memory usage

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
  "size": 12345
}
```

### Example Usage

Upload a file using curl:

```bash
curl -X POST \
  "http://localhost:8080/upload?dest=/tmp/myupload.txt" \
  -F "file=@/path/to/local/file.txt"
```

## Development

### Project Structure

- `main.go` - Main application entry point and HTTP server
- `Makefile` - Build and run commands
- `Dockerfile` - Container definition for Docker deployment

### Makefile Commands

- `make run` - Run the application locally
- `make build` - Build the binary
- `make docker-build` - Build Docker image
- `make docker-run` - Run in Docker container
- `make clean` - Clean build artifacts# golang-upload-gb
