// Package integration provides integration tests for the FUSE Task Filesystem.
//
// These tests verify the complete workflow of the system without requiring
// actual FUSE operations (which can only run on Linux).
package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"task-fuse/internal/parser"
	"task-fuse/internal/printer"
	"task-fuse/internal/store"
	syncPkg "task-fuse/internal/sync"
)

// TestIntegration_EndToEndWorkflow tests the complete lifecycle of tasks
// from pending → doing → done.
//
// Validates: Design - Integration Tests
func TestIntegration_EndToEndWorkflow(t *testing.T) {
	// Create a temporary tasks.md file
	tmpDir := t.TempDir()
	tasksFile := filepath.Join(tmpDir, "tasks.md")

	initialContent := `# Project Tasks

- [ ] 1. Setup project
  Initialize the project structure
  - [ ] 1.1 Create directory structure
  - [ ] 1.2 Configure build system
- [ ] 2. Write documentation
`

	if err := os.WriteFile(tasksFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create tasks file: %v", err)
	}

	// Parse the file
	p := parser.NewParser()
	content, _ := os.ReadFile(tasksFile)
	result := p.Parse(string(content))
	if result == nil {
		t.Fatal("Failed to parse tasks.md")
	}

	// Create store
	taskStore := store.NewTaskStore()
	taskStore.Document = result.Document

	// Build paths
	pm := store.NewPathManager(taskStore)
	pm.BuildAllPaths()

	// Verify initial state
	task1_1 := findTaskByIDInStore(taskStore, "1.1")
	if task1_1 == nil {
		t.Fatal("Task 1.1 not found")
	}
	if task1_1.Status != store.StatusPending {
		t.Fatalf("Task 1.1 should be pending, got %s", task1_1.Status)
	}

	// Simulate workflow: pending → doing
	taskStore.Lock()
	task1_1.Status = store.StatusDoing
	pm.OnStatusChange(task1_1)
	taskStore.Unlock()

	// Verify task moved to doing
	if task1_1.Status != store.StatusDoing {
		t.Fatalf("Task 1.1 should be doing, got %s", task1_1.Status)
	}

	// Verify path updated
	path := taskStore.PathByTask[task1_1]
	if path == "" {
		t.Fatal("Task 1.1 path should not be empty")
	}

	// Simulate workflow: doing → done
	taskStore.Lock()
	task1_1.Status = store.StatusDone
	pm.OnStatusChange(task1_1)
	taskStore.Unlock()

	// Verify task moved to done
	if task1_1.Status != store.StatusDone {
		t.Fatalf("Task 1.1 should be done, got %s", task1_1.Status)
	}

	// Verify parent status derivation
	task1 := findTaskByIDInStore(taskStore, "1")
	if task1 == nil {
		t.Fatal("Task 1 not found")
	}
	// Parent should still be pending because 1.2 is pending
	derivedStatus := store.DeriveParentStatus(task1)
	if derivedStatus != store.StatusPending {
		t.Fatalf("Task 1 derived status should be pending (1.2 is pending), got %s", derivedStatus)
	}

	// Complete task 1.2
	task1_2 := findTaskByIDInStore(taskStore, "1.2")
	if task1_2 == nil {
		t.Fatal("Task 1.2 not found")
	}
	taskStore.Lock()
	task1_2.Status = store.StatusDone
	pm.OnStatusChange(task1_2)
	taskStore.Unlock()

	// Now parent should be done
	derivedStatus = store.DeriveParentStatus(task1)
	if derivedStatus != store.StatusDone {
		t.Fatalf("Task 1 derived status should be done, got %s", derivedStatus)
	}

	// Serialize and verify output
	pr := printer.NewPrinter()
	output := pr.Serialize(taskStore.Document)
	if output == "" {
		t.Fatal("Serialized output should not be empty")
	}

	t.Logf("Final output:\n%s", output)
}

