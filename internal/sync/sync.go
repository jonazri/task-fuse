// Package sync provides the Sync_Engine component responsible for
// bidirectional synchronization between the filesystem and tasks.md.
package sync

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"task-fuse/internal/parser"
	"task-fuse/internal/printer"
	"task-fuse/internal/store"
)

// FileChange represents an external modification to the tasks.md file.
type FileChange struct {
	Timestamp time.Time
	Content   string
}

// SyncEngine maintains bidirectional synchronization between the
// FUSE filesystem and the tasks.md file.
//
// The engine:
// - Watches the tasks.md file for external changes
// - Debounces rapid changes (100ms window)
// - Queues changes during filesystem operations
// - Resolves conflicts (filesystem operations take precedence)
type SyncEngine struct {
	// store is the TaskStore containing all task data
	store *store.TaskStore

	// parser is used to parse tasks.md content
	parser *parser.Parser

	// printer is used to serialize tasks back to markdown
	printer *printer.Printer

	// tasksFile is the path to the tasks.md file
	tasksFile string

	// watcher is the fsnotify file watcher
	watcher *fsnotify.Watcher

	// stopChan signals the watcher goroutine to stop
	stopChan chan struct{}

	// debounceTimer is used to debounce rapid file changes
	debounceTimer *time.Timer

	// debounceDuration is the debounce window (default 100ms)
	debounceDuration time.Duration

	// pollInterval is the polling interval for fallback mode
	pollInterval time.Duration

	// usePolling indicates whether to use polling instead of fsnotify
	usePolling bool

	// lastModTime tracks the last modification time for polling
	lastModTime time.Time

	// mu protects internal state
	mu sync.Mutex

	// running indicates whether the engine is currently running
	running bool

	// pendingSync indicates a sync is pending (debounced)
	pendingSync bool

	// inOperation indicates a filesystem operation is in progress
	inOperation bool

	// queuedChanges holds changes that occurred during operations
	queuedChanges []FileChange
}

// NewSyncEngine creates a new SyncEngine instance.
//
// Parameters:
//   - store: The TaskStore containing all task data
//   - parser: The parser for reading tasks.md
//   - printer: The printer for writing tasks.md
//   - tasksFile: Path to the tasks.md file
//
// Returns a new SyncEngine ready to be started.
func NewSyncEngine(store *store.TaskStore, parser *parser.Parser, printer *printer.Printer, tasksFile string) *SyncEngine {
	return &SyncEngine{
		store:            store,
		parser:           parser,
		printer:          printer,
		tasksFile:        tasksFile,
		stopChan:         make(chan struct{}),
		debounceDuration: 100 * time.Millisecond,
		pollInterval:     1 * time.Second,
		usePolling:       false,
		queuedChanges:    make([]FileChange, 0),
	}
}

// SetDebounceDuration sets the debounce duration for file change detection.
// Default is 100ms.
func (s *SyncEngine) SetDebounceDuration(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.debounceDuration = d
}

// SetPollInterval sets the polling interval for fallback mode.
// Default is 1 second.
func (s *SyncEngine) SetPollInterval(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pollInterval = d
}

// StartWatching starts watching the tasks.md file for changes.
// Uses fsnotify for efficient file watching, with polling as fallback.
//
// Returns an error if the watcher cannot be started.
//
// Validates: Requirements 4.5
func (s *SyncEngine) StartWatching() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	// Try to create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		// Fall back to polling
		s.mu.Lock()
		s.usePolling = true
		s.mu.Unlock()
		go s.pollLoop()
		return nil
	}

	s.watcher = watcher

	// Add the tasks file to the watcher
	err = s.watcher.Add(s.tasksFile)
	if err != nil {
		s.watcher.Close()
		// Fall back to polling
		s.mu.Lock()
		s.usePolling = true
		s.mu.Unlock()
		go s.pollLoop()
		return nil
	}

	// Start the watcher goroutine
	go s.watchLoop()

	return nil
}

