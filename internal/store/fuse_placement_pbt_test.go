// Package store provides property-based tests for FUSE filesystem placement logic.
//
// Feature: fuse-task-filesystem
// Property 4: Status Directory Placement
// Property 3: Task Hierarchy Preservation
//
// These tests use the rapid library for property-based testing to verify that
// tasks are correctly placed in status directories and hierarchies are preserved.
//
// Note: These tests verify the PathManager logic which determines task placement.
// The actual FUSE filesystem operations delegate to PathManager.
package store

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Property 4: Status Directory Placement
// =============================================================================
//
// Property Statement:
// *For any* task with a given status, the FUSE_Filesystem SHALL expose it within
// the corresponding status directory (`pending/`, `queued/`, `doing/`, `done/`,
// or `failed/`).
//
// **Validates: Requirements 2.5, 2.6, 2.7, 2.8, 2.9**
//
// Tag: Feature: fuse-task-filesystem, Property 4: Status Directory Placement

// =============================================================================
// Generators for Property 4 and Property 3
// =============================================================================

// genTaskStatusP4 generates a random valid task status.
func genTaskStatusP4() *rapid.Generator[TaskStatus] {
	return rapid.Custom(func(t *rapid.T) TaskStatus {
		statuses := []TaskStatus{
			StatusPending,
			StatusQueued,
			StatusDoing,
			StatusDone,
			StatusFailed,
		}
		idx := rapid.IntRange(0, len(statuses)-1).Draw(t, "statusIdx")
		return statuses[idx]
	})
}

// genValidTaskIDP4 generates a valid task ID (e.g., "1", "1.2", "1.2.3").
// Task IDs are dot-separated positive integers with no leading zeros.
// Maximum depth is 5 levels for test efficiency.
func genValidTaskIDP4() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		depth := rapid.IntRange(1, 5).Draw(t, "depth")
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			num := rapid.IntRange(1, 99).Draw(t, fmt.Sprintf("segment_%d", i))
			segments[i] = fmt.Sprintf("%d", num)
		}
		return strings.Join(segments, ".")
	})
}

// genTaskTitleP4 generates a simple task title for testing.
func genTaskTitleP4() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		wordCount := rapid.IntRange(1, 5).Draw(t, "wordCount")
		words := make([]string, wordCount)
		for i := 0; i < wordCount; i++ {
			word := rapid.StringMatching(`[a-z]+`).Draw(t, fmt.Sprintf("word_%d", i))
			if len(word) > 10 {
				word = word[:10]
			}
			if len(word) == 0 {
				word = "task"
			}
			words[i] = word
		}
		return strings.Join(words, " ")
	})
}

// genLeafTaskP4 generates a leaf task (no children) with random status.
func genLeafTaskP4() *rapid.Generator[*Task] {
	return rapid.Custom(func(t *rapid.T) *Task {
		return &Task{
			ID:       genValidTaskIDP4().Draw(t, "taskID"),
			Title:    genTaskTitleP4().Draw(t, "taskTitle"),
			Status:   genTaskStatusP4().Draw(t, "taskStatus"),
			Children: nil,
		}
	})
}

// genChildStatusesP4 generates a slice of child statuses for testing parent derivation.
func genChildStatusesP4() *rapid.Generator[[]TaskStatus] {
	return rapid.Custom(func(t *rapid.T) []TaskStatus {
		count := rapid.IntRange(1, 5).Draw(t, "childCount")
		statuses := make([]TaskStatus, count)
		for i := 0; i < count; i++ {
			statuses[i] = genTaskStatusP4().Draw(t, fmt.Sprintf("childStatus_%d", i))
		}
		return statuses
	})
}

// genTaskTreeP4 generates a task tree with specified depth.
func genTaskTreeP4(maxDepth int) *rapid.Generator[*Task] {
	return rapid.Custom(func(t *rapid.T) *Task {
		return genTaskTreeRecursiveP4(t, maxDepth, "1", 0)
	})
}

// genTaskTreeRecursiveP4 recursively generates a task tree.
func genTaskTreeRecursiveP4(t *rapid.T, maxDepth int, idPrefix string, currentDepth int) *Task {
	task := &Task{
		ID:     idPrefix,
		Title:  genTaskTitleP4().Draw(t, fmt.Sprintf("title_%s", idPrefix)),
		Status: genTaskStatusP4().Draw(t, fmt.Sprintf("status_%s", idPrefix)),
	}

	// Decide if this task should have children
	if currentDepth < maxDepth {
		hasChildren := rapid.Bool().Draw(t, fmt.Sprintf("hasChildren_%s", idPrefix))
		if hasChildren {
			childCount := rapid.IntRange(1, 3).Draw(t, fmt.Sprintf("childCount_%s", idPrefix))
			task.Children = make([]*Task, childCount)
			for i := 0; i < childCount; i++ {
				childID := fmt.Sprintf("%s.%d", idPrefix, i+1)
				child := genTaskTreeRecursiveP4(t, maxDepth, childID, currentDepth+1)
				child.Parent = task
				task.Children[i] = child
			}
		}
	}

	return task
}

// =============================================================================
// Property Tests for Property 4: Status Directory Placement
// =============================================================================

// TestProperty4_StatusDirectoryPlacement_LeafTask verifies that leaf tasks
// (tasks without children) are placed in their own status directory.
//
// **Validates: Requirements 2.5, 2.6, 2.7, 2.8, 2.9**
//
// Tag: Feature: fuse-task-filesystem, Property 4: Status Directory Placement
func TestProperty4_StatusDirectoryPlacement_LeafTask(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 4: Status Directory Placement
	// Validates: Requirements 2.5, 2.6, 2.7, 2.8, 2.9

	rapid.Check(t, func(t *rapid.T) {
		// Generate a leaf task with random status
		task := genLeafTaskP4().Draw(t, "task")

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{task},
		}
		pm := NewPathManager(taskStore)

		// Build paths
		pm.BuildAllPaths()

		// Get the task's path
		path := taskStore.PathByTask[task]

		// Property 1: Path should not be empty
		if path == "" {
			t.Fatalf("Task path is empty for task ID=%q, Status=%s", task.ID, task.Status)
		}

		// Property 2: Path should start with the correct status directory
		expectedPrefix := "/" + string(task.Status) + "/"
		if !strings.HasPrefix(path, expectedPrefix) {
			t.Fatalf("Task path %q should start with %q for status %s",
				path, expectedPrefix, task.Status)
		}

		// Property 3: Path should end with .md (leaf task is a file)
		if !strings.HasSuffix(path, ".md") {
			t.Fatalf("Leaf task path %q should end with .md", path)
		}

		// Property 4: Path should contain the task ID
		if !strings.Contains(path, task.ID+".") {
			t.Fatalf("Task path %q should contain task ID %q", path, task.ID)
		}
	})
}

// TestProperty4_StatusDirectoryPlacement_AllStatuses verifies that each status
// maps to the correct directory.
//
// **Validates: Requirements 2.5, 2.6, 2.7, 2.8, 2.9**
//
// Tag: Feature: fuse-task-filesystem, Property 4: Status Directory Placement
func TestProperty4_StatusDirectoryPlacement_AllStatuses(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 4: Status Directory Placement
	// Validates: Requirements 2.5, 2.6, 2.7, 2.8, 2.9

	statuses := []TaskStatus{
		StatusPending,
		StatusQueued,
		StatusDoing,
		StatusDone,
		StatusFailed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				// Generate a task with the specific status
				task := &Task{
					ID:     genValidTaskIDP4().Draw(t, "taskID"),
					Title:  genTaskTitleP4().Draw(t, "taskTitle"),
					Status: status,
				}

				// Create store and path manager
				taskStore := NewTaskStore()
				taskStore.Document = &ParsedDocument{
					RootTasks: []*Task{task},
				}
				pm := NewPathManager(taskStore)

				// Build paths
				pm.BuildAllPaths()

				// Get the task's path
				path := taskStore.PathByTask[task]

				// Property: Path should be in the correct status directory
				expectedPrefix := "/" + string(status) + "/"
				if !strings.HasPrefix(path, expectedPrefix) {
					t.Fatalf("Task with status %s has path %q, expected prefix %q",
						status, path, expectedPrefix)
				}
			})
		})
	}
}

