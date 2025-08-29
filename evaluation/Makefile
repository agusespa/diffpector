EVAL_GO_FILE := ./cmd/eval/main.go
EVAL_RESULTS_DIR := evaluation/results
EVAL_CONFIG_FILE := evaluation/model_configs.json
EVAL_SUITE_FILE := evaluation/test_suite.json

.PHONY: eval-help eval-compare-models eval-compare-prompts eval-list-prompts eval-prompts eval-models eval-models-sm eval-comprehensive eval-clean

eval-help:
	@echo "=============================="
	@echo " Diffpector Evaluation Tool"
	@echo "=============================="
	@echo ""
	@echo "🚀 Running Evaluations:"
	@echo "  make eval-prompts          - Compare prompt variants using default model"
	@echo "  make eval-models           - Compare model performance using default prompt"
	@echo "  make eval-comprehensive    - Run full evaluation matrix (all models × all prompts)"
	@echo ""
	@echo "📊 Analyzing Results:"
	@echo "  make eval-compare-prompts  - Analyze existing prompt comparison results"
	@echo "  make eval-compare-models   - Analyze existing model comparison results"
	@echo ""
	@echo "🔧 Utilities:"
	@echo "  make eval-list-prompts     - List available prompt variants"
	@echo "  make eval-clean            - Clean evaluation results"
	@echo "  make eval-help             - Show this help message"
	@echo ""
	@echo "⚙️  Configuration: Edit model_configs.json to customize models, prompts, and run counts"
	@echo ""

eval-list-prompts:
	@echo "Listing available prompt variants..."
	@go run $(EVAL_GO_FILE) --list-prompts

eval-compare-prompts:
	@echo "Comparing prompt variants..."
	@go run $(EVAL_GO_FILE) --compare-prompts --results $(EVAL_RESULTS_DIR)

eval-compare-models:
	@echo "Comparing existing evaluation results..."
	@go run $(EVAL_GO_FILE) --compare --results $(EVAL_RESULTS_DIR)

eval-prompts:
	@echo "Running prompt evaluation..."
	@go run $(EVAL_GO_FILE) --variant prompt-comparison --config $(EVAL_CONFIG_FILE) --suite $(EVAL_SUITE_FILE) --results $(EVAL_RESULTS_DIR)

eval-models:
	@echo "Running model evaluation..."
	@go run $(EVAL_GO_FILE) --variant model-comparison --config $(EVAL_CONFIG_FILE) --suite $(EVAL_SUITE_FILE) --results $(EVAL_RESULTS_DIR)

eval-models-sm:
	@echo "Running model evaluation..."
	@go run $(EVAL_GO_FILE) --variant small-model-comparison --config $(EVAL_CONFIG_FILE) --suite $(EVAL_SUITE_FILE) --results $(EVAL_RESULTS_DIR)

eval-comprehensive:
	@echo "Running comprehensive evaluation..."
	@go run $(EVAL_GO_FILE) --variant full-comprehensive --config $(EVAL_CONFIG_FILE) --suite $(EVAL_SUITE_FILE) --results $(EVAL_RESULTS_DIR)

eval-clean:
	@echo "Cleaning evaluation results..."
	@rm -rf $(EVAL_RESULTS_DIR)
	@echo "Evaluation cleanup complete."
