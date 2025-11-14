package evaluation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agusespa/diffpector/internal/agent"
	"github.com/agusespa/diffpector/internal/llm"
	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/internal/types"
	"github.com/agusespa/diffpector/internal/utils"
	"github.com/sourcegraph/go-diff/diff"
)

type EvaluationConfig = types.EvaluationConfig
type EvaluationRun = types.EvaluationRun
type TestCaseResult = types.TestCaseResult

type Evaluator struct {
	suite          *types.EvaluationSuite
	resultsDir     string
	toolRegistry   *tools.ToolRegistry
	parserRegistry *tools.ParserRegistry
}

func NewEvaluator(suitePath string, resultsDir string) (*Evaluator, error) {
	suite, err := LoadSuite(suitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load evaluation suite: %w", err)
	}

	parserRegistry := tools.NewParserRegistry()
	toolRegistry := tools.NewToolRegistry()

	toolRegistry.Register(tools.ToolNameSymbolContext, tools.NewSymbolContextTool(suite.MockFilesDir, parserRegistry))
	// Register a mock human loop tool for evaluation (returns empty response)
	toolRegistry.Register(tools.ToolNameHumanLoop, &mockHumanLoopTool{})

	return &Evaluator{
		suite:          suite,
		resultsDir:     resultsDir,
		toolRegistry:   toolRegistry,
		parserRegistry: parserRegistry,
	}, nil
}

