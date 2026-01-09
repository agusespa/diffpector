# Diffpector Review Agent

A local code review agent that analyzes Git commits to identify potential problems, code quality issues, and security vulnerabilities. Supports both Ollama and llama.cpp backends.

### Features
- **Local-Only**: Runs entirely on your machine - no cloud dependencies
- **Multi-Language Support**: Analyzes Go, Java and TypeScript code with symbol-aware context
- **Git Integration**: Analyzes commits and diffs
- **Code Quality Analysis**: Identifies potential bugs, security issues, and code smells
- **Detailed Reports**: Generates comprehensive code review reports
- **Flexible Backend**: Use Ollama or llama.cpp with OpenAI-compatible API

## Installation

Download the latest binary for your platform from [Releases](https://github.com/yourusername/diffpector/releases):
- Extract the archive
- Move the binary to your PATH (e.g., `/usr/local/bin` on macOS/Linux)

## Prerequisites

### LLM Backend Setup

Choose one of the following backends:

#### Option 1: Ollama (Recommended for ease of use)
1. Install [Ollama](https://ollama.ai)
2. Download your preferred model: `ollama pull qwen2.5-coder:14b`
3. Start the server: `ollama serve`

#### Option 2: llama.cpp (For more control)
1. Build [llama.cpp](https://github.com/ggerganov/llama.cpp) with the server enabled
2. Download a GGUF model file
3. Start the server with OpenAI-compatible API:
   ```bash
   llama-server -m /path/to/model.gguf --port 8080
   ```

### Git Repository
- Must be run from within a Git repository
- Requires commits to analyze

**Important**: Run diffpector from your project's root directory (where your `.git` folder is located). The tool needs to be executed from the repository root to properly analyze symbol context and cross-references.

## Configuration

The agent uses default configuration for llama.cpp. Override by creating a `diffpectrc.json` file in your project root.

### llama.cpp Configuration (Default)
```json
{
  "llm": {
    "provider": "openai",
    "base_url": "http://localhost:8080"
  }
}
```

The `model` field is optional for llama.cpp since the model is already loaded when you start the server. The `api_key` field is also optional for local servers.

### Ollama Configuration
```json
{
  "llm": {
    "provider": "ollama",
    "model": "qwen2.5-coder:14b",
    "base_url": "http://localhost:11434"
  }
}
```

For Ollama, you must specify the `model` field.

### Recommended Models
- **qwen 3 coder (30b, q4)** - best balance between accuracy and performance (if memory constrained use **qwen 2.5 coder (14b, q4)** instead)
