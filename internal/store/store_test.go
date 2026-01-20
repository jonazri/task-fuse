package store

import (
	"sync"
	"testing"
)

func TestNewTaskStore(t *testing.T) {
	store := NewTaskStore()

	if store == nil {
		t.Fatal("NewTaskStore() returned nil")
	}

	if store.Document != nil {
		t.Error("NewTaskStore() should have nil Document")
	}

	if store.TasksByPath == nil {
		t.Error("NewTaskStore() should initialize TasksByPath map")
	}

	if store.TasksByID == nil {
		t.Error("NewTaskStore() should initialize TasksByID map")
	}

	if store.PathByTask == nil {
		t.Error("NewTaskStore() should initialize PathByTask map")
	}

	if len(store.TasksByPath) != 0 {
		t.Error("NewTaskStore() should have empty TasksByPath map")
	}

	if len(store.TasksByID) != 0 {
		t.Error("NewTaskStore() should have empty TasksByID map")
	}

	if len(store.PathByTask) != 0 {
		t.Error("NewTaskStore() should have empty PathByTask map")
	}
}

func TestPathEntry(t *testing.T) {
	task := &Task{
		ID:     "1.1",
		Title:  "Test Task",
		Status: StatusPending,
	}

	entry := &PathEntry{
		Task:     task,
		Status:   StatusPending,
		FullPath: "/pending/1.1.test_task.md",
	}

	if entry.Task != task {
		t.Error("PathEntry.Task should reference the task")
	}

	if entry.Status != StatusPending {
		t.Errorf("PathEntry.Status = %v, want %v", entry.Status, StatusPending)
	}

	if entry.FullPath != "/pending/1.1.test_task.md" {
		t.Errorf("PathEntry.FullPath = %v, want /pending/1.1.test_task.md", entry.FullPath)
	}
}

func TestTaskStore_AddPathEntry(t *testing.T) {
	store := NewTaskStore()

	task := &Task{
		ID:     "1.2",
		Title:  "Add Entry Test",
		Status: StatusDoing,
	}

	entry := &PathEntry{
		Task:     task,
		Status:   StatusDoing,
		FullPath: "/doing/1.2.add_entry_test.md",
	}

	store.AddPathEntry(entry)

	// Verify TasksByPath
	if got := store.TasksByPath["/doing/1.2.add_entry_test.md"]; got != entry {
		t.Error("AddPathEntry should add entry to TasksByPath")
	}

	// Verify TasksByID
	if got := store.TasksByID["1.2"]; got != entry {
		t.Error("AddPathEntry should add entry to TasksByID")
	}

	// Verify PathByTask
	if got := store.PathByTask[task]; got != "/doing/1.2.add_entry_test.md" {
		t.Error("AddPathEntry should add path to PathByTask")
	}
}

func TestTaskStore_AddPathEntry_NilEntry(t *testing.T) {
	store := NewTaskStore()

	// Should not panic with nil entry
	store.AddPathEntry(nil)

	if len(store.TasksByPath) != 0 {
		t.Error("AddPathEntry(nil) should not add anything")
	}
}

func TestTaskStore_AddPathEntry_NilTask(t *testing.T) {
	store := NewTaskStore()

	entry := &PathEntry{
		Task:     nil,
		Status:   StatusPending,
		FullPath: "/pending/test.md",
	}

	// Should not panic with nil task
	store.AddPathEntry(entry)

	if len(store.TasksByPath) != 0 {
		t.Error("AddPathEntry with nil Task should not add anything")
	}
}

func TestTaskStore_RemovePathEntry(t *testing.T) {
	store := NewTaskStore()

	task := &Task{
		ID:     "1.3",
		Title:  "Remove Entry Test",
		Status: StatusDone,
	}

	entry := &PathEntry{
		Task:     task,
		Status:   StatusDone,
		FullPath: "/done/1.3.remove_entry_test.md",
	}

	store.AddPathEntry(entry)

	// Verify entry was added
	if store.TasksByPath["/done/1.3.remove_entry_test.md"] == nil {
		t.Fatal("Entry should be added before removal test")
	}

	// Remove the entry
	store.RemovePathEntry("/done/1.3.remove_entry_test.md")

	// Verify removal from all indexes
	if store.TasksByPath["/done/1.3.remove_entry_test.md"] != nil {
		t.Error("RemovePathEntry should remove from TasksByPath")
	}

	if store.TasksByID["1.3"] != nil {
		t.Error("RemovePathEntry should remove from TasksByID")
	}

	if store.PathByTask[task] != "" {
		t.Error("RemovePathEntry should remove from PathByTask")
	}
}

