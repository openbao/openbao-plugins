BINARY    := home-engine
CMD       := ./cmd
GOFLAGS   := -trimpath
LDFLAGS   := -s -w

# Detect architecture for the local build
GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.PHONY: all build test lint clean release help

all: build

## build: Compile the plugin binary for the current platform.
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
		-o bin/$(BINARY) $(CMD)

## test: Run all unit tests.
test:
	go test ./... -v -count=1

## lint: Run go vet.
lint:
	go vet ./...

## release: Cross-compile release binaries for common platforms.
release:
	@for PLATFORM in \
		linux/amd64 \
		linux/arm64 \
		darwin/amd64 \
		darwin/arm64; do \
		OS=$$(echo $$PLATFORM | cut -d/ -f1); \
		ARCH=$$(echo $$PLATFORM | cut -d/ -f2); \
		OUT=bin/$(BINARY)_$${OS}_$${ARCH}; \
		echo "Building $$OUT..."; \
		CGO_ENABLED=0 GOOS=$$OS GOARCH=$$ARCH \
			go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
			-o $$OUT $(CMD); \
	done
	@echo "Release binaries written to bin/"

## clean: Remove build artefacts.
clean:
	rm -rf bin/

## help: Show this help message.
help:
	@grep -E '^## ' Makefile | sed 's/## //'
