package types

type Issue struct {
	Severity    string `json:"severity"`
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	Description string `json:"description"`
}

type PromptVariant struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Template    string `json:"template"`
}
