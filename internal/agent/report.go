package agent

import (
	"fmt"
	"strings"

	"github.com/agusespa/diffpector/internal/tools"
	"github.com/agusespa/diffpector/internal/types"
	"github.com/agusespa/diffpector/internal/utils"
)

type ReportGenerator struct {
	readTool  tools.Tool
	writeTool tools.Tool
}

func NewReportGenerator(readTool, writeTool tools.Tool) *ReportGenerator {
	return &ReportGenerator{
		readTool:  readTool,
		writeTool: writeTool,
	}
}

func (r *ReportGenerator) GenerateMarkdownReport(issues []types.Issue) (criticalCount, warningCount, minorCount int, err error) {
	var reportBuilder strings.Builder
	reportBuilder.WriteString("# Code Review Report\n\n")

	criticalCount = 0
	warningCount = 0
	minorCount = 0

	for _, issue := range issues {
		content, err := r.readTool.Execute(map[string]any{"filename": issue.FilePath})
		if err != nil {
			reportBuilder.WriteString(fmt.Sprintf("## ⚪️ Could not retrieve code for issue: %s\n", issue.Description))
			reportBuilder.WriteString(fmt.Sprintf("**File:** `%s`\n", issue.FilePath))
			reportBuilder.WriteString(fmt.Sprintf("**Error:** %v\n\n---\n\n", err))
			continue
		}

		lines := strings.Split(content, "\n")
		if issue.StartLine > len(lines) || issue.EndLine > len(lines) || issue.StartLine > issue.EndLine || issue.StartLine <= 0 {
			reportBuilder.WriteString(fmt.Sprintf("## ⚪️ Invalid line numbers for issue: %s\n", issue.Description))
			reportBuilder.WriteString(fmt.Sprintf("**File:** `%s`\n", issue.FilePath))
			reportBuilder.WriteString(fmt.Sprintf("**Line Range:** %d-%d\n\n---\n\n", issue.StartLine, issue.EndLine))
			continue
		}

		severityIcon := r.getSeverityIcon(issue.Severity)
		switch issue.Severity {
		case "CRITICAL":
			criticalCount++
		case "WARNING":
			warningCount++
		case "MINOR":
			minorCount++
		}

		reportBuilder.WriteString(fmt.Sprintf("## %s %s: %s\n", severityIcon, issue.Severity, issue.Description))
		reportBuilder.WriteString(fmt.Sprintf("**File:** `%s`\n", issue.FilePath))
		reportBuilder.WriteString(fmt.Sprintf("**Location:** Lines %d-%d\n", issue.StartLine, issue.EndLine))
		reportBuilder.WriteString("**Code:**\n")
		language := utils.DetectLanguageFromFilePath(issue.FilePath)
		reportBuilder.WriteString(fmt.Sprintf("```%s\n", language))
		for i := issue.StartLine - 1; i < issue.EndLine; i++ {
			reportBuilder.WriteString(lines[i] + "\n")
		}
		reportBuilder.WriteString("```\n\n---\n\n")
	}

	var summary = fmt.Sprintf("\n\n**Summary:** %d critical, %d warnings, %d minor issues\n", criticalCount, warningCount, minorCount)
	reportBuilder.WriteString(summary)

	writeArgs := map[string]any{
		"filename": "diffpector_report.md",
		"content":  reportBuilder.String(),
	}

	_, err = r.writeTool.Execute(writeArgs)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to write markdown code review: %w", err)
	}

	return criticalCount, warningCount, minorCount, nil
}

func (r *ReportGenerator) getSeverityIcon(severity string) string {
	switch severity {
	case "CRITICAL":
		return "🔴"
	case "WARNING":
		return "🟡"
	case "MINOR":
		return "🔵"
	default:
		return "⚪️"
	}
}

func PrintReviewSummary(criticalCount, warningCount, minorCount int) {
	fmt.Println("---")
	if criticalCount+warningCount+minorCount > 0 {
		fmt.Printf("⚠️ Code review didn't pass - %d critical, %d warnings and %d minor issues were found\n",
			criticalCount, warningCount, minorCount)
	}
	fmt.Println("💾 Detailed report saved to diffpector_report.md")
}
