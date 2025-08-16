package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/agusespa/diffpector/internal/evaluation"
	"github.com/agusespa/diffpector/internal/llm"
	"github.com/agusespa/diffpector/internal/prompts"
)

func main() {
	var (
		suiteFile      = flag.String("suite", "evaluation/test_suite.json", "Path to evaluation test suite")
		resultsDir     = flag.String("results", "evaluation/results", "Directory to store results")
		configFile     = flag.String("config", "evaluation/model_configs.json", "Path to evaluation config file")
		variant        = flag.String("variant", "", "Variant Key of the specific configuration to run from the config file")
		compare        = flag.Bool("compare", false, "Compare existing results instead of running new evaluation")
		comparePrompts = flag.Bool("compare-prompts", false, "Compare prompt variants")
		listPrompts    = flag.Bool("list-prompts", false, "List available prompt variants")
	)
	flag.Parse()

	fmt.Println("")
	fmt.Println("============================")
	fmt.Println(" Diffpector Evaluation Tool ")
	fmt.Println("============================")
	fmt.Println("")

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

	if *comparePrompts {
		if err := evaluation.ComparePrompts(*resultsDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error comparing prompts: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *variant == "" {
		printHelp()
		return
	}

	if err := runEvaluation(*suiteFile, *resultsDir, *configFile, *variant); err != nil {
		fmt.Fprintf(os.Stderr, "Error running evaluation: %v\n", err)
		os.Exit(1)
	}
}

func runEvaluation(suiteFile, resultsDir, configFile, variantKey string) error {
	configs, err := evaluation.LoadConfigs(configFile)
	if err != nil {
		return fmt.Errorf("failed to load evaluation configs: %w", err)
	}

	evaluator, err := evaluation.NewEvaluator(suiteFile, resultsDir)
	if err != nil {
		return fmt.Errorf("failed to create evaluator: %w", err)
	}

	for _, config := range configs {
		if variantKey != "" && config.Key != variantKey {
			continue
		}
		fmt.Printf("--- Running Configuration: %s ---\n", config.Key)
		if config.Description != "" {
			fmt.Printf("    %s\n", config.Description)
		}

		// Use config runs, default to 1 if not specified
		configRuns := config.Runs
		if configRuns == 0 {
			configRuns = 1
		}

		for _, model := range config.Models {
			for _, prompt := range config.Prompts {
				runSingleEvaluation(evaluator, config.Provider, model, prompt, config.BaseURL, configRuns)
			}
		}
	}

	fmt.Println("All evaluations complete.")
	fmt.Println("To compare results, run: make eval-compare-{prompts/models}")

	return nil
}

func runSingleEvaluation(evaluator *evaluation.Evaluator, provider, model, prompt, baseURL string, runs int) {
	model = strings.TrimSpace(model)
	prompt = strings.TrimSpace(prompt)

	if _, err := prompts.GetPromptVariant(prompt); err != nil {
		fmt.Printf("Warning: skipping unknown prompt variant '%s'\n", prompt)
		return
	}

	fmt.Printf("=== Running evaluation: %s with %s prompt ===\n", model, prompt)

	llmConfig := llm.ProviderConfig{
		Type:    llm.ProviderType(provider),
		Model:   model,
		BaseURL: baseURL,
	}

	// Run evaluation
	result, err := evaluator.RunEvaluation(llmConfig, prompt, runs)
	if err != nil {
		fmt.Printf("Error running evaluation for %s/%s: %v\n", model, prompt, err)
		return
	}

	if runs == 1 {
		// For single runs, print the individual run summary
		if len(result.IndividualRuns) > 0 {
			evaluation.PrintSummary(&result.IndividualRuns[0])
		}
	} else {
		// For multiple runs, print evaluation summary
		evaluation.PrintEvaluationSummary(result)
	}

	// Save results
	if err := evaluator.SaveEvaluationResults(result); err != nil {
		fmt.Printf("Warning: failed to save results: %v\n", err)
	}
	fmt.Println()
}

func printHelp() {
	fmt.Println("Use 'make eval-help' to see available evaluation commands")
}

func printPromptVariants() {
	fmt.Println("Available prompt variants:")
	for _, name := range prompts.ListPromptVariants() {
		variant, _ := prompts.GetPromptVariant(name)
		fmt.Printf("  %s: %s\n", name, variant.Description)
	}
}
