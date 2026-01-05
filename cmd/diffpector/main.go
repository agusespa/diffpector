package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/agusespa/diffpector/internal/agent"
	"github.com/agusespa/diffpector/internal/llm"
	"github.com/agusespa/diffpector/internal/prompts"
	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/internal/utils"
	"github.com/agusespa/diffpector/pkg/config"
)

func main() {
	fmt.Println("")
	fmt.Println("=========================")
	fmt.Println(" Diffpector Review Agent ")
	fmt.Println("=========================")
	fmt.Println("")

	if err := runMainMenu(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMainMenu() error {
	for {
		fmt.Println("Which mode do you want to run?")
		fmt.Println()
		fmt.Println("1. DIFF MODE: Review staged changes (local Git diff)")
		fmt.Println("2. PR MODE: Review remote branch (compare with current branch)")
		fmt.Println("3. Help")
		fmt.Println("0. Exit")
		fmt.Println()
		fmt.Print("Enter your choice: ")

		var choice string
		_, err := fmt.Scanln(&choice)
		if err != nil {
			fmt.Printf("Error reading input: %v", err)
			continue
		}
		fmt.Println()

		switch choice {
		case "1":
			return runCodeReview("diff", "")
		case "2":
			fmt.Println("Branch Review")
			fmt.Println("-------------")
			fmt.Println("Note: Make sure you're running from the Git repository root directory.")
			fmt.Println("The tool will fetch the branch and compare it with your current branch.")
			fmt.Println("Ensure your SSH keys are set up for the Git platform.")
			fmt.Println()
			fmt.Print("Enter the branch name to review (e.g., feature/new-login): ")
			var branchName string
			_, err := fmt.Scanln(&branchName)
			if err != nil {
				fmt.Printf("Error reading input: %v", err)
				continue
			}

			if branchName == "" {
				fmt.Println("Error: Branch name is required")
				fmt.Println()
				continue
			}

			fmt.Println()
			return runCodeReview("branch", branchName)
		case "3":
			showHelp()
			fmt.Println()
			continue
		case "0":
			fmt.Println("Goodbye!")
			return nil
		default:
			fmt.Printf("Invalid choice '%s'. Please enter a number between 0-3.\n", choice)
			fmt.Println()
			continue
		}
	}
}

func runCodeReview(mode, target string) error {
	reportErr := agent.NotifyUserIfReportNotIgnored(".gitignore")
	if reportErr != nil {
		return fmt.Errorf("report check failed: %w", reportErr)
	}

	cfg, err := config.LoadConfig("diffpectrc.json")
	if err != nil {
		return fmt.Errorf("failed to load config from diffpectrc.json: %w", err)
	}

	if !slices.Contains(llm.SupportedProviders, cfg.LLM.Provider) {
		return fmt.Errorf("unsupported LLM provider: %s", cfg.LLM.Provider)
	}

	if err := utils.ValidateModel(cfg.LLM.Model); err != nil {
		return fmt.Errorf("model validation failed: %w", err)
	}

	providerConfig := llm.ProviderConfig{
		Type:    llm.ProviderType(cfg.LLM.Provider),
		Model:   cfg.LLM.Model,
		BaseURL: cfg.LLM.BaseURL,
		APIKey:  cfg.LLM.APIKey,
	}

	llmProvider, err := llm.NewProvider(providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}

	modelDisplay := llmProvider.GetModel()
	if modelDisplay == "" || modelDisplay == "llama.cpp" {
		modelDisplay = "model loaded at server startup"
	}
	fmt.Printf("Using %s API with %s\n\n", cfg.LLM.Provider, modelDisplay)

	parserRegistry := tools.NewParserRegistry()
	toolRegistry := tools.NewToolRegistry()
	rootDir := "."
	toolsToRegister := map[tools.ToolName]tools.Tool{
		tools.ToolNameGitDiff:       &tools.GitDiffTool{},
		tools.ToolNameGitGrep:       &tools.GitGrepTool{},
		tools.ToolNameWriteFile:     &tools.WriteFileTool{},
		tools.ToolNameReadFile:      &tools.ReadFileTool{},
		tools.ToolNameHumanLoop:     &tools.HumanLoopTool{},
		tools.ToolNameSymbolContext: tools.NewSymbolContextTool(rootDir, parserRegistry),
	}

	for name, tool := range toolsToRegister {
		toolRegistry.Register(name, tool)
	}

	codeReviewAgent := agent.NewCodeReviewAgent(llmProvider, parserRegistry, toolRegistry, prompts.DEFAULT_PROMPT)

	switch mode {
	case "diff":
		return codeReviewAgent.ReviewStagedChanges()
	case "branch":
		return fmt.Errorf("%s mode is not supported yet", mode)
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
}

func showHelp() {
	fmt.Println("Diffpector Review Agent")
	fmt.Println("-----------------------")
	fmt.Println()
	fmt.Println("AI-powered code review for staged Git changes and pull/merge requests")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("• Local-only operation with Ollama - no cloud dependencies")
	fmt.Println("• Multi-language support: Go, TypeScript, Java, Python, C")
	fmt.Println("• Symbol-aware context analysis")
	fmt.Println("• Support for remote branch comparison")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("• Place a diffrectrc.json file in the current directory")
	fmt.Println("• For branch reviews, ensure SSH keys are set up for your Git platform")
	fmt.Println("• Always run from your Git repository root for proper symbol analysis")
	fmt.Println()
}
