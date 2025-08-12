# Diffpector Review Agent

A local code review agent powered by Ollama that analyzes Git commits to identify potential problems, code quality issues, and security vulnerabilities.

### Features
- **Local-Only**: Runs entirely with Ollama - no cloud dependencies
- **Multi-Language Support**: Analyzes Go and Java code with symbol-aware context
- **Git Integration**: Analyzes commits and diffs
- **Code Quality Analysis**: Identifies potential bugs, security issues, and code smells
- **Detailed Reports**: Generates comprehensive code review reports

## Installation

### Pre-built Binaries
Download the latest binary for your platform from [Releases](https://github.com/yourusername/diffpector/releases):
- Extract the archive
- Move the binary to your PATH (e.g., `/usr/local/bin` on macOS/Linux)

### Go Install
```bash
go install github.com/yourusername/diffpector/cmd/diffpector@latest
```

## Prerequisites

### Ollama Setup
Install Ollama and download the model you want to use.
Run `ollama serve`.

### Git Repository
- Must be run from within a Git repository
- Requires commits to analyze

## Usage

```bash
# Basic usage (uses config.json in current directory)
diffpector

# Use custom config file
diffpector --config my-config.json

# Show help
diffpector --help
```

**Important**: Run diffpector from your project's root directory (where your `.git` folder is located). The tool needs to be executed from the repository root to properly analyze symbol context and cross-references.

## Supported Languages

Diffpector provides intelligent symbol analysis for the following languages:

- **Go** (.go files): Functions, methods, types, constants, variables
- **Java** (.java files): Classes, interfaces, methods, constructors, fields, constants, enums, annotations

The tool automatically detects the language based on file extensions and provides context-aware analysis including symbol usage tracking and cross-reference detection.

## Configuration

The `config.json` file contains the variables for the api and model.
The only api currently supported is `ollama`.

```json
{
  "llm": {
    "provider": "ollama",
    "model": "mistral",
    "base_url": "http://localhost:11434",
  }
}
```

### Recommended Models
- **qwen2.5-coder:14b** - Best overall performance for code review
- **qwen2.5-coder:7b** - Faster option for quick reviews