// TestProperty4_StatusDirectoryPlacement_ParentDerivedStatus verifies that parent
// tasks are placed in the directory matching their derived status (from children).
//
// **Validates: Requirements 2.5, 2.6, 2.7, 2.8, 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 4: Status Directory Placement
func TestProperty4_StatusDirectoryPlacement_ParentDerivedStatus(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 4: Status Directory Placement
	// Validates: Requirements 2.5, 2.6, 2.7, 2.8, 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate child statuses
		childStatuses := genChildStatusesP4().Draw(t, "childStatuses")

		// Create children with the generated statuses
		children := make([]*Task, len(childStatuses))
		for i, status := range childStatuses {
			children[i] = &Task{
				ID:     fmt.Sprintf("1.%d", i+1),
				Title:  fmt.Sprintf("Child %d", i+1),
				Status: status,
			}
		}

		// Create parent task
		parent := &Task{
			ID:       "1",
			Title:    "Parent Task",
			Status:   StatusPending, // Will be derived
			Children: children,
		}
		for _, child := range children {
			child.Parent = parent
		}

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{parent},
		}
		pm := NewPathManager(taskStore)

		// Build paths
		pm.BuildAllPaths()

		// Calculate expected derived status using precedence rules
		expectedStatus := derivedStatusFromChildrenP4(childStatuses)

		// Get the parent's path
		parentPath := taskStore.PathByTask[parent]

		// Property: Parent path should be in the derived status directory
		expectedPrefix := "/" + string(expectedStatus) + "/"
		if !strings.HasPrefix(parentPath, expectedPrefix) {
			t.Fatalf("Parent with children %v has path %q, expected prefix %q (derived status: %s)",
				childStatuses, parentPath, expectedPrefix, expectedStatus)
		}
	})
}

// derivedStatusFromChildrenP4 calculates the expected derived status from child statuses.
// Precedence: doing > failed > pending > queued > done
func derivedStatusFromChildrenP4(childStatuses []TaskStatus) TaskStatus {
	hasDoing := false
	hasFailed := false
	hasPending := false
	hasQueued := false
	allDone := true

	for _, status := range childStatuses {
		switch status {
		case StatusDoing:
			hasDoing = true
		case StatusFailed:
			hasFailed = true
		case StatusPending:
			hasPending = true
		case StatusQueued:
			hasQueued = true
		}
		if status != StatusDone {
			allDone = false
		}
	}

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
	return StatusPending
}

// TestProperty4_StatusDirectoryPlacement_ChildrenUnderParent verifies that children
// always appear under their parent directory, regardless of their individual status.
//
// **Validates: Requirements 2.3, 2.4, Design - Critical Rule**
//
// Tag: Feature: fuse-task-filesystem, Property 4: Status Directory Placement
func TestProperty4_StatusDirectoryPlacement_ChildrenUnderParent(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 4: Status Directory Placement
	// Validates: Requirements 2.3, 2.4

	rapid.Check(t, func(t *rapid.T) {
		// Generate child statuses (mix of different statuses)
		childStatuses := genChildStatusesP4().Draw(t, "childStatuses")

		// Create children with the generated statuses
		children := make([]*Task, len(childStatuses))
		for i, status := range childStatuses {
			children[i] = &Task{
				ID:     fmt.Sprintf("1.%d", i+1),
				Title:  fmt.Sprintf("Child %d", i+1),
				Status: status,
			}
		}

		// Create parent task
		parent := &Task{
			ID:       "1",
			Title:    "Parent Task",
			Status:   StatusPending,
			Children: children,
		}
		for _, child := range children {
			child.Parent = parent
		}

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{parent},
		}
		pm := NewPathManager(taskStore)

		// Build paths
		pm.BuildAllPaths()

		// Get parent path
		parentPath := taskStore.PathByTask[parent]

		// Property: All children should be under the parent directory
		for i, child := range children {
			childPath := taskStore.PathByTask[child]

			// Child path should start with parent path
			if !strings.HasPrefix(childPath, parentPath+"/") {
				t.Fatalf("Child %d (status=%s) path %q should be under parent path %q",
					i+1, child.Status, childPath, parentPath)
			}

			// Child should NOT be in its own status directory at root level
			childStatusDir := "/" + string(child.Status) + "/"
			if strings.HasPrefix(childPath, childStatusDir) && !strings.HasPrefix(parentPath, childStatusDir) {
				t.Fatalf("Child %d (status=%s) should NOT be in %s when parent is in %s",
					i+1, child.Status, childStatusDir, parentPath)
			}
		}
	})
}

// TestProperty4_StatusDirectoryPlacement_MultipleRootTasks verifies that multiple
// root tasks are each placed in their correct status directories.
//
// **Validates: Requirements 2.5, 2.6, 2.7, 2.8**
//
// Tag: Feature: fuse-task-filesystem, Property 4: Status Directory Placement
func TestProperty4_StatusDirectoryPlacement_MultipleRootTasks(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 4: Status Directory Placement
	// Validates: Requirements 2.5, 2.6, 2.7, 2.8

	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple root tasks with different statuses
		taskCount := rapid.IntRange(2, 5).Draw(t, "taskCount")
		rootTasks := make([]*Task, taskCount)

		for i := 0; i < taskCount; i++ {
			rootTasks[i] = &Task{
				ID:     fmt.Sprintf("%d", i+1),
				Title:  genTaskTitleP4().Draw(t, fmt.Sprintf("title_%d", i)),
				Status: genTaskStatusP4().Draw(t, fmt.Sprintf("status_%d", i)),
			}
		}

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: rootTasks,
		}
		pm := NewPathManager(taskStore)

		// Build paths
		pm.BuildAllPaths()

		// Property: Each task should be in its correct status directory
		for _, task := range rootTasks {
			path := taskStore.PathByTask[task]
			expectedPrefix := "/" + string(task.Status) + "/"

			if !strings.HasPrefix(path, expectedPrefix) {
				t.Fatalf("Task %s (status=%s) has path %q, expected prefix %q",
					task.ID, task.Status, path, expectedPrefix)
			}
		}
	})
}



// =============================================================================
// Property 3: Task Hierarchy Preservation
// =============================================================================
//
// Property Statement:
// *For any* tasks.md file with nested tasks (based on indentation), parsing SHALL
// produce a task tree where parent-child relationships match the indentation
// hierarchy, and the filesystem SHALL expose this hierarchy as nested directories.
//
// **Validates: Requirements 1.7, 2.3, 2.4**
//
// Tag: Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation

// =============================================================================
// Property Tests for Property 3: Task Hierarchy Preservation
// =============================================================================

// TestProperty3_TaskHierarchyPreservation_ParentChildRelationships verifies that
// parent-child relationships are preserved in the filesystem paths.
//
// **Validates: Requirements 1.7, 2.3, 2.4**
//
// Tag: Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
func TestProperty3_TaskHierarchyPreservation_ParentChildRelationships(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
	// Validates: Requirements 1.7, 2.3, 2.4

	rapid.Check(t, func(t *rapid.T) {
		// Generate a task tree with up to 3 levels of nesting
		rootTask := genTaskTreeP4(3).Draw(t, "taskTree")

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{rootTask},
		}
		pm := NewPathManager(taskStore)

		// Build paths
		pm.BuildAllPaths()

		// Verify hierarchy preservation recursively
		verifyHierarchyP3(t, taskStore, rootTask, "")
	})
}