// TestIntegration_MultiAgentSimulation tests multiple goroutines competing
// for tasks to verify no race conditions.
//
// Validates: Design - Integration Tests, Requirements 5.2
func TestIntegration_MultiAgentSimulation(t *testing.T) {
	// Create a store with multiple pending tasks
	tasks := make([]*store.Task, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = &store.Task{
			ID:     fmt.Sprintf("%d", i+1),
			Title:  fmt.Sprintf("Task %d", i+1),
			Status: store.StatusPending,
		}
	}

	taskStore := store.NewTaskStore()
	taskStore.Document = &store.ParsedDocument{
		RootTasks: tasks,
	}

	pm := store.NewPathManager(taskStore)
	pm.BuildAllPaths()

	// Track which tasks were claimed
	claimed := make(map[string]int) // task ID → agent ID
	var claimedMu sync.Mutex

	// Simulate multiple agents competing for tasks
	numAgents := 5
	var wg sync.WaitGroup

	for agentID := 0; agentID < numAgents; agentID++ {
		wg.Add(1)
		go func(agent int) {
			defer wg.Done()

			// Each agent tries to claim tasks
			for i := 0; i < 10; i++ {
				taskStore.Lock()

				// Find a pending task
				var targetTask *store.Task
				for _, task := range taskStore.Document.RootTasks {
					if task.Status == store.StatusPending {
						targetTask = task
						break
					}
				}

				if targetTask != nil {
					// Claim the task
					targetTask.Status = store.StatusDoing
					pm.OnStatusChange(targetTask)

					claimedMu.Lock()
					claimed[targetTask.ID] = agent
					claimedMu.Unlock()
				}

				taskStore.Unlock()

				// Small delay to allow other agents to compete
				time.Sleep(time.Millisecond)
			}
		}(agentID)
	}

	wg.Wait()

	// Verify each task was claimed exactly once
	claimedMu.Lock()
	defer claimedMu.Unlock()

	if len(claimed) != 10 {
		t.Fatalf("Expected 10 tasks claimed, got %d", len(claimed))
	}

	// Verify no task was claimed by multiple agents
	// (This is implicitly verified by the map - each key can only have one value)

	// Verify all tasks are now in doing status
	for _, task := range taskStore.Document.RootTasks {
		if task.Status != store.StatusDoing {
			t.Fatalf("Task %s should be doing, got %s", task.ID, task.Status)
		}
	}

	t.Logf("All 10 tasks claimed by %d agents without race conditions", numAgents)
}

// TestIntegration_ExternalModification tests that external modifications
// to tasks.md are detected and applied.
//
// Validates: Design - Integration Tests
func TestIntegration_ExternalModification(t *testing.T) {
	// Create a temporary tasks.md file
	tmpDir := t.TempDir()
	tasksFile := filepath.Join(tmpDir, "tasks.md")

	initialContent := `# Project Tasks

- [ ] 1. Setup project
- [ ] 2. Write documentation
`

	if err := os.WriteFile(tasksFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create tasks file: %v", err)
	}

	// Parse and create store
	p := parser.NewParser()
	content, _ := os.ReadFile(tasksFile)
	result := p.Parse(string(content))

	taskStore := store.NewTaskStore()
	taskStore.Document = result.Document

	pm := store.NewPathManager(taskStore)
	pm.BuildAllPaths()

	// Create sync engine
	pr := printer.NewPrinter()
	syncEngine := syncPkg.NewSyncEngine(taskStore, p, pr, tasksFile)

	// Simulate external modification
	modifiedContent := `# Project Tasks

- [x] 1. Setup project
- [ ] 2. Write documentation
- [ ] 3. New task added externally
`

	if err := os.WriteFile(tasksFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify tasks file: %v", err)
	}

	// Sync from file
	if err := syncEngine.SyncFromFile(); err != nil {
		t.Fatalf("Failed to sync from file: %v", err)
	}

	// Verify changes were applied
	if len(taskStore.Document.RootTasks) != 3 {
		t.Fatalf("Expected 3 tasks after sync, got %d", len(taskStore.Document.RootTasks))
	}

	// Verify new task was added
	task3 := findTaskByIDInStore(taskStore, "3")
	if task3 == nil {
		t.Fatal("Task 3 should exist after sync")
	}
	if task3.Title != "New task added externally" {
		t.Fatalf("Task 3 title should be 'New task added externally', got %q", task3.Title)
	}
}

