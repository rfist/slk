.PHONY: build test lint run clean

BINARY=slk
BUILD_DIR=bin

# Build identity injected into cmd/slk's version vars so --version and
# the in-TUI :version command identify the exact build. "-dirty" marks
# uncommitted changes in the working tree.
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)$(shell git diff --quiet 2>/dev/null || echo -dirty)
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-s -w -X main.commit=$(GIT_COMMIT) -X main.date=$(BUILD_DATE)

build:
	go build -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(BINARY) ./cmd/slk

test:
	go test ./... -v -race

lint:
	golangci-lint run ./...

run: build
	./$(BUILD_DIR)/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)