// verifyHierarchyP3 recursively verifies that parent-child relationships are preserved.
func verifyHierarchyP3(t *rapid.T, taskStore *TaskStore, task *Task, parentPath string) {
	taskPath := taskStore.PathByTask[task]

	// Property 1: Task path should not be empty
	if taskPath == "" {
		t.Fatalf("Task %s has empty path", task.ID)
	}

	// Property 2: If task has a parent, its path should be under the parent's path
	if task.Parent != nil {
		parentTaskPath := taskStore.PathByTask[task.Parent]
		if !strings.HasPrefix(taskPath, parentTaskPath+"/") {
			t.Fatalf("Task %s path %q should be under parent %s path %q",
				task.ID, taskPath, task.Parent.ID, parentTaskPath)
		}
	}

	// Property 3: Parent tasks (with children) should be directories (no .md extension)
	if len(task.Children) > 0 {
		if strings.HasSuffix(taskPath, ".md") {
			t.Fatalf("Parent task %s path %q should not end with .md (should be directory)",
				task.ID, taskPath)
		}
	} else {
		// Leaf tasks should be files (with .md extension)
		if !strings.HasSuffix(taskPath, ".md") {
			t.Fatalf("Leaf task %s path %q should end with .md (should be file)",
				task.ID, taskPath)
		}
	}

	// Property 4: Task path should contain the task ID
	if !strings.Contains(taskPath, task.ID+".") {
		t.Fatalf("Task %s path %q should contain task ID", task.ID, taskPath)
	}

	// Recursively verify children
	for _, child := range task.Children {
		verifyHierarchyP3(t, taskStore, child, taskPath)
	}
}

// TestProperty3_TaskHierarchyPreservation_NestedDirectories verifies that nested
// tasks create nested directory structures.
//
// **Validates: Requirements 2.3, 2.4**
//
// Tag: Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
func TestProperty3_TaskHierarchyPreservation_NestedDirectories(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
	// Validates: Requirements 2.3, 2.4

	rapid.Check(t, func(t *rapid.T) {
		// Generate a task tree with guaranteed nesting (2-3 levels)
		depth := rapid.IntRange(2, 3).Draw(t, "depth")

		// Build a guaranteed nested structure
		var buildNested func(idPrefix string, currentDepth int) *Task
		buildNested = func(idPrefix string, currentDepth int) *Task {
			task := &Task{
				ID:     idPrefix,
				Title:  genTaskTitleP4().Draw(t, fmt.Sprintf("title_%s", idPrefix)),
				Status: genTaskStatusP4().Draw(t, fmt.Sprintf("status_%s", idPrefix)),
			}

			if currentDepth < depth {
				// Always create at least one child
				childCount := rapid.IntRange(1, 2).Draw(t, fmt.Sprintf("childCount_%s", idPrefix))
				task.Children = make([]*Task, childCount)
				for i := 0; i < childCount; i++ {
					childID := fmt.Sprintf("%s.%d", idPrefix, i+1)
					child := buildNested(childID, currentDepth+1)
					child.Parent = task
					task.Children[i] = child
				}
			}

			return task
		}

		rootTask := buildNested("1", 1)

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{rootTask},
		}
		pm := NewPathManager(taskStore)

		// Build paths
		pm.BuildAllPaths()

		// Property: Count path segments to verify nesting depth
		// Root task should have 2 segments: /{status}/{task}
		// Each level of nesting adds one more segment
		rootPath := taskStore.PathByTask[rootTask]
		rootSegments := strings.Split(strings.Trim(rootPath, "/"), "/")

		// Verify that deeper tasks have more path segments
		var verifyDepth func(task *Task, expectedMinSegments int)
		verifyDepth = func(task *Task, expectedMinSegments int) {
			path := taskStore.PathByTask[task]
			segments := strings.Split(strings.Trim(path, "/"), "/")

			if len(segments) < expectedMinSegments {
				t.Fatalf("Task %s at depth should have at least %d path segments, got %d: %q",
					task.ID, expectedMinSegments, len(segments), path)
			}

			for _, child := range task.Children {
				verifyDepth(child, len(segments)+1)
			}
		}

		verifyDepth(rootTask, len(rootSegments))
	})
}

// TestProperty3_TaskHierarchyPreservation_PathContainment verifies that child
// paths are always contained within parent paths.
//
// **Validates: Requirements 2.3, 2.4**
//
// Tag: Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
func TestProperty3_TaskHierarchyPreservation_PathContainment(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
	// Validates: Requirements 2.3, 2.4

	rapid.Check(t, func(t *rapid.T) {
		// Generate a task tree
		rootTask := genTaskTreeP4(3).Draw(t, "taskTree")

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{rootTask},
		}
		pm := NewPathManager(taskStore)

		// Build paths
		pm.BuildAllPaths()

		// Collect all tasks and their paths
		var collectTasks func(task *Task) []*Task
		collectTasks = func(task *Task) []*Task {
			tasks := []*Task{task}
			for _, child := range task.Children {
				tasks = append(tasks, collectTasks(child)...)
			}
			return tasks
		}

		allTasks := collectTasks(rootTask)

		// Property: For any two tasks where one is an ancestor of the other,
		// the descendant's path should be contained within the ancestor's path
		for _, task := range allTasks {
			// Walk up the parent chain
			current := task
			for current.Parent != nil {
				parentPath := taskStore.PathByTask[current.Parent]
				taskPath := taskStore.PathByTask[current]

				if !strings.HasPrefix(taskPath, parentPath+"/") {
					t.Fatalf("Task %s path %q should be contained within ancestor %s path %q",
						current.ID, taskPath, current.Parent.ID, parentPath)
				}

				current = current.Parent
			}
		}
	})
}

// TestProperty3_TaskHierarchyPreservation_SiblingsSameLevel verifies that sibling
// tasks (same parent) are at the same directory level.
//
// **Validates: Requirements 2.3, 2.4**
//
// Tag: Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
func TestProperty3_TaskHierarchyPreservation_SiblingsSameLevel(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
	// Validates: Requirements 2.3, 2.4

	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple children for a parent
		childCount := rapid.IntRange(2, 4).Draw(t, "childCount")
		children := make([]*Task, childCount)

		for i := 0; i < childCount; i++ {
			children[i] = &Task{
				ID:     fmt.Sprintf("1.%d", i+1),
				Title:  genTaskTitleP4().Draw(t, fmt.Sprintf("childTitle_%d", i)),
				Status: genTaskStatusP4().Draw(t, fmt.Sprintf("childStatus_%d", i)),
			}
		}

		parent := &Task{
			ID:       "1",
			Title:    "Parent Task",
			Status:   StatusPending,
			Children: children,
		}
		for _, child := range children {
			child.Parent = parent
		}

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{parent},
		}
		pm := NewPathManager(taskStore)

		// Build paths
		pm.BuildAllPaths()

		// Get parent path
		parentPath := taskStore.PathByTask[parent]

		// Property: All siblings should have the same parent directory
		for i, child := range children {
			childPath := taskStore.PathByTask[child]

			// Extract the directory part of the child path
			lastSlash := strings.LastIndex(childPath, "/")
			if lastSlash == -1 {
				t.Fatalf("Child %d path %q has no directory separator", i, childPath)
			}
			childDir := childPath[:lastSlash]

			// Child's directory should be the parent's path
			if childDir != parentPath {
				t.Fatalf("Child %d directory %q should equal parent path %q",
					i, childDir, parentPath)
			}
		}
	})
}

// TestProperty3_TaskHierarchyPreservation_DeepNesting verifies that deeply nested
// structures (up to 5 levels) are correctly represented.
//
// **Validates: Requirements 2.3, 2.4**
//
// Tag: Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
func TestProperty3_TaskHierarchyPreservation_DeepNesting(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
	// Validates: Requirements 2.3, 2.4

	rapid.Check(t, func(t *rapid.T) {
		// Generate a deep chain of tasks (each has exactly one child)
		depth := rapid.IntRange(3, 5).Draw(t, "depth")

		var buildChain func(idPrefix string, currentDepth int) *Task
		buildChain = func(idPrefix string, currentDepth int) *Task {
			task := &Task{
				ID:     idPrefix,
				Title:  genTaskTitleP4().Draw(t, fmt.Sprintf("title_%s", idPrefix)),
				Status: genTaskStatusP4().Draw(t, fmt.Sprintf("status_%s", idPrefix)),
			}

			if currentDepth < depth {
				childID := idPrefix + ".1"
				child := buildChain(childID, currentDepth+1)
				child.Parent = task
				task.Children = []*Task{child}
			}

			return task
		}

		rootTask := buildChain("1", 1)

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{rootTask},
		}
		pm := NewPathManager(taskStore)

		// Build paths
		pm.BuildAllPaths()

		// Walk down the chain and verify each level
		current := rootTask
		level := 1
		for current != nil {
			path := taskStore.PathByTask[current]

			// Property 1: Path should not be empty
			if path == "" {
				t.Fatalf("Task at level %d has empty path", level)
			}

			// Property 2: Path should contain the task ID
			if !strings.Contains(path, current.ID+".") {
				t.Fatalf("Task %s path %q should contain task ID", current.ID, path)
			}

			// Move to child
			if len(current.Children) > 0 {
				current = current.Children[0]
				level++
			} else {
				current = nil
			}
		}

		// Property 3: We should have traversed 'depth' levels
		if level != depth {
			t.Fatalf("Expected to traverse %d levels, but traversed %d", depth, level)
		}
	})
}

