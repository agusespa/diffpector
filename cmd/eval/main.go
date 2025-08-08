package main

import (
	"github.com/agusespa/diffpector/internal/evaluation"
	"github.com/agusespa/diffpector/internal/llm"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	var (
		suiteFile   = flag.String("suite", "evaluation/test_suite.json", "Path to evaluation test suite")
		resultsDir  = flag.String("results", "evaluation/results", "Directory to store results")
		configFile  = flag.String("config", "evaluation/model_configs.json", "Path to evaluation config file")
		variant     = flag.String("variant", "", "Variant Key of the specific configuration to run from the config file")
		compare     = flag.Bool("compare", false, "Compare existing results instead of running new evaluation")
		listPrompts = flag.Bool("list-prompts", false, "List available prompt variants")
		showHelp    = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	fmt.Println("")
	fmt.Println("============================")
	fmt.Println(" Diffpector Evaluation Tool ")
	fmt.Println("============================")
	fmt.Println("")

	if *showHelp {
		printHelp()
		return
	}

	if *listPrompts {
		printPromptVariants()
		return
	}

	if *compare {
		if err := evaluation.CompareResults(*resultsDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error comparing results: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := runEvaluation(*suiteFile, *resultsDir, *configFile, *variant); err != nil {
		fmt.Fprintf(os.Stderr, "Error running evaluation: %v\n", err)
		os.Exit(1)
	}
}

func runEvaluation(suiteFile, resultsDir, configFile, variantKey string) error {
	configs, err := evaluation.LoadEvaluationConfigs(configFile)
	if err != nil {
		return fmt.Errorf("failed to load evaluation configs: %w", err)
	}

	evaluator, err := evaluation.NewEvaluator(suiteFile, resultsDir)
	if err != nil {
		return fmt.Errorf("failed to create evaluator: %w", err)
	}

	for _, config := range configs {
		if variantKey != "" && config.Variant != variantKey {
			continue
		}
		fmt.Printf("---"+" Running Configuration: %s ---", config.Variant)
		for _, model := range config.Models {
			for _, prompt := range config.Prompts {
				runSingleEvaluation(evaluator, config.Provider, model, prompt, config.BaseURL)
			}
		}
	}

	fmt.Println("All evaluations complete.")
	fmt.Printf("To compare results, run: %s --compare --results %s\n", os.Args[0], resultsDir)

	return nil
}

func runSingleEvaluation(evaluator *evaluation.Evaluator, provider, model, prompt, baseURL string) {
	model = strings.TrimSpace(model)
	prompt = strings.TrimSpace(prompt)

	if _, err := evaluation.GetPromptVariant(prompt); err != nil {
		fmt.Printf("Warning: skipping unknown prompt variant '%s'\n", prompt)
		return
	}

	fmt.Printf("=== Running evaluation: %s with %s prompt ===\n", model, prompt)

	llmConfig := llm.ProviderConfig{
		Type:    llm.ProviderType(provider),
		Model:   model,
		BaseURL: baseURL,
	}

	run, err := evaluator.RunEvaluation(llmConfig, prompt)
	if err != nil {
		fmt.Printf("Error running evaluation for %s/%s: %v\n", model, prompt, err)
		return
	}

	evaluation.PrintSummary(run)

	if err := evaluator.SaveResults(run); err != nil {
		fmt.Printf("Warning: failed to save results: %v\n", err)
	}
	fmt.Println()
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Printf("  %s [options]\n", os.Args[0])
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s --config evaluation/model_configs.json --variant quick-sanity-check\n", os.Args[0])
	fmt.Printf("  %s --compare\n", os.Args[0])
	fmt.Printf("  %s --list-prompts\n", os.Args[0])
}

func printPromptVariants() {
	fmt.Println("Available prompt variants:")
	for _, name := range evaluation.ListPromptVariants() {
		variant, _ := evaluation.GetPromptVariant(name)
		fmt.Printf("  %s: %s\n", name, variant.Description)
	}
}