func (e *Evaluator) RunEvaluation(modelConfig llm.ProviderConfig, promptVariant string, numRuns int) (*types.EvaluationResult, error) {
	if numRuns < 1 {
		numRuns = 1
	}

	provider, err := llm.NewProvider(modelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	result := &types.EvaluationResult{
		Model:          modelConfig.Model,
		Provider:       string(modelConfig.Type),
		PromptVariant:  promptVariant,
		TotalRuns:      numRuns,
		StartTime:      time.Now(),
		IndividualRuns: make([]EvaluationRun, 0, numRuns),
		TestCaseStats:  make(map[string]types.TestCaseStats),
	}

	PrintRunHeader(modelConfig.Model, promptVariant, numRuns)

	for runNum := 1; runNum <= numRuns; runNum++ {
		if numRuns > 1 {
			PrintMultiRunProgress(runNum, numRuns)
		}

		run, err := e.runSingleEvaluation(modelConfig, promptVariant, provider, runNum)
		if err != nil {
			return nil, fmt.Errorf("failed to run evaluation %d: %w", runNum, err)
		}

		result.IndividualRuns = append(result.IndividualRuns, *run)
	}

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	CalculateEvaluationStats(result)

	return result, nil
}

func (e *Evaluator) runSingleEvaluation(modelConfig llm.ProviderConfig, promptVariant string, provider llm.Provider, runNum int) (*EvaluationRun, error) {
	run := &EvaluationRun{
		Model:         modelConfig.Model,
		Provider:      string(modelConfig.Type),
		PromptVariant: promptVariant,
		StartTime:     time.Now(),
		Results:       make([]TestCaseResult, 0, len(e.suite.TestCases)),
		RunNumber:     runNum,
	}

	for i, testCase := range e.suite.TestCases {
		prefix := fmt.Sprintf("[%d/%d] %s", i+1, len(e.suite.TestCases), testCase.Name)
		fmt.Println(prefix)

		result, err := e.runSingleTest(testCase, provider, promptVariant)
		if err != nil {
			result = &TestCaseResult{
				TestCase:      testCase,
				Model:         modelConfig.Model,
				PromptHash:    promptVariant,
				ExecutionTime: time.Since(run.StartTime),
				Success:       false,
				Errors:        []string{err.Error()},
				Timestamp:     time.Now(),
				Issues:        []types.Issue{},
			}
		}
		run.Results = append(run.Results, *result)
		PrintTestResult(result, err)
	}

	run.EndTime = time.Now()
	run.TotalDuration = run.EndTime.Sub(run.StartTime)
	CalculateRunSummary(run)

	return run, nil
}

func (e *Evaluator) runSingleTest(testCase types.TestCase, provider llm.Provider, promptVariant string) (*TestCaseResult, error) {
	startTime := time.Now()

	agent := e.createTestAgent(provider, promptVariant)

	diffPath := filepath.Join(e.suite.BaseDir, testCase.DiffFile)
	diffContent, err := os.ReadFile(diffPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read diff file: %w", err)
	}

	fileDiffs, err := diff.ParseMultiFileDiff([]byte(diffContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	diffMap := make(map[string]types.DiffData)
	for _, fd := range fileDiffs {
		name := fd.NewName
		if name == "/dev/null" {
			name = fd.OrigName
		}
		name = stripGitPrefix(name)

		diffContentBytes, err := diff.PrintFileDiff(fd)
		if err != nil {
			return nil, fmt.Errorf("failed to print diff for file %s: %w", name, err)
		}

		absolutePath := name
		if !filepath.IsAbs(name) {
			var err error
			absolutePath, err = filepath.Abs(name)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path for %s: %w", name, err)
			}
		}

		diffData := types.DiffData{
			AbsolutePath: absolutePath,
			Diff:         string(diffContentBytes),
		}
		diffMap[name] = diffData
	}

	changedFilesPaths := make([]string, 0, len(diffMap))
	for fileName := range diffMap {
		changedFilesPaths = append(changedFilesPaths, fileName)
	}

	primaryLanguage, err := agent.ValidateAndDetectLanguage(changedFilesPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to detect language: %w", err)
	}

	review, err := agent.ReviewChangesWithResult(diffMap, primaryLanguage, false)
	if err != nil {
		return nil, fmt.Errorf("agent review failed: %w", err)
	}

	issues, err := utils.ParseIssuesFromResponse(review)
	if err != nil {
		// Check if this is a format violation (model didn't follow instructions)
		if utils.IsFormatViolation(err) {
			// For evaluation purposes, treat format violations as a failure
			// but still return a result so we can track this metric
			return &TestCaseResult{
				TestCase:      testCase,
				Model:         provider.GetModel(),
				PromptHash:    promptVariant,
				Issues:        []types.Issue{}, // Empty since we couldn't parse
				ExecutionTime: time.Since(startTime),
				Success:       false, // Mark as failure due to format violation
				Score:         0.0,   // Zero score for format violations
				Errors:        []string{fmt.Sprintf("Format violation: %v", err)},
				Timestamp:     time.Now(),
			}, nil
		}
		// For other parsing errors, return the error
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	score := CalculateScore(testCase.Expected, issues)

	return &TestCaseResult{
		TestCase:      testCase,
		Model:         provider.GetModel(),
		PromptHash:    promptVariant,
		Issues:        issues,
		ExecutionTime: time.Since(startTime),
		Success:       true,
		Score:         score,
		Timestamp:     time.Now(),
	}, nil
}

func stripGitPrefix(path string) string {
	if strings.HasPrefix(path, "a/") || strings.HasPrefix(path, "b/") {
		return path[2:]
	}
	return path
}

func (e *Evaluator) SaveEvaluationResults(result *types.EvaluationResult) error {
	return SaveEvaluationResults(e.resultsDir, result)
}

func (e *Evaluator) createTestAgent(provider llm.Provider, promptVariant string) *agent.CodeReviewAgent {
	return agent.NewCodeReviewAgent(provider, e.parserRegistry, e.toolRegistry, promptVariant)
}

// mockHumanLoopTool is a mock implementation for evaluation that doesn't require user input
type mockHumanLoopTool struct{}

func (t *mockHumanLoopTool) Name() string {
	return string(tools.ToolNameHumanLoop)
}

func (t *mockHumanLoopTool) Description() string {
	return "Mock human loop tool for evaluation - returns empty response"
}

func (t *mockHumanLoopTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "The specific question to ask the developer",
			},
		},
		"required": []string{"question"},
	}
}

func (t *mockHumanLoopTool) Execute(args map[string]any) (any, error) {
	// For evaluation, we don't want interactive input, so return empty response
	return "No additional context available during evaluation.", nil
}
