package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/internal/evaluation"
	"github.com/agusespa/diffpector/internal/llm"
	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/internal/types"
	"github.com/agusespa/diffpector/pkg/config"
	"github.com/agusespa/diffpector/pkg/spinner"
)

type CodeReviewAgent struct {
	llmProvider   llm.Provider
	toolRegistry  *tools.Registry
	config        *config.Config
	promptVariant string
}

func NewCodeReviewAgent(provider llm.Provider, registry *tools.Registry, cfg *config.Config) *CodeReviewAgent {
	return &CodeReviewAgent{
		llmProvider:   provider,
		toolRegistry:  registry,
		config:        cfg,
		promptVariant: "default",
	}
}

func (a *CodeReviewAgent) ReviewStagedChanges() error {
	fmt.Println("Starting code review on staged changes...")
	return a.executeReview()
}

func (a *CodeReviewAgent) executeReview() error {
	// Step 1: Get staged files
	stagedFilesTool, exists := a.toolRegistry.Get("git_staged_files")
	if !exists {
		return fmt.Errorf("git_staged_files tool not available")
	}

	stagedFilesOutput, err := stagedFilesTool.Execute(map[string]any{})
	if err != nil {
		return fmt.Errorf("failed to get staged files: %w", err)
	}

	changedFiles := ParseStagedFiles(stagedFilesOutput)

	fmt.Print("Files to be reviewed:")
	if len(changedFiles) == 0 {
		fmt.Println("   ✕ no staged changes found - use 'git add' to stage files for review")
		return nil
	}

	for _, file := range changedFiles {
		fmt.Printf("\n   ✓ %s", file)
	}

	// Step 2: Get diff for analysis
	diffTool, exists := a.toolRegistry.Get("git_diff")
	if !exists {
		return fmt.Errorf("git_diff tool not available")
	}

	diff, err := diffTool.Execute(map[string]any{})
	if err != nil {
		return fmt.Errorf("failed to get staged diff: %w", err)
	}

	// Step 3: Enhanced context gathering with symbol analysis
	fmt.Println()
	contextSpinner := spinner.New("Analyzing symbols and gathering context...")
	contextSpinner.Start()

	reviewContext, err := a.GatherEnhancedContext(diff, changedFiles)
	contextSpinner.Stop()

	if err != nil {
		fmt.Printf("   ⚠️  Context gathering failed: %v\n", err)
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

func (a *CodeReviewAgent) GatherEnhancedContext(diff string, changedFiles []string) (*types.ReviewContext, error) {
	context := &types.ReviewContext{
		Diff:         diff,
		ChangedFiles: changedFiles,
		FileContents: make(map[string]string),
	}

	readTool, exists := a.toolRegistry.Get("read_file")
	if !exists {
		return nil, fmt.Errorf("read_file tool not available")
	}

	// Step 1: Read changed files first (needed for symbol analysis)
	for _, file := range changedFiles {
		content, err := readTool.Execute(map[string]any{"filename": file})
		if err != nil {
			fmt.Printf("   ⚠️  Failed to read %s: %v\n", file, err)
			continue
		}
		context.FileContents[file] = content
	}

	// Step 2: Perform symbol analysis with file contents
	symbolContextTool, symbolExists := a.toolRegistry.Get("symbol_context")
	if symbolExists {
		symbolAnalysis, err := symbolContextTool.Execute(map[string]any{
			"diff":          diff,
			"changed_files": changedFiles,
			"file_contents": context.FileContents,
		})
		if err != nil {
			fmt.Printf("   ⚠️  Symbol analysis failed: %v\n", err)
		} else {
			context.SymbolAnalysis = symbolAnalysis
		}
	}

	return context, nil
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
	prompt, err := evaluation.BuildPromptWithTemplate(a.promptVariant, context)
	if err != nil {
		return fmt.Errorf("failed to build review prompt: %w", err)
	}

	analySisSpinner := spinner.New("Analyzing changes...")
	analySisSpinner.Start()

	review, err := a.llmProvider.Generate(prompt)
	if err != nil {
		analySisSpinner.Stop()
		return fmt.Errorf("failed to generate code review: %w", err)
	}

	analySisSpinner.Stop()

	if strings.TrimSpace(review) == "APPROVED" {
		fmt.Println("---")
		fmt.Println("✅ Code review passed - no issues found")
		return nil
	}

	var issues []types.Issue
	// Try parsing as-is first, then fallback to cleaning if needed
	if err := json.Unmarshal([]byte(review), &issues); err != nil {
		// Fallback: try cleaning markdown formatting
		cleanedReview := review
		startIndex := strings.Index(cleanedReview, "[")
		if startIndex == -1 {
			startIndex = strings.Index(cleanedReview, "{")
		}

		if startIndex != -1 {
			endIndex := strings.LastIndex(cleanedReview, "]")
			if endIndex == -1 {
				endIndex = strings.LastIndex(cleanedReview, "}")
			}

			if endIndex != -1 {
				cleanedReview = cleanedReview[startIndex : endIndex+1]
			}
		}

		if err := json.Unmarshal([]byte(cleanedReview), &issues); err != nil {
			return fmt.Errorf("failed to decode llm response into JSON. Response: %s, Error: %w", review, err)
		}
	}

	if err := a.BuildAndWriteMarkdownReport(issues); err != nil {
		return err
	}

	return nil
}

func (a *CodeReviewAgent) BuildAndWriteMarkdownReport(issues []types.Issue) error {
	writeTool, exists := a.toolRegistry.Get("write_file")
	if !exists {
		return fmt.Errorf("write_file tool not available")
	}
	readTool, exists := a.toolRegistry.Get("read_file")
	if !exists {
		return fmt.Errorf("read_file tool not available")
	}

	reportGen := NewReportGenerator(readTool, writeTool)

	if err := reportGen.GenerateMarkdownReport(issues); err != nil {
		return err
	}

	criticalCount, warningCount, minorCount := CountIssuesBySeverity(issues)
	PrintReviewSummary(criticalCount, warningCount, minorCount)

	return nil
}