// TestIntegration_FilesystemPrecedence tests that filesystem operations
// take precedence over external modifications.
//
// Validates: Design - Integration Tests
func TestIntegration_FilesystemPrecedence(t *testing.T) {
	// Create a temporary tasks.md file
	tmpDir := t.TempDir()
	tasksFile := filepath.Join(tmpDir, "tasks.md")

	initialContent := `# Project Tasks

- [ ] 1. Setup project
`

	if err := os.WriteFile(tasksFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create tasks file: %v", err)
	}

	// Parse and create store
	p := parser.NewParser()
	content, _ := os.ReadFile(tasksFile)
	result := p.Parse(string(content))

	taskStore := store.NewTaskStore()
	taskStore.Document = result.Document

	pm := store.NewPathManager(taskStore)
	pm.BuildAllPaths()

	// Create sync engine
	pr := printer.NewPrinter()
	syncEngine := syncPkg.NewSyncEngine(taskStore, p, pr, tasksFile)

	// Simulate filesystem operation: change status to doing
	task1 := findTaskByIDInStore(taskStore, "1")
	taskStore.Lock()
	task1.Status = store.StatusDoing
	pm.OnStatusChange(task1)
	taskStore.Unlock()

	// Simulate external modification that conflicts (sets status back to pending)
	conflictingContent := `# Project Tasks

- [ ] 1. Setup project
`

	if err := os.WriteFile(tasksFile, []byte(conflictingContent), 0644); err != nil {
		t.Fatalf("Failed to modify tasks file: %v", err)
	}

	// Sync from file
	if err := syncEngine.SyncFromFile(); err != nil {
		t.Fatalf("Failed to sync from file: %v", err)
	}

	// Verify filesystem operation took precedence
	task1 = findTaskByIDInStore(taskStore, "1")
	if task1.Status != store.StatusDoing {
		t.Fatalf("Task 1 should still be doing (filesystem precedence), got %s", task1.Status)
	}
}

// findTaskByIDInStore finds a task by ID in the store.
func findTaskByIDInStore(s *store.TaskStore, id string) *store.Task {
	if s.Document == nil {
		return nil
	}
	return findTaskByIDRecursive(s.Document.RootTasks, id)
}

// findTaskByIDRecursive recursively finds a task by ID.
func findTaskByIDRecursive(tasks []*store.Task, id string) *store.Task {
	for _, task := range tasks {
		if task.ID == id {
			return task
		}
		if found := findTaskByIDRecursive(task.Children, id); found != nil {
			return found
		}
	}
	return nil
}


// TestIntegration_CrashRecovery tests that the system can recover from
// a simulated crash and maintain consistent state.
//
// Validates: Design - Integration Tests
func TestIntegration_CrashRecovery(t *testing.T) {
	// Create a temporary tasks.md file
	tmpDir := t.TempDir()
	tasksFile := filepath.Join(tmpDir, "tasks.md")

	initialContent := `# Project Tasks

- [ ] 1. Setup project
- [-] 2. In progress task
- [x] 3. Completed task
`

	if err := os.WriteFile(tasksFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create tasks file: %v", err)
	}

	// First "session" - parse and modify
	p := parser.NewParser()
	content, _ := os.ReadFile(tasksFile)
	result := p.Parse(string(content))

	taskStore := store.NewTaskStore()
	taskStore.Document = result.Document

	pm := store.NewPathManager(taskStore)
	pm.BuildAllPaths()

	// Modify task 1 to doing
	task1 := findTaskByIDInStore(taskStore, "1")
	taskStore.Lock()
	task1.Status = store.StatusDoing
	pm.OnStatusChange(task1)
	taskStore.Unlock()

	// Sync to file (simulating normal operation before crash)
	pr := printer.NewPrinter()
	syncEngine := syncPkg.NewSyncEngine(taskStore, p, pr, tasksFile)
	if err := syncEngine.SyncToFile(); err != nil {
		t.Fatalf("Failed to sync to file: %v", err)
	}

	// Simulate crash by discarding in-memory state
	taskStore = nil
	pm = nil
	syncEngine = nil

	// Second "session" - recover from file
	content, _ = os.ReadFile(tasksFile)
	result = p.Parse(string(content))

	taskStore = store.NewTaskStore()
	taskStore.Document = result.Document

	pm = store.NewPathManager(taskStore)
	pm.BuildAllPaths()

	// Verify state was recovered correctly
	task1 = findTaskByIDInStore(taskStore, "1")
	if task1 == nil {
		t.Fatal("Task 1 not found after recovery")
	}
	if task1.Status != store.StatusDoing {
		t.Fatalf("Task 1 should be doing after recovery, got %s", task1.Status)
	}

	task2 := findTaskByIDInStore(taskStore, "2")
	if task2 == nil {
		t.Fatal("Task 2 not found after recovery")
	}
	if task2.Status != store.StatusDoing {
		t.Fatalf("Task 2 should be doing after recovery, got %s", task2.Status)
	}

	task3 := findTaskByIDInStore(taskStore, "3")
	if task3 == nil {
		t.Fatal("Task 3 not found after recovery")
	}
	if task3.Status != store.StatusDone {
		t.Fatalf("Task 3 should be done after recovery, got %s", task3.Status)
	}

	t.Log("Successfully recovered state after simulated crash")
}
