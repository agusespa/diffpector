package agent

import (
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/internal/llm"
	"github.com/agusespa/diffpector/internal/prompts"
	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/internal/types"
	"github.com/agusespa/diffpector/internal/utils"
	"github.com/agusespa/diffpector/pkg/spinner"
)

type CodeReviewAgent struct {
	llmProvider    llm.Provider
	promptVariant  string
	parserRegistry *tools.ParserRegistry
	toolRegistry   *tools.ToolRegistry
}

func NewCodeReviewAgent(provider llm.Provider, parserRegistry *tools.ParserRegistry, registry *tools.ToolRegistry, promptVariant string) *CodeReviewAgent {
	return &CodeReviewAgent{
		llmProvider:    provider,
		promptVariant:  promptVariant,
		parserRegistry: parserRegistry,
		toolRegistry:   registry,
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
		fmt.Println("- no staged changes found (use 'git add' to stage files for review)")
		return nil
	}

	for _, file := range changedFilesPaths {
		fmt.Printf("\n- %s", file)
	}
	fmt.Println()

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
	var allIssues []types.Issue
	totalFiles := len(diffMap)
	currentFile := 0

	fmt.Println()
	fmt.Printf("Starting sequential review of %d file(s):", totalFiles)
	fmt.Println()

	for filePath, diffData := range diffMap {
		currentFile++
		fmt.Printf("- [%d/%d] Reviewing %s\n", currentFile, totalFiles, filePath)

		singleFileMap := map[string]types.DiffData{filePath: diffData}

		ctxSpinner := spinner.New("Gathering context...")
		ctxSpinner.Start()
		err := a.UpdateDiffContext(singleFileMap, primaryLanguage)
		ctxSpinner.Stop()
		if err != nil {
			fmt.Printf("  [!] Context gathering failed: %v\n", err)
			continue
		}

		diffMap[filePath] = singleFileMap[filePath]

		review, err := a.GenerateReview(singleFileMap)
		if err != nil {
			fmt.Printf("  [!] Review failed: %v\n", err)
			continue
		}

		issues, err := utils.ParseIssuesFromResponse(review)
		if err != nil {
			fmt.Printf("  [!] Failed to parse review: %v\n", err)
			continue
		}

		if len(issues) == 0 {
			fmt.Printf("  [✓] No issues found\n")
		} else {
			fmt.Printf("  [✕] Found %d issue(s)\n", len(issues))
		}

		allIssues = append(allIssues, issues...)
	}

	fmt.Println()
	fmt.Printf("Review complete - analyzed %d file(s)\n", totalFiles)

	return a.GenerateFinalReport(allIssues)
}

func (a *CodeReviewAgent) ReviewChangesWithResult(diffMap map[string]types.DiffData, primaryLanguage string, printResults bool) (string, error) {
	ctxSpinner := spinner.New("Gathering context...")
	ctxSpinner.Start()
	err := a.UpdateDiffContext(diffMap, primaryLanguage)
	ctxSpinner.Stop()
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

	for key, diffData := range diffMap {
		updatedDataResult, err := symbolContextTool.Execute(map[string]any{"diffData": diffData, "primaryLanguage": primaryLanguage})
		if err != nil {
			return fmt.Errorf("symbol analysis failed: %w", err)
		}
		updatedData, ok := updatedDataResult.(types.DiffData)
		if !ok {
			return fmt.Errorf("symbol context tool returned unexpected type: %T", updatedDataResult)
		}

		diffData.DiffContext = updatedData.DiffContext
		diffData.AffectedSymbols = updatedData.AffectedSymbols

		diffMap[key] = diffData
	}

	return nil
}

func (a *CodeReviewAgent) GenerateReview(diffMap map[string]types.DiffData) (string, error) {
	var combinedContext strings.Builder

	for path, data := range diffMap {
		combinedContext.WriteString(fmt.Sprintf(">>> Diff for changed file: %s\n%s\n", path, data.Diff))

		if data.DiffContext != "" {
			combinedContext.WriteString(fmt.Sprintf("\n>>>> Expanded Diff Context\n%s\n", data.DiffContext))
		}

		combinedContext.WriteString("\n>>>> Affected Symbols\n")
		for _, usage := range data.AffectedSymbols {
			combinedContext.WriteString(usage.Snippets)
		}
	}

	prompt, err := prompts.BuildPromptWithTemplate(a.promptVariant, combinedContext.String())
	if err != nil {
		return "", fmt.Errorf("failed to build review prompt: %w", err)
	}

	history := []llm.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	humanLoopTool := a.toolRegistry.Get(tools.ToolNameHumanLoop)
	availableTools := a.toLLMTools(humanLoopTool)

	maxIterations := 10

	for range maxIterations {
		spinner := spinner.New("Analyzing changes...")
		spinner.Start()

		response, err := a.llmProvider.ChatWithTools(history, availableTools)
		spinner.Stop()

		if err != nil {
			return "", fmt.Errorf("failed to generate code review: %w", err)
		}

		if len(response.ToolCalls) > 0 {
			for _, toolCall := range response.ToolCalls {
				if toolCall.Name == string(tools.ToolNameHumanLoop) {
					question, ok := toolCall.Arguments["question"].(string)
					if !ok {
						return "", fmt.Errorf("invalid question argument in tool call")
					}

					userResponse, err := humanLoopTool.Execute(map[string]any{
						"question": question,
					})
					if err != nil {
						return "", fmt.Errorf("failed to get user input: %w", err)
					}

					userInput, ok := userResponse.(string)
					if !ok {
						return "", fmt.Errorf("human loop tool returned unexpected type: %T", userResponse)
					}

					history = append(history, llm.Message{
						Role:    "assistant",
						Content: question,
					})
					history = append(history, llm.Message{
						Role:    "user",
						Content: userInput,
					})
				}
			}
		} else if response.Content != "" {
			// No tool calls and we have content - this is the final response
			return response.Content, nil
		} else {
			// No tool calls and no content
			return "", fmt.Errorf("LLM returned empty response without tool calls")
		}
	}

	return "", fmt.Errorf("conversation exceeded maximum iterations without completion")
}

func (a *CodeReviewAgent) toLLMTools(toolsToConvert ...tools.Tool) []llm.Tool {
	llmTools := make([]llm.Tool, len(toolsToConvert))
	for i, tool := range toolsToConvert {
		llmTools[i] = llm.Tool{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Schema(),
			},
		}
	}
	return llmTools
}

func (a *CodeReviewAgent) ProcessAndPrintReview(review string) error {
	issues, err := utils.ParseIssuesFromResponse(review)
	if err != nil {
		return fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return a.GenerateFinalReport(issues)
}

func (a *CodeReviewAgent) GenerateFinalReport(allIssues []types.Issue) error {
	writeTool := a.toolRegistry.Get(tools.ToolNameWriteFile)
	readTool := a.toolRegistry.Get(tools.ToolNameReadFile)
	reportGen := NewReportGenerator(readTool, writeTool)

	criticalCount, warningCount, minorCount, err := reportGen.GenerateMarkdownReport(allIssues)
	if err != nil {
		return err
	}

	PrintReviewSummary(criticalCount, warningCount, minorCount)

	return nil
}