// StopWatching stops watching the tasks.md file.
// Flushes any pending changes before stopping.
//
// Validates: Requirements 7.3
func (s *SyncEngine) StopWatching() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	// Signal stop
	close(s.stopChan)

	// Close watcher if using fsnotify
	if s.watcher != nil {
		s.watcher.Close()
	}

	// Cancel any pending debounce timer
	s.mu.Lock()
	if s.debounceTimer != nil {
		s.debounceTimer.Stop()
	}
	s.mu.Unlock()

	// Flush pending changes
	s.SyncToFile()
}

// GracefulShutdown performs a graceful shutdown of the sync engine.
// Waits for in-progress operations to complete and flushes pending changes.
//
// Parameters:
//   - timeout: maximum time to wait for operations to complete
//
// Returns an error if the timeout is exceeded.
//
// Validates: Requirements 7.3
func (s *SyncEngine) GracefulShutdown(timeout time.Duration) error {
	// Wait for any in-progress operations
	deadline := time.Now().Add(timeout)
	for {
		s.mu.Lock()
		inOp := s.inOperation
		s.mu.Unlock()

		if !inOp {
			break
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for operations to complete")
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Stop watching and flush changes
	s.StopWatching()

	return nil
}

// watchLoop is the main loop for fsnotify-based file watching.
func (s *SyncEngine) watchLoop() {
	for {
		select {
		case <-s.stopChan:
			return

		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Only process write events
			if event.Op&fsnotify.Write == fsnotify.Write {
				s.handleFileChange()
			}

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			// Log error and continue
			slog.Error("file watcher error", "error", err, "file", s.tasksFile)
		}
	}
}

// pollLoop is the fallback polling loop when fsnotify is unavailable.
//
// Validates: Design - Graceful Degradation
func (s *SyncEngine) pollLoop() {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	// Get initial modification time
	info, err := os.Stat(s.tasksFile)
	if err == nil {
		s.mu.Lock()
		s.lastModTime = info.ModTime()
		s.mu.Unlock()
	}

	for {
		select {
		case <-s.stopChan:
			return

		case <-ticker.C:
			info, err := os.Stat(s.tasksFile)
			if err != nil {
				continue
			}

			s.mu.Lock()
			lastMod := s.lastModTime
			s.mu.Unlock()

			if info.ModTime().After(lastMod) {
				s.mu.Lock()
				s.lastModTime = info.ModTime()
				s.mu.Unlock()
				s.handleFileChange()
			}
		}
	}
}

// handleFileChange handles a detected file change with debouncing.
func (s *SyncEngine) handleFileChange() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel any existing debounce timer
	if s.debounceTimer != nil {
		s.debounceTimer.Stop()
	}

	// Set pending sync flag
	s.pendingSync = true

	// Start new debounce timer
	s.debounceTimer = time.AfterFunc(s.debounceDuration, func() {
		s.mu.Lock()
		s.pendingSync = false
		inOp := s.inOperation
		s.mu.Unlock()

		if inOp {
			// Queue the change for later processing
			content, err := os.ReadFile(s.tasksFile)
			if err == nil {
				s.QueueExternalChange(FileChange{
					Timestamp: time.Now(),
					Content:   string(content),
				})
			}
		} else {
			// Process immediately
			s.SyncFromFile()
		}
	})
}

// BeginOperation marks the start of a filesystem operation.
// External changes during operations are queued for later processing.
//
// Validates: Requirements 4.6
func (s *SyncEngine) BeginOperation() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inOperation = true
}

// EndOperation marks the end of a filesystem operation.
// Processes any queued external changes.
//
// Validates: Requirements 4.6
func (s *SyncEngine) EndOperation() {
	s.mu.Lock()
	s.inOperation = false
	s.mu.Unlock()

	// Process queued changes
	s.ProcessQueuedChanges()
}

// QueueExternalChange queues an external modification for later processing.
//
// Validates: Requirements 4.6
func (s *SyncEngine) QueueExternalChange(change FileChange) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queuedChanges = append(s.queuedChanges, change)
}

// ProcessQueuedChanges processes all queued external changes.
// Only the most recent change is applied (older changes are discarded).
//
// Validates: Requirements 4.6
func (s *SyncEngine) ProcessQueuedChanges() {
	s.mu.Lock()
	if len(s.queuedChanges) == 0 {
		s.mu.Unlock()
		return
	}

	// Get the most recent change
	latestChange := s.queuedChanges[len(s.queuedChanges)-1]
	s.queuedChanges = s.queuedChanges[:0] // Clear queue
	s.mu.Unlock()

	// Process the change
	s.processExternalChange(latestChange)
}

