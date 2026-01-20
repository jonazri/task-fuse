// Package printer provides the Pretty_Printer component that serializes
// task objects back to valid tasks.md markdown format.
package printer

import (
	"regexp"
	"strings"

	"task-fuse/internal/store"
)

// Printer handles serialization of task structures back to markdown format.
// It preserves all non-task content and only modifies checkbox characters
// when task status changes.
type Printer struct{}

// NewPrinter creates a new Printer instance.
func NewPrinter() *Printer {
	return &Printer{}
}

// StatusToCheckbox converts a TaskStatus to its corresponding markdown checkbox syntax.
// Returns the full checkbox prefix including the dash and brackets.
//
// Mapping:
//   - StatusPending → "- [ ]"
//   - StatusQueued  → "- [~]"
//   - StatusDoing   → "- [-]"
//   - StatusDone    → "- [x]"
//   - StatusFailed  → "- [!]"
//
// For any unknown status, returns "- [ ]" (pending) as a safe default.
//
// Validates: Requirements 1.9
func (p *Printer) StatusToCheckbox(status store.TaskStatus) string {
	switch status {
	case store.StatusPending:
		return "- [ ]"
	case store.StatusQueued:
		return "- [~]"
	case store.StatusDoing:
		return "- [-]"
	case store.StatusDone:
		return "- [x]"
	case store.StatusFailed:
		return "- [!]"
	default:
		// Default to pending checkbox for unknown statuses
		return "- [ ]"
	}
}

// checkboxRegex matches the checkbox pattern in a line to find the checkbox portion.
// Pattern: - \[([ x\-~!])\]
// This is used to locate and replace the checkbox character when status changes.
var checkboxRegex = regexp.MustCompile(`- \[([ x\-~!])\]`)

// getCheckboxCharForStatus returns the single character used inside the checkbox brackets
// for a given status.
func (p *Printer) getCheckboxCharForStatus(status store.TaskStatus) string {
	switch status {
	case store.StatusPending:
		return " "
	case store.StatusQueued:
		return "~"
	case store.StatusDoing:
		return "-"
	case store.StatusDone:
		return "x"
	case store.StatusFailed:
		return "!"
	default:
		return " " // Default to pending
	}
}

// FormatTask serializes a task and its children back to markdown format.
// It preserves the original indentation and content, only modifying the checkbox
// character when the task's status has changed.
//
// Serialization Rules (from Design - Pretty_Printer):
//  1. Output task line with correct checkbox and indentation
//  2. Output description lines (unchanged)
//  3. Recursively output children
//
// Preservation Guarantees:
//   - Line order is preserved (tasks maintain original relative order)
//   - Non-task content is output exactly as parsed (rawContent)
//   - Only checkbox characters are modified when status changes
//   - Indentation is preserved from original
//
// The function uses the task's RawContent field which contains the original line.
// If the status has changed, it replaces only the checkbox character in the original
// line, preserving all other formatting including indentation.
//
// Validates: Requirements 1.9
func (p *Printer) FormatTask(task *store.Task) string {
	if task == nil {
		return ""
	}

	var result strings.Builder

	// Format the task line itself
	// We need to replace the checkbox character in the original RawContent
	// while preserving everything else (indentation, title, etc.)
	taskLine := p.formatTaskLine(task)
	result.WriteString(taskLine)
	result.WriteString("\n")

	// Output description lines (unchanged - use RawContent)
	for _, descLine := range task.Description {
		result.WriteString(descLine.RawContent)
		result.WriteString("\n")
	}

	// Recursively format children
	for _, child := range task.Children {
		result.WriteString(p.FormatTask(child))
	}

	return result.String()
}

// formatTaskLine formats a single task line, replacing the checkbox character
// if the status has changed while preserving all other content.
// If RawContent is empty (task created programmatically), generates a new line.
func (p *Printer) formatTaskLine(task *store.Task) string {
	rawContent := task.RawContent

	// If no raw content, generate a new task line
	if rawContent == "" {
		// Generate: "- [X] {ID} {Title}" with appropriate indentation
		indent := strings.Repeat("  ", task.IndentLevel)
		checkbox := p.StatusToCheckbox(task.Status)
		return indent + checkbox + " " + task.ID + " " + task.Title
	}

	// Find the checkbox in the raw content and replace the character inside brackets
	// The checkbox pattern is "- [X]" where X is the status character
	loc := checkboxRegex.FindStringIndex(rawContent)
	if loc == nil {
		// No checkbox found in raw content - this shouldn't happen for valid tasks
		// but return the raw content as-is for safety
		return rawContent
	}

	// The checkbox is at loc[0]:loc[1]
	// The character inside brackets is at loc[0]+3 (after "- [")
	// We need to replace just that character with the new status character
	newCheckboxChar := p.getCheckboxCharForStatus(task.Status)

	// Build the new line:
	// - Everything before the checkbox character (including "- [")
	// - The new checkbox character
	// - Everything after the checkbox character (including "]" and the rest)
	charPos := loc[0] + 3 // Position of the character inside brackets
	newLine := rawContent[:charPos] + newCheckboxChar + rawContent[charPos+1:]

	return newLine
}

// Serialize serializes a ParsedDocument back to markdown format.
// It outputs preamble, root tasks with children, and epilogue in original order,
// preserving all non-task content exactly as parsed.
//
// Serialization Rules (from Design - Pretty_Printer):
//  1. Output preamble lines first (unchanged)
//  2. For each root task (in original line order):
//     - Output task line with correct checkbox and indentation
//     - Output description lines (unchanged)
//     - Recursively output children
//  3. Output epilogue lines last (unchanged)
//
// Preservation Guarantees:
//   - Line order is preserved (tasks maintain original relative order)
//   - Non-task content is output exactly as parsed (rawContent)
//   - Only checkbox characters are modified when status changes
//   - Indentation is preserved from original
//
// The final output does NOT have a trailing newline after the last line
// (to match original file format).
//
// Validates: Requirements 1.9, 1.10
func (p *Printer) Serialize(doc *store.ParsedDocument) string {
	// Handle nil document gracefully
	if doc == nil {
		return ""
	}

	var result strings.Builder

	// Track if we've written any content (to handle trailing newline correctly)
	hasContent := false

	// 1. Output preamble lines first (unchanged, using RawContent)
	for _, line := range doc.Preamble {
		if hasContent {
			result.WriteString("\n")
		}
		result.WriteString(line.RawContent)
		hasContent = true
	}

	// 2. For each root task, serialize it and its children
	for _, task := range doc.RootTasks {
		taskOutput := p.FormatTask(task)
		if taskOutput != "" {
			if hasContent {
				result.WriteString("\n")
			}
			// FormatTask returns content with trailing newlines for each line
			// We need to remove the final trailing newline and add newlines between lines
			taskOutput = strings.TrimSuffix(taskOutput, "\n")
			result.WriteString(taskOutput)
			hasContent = true
		}
	}

	// 3. Output epilogue lines last (unchanged, using RawContent)
	for _, line := range doc.Epilogue {
		if hasContent {
			result.WriteString("\n")
		}
		result.WriteString(line.RawContent)
		hasContent = true
	}

	return result.String()
}
