# Diffpector Evaluation Pipeline

Systematically tests different prompts and models to establish baselines and optimize the code review agent's performance.

## Overview

The evaluation system **automatically manages llama-server**, loading one model at a time. You don't need to manually start servers - just specify the model paths in the config and the pipeline handles everything.

The evaluation system consists of:

- **Test Suite**: A collection of test cases with known expected outcomes
- **Prompt Variants**: Different prompt templates optimized for various scenarios
- **Model Configurations**: Model paths for sequential testing
- **Scoring System**: Automated scoring based on expected vs actual results
- **Comparison Tools**: Compare performance across different configurations

## Configuration

Edit `eval_configs.json` to define your evaluation scenarios. Each configuration specifies:

- **key**: Unique identifier for the evaluation run
- **servers**: List of models to test (each with name and model_path)
- **prompts**: Prompt variants to test
- **runs**: Number of times to run each test (for statistical significance)

### Server Configuration

Each server entry requires:
- **name**: Descriptive name for the model (used in results)
- **model_path**: Path to the GGUF model file (can be absolute or relative)

### Example Configuration

```json
{
  "key": "model-comparison",
  "servers": [
    {
      "name": "qwen3-30b",
      "model_path": "/Users/username/models/qwen3-30b.gguf"
    },
    {
      "name": "qwen2.5-14b",
      "model_path": "/Users/username/models/qwen2.5-14b.gguf"
    }
  ],
  "prompts": ["optimized"],
  "runs": 3
}
```

The pipeline will:
1. Start llama-server with qwen3-30b
2. Run all evaluations for that model
3. Stop the server
4. Start llama-server with qwen2.5-14b
5. Run all evaluations for that model
6. Stop the server
7. Generate comparison results

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

### Advanced Options

You can customize the llama-server path, port, and additional arguments:

```bash
# Use a custom llama-server binary
go run cmd/eval/main.go --variant model-comparison --llama-server /path/to/llama-server

# Use a different port (default is 8080)
go run cmd/eval/main.go --variant model-comparison --port 8081

# Customize llama-server arguments (context size, GPU layers, threads, etc.)
go run cmd/eval/main.go --variant model-comparison \
  --server-args "-c 65536 -n 8192 -ngl 99 -b 2048 -ub 1024 --threads 12"
```

Default server arguments: `-c 65536 -n 8192 -ngl 99 -b 2048 -ub 1024 --threads 12`

These arguments configure:
- `-c 65536`: Context size
- `-n 8192`: Max tokens to predict
- `-ngl 99`: GPU layers to offload
- `-b 2048`: Batch size
- `-ub 1024`: Unbatch size
- `--threads 12`: CPU threads to use

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
