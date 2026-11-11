BINARY_NAME := osinfo
PKG_PATH    := ./cmd/osinfo
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE  := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X github.com/hieulvt/osinfo/internal/version.Version=$(VERSION) \
	-X github.com/hieulvt/osinfo/internal/version.Commit=$(COMMIT) \
	-X github.com/hieulvt/osinfo/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: all build install test test-race vet fmt tidy build-all clean

all: vet test build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) $(PKG_PATH)

install:
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" $(PKG_PATH)

test:
	go test -v ./...

test-race:
	go test -race -coverprofile=coverage.out ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

build-all:
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-amd64 $(PKG_PATH)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-arm64 $(PKG_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-arm64 $(PKG_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-amd64.exe $(PKG_PATH)

clean:
	rm -rf bin dist coverage.out