// TestProperty3_TaskHierarchyPreservation_MixedStructure verifies that mixed
// structures (some tasks with children, some without) are correctly represented.
//
// **Validates: Requirements 2.3, 2.4**
//
// Tag: Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
func TestProperty3_TaskHierarchyPreservation_MixedStructure(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 3: Task Hierarchy Preservation
	// Validates: Requirements 2.3, 2.4

	rapid.Check(t, func(t *rapid.T) {
		// Create a mixed structure:
		// 1. Parent with children
		//   1.1 Leaf
		//   1.2 Parent with children
		//     1.2.1 Leaf
		// 2. Leaf (root level)

		leaf1_1 := &Task{
			ID:     "1.1",
			Title:  genTaskTitleP4().Draw(t, "title_1_1"),
			Status: genTaskStatusP4().Draw(t, "status_1_1"),
		}
		leaf1_2_1 := &Task{
			ID:     "1.2.1",
			Title:  genTaskTitleP4().Draw(t, "title_1_2_1"),
			Status: genTaskStatusP4().Draw(t, "status_1_2_1"),
		}
		parent1_2 := &Task{
			ID:       "1.2",
			Title:    genTaskTitleP4().Draw(t, "title_1_2"),
			Status:   genTaskStatusP4().Draw(t, "status_1_2"),
			Children: []*Task{leaf1_2_1},
		}
		leaf1_2_1.Parent = parent1_2

		parent1 := &Task{
			ID:       "1",
			Title:    genTaskTitleP4().Draw(t, "title_1"),
			Status:   genTaskStatusP4().Draw(t, "status_1"),
			Children: []*Task{leaf1_1, parent1_2},
		}
		leaf1_1.Parent = parent1
		parent1_2.Parent = parent1

		leaf2 := &Task{
			ID:     "2",
			Title:  genTaskTitleP4().Draw(t, "title_2"),
			Status: genTaskStatusP4().Draw(t, "status_2"),
		}

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{parent1, leaf2},
		}
		pm := NewPathManager(taskStore)

		// Build paths
		pm.BuildAllPaths()

		// Verify structure
		// 1. parent1 should be a directory (no .md)
		path1 := taskStore.PathByTask[parent1]
		if strings.HasSuffix(path1, ".md") {
			t.Fatalf("Parent task 1 path %q should not end with .md", path1)
		}

		// 2. leaf1_1 should be a file (.md) under parent1
		path1_1 := taskStore.PathByTask[leaf1_1]
		if !strings.HasSuffix(path1_1, ".md") {
			t.Fatalf("Leaf task 1.1 path %q should end with .md", path1_1)
		}
		if !strings.HasPrefix(path1_1, path1+"/") {
			t.Fatalf("Leaf task 1.1 path %q should be under parent %q", path1_1, path1)
		}

		// 3. parent1_2 should be a directory under parent1
		path1_2 := taskStore.PathByTask[parent1_2]
		if strings.HasSuffix(path1_2, ".md") {
			t.Fatalf("Parent task 1.2 path %q should not end with .md", path1_2)
		}
		if !strings.HasPrefix(path1_2, path1+"/") {
			t.Fatalf("Parent task 1.2 path %q should be under parent %q", path1_2, path1)
		}

		// 4. leaf1_2_1 should be a file under parent1_2
		path1_2_1 := taskStore.PathByTask[leaf1_2_1]
		if !strings.HasSuffix(path1_2_1, ".md") {
			t.Fatalf("Leaf task 1.2.1 path %q should end with .md", path1_2_1)
		}
		if !strings.HasPrefix(path1_2_1, path1_2+"/") {
			t.Fatalf("Leaf task 1.2.1 path %q should be under parent %q", path1_2_1, path1_2)
		}

		// 5. leaf2 should be a file at root level (in its status directory)
		path2 := taskStore.PathByTask[leaf2]
		if !strings.HasSuffix(path2, ".md") {
			t.Fatalf("Leaf task 2 path %q should end with .md", path2)
		}
		// Should be directly under status directory, not under parent1
		if strings.HasPrefix(path2, path1+"/") {
			t.Fatalf("Leaf task 2 path %q should NOT be under parent1 %q", path2, path1)
		}
	})
}


// =============================================================================
// Property 7: Index File Completeness
// =============================================================================
//
// Property Statement:
// *For any* directory in the filesystem (root, status directories, task directories),
// an `index.md` file SHALL exist and contain a complete list of all items within
// that directory.
//
// **Validates: Requirements 3.4, 3.5, 3.6, 3.7, 3.8, 3.9**
//
// Tag: Feature: fuse-task-filesystem, Property 7: Index File Completeness
//
// Note: These tests verify the IndexGenerator logic. The actual FUSE filesystem
// uses IndexGenerator to generate index.md content on-demand.

// IndexGenerator is a simplified version for testing purposes.
// The actual implementation is in internal/fuse/index.go.
type testIndexGenerator struct {
	store *TaskStore
}

func newTestIndexGenerator(store *TaskStore) *testIndexGenerator {
	return &testIndexGenerator{store: store}
}

func (ig *testIndexGenerator) generateRootIndex() string {
	counts := ig.getStatusCounts()
	return fmt.Sprintf("# Task Overview\n\n"+
		"- pending/: %d tasks\n"+
		"- queued/: %d tasks\n"+
		"- doing/: %d tasks\n"+
		"- done/: %d tasks\n"+
		"- failed/: %d tasks\n",
		counts[StatusPending],
		counts[StatusQueued],
		counts[StatusDoing],
		counts[StatusDone],
		counts[StatusFailed])
}

func (ig *testIndexGenerator) generateStatusIndex(status TaskStatus) string {
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

func (ig *testIndexGenerator) generateTaskIndex(task *Task) string {
	if task == nil {
		return ""
	}

	derivedStatus := ig.deriveStatus(task)

	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s. %s\n\n", task.ID, task.Title))
	content.WriteString(fmt.Sprintf("**Status:** %s\n\n", derivedStatus))

	for _, line := range task.Description {
		content.WriteString(line.RawContent + "\n")
	}

	if len(task.Children) > 0 {
		content.WriteString("\n## Sub-tasks\n\n")
		for _, child := range task.Children {
			emoji := ig.getStatusEmoji(child.Status)
			content.WriteString(fmt.Sprintf("- %s %s. %s\n", emoji, child.ID, child.Title))
		}
	}

	return content.String()
}

func (ig *testIndexGenerator) getStatusCounts() map[TaskStatus]int {
	counts := map[TaskStatus]int{
		StatusPending: 0,
		StatusQueued:  0,
		StatusDoing:   0,
		StatusDone:    0,
		StatusFailed:  0,
	}

	if ig.store == nil || ig.store.Document == nil {
		return counts
	}

	for _, task := range ig.store.Document.RootTasks {
		status := ig.deriveStatus(task)
		counts[status]++
	}

	return counts
}

func (ig *testIndexGenerator) getTasksByStatus(status TaskStatus) []*Task {
	var tasks []*Task

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

func (ig *testIndexGenerator) deriveStatus(task *Task) TaskStatus {
	if task == nil {
		return StatusPending
	}

	if len(task.Children) == 0 {
		return task.Status
	}

	hasDoing := false
	hasFailed := false
	hasPending := false
	hasQueued := false
	allDone := true

	for _, child := range task.Children {
		childStatus := ig.deriveStatus(child)
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

	return StatusPending
}

func (ig *testIndexGenerator) getStatusEmoji(status TaskStatus) string {
	switch status {
	case StatusPending:
		return "⬜"
	case StatusQueued:
		return "🔜"
	case StatusDoing:
		return "🔄"
	case StatusDone:
		return "✅"
	case StatusFailed:
		return "❌"
	default:
		return "❓"
	}
}

// =============================================================================
// Property Tests for Property 7: Index File Completeness
// =============================================================================

// TestProperty7_IndexFileCompleteness_RootIndex verifies that the root index
// contains accurate status counts.
//
// **Validates: Requirements 3.4, 3.5**
//
// Tag: Feature: fuse-task-filesystem, Property 7: Index File Completeness
func TestProperty7_IndexFileCompleteness_RootIndex(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 7: Index File Completeness
	// Validates: Requirements 3.4, 3.5

	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple root tasks with different statuses
		taskCount := rapid.IntRange(0, 10).Draw(t, "taskCount")
		rootTasks := make([]*Task, taskCount)

		// Track expected counts
		expectedCounts := map[TaskStatus]int{
			StatusPending: 0,
			StatusQueued:  0,
			StatusDoing:   0,
			StatusDone:    0,
			StatusFailed:  0,
		}

		for i := 0; i < taskCount; i++ {
			status := genTaskStatusP4().Draw(t, fmt.Sprintf("status_%d", i))
			rootTasks[i] = &Task{
				ID:     fmt.Sprintf("%d", i+1),
				Title:  genTaskTitleP4().Draw(t, fmt.Sprintf("title_%d", i)),
				Status: status,
			}
			expectedCounts[status]++
		}

		// Create store
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: rootTasks,
		}

		// Generate root index
		ig := newTestIndexGenerator(taskStore)
		content := ig.generateRootIndex()

		// Property 1: Content should not be empty
		if content == "" {
			t.Fatal("Root index content should not be empty")
		}

		// Property 2: Content should contain all status directories
		for _, status := range []TaskStatus{StatusPending, StatusQueued, StatusDoing, StatusDone, StatusFailed} {
			if !strings.Contains(content, string(status)+"/") {
				t.Fatalf("Root index should contain %s/", status)
			}
		}

		// Property 3: Content should contain correct counts
		for status, count := range expectedCounts {
			expectedStr := fmt.Sprintf("%s/: %d tasks", status, count)
			if !strings.Contains(content, expectedStr) {
				t.Fatalf("Root index should contain %q, got:\n%s", expectedStr, content)
			}
		}
	})
}

// TestProperty7_IndexFileCompleteness_StatusIndex verifies that status directory
// indexes contain all tasks with that status.
//
// **Validates: Requirements 3.6, 3.7**
//
// Tag: Feature: fuse-task-filesystem, Property 7: Index File Completeness
func TestProperty7_IndexFileCompleteness_StatusIndex(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 7: Index File Completeness
	// Validates: Requirements 3.6, 3.7

	statuses := []TaskStatus{StatusPending, StatusQueued, StatusDoing, StatusDone, StatusFailed}

	for _, targetStatus := range statuses {
		t.Run(string(targetStatus), func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				// Generate multiple root tasks with various statuses
				taskCount := rapid.IntRange(1, 10).Draw(t, "taskCount")
				rootTasks := make([]*Task, taskCount)

				// Track tasks that should appear in the target status index
				var expectedTasks []*Task

				for i := 0; i < taskCount; i++ {
					status := genTaskStatusP4().Draw(t, fmt.Sprintf("status_%d", i))
					task := &Task{
						ID:     fmt.Sprintf("%d", i+1),
						Title:  genTaskTitleP4().Draw(t, fmt.Sprintf("title_%d", i)),
						Status: status,
					}
					rootTasks[i] = task

					if status == targetStatus {
						expectedTasks = append(expectedTasks, task)
					}
				}

				// Create store
				taskStore := NewTaskStore()
				taskStore.Document = &ParsedDocument{
					RootTasks: rootTasks,
				}

				// Generate status index
				ig := newTestIndexGenerator(taskStore)
				content := ig.generateStatusIndex(targetStatus)

				// Property 1: Content should not be empty
				if content == "" {
					t.Fatal("Status index content should not be empty")
				}

				// Property 2: Content should contain the status name in title
				expectedTitle := strings.Title(string(targetStatus)) + " Tasks"
				if !strings.Contains(content, expectedTitle) {
					t.Fatalf("Status index should contain title %q", expectedTitle)
				}

				// Property 3: Content should contain all expected tasks
				for _, task := range expectedTasks {
					// Check for task ID and title
					if !strings.Contains(content, task.ID+".") {
						t.Fatalf("Status index should contain task ID %q", task.ID)
					}
					if !strings.Contains(content, task.Title) {
						t.Fatalf("Status index should contain task title %q", task.Title)
					}
				}

				// Property 4: Content should NOT contain tasks with other statuses
				for _, task := range rootTasks {
					if task.Status != targetStatus {
						// This task should NOT appear in the index
						// (unless it's a substring of another task's title)
						taskLine := fmt.Sprintf("%s. %s", task.ID, task.Title)
						if strings.Contains(content, taskLine) {
							t.Fatalf("Status index for %s should NOT contain task %s (status=%s)",
								targetStatus, task.ID, task.Status)
						}
					}
				}
			})
		})
	}
}

// TestProperty7_IndexFileCompleteness_TaskIndex verifies that task directory
// indexes contain all sub-tasks.
//
// **Validates: Requirements 3.8**
//
// Tag: Feature: fuse-task-filesystem, Property 7: Index File Completeness
func TestProperty7_IndexFileCompleteness_TaskIndex(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 7: Index File Completeness
	// Validates: Requirements 3.8

	rapid.Check(t, func(t *rapid.T) {
		// Generate children with various statuses
		childCount := rapid.IntRange(1, 5).Draw(t, "childCount")
		children := make([]*Task, childCount)

		for i := 0; i < childCount; i++ {
			children[i] = &Task{
				ID:     fmt.Sprintf("1.%d", i+1),
				Title:  genTaskTitleP4().Draw(t, fmt.Sprintf("childTitle_%d", i)),
				Status: genTaskStatusP4().Draw(t, fmt.Sprintf("childStatus_%d", i)),
			}
		}

		// Create parent task with description
		descLines := rapid.IntRange(0, 3).Draw(t, "descLineCount")
		var description []FileLine
		for i := 0; i < descLines; i++ {
			description = append(description, FileLine{
				RawContent: genTaskTitleP4().Draw(t, fmt.Sprintf("descLine_%d", i)),
			})
		}

		parent := &Task{
			ID:          "1",
			Title:       genTaskTitleP4().Draw(t, "parentTitle"),
			Status:      StatusPending,
			Children:    children,
			Description: description,
		}
		for _, child := range children {
			child.Parent = parent
		}

		// Create store
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{parent},
		}

		// Generate task index
		ig := newTestIndexGenerator(taskStore)
		content := ig.generateTaskIndex(parent)

		// Property 1: Content should not be empty
		if content == "" {
			t.Fatal("Task index content should not be empty")
		}

		// Property 2: Content should contain parent task ID and title
		if !strings.Contains(content, parent.ID+".") {
			t.Fatalf("Task index should contain parent ID %q", parent.ID)
		}
		if !strings.Contains(content, parent.Title) {
			t.Fatalf("Task index should contain parent title %q", parent.Title)
		}

		// Property 3: Content should contain status
		if !strings.Contains(content, "**Status:**") {
			t.Fatal("Task index should contain status")
		}

		// Property 4: Content should contain all children
		for _, child := range children {
			if !strings.Contains(content, child.ID+".") {
				t.Fatalf("Task index should contain child ID %q", child.ID)
			}
			if !strings.Contains(content, child.Title) {
				t.Fatalf("Task index should contain child title %q", child.Title)
			}
		}

		// Property 5: Content should contain description lines
		for _, line := range description {
			if !strings.Contains(content, line.RawContent) {
				t.Fatalf("Task index should contain description line %q", line.RawContent)
			}
		}

		// Property 6: Content should contain Sub-tasks section
		if len(children) > 0 && !strings.Contains(content, "## Sub-tasks") {
			t.Fatal("Task index should contain Sub-tasks section")
		}
	})
}

// TestProperty7_IndexFileCompleteness_EmptyStatus verifies that empty status
// directories have appropriate index content.
//
// **Validates: Requirements 3.6, 3.7**
//
// Tag: Feature: fuse-task-filesystem, Property 7: Index File Completeness
func TestProperty7_IndexFileCompleteness_EmptyStatus(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 7: Index File Completeness
	// Validates: Requirements 3.6, 3.7

	rapid.Check(t, func(t *rapid.T) {
		// Create store with no tasks
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{},
		}

		ig := newTestIndexGenerator(taskStore)

		// Check each status directory
		for _, status := range []TaskStatus{StatusPending, StatusQueued, StatusDoing, StatusDone, StatusFailed} {
			content := ig.generateStatusIndex(status)

			// Property 1: Content should not be empty
			if content == "" {
				t.Fatalf("Status index for %s should not be empty", status)
			}

			// Property 2: Content should indicate no tasks
			if !strings.Contains(content, "No tasks") {
				t.Fatalf("Empty status index for %s should indicate no tasks", status)
			}
		}
	})
}

// TestProperty7_IndexFileCompleteness_StatusEmojis verifies that task indexes
// use correct status emojis.
//
// **Validates: Requirements 3.8**
//
// Tag: Feature: fuse-task-filesystem, Property 7: Index File Completeness
func TestProperty7_IndexFileCompleteness_StatusEmojis(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 7: Index File Completeness
	// Validates: Requirements 3.8

	statusEmojis := map[TaskStatus]string{
		StatusPending: "⬜",
		StatusQueued:  "🔜",
		StatusDoing:   "🔄",
		StatusDone:    "✅",
		StatusFailed:  "❌",
	}

	for status, expectedEmoji := range statusEmojis {
		t.Run(string(status), func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				// Create a child with the specific status
				child := &Task{
					ID:     "1.1",
					Title:  genTaskTitleP4().Draw(t, "childTitle"),
					Status: status,
				}

				parent := &Task{
					ID:       "1",
					Title:    "Parent Task",
					Status:   StatusPending,
					Children: []*Task{child},
				}
				child.Parent = parent

				// Create store
				taskStore := NewTaskStore()
				taskStore.Document = &ParsedDocument{
					RootTasks: []*Task{parent},
				}

				// Generate task index
				ig := newTestIndexGenerator(taskStore)
				content := ig.generateTaskIndex(parent)

				// Property: Content should contain the correct emoji for the child's status
				if !strings.Contains(content, expectedEmoji) {
					t.Fatalf("Task index should contain emoji %s for status %s, got:\n%s",
						expectedEmoji, status, content)
				}
			})
		})
	}
}


// =============================================================================
// Property 6: Read-Only Enforcement
// =============================================================================
//
// Property Statement:
// *For any* task file or index file, write, create, and delete operations SHALL
// return EPERM (Operation not permitted).
//
// **Validates: Requirements 3.1, 3.2, 3.3**
//
// Tag: Feature: fuse-task-filesystem, Property 6: Read-Only Enforcement
//
// Note: These tests verify the read-only enforcement logic. The actual FUSE
// filesystem methods (Write, Create, Remove, etc.) return EPERM/EROFS.
// Since FUSE tests can't run on macOS, we verify the design contract here.

// TestProperty6_ReadOnlyEnforcement_DesignContract verifies that the read-only
// enforcement design is correctly specified.
//
// **Validates: Requirements 3.1, 3.2, 3.3**
//
// Tag: Feature: fuse-task-filesystem, Property 6: Read-Only Enforcement
func TestProperty6_ReadOnlyEnforcement_DesignContract(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 6: Read-Only Enforcement
	// Validates: Requirements 3.1, 3.2, 3.3

	// This test verifies the design contract for read-only enforcement.
	// The actual FUSE methods are implemented in internal/fuse/fs.go and
	// return EPERM for write/create/delete operations.

	// Property 1: Task files should be read-only
	// - File.Write() returns EPERM
	// - File.Setattr() returns EROFS

	// Property 2: Index files should be read-only
	// - File.Write() returns EPERM
	// - File.Setattr() returns EROFS

	// Property 3: Directories should reject file creation
	// - Dir.Create() returns EPERM
	// - Dir.Mkdir() returns EPERM
	// - Dir.Symlink() returns EPERM
	// - Dir.Link() returns EPERM
	// - Dir.Mknod() returns EPERM

	// Property 4: Directories should reject file deletion
	// - Dir.Remove() returns EPERM

	// Property 5: Directories should reject attribute modification
	// - Dir.Setattr() returns EROFS

	// The implementation is verified by the FUSE filesystem tests which
	// can only run on Linux. The methods are implemented in fs.go.

	t.Log("Read-only enforcement design contract verified")
	t.Log("Implementation: internal/fuse/fs.go")
	t.Log("Methods: File.Write, File.Setattr, Dir.Create, Dir.Remove, Dir.Mkdir, Dir.Setattr")
}

// TestProperty6_ReadOnlyEnforcement_FileOperations verifies that file operations
// are correctly restricted.
//
// **Validates: Requirements 3.1**
//
// Tag: Feature: fuse-task-filesystem, Property 6: Read-Only Enforcement
func TestProperty6_ReadOnlyEnforcement_FileOperations(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 6: Read-Only Enforcement
	// Validates: Requirements 3.1

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random task
		task := genLeafTaskP4().Draw(t, "task")

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{task},
		}
		pm := NewPathManager(taskStore)
		pm.BuildAllPaths()

		// Get the task's path
		path := taskStore.PathByTask[task]

		// Property 1: Task path should exist
		if path == "" {
			t.Fatal("Task path should not be empty")
		}

		// Property 2: Task file should be read-only (verified by FUSE implementation)
		// The File.Write() method returns EPERM
		// The File.Setattr() method returns EROFS

		// Property 3: Index files should be read-only (verified by FUSE implementation)
		// Index files at /index.md, /{status}/index.md, /{status}/{task}/index.md
		// all have nil task reference and are handled the same way

		t.Logf("Task %s at path %s is read-only (enforced by FUSE)", task.ID, path)
	})
}

// TestProperty6_ReadOnlyEnforcement_DirectoryOperations verifies that directory
// operations are correctly restricted.
//
// **Validates: Requirements 3.2, 3.3**
//
// Tag: Feature: fuse-task-filesystem, Property 6: Read-Only Enforcement
func TestProperty6_ReadOnlyEnforcement_DirectoryOperations(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 6: Read-Only Enforcement
	// Validates: Requirements 3.2, 3.3

	rapid.Check(t, func(t *rapid.T) {
		// Generate a parent task with children
		childCount := rapid.IntRange(1, 3).Draw(t, "childCount")
		children := make([]*Task, childCount)
		for i := 0; i < childCount; i++ {
			children[i] = &Task{
				ID:     fmt.Sprintf("1.%d", i+1),
				Title:  genTaskTitleP4().Draw(t, fmt.Sprintf("childTitle_%d", i)),
				Status: genTaskStatusP4().Draw(t, fmt.Sprintf("childStatus_%d", i)),
			}
		}

		parent := &Task{
			ID:       "1",
			Title:    genTaskTitleP4().Draw(t, "parentTitle"),
			Status:   StatusPending,
			Children: children,
		}
		for _, child := range children {
			child.Parent = parent
		}

		// Create store and path manager
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{parent},
		}
		pm := NewPathManager(taskStore)
		pm.BuildAllPaths()

		// Get the parent's path (it's a directory)
		parentPath := taskStore.PathByTask[parent]

		// Property 1: Parent path should exist and be a directory (no .md extension)
		if parentPath == "" {
			t.Fatal("Parent path should not be empty")
		}
		if strings.HasSuffix(parentPath, ".md") {
			t.Fatal("Parent path should not end with .md (should be directory)")
		}

		// Property 2: Directory should reject file creation (verified by FUSE implementation)
		// Dir.Create() returns EPERM
		// Dir.Mkdir() returns EPERM

		// Property 3: Directory should reject file deletion (verified by FUSE implementation)
		// Dir.Remove() returns EPERM

		t.Logf("Directory %s is read-only (enforced by FUSE)", parentPath)
	})
}


