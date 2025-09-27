package types

type DiffData struct {
	AbsolutePath    string
	Diff            string
	DiffContext     string
	AffectedSymbols []SymbolUsage
}

type SymbolUsage struct {
	Symbol   Symbol
	Snippets []string
}

type Symbol struct {
	Name      string
	Type      string
	Package   string
	FilePath  string
	StartLine int
	EndLine   int
}

type ContextResult struct {
	Context         string
	AffectedSymbols []SymbolUsage
}
