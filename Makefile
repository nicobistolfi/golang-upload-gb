.PHONY: run build docker-build docker-run clean

APP_NAME = upload-gb
DOCKER_IMAGE = $(APP_NAME):latest

# Run locally
run:
	go run main.go

# Build binary
build:
	CGO_ENABLED=0 go build -o $(APP_NAME) main.go

# Build Docker image
docker-build: build
	docker build -t $(DOCKER_IMAGE) .

# Run Docker container
docker-run: docker-build
	docker run -p 8080:8080 $(DOCKER_IMAGE)

# Clean build artifacts
clean:
	rm -f $(APP_NAME)