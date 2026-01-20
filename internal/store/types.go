// Package store provides the core data types and TaskStore for managing
// task data with concurrent access support.
package store

// TaskStatus represents the status of a task.
// Valid values are: pending, queued, doing, done, failed.
type TaskStatus string

const (
	// StatusPending represents a task that has not been started.
	// Checkbox syntax: - [ ]
	StatusPending TaskStatus = "pending"

	// StatusQueued represents a task that is queued for execution.
	// Checkbox syntax: - [~]
	StatusQueued TaskStatus = "queued"

	// StatusDoing represents a task that is currently in progress.
	// Checkbox syntax: - [-]
	StatusDoing TaskStatus = "doing"

	// StatusDone represents a task that has been completed successfully.
	// Checkbox syntax: - [x]
	StatusDone TaskStatus = "done"

	// StatusFailed represents a task that has failed.
	// Checkbox syntax: - [!]
	StatusFailed TaskStatus = "failed"
)

// FileLine represents any line in the file.
// It captures the original line content along with metadata about its position
// and indentation level.
type FileLine struct {
	// LineNumber is the 1-based line number in the original file.
	LineNumber int

	// RawContent is the original line content exactly as it appeared in the file.
	RawContent string

	// IndentLevel is the computed indent level.
	// Tabs are converted to 2 spaces, then total spaces are divided by 2 (floor).
	// For example: 2-4 spaces = 1 level, 1 tab = 1 level.
	IndentLevel int
}

// NonTaskContent represents non-task lines such as headings, preamble, comments,
// or any other content that is not a valid task line.
type NonTaskContent struct {
	FileLine
}

// Task represents a parsed task line.
//
// Task ID Format: Dot-separated positive integers matching regex ^[1-9][0-9]*(\.[1-9][0-9]*)*$
// Examples: "1", "1.2", "1.2.3", "10.5.2"
// Invalid: "01.1" (leading zero), "1.2.a" (non-numeric), "1..2" (empty segment)
// Maximum depth: 10 levels
type Task struct {
	// FileLine embeds the original line information.
	FileLine

	// ID is the numeric task identifier like "1", "1.1", "1.2.3".
	// REQUIRED - must match Task ID Format (dot-separated positive integers).
	ID string

	// Title is the task title text (everything after the ID on the task line).
	Title string

	// Status is the current task status (pending, queued, doing, done, failed).
	Status TaskStatus

	// Description contains lines attached to this task.
	// These are lines that follow the task line and are more indented,
	// but are not valid sub-tasks.
	Description []FileLine

	// Children contains sub-tasks (empty for leaf tasks).
	// A task with children is a "parent task" and is represented as a directory
	// in the filesystem.
	Children []*Task

	// Parent is a reference to the parent task (nil for root tasks).
	Parent *Task
}

// ParsedDocument represents the complete parsed tasks.md file.
// It preserves all content from the original file, including non-task content.
type ParsedDocument struct {
	// Preamble contains all lines before the first task.
	// This typically includes headings, introduction text, etc.
	Preamble []NonTaskContent

	// RootTasks contains top-level tasks with their nested children.
	// These are tasks with zero indentation.
	RootTasks []*Task

	// Epilogue contains all lines after the last task.
	Epilogue []NonTaskContent
}
