BINARY_NAME=terralink
MAIN_PACKAGE_PATH=.
OUTPUT_DIR=dist
VERSION ?= dev

# Build flags
BUILD_PKG_PATH = terralink/cmd
BUILD_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_LDFLAGS ?= -ldflags "-X '$(BUILD_PKG_PATH).Version=$(VERSION)' -X '$(BUILD_PKG_PATH).Commit=$(BUILD_COMMIT)' -X '$(BUILD_PKG_PATH).BuildDate=$(BUILD_DATE)'"


PLATFORMS=linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64


.PHONY: all
all: build


.PHONY: build
build:
	@echo "Starting build for version: $(VERSION)..."
	@mkdir -p $(OUTPUT_DIR)
	@$(foreach platform,$(PLATFORMS), $(call build_platform,$(platform)))
	@echo "Build complete. Binaries are in $(OUTPUT_DIR)/"

# A helper function to build for a single platform
define build_platform
	$(eval parts = $(subst /, ,$(1)))
	$(eval os = $(word 1,$(parts)))
	$(eval arch = $(word 2,$(parts)))
	@echo "--> Building for $(os)/$(arch)..."
	$(eval output_name = "$(OUTPUT_DIR)/$(BINARY_NAME)-$(VERSION)-$(os)-$(arch)")
	$(if $(findstring windows,$(os)),$(eval output_name = "$(output_name).exe"))
	@GOOS=$(os) GOARCH=$(arch) go build -v $(BUILD_LDFLAGS) -o $(output_name) $(MAIN_PACKAGE_PATH)
endef


.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "Tests completed."

lint:
	@echo "Running linter..."
	@golangci-lint run
	@echo "Linting completed."
# Target to clean up the build artifacts
.PHONY: clean
clean:
	@echo "Cleaning up build artifacts..."
	@rm -rf $(OUTPUT_DIR)
	@echo "Done."