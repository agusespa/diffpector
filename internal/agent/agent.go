package agent

import (
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/internal/llm"
	"github.com/agusespa/diffpector/internal/prompts"
	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/internal/types"
	"github.com/agusespa/diffpector/internal/utils"
	"github.com/agusespa/diffpector/pkg/config"
	"github.com/agusespa/diffpector/pkg/spinner"
)

type CodeReviewAgent struct {
	llmProvider    llm.Provider
	toolRegistry   *tools.Registry
	config         *config.Config
	promptVariant  string
	parserRegistry *tools.ParserRegistry
}

func NewCodeReviewAgent(provider llm.Provider, registry *tools.Registry, cfg *config.Config, parserRegistry *tools.ParserRegistry, promptVariant string) *CodeReviewAgent {
	return &CodeReviewAgent{
		llmProvider:    provider,
		toolRegistry:   registry,
		config:         cfg,
		promptVariant:  promptVariant,
		parserRegistry: parserRegistry,
	}
}

func (a *CodeReviewAgent) ReviewStagedChanges() error {
	fmt.Println("Starting code review on staged changes...")
	return a.executeReview()
}

func (a *CodeReviewAgent) executeReview() error {
	diffTool := a.toolRegistry.Get(tools.ToolNameGitDiff)

	diffResult, err := diffTool.Execute(map[string]any{})
	if err != nil {
		return fmt.Errorf("failed to get staged diff list: %w", err)
	}
	diffMap, ok := diffResult.(map[string]types.DiffData)
	if !ok {
		return fmt.Errorf("diff tool returned unexpected type: %T", diffResult)
	}

	changedFilesPaths := make([]string, 0, len(diffMap))
	for fileName := range diffMap {
		changedFilesPaths = append(changedFilesPaths, fileName)
	}

	fmt.Print("Files to be reviewed:")
	if len(changedFilesPaths) == 0 {
		fmt.Println("   - no staged changes found - use 'git add' to stage files for review")
		return nil
	}

	for _, file := range changedFilesPaths {
		fmt.Printf("\n   + %s", file)
	}

	primaryLanguage, err := a.ValidateAndDetectLanguage(changedFilesPaths)
	if err != nil {
		return err
	}

	return a.ReviewChanges(diffMap, primaryLanguage)
}

func (a *CodeReviewAgent) ValidateAndDetectLanguage(changedFiles []string) (string, error) {
	var primaryLanguage string

	for _, filePath := range changedFiles {
		parser := a.parserRegistry.GetParser(filePath)
		if parser != nil {
			lang := strings.ToLower(parser.Language())

			if primaryLanguage == "" {
				primaryLanguage = lang
			} else if primaryLanguage != lang {
				return "", fmt.Errorf("multi-language changes detected: %v and %v. Currently only single-language diffs are supported", primaryLanguage, lang)
			}
		} else if a.parserRegistry.IsKnownLanguage(filePath) {
			return "", fmt.Errorf("unsupported language file: %s. No parser available for this file type", filePath)
		}
	}

	return primaryLanguage, nil
}
func (a *CodeReviewAgent) ReviewChanges(diffMap map[string]types.DiffData, primaryLanguage string) error {
	_, err := a.ReviewChangesWithResult(diffMap, primaryLanguage, true)
	return err
}

func (a *CodeReviewAgent) ReviewChangesWithResult(diffMap map[string]types.DiffData, primaryLanguage string, printResults bool) (string, error) {
	err := a.UpdateDiffContext(diffMap, primaryLanguage)
	if err != nil {
		return "", fmt.Errorf("context gathering failed: %w", err)
	}

	review, err := a.GenerateReview(diffMap)
	if err != nil {
		return "", fmt.Errorf("generate review failed: %w", err)
	}

	// Optionally process and print results (for CLI usage)
	if printResults {
		err = a.ProcessAndPrintReview(review)
		if err != nil {
			return review, fmt.Errorf("failed to process review: %w", err)
		}
	}

	return review, nil
}

func (a *CodeReviewAgent) UpdateDiffContext(diffMap map[string]types.DiffData, primaryLanguage string) error {
	symbolContextTool := a.toolRegistry.Get(tools.ToolNameSymbolContext)
	for _, diffData := range diffMap {
		updatedDataResult, err := symbolContextTool.Execute(map[string]any{"diffData": diffData, "primaryLanguage": primaryLanguage})
		if err != nil {
			return fmt.Errorf("symbol analysis failed: %w", err)
		}
		updatedData, ok := updatedDataResult.(types.DiffData)
		if !ok {
			return fmt.Errorf("symbol context tool returned unexpected type: %T", updatedDataResult)
		}

		diffData.DiffContext = updatedData.DiffContext
	}

	return nil
}

func (a *CodeReviewAgent) GenerateReview(context map[string]types.DiffData) (string, error) {
	prompt, err := prompts.BuildPromptWithTemplate(a.promptVariant, context)
	if err != nil {
		return "", fmt.Errorf("failed to build review prompt: %w", err)
	}
	fmt.Println(prompt)

	spinner := spinner.New("Analyzing changes...")
	spinner.Start()

	review, err := a.llmProvider.Generate(prompt)
	spinner.Stop()

	if err != nil {
		return "", fmt.Errorf("failed to generate code review: %w", err)
	}

	return review, nil
}

func (a *CodeReviewAgent) ProcessAndPrintReview(review string) error {
	issues, err := utils.ParseIssuesFromResponse(review)
	if err != nil {
		return fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println("---")
		fmt.Println("✅ Code review passed - no issues found")
		return nil
	}

	writeTool := a.toolRegistry.Get(tools.ToolNameWriteFile)
	readTool := a.toolRegistry.Get(tools.ToolNameReadFile)

	reportGen := NewReportGenerator(readTool, writeTool)

	criticalCount, warningCount, minorCount, err := reportGen.GenerateMarkdownReport(issues)
	if err != nil {
		return err
	}

	PrintReviewSummary(criticalCount, warningCount, minorCount)

	return nil
}
