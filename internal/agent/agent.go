package agent

import (
	"fmt"
	"os"
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

// TODO reveiw code below this line

func (a *CodeReviewAgent) GenerateReview(context *types.ReviewContext) (string, error) {
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

func (a *CodeReviewAgent) ReviewBranch(branchName string) error {
	fmt.Printf("Starting code review on branch: %s\n", branchName)
	return a.executeBranchReview(branchName)
}

func (a *CodeReviewAgent) executeBranchReview(branchName string) error {
	// Step 1: Verify we're in a Git repository for symbol context
	if err := a.verifyGitRepository(); err != nil {
		return fmt.Errorf("branch review requires running from a Git repository: %w", err)
	}

	// Step 2: Fetch branch and get diff
	branchFetchTool := a.toolRegistry.Get(tools.ToolNameBranchFetch)

	diff, err := branchFetchTool.Execute(map[string]any{"branch_name": branchName})
	if err != nil {
		return fmt.Errorf("failed to fetch branch data: %w", err)
	}

	// Step 3: Extract changed files from diff
	changedFiles := a.extractFilesFromDiff(diff)

	fmt.Printf("Branch Review: %s\n", branchName)

	fmt.Print("Files to be reviewed:")
	if len(changedFiles) == 0 {
		fmt.Println("   ✕ no changed files found in branch")
		return nil
	}

	for _, file := range changedFiles {
		fmt.Printf("\n   ✓ %s", file)
	}
	fmt.Println()

	// Step 4: Use the core review logic
	return a.ReviewChanges(diff, changedFiles)
}

func (a *CodeReviewAgent) verifyGitRepository() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository (no .git directory found)")
	}
	return nil
}

func (a *CodeReviewAgent) readLocalFileContents(files []string) (map[string]string, error) {
	fileContents := make(map[string]string)
	readTool := a.toolRegistry.Get(tools.ToolNameReadFile)

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}

		content, err := readTool.Execute(map[string]any{"filename": file})
		if err != nil {
			fmt.Printf("Warning: Could not read local file %s: %v\n", file, err)
			continue
		}

		absPath, err := filepath.Abs(file)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", file, err)
		}

		fileContents[absPath] = content
	}

	return fileContents, nil
}

func (a *CodeReviewAgent) extractFilesFromDiff(diff string) []string {
	var files []string
	lines := strings.SplitSeq(diff, "\n")

	for line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				filename := strings.TrimPrefix(parts[3], "b/")
				files = append(files, filename)
			}
		}
	}

	return files
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
