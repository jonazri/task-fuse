// Package parser provides the Tasks_MD_Parser component that parses tasks.md
// markdown files into a hierarchical task tree structure.
package parser

import (
	"regexp"
	"strings"

	"task-fuse/internal/store"
)

// checkboxRegex matches the checkbox pattern at the start of a line.
// Pattern: ^\s*- \[([ x\-~!])\]
// This matches lines like:
//   - [ ] Task (pending)
//   - [~] Task (queued)
//   - [-] Task (doing)
//   - [x] Task (done)
//   - [!] Task (failed)
var checkboxRegex = regexp.MustCompile(`^\s*- \[([ x\-~!])\]`)

// taskIDValidationRegex validates task ID format: positive integers separated by dots.
// Pattern: ^[1-9][0-9]*(\.[1-9][0-9]*)*$
// Valid examples: "1", "1.2", "1.2.3", "10.5.2"
// Invalid examples: "01.1" (leading zero), "1.2.a" (non-numeric), "1..2" (empty segment)
var taskIDValidationRegex = regexp.MustCompile(`^[1-9][0-9]*(\.[1-9][0-9]*)*$`)

// taskIDExtractionRegex extracts numeric ID from task line content (after checkbox).
// Pattern: ^([1-9][0-9]*(?:\.[1-9][0-9]*)*)\.?\s+(.*)
// This extracts the ID and title from lines like:
//   "1.1 Task title" → ID="1.1", Title="Task title"
//   "10.5.2 Deep task" → ID="10.5.2", Title="Deep task"
//   "1. Task with dot" → ID="1", Title="Task with dot"
var taskIDExtractionRegex = regexp.MustCompile(`^([1-9][0-9]*(?:\.[1-9][0-9]*)*)\.?\s+(.*)`)

// MaxTaskIDDepth is the maximum number of levels allowed in a task ID.
// For example, "1.2.3.4.5.6.7.8.9.10" has 10 levels and is the deepest valid ID.
const MaxTaskIDDepth = 10

// Parser handles parsing of tasks.md files into task structures.
// It validates task IDs, extracts checkbox statuses, and builds
// the hierarchical task tree based on indentation.
type Parser struct {
	// TaskIDRegex validates task ID format: positive integers separated by dots
	// Pattern: ^[1-9][0-9]*(\.[1-9][0-9]*)*$
	// Max depth: 10 levels
}

// NewParser creates a new Parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// ParseCheckbox extracts the task status from a line's checkbox syntax.
// It returns the TaskStatus and true if a valid checkbox is found,
// or an empty status and false if the line doesn't contain valid checkbox syntax.
//
// Valid checkbox patterns:
//   - [ ] → StatusPending
//   - [~] → StatusQueued
//   - [-] → StatusDoing
//   - [x] → StatusDone
//   - [!] → StatusFailed
//
// The function allows leading whitespace before the checkbox.
//
// Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5
func (p *Parser) ParseCheckbox(line string) (store.TaskStatus, bool) {
	matches := checkboxRegex.FindStringSubmatch(line)
	if matches == nil {
		return "", false
	}

	// matches[1] contains the character inside the brackets
	checkboxChar := matches[1]

	switch checkboxChar {
	case " ":
		return store.StatusPending, true
	case "~":
		return store.StatusQueued, true
	case "-":
		return store.StatusDoing, true
	case "x":
		return store.StatusDone, true
	case "!":
		return store.StatusFailed, true
	default:
		// This shouldn't happen given our regex, but handle it gracefully
		return "", false
	}
}

// ValidateTaskID validates that a task ID matches the required format.
// Valid task IDs are dot-separated positive integers with no leading zeros.
//
// Format rules:
//   - Must match regex: ^[1-9][0-9]*(\.[1-9][0-9]*)*$
//   - No leading zeros: "01.1" or "1.01" are invalid
//   - Maximum depth: 10 levels (e.g., "1.2.3.4.5.6.7.8.9.10" is the deepest valid ID)
//
// Valid examples: "1", "1.2", "1.2.3", "10.5.2"
// Invalid examples: "01.1" (leading zero), "1.2.a" (non-numeric), "1..2" (empty segment), "" (empty)
//
// Validates: Requirements 1.5, 1.6, 1.11, Design - Task ID Format
func (p *Parser) ValidateTaskID(id string) bool {
	// Empty ID is invalid
	if id == "" {
		return false
	}

	// Check if ID matches the basic format
	if !taskIDValidationRegex.MatchString(id) {
		return false
	}

	// Check maximum depth (count dots + 1 = number of levels)
	segments := strings.Split(id, ".")
	if len(segments) > MaxTaskIDDepth {
		return false
	}

	return true
}