// processExternalChange processes a single external change.
func (s *SyncEngine) processExternalChange(change FileChange) {
	// Parse the new content
	result := s.parser.Parse(change.Content)
	if result == nil {
		return
	}

	// Apply changes to the store
	// Note: Conflict resolution happens here - filesystem operations take precedence
	s.applyExternalChanges(result.Document)
}

// applyExternalChanges applies parsed changes to the store.
// Filesystem operations take precedence over external modifications.
//
// Validates: Requirements 4.4
func (s *SyncEngine) applyExternalChanges(newDoc *store.ParsedDocument) {
	if newDoc == nil {
		return
	}

	s.store.Lock()
	defer s.store.Unlock()

	oldDoc := s.store.Document
	if oldDoc == nil {
		// No existing document, just use the new one
		s.store.Document = newDoc
		pm := store.NewPathManager(s.store)
		pm.BuildAllPaths()
		return
	}

	// Build a map of old tasks by ID for diffing
	oldTasksByID := make(map[string]*store.Task)
	collectTasksByID(oldDoc.RootTasks, oldTasksByID)

	// Build a map of new tasks by ID
	newTasksByID := make(map[string]*store.Task)
	collectTasksByID(newDoc.RootTasks, newTasksByID)

	// Detect orphaned children (children whose parents were removed)
	orphanedChildren := s.detectOrphanedChildren(oldTasksByID, newTasksByID)
	if len(orphanedChildren) > 0 {
		// Log warning about orphaned children (Validates: Requirements 4.9)
		slog.Warn("orphaned children detected and promoted",
			"orphaned_task_ids", orphanedChildren,
			"count", len(orphanedChildren))
		// Promote orphaned children to their grandparent level
		s.promoteOrphanedChildren(newDoc, orphanedChildren, oldTasksByID)
	}

	// Apply non-conflicting changes
	// For each task in the new document:
	// - If task exists in old document with same status, preserve old task's status
	//   (filesystem operations take precedence)
	// - If task is new, add it
	// - If task was removed, remove it (unless it was modified by filesystem)
	for id, newTask := range newTasksByID {
		if oldTask, exists := oldTasksByID[id]; exists {
			// Task exists in both - preserve filesystem status if different
			// (filesystem operations take precedence)
			if oldTask.Status != newTask.Status {
				// Keep the old status (filesystem wins)
				newTask.Status = oldTask.Status
			}
			// Preserve description from new document (content changes are allowed)
		}
	}

	// Update the document
	s.store.Document = newDoc

	// Rebuild paths
	pm := store.NewPathManager(s.store)
	pm.BuildAllPaths()
}

// detectOrphanedChildren detects children whose parents were removed in an external edit.
// Returns a list of orphaned task IDs.
//
// Validates: Requirements 4.9
func (s *SyncEngine) detectOrphanedChildren(oldTasks, newTasks map[string]*store.Task) []string {
	var orphaned []string

	for id, oldTask := range oldTasks {
		// Check if this task still exists in new document
		if _, exists := newTasks[id]; !exists {
			// Task was removed - check if it had children that still exist
			for _, child := range oldTask.Children {
				if _, childExists := newTasks[child.ID]; childExists {
					// Child exists but parent was removed - child is orphaned
					orphaned = append(orphaned, child.ID)
				}
			}
		}
	}

	return orphaned
}

