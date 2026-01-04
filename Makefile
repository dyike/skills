# Skills Build System
# Usage:
#   make <skill-name>                        - Build skill (e.g., make news-scraper)
#   make install-<skill-name>                - Install skill to ~/.claude/skills
#   make install-<skill-name> INSTALL_DIR=x  - Install skill to custom path
#   make all                                 - Build all skills
#   make clean                               - Clean build artifacts

# Configuration
BUILD_DIR := build
INSTALL_DIR ?= $(HOME)/.claude/skills
GO := go
GOFLAGS := -ldflags="-s -w"

# Find all skills (directories under cmd/)
SKILLS := $(notdir $(wildcard cmd/*))

# Default target
.PHONY: all clean help list $(SKILLS) $(addprefix install-,$(SKILLS))

all: $(SKILLS)

# Help target
help:
	@echo "Skills Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make <skill-name>                        Build a specific skill"
	@echo "  make install-<skill-name>                Build and install a skill"
	@echo "  make install-<skill-name> INSTALL_DIR=x  Install to custom path"
	@echo "  make all                                 Build all skills"
	@echo "  make list                                List available skills"
	@echo "  make clean                               Clean build artifacts"
	@echo ""
	@echo "Default install path: $(INSTALL_DIR)"
	@echo ""
	@echo "Available skills:"
	@for skill in $(SKILLS); do echo "  - $$skill"; done

# List available skills
list:
	@echo "Available skills:"
	@for skill in $(SKILLS); do echo "  $$skill"; done

# Build a skill
# Creates: build/<skill-name>/
#          ├── SKILL.md
#          ├── scripts/<binary>
#          ├── references/
#          └── assets/
define BUILD_SKILL
$(1): 
	@echo "Building skill: $(1)"
	@mkdir -p $(BUILD_DIR)/$(1)/scripts
	@# Build Go binary
	@if [ -d "cmd/$(1)" ]; then \
		echo "  Compiling Go binary..."; \
		CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(1)/scripts/$(1) ./cmd/$(1); \
	fi
	@# Copy SKILL.md
	@if [ -f "skill/$(1)/SKILL.md" ]; then \
		echo "  Copying SKILL.md..."; \
		cp skill/$(1)/SKILL.md $(BUILD_DIR)/$(1)/; \
	fi
	@# Copy references if exists
	@if [ -d "skill/$(1)/references" ]; then \
		echo "  Copying references..."; \
		mkdir -p $(BUILD_DIR)/$(1)/references; \
		cp -r skill/$(1)/references/* $(BUILD_DIR)/$(1)/references/; \
	fi
	@# Copy assets if exists
	@if [ -d "skill/$(1)/assets" ]; then \
		echo "  Copying assets..."; \
		mkdir -p $(BUILD_DIR)/$(1)/assets; \
		cp -r skill/$(1)/assets/* $(BUILD_DIR)/$(1)/assets/; \
	fi
	@# Copy additional scripts if exists
	@if [ -d "skill/$(1)/scripts" ]; then \
		echo "  Copying scripts..."; \
		cp -r skill/$(1)/scripts/* $(BUILD_DIR)/$(1)/scripts/; \
	fi
	@echo "  Done! Output: $(BUILD_DIR)/$(1)/"
endef

# Generate build rules for each skill
$(foreach skill,$(SKILLS),$(eval $(call BUILD_SKILL,$(skill))))

# Install a skill
define INSTALL_SKILL
install-$(1): $(1)
	@echo "Installing skill: $(1) to $(INSTALL_DIR)/$(1)"
	@mkdir -p $(INSTALL_DIR)
	@rm -rf $(INSTALL_DIR)/$(1)
	@cp -r $(BUILD_DIR)/$(1) $(INSTALL_DIR)/
	@echo "  Installed to: $(INSTALL_DIR)/$(1)/"
endef

# Generate install rules for each skill
$(foreach skill,$(SKILLS),$(eval $(call INSTALL_SKILL,$(skill))))

# Clean build artifacts
clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@echo "Done!"

# Install all skills
install-all: all
	@for skill in $(SKILLS); do \
		echo "Installing $$skill..."; \
		mkdir -p $(INSTALL_DIR); \
		rm -rf $(INSTALL_DIR)/$$skill; \
		cp -r $(BUILD_DIR)/$$skill $(INSTALL_DIR)/; \
	done
	@echo "All skills installed to $(INSTALL_DIR)/"