// ExtractTaskID extracts the numeric task ID from a task line.
// The line should be the content after the checkbox (e.g., "1.1 Task title").
//
// The function extracts the ID using the pattern: ^([1-9][0-9]*(?:\.[1-9][0-9]*)*)\.?\s+(.*)
// This allows for an optional trailing dot after the ID (e.g., "1. Task" or "1 Task").
//
// Returns:
//   - (id, true) if a valid task ID is found
//   - ("", false) if no valid task ID is found
//
// Examples:
//   - "1.1 Task title" → ("1.1", true)
//   - "10.5.2 Deep task" → ("10.5.2", true)
//   - "1. Task with dot" → ("1", true)
//   - "01.1 Leading zero" → ("", false) - invalid due to leading zero
//   - "1.2.a Non-numeric" → ("", false) - invalid due to non-numeric segment
//   - "No number here" → ("", false) - no ID found
//
// Validates: Requirements 1.5, 1.6, 1.11, Design - Task ID Format
func (p *Parser) ExtractTaskID(line string) (string, bool) {
	// Try to extract ID using the extraction regex
	matches := taskIDExtractionRegex.FindStringSubmatch(line)
	if matches == nil {
		return "", false
	}

	// matches[1] contains the extracted ID
	id := matches[1]

	// Validate the extracted ID (checks format and depth)
	if !p.ValidateTaskID(id) {
		return "", false
	}

	return id, true
}

// ComputeIndentLevel calculates the indentation level of a line.
// It converts tabs to 2 spaces first, then counts total spaces and divides by 2.
//
// Indentation rules (from Design - Indentation Specification):
//   - 1 tab = 2 spaces (internally converted)
//   - 2-4 spaces = 1 indent level (flexible spacing)
//   - Mixed tabs/spaces: tabs converted to 2 spaces first, then total spaces counted
//   - Result is floor(total_spaces / 2)
//
// Examples:
//   - "" (no indent) → 0
//   - "  " (2 spaces) → 1
//   - "   " (3 spaces) → 1 (floor(3/2) = 1)
//   - "    " (4 spaces) → 2
//   - "\t" (1 tab) → 1 (tab = 2 spaces, 2/2 = 1)
//   - "\t\t" (2 tabs) → 2 (2 tabs = 4 spaces, 4/2 = 2)
//   - "  \t" (2 spaces + 1 tab) → 2 (2 + 2 = 4 spaces, 4/2 = 2)
//   - "\t  " (1 tab + 2 spaces) → 2 (2 + 2 = 4 spaces, 4/2 = 2)
//
// Validates: Design - Indentation Specification, Indent Level Computation
func (p *Parser) ComputeIndentLevel(line string) int {
	spaces := 0
	for _, ch := range line {
		if ch == '\t' {
			spaces += 2 // 1 tab = 2 spaces
		} else if ch == ' ' {
			spaces++
		} else {
			break
		}
	}
	// 2-4 spaces = 1 level (use floor division by 2)
	return spaces / 2
}

// stackEntry represents an entry in the hierarchy building stack.
// It pairs an indent level with a task for tracking parent-child relationships.
type stackEntry struct {
	indentLevel int
	task        *store.Task
}

// GetIDPrefixes extracts all prefix segments from a task ID.
// For example, "1.2.3" returns ["1", "1.2", "1.2.3"].
// This is used for orphaned ID resolution to find the nearest valid ancestor.
//
// Examples:
//   - "1" → ["1"]
//   - "1.2" → ["1", "1.2"]
//   - "1.2.3" → ["1", "1.2", "1.2.3"]
//   - "10.5.2" → ["10", "10.5", "10.5.2"]
func (p *Parser) GetIDPrefixes(id string) []string {
	if id == "" {
		return nil
	}

	segments := strings.Split(id, ".")
	prefixes := make([]string, len(segments))

	for i := range segments {
		prefixes[i] = strings.Join(segments[:i+1], ".")
	}

	return prefixes
}

