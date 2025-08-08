package main

import (
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/agusespa/diffpector/internal/agent"
	"github.com/agusespa/diffpector/internal/llm"
	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/pkg/config"
)

func main() {
	configFile := flag.String("config", "config.json", "Path to configuration file")
	showHelp := flag.Bool("help", false, "Show help message")
	flag.Parse()

	fmt.Println("")
	fmt.Println("=========================")
	fmt.Println(" Diffpector Review Agent ")
	fmt.Println("=========================")
	fmt.Println("")

	if *showHelp {
		fmt.Println("Code Review Agent - AI-powered code review for staged Git changes")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Printf("  %s [options]\n", os.Args[0])
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Printf("  %s                           # Use default config.json\n", os.Args[0])
		fmt.Printf("  %s --config custom.json      # Use custom config\n", os.Args[0])
		fmt.Printf("  %s --allow-remote            # Allow remote providers\n", os.Args[0])
		fmt.Println()
		return
	}

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config from %s: %v\n", *configFile, err)
		os.Exit(1)
	}

	if !slices.Contains(llm.SupportedProviders, cfg.LLM.Provider) {
		fmt.Fprintf(os.Stderr, "Error: Unsupported LLM provider: %v\n", err)
		os.Exit(1)
	}

	providerConfig := llm.ProviderConfig{
		Type:    llm.ProviderType(cfg.LLM.Provider),
		Model:   cfg.LLM.Model,
		BaseURL: cfg.LLM.BaseURL,
	}

	llmProvider, err := llm.NewProvider(providerConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create LLM provider: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Using %s with model: %s\n", cfg.LLM.Provider, llmProvider.GetModel())

	toolRegistry := tools.NewRegistry()

	toolsToRegister := map[string]tools.Tool{
		"git_diff":         &tools.GitDiffTool{},
		"git_staged_files": &tools.GitStagedFilesTool{},
		"git_grep":         &tools.GitGrepTool{},
		"write_file":       &tools.WriteFileTool{},
		"read_file":        &tools.ReadFileTool{},
		"append_file":      &tools.AppendFileTool{},
		"symbol_context":   tools.NewSymbolContextTool(),
	}

	for name, tool := range toolsToRegister {
		toolRegistry.Register(name, tool)
	}

	codeReviewAgent := agent.NewCodeReviewAgent(llmProvider, toolRegistry, cfg)

	fmt.Println("")
	fmt.Println("-------------------------")
	fmt.Println("")

	if err := codeReviewAgent.ReviewStagedChanges(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Code review failed: %v\n", err)
		os.Exit(1)
	}
}
