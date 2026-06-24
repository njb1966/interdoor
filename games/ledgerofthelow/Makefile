BINARY_NODE     = interdoor-node
BINARY_HUB      = interdoor-hub
BINARY_HUB_ADMIN = interdoor-hub-admin
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     = -X main.version=$(VERSION) -s -w

.PHONY: all build test release clean

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NODE)     ./cmd/node
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_HUB)      ./cmd/hub
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_HUB_ADMIN) ./cmd/hubadmin

test:
	go test ./...

release:
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NODE)-linux-amd64     ./cmd/node
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NODE)-linux-arm64     ./cmd/node
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY_HUB)-linux-amd64      ./cmd/hub
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY_HUB)-linux-arm64      ./cmd/hub
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY_HUB_ADMIN)-linux-amd64 ./cmd/hubadmin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY_HUB_ADMIN)-linux-arm64 ./cmd/hubadmin
	@echo "release binaries in dist/"

clean:
	rm -rf bin/ dist/
