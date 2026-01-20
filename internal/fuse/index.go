// Package fuse provides index file generation for the task filesystem.
package fuse

import (
	"fmt"
	"strings"

	"task-fuse/internal/store"
)

// IndexGenerator generates index.md content for directories in the filesystem.
// Index files are virtual and generated on-demand at read time to ensure they
// always reflect the current state.
//
// Validates: Requirements 3.4, 3.5, 3.6, 3.7, 3.8, 3.9
type IndexGenerator struct {
	store *store.TaskStore
}

// NewIndexGenerator creates a new IndexGenerator with the given TaskStore.
func NewIndexGenerator(taskStore *store.TaskStore) *IndexGenerator {
	return &IndexGenerator{
		store: taskStore,
	}
}

// GenerateRootIndex generates the root index.md content.
// Format: Overview with status counts.
//
// Example output:
//
//	# Task Overview
//
//	- pending/: 5 tasks
//	- queued/: 2 tasks
//	- doing/: 3 tasks
//	- done/: 10 tasks
//	- failed/: 1 tasks
//
// Validates: Requirements 3.4, 3.5
func (ig *IndexGenerator) GenerateRootIndex() string {
	counts := ig.getStatusCounts()

	return fmt.Sprintf("# Task Overview\n\n"+
		"- pending/: %d tasks\n"+
		"- queued/: %d tasks\n"+
		"- doing/: %d tasks\n"+
		"- done/: %d tasks\n"+
		"- failed/: %d tasks\n",
		counts[store.StatusPending],
		counts[store.StatusQueued],
		counts[store.StatusDoing],
		counts[store.StatusDone],
		counts[store.StatusFailed])
}

// GenerateStatusIndex generates a status directory index.md content.
// Format: Markdown list of all tasks within the status directory.
//
// Example output:
//
//	# Pending Tasks
//
//	- 📁 1. Setup project
//	- 📄 2. Write documentation
//
// Validates: Requirements 3.6, 3.7
func (ig *IndexGenerator) GenerateStatusIndex(status store.TaskStatus) string {
	tasks := ig.getTasksByStatus(status)
	title := strings.Title(string(status))

	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s Tasks\n\n", title))

	if len(tasks) == 0 {
		content.WriteString("No tasks in this status.\n")
		return content.String()
	}

	for _, task := range tasks {
		icon := "📄"
		if len(task.Children) > 0 {
			icon = "📁"
		}
		content.WriteString(fmt.Sprintf("- %s %s. %s\n", icon, task.ID, task.Title))
	}

	return content.String()
}

// GenerateTaskIndex generates a parent task directory index.md content.
// Format: Parent task description followed by sub-task list.
//
// Example output:
//
//	# 1. Setup project
//
//	**Status:** doing
//
//	Initialize the project structure
//
//	## Sub-tasks
//
//	- ✅ 1.1. Create directory structure
//	- 🔄 1.2. Configure build system
//	- ⬜ 1.3. Add dependencies
//
// Validates: Requirements 3.8
func (ig *IndexGenerator) GenerateTaskIndex(task *store.Task) string {
	if task == nil {
		return ""
	}

	// Derive status for parent tasks
	derivedStatus := ig.deriveStatus(task)

	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s. %s\n\n", task.ID, task.Title))
	content.WriteString(fmt.Sprintf("**Status:** %s\n\n", derivedStatus))

	// Include parent's description (preserved exactly as in tasks.md)
	for _, line := range task.Description {
		content.WriteString(line.RawContent + "\n")
	}

	// Add sub-tasks section if there are children
	if len(task.Children) > 0 {
		content.WriteString("\n## Sub-tasks\n\n")
		for _, child := range task.Children {
			emoji := ig.getStatusEmoji(child.Status)
			content.WriteString(fmt.Sprintf("- %s %s. %s\n", emoji, child.ID, child.Title))
		}
	}

	return content.String()
}

// GenerateTaskFileContent generates content for a leaf task file.
// Format follows Task File Content Format specification in requirements.md:
//
//	# {task_id}. {title}
//
//	**Status:** {status}
//
//	{description lines, preserved exactly as in tasks.md}
//
// Validates: Requirements 2.9
func (ig *IndexGenerator) GenerateTaskFileContent(task *store.Task) string {
	if task == nil {
		return ""
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s. %s\n\n", task.ID, task.Title))
	content.WriteString(fmt.Sprintf("**Status:** %s\n\n", task.Status))

	for _, line := range task.Description {
		content.WriteString(line.RawContent + "\n")
	}

	return content.String()
}

// getStatusCounts returns a map of status to task count.
// Only counts root-level tasks (tasks that appear directly in status directories).
func (ig *IndexGenerator) getStatusCounts() map[store.TaskStatus]int {
	counts := map[store.TaskStatus]int{
		store.StatusPending: 0,
		store.StatusQueued:  0,
		store.StatusDoing:   0,
		store.StatusDone:    0,
		store.StatusFailed:  0,
	}

	if ig.store == nil || ig.store.Document == nil {
		return counts
	}

	// Count root tasks by their derived status
	for _, task := range ig.store.Document.RootTasks {
		status := ig.deriveStatus(task)
		counts[status]++
	}

	return counts
}

// getTasksByStatus returns all root-level tasks with the given derived status.
func (ig *IndexGenerator) getTasksByStatus(status store.TaskStatus) []*store.Task {
	var tasks []*store.Task

	if ig.store == nil || ig.store.Document == nil {
		return tasks
	}

	for _, task := range ig.store.Document.RootTasks {
		derivedStatus := ig.deriveStatus(task)
		if derivedStatus == status {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// deriveStatus derives the status for a task.
// For leaf tasks: returns the task's own status.
// For parent tasks: derives status from children using precedence rules.
// Precedence: doing > failed > pending > queued > done
//
// This method delegates to the canonical store.DeriveParentStatus function
// to avoid code duplication.
func (ig *IndexGenerator) deriveStatus(task *store.Task) store.TaskStatus {
	return store.DeriveParentStatus(task)
}

// getStatusEmoji returns the emoji for a given status.
func (ig *IndexGenerator) getStatusEmoji(status store.TaskStatus) string {
	switch status {
	case store.StatusPending:
		return "⬜"
	case store.StatusQueued:
		return "🔜"
	case store.StatusDoing:
		return "🔄"
	case store.StatusDone:
		return "✅"
	case store.StatusFailed:
		return "❌"
	default:
		return "❓"
	}
}
