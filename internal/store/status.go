// Package store provides status derivation utilities for parent tasks.
package store

// PropagateStatusChange propagates a status change up the task tree.
// When a task's status changes, this function recalculates all ancestor
// statuses and updates paths for affected subtrees.
//
// Parameters:
//   - task: The task whose status has changed
//   - store: The TaskStore containing the task data
//   - pm: The PathManager for updating paths
//
// Note: This function does NOT acquire locks - the caller must hold the write lock.
//
// Validates: Requirements 4.7
func PropagateStatusChange(task *Task, store *TaskStore, pm *PathManager) {
	if task == nil || store == nil || pm == nil {
		return
	}

	// The PathManager.OnStatusChange already handles:
	// 1. Finding the root ancestor
	// 2. Deriving the root's status (which recursively derives all ancestor statuses)
	// 3. Rebuilding paths for the entire subtree
	//
	// This ensures that when a child's status changes:
	// - All ancestor statuses are recalculated using DeriveStatus
	// - The entire subtree moves to the correct status directory
	// - All path indexes are updated
	pm.OnStatusChange(task)
}

// DeriveParentStatus derives the status for a task based on its children.
// For leaf tasks (no children): returns the task's own status.
// For parent tasks: derives status from children using precedence rules.
// Precedence: doing > failed > pending > queued > done
//
// This is a standalone function that can be used without a PathManager.
//
// Validates: Requirements 4.7
func DeriveParentStatus(task *Task) TaskStatus {
	if task == nil {
		return StatusPending
	}

	// Leaf task: return its own status
	if len(task.Children) == 0 {
		return task.Status
	}

	// Parent task: derive from children
	// Precedence: doing > failed > pending > queued > done
	hasDoing := false
	hasFailed := false
	hasPending := false
	hasQueued := false
	allDone := true

	for _, child := range task.Children {
		// Recursively derive status for nested children
		childStatus := DeriveParentStatus(child)
		switch childStatus {
		case StatusDoing:
			hasDoing = true
		case StatusFailed:
			hasFailed = true
		case StatusPending:
			hasPending = true
		case StatusQueued:
			hasQueued = true
		}
		if childStatus != StatusDone {
			allDone = false
		}
	}

	// Apply precedence rules: doing > failed > pending > queued > done
	if hasDoing {
		return StatusDoing
	}
	if hasFailed {
		return StatusFailed
	}
	if hasPending {
		return StatusPending
	}
	if hasQueued {
		return StatusQueued
	}
	if allDone {
		return StatusDone
	}

	// Default to pending (shouldn't reach here with valid children)
	return StatusPending
}
