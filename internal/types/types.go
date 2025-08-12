package types

import "time"

// Issue represents a code review issue found by the agent
type Issue struct {
	Severity    string `json:"severity"`
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	Description string `json:"description"`
}

// ReviewContext contains all the context needed for code review
type ReviewContext struct {
	Diff           string
	ChangedFiles   []string
	FileContents   map[string]string
	SymbolAnalysis string
}

// CodeReviewer interface defines the contract for code review agents
type CodeReviewer interface {
	ReviewStagedChanges() error
	ReviewStagedChangesWithResults() ([]Issue, error)
}

// TestCase represents a single evaluation test case
type TestCase struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	DiffFile    string            `json:"diff_file,omitempty"`
	Expected    ExpectedResults   `json:"expected"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ExpectedResults defines what results are expected from a test case
type ExpectedResults struct {
	ShouldFindIssues bool     `json:"should_find_issues"`
	ExpectedSeverity []string `json:"expected_severity,omitempty"`
	ExpectedFiles    []string `json:"expected_files,omitempty"`
	MinIssues        int      `json:"min_issues,omitempty"`
	MaxIssues        int      `json:"max_issues,omitempty"`
}

// EvaluationSuite contains a collection of test cases
type EvaluationSuite struct {
	TestCases    []TestCase `json:"test_cases"`
	BaseDir      string     `json:"base_dir"`
	MockFilesDir string     `json:"mock_files_dir,omitempty"`
}

// EvaluationRun represents a single evaluation run
type EvaluationRun struct {
	Model         string                `json:"model"`
	Provider      string                `json:"provider"`
	PromptVariant string                `json:"prompt_variant"`
	StartTime     time.Time             `json:"start_time"`
	EndTime       time.Time             `json:"end_time"`
	TotalDuration time.Duration         `json:"total_duration"`
	Results       []TestCaseResult      `json:"results"`
	AverageScore  float64               `json:"average_score"`
	SuccessRate   float64               `json:"success_rate"`
	RunNumber     int                   `json:"run_number,omitempty"`
}

// EvaluationResult represents the complete result of an evaluation (single or multiple runs)
type EvaluationResult struct {
	Model           string                    `json:"model"`
	Provider        string                    `json:"provider"`
	PromptVariant   string                    `json:"prompt_variant"`
	TotalRuns       int                       `json:"total_runs"`
	StartTime       time.Time                 `json:"start_time"`
	EndTime         time.Time                 `json:"end_time"`
	TotalDuration   time.Duration             `json:"total_duration"`
	IndividualRuns  []EvaluationRun           `json:"individual_runs"`
	AggregatedStats EvaluationStats           `json:"aggregated_stats"`
	TestCaseStats   map[string]TestCaseStats  `json:"test_case_stats"`
}

// EvaluationStats contains statistical analysis across multiple runs
type EvaluationStats struct {
	AverageScore       float64 `json:"average_score"`
	ScoreStdDev        float64 `json:"score_std_dev"`
	MinScore           float64 `json:"min_score"`
	MaxScore           float64 `json:"max_score"`
	AverageSuccessRate float64 `json:"average_success_rate"`
	SuccessRateStdDev  float64 `json:"success_rate_std_dev"`
	AverageDuration    float64 `json:"average_duration_seconds"`
	DurationStdDev     float64 `json:"duration_std_dev_seconds"`
}

// TestCaseStats contains statistics for a specific test case across runs
type TestCaseStats struct {
	TestCaseName      string  `json:"test_case_name"`
	AverageScore      float64 `json:"average_score"`
	ScoreStdDev       float64 `json:"score_std_dev"`
	SuccessRate       float64 `json:"success_rate"`
	ConsistencyScore  float64 `json:"consistency_score"` // How consistent the results are
}

// TestCaseResult represents the result of a single test case evaluation
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

// EvaluationConfig represents configuration for an evaluation run
type EvaluationConfig struct {
	Key         string   `json:"key"`
	Description string   `json:"description,omitempty"`
	Provider    string   `json:"provider"`
	BaseURL     string   `json:"base_url"`
	Models      []string `json:"models"`
	Prompts     []string `json:"prompts"`
	Runs        int      `json:"runs,omitempty"`
}

// PromptVariant represents a prompt template variant
type PromptVariant struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Template    string `json:"template"`
}

// TestEnvironment represents a simulated test environment
type TestEnvironment struct {
	Files map[string]string // filename -> content
	Diff  string
}
