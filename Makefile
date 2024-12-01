# Define target platforms
GOOSES := darwin linux
GOARCHES := arm64 amd64

# Get current platform for symlinking
CURRENT_OS := $(shell go env GOOS)
CURRENT_ARCH := $(shell go env GOARCH)

# Enable CGO
export CGO_ENABLED=1

# Set install location
PREFIX ?= /usr/local
INSTALL_PATH = $(PREFIX)/bin

# Generate all possible OS/ARCH combinations
TARGETS := $(foreach os,$(GOOSES),$(foreach arch,$(GOARCHES),bin/$(os)/$(arch)/llmcat))

.PHONY: all clean

all: bin/llmcat

# Pattern rule for building specific OS/ARCH combinations
bin/%/llmcat:
	@echo "Building for $(firstword $(subst /, ,$*))/$(word 2,$(subst /, ,$*))..."
	@mkdir -p $(dir $@)
	CGO_ENABLED=1 GOOS=$(firstword $(subst /, ,$*)) GOARCH=$(word 2,$(subst /, ,$*)) go build -o $@ ./cmd/llmcat

# Rule for building current platform and creating symlink
bin/llmcat: bin/$(CURRENT_OS)/$(CURRENT_ARCH)/llmcat
	@echo "Creating symlink for current platform ($(CURRENT_OS)/$(CURRENT_ARCH))..."
	@mkdir -p bin
	@ln -sf $(CURRENT_OS)/$(CURRENT_ARCH)/llmcat $@
	@chmod +x $@

install: bin/llmcat
	@echo "Installing to $(INSTALL_PATH)/llmcat..."
	@mkdir -p $(INSTALL_PATH)
	@install -m 755 bin/llmcat $(INSTALL_PATH)/llmcat

clean:
	@echo "Cleaning up..."
	@rm -rf bin/

# Debug target to print variables
debug:
	@echo "Targets: $(TARGETS)"
	@echo "Current OS: $(CURRENT_OS)"
	@echo "Current ARCH: $(CURRENT_ARCH)"
