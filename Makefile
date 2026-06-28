# Detect the host OS from the Go toolchain (respecting any GOOS override) so we
# can add the .exe suffix when building for Windows.
GOOS := $(shell go env GOOS)

BUILD_DIR := build
BINARY    := mssp
# Windows executables need the .exe suffix.
ifeq ($(GOOS),windows)
	EXT := .exe
endif
OUTPUT := $(BUILD_DIR)/$(BINARY)$(EXT)

.PHONY: build test lint setup clean

# Build for the host platform into build/.
build:
	go build -o $(OUTPUT) ./cmd/mssp
	@echo "built $(OUTPUT)"

test:
	go test ./...

# Run the same linters CI runs (golangci-lint defaults, no config file).
lint:
	golangci-lint run ./...

# One-time, per-clone: point git at the tracked hooks in .githooks/.
setup:
	git config core.hooksPath .githooks
	@echo "git hooks activated"

clean:
	rm -rf $(BUILD_DIR)
