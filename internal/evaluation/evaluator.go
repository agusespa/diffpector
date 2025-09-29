package evaluation

import (
	"fmt"
	"time"

	"github.com/agusespa/diffpector/internal/agent"
	"github.com/agusespa/diffpector/internal/llm"
	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/internal/types"
	"github.com/agusespa/diffpector/internal/utils"
	"github.com/agusespa/diffpector/pkg/config"
)

type EvaluationConfig = types.EvaluationConfig
type EvaluationRun = types.EvaluationRun
type TestCaseResult = types.TestCaseResult

type Evaluator struct {
	suite          *types.EvaluationSuite
	envBuilder     *TestEnvironmentBuilder
	statsCalc      *StatisticsCalculator
	resultsMgr     *ResultsManager
	toolRegistry   *tools.Registry
	parserRegistry *tools.ParserRegistry
}

func NewEvaluator(suitePath string, resultsDir string) (*Evaluator, error) {
	suite, err := LoadSuite(suitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load evaluation suite: %w", err)
	}

	parserRegistry := tools.NewParserRegistry()
	toolRegistry := tools.NewRegistry()

	toolRegistry.Register(tools.ToolNameSymbolContext, tools.NewSymbolContextTool(".", parserRegistry))

	return &Evaluator{
		suite:          suite,
		envBuilder:     NewTestEnvironmentBuilder(suite.BaseDir, suite.MockFilesDir),
		statsCalc:      NewStatisticsCalculator(),
		resultsMgr:     NewResultsManager(resultsDir),
		toolRegistry:   toolRegistry,
		parserRegistry: parserRegistry,
	}, nil
}

func (e *Evaluator) RunEvaluation(modelConfig llm.ProviderConfig, promptVariant string, numRuns int) (*types.EvaluationResult, error) {
	if numRuns <= 0 {
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

	e.statsCalc.CalculateEvaluationStats(result)

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
	e.statsCalc.CalculateRunSummary(run)

	return run, nil
}

func (e *Evaluator) runSingleTest(testCase types.TestCase, provider llm.Provider, promptVariant string) (*TestCaseResult, error) {
	startTime := time.Now()

	// Load test environment (diff and files)
	env, err := e.envBuilder.CreateTestEnvironment(testCase)
	if err != nil {
		return nil, fmt.Errorf("failed to create test environment: %w", err)
	}

	// Create agent with the test provider and prompt variant
	agent := e.createTestAgent(provider, promptVariant)
	diffMap, err := e.createDiffMap(env)
	if err != nil {
		return nil, fmt.Errorf("failed to create diff map: %w", err)
	}

	// Use the complete agent review process (same as main app) but get the result
	review, err := agent.ReviewChangesWithResult(diffMap, "", false)
	if err != nil {
		return nil, fmt.Errorf("agent review failed: %w", err)
	}

	// Use the shared parsing logic
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

func (e *Evaluator) createDiffMap(env *types.TestEnvironment) (map[string]types.DiffData, error) {
	diffMap := make(map[string]types.DiffData)
	filenames := e.envBuilder.ExtractFilenamesFromDiff(env.Diff)

	for _, filename := range filenames {
		absolutePath := e.envBuilder.getAbsPath(filename)
		diffMap[filename] = types.DiffData{
			AbsolutePath: absolutePath,
			Diff:         env.Diff,
		}
	}

	return diffMap, nil
}

func (e *Evaluator) SaveEvaluationResults(result *types.EvaluationResult) error {
	return e.resultsMgr.SaveEvaluationResults(result)
}

func (e *Evaluator) createTestAgent(provider llm.Provider, promptVariant string) *agent.CodeReviewAgent {
	cfg := &config.Config{}
	return agent.NewCodeReviewAgent(provider, e.toolRegistry, cfg, e.parserRegistry, promptVariant)
}
