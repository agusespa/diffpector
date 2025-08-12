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

var version = "dev"

func main() {
	showHelp := flag.Bool("help", false, "Show help message")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("diffpector version %s\n", version)
		return
	}

	reportErr := agent.NotifyUserIfReportNotIgnored(".gitignore")
	if reportErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", reportErr)
		os.Exit(1)
	}

	configFile := flag.String("config", "config.json", "Path to configuration file")

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config from %s: %v\n", *configFile, err)
		os.Exit(1)
	}

	if !slices.Contains(llm.SupportedProviders, cfg.LLM.Provider) {
		fmt.Fprintf(os.Stderr, "Error: Unsupported LLM provider: %v\n", err)
		os.Exit(1)
	}

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

	providerConfig := llm.ProviderConfig{
		Type:    llm.ProviderType(cfg.LLM.Provider),
		Model:   cfg.LLM.Model,
		BaseURL: cfg.LLM.BaseURL,
	}

	llmProvider, err := llm.NewProvider(providerConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create LLM provider: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Using %s API with %s model\n\n", cfg.LLM.Provider, llmProvider.GetModel())

	parserRegistry := tools.NewParserRegistry()
	
	toolRegistry := tools.NewRegistry()

	toolsToRegister := map[tools.ToolName]tools.Tool{
		tools.ToolNameGitDiff:        &tools.GitDiffTool{},
		tools.ToolNameGitStagedFiles: &tools.GitStagedFilesTool{},
		tools.ToolNameGitGrep:        &tools.GitGrepTool{},
		tools.ToolNameWriteFile:      &tools.WriteFileTool{},
		tools.ToolNameReadFile:       &tools.ReadFileTool{},
		tools.ToolNameAppendFile:     &tools.AppendFileTool{},
		tools.ToolNameSymbolContext:  tools.NewSymbolContextTool(".", parserRegistry),
	}

	for name, tool := range toolsToRegister {
		toolRegistry.Register(name, tool)
	}

	codeReviewAgent := agent.NewCodeReviewAgent(llmProvider, toolRegistry, cfg, parserRegistry)

	if err := codeReviewAgent.ReviewStagedChanges(); err != nil {
		fmt.Fprintf(os.Stderr, "Code review failed: %v\n", err)
		os.Exit(1)
	}
}