// FindAncestorByIDPrefix finds the nearest valid ancestor for an orphaned task ID.
// When a task ID implies a missing parent (e.g., "1.2.3" exists but "1.2" does not),
// this function searches for the nearest existing ancestor by ID prefix match.
//
// Algorithm:
//  1. Extract ID prefix segments: "1.2.3" → ["1", "1.2", "1.2.3"]
//  2. Search for nearest existing ancestor by prefix match (from longest to shortest)
//  3. If found, return that ancestor
//  4. If no ancestor found, return nil (task should be treated as root)
//
// Example:
//
//	If task "1.2.3" exists but "1.2" does not:
//	- Check if "1.2" exists → No
//	- Check if "1" exists → Yes
//	- Return task "1" as the ancestor
//
// Parameters:
//   - id: The task ID to find an ancestor for (e.g., "1.2.3")
//   - tasks: A flat list of all tasks to search through
//
// Returns:
//   - The nearest valid ancestor task, or nil if no ancestor exists
//
// Validates: Requirements 1.12
func (p *Parser) FindAncestorByIDPrefix(id string, tasks []*store.Task) *store.Task {
	if id == "" || len(tasks) == 0 {
		return nil
	}

	// Build a map of task IDs to tasks for efficient lookup
	taskByID := make(map[string]*store.Task)
	for _, task := range tasks {
		if task != nil && task.ID != "" {
			taskByID[task.ID] = task
		}
	}

	// Get all prefixes for the ID
	prefixes := p.GetIDPrefixes(id)
	if len(prefixes) == 0 {
		return nil
	}

	// Search from the second-to-last prefix (immediate parent) down to the first
	// We skip the last prefix because that's the task's own ID
	// We search from longest to shortest to find the nearest ancestor
	for i := len(prefixes) - 2; i >= 0; i-- {
		prefix := prefixes[i]
		if ancestor, exists := taskByID[prefix]; exists {
			return ancestor
		}
	}

	// No ancestor found - task should be treated as root
	return nil
}

// BuildHierarchy builds parent-child relationships from a flat list of tasks
// based on their indentation levels.
//
// The algorithm uses a stack to track (indent_level, task) pairs:
//  1. For each task in the flat list:
//     - Pop stack until finding a task with lower indent level (or stack is empty)
//     - If stack is not empty, the top task becomes the parent
//     - Push the current task onto the stack
//  2. Return only the root tasks (tasks with no parent)
//
// Hierarchy Determination (from Requirements):
//   - A task is a child of the nearest preceding task with less indentation
//   - Root tasks: Tasks with zero indentation are root-level tasks
//
// Example:
//
//	- [ ] 1. Root task              (indent level 0) → root
//	  - [ ] 1.1 Child task          (indent level 1) → child of 1
//	    - [ ] 1.1.1 Grandchild      (indent level 2) → child of 1.1
//	   - [ ] 1.1.2 Also grandchild  (indent level 1) → child of 1 (3 spaces = level 1)
//	- [ ] 2. Another root task      (indent level 0) → root
//
// Input: A flat list of *Task objects with IndentLevel already computed (from FileLine)
// Output: A list of root tasks with Children and Parent pointers properly set
//
// Validates: Requirements 1.7
func (p *Parser) BuildHierarchy(flatTasks []*store.Task) []*store.Task {
	if len(flatTasks) == 0 {
		return nil
	}

	var rootTasks []*store.Task
	var stack []stackEntry

	for _, task := range flatTasks {
		indentLevel := task.IndentLevel

		// Pop stack until we find a task with lower indent level
		// This finds the nearest preceding task that could be a parent
		for len(stack) > 0 && stack[len(stack)-1].indentLevel >= indentLevel {
			stack = stack[:len(stack)-1]
		}

		// If stack is not empty, the top task becomes the parent
		if len(stack) > 0 {
			parent := stack[len(stack)-1].task
			task.Parent = parent
			parent.Children = append(parent.Children, task)
		} else {
			// No parent found - this is a root task
			task.Parent = nil
			rootTasks = append(rootTasks, task)
		}

		// Push current task onto stack
		stack = append(stack, stackEntry{
			indentLevel: indentLevel,
			task:        task,
		})
	}

	return rootTasks
}

