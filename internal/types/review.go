package types

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
