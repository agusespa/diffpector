# Diffpector Evaluation Pipeline

Systematically tests different prompts and models to establish baselines and optimize the code review agent's performance.

## Overview

The evaluation system is **server-based**, allowing you to test different models by running multiple llama.cpp servers on different ports.

The evaluation system consists of:

- **Test Suite**: A collection of test cases with known expected outcomes
- **Prompt Variants**: Different prompt templates optimized for various scenarios
- **Server Configurations**: Multiple server endpoints, each running a different model
- **Scoring System**: Automated scoring based on expected vs actual results
- **Comparison Tools**: Compare performance across different configurations

## Configuration

Edit `eval_configs.json` to define your evaluation scenarios. Each configuration specifies:

- **key**: Unique identifier for the evaluation run
- **servers**: List of llama.cpp server endpoints to test (each represents one model)
- **prompts**: Prompt variants to test
- **runs**: Number of times to run each test (for statistical significance)

### Server Configuration

Each server entry requires:
- **name**: Descriptive name for the model (used in results)
- **base_url**: Server endpoint URL

### Example: Testing Multiple Models

To compare different models, run each llama.cpp server on a different port:

```bash
# Terminal 1: Start first model
./llama-server -m qwen3-30b.gguf --port 8080

# Terminal 2: Start second model
./llama-server -m qwen2.5-14b.gguf --port 8081
```

Then configure in `eval_configs.json`:

```json
{
  "key": "model-comparison",
  "servers": [
    {
      "name": "qwen3-30b",
      "base_url": "http://localhost:8080"
    },
    {
      "name": "qwen2.5-14b",
      "base_url": "http://localhost:8081"
    }
  ],
  "prompts": ["optimized"],
  "runs": 3
}
```

## Test Cases

The evaluation includes test cases for:

- **Security Issues**: SQL injection, XSS, authentication flaws, path traversal
- **Performance Problems**: N+1 queries, memory leaks, inefficient algorithms
- **Bug Detection**: Nil pointer dereferences, race conditions, logic errors
- **Code Quality**: Error handling, maintainability, style issues
- **False Positives**: Clean refactoring that shouldn't trigger issues

Each test case specifies:
- **name**: Test identifier
- **description**: What the test checks
- **diff_file**: Path to the diff file
- **expected**: Expected findings (severity, files, issue count)

## Usage

Run evaluations using the Makefile commands:

```bash
# Run a specific configuration
make eval-run VARIANT=model-comparison

# Compare results
make eval-compare-models
make eval-compare-prompts
```

## Results and Scoring

### Scoring System

Each test case is scored based on:
- **Issue Detection**: Did it find issues when expected?
- **Issue Count**: Are the number of issues within expected bounds?
- **Severity Matching**: Do the severities match expectations?
- **File Accuracy**: Are the correct files flagged?

Scores range from 0.0 to 1.0, with 1.0 being perfect.

### Result Files

Results are saved as JSON files in `evaluation/results/` with format:
```
eval_{server-name}_{prompt}_{runs}runs_{timestamp}.json
```

Each result file contains:
- Test case results and scores
- Execution times
- Server/model and prompt configuration
- Detailed issue analysis
- Aggregated statistics (mean, standard deviation)

## Model Evaluation History

- **codellama:13b** - Too many format violation errors
- **codestral** - Too many format violation errors
- **qwen3:14b** - Less accuracy and worse performance than qwen2.5-coder:14b
- **qwen2.5-coder:14b** - Best balance between accuracy and performance
- **qwen3-coder:30b** - Slight edge on accuracy over 14b model
