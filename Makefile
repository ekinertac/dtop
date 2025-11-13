.PHONY: build run clean install test

# Build the binary
build:
	go build -o dtop

# Run the application
run: build
	./dtop

# Clean build artifacts
clean:
	rm -f dtop
	go clean

# Install to $GOPATH/bin
install:
	go install

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o dtop-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -o dtop-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o dtop-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -o dtop-windows-amd64.exe