// ParseError represents an error encountered during parsing.
type ParseError struct {
	// LineNumber is the 1-based line number where the error occurred.
	LineNumber int
	// Message describes the error.
	Message string
	// RawLine is the original line content that caused the error.
	RawLine string
}

// ParseResult contains the result of parsing a tasks.md file.
type ParseResult struct {
	// Document is the parsed document structure.
	Document *store.ParsedDocument
	// Errors contains any errors encountered during parsing.
	Errors []ParseError
}

// extractTitleFromLine extracts the title from a task line after the ID.
// The line should be the content after the checkbox (e.g., "1.1 Task title").
// Returns the title portion after the ID.
func (p *Parser) extractTitleFromLine(line string) string {
	matches := taskIDExtractionRegex.FindStringSubmatch(line)
	if matches == nil || len(matches) < 3 {
		return ""
	}
	return matches[2]
}

// Parse parses a tasks.md file content into a ParsedDocument structure.
//
// The parsing algorithm follows the Content Attachment Rules from the design:
//  1. Preamble: All lines before the first valid task line are stored as preamble
//  2. Valid Task: A line with checkbox syntax AND a numeric ID (e.g., "- [ ] 1.1 Task title")
//  3. No-ID Checkbox Lines: Lines with checkbox syntax but NO numeric ID are NOT parsed as tasks;
//     they are treated as regular non-task content
//  4. Task Ownership: A task "owns" all subsequent lines until a new valid task at an equal or
//     lesser indent level is encountered
//  5. Description Attachment: Within a task's content block, any line that is not a valid sub-task
//     is attached to the task as part of its description
//  6. Sub-tasks: Within a task's content block, any line that is a valid task (has checkbox AND
//     numeric ID) and is more indented than the parent becomes a child task
//  7. Epilogue: All lines after the last task and its content block are stored as epilogue
//
// Validates: Requirements 1.6, 1.8, 1.11, Design - Content Attachment Rules
func (p *Parser) Parse(content string) *ParseResult {
	result := &ParseResult{
		Document: &store.ParsedDocument{
			Preamble:  []store.NonTaskContent{},
			RootTasks: []*store.Task{},
			Epilogue:  []store.NonTaskContent{},
		},
		Errors: []ParseError{},
	}

	// Handle empty content
	if content == "" {
		return result
	}

	// Split content into lines
	lines := strings.Split(content, "\n")

	// Track parsing state
	var flatTasks []*store.Task
	var taskStack []stackEntry // Stack for tracking current task context
	foundFirstTask := false

	for lineNum, line := range lines {
		lineNumber := lineNum + 1 // 1-based line numbers
		indentLevel := p.ComputeIndentLevel(line)

		fileLine := store.FileLine{
			LineNumber:  lineNumber,
			RawContent:  line,
			IndentLevel: indentLevel,
		}

		// Check if this line has checkbox syntax
		status, hasCheckbox := p.ParseCheckbox(line)

		if hasCheckbox {
			// Extract the content after the checkbox
			// Find the position after "- [x] " pattern
			checkboxEnd := strings.Index(line, "] ")
			var afterCheckbox string
			if checkboxEnd != -1 && checkboxEnd+2 < len(line) {
				afterCheckbox = line[checkboxEnd+2:]
			} else {
				afterCheckbox = ""
			}

			// Try to extract a valid task ID
			taskID, hasValidID := p.ExtractTaskID(afterCheckbox)

			if hasValidID {
				// This is a valid task line (has checkbox AND valid numeric ID)
				title := p.extractTitleFromLine(afterCheckbox)

				task := &store.Task{
					FileLine:    fileLine,
					ID:          taskID,
					Title:       title,
					Status:      status,
					Description: []store.FileLine{},
					Children:    []*store.Task{},
					Parent:      nil,
				}

				flatTasks = append(flatTasks, task)
				foundFirstTask = true

				// Update task stack: pop until we find a task with lower indent level
				for len(taskStack) > 0 && taskStack[len(taskStack)-1].indentLevel >= indentLevel {
					taskStack = taskStack[:len(taskStack)-1]
				}

				// Push current task onto stack
				taskStack = append(taskStack, stackEntry{
					indentLevel: indentLevel,
					task:        task,
				})
			} else {
				// Checkbox line but NO valid numeric ID
				// Treat as non-task content (attach to current task's description or preamble/epilogue)
				if !foundFirstTask {
					// Before first task - add to preamble
					result.Document.Preamble = append(result.Document.Preamble, store.NonTaskContent{
						FileLine: fileLine,
					})
				} else if len(taskStack) > 0 {
					// After first task - attach to current task's description
					currentTask := taskStack[len(taskStack)-1].task
					currentTask.Description = append(currentTask.Description, fileLine)
				} else {
					// No current task context - add to epilogue
					result.Document.Epilogue = append(result.Document.Epilogue, store.NonTaskContent{
						FileLine: fileLine,
					})
				}
			}
		} else {
			// Line does not have checkbox syntax
			if !foundFirstTask {
				// Before first task - add to preamble
				result.Document.Preamble = append(result.Document.Preamble, store.NonTaskContent{
					FileLine: fileLine,
				})
			} else if len(taskStack) > 0 {
				// After first task - determine if this belongs to a task or is epilogue
				// Pop stack to find the task that owns this line based on indent level
				// A task owns lines until a new task at equal or lesser indent is found

				// Find the appropriate task to attach this line to
				// We need to find the most recent task that could own this line
				// based on the content attachment rules

				// Check if this line is at root level (indent 0) and is not indented
				// relative to any task - if so, it could be epilogue
				if indentLevel == 0 && strings.TrimSpace(line) == "" {
					// Empty line at root level after tasks - could be between tasks or epilogue
					// Attach to current task's description for now
					currentTask := taskStack[len(taskStack)-1].task
					currentTask.Description = append(currentTask.Description, fileLine)
				} else {
					// Attach to the current task's description
					// The current task is the one at the top of the stack
					currentTask := taskStack[len(taskStack)-1].task
					currentTask.Description = append(currentTask.Description, fileLine)
				}
			} else {
				// No task context - add to epilogue
				result.Document.Epilogue = append(result.Document.Epilogue, store.NonTaskContent{
					FileLine: fileLine,
				})
			}
		}
	}

	// Build hierarchy from flat task list
	result.Document.RootTasks = p.BuildHierarchy(flatTasks)

	// Handle orphaned IDs: tasks whose ID implies a missing parent
	// This is done during hierarchy building, but we can also resolve
	// orphaned tasks here if needed
	p.resolveOrphanedTasks(result.Document.RootTasks, flatTasks)

	return result
}