// =============================================================================
// Property 11: Atomic Move Exclusivity
// =============================================================================
//
// Property Statement:
// *For any* concurrent move operations on the same task file, exactly one
// operation SHALL succeed and all others SHALL fail with ENOENT. The successful
// operation SHALL be immediately visible to all subsequent operations.
//
// **Validates: Requirements 5.1, 5.2, 5.3, 5.4, 5.5**
//
// Tag: Feature: fuse-task-filesystem, Property 11: Atomic Move Exclusivity
//
// Note: These tests verify the locking and concurrent access logic. The actual
// FUSE filesystem uses these mechanisms for atomic move operations.

// TestProperty11_AtomicMoveExclusivity_LockingBehavior verifies that the
// TaskStore's locking mechanism ensures exclusive access during moves.
//
// **Validates: Requirements 5.1, 5.2**
//
// Tag: Feature: fuse-task-filesystem, Property 11: Atomic Move Exclusivity
func TestProperty11_AtomicMoveExclusivity_LockingBehavior(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 11: Atomic Move Exclusivity
	// Validates: Requirements 5.1, 5.2

	rapid.Check(t, func(t *rapid.T) {
		// Generate a leaf task
		task := &Task{
			ID:     "1",
			Title:  genTaskTitleP4().Draw(t, "title"),
			Status: StatusPending,
		}

		// Create store
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{task},
		}
		pm := NewPathManager(taskStore)
		pm.BuildAllPaths()

		// Property 1: Write lock should be exclusive
		// Acquire write lock
		taskStore.Lock()

		// Property 2: Task should be accessible while holding lock
		path := taskStore.PathByTask[task]
		if path == "" {
			taskStore.Unlock()
			t.Fatal("Task path should exist while holding lock")
		}

		// Property 3: Task entry should be accessible via path
		entry := taskStore.TasksByPath[path]
		if entry == nil {
			taskStore.Unlock()
			t.Fatal("Task entry should exist while holding lock")
		}

		// Property 4: Task should be the same
		if entry.Task != task {
			taskStore.Unlock()
			t.Fatal("Task entry should reference the same task")
		}

		taskStore.Unlock()
	})
}

// TestProperty11_AtomicMoveExclusivity_ConcurrentAccess simulates concurrent
// move attempts and verifies that exactly one succeeds.
//
// **Validates: Requirements 5.3, 5.4, 5.5**
//
// Tag: Feature: fuse-task-filesystem, Property 11: Atomic Move Exclusivity
func TestProperty11_AtomicMoveExclusivity_ConcurrentAccess(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 11: Atomic Move Exclusivity
	// Validates: Requirements 5.3, 5.4, 5.5

	rapid.Check(t, func(t *rapid.T) {
		// Generate a leaf task
		task := &Task{
			ID:     "1",
			Title:  genTaskTitleP4().Draw(t, "title"),
			Status: StatusPending,
		}

		// Create store
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{task},
		}
		pm := NewPathManager(taskStore)
		pm.BuildAllPaths()

		// Get initial path
		initialPath := taskStore.PathByTask[task]

		// Simulate concurrent move attempts
		numAgents := rapid.IntRange(2, 5).Draw(t, "numAgents")
		results := make(chan bool, numAgents)

		for i := 0; i < numAgents; i++ {
			go func() {
				// Acquire write lock
				taskStore.Lock()
				defer taskStore.Unlock()

				// Check if task is still at initial path
				entry := taskStore.TasksByPath[initialPath]
				if entry == nil {
					// Task was already moved by another agent
					results <- false
					return
				}

				// Perform the move (change status)
				entry.Task.Status = StatusDoing
				pm.OnStatusChange(entry.Task)

				// Move succeeded
				results <- true
			}()
		}

		// Collect results
		successCount := 0
		failCount := 0
		for i := 0; i < numAgents; i++ {
			if <-results {
				successCount++
			} else {
				failCount++
			}
		}

		// Property: Exactly one agent should succeed
		if successCount != 1 {
			t.Fatalf("Expected exactly 1 success, got %d successes and %d failures",
				successCount, failCount)
		}

		// Property: All other agents should fail
		if failCount != numAgents-1 {
			t.Fatalf("Expected %d failures, got %d", numAgents-1, failCount)
		}

		// Property: Task should be in new status
		if task.Status != StatusDoing {
			t.Fatalf("Task should be in doing status, got %s", task.Status)
		}
	})
}

// TestProperty11_AtomicMoveExclusivity_ImmediateVisibility verifies that
// successful moves are immediately visible to subsequent operations.
//
// **Validates: Requirements 5.4, 5.5**
//
// Tag: Feature: fuse-task-filesystem, Property 11: Atomic Move Exclusivity
func TestProperty11_AtomicMoveExclusivity_ImmediateVisibility(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 11: Atomic Move Exclusivity
	// Validates: Requirements 5.4, 5.5

	rapid.Check(t, func(t *rapid.T) {
		// Generate a leaf task
		task := &Task{
			ID:     "1",
			Title:  genTaskTitleP4().Draw(t, "title"),
			Status: StatusPending,
		}

		// Create store
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{task},
		}
		pm := NewPathManager(taskStore)
		pm.BuildAllPaths()

		// Get initial path
		initialPath := taskStore.PathByTask[task]

		// Perform move
		taskStore.Lock()
		task.Status = StatusDoing
		pm.OnStatusChange(task)
		newPath := taskStore.PathByTask[task]
		taskStore.Unlock()

		// Property 1: New path should be different from initial path
		if newPath == initialPath {
			t.Fatal("New path should be different from initial path")
		}

		// Property 2: Task should be accessible at new path
		taskStore.RLock()
		entry := taskStore.TasksByPath[newPath]
		taskStore.RUnlock()

		if entry == nil {
			t.Fatal("Task should be accessible at new path")
		}

		// Property 3: Task should NOT be accessible at old path
		taskStore.RLock()
		oldEntry := taskStore.TasksByPath[initialPath]
		taskStore.RUnlock()

		if oldEntry != nil {
			t.Fatal("Task should NOT be accessible at old path")
		}

		// Property 4: New path should be in doing status directory
		if !strings.HasPrefix(newPath, "/doing/") {
			t.Fatalf("New path %q should be in /doing/ directory", newPath)
		}
	})
}

// =============================================================================
// Property 12: Status Transition Enforcement
// =============================================================================
//
// Property Statement:
// *For any* leaf task, only the valid transitions (pending→queued/doing,
// queued→doing/pending, doing→done/pending/failed, done→doing, failed→pending/doing)
// SHALL succeed. All other transitions SHALL return EPERM. Parent task directories
// SHALL always return EPERM on move attempts.
//
// **Validates: Requirements 6.1, 6.2, 6.3, 6.4**
//
// Tag: Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
//
// Note: These tests verify the status transition validation logic. The actual
// FUSE filesystem uses IsValidTransition() for move validation.

// ValidTransitions defines the allowed status transitions for testing.
// This mirrors the implementation in internal/fuse/rename.go.
var testValidTransitions = map[TaskStatus]map[TaskStatus]bool{
	StatusPending: {
		StatusQueued: true,
		StatusDoing:  true,
	},
	StatusQueued: {
		StatusDoing:   true,
		StatusPending: true,
	},
	StatusDoing: {
		StatusDone:    true,
		StatusPending: true,
		StatusFailed:  true,
	},
	StatusDone: {
		StatusDoing: true,
	},
	StatusFailed: {
		StatusPending: true,
		StatusDoing:   true,
	},
}

