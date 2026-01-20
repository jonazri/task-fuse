package store

import (
	"strings"
	"testing"
)

// Helper function to create a simple task for testing
func createTestTask(id, title string, status TaskStatus) *Task {
	return &Task{
		ID:     id,
		Title:  title,
		Status: status,
	}
}

// Helper function to create a task with children
func createTestTaskWithChildren(id, title string, status TaskStatus, children []*Task) *Task {
	task := &Task{
		ID:       id,
		Title:    title,
		Status:   status,
		Children: children,
	}
	// Set parent references
	for _, child := range children {
		child.Parent = task
	}
	return task
}

// TestNewPathManager tests the PathManager constructor.
func TestNewPathManager(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	if pm == nil {
		t.Fatal("NewPathManager returned nil")
	}
	if pm.store != store {
		t.Error("PathManager store reference is incorrect")
	}
}

// TestNewPathManager_NilStore tests PathManager with nil store.
func TestNewPathManager_NilStore(t *testing.T) {
	pm := NewPathManager(nil)

	if pm == nil {
		t.Fatal("NewPathManager returned nil")
	}
	if pm.store != nil {
		t.Error("PathManager store should be nil")
	}
}

// TestDeriveStatus_LeafTask tests status derivation for leaf tasks.
func TestDeriveStatus_LeafTask(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	tests := []struct {
		name     string
		status   TaskStatus
		expected TaskStatus
	}{
		{"pending leaf", StatusPending, StatusPending},
		{"queued leaf", StatusQueued, StatusQueued},
		{"doing leaf", StatusDoing, StatusDoing},
		{"done leaf", StatusDone, StatusDone},
		{"failed leaf", StatusFailed, StatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := createTestTask("1.1", "Test Task", tt.status)
			result := pm.DeriveStatus(task)
			if result != tt.expected {
				t.Errorf("DeriveStatus() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestDeriveStatus_NilTask tests status derivation for nil task.
func TestDeriveStatus_NilTask(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	result := pm.DeriveStatus(nil)
	if result != StatusPending {
		t.Errorf("DeriveStatus(nil) = %q, want %q", result, StatusPending)
	}
}

// TestDeriveStatus_ParentWithDoingChild tests that doing takes highest precedence.
func TestDeriveStatus_ParentWithDoingChild(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Parent with one doing child among others
	children := []*Task{
		createTestTask("1.1", "Done Task", StatusDone),
		createTestTask("1.2", "Doing Task", StatusDoing),
		createTestTask("1.3", "Pending Task", StatusPending),
	}
	parent := createTestTaskWithChildren("1", "Parent Task", StatusPending, children)

	result := pm.DeriveStatus(parent)
	if result != StatusDoing {
		t.Errorf("DeriveStatus() = %q, want %q (doing has highest precedence)", result, StatusDoing)
	}
}

// TestDeriveStatus_ParentWithFailedChild tests that failed has second precedence.
func TestDeriveStatus_ParentWithFailedChild(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Parent with failed child but no doing child
	children := []*Task{
		createTestTask("1.1", "Done Task", StatusDone),
		createTestTask("1.2", "Failed Task", StatusFailed),
		createTestTask("1.3", "Pending Task", StatusPending),
	}
	parent := createTestTaskWithChildren("1", "Parent Task", StatusPending, children)

	result := pm.DeriveStatus(parent)
	if result != StatusFailed {
		t.Errorf("DeriveStatus() = %q, want %q (failed has second precedence)", result, StatusFailed)
	}
}

// TestDeriveStatus_ParentWithPendingChild tests that pending has third precedence.
func TestDeriveStatus_ParentWithPendingChild(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Parent with pending child but no doing or failed
	children := []*Task{
		createTestTask("1.1", "Done Task", StatusDone),
		createTestTask("1.2", "Pending Task", StatusPending),
		createTestTask("1.3", "Queued Task", StatusQueued),
	}
	parent := createTestTaskWithChildren("1", "Parent Task", StatusPending, children)

	result := pm.DeriveStatus(parent)
	if result != StatusPending {
		t.Errorf("DeriveStatus() = %q, want %q (pending has third precedence)", result, StatusPending)
	}
}

// TestDeriveStatus_ParentWithQueuedChild tests that queued has fourth precedence.
func TestDeriveStatus_ParentWithQueuedChild(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Parent with queued child but no doing, failed, or pending
	children := []*Task{
		createTestTask("1.1", "Done Task", StatusDone),
		createTestTask("1.2", "Queued Task", StatusQueued),
	}
	parent := createTestTaskWithChildren("1", "Parent Task", StatusPending, children)

	result := pm.DeriveStatus(parent)
	if result != StatusQueued {
		t.Errorf("DeriveStatus() = %q, want %q (queued has fourth precedence)", result, StatusQueued)
	}
}

// TestDeriveStatus_ParentAllDone tests that parent is done when all children done.
func TestDeriveStatus_ParentAllDone(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Parent with all done children
	children := []*Task{
		createTestTask("1.1", "Done Task 1", StatusDone),
		createTestTask("1.2", "Done Task 2", StatusDone),
		createTestTask("1.3", "Done Task 3", StatusDone),
	}
	parent := createTestTaskWithChildren("1", "Parent Task", StatusPending, children)

	result := pm.DeriveStatus(parent)
	if result != StatusDone {
		t.Errorf("DeriveStatus() = %q, want %q (all children done)", result, StatusDone)
	}
}

// TestDeriveStatus_NestedHierarchy tests status derivation with nested tasks.
func TestDeriveStatus_NestedHierarchy(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Create nested hierarchy:
	// 1. Parent (derived)
	//   1.1 Child (derived)
	//     1.1.1 Grandchild (doing)
	//     1.1.2 Grandchild (done)
	grandchildren := []*Task{
		createTestTask("1.1.1", "Grandchild Doing", StatusDoing),
		createTestTask("1.1.2", "Grandchild Done", StatusDone),
	}
	child := createTestTaskWithChildren("1.1", "Child", StatusPending, grandchildren)
	parent := createTestTaskWithChildren("1", "Parent", StatusPending, []*Task{child})

	// Parent should derive to doing because grandchild is doing
	result := pm.DeriveStatus(parent)
	if result != StatusDoing {
		t.Errorf("DeriveStatus() = %q, want %q (nested doing)", result, StatusDoing)
	}
}

// TestRebuildPaths_SingleLeafTask tests path building for a single leaf task.
func TestRebuildPaths_SingleLeafTask(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	task := createTestTask("1.1", "Test Task", StatusPending)
	store.Document = &ParsedDocument{RootTasks: []*Task{task}}

	pm.RebuildPaths(task, StatusPending, "/pending")

	// Check path was set correctly
	expectedPath := "/pending/1.1.test_task.md"
	if store.PathByTask[task] != expectedPath {
		t.Errorf("PathByTask = %q, want %q", store.PathByTask[task], expectedPath)
	}

	// Check TasksByPath index
	entry := store.TasksByPath[expectedPath]
	if entry == nil {
		t.Fatal("TasksByPath entry is nil")
	}
	if entry.Task != task {
		t.Error("TasksByPath entry has wrong task")
	}
	if entry.Status != StatusPending {
		t.Errorf("TasksByPath entry status = %q, want %q", entry.Status, StatusPending)
	}
	if entry.FullPath != expectedPath {
		t.Errorf("TasksByPath entry path = %q, want %q", entry.FullPath, expectedPath)
	}

	// Check TasksByID index
	idEntry := store.TasksByID["1.1"]
	if idEntry == nil {
		t.Fatal("TasksByID entry is nil")
	}
	if idEntry.Task != task {
		t.Error("TasksByID entry has wrong task")
	}
}

// TestRebuildPaths_ParentWithChildren tests path building for parent with children.
func TestRebuildPaths_ParentWithChildren(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	children := []*Task{
		createTestTask("1.1", "Child One", StatusDone),
		createTestTask("1.2", "Child Two", StatusPending),
	}
	parent := createTestTaskWithChildren("1", "Parent Task", StatusPending, children)
	store.Document = &ParsedDocument{RootTasks: []*Task{parent}}

	// Parent derives to pending (has pending child)
	pm.RebuildPaths(parent, StatusPending, "/pending")

	// Check parent path
	parentPath := "/pending/1.parent_task"
	if store.PathByTask[parent] != parentPath {
		t.Errorf("Parent PathByTask = %q, want %q", store.PathByTask[parent], parentPath)
	}

	// Check children paths (under parent directory)
	child1Path := parentPath + "/1.1.child_one.md"
	if store.PathByTask[children[0]] != child1Path {
		t.Errorf("Child1 PathByTask = %q, want %q", store.PathByTask[children[0]], child1Path)
	}

	child2Path := parentPath + "/1.2.child_two.md"
	if store.PathByTask[children[1]] != child2Path {
		t.Errorf("Child2 PathByTask = %q, want %q", store.PathByTask[children[1]], child2Path)
	}
}

// TestRebuildPaths_NilTask tests that nil task is handled gracefully.
func TestRebuildPaths_NilTask(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Should not panic
	pm.RebuildPaths(nil, StatusPending, "/pending")
}

// TestRebuildPaths_UpdatesOldPath tests that old paths are removed when rebuilding.
func TestRebuildPaths_UpdatesOldPath(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	task := createTestTask("1.1", "Test Task", StatusPending)
	store.Document = &ParsedDocument{RootTasks: []*Task{task}}

	// First build with pending status
	pm.RebuildPaths(task, StatusPending, "/pending")
	oldPath := "/pending/1.1.test_task.md"

	// Verify old path exists
	if store.TasksByPath[oldPath] == nil {
		t.Fatal("Old path should exist")
	}

	// Rebuild with doing status
	pm.RebuildPaths(task, StatusDoing, "/doing")
	newPath := "/doing/1.1.test_task.md"

	// Old path should be removed
	if store.TasksByPath[oldPath] != nil {
		t.Error("Old path should be removed")
	}

	// New path should exist
	if store.TasksByPath[newPath] == nil {
		t.Error("New path should exist")
	}

	// PathByTask should point to new path
	if store.PathByTask[task] != newPath {
		t.Errorf("PathByTask = %q, want %q", store.PathByTask[task], newPath)
	}
}

// TestRebuildPaths_DeepNesting tests path building for deeply nested tasks.
func TestRebuildPaths_DeepNesting(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Create 3-level hierarchy
	grandchild := createTestTask("1.1.1", "Grandchild", StatusDoing)
	child := createTestTaskWithChildren("1.1", "Child", StatusPending, []*Task{grandchild})
	parent := createTestTaskWithChildren("1", "Parent", StatusPending, []*Task{child})
	store.Document = &ParsedDocument{RootTasks: []*Task{parent}}

	// Parent derives to doing (grandchild is doing)
	pm.RebuildPaths(parent, StatusDoing, "/doing")

	// Check all paths
	parentPath := "/doing/1.parent"
	childPath := parentPath + "/1.1.child"
	grandchildPath := childPath + "/1.1.1.grandchild.md"

	if store.PathByTask[parent] != parentPath {
		t.Errorf("Parent path = %q, want %q", store.PathByTask[parent], parentPath)
	}
	if store.PathByTask[child] != childPath {
		t.Errorf("Child path = %q, want %q", store.PathByTask[child], childPath)
	}
	if store.PathByTask[grandchild] != grandchildPath {
		t.Errorf("Grandchild path = %q, want %q", store.PathByTask[grandchild], grandchildPath)
	}
}

// TestOnStatusChange_FindsRootAndRebuilds tests that OnStatusChange finds root.
func TestOnStatusChange_FindsRootAndRebuilds(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Create hierarchy
	child := createTestTask("1.1", "Child", StatusDoing)
	parent := createTestTaskWithChildren("1", "Parent", StatusPending, []*Task{child})
	store.Document = &ParsedDocument{RootTasks: []*Task{parent}}

	// Call OnStatusChange on child
	pm.OnStatusChange(child)

	// Parent should be in doing directory (derived from child)
	parentPath := store.PathByTask[parent]
	if !strings.HasPrefix(parentPath, "/doing/") {
		t.Errorf("Parent path = %q, should start with /doing/", parentPath)
	}

	// Child should be under parent
	childPath := store.PathByTask[child]
	if !strings.HasPrefix(childPath, parentPath+"/") {
		t.Errorf("Child path = %q, should be under parent %q", childPath, parentPath)
	}
}

// TestOnStatusChange_NilTask tests that nil task is handled gracefully.
func TestOnStatusChange_NilTask(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Should not panic
	pm.OnStatusChange(nil)
}

// TestOnStatusChange_RootTask tests OnStatusChange on a root task.
func TestOnStatusChange_RootTask(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	task := createTestTask("1", "Root Task", StatusDone)
	store.Document = &ParsedDocument{RootTasks: []*Task{task}}

	pm.OnStatusChange(task)

	// Task should be in done directory
	path := store.PathByTask[task]
	if !strings.HasPrefix(path, "/done/") {
		t.Errorf("Path = %q, should start with /done/", path)
	}
}

// TestOnStatusChange_StatusTransition tests path changes on status transition.
func TestOnStatusChange_StatusTransition(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	task := createTestTask("1", "Task", StatusPending)
	store.Document = &ParsedDocument{RootTasks: []*Task{task}}

	// Initial build
	pm.OnStatusChange(task)
	if !strings.HasPrefix(store.PathByTask[task], "/pending/") {
		t.Errorf("Initial path should be in /pending/")
	}

	// Change status to doing
	task.Status = StatusDoing
	pm.OnStatusChange(task)
	if !strings.HasPrefix(store.PathByTask[task], "/doing/") {
		t.Errorf("After status change, path should be in /doing/")
	}

	// Change status to done
	task.Status = StatusDone
	pm.OnStatusChange(task)
	if !strings.HasPrefix(store.PathByTask[task], "/done/") {
		t.Errorf("After status change, path should be in /done/")
	}
}

// TestBuildAllPaths_EmptyDocument tests BuildAllPaths with empty document.
func TestBuildAllPaths_EmptyDocument(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	store.Document = &ParsedDocument{RootTasks: []*Task{}}

	// Should not panic
	pm.BuildAllPaths()

	// Maps should be empty
	if len(store.TasksByPath) != 0 {
		t.Errorf("TasksByPath should be empty, got %d entries", len(store.TasksByPath))
	}
}

// TestBuildAllPaths_NilDocument tests BuildAllPaths with nil document.
func TestBuildAllPaths_NilDocument(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Should not panic
	pm.BuildAllPaths()
}

// TestBuildAllPaths_MultipleRootTasks tests BuildAllPaths with multiple roots.
func TestBuildAllPaths_MultipleRootTasks(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	task1 := createTestTask("1", "Task One", StatusPending)
	task2 := createTestTask("2", "Task Two", StatusDoing)
	task3 := createTestTask("3", "Task Three", StatusDone)
	store.Document = &ParsedDocument{RootTasks: []*Task{task1, task2, task3}}

	pm.BuildAllPaths()

	// Check each task is in correct status directory
	if !strings.HasPrefix(store.PathByTask[task1], "/pending/") {
		t.Errorf("Task1 path = %q, should be in /pending/", store.PathByTask[task1])
	}
	if !strings.HasPrefix(store.PathByTask[task2], "/doing/") {
		t.Errorf("Task2 path = %q, should be in /doing/", store.PathByTask[task2])
	}
	if !strings.HasPrefix(store.PathByTask[task3], "/done/") {
		t.Errorf("Task3 path = %q, should be in /done/", store.PathByTask[task3])
	}

	// Check all indexes are populated
	if len(store.TasksByPath) != 3 {
		t.Errorf("TasksByPath should have 3 entries, got %d", len(store.TasksByPath))
	}
	if len(store.TasksByID) != 3 {
		t.Errorf("TasksByID should have 3 entries, got %d", len(store.TasksByID))
	}
	if len(store.PathByTask) != 3 {
		t.Errorf("PathByTask should have 3 entries, got %d", len(store.PathByTask))
	}
}

// TestBuildAllPaths_ClearsExistingIndexes tests that BuildAllPaths clears old data.
func TestBuildAllPaths_ClearsExistingIndexes(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Add some existing data
	oldTask := createTestTask("99", "Old Task", StatusPending)
	store.TasksByPath["/old/path"] = &PathEntry{Task: oldTask}
	store.TasksByID["99"] = &PathEntry{Task: oldTask}
	store.PathByTask[oldTask] = "/old/path"

	// Set up new document
	newTask := createTestTask("1", "New Task", StatusDoing)
	store.Document = &ParsedDocument{RootTasks: []*Task{newTask}}

	pm.BuildAllPaths()

	// Old data should be gone
	if store.TasksByPath["/old/path"] != nil {
		t.Error("Old path should be cleared")
	}
	if store.TasksByID["99"] != nil {
		t.Error("Old ID should be cleared")
	}
	if store.PathByTask[oldTask] != "" {
		t.Error("Old task path should be cleared")
	}

	// New data should exist
	if store.TasksByID["1"] == nil {
		t.Error("New task should be indexed by ID")
	}
}

// TestBuildAllPaths_ComplexHierarchy tests BuildAllPaths with complex structure.
func TestBuildAllPaths_ComplexHierarchy(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Create complex hierarchy:
	// 1. Parent (pending - has pending child)
	//   1.1 Child Done
	//   1.2 Child Pending
	// 2. Parent (doing - has doing child)
	//   2.1 Child Doing
	child1_1 := createTestTask("1.1", "Child Done", StatusDone)
	child1_2 := createTestTask("1.2", "Child Pending", StatusPending)
	parent1 := createTestTaskWithChildren("1", "Parent One", StatusPending, []*Task{child1_1, child1_2})

	child2_1 := createTestTask("2.1", "Child Doing", StatusDoing)
	parent2 := createTestTaskWithChildren("2", "Parent Two", StatusPending, []*Task{child2_1})

	store.Document = &ParsedDocument{RootTasks: []*Task{parent1, parent2}}

	pm.BuildAllPaths()

	// Parent1 should be in pending (has pending child)
	if !strings.HasPrefix(store.PathByTask[parent1], "/pending/") {
		t.Errorf("Parent1 path = %q, should be in /pending/", store.PathByTask[parent1])
	}

	// Parent2 should be in doing (has doing child)
	if !strings.HasPrefix(store.PathByTask[parent2], "/doing/") {
		t.Errorf("Parent2 path = %q, should be in /doing/", store.PathByTask[parent2])
	}

	// All 5 tasks should be indexed
	if len(store.TasksByPath) != 5 {
		t.Errorf("TasksByPath should have 5 entries, got %d", len(store.TasksByPath))
	}
}

// TestRebuildPaths_CollisionResolution tests that filename collisions are resolved.
// Note: In practice, collisions are rare because task IDs are unique and part of the filename.
// This test verifies the collision resolution mechanism works when filenames would otherwise collide.
func TestRebuildPaths_CollisionResolution(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Create two tasks with same title but different IDs
	// Since IDs are part of filename, they won't actually collide
	task1 := createTestTask("1.1", "Same Title", StatusPending)
	task2 := createTestTask("1.2", "Same Title", StatusPending)
	parent := createTestTaskWithChildren("1", "Parent", StatusPending, []*Task{task1, task2})
	store.Document = &ParsedDocument{RootTasks: []*Task{parent}}

	pm.BuildAllPaths()

	// Both tasks should have unique paths (different IDs = different filenames)
	path1 := store.PathByTask[task1]
	path2 := store.PathByTask[task2]

	if path1 == path2 {
		t.Errorf("Tasks should have different paths: %q == %q", path1, path2)
	}

	// Verify both paths contain the expected ID prefixes
	if !strings.Contains(path1, "1.1.same_title.md") {
		t.Errorf("Task1 path = %q, should contain '1.1.same_title.md'", path1)
	}
	if !strings.Contains(path2, "1.2.same_title.md") {
		t.Errorf("Task2 path = %q, should contain '1.2.same_title.md'", path2)
	}
}

// TestDeriveStatus_Precedence tests the full precedence order.
func TestDeriveStatus_Precedence(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	tests := []struct {
		name           string
		childStatuses  []TaskStatus
		expectedStatus TaskStatus
	}{
		{
			name:           "doing beats all",
			childStatuses:  []TaskStatus{StatusDone, StatusFailed, StatusPending, StatusQueued, StatusDoing},
			expectedStatus: StatusDoing,
		},
		{
			name:           "failed beats pending/queued/done",
			childStatuses:  []TaskStatus{StatusDone, StatusPending, StatusQueued, StatusFailed},
			expectedStatus: StatusFailed,
		},
		{
			name:           "pending beats queued/done",
			childStatuses:  []TaskStatus{StatusDone, StatusQueued, StatusPending},
			expectedStatus: StatusPending,
		},
		{
			name:           "queued beats done",
			childStatuses:  []TaskStatus{StatusDone, StatusQueued},
			expectedStatus: StatusQueued,
		},
		{
			name:           "all done means done",
			childStatuses:  []TaskStatus{StatusDone, StatusDone, StatusDone},
			expectedStatus: StatusDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var children []*Task
			for i, status := range tt.childStatuses {
				children = append(children, createTestTask(
					"1."+itoa(i+1),
					"Child",
					status,
				))
			}
			parent := createTestTaskWithChildren("1", "Parent", StatusPending, children)

			result := pm.DeriveStatus(parent)
			if result != tt.expectedStatus {
				t.Errorf("DeriveStatus() = %q, want %q", result, tt.expectedStatus)
			}
		})
	}
}

// TestOnStatusChange_ChildStatusAffectsParentPath tests that child status change
// causes parent to move to correct status directory.
func TestOnStatusChange_ChildStatusAffectsParentPath(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Create parent with pending child
	child := createTestTask("1.1", "Child", StatusPending)
	parent := createTestTaskWithChildren("1", "Parent", StatusPending, []*Task{child})
	store.Document = &ParsedDocument{RootTasks: []*Task{parent}}

	// Initial build - parent should be in pending
	pm.BuildAllPaths()
	if !strings.HasPrefix(store.PathByTask[parent], "/pending/") {
		t.Errorf("Initial parent path should be in /pending/")
	}

	// Change child to doing
	child.Status = StatusDoing
	pm.OnStatusChange(child)

	// Parent should now be in doing
	if !strings.HasPrefix(store.PathByTask[parent], "/doing/") {
		t.Errorf("After child doing, parent path = %q, should be in /doing/", store.PathByTask[parent])
	}

	// Change child to done
	child.Status = StatusDone
	pm.OnStatusChange(child)

	// Parent should now be in done (only child is done)
	if !strings.HasPrefix(store.PathByTask[parent], "/done/") {
		t.Errorf("After child done, parent path = %q, should be in /done/", store.PathByTask[parent])
	}
}

// TestPathManager_NilStore tests PathManager methods with nil store.
func TestPathManager_NilStore(t *testing.T) {
	pm := NewPathManager(nil)

	task := createTestTask("1", "Task", StatusPending)

	// These should not panic
	pm.RebuildPaths(task, StatusPending, "/pending")
	pm.OnStatusChange(task)
	pm.BuildAllPaths()

	// DeriveStatus should still work
	result := pm.DeriveStatus(task)
	if result != StatusPending {
		t.Errorf("DeriveStatus() = %q, want %q", result, StatusPending)
	}
}

// TestRebuildPaths_ChildrenStayWithParent tests that children stay under parent
// directory regardless of their individual status.
func TestRebuildPaths_ChildrenStayWithParent(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	// Create parent with children of different statuses
	children := []*Task{
		createTestTask("1.1", "Done Child", StatusDone),
		createTestTask("1.2", "Doing Child", StatusDoing),
		createTestTask("1.3", "Pending Child", StatusPending),
	}
	parent := createTestTaskWithChildren("1", "Parent", StatusPending, children)
	store.Document = &ParsedDocument{RootTasks: []*Task{parent}}

	pm.BuildAllPaths()

	// Parent should be in doing (has doing child)
	parentPath := store.PathByTask[parent]
	if !strings.HasPrefix(parentPath, "/doing/") {
		t.Errorf("Parent path = %q, should be in /doing/", parentPath)
	}

	// ALL children should be under parent, not in their own status directories
	for _, child := range children {
		childPath := store.PathByTask[child]
		if !strings.HasPrefix(childPath, parentPath+"/") {
			t.Errorf("Child %s path = %q, should be under parent %q", child.ID, childPath, parentPath)
		}
	}
}

// TestBuildAllPaths_IndexConsistency tests that all indexes are consistent.
func TestBuildAllPaths_IndexConsistency(t *testing.T) {
	store := NewTaskStore()
	pm := NewPathManager(store)

	task := createTestTask("1", "Task", StatusPending)
	store.Document = &ParsedDocument{RootTasks: []*Task{task}}

	pm.BuildAllPaths()

	// Get path from PathByTask
	path := store.PathByTask[task]
	if path == "" {
		t.Fatal("PathByTask should have entry")
	}

	// TasksByPath should have same entry
	entry := store.TasksByPath[path]
	if entry == nil {
		t.Fatal("TasksByPath should have entry for path")
	}
	if entry.Task != task {
		t.Error("TasksByPath entry should point to same task")
	}
	if entry.FullPath != path {
		t.Error("TasksByPath entry FullPath should match")
	}

	// TasksByID should have same entry
	idEntry := store.TasksByID[task.ID]
	if idEntry == nil {
		t.Fatal("TasksByID should have entry")
	}
	if idEntry != entry {
		t.Error("TasksByID and TasksByPath should point to same entry")
	}
}

// TestGenerateDirname tests the GenerateDirname function for parent tasks.
func TestGenerateDirname(t *testing.T) {
	tests := []struct {
		name     string
		task     *Task
		expected string
	}{
		{
			name: "simple parent task",
			task: &Task{
				ID:    "1",
				Title: "Setup Project",
			},
			expected: "1.setup_project",
		},
		{
			name: "parent with special chars",
			task: &Task{
				ID:    "1.2",
				Title: "Configure Build!!!",
			},
			expected: "1.2.configure_build",
		},
		{
			name: "parent with spaces",
			task: &Task{
				ID:    "2",
				Title: "   Trim   Spaces   ",
			},
			expected: "2.trim_spaces",
		},
		{
			name: "parent with empty title",
			task: &Task{
				ID:    "3",
				Title: "",
			},
			expected: "3.task",
		},
		{
			name:     "nil task",
			task:     nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateDirname(tt.task)
			if result != tt.expected {
				t.Errorf("GenerateDirname() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestGenerateDirname_NoMdExtension verifies directory names don't have .md extension.
func TestGenerateDirname_NoMdExtension(t *testing.T) {
	task := &Task{
		ID:    "1",
		Title: "Parent Task",
	}

	result := GenerateDirname(task)

	if strings.HasSuffix(result, ".md") {
		t.Errorf("GenerateDirname() = %q, should not end with .md", result)
	}
}

// TestGenerateFilename_HasMdExtension verifies file names have .md extension.
func TestGenerateFilename_HasMdExtension(t *testing.T) {
	task := &Task{
		ID:    "1.1",
		Title: "Leaf Task",
	}

	result := GenerateFilename(task)

	if !strings.HasSuffix(result, ".md") {
		t.Errorf("GenerateFilename() = %q, should end with .md", result)
	}
}
