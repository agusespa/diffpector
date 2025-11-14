package types

import "time"

type EvaluationConfig struct {
	Key      string   `json:"key"`
	Provider string   `json:"provider"`
	BaseURL  string   `json:"base_url"`
	Models   []string `json:"models"`
	Prompts  []string `json:"prompts"`
	Runs     int      `json:"runs,omitempty"`
}

type EvaluationSuite struct {
	TestCases    []TestCase `json:"test_cases"`
	BaseDir      string     `json:"base_dir"`
	MockFilesDir string     `json:"mock_files_dir,omitempty"`
}

type TestCase struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	DiffFile    string          `json:"diff_file,omitempty"`
	Expected    ExpectedResults `json:"expected"`
}

type ExpectedResults struct {
	ShouldFindIssues bool     `json:"should_find_issues"`
	ExpectedSeverity []string `json:"expected_severity,omitempty"`
	ExpectedFiles    []string `json:"expected_files,omitempty"`
	MinIssues        int      `json:"min_issues,omitempty"`
	MaxIssues        int      `json:"max_issues,omitempty"`
}

type EvaluationRun struct {
	Model         string           `json:"model"`
	Provider      string           `json:"provider"`
	PromptVariant string           `json:"prompt_variant"`
	StartTime     time.Time        `json:"start_time"`
	EndTime       time.Time        `json:"end_time"`
	TotalDuration time.Duration    `json:"total_duration"`
	Results       []TestCaseResult `json:"results"`
	AverageScore  float64          `json:"average_score"`
	SuccessRate   float64          `json:"success_rate"`
	RunNumber     int              `json:"run_number,omitempty"`
}

type EvaluationResult struct {
	Model           string                   `json:"model"`
	Provider        string                   `json:"provider"`
	PromptVariant   string                   `json:"prompt_variant"`
	TotalRuns       int                      `json:"total_runs"`
	StartTime       time.Time                `json:"start_time"`
	EndTime         time.Time                `json:"end_time"`
	TotalDuration   time.Duration            `json:"total_duration"`
	IndividualRuns  []EvaluationRun          `json:"individual_runs"`
	AggregatedStats EvaluationStats          `json:"aggregated_stats"`
	TestCaseStats   map[string]TestCaseStats `json:"test_case_stats"`
}

type EvaluationStats struct {
	AverageScore       float64 `json:"average_score"`
	ScoreStdDev        float64 `json:"score_std_dev"`
	AverageSuccessRate float64 `json:"average_success_rate"`
	SuccessRateStdDev  float64 `json:"success_rate_std_dev"`
	AverageDuration    float64 `json:"average_duration_seconds"`
	DurationStdDev     float64 `json:"duration_std_dev_seconds"`
}

type TestCaseStats struct {
	TestCaseName string  `json:"test_case_name"`
	AverageScore float64 `json:"average_score"`
	ScoreStdDev  float64 `json:"score_std_dev"`
}

type TestCaseResult struct {
	TestCase      TestCase      `json:"test_case"`
	Model         string        `json:"model"`
	PromptHash    string        `json:"prompt_hash"`
	Issues        []Issue       `json:"issues"`
	ExecutionTime time.Duration `json:"execution_time"`
	Success       bool          `json:"success"`
	Score         float64       `json:"score"`
	Errors        []string      `json:"errors,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
}
