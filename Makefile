# Detect the host OS/arch from the Go toolchain itself, so the build matches
# whatever `go` would target by default (and respects any GOOS/GOARCH override).
GOOS   := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

BUILD_DIR := build
BINARY    := mssp
# Windows executables need the .exe suffix.
ifeq ($(GOOS),windows)
	EXT := .exe
endif
OUTPUT := $(BUILD_DIR)/$(BINARY)-$(GOOS)-$(GOARCH)$(EXT)

.PHONY: build test clean

# Build for the host platform into build/.
build:
	go build -o $(OUTPUT) .
	@echo "built $(OUTPUT)"

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
j