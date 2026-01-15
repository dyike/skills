# Skills Build System
# Usage:
#   make build SKILL=<skill-name>                     - Build a skill
#   make install SKILL=<skill-name> TARGET=gemini     - Install to Gemini
#   make install SKILL=<skill-name> TARGET=claude     - Install to Claude
#   make install SKILL=<skill-name> TARGET=/path/to   - Install to custom path
#   make all                                          - Build all skills
#   make clean                                        - Clean build artifacts

# Configuration
BUILD_DIR := build
GO := go
GOFLAGS := -ldflags="-s -w"

# Default paths
GEMINI_DIR := $(HOME)/.gemini/antigravity/skills
CLAUDE_DIR := $(HOME)/.claude/skills

# Find all skills (directories under cmd/)
SKILLS := $(notdir $(wildcard cmd/*))

# Default target
.PHONY: all clean help list build install

all: build-all

# Help target
help:
	@echo "Skills Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build SKILL=<name>                     Build a specific skill"
	@echo "  make build-all                              Build all skills"
	@echo "  make install SKILL=<name> TARGET=gemini     Install to Gemini (~/.gemini/antigravity/skills)"
	@echo "  make install SKILL=<name> TARGET=claude     Install to Claude (~/.claude/skills)"
	@echo "  make install SKILL=<name> TARGET=/path/to   Install to custom path"
	@echo "  make list                                   List available skills"
	@echo "  make clean                                  Clean build artifacts"
	@echo ""
	@echo "Examples:"
	@echo "  make build SKILL=news-scraper"
	@echo "  make install SKILL=news-scraper TARGET=gemini"
	@echo "  make install SKILL=trade TARGET=claude"
	@echo "  make install SKILL=trade TARGET=/opt/skills"
	@echo ""
	@echo "Available skills:"
	@for skill in $(SKILLS); do echo "  - $$skill"; done

# List available skills
list:
	@echo "Available skills:"
	@for skill in $(SKILLS); do echo "  $$skill"; done

# Build a specific skill
build:
ifndef SKILL
	$(error SKILL is required. Usage: make build SKILL=<skill-name>)
endif
	@echo "Building skill: $(SKILL)"
	@if [ ! -d "cmd/$(SKILL)" ]; then \
		echo "Error: Skill '$(SKILL)' not found in cmd/"; \
		exit 1; \
	fi
	@mkdir -p $(BUILD_DIR)/$(SKILL)/scripts
	@# Build Go binary
	@echo "  Compiling Go binary..."
	@CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(SKILL)/scripts/$(SKILL) ./cmd/$(SKILL)
	@# Copy SKILL.md
	@if [ -f "skill/$(SKILL)/SKILL.md" ]; then \
		echo "  Copying SKILL.md..."; \
		cp skill/$(SKILL)/SKILL.md $(BUILD_DIR)/$(SKILL)/; \
	fi
	@# Copy references if exists
	@if [ -d "skill/$(SKILL)/references" ]; then \
		echo "  Copying references..."; \
		mkdir -p $(BUILD_DIR)/$(SKILL)/references; \
		cp -r skill/$(SKILL)/references/* $(BUILD_DIR)/$(SKILL)/references/; \
	fi
	@# Copy assets if exists
	@if [ -d "skill/$(SKILL)/assets" ]; then \
		echo "  Copying assets..."; \
		mkdir -p $(BUILD_DIR)/$(SKILL)/assets; \
		cp -r skill/$(SKILL)/assets/* $(BUILD_DIR)/$(SKILL)/assets/; \
	fi
	@# Copy additional scripts if exists
	@if [ -d "skill/$(SKILL)/scripts" ]; then \
		echo "  Copying scripts..."; \
		cp -r skill/$(SKILL)/scripts/* $(BUILD_DIR)/$(SKILL)/scripts/; \
	fi
	@echo "  Done! Output: $(BUILD_DIR)/$(SKILL)/"

# Build all skills
build-all:
	@for skill in $(SKILLS); do \
		echo "Building $$skill..."; \
		$(MAKE) build SKILL=$$skill; \
	done

# Install a skill
install:
ifndef SKILL
	$(error SKILL is required. Usage: make install SKILL=<skill-name> TARGET=<gemini|claude|path>)
endif
ifndef TARGET
	$(error TARGET is required. Usage: make install SKILL=<skill-name> TARGET=<gemini|claude|path>)
endif
	@# Determine install directory
	$(eval INSTALL_DIR := $(if $(filter gemini,$(TARGET)),$(GEMINI_DIR),$(if $(filter claude,$(TARGET)),$(CLAUDE_DIR),$(TARGET))))
	@# Build first if not built
	@if [ ! -d "$(BUILD_DIR)/$(SKILL)" ]; then \
		echo "Building $(SKILL) first..."; \
		$(MAKE) build SKILL=$(SKILL); \
	fi
	@echo "Installing skill: $(SKILL) to $(INSTALL_DIR)/$(SKILL)"
	@mkdir -p $(INSTALL_DIR)
	@rm -rf $(INSTALL_DIR)/$(SKILL)
	@cp -r $(BUILD_DIR)/$(SKILL) $(INSTALL_DIR)/
	@echo "  Installed to: $(INSTALL_DIR)/$(SKILL)/"

# Install all skills to a target
install-all:
ifndef TARGET
	$(error TARGET is required. Usage: make install-all TARGET=<gemini|claude|path>)
endif
	@for skill in $(SKILLS); do \
		$(MAKE) install SKILL=$$skill TARGET=$(TARGET); \
	done

# Clean build artifacts
clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@echo "Done!"

# Shortcut aliases for common operations
gemini: 
ifndef SKILL
	$(error SKILL is required. Usage: make gemini SKILL=<skill-name>)
endif
	@$(MAKE) install SKILL=$(SKILL) TARGET=gemini

claude:
ifndef SKILL
	$(error SKILL is required. Usage: make claude SKILL=<skill-name>)
endif
	@$(MAKE) install SKILL=$(SKILL) TARGET=claude
