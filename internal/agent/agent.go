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

func NewCodeReviewAgent(provider llm.Provider, registry *tools.Registry, cfg *config.Config, parserRegistry *tools.ParserRegistry) *CodeReviewAgent {
	return &CodeReviewAgent{
		llmProvider:    provider,
		toolRegistry:   registry,
		config:         cfg,
		promptVariant:  prompts.DEFAULT_PROMPT,
		parserRegistry: parserRegistry,
	}
}

func (a *CodeReviewAgent) ReviewStagedChanges() error {
	fmt.Println("Starting code review on staged changes...")
	return a.executeReview()
}

func (a *CodeReviewAgent) executeReview() error {
	// Step 1: Get staged files
	stagedFilesTool := a.toolRegistry.Get(tools.ToolNameGitStagedFiles)

	stagedFilesOutput, err := stagedFilesTool.Execute(map[string]any{})
	if err != nil {
		return fmt.Errorf("failed to get staged files: %w", err)
	}

	changedFiles := utils.ParseStagedFiles(stagedFilesOutput)

	fmt.Print("Files to be reviewed:")
	if len(changedFiles) == 0 {
		fmt.Println("   ✕ no staged changes found - use 'git add' to stage files for review")
		return nil
	}

	for _, file := range changedFiles {
		fmt.Printf("\n   ✓ %s", file)
	}

	// Step 2: Validate language support and detect primary language
	primaryLanguage, err := a.validateAndDetectLanguage(changedFiles)
	if err != nil {
		fmt.Printf("\n   ✕ %v\n", err)
		return err
	}

	// Step 3: Get diff for analysis
	diffTool := a.toolRegistry.Get(tools.ToolNameGitDiff)

	diff, err := diffTool.Execute(map[string]any{})
	if err != nil {
		return fmt.Errorf("failed to get staged diff: %w", err)
	}

	// Step 4: Enhanced context gathering with symbol analysis
	fmt.Println()
	contextSpinner := spinner.New("Analyzing symbols and gathering context...")
	contextSpinner.Start()

	reviewContext, err := a.GatherEnhancedContext(diff, changedFiles, primaryLanguage)
	contextSpinner.Stop()

	if err != nil {
		fmt.Printf("Context gathering failed: %v\n", err)
		return err
	}

	a.PrintContextSummary(reviewContext)
	fmt.Println()

	err = a.GenerateReview(reviewContext)
	if err != nil {
		fmt.Printf("Generate Review failed: %v\n", err)
		return err
	}

	return nil
}

func (a *CodeReviewAgent) GatherEnhancedContext(diff string, changedFiles []string, primaryLanguage string) (*types.ReviewContext, error) {
	fileContents, err := a.readFileContents(changedFiles)
	if err != nil {
		return nil, err
	}

	context := &types.ReviewContext{
		Diff:         diff,
		ChangedFiles: changedFiles,
		FileContents: fileContents,
	}

	symbolContextTool := a.toolRegistry.Get(tools.ToolNameSymbolContext)
	symbolAnalysis, err := symbolContextTool.Execute(map[string]any{
		"diff":             diff,
		"file_contents":    context.FileContents,
		"primary_language": primaryLanguage,
	})
	if err != nil {
		return nil, fmt.Errorf("symbol analysis failed: %w", err)
	}
	context.SymbolAnalysis = symbolAnalysis

	return context, nil
}

func (a *CodeReviewAgent) readFileContents(files []string) (map[string]string, error) {
	fileContents := make(map[string]string)
	readTool := a.toolRegistry.Get(tools.ToolNameReadFile)

	for _, file := range files {
		content, err := readTool.Execute(map[string]any{"filename": file})
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file, err)
		}
		fileContents[file] = content
	}

	return fileContents, nil
}

func (a *CodeReviewAgent) PrintContextSummary(context *types.ReviewContext) {
	fmt.Print("Context gathered from:")

	totalFiles := len(context.ChangedFiles)
	if totalFiles > 0 {
		for _, file := range context.ChangedFiles {
			fmt.Printf("\n   ✓ %s (changed)", file)
		}

		if context.SymbolAnalysis != "" {
			fmt.Printf("\n   ✓ Symbol analysis completed")
		}
	} else {
		fmt.Println("\n   ✕ No additional context found")
	}
}

func (a *CodeReviewAgent) GenerateReview(context *types.ReviewContext) error {
	prompt, err := prompts.BuildPromptWithTemplate(a.promptVariant, context)
	if err != nil {
		return fmt.Errorf("failed to build review prompt: %w", err)
	}

	spinner := spinner.New("Analyzing changes...")
	spinner.Start()

	review, err := a.llmProvider.Generate(prompt)
	if err != nil {
		spinner.Stop()
		return fmt.Errorf("failed to generate code review: %w", err)
	}

	issues, err := utils.ParseIssuesFromResponse(review)
	if err != nil {
		spinner.Stop()
		return fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println("---")
		fmt.Println("✅ Code review passed - no issues found")
		spinner.Stop()
		return nil
	}

	writeTool := a.toolRegistry.Get(tools.ToolNameWriteFile)
	readTool := a.toolRegistry.Get(tools.ToolNameReadFile)

	reportGen := NewReportGenerator(readTool, writeTool)

	criticalCount, warningCount, minorCount, err := reportGen.GenerateMarkdownReport(issues)
	if err != nil {
		return err
	}

	spinner.Stop()

	PrintReviewSummary(criticalCount, warningCount, minorCount)

	return nil
}

// validateAndDetectLanguage checks if all programming language files are supported and returns the primary language
func (a *CodeReviewAgent) validateAndDetectLanguage(changedFiles []string) (string, error) {
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