// promoteOrphanedChildren promotes orphaned children to their grandparent level.
// Preserves original task IDs (no renumbering).
//
// Validates: Requirements 4.8
func (s *SyncEngine) promoteOrphanedChildren(doc *store.ParsedDocument, orphanedIDs []string, oldTasks map[string]*store.Task) {
	if doc == nil || len(orphanedIDs) == 0 {
		return
	}

	// Build a set of orphaned IDs for quick lookup
	orphanedSet := make(map[string]bool)
	for _, id := range orphanedIDs {
		orphanedSet[id] = true
	}

	// Find orphaned tasks in the new document and update their parent references
	for _, id := range orphanedIDs {
		// Find the task in the new document
		task := findTaskByID(doc.RootTasks, id)
		if task == nil {
			continue
		}

		// Find the old parent
		oldTask, exists := oldTasks[id]
		if !exists || oldTask.Parent == nil {
			continue
		}

		oldParent := oldTask.Parent
		grandparent := oldParent.Parent

		// Remove from current parent (if any)
		if task.Parent != nil {
			removeChildFromParent(task.Parent, task)
		}

		// Add to grandparent or root level
		if grandparent != nil {
			// Find grandparent in new document
			newGrandparent := findTaskByID(doc.RootTasks, grandparent.ID)
			if newGrandparent != nil {
				task.Parent = newGrandparent
				newGrandparent.Children = append(newGrandparent.Children, task)
			} else {
				// Grandparent doesn't exist, promote to root
				task.Parent = nil
				doc.RootTasks = append(doc.RootTasks, task)
			}
		} else {
			// Parent was a root task, promote child to root
			task.Parent = nil
			// Check if already in root tasks
			found := false
			for _, rt := range doc.RootTasks {
				if rt.ID == task.ID {
					found = true
					break
				}
			}
			if !found {
				doc.RootTasks = append(doc.RootTasks, task)
			}
		}
	}
}

// findTaskByID recursively finds a task by ID in a task tree.
func findTaskByID(tasks []*store.Task, id string) *store.Task {
	for _, task := range tasks {
		if task.ID == id {
			return task
		}
		if found := findTaskByID(task.Children, id); found != nil {
			return found
		}
	}
	return nil
}

// removeChildFromParent removes a child task from its parent's children list.
func removeChildFromParent(parent, child *store.Task) {
	if parent == nil {
		return
	}
	for i, c := range parent.Children {
		if c.ID == child.ID {
			parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
			return
		}
	}
}

// collectTasksByID recursively collects all tasks into a map by ID.
func collectTasksByID(tasks []*store.Task, result map[string]*store.Task) {
	for _, task := range tasks {
		result[task.ID] = task
		if len(task.Children) > 0 {
			collectTasksByID(task.Children, result)
		}
	}
}

// SyncFromFile reads the tasks.md file and updates the store.
//
// Returns:
//   - nil: success
//   - os.ErrNotExist: file not found (maps to EIO in FUSE)
//   - os.ErrPermission: permission denied (maps to EACCES in FUSE)
//   - other errors: various I/O errors
//
// Validates: Requirements 4.2, 7.1, 7.2
func (s *SyncEngine) SyncFromFile() error {
	content, err := os.ReadFile(s.tasksFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File not found - return specific error
			return os.ErrNotExist
		}
		if os.IsPermission(err) {
			// Permission denied - return specific error
			return os.ErrPermission
		}
		return err
	}

	result := s.parser.Parse(string(content))
	if result == nil {
		return nil
	}

	s.applyExternalChanges(result.Document)
	return nil
}

// SyncToFile writes the current store state to tasks.md.
// Uses atomic write (write to temp file, then rename).
//
// Validates: Requirements 4.1
func (s *SyncEngine) SyncToFile() error {
	s.store.RLock()
	doc := s.store.Document
	s.store.RUnlock()

	if doc == nil {
		return nil
	}

	// Serialize the document
	content := s.printer.Serialize(doc)

	// Write to temporary file
	tempFile := s.tasksFile + ".tmp"
	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		return err
	}

	// Atomic rename
	err = os.Rename(tempFile, s.tasksFile)
	if err != nil {
		// Clean up temp file on failure
		if removeErr := os.Remove(tempFile); removeErr != nil {
			slog.Error("failed to remove temp file after rename failure",
				"temp_file", tempFile,
				"rename_error", err,
				"remove_error", removeErr)
		}
		return err
	}

	return nil
}

// IsRunning returns whether the sync engine is currently running.
func (s *SyncEngine) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// IsUsingPolling returns whether the engine is using polling mode.
func (s *SyncEngine) IsUsingPolling() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usePolling
}
