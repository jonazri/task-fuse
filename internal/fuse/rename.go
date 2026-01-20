// Package fuse provides the rename handler for atomic task status transitions.
package fuse

import (
	"context"
	"strings"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"task-fuse/internal/store"
)

// ValidTransitions defines the allowed status transitions.
// Key is the source status, value is a set of allowed destination statuses.
//
// Allowed transitions per Requirements 6.1, 6.2:
// - pending → queued, doing
// - queued → doing, pending
// - doing → done, pending, failed
// - done → doing
// - failed → pending, doing
var ValidTransitions = map[store.TaskStatus]map[store.TaskStatus]bool{
	store.StatusPending: {
		store.StatusQueued: true,
		store.StatusDoing:  true,
	},
	store.StatusQueued: {
		store.StatusDoing:   true,
		store.StatusPending: true,
	},
	store.StatusDoing: {
		store.StatusDone:    true,
		store.StatusPending: true,
		store.StatusFailed:  true,
	},
	store.StatusDone: {
		store.StatusDoing: true,
	},
	store.StatusFailed: {
		store.StatusPending: true,
		store.StatusDoing:   true,
	},
}

// IsValidTransition checks if a status transition is allowed.
func IsValidTransition(from, to store.TaskStatus) bool {
	allowed, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

// Rename handles the rename/move operation for status transitions.
// This is the core mechanism for agents to claim and transition tasks.
//
// The rename operation is used to move task files between status directories:
//   - mv /pending/1.task.md /doing/1.task.md  (claim task)
//   - mv /doing/1.task.md /done/1.task.md     (complete task)
//
// Validation rules (per Design - Rename Validation Rules):
//   - Same-directory renames: EPERM (no renaming files within a status dir)
//   - Filename changes: EPERM (cannot change task title/ID via filesystem)
//   - Parent directory moves: EPERM (parent status is derived)
//   - Invalid transitions: EPERM (e.g., pending→done)
//
// Returns:
//   - nil: success
//   - fuse.EPERM: same-dir rename, filename change, parent dir, invalid transition
//   - fuse.ENOENT: source doesn't exist or concurrent move
//
// Implements fs.NodeRenamer interface.
// Validates: Requirements 5.1, 5.2, 5.3, 5.4, 5.5, 6.1, 6.2, 6.3, 6.4
func (d *Dir) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	// Get the destination directory
	destDir, ok := newDir.(*Dir)
	if !ok {
		return fuse.EPERM
	}

	// Build source path
	srcPath := d.path + "/" + req.OldName
	// Note: dstPath would be destDir.path + "/" + req.NewName
	// but we don't need it since we use PathManager to rebuild paths

	// Extract source and destination status directories early (before lock)
	srcStatus := extractStatusFromPath(d.path)
	dstStatus := extractStatusFromPath(destDir.path)

	// Validate this is a cross-directory move (not same directory)
	// This check doesn't need the lock
	if srcStatus == dstStatus {
		return fuse.EPERM
	}

	// Validate filename is unchanged (doesn't need lock)
	if req.OldName != req.NewName {
		return fuse.EPERM
	}

	// Acquire write lock and use defer for release
	d.fs.store.Lock()
	unlocked := false
	defer func() {
		if !unlocked {
			d.fs.store.Unlock()
		}
	}()

	// Validate source path exists
	srcEntry := d.fs.store.TasksByPath[srcPath]
	if srcEntry == nil {
		return fuse.ENOENT
	}

	// Validate this is a leaf task file (not a parent directory)
	// Parent task directories cannot be moved (their status is derived)
	if len(srcEntry.Task.Children) > 0 {
		return fuse.EPERM
	}

	// Validate the status transition is allowed
	if !IsValidTransition(srcEntry.Task.Status, dstStatus) {
		return fuse.EPERM
	}

	// Perform the status transition
	srcEntry.Task.Status = dstStatus

	// Update paths using PathManager
	pm := store.NewPathManager(d.fs.store)
	pm.OnStatusChange(srcEntry.Task)

	// Release lock before sync (SyncToFile needs RLock)
	d.fs.store.Unlock()
	unlocked = true

	// Trigger sync to file asynchronously (outside of lock)
	if d.fs.syncEngine != nil {
		// Type assert to access SyncToFile method
		if syncer, ok := d.fs.syncEngine.(interface{ SyncToFile() error }); ok {
			go syncer.SyncToFile()
		}
	}

	return nil
}

// extractStatusFromPath extracts the status from a path.
// For paths like "/pending", "/pending/1.task", returns the status.
// For paths like "/pending/1.parent/1.1.child", returns the root status.
func extractStatusFromPath(path string) store.TaskStatus {
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Get the first path component
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 {
		return store.StatusPending
	}

	statusName := parts[0]
	status, ok := statusDirSet[statusName]
	if !ok {
		return store.StatusPending
	}

	return status
}

