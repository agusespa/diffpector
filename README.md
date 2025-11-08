# Diffpector Review Agent

A local code review agent powered by Ollama that analyzes Git commits to identify potential problems, code quality issues, and security vulnerabilities.

### Features
- **Local-Only**: Runs entirely with Ollama - no cloud dependencies
- **Multi-Language Support**: Analyzes Go, Java and TypeScript code with symbol-aware context
- **Git Integration**: Analyzes commits and diffs
- **Code Quality Analysis**: Identifies potential bugs, security issues, and code smells
- **Detailed Reports**: Generates comprehensive code review reports

## Installation

Download the latest binary for your platform from [Releases](https://github.com/yourusername/diffpector/releases):
- Extract the archive
- Move the binary to your PATH (e.g., `/usr/local/bin` on macOS/Linux)

## Prerequisites

### Ollama Setup
Install Ollama and download the model you want to use.
Run `ollama serve`.

### Git Repository
- Must be run from within a Git repository
- Requires commits to analyze

**Important**: Run diffpector from your project's root directory (where your `.git` folder is located). The tool needs to be executed from the repository root to properly analyze symbol context and cross-references.

## Configuration
The agent will execute by default with the following configuration parameters:
```json
{
  "llm": {
    "provider": "ollama",
    "model": "qwen2.5-coder:14b",
    "base_url": "http://localhost:11434",
  }
}
```

The defaults can be overriden by creating a `diffpectrc.json` file contains the variables for the api and model.
The only api currently supported is `ollama`.

### Recommended Models
- **qwen2.5-coder:14b** - best balance between accuracy and performance
- **qwen2.5-coder:7b** - acceptable compromise for quick reviews
