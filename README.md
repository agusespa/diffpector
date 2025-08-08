# Diffpector Review Agent

A local code review agent powered by Ollama that analyzes Git commits to identify potential problems, code quality issues, and security vulnerabilities.

### Features
- **Local-Only**: Runs entirely with Ollama - no cloud dependencies
- **Git Integration**: Analyzes commits and diffs
- **Code Quality Analysis**: Identifies potential bugs, security issues, and code smells
- **Detailed Reports**: Generates comprehensive code review reports

## Prerequisites

### Ollama Setup
Install Ollama and download the model you want to use.
Run `ollama serve`.

### Git Repository
- Must be run from within a Git repository
- Requires commits to analyze


## Configuration

The `config.json` file contains the variables for the api and model.
The only api currently supported is `ollama`.

```json
{
  "llm": {
    "provider": "ollama",
    "model": "codellama",
    "base_url": "http://localhost:11434",
  }
}
```

### Recommended Models
- **qwen2.5-coder:14b** - Best overall performance for code review
- **codellama** - Good alternative with strong code understanding
- **qwen2.5-coder:7b** - Faster option for quick reviews