func TestTaskStore_RemovePathEntry_NonExistent(t *testing.T) {
	store := NewTaskStore()

	// Should not panic when removing non-existent path
	store.RemovePathEntry("/nonexistent/path.md")

	if len(store.TasksByPath) != 0 {
		t.Error("RemovePathEntry on non-existent path should not add anything")
	}
}

func TestTaskStore_GetTaskByPath(t *testing.T) {
	store := NewTaskStore()

	task := &Task{
		ID:     "2.1",
		Title:  "Get By Path Test",
		Status: StatusQueued,
	}

	entry := &PathEntry{
		Task:     task,
		Status:   StatusQueued,
		FullPath: "/queued/2.1.get_by_path_test.md",
	}

	store.AddPathEntry(entry)

	// Test successful lookup
	got := store.GetTaskByPath("/queued/2.1.get_by_path_test.md")
	if got != entry {
		t.Error("GetTaskByPath should return the correct entry")
	}

	// Test non-existent path
	got = store.GetTaskByPath("/nonexistent/path.md")
	if got != nil {
		t.Error("GetTaskByPath should return nil for non-existent path")
	}
}

func TestTaskStore_GetTaskByID(t *testing.T) {
	store := NewTaskStore()

	task := &Task{
		ID:     "3.2.1",
		Title:  "Get By ID Test",
		Status: StatusFailed,
	}

	entry := &PathEntry{
		Task:     task,
		Status:   StatusFailed,
		FullPath: "/failed/3.2.1.get_by_id_test.md",
	}

	store.AddPathEntry(entry)

	// Test successful lookup
	got := store.GetTaskByID("3.2.1")
	if got != entry {
		t.Error("GetTaskByID should return the correct entry")
	}

	// Test non-existent ID
	got = store.GetTaskByID("999.999")
	if got != nil {
		t.Error("GetTaskByID should return nil for non-existent ID")
	}
}

func TestTaskStore_GetPathByTask(t *testing.T) {
	store := NewTaskStore()

	task := &Task{
		ID:     "4.1",
		Title:  "Get Path By Task Test",
		Status: StatusPending,
	}

	entry := &PathEntry{
		Task:     task,
		Status:   StatusPending,
		FullPath: "/pending/4.1.get_path_by_task_test.md",
	}

	store.AddPathEntry(entry)

	// Test successful lookup
	got := store.GetPathByTask(task)
	if got != "/pending/4.1.get_path_by_task_test.md" {
		t.Errorf("GetPathByTask = %v, want /pending/4.1.get_path_by_task_test.md", got)
	}

	// Test non-existent task
	otherTask := &Task{ID: "999"}
	got = store.GetPathByTask(otherTask)
	if got != "" {
		t.Error("GetPathByTask should return empty string for non-existent task")
	}
}

func TestTaskStore_UpdateTaskPath(t *testing.T) {
	store := NewTaskStore()

	task := &Task{
		ID:     "5.1",
		Title:  "Update Path Test",
		Status: StatusPending,
	}

	// Add initial entry
	initialEntry := &PathEntry{
		Task:     task,
		Status:   StatusPending,
		FullPath: "/pending/5.1.update_path_test.md",
	}
	store.AddPathEntry(initialEntry)

	// Update the path (simulating a status change)
	store.UpdateTaskPath(task, "/doing/5.1.update_path_test.md", StatusDoing)

	// Verify old path is removed
	if store.TasksByPath["/pending/5.1.update_path_test.md"] != nil {
		t.Error("UpdateTaskPath should remove old path from TasksByPath")
	}

	// Verify new path is added
	newEntry := store.TasksByPath["/doing/5.1.update_path_test.md"]
	if newEntry == nil {
		t.Fatal("UpdateTaskPath should add new path to TasksByPath")
	}

	if newEntry.Status != StatusDoing {
		t.Errorf("UpdateTaskPath entry.Status = %v, want %v", newEntry.Status, StatusDoing)
	}

	// Verify PathByTask is updated
	if store.PathByTask[task] != "/doing/5.1.update_path_test.md" {
		t.Error("UpdateTaskPath should update PathByTask")
	}

	// Verify TasksByID still points to the task
	if store.TasksByID["5.1"] != newEntry {
		t.Error("UpdateTaskPath should update TasksByID")
	}
}