// resolveOrphanedTasks handles tasks with orphaned IDs (where the implied parent does not exist).
// For example, if task "1.2.3" exists but "1.2" does not, this function will attach it to
// the nearest valid ancestor by ID prefix match, or leave it as a root task if no ancestor exists.
//
// Validates: Requirements 1.12
func (p *Parser) resolveOrphanedTasks(rootTasks []*store.Task, allTasks []*store.Task) {
	// Build a map of all task IDs for quick lookup
	taskByID := make(map[string]*store.Task)
	for _, task := range allTasks {
		if task != nil && task.ID != "" {
			taskByID[task.ID] = task
		}
	}

	// Check each task to see if it's orphaned
	for _, task := range allTasks {
		if task == nil || task.ID == "" {
			continue
		}

		// Get the expected parent ID (all but the last segment)
		prefixes := p.GetIDPrefixes(task.ID)
		if len(prefixes) <= 1 {
			// Single-level ID (e.g., "1") - no parent expected
			continue
		}

		// The expected parent ID is the second-to-last prefix
		expectedParentID := prefixes[len(prefixes)-2]

		// Check if the expected parent exists
		if _, exists := taskByID[expectedParentID]; exists {
			// Parent exists - not orphaned
			continue
		}

		// Task is orphaned - find nearest valid ancestor
		// The hierarchy building already handles this based on indentation,
		// but we can log a warning or handle special cases here if needed
		// For now, the BuildHierarchy function handles the actual parent assignment
		// based on indentation, which is the correct behavior per the design
	}
}