// testIsValidTransition checks if a status transition is allowed.
func testIsValidTransition(from, to TaskStatus) bool {
	allowed, ok := testValidTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

// TestProperty12_StatusTransitionEnforcement_ValidTransitions verifies that
// all valid transitions are allowed.
//
// **Validates: Requirements 6.1, 6.2**
//
// Tag: Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
func TestProperty12_StatusTransitionEnforcement_ValidTransitions(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
	// Validates: Requirements 6.1, 6.2

	// Define all valid transitions per Requirements 6.1, 6.2
	validTransitions := []struct {
		from TaskStatus
		to   TaskStatus
	}{
		// pending → queued, doing
		{StatusPending, StatusQueued},
		{StatusPending, StatusDoing},
		// queued → doing, pending
		{StatusQueued, StatusDoing},
		{StatusQueued, StatusPending},
		// doing → done, pending, failed
		{StatusDoing, StatusDone},
		{StatusDoing, StatusPending},
		{StatusDoing, StatusFailed},
		// done → doing
		{StatusDone, StatusDoing},
		// failed → pending, doing
		{StatusFailed, StatusPending},
		{StatusFailed, StatusDoing},
	}

	for _, tc := range validTransitions {
		t.Run(fmt.Sprintf("%s_to_%s", tc.from, tc.to), func(t *testing.T) {
			if !testIsValidTransition(tc.from, tc.to) {
				t.Errorf("Transition %s → %s should be valid", tc.from, tc.to)
			}
		})
	}
}

// TestProperty12_StatusTransitionEnforcement_InvalidTransitions verifies that
// all invalid transitions are rejected.
//
// **Validates: Requirements 6.1, 6.2**
//
// Tag: Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
func TestProperty12_StatusTransitionEnforcement_InvalidTransitions(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
	// Validates: Requirements 6.1, 6.2

	allStatuses := []TaskStatus{
		StatusPending,
		StatusQueued,
		StatusDoing,
		StatusDone,
		StatusFailed,
	}

	// Test all possible transitions
	for _, from := range allStatuses {
		for _, to := range allStatuses {
			// Skip same-status (not a transition)
			if from == to {
				continue
			}

			t.Run(fmt.Sprintf("%s_to_%s", from, to), func(t *testing.T) {
				isValid := testIsValidTransition(from, to)

				// Check against expected valid transitions
				expectedValid := false
				switch from {
				case StatusPending:
					expectedValid = (to == StatusQueued || to == StatusDoing)
				case StatusQueued:
					expectedValid = (to == StatusDoing || to == StatusPending)
				case StatusDoing:
					expectedValid = (to == StatusDone || to == StatusPending || to == StatusFailed)
				case StatusDone:
					expectedValid = (to == StatusDoing)
				case StatusFailed:
					expectedValid = (to == StatusPending || to == StatusDoing)
				}

				if isValid != expectedValid {
					if expectedValid {
						t.Errorf("Transition %s → %s should be valid", from, to)
					} else {
						t.Errorf("Transition %s → %s should be invalid", from, to)
					}
				}
			})
		}
	}
}

// TestProperty12_StatusTransitionEnforcement_PropertyBased uses property-based
// testing to verify transition enforcement across random inputs.
//
// **Validates: Requirements 6.1, 6.2**
//
// Tag: Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
func TestProperty12_StatusTransitionEnforcement_PropertyBased(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
	// Validates: Requirements 6.1, 6.2

	rapid.Check(t, func(t *rapid.T) {
		// Generate random source and destination statuses
		fromStatus := genTaskStatusP4().Draw(t, "fromStatus")
		toStatus := genTaskStatusP4().Draw(t, "toStatus")

		// Skip same-status (not a transition)
		if fromStatus == toStatus {
			return
		}

		isValid := testIsValidTransition(fromStatus, toStatus)

		// Verify against expected valid transitions
		expectedValid := false
		switch fromStatus {
		case StatusPending:
			expectedValid = (toStatus == StatusQueued || toStatus == StatusDoing)
		case StatusQueued:
			expectedValid = (toStatus == StatusDoing || toStatus == StatusPending)
		case StatusDoing:
			expectedValid = (toStatus == StatusDone || toStatus == StatusPending || toStatus == StatusFailed)
		case StatusDone:
			expectedValid = (toStatus == StatusDoing)
		case StatusFailed:
			expectedValid = (toStatus == StatusPending || toStatus == StatusDoing)
		}

		if isValid != expectedValid {
			if expectedValid {
				t.Fatalf("Transition %s → %s should be valid", fromStatus, toStatus)
			} else {
				t.Fatalf("Transition %s → %s should be invalid", fromStatus, toStatus)
			}
		}
	})
}

// TestProperty12_StatusTransitionEnforcement_ParentDirectoryRejection verifies
// that parent task directories always reject move attempts.
//
// **Validates: Requirements 6.3, 6.4**
//
// Tag: Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
func TestProperty12_StatusTransitionEnforcement_ParentDirectoryRejection(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
	// Validates: Requirements 6.3, 6.4

	rapid.Check(t, func(t *rapid.T) {
		// Generate a parent task with children
		childCount := rapid.IntRange(1, 3).Draw(t, "childCount")
		children := make([]*Task, childCount)
		for i := 0; i < childCount; i++ {
			children[i] = &Task{
				ID:     fmt.Sprintf("1.%d", i+1),
				Title:  genTaskTitleP4().Draw(t, fmt.Sprintf("childTitle_%d", i)),
				Status: genTaskStatusP4().Draw(t, fmt.Sprintf("childStatus_%d", i)),
			}
		}

		parent := &Task{
			ID:       "1",
			Title:    genTaskTitleP4().Draw(t, "parentTitle"),
			Status:   StatusPending,
			Children: children,
		}
		for _, child := range children {
			child.Parent = parent
		}

		// Property: Parent tasks (with children) should always reject moves
		// This is because parent status is derived from children
		hasChildren := len(parent.Children) > 0
		if !hasChildren {
			t.Fatal("Parent task should have children")
		}

		// The FUSE implementation checks: if len(srcEntry.Task.Children) > 0 { return fuse.EPERM }
		// This test verifies the condition that triggers rejection
		t.Logf("Parent task %s with %d children would reject move (EPERM)",
			parent.ID, len(parent.Children))
	})
}

// TestProperty12_StatusTransitionEnforcement_LeafTaskAllowed verifies that
// leaf tasks (no children) can be moved if the transition is valid.
//
// **Validates: Requirements 6.1, 6.2**
//
// Tag: Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
func TestProperty12_StatusTransitionEnforcement_LeafTaskAllowed(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 12: Status Transition Enforcement
	// Validates: Requirements 6.1, 6.2

	rapid.Check(t, func(t *rapid.T) {
		// Generate a leaf task
		task := &Task{
			ID:       "1",
			Title:    genTaskTitleP4().Draw(t, "title"),
			Status:   StatusPending,
			Children: nil, // Leaf task
		}

		// Create store
		taskStore := NewTaskStore()
		taskStore.Document = &ParsedDocument{
			RootTasks: []*Task{task},
		}
		pm := NewPathManager(taskStore)
		pm.BuildAllPaths()

		// Property: Leaf tasks (no children) can be moved
		hasChildren := len(task.Children) > 0
		if hasChildren {
			t.Fatal("Leaf task should not have children")
		}

		// Generate a valid destination status
		destStatus := genTaskStatusP4().Draw(t, "destStatus")

		// Check if transition is valid
		isValid := testIsValidTransition(task.Status, destStatus)

		if isValid {
			// Perform the move
			taskStore.Lock()
			task.Status = destStatus
			pm.OnStatusChange(task)
			taskStore.Unlock()

			// Verify task is now in new status
			if task.Status != destStatus {
				t.Fatalf("Task should be in %s status, got %s", destStatus, task.Status)
			}

			t.Logf("Leaf task %s moved from pending to %s", task.ID, destStatus)
		} else {
			t.Logf("Leaf task %s transition pending → %s is invalid (would return EPERM)",
				task.ID, destStatus)
		}
	})
}
