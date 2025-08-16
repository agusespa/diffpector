package agent

import (
	"fmt"
	"path/filepath"
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



// GenerateReview generates the review and returns the raw LLM response
func (a *CodeReviewAgent) GenerateReview(context *types.ReviewContext) (string, error) {
	prompt, err := prompts.BuildPromptWithTemplate(a.promptVariant, context)
	if err != nil {
		return "", fmt.Errorf("failed to build review prompt: %w", err)
	}

	spinner := spinner.New("Analyzing changes...")
	spinner.Start()

	review, err := a.llmProvider.Generate(prompt)
	spinner.Stop()

	if err != nil {
		return "", fmt.Errorf("failed to generate code review: %w", err)
	}

	return review, nil
}

// ProcessAndPrintReview processes the LLM response and prints the results
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

func (a *CodeReviewAgent) ReviewStagedChanges() error {
	fmt.Println("Starting code review on staged changes...")
	return a.executeReview()
}

// ReviewChanges performs code review on provided diff and file contents
func (a *CodeReviewAgent) ReviewChanges(diff string, fileContents map[string]string, changedFiles []string) error {
	_, err := a.ReviewChangesWithResult(diff, fileContents, changedFiles, true)
	return err
}

// ReviewChangesWithResult performs code review and optionally returns the result
func (a *CodeReviewAgent) ReviewChangesWithResult(diff string, fileContents map[string]string, changedFiles []string, printResults bool) (string, error) {
	// Step 1: Validate language support and detect primary language
	primaryLanguage, err := a.ValidateAndDetectLanguage(changedFiles)
	if err != nil {
		return "", err
	}

	// Step 2: Enhanced context gathering with symbol analysis
	reviewContext, err := a.GatherEnhancedContextWithFiles(diff, changedFiles, primaryLanguage, fileContents)
	if err != nil {
		return "", fmt.Errorf("context gathering failed: %w", err)
	}

	// Step 3: Generate review using the same process as the main app
	review, err := a.GenerateReview(reviewContext)
	if err != nil {
		return "", fmt.Errorf("generate review failed: %w", err)
	}

	// Step 4: Optionally process and print results (for CLI usage)
	if printResults {
		err = a.ProcessAndPrintReview(review)
		if err != nil {
			return review, fmt.Errorf("failed to process review: %w", err)
		}
	}

	return review, nil
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

	// Step 2: Get diff for analysis
	diffTool := a.toolRegistry.Get(tools.ToolNameGitDiff)

	diff, err := diffTool.Execute(map[string]any{})
	if err != nil {
		return fmt.Errorf("failed to get staged diff: %w", err)
	}

	// Step 3: Read file contents
	fileContents, err := a.readFileContents(changedFiles)
	if err != nil {
		return err
	}

	// Step 4: Use the core review logic
	return a.ReviewChanges(diff, fileContents, changedFiles)
}

func (a *CodeReviewAgent) GatherEnhancedContext(diff string, changedFiles []string, primaryLanguage string) (*types.ReviewContext, error) {
	return a.GatherEnhancedContextWithFiles(diff, changedFiles, primaryLanguage, nil)
}

func (a *CodeReviewAgent) GatherEnhancedContextWithFiles(diff string, changedFiles []string, primaryLanguage string, preloadedFiles map[string]string) (*types.ReviewContext, error) {
	var fileContents map[string]string
	var err error

	if preloadedFiles != nil {
		fileContents = preloadedFiles
	} else {
		fileContents, err = a.readFileContents(changedFiles)
		if err != nil {
			return nil, err
		}
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

		absPath, err := filepath.Abs(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", file, err)
		}

		fileContents[absPath] = content
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



// ValidateAndDetectLanguage checks if all programming language files are supported and returns the primary language
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
