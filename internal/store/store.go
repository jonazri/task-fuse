// Package store provides the TaskStore for managing task data with
// concurrent access support using read-write locks.
package store

import (
	"sync"
)

// PathEntry provides efficient path lookup for tasks.
// It maps a task to its current location in the filesystem.
type PathEntry struct {
	// Task is a pointer to the task this entry represents.
	Task *Task

	// Status is the current status of the task.
	// For leaf tasks, this is the task's own status.
	// For parent tasks, this may be the derived status (which can differ
	// from the task's stored status during transitions).
	Status TaskStatus

	// FullPath is the full path from the mount root.
	// Example: "/pending/1.setup_project/1.1.create_structure.md"
	FullPath string
}

// TaskStore holds all task data with concurrent access support.
// It provides multiple indexes for efficient lookup by path, ID, or task pointer.
//
// Locking Strategy:
//   - Read operations (readdir, read, getattr): Acquire read lock via RLock()
//   - Write operations (rename): Acquire write lock via Lock()
//   - Multiple concurrent reads are allowed
//   - Writes have exclusive access (no concurrent reads or writes)
//
// This allows high read throughput while ensuring atomic status transitions.
type TaskStore struct {
	// Document is the parsed tasks.md document containing all tasks.
	Document *ParsedDocument

	// TasksByPath maps full filesystem paths to PathEntry.
	// Example key: "/pending/1.setup_project/1.1.create_structure.md"
	TasksByPath map[string]*PathEntry

	// TasksByID maps task IDs to PathEntry.
	// Example key: "1.1"
	TasksByID map[string]*PathEntry

	// PathByTask maps task pointers to their current filesystem path.
	// This is the reverse mapping of TasksByPath.
	PathByTask map[*Task]string

	// RWLock provides read-write locking for concurrent access.
	// Use RLock()/RUnlock() for read operations.
	// Use Lock()/Unlock() for write operations.
	RWLock sync.RWMutex
}

// NewTaskStore creates a new TaskStore with initialized maps.
// The returned store has empty maps and no document loaded.
// Use SetDocument() or populate the maps directly after creation.
func NewTaskStore() *TaskStore {
	return &TaskStore{
		Document:    nil,
		TasksByPath: make(map[string]*PathEntry),
		TasksByID:   make(map[string]*PathEntry),
		PathByTask:  make(map[*Task]string),
	}
}

// RLock acquires a read lock on the store.
// Multiple goroutines can hold read locks simultaneously.
// Use this for read operations like readdir, read, getattr.
// Must be paired with RUnlock().
func (s *TaskStore) RLock() {
	s.RWLock.RLock()
}

// RUnlock releases a read lock on the store.
// Must be called after RLock() when the read operation is complete.
func (s *TaskStore) RUnlock() {
	s.RWLock.RUnlock()
}

// Lock acquires a write lock on the store.
// Only one goroutine can hold the write lock, and it blocks all readers.
// Use this for write operations like rename (status transitions).
// Must be paired with Unlock().
func (s *TaskStore) Lock() {
	s.RWLock.Lock()
}

// Unlock releases a write lock on the store.
// Must be called after Lock() when the write operation is complete.
func (s *TaskStore) Unlock() {
	s.RWLock.Unlock()
}

// SetDocument sets the parsed document and clears all indexes.
// This should be called when loading or reloading the tasks.md file.
// The caller is responsible for rebuilding the path indexes after calling this.
// Note: This method does NOT acquire locks - the caller must hold the write lock.
func (s *TaskStore) SetDocument(doc *ParsedDocument) {
	s.Document = doc
	s.TasksByPath = make(map[string]*PathEntry)
	s.TasksByID = make(map[string]*PathEntry)
	s.PathByTask = make(map[*Task]string)
}

// GetTaskByPath returns the PathEntry for the given filesystem path.
// Returns nil if no task exists at that path.
// Note: This method does NOT acquire locks - the caller must hold at least a read lock.
func (s *TaskStore) GetTaskByPath(path string) *PathEntry {
	return s.TasksByPath[path]
}

// GetTaskByID returns the PathEntry for the given task ID.
// Returns nil if no task exists with that ID.
// Note: This method does NOT acquire locks - the caller must hold at least a read lock.
func (s *TaskStore) GetTaskByID(id string) *PathEntry {
	return s.TasksByID[id]
}

// GetPathByTask returns the current filesystem path for the given task.
// Returns empty string if the task is not in the store.
// Note: This method does NOT acquire locks - the caller must hold at least a read lock.
func (s *TaskStore) GetPathByTask(task *Task) string {
	return s.PathByTask[task]
}

// AddPathEntry adds or updates a path entry in all indexes.
// This updates TasksByPath, TasksByID, and PathByTask atomically.
// Note: This method does NOT acquire locks - the caller must hold the write lock.
func (s *TaskStore) AddPathEntry(entry *PathEntry) {
	if entry == nil || entry.Task == nil {
		return
	}

	// Update all indexes
	s.TasksByPath[entry.FullPath] = entry
	s.TasksByID[entry.Task.ID] = entry
	s.PathByTask[entry.Task] = entry.FullPath
}

// RemovePathEntry removes a path entry from all indexes.
// Note: This method does NOT acquire locks - the caller must hold the write lock.
func (s *TaskStore) RemovePathEntry(path string) {
	entry := s.TasksByPath[path]
	if entry == nil {
		return
	}

	// Remove from all indexes
	delete(s.TasksByPath, path)
	if entry.Task != nil {
		delete(s.TasksByID, entry.Task.ID)
		delete(s.PathByTask, entry.Task)
	}
}

// UpdateTaskPath updates the path for a task, removing the old path entry
// and adding a new one with the updated path.
// Note: This method does NOT acquire locks - the caller must hold the write lock.
func (s *TaskStore) UpdateTaskPath(task *Task, newPath string, newStatus TaskStatus) {
	if task == nil {
		return
	}

	// Remove old path entry if it exists
	oldPath := s.PathByTask[task]
	if oldPath != "" {
		delete(s.TasksByPath, oldPath)
	}

	// Create and add new entry
	entry := &PathEntry{
		Task:     task,
		Status:   newStatus,
		FullPath: newPath,
	}
	s.AddPathEntry(entry)
}
