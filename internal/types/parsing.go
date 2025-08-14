package types

// Symbol represents a code symbol (function, type, variable, etc.) found during parsing.
type Symbol struct {
	Name      string // The name of the symbol
	Package   string // The package/namespace the symbol belongs to
	FilePath  string // The file path where the symbol is defined
	StartLine int    // The starting line number of the symbol definition
	EndLine   int    // The ending line number of the symbol definition
}

// SymbolUsage represents a usage/reference of a symbol in code.
type SymbolUsage struct {
	SymbolName string // The name of the symbol being used
	FilePath   string // The file path where the usage occurs
	LineNumber int    // The line number where the usage occurs
	Context    string // Contextual information around the usage
}
