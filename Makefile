.PHONY: run build docker-build docker-run benchmark clean

APP_NAME = upload-gb
DOCKER_IMAGE = $(APP_NAME):latest
BENCHMARK_SIZE = 100

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
	docker run -p 8080:8080 -v $(PWD):/app/data $(DOCKER_IMAGE)

# Run benchmark tests
benchmark: build
	@echo "Creating test file of $(BENCHMARK_SIZE)MB..."
	dd if=/dev/urandom of=benchmark_testfile bs=1M count=$(BENCHMARK_SIZE)
	@echo "Starting server in background..."
	./$(APP_NAME) &
	@sleep 2
	@echo "Running benchmark..."
	time curl -s -X POST \
		"http://localhost:8080/upload?dest=/tmp/benchmark_$(BENCHMARK_SIZE)mb.dat" \
		-F "file=@benchmark_testfile" | jq
	@echo "Benchmark results saved to benchmark.txt"
	@echo "Cleaning up..."
	rm -f benchmark_testfile
	pkill -f $(APP_NAME)

# Clean build artifacts
clean:
	rm -f $(APP_NAME)
	rm -f benchmark_testfile