---
name: generate-image
description: Generate images using Gemini API via modu gen_image_repo. Supports custom prompts, system prompts, and configurable API endpoints.
---

# Generate Image

Generate images using Gemini API.

## Usage

```bash
./scripts/generate-image [prompt] [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--api-key` | `-k` | - | API key (or set GEMINI_API_KEY env) |
| `--base-url` | `-b` | - | API base URL (or set GEMINI_BASE_URL env) |
| `--output` | `-o` | generated_image.jpg | Output file path |
| `--system` | `-s` | - | System prompt (optional) |
| `--prompt-file` | `-p` | - | Read prompt from file |
| `--env-file` | `-e` | .env | Environment file to load |

## Examples

```bash
# Basic usage
./scripts/generate-image "a sunset over mountains"

# Read prompt from file
./scripts/generate-image -p prompt.txt
./scripts/generate-image @prompt.txt

# Custom output
./scripts/generate-image -o my_image.png "futuristic city"

# With custom env file
./scripts/generate-image -e .env.local "abstract art"
```

## Environment Variables

Uses `modu/pkg/env` to load from `.env` file:

```env
# .env
GEMINI_API_KEY=your-api-key
GEMINI_BASE_URL=http://127.0.0.1:8045
```

| Variable | Description | Default |
|----------|-------------|---------|
| `GEMINI_API_KEY` | API key (required) | - |
| `GEMINI_BASE_URL` | API base URL | http://127.0.0.1:8045 |
