BINARY    := git-syncer
MODULE    := github.com/minhajuddin/git-syncer
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -ldflags "-s -w -X main.version=$(VERSION)"

# Build targets
.PHONY: build build-linux build-darwin build-all
build:
	go build $(LDFLAGS) -o $(BINARY) .

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 .

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 .

build-all: build-linux build-darwin

# Test targets
.PHONY: test test-verbose test-race
test:
	go test -count=1 ./...

test-verbose:
	go test -v -count=1 ./...

test-race:
	go test -race -count=1 ./...

# Code quality
.PHONY: fmt vet lint check
fmt:
	go fmt ./...

vet:
	go vet ./...

lint: vet
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

check: fmt vet test

# Install / uninstall
.PHONY: install uninstall
install: build
	cp $(BINARY) $(GOPATH)/bin/$(BINARY) 2>/dev/null || cp $(BINARY) $(HOME)/go/bin/$(BINARY)

uninstall:
	rm -f $(GOPATH)/bin/$(BINARY) 2>/dev/null || rm -f $(HOME)/go/bin/$(BINARY)

# Maintenance
.PHONY: tidy clean
tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
	rm -rf dist/
