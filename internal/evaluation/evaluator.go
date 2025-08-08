package evaluation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/agusespa/diffpector/internal/llm"
	"github.com/agusespa/diffpector/internal/types"
)

type EvaluationConfig = types.EvaluationConfig
type EvaluationRun = types.EvaluationRun
type EvaluationResult = types.EvaluationResult
type PromptVariant = types.PromptVariant

type Evaluator struct {
	suite      *types.EvaluationSuite
	resultsDir string
	templates  *template.Template
}

func NewEvaluator(suitePath string, resultsDir string) (*Evaluator, error) {
	suite, err := LoadEvaluationSuite(suitePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load evaluation suite: %w", err)
	}

	templates, err := LoadPromptTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt templates: %w", err)
	}

	return &Evaluator{
		suite:      suite,
		resultsDir: resultsDir,
		templates:  templates,
	}, nil
}

func (e *Evaluator) RunEvaluation(modelConfig llm.ProviderConfig, promptVariant string) (*EvaluationRun, error) {
	provider, err := llm.NewProvider(modelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	run := &EvaluationRun{
		Model:         modelConfig.Model,
		Provider:      string(modelConfig.Type),
		PromptVariant: promptVariant,
		StartTime:     time.Now(),
		Results:       make([]EvaluationResult, 0, len(e.suite.TestCases)),
	}

	fmt.Printf("Running evaluation for model: %s, prompt: %s\n", modelConfig.Model, promptVariant)

	for i, testCase := range e.suite.TestCases {
		fmt.Printf("  [%d/%d] %s... ", i+1, len(e.suite.TestCases), testCase.Name)

		result, err := e.runSingleTest(testCase, provider, promptVariant)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			result = &EvaluationResult{
				TestCase:      testCase,
				Model:         modelConfig.Model,
				PromptHash:    promptVariant,
				ExecutionTime: time.Since(run.StartTime),
				Success:       false,
				Errors:        []string{err.Error()},
				Timestamp:     time.Now(),
				Issues:        []types.Issue{},
			}
		} else {
			fmt.Printf("DONE (%.2fs, score: %.2f)\n", result.ExecutionTime.Seconds(), result.Score)
		}
		run.Results = append(run.Results, *result)
	}

	run.EndTime = time.Now()
	run.TotalDuration = run.EndTime.Sub(run.StartTime)
	CalculateSummary(run)

	return run, nil
}

func (e *Evaluator) runSingleTest(testCase types.TestCase, provider llm.Provider, promptVariant string) (*EvaluationResult, error) {
	startTime := time.Now()

	env, err := e.createTestEnvironment(testCase)
	if err != nil {
		return nil, fmt.Errorf("failed to create test environment: %w", err)
	}

	prompt, err := e.buildPrompt(promptVariant, env)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	review, err := provider.Generate(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate review: %w", err)
	}

	issues, err := e.parseIssues(review)
	if err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	score := e.calculateScore(testCase.Expected, issues)

	return &EvaluationResult{
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

func (e *Evaluator) createTestEnvironment(testCase types.TestCase) (*types.TestEnvironment, error) {
	if testCase.DiffFile == "" {
		return &types.TestEnvironment{
			Files: make(map[string]string),
			Diff:  "",
		}, nil
	}

	diffPath := filepath.Join(e.suite.BaseDir, testCase.DiffFile)
	diffContent, err := os.ReadFile(diffPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read diff file %s: %w", diffPath, err)
	}

	files, err := e.parseDiffToFiles(string(diffContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	return &types.TestEnvironment{
		Files: files,
		Diff:  string(diffContent),
	}, nil
}

func (e *Evaluator) parseDiffToFiles(diff string) (map[string]string, error) {
	filenames := e.extractFilenamesFromDiff(diff)

	files := make(map[string]string)
	for _, filename := range filenames {
		content, err := e.loadMockFileContent(filename, diff)
		if err != nil {
			return nil, fmt.Errorf("failed to load mock content for %s: %w", filename, err)
		}
		files[filename] = content
	}

	return files, nil
}

func (e *Evaluator) extractFilenamesFromDiff(diff string) []string {
	var filenames []string
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "+++") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				filename := strings.TrimPrefix(parts[1], "b/")
				if filename != "/dev/null" {
					filenames = append(filenames, filename)
				}
			}
		}
	}

	return filenames
}

func (e *Evaluator) loadMockFileContent(filename, diff string) (string, error) {
	if e.suite.MockFilesDir == "" {
		return "", fmt.Errorf("mock files directory not configured")
	}

	fullPath := filepath.Join(e.suite.MockFilesDir, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read mock file %s: %w", fullPath, err)
	}

	return string(content), nil
}

func (e *Evaluator) buildPrompt(variant string, env *types.TestEnvironment) (string, error) {
	data := struct {
		Diff           string
		FileContents   map[string]string
		SymbolAnalysis string
	}{
		Diff:           env.Diff,
		FileContents:   env.Files,
		SymbolAnalysis: "",
	}

	var result strings.Builder
	err := e.templates.ExecuteTemplate(&result, variant, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", variant, err)
	}

	return result.String(), nil
}

func (e *Evaluator) parseIssues(review string) ([]types.Issue, error) {
	review = strings.TrimSpace(review)

	if review == "APPROVED" {
		return []types.Issue{}, nil
	}

	if strings.HasPrefix(review, "```") {
		lines := strings.Split(review, "\n")
		if len(lines) > 2 {
			review = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var issues []types.Issue
	if err := json.Unmarshal([]byte(review), &issues); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return issues, nil
}

func (e *Evaluator) calculateScore(expected types.ExpectedResults, actual []types.Issue) float64 {
	scorer := NewSimpleScorer()
	return scorer.Score(expected, actual)
}

// SaveResults saves evaluation results to a JSON file
func (e *Evaluator) SaveResults(run *EvaluationRun) error {
	if err := os.MkdirAll(e.resultsDir, 0755); err != nil {
		return fmt.Errorf("failed to create results directory at %s: %w", e.resultsDir, err)
	}

	filename := fmt.Sprintf("eval_%s_%s_%d.json",
		run.Model, run.PromptVariant, run.StartTime.Unix())
	filepath := filepath.Join(e.resultsDir, filename)

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write results file to %s: %w", filepath, err)
	}

	fmt.Printf("Results saved to: %s\n", filepath)
	return nil
}

func LoadEvaluationSuite(path string) (*types.EvaluationSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read suite file at %s: %w", path, err)
	}

	var suite types.EvaluationSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse suite file %s: %w", path, err)
	}

	return &suite, nil
}

func LoadEvaluationConfigs(path string) ([]EvaluationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", path, err)
	}

	var configs []EvaluationConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return configs, nil
}

func CalculateSummary(r *types.EvaluationRun) {
	if len(r.Results) == 0 {
		return
	}

	var totalScore float64
	var successfulTests int
	for _, result := range r.Results {
		totalScore += result.Score
		if result.Success {
			successfulTests++
		}
	}

	r.AverageScore = totalScore / float64(len(r.Results))
	r.SuccessRate = (float64(successfulTests) / float64(len(r.Results))) * 100
}

func PrintSummary(r *types.EvaluationRun) {
	fmt.Printf("\n--- Summary for %s (%s) ---\n", r.Model, r.PromptVariant)
	fmt.Printf("  Average Score: %.2f\n", r.AverageScore)
	fmt.Printf("  Success Rate:  %.2f%%\n", r.SuccessRate)
	fmt.Printf("  Total Duration:  %.2fs\n", r.TotalDuration.Seconds())
	fmt.Println()
}