func TestTaskStore_UpdateTaskPath_NilTask(t *testing.T) {
	store := NewTaskStore()

	// Should not panic with nil task
	store.UpdateTaskPath(nil, "/pending/test.md", StatusPending)

	if len(store.TasksByPath) != 0 {
		t.Error("UpdateTaskPath(nil, ...) should not add anything")
	}
}

func TestTaskStore_SetDocument(t *testing.T) {
	store := NewTaskStore()

	// Add some entries first
	task := &Task{
		ID:     "6.1",
		Title:  "Set Document Test",
		Status: StatusPending,
	}
	entry := &PathEntry{
		Task:     task,
		Status:   StatusPending,
		FullPath: "/pending/6.1.set_document_test.md",
	}
	store.AddPathEntry(entry)

	// Create a new document
	doc := &ParsedDocument{
		RootTasks: []*Task{
			{ID: "1", Title: "New Task", Status: StatusPending},
		},
	}

	// Set the document
	store.SetDocument(doc)

	// Verify document is set
	if store.Document != doc {
		t.Error("SetDocument should set the Document field")
	}

	// Verify indexes are cleared
	if len(store.TasksByPath) != 0 {
		t.Error("SetDocument should clear TasksByPath")
	}

	if len(store.TasksByID) != 0 {
		t.Error("SetDocument should clear TasksByID")
	}

	if len(store.PathByTask) != 0 {
		t.Error("SetDocument should clear PathByTask")
	}
}

func TestTaskStore_RWLock_ReadLock(t *testing.T) {
	store := NewTaskStore()

	// Test that multiple read locks can be acquired
	store.RLock()
	store.RLock()

	// Both should succeed without deadlock - access document to verify lock held
	_ = store.Document
	store.RUnlock()
	store.RUnlock()
}

func TestTaskStore_RWLock_WriteLock(t *testing.T) {
	store := NewTaskStore()

	// Test that write lock can be acquired and released
	store.Lock()
	// Access document to verify lock held
	_ = store.Document
	store.Unlock()
}

func TestTaskStore_ConcurrentReads(t *testing.T) {
	store := NewTaskStore()

	// Add a task
	task := &Task{
		ID:     "7.1",
		Title:  "Concurrent Read Test",
		Status: StatusPending,
	}
	entry := &PathEntry{
		Task:     task,
		Status:   StatusPending,
		FullPath: "/pending/7.1.concurrent_read_test.md",
	}
	store.AddPathEntry(entry)

	// Run multiple concurrent reads
	var wg sync.WaitGroup
	numReaders := 10

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.RLock()
			defer store.RUnlock()

			// Perform read operations
			_ = store.GetTaskByPath("/pending/7.1.concurrent_read_test.md")
			_ = store.GetTaskByID("7.1")
			_ = store.GetPathByTask(task)
		}()
	}

	wg.Wait()
}

func TestTaskStore_ConcurrentReadWrite(t *testing.T) {
	store := NewTaskStore()

	// Add initial task
	task := &Task{
		ID:     "8.1",
		Title:  "Concurrent RW Test",
		Status: StatusPending,
	}
	entry := &PathEntry{
		Task:     task,
		Status:   StatusPending,
		FullPath: "/pending/8.1.concurrent_rw_test.md",
	}
	store.AddPathEntry(entry)

	var wg sync.WaitGroup

	// Start readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				store.RLock()
				_ = store.GetTaskByID("8.1")
				store.RUnlock()
			}
		}()
	}

	// Start a writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 10; j++ {
			store.Lock()
			store.UpdateTaskPath(task, "/doing/8.1.concurrent_rw_test.md", StatusDoing)
			store.UpdateTaskPath(task, "/pending/8.1.concurrent_rw_test.md", StatusPending)
			store.Unlock()
		}
	}()

	wg.Wait()
}
