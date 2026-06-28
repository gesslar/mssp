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

.PHONY: build test clean

# Build for the host platform into build/.
build:
	go build -o $(OUTPUT) ./cmd/mssp
	@echo "built $(OUTPUT)"

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
