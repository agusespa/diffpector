# Diffpector Evaluation Pipeline

Systematically tests different prompts and models to establish baselines and optimize the code review agent's performance.

## Overview

The evaluation system consists of:

- **Test Suite**: A collection of test cases with known expected outcomes
- **Prompt Variants**: Different prompt templates optimized for various scenarios
- **Model Testing**: Support for testing multiple models and configurations
- **Scoring System**: Automated scoring based on expected vs actual results
- **Comparison Tools**: Compare performance across different configurations

## Test Cases

The evaluation includes test cases for:

- **Security Issues**: SQL injection, XSS, authentication flaws
- **Performance Problems**: N+1 queries, memory leaks, inefficient algorithms
- **Bug Detection**: Nil pointer dereferences, race conditions, logic errors
- **Code Quality**: Error handling, maintainability, style issues
- **False Positives**: Clean refactoring that shouldn't trigger issues

## Prompt Variants

Available prompt variants:

- **default**: Current production prompt

## Usage

Run evaluations using the main CLI or directly with the evaluation package.

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
eval_{model}_{prompt}_{timestamp}.json
```

Each result file contains:
- Test case results and scores
- Execution times
- Model and prompt configuration
- Detailed issue analysis

## Model evaluation history
- codellama:13b > too many format violation errors
- codestral > too many format violation errors
- qwen3:14b > less accuracy and worse performance than qwen2.5-coder:14b
