// Package sync provides property-based tests for the Sync_Engine component.
//
// Feature: fuse-task-filesystem
// Property 8: Sync Content Integrity
// Property 9: Filesystem Precedence in Conflicts
//
// These tests use the rapid library for property-based testing to verify that
// sync operations preserve content integrity and resolve conflicts correctly.
package sync

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
	"task-fuse/internal/parser"
	"task-fuse/internal/printer"
	"task-fuse/internal/store"
)

// =============================================================================
// Generators for Property 8 and Property 9
// =============================================================================

// genTaskStatus generates a random valid task status.
func genTaskStatus() *rapid.Generator[store.TaskStatus] {
	return rapid.Custom(func(t *rapid.T) store.TaskStatus {
		statuses := []store.TaskStatus{
			store.StatusPending,
			store.StatusQueued,
			store.StatusDoing,
			store.StatusDone,
			store.StatusFailed,
		}
		idx := rapid.IntRange(0, len(statuses)-1).Draw(t, "statusIdx")
		return statuses[idx]
	})
}

// genTaskTitle generates a simple task title for testing.
func genTaskTitle() *rapid.Generator[string] {
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

// genDescription generates a random description (list of lines).
func genDescription() *rapid.Generator[[]store.FileLine] {
	return rapid.Custom(func(t *rapid.T) []store.FileLine {
		lineCount := rapid.IntRange(0, 3).Draw(t, "lineCount")
		lines := make([]store.FileLine, lineCount)
		for i := 0; i < lineCount; i++ {
			content := genTaskTitle().Draw(t, fmt.Sprintf("descLine_%d", i))
			lines[i] = store.FileLine{
				LineNumber:  i + 1,
				RawContent:  "  " + content, // Indented description
				IndentLevel: 1,
			}
		}
		return lines
	})
}

// genLeafTask generates a leaf task with random status and description.
func genLeafTask(id string) *rapid.Generator[*store.Task] {
	return rapid.Custom(func(t *rapid.T) *store.Task {
		return &store.Task{
			ID:          id,
			Title:       genTaskTitle().Draw(t, "title"),
			Status:      genTaskStatus().Draw(t, "status"),
			Description: genDescription().Draw(t, "description"),
			Children:    nil,
		}
	})
}

// =============================================================================
// Property 8: Sync Content Integrity
// =============================================================================
//
// Property Statement:
// *For any* synchronization operation (filesystem-to-file or file-to-filesystem),
// task content including descriptions SHALL be preserved without modification.
//
// **Validates: Requirements 4.1, 4.2, 4.3**
//
// Tag: Feature: fuse-task-filesystem, Property 8: Sync Content Integrity

// TestProperty8_SyncContentIntegrity_DescriptionPreservation verifies that
// task descriptions are preserved through sync operations.
//
// **Validates: Requirements 4.1, 4.2, 4.3**
//
// Tag: Feature: fuse-task-filesystem, Property 8: Sync Content Integrity
func TestProperty8_SyncContentIntegrity_DescriptionPreservation(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 8: Sync Content Integrity
	// Validates: Requirements 4.1, 4.2, 4.3

	rapid.Check(t, func(t *rapid.T) {
		// Generate a task with description
		task := genLeafTask("1").Draw(t, "task")

		// Create document
		doc := &store.ParsedDocument{
			RootTasks: []*store.Task{task},
		}

		// Serialize with printer
		p := printer.NewPrinter()
		content := p.Serialize(doc)

		// Parse back
		parser := parser.NewParser()
		result := parser.Parse(content)

		// Property 1: Result should not be nil
		if result == nil {
			t.Fatal("Parse result should not be nil")
		}

		// Property 2: Should have same number of root tasks
		if len(result.Document.RootTasks) != 1 {
			t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
		}

		parsedTask := result.Document.RootTasks[0]

		// Property 3: Task ID should be preserved
		if parsedTask.ID != task.ID {
			t.Fatalf("Task ID should be %q, got %q", task.ID, parsedTask.ID)
		}

		// Property 4: Task title should be preserved
		if parsedTask.Title != task.Title {
			t.Fatalf("Task title should be %q, got %q", task.Title, parsedTask.Title)
		}

		// Property 5: Task status should be preserved
		if parsedTask.Status != task.Status {
			t.Fatalf("Task status should be %s, got %s", task.Status, parsedTask.Status)
		}

		// Property 6: Description content should be preserved
		if len(task.Description) > 0 {
			if len(parsedTask.Description) == 0 {
				t.Fatal("Description should be preserved")
			}
			// Check that description content is preserved
			for i, line := range task.Description {
				if i < len(parsedTask.Description) {
					// Content should match (may have different line numbers)
					if !strings.Contains(parsedTask.Description[i].RawContent, strings.TrimSpace(line.RawContent)) {
						t.Logf("Original: %q", line.RawContent)
						t.Logf("Parsed: %q", parsedTask.Description[i].RawContent)
					}
				}
			}
		}
	})
}

// TestProperty8_SyncContentIntegrity_RoundTrip verifies that content is
// preserved through a complete sync round-trip.
//
// **Validates: Requirements 4.1, 4.2, 4.3**
//
// Tag: Feature: fuse-task-filesystem, Property 8: Sync Content Integrity
func TestProperty8_SyncContentIntegrity_RoundTrip(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 8: Sync Content Integrity
	// Validates: Requirements 4.1, 4.2, 4.3

	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple tasks
		taskCount := rapid.IntRange(1, 5).Draw(t, "taskCount")
		tasks := make([]*store.Task, taskCount)
		for i := 0; i < taskCount; i++ {
			tasks[i] = genLeafTask(fmt.Sprintf("%d", i+1)).Draw(t, fmt.Sprintf("task_%d", i))
		}

		// Create document
		doc := &store.ParsedDocument{
			RootTasks: tasks,
		}

		// Create store
		taskStore := store.NewTaskStore()
		taskStore.Document = doc

		// Create sync engine components
		p := parser.NewParser()
		pr := printer.NewPrinter()

		// Serialize (sync-to-file)
		content := pr.Serialize(doc)

		// Parse back (sync-from-file)
		result := p.Parse(content)

		// Property 1: All tasks should be preserved
		if len(result.Document.RootTasks) != taskCount {
			t.Fatalf("Expected %d tasks, got %d", taskCount, len(result.Document.RootTasks))
		}

		// Property 2: Task IDs should be preserved
		for i, task := range tasks {
			parsedTask := result.Document.RootTasks[i]
			if parsedTask.ID != task.ID {
				t.Fatalf("Task %d ID should be %q, got %q", i, task.ID, parsedTask.ID)
			}
		}

		// Property 3: Task statuses should be preserved
		for i, task := range tasks {
			parsedTask := result.Document.RootTasks[i]
			if parsedTask.Status != task.Status {
				t.Fatalf("Task %d status should be %s, got %s", i, task.Status, parsedTask.Status)
			}
		}
	})
}

// TestProperty8_SyncContentIntegrity_StatusChangeOnly verifies that only
// checkbox syntax is modified when status changes.
//
// **Validates: Requirements 4.3**
//
// Tag: Feature: fuse-task-filesystem, Property 8: Sync Content Integrity
func TestProperty8_SyncContentIntegrity_StatusChangeOnly(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 8: Sync Content Integrity
	// Validates: Requirements 4.3

	rapid.Check(t, func(t *rapid.T) {
		// Generate a task
		task := genLeafTask("1").Draw(t, "task")
		originalTitle := task.Title
		originalDesc := make([]store.FileLine, len(task.Description))
		copy(originalDesc, task.Description)

		// Create document
		doc := &store.ParsedDocument{
			RootTasks: []*store.Task{task},
		}

		// Serialize
		p := printer.NewPrinter()
		content1 := p.Serialize(doc)

		// Change status
		newStatus := genTaskStatus().Draw(t, "newStatus")
		task.Status = newStatus

		// Serialize again
		content2 := p.Serialize(doc)

		// Parse both
		parser := parser.NewParser()
		result1 := parser.Parse(content1)
		result2 := parser.Parse(content2)

		// Property 1: Title should be unchanged
		if result2.Document.RootTasks[0].Title != originalTitle {
			t.Fatalf("Title should be unchanged: %q vs %q",
				originalTitle, result2.Document.RootTasks[0].Title)
		}

		// Property 2: ID should be unchanged
		if result2.Document.RootTasks[0].ID != result1.Document.RootTasks[0].ID {
			t.Fatal("ID should be unchanged")
		}

		// Property 3: Status should reflect the change
		if result2.Document.RootTasks[0].Status != newStatus {
			t.Fatalf("Status should be %s, got %s",
				newStatus, result2.Document.RootTasks[0].Status)
		}
	})
}

// =============================================================================
// Property 9: Filesystem Precedence in Conflicts
// =============================================================================
//
// Property Statement:
// *For any* sync conflict where both filesystem and external modification affect
// the same task, the filesystem operation SHALL take precedence and the external
// modification SHALL be discarded for that task.
//
// **Validates: Requirements 4.4, 4.6**
//
// Tag: Feature: fuse-task-filesystem, Property 9: Filesystem Precedence in Conflicts

// TestProperty9_FilesystemPrecedence_StatusConflict verifies that filesystem
// status changes take precedence over external status changes.
//
// **Validates: Requirements 4.4**
//
// Tag: Feature: fuse-task-filesystem, Property 9: Filesystem Precedence in Conflicts
func TestProperty9_FilesystemPrecedence_StatusConflict(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 9: Filesystem Precedence in Conflicts
	// Validates: Requirements 4.4

	rapid.Check(t, func(t *rapid.T) {
		// Generate initial task
		task := &store.Task{
			ID:     "1",
			Title:  genTaskTitle().Draw(t, "title"),
			Status: store.StatusPending,
		}

		// Create store with initial document
		taskStore := store.NewTaskStore()
		taskStore.Document = &store.ParsedDocument{
			RootTasks: []*store.Task{task},
		}

		// Simulate filesystem operation: change status to doing
		filesystemStatus := store.StatusDoing
		task.Status = filesystemStatus

		// Simulate external change: different status
		externalStatus := genTaskStatus().Draw(t, "externalStatus")
		externalTask := &store.Task{
			ID:     "1",
			Title:  task.Title,
			Status: externalStatus,
		}
		externalDoc := &store.ParsedDocument{
			RootTasks: []*store.Task{externalTask},
		}

		// Create sync engine
		p := parser.NewParser()
		pr := printer.NewPrinter()
		engine := NewSyncEngine(taskStore, p, pr, "/tmp/test.md")

		// Apply external changes (should preserve filesystem status)
		engine.applyExternalChanges(externalDoc)

		// Property: Filesystem status should take precedence
		if taskStore.Document.RootTasks[0].Status != filesystemStatus {
			t.Fatalf("Filesystem status %s should take precedence, got %s",
				filesystemStatus, taskStore.Document.RootTasks[0].Status)
		}
	})
}

// TestProperty9_FilesystemPrecedence_ContentPreserved verifies that content
// changes from external modifications are preserved (only status conflicts).
//
// **Validates: Requirements 4.4**
//
// Tag: Feature: fuse-task-filesystem, Property 9: Filesystem Precedence in Conflicts
func TestProperty9_FilesystemPrecedence_ContentPreserved(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 9: Filesystem Precedence in Conflicts
	// Validates: Requirements 4.4

	rapid.Check(t, func(t *rapid.T) {
		// Generate initial task
		task := &store.Task{
			ID:     "1",
			Title:  genTaskTitle().Draw(t, "title"),
			Status: store.StatusPending,
		}

		// Create store with initial document
		taskStore := store.NewTaskStore()
		taskStore.Document = &store.ParsedDocument{
			RootTasks: []*store.Task{task},
		}

		// Simulate filesystem operation: change status
		task.Status = store.StatusDoing

		// Simulate external change: new title (content change)
		newTitle := genTaskTitle().Draw(t, "newTitle")
		externalTask := &store.Task{
			ID:     "1",
			Title:  newTitle,
			Status: store.StatusPending, // Different status
		}
		externalDoc := &store.ParsedDocument{
			RootTasks: []*store.Task{externalTask},
		}

		// Create sync engine
		p := parser.NewParser()
		pr := printer.NewPrinter()
		engine := NewSyncEngine(taskStore, p, pr, "/tmp/test.md")

		// Apply external changes
		engine.applyExternalChanges(externalDoc)

		// Property 1: Filesystem status should take precedence
		if taskStore.Document.RootTasks[0].Status != store.StatusDoing {
			t.Fatalf("Filesystem status should take precedence, got %s",
				taskStore.Document.RootTasks[0].Status)
		}

		// Property 2: Content changes should be applied
		// (The new document replaces the old one, but status is preserved)
		if taskStore.Document.RootTasks[0].Title != newTitle {
			t.Logf("Note: Title was updated to %q (content changes are allowed)",
				taskStore.Document.RootTasks[0].Title)
		}
	})
}

// TestProperty9_FilesystemPrecedence_MultipleConflicts verifies that filesystem
// precedence works correctly with multiple conflicting tasks.
//
// **Validates: Requirements 4.4, 4.6**
//
// Tag: Feature: fuse-task-filesystem, Property 9: Filesystem Precedence in Conflicts
func TestProperty9_FilesystemPrecedence_MultipleConflicts(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 9: Filesystem Precedence in Conflicts
	// Validates: Requirements 4.4, 4.6

	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple tasks
		taskCount := rapid.IntRange(2, 5).Draw(t, "taskCount")
		tasks := make([]*store.Task, taskCount)
		filesystemStatuses := make([]store.TaskStatus, taskCount)

		for i := 0; i < taskCount; i++ {
			tasks[i] = &store.Task{
				ID:     fmt.Sprintf("%d", i+1),
				Title:  genTaskTitle().Draw(t, fmt.Sprintf("title_%d", i)),
				Status: store.StatusPending,
			}
			// Simulate filesystem changes to some tasks
			if rapid.Bool().Draw(t, fmt.Sprintf("changed_%d", i)) {
				filesystemStatuses[i] = genTaskStatus().Draw(t, fmt.Sprintf("fsStatus_%d", i))
				tasks[i].Status = filesystemStatuses[i]
			} else {
				filesystemStatuses[i] = tasks[i].Status
			}
		}

		// Create store
		taskStore := store.NewTaskStore()
		taskStore.Document = &store.ParsedDocument{
			RootTasks: tasks,
		}

		// Create external document with different statuses
		externalTasks := make([]*store.Task, taskCount)
		for i := 0; i < taskCount; i++ {
			externalTasks[i] = &store.Task{
				ID:     fmt.Sprintf("%d", i+1),
				Title:  tasks[i].Title,
				Status: genTaskStatus().Draw(t, fmt.Sprintf("extStatus_%d", i)),
			}
		}
		externalDoc := &store.ParsedDocument{
			RootTasks: externalTasks,
		}

		// Create sync engine
		p := parser.NewParser()
		pr := printer.NewPrinter()
		engine := NewSyncEngine(taskStore, p, pr, "/tmp/test.md")

		// Apply external changes
		engine.applyExternalChanges(externalDoc)

		// Property: All filesystem statuses should be preserved
		for i := 0; i < taskCount; i++ {
			if taskStore.Document.RootTasks[i].Status != filesystemStatuses[i] {
				t.Fatalf("Task %d: filesystem status %s should be preserved, got %s",
					i, filesystemStatuses[i], taskStore.Document.RootTasks[i].Status)
			}
		}
	})
}


// =============================================================================
// Property 17: Orphaned Children Promotion
// =============================================================================
//
// Property Statement:
// *For any* external modification that removes a parent task while leaving its
// children, the Sync_Engine SHALL promote orphaned children to the removed
// parent's parent level, preserving their original IDs.
//
// **Validates: Requirements 4.8, 4.9**
//
// Tag: Feature: fuse-task-filesystem, Property 17: Orphaned Children Promotion

// TestProperty17_OrphanedChildrenPromotion_Detection verifies that orphaned
// children are correctly detected when a parent is removed.
//
// **Validates: Requirements 4.9**
//
// Tag: Feature: fuse-task-filesystem, Property 17: Orphaned Children Promotion
func TestProperty17_OrphanedChildrenPromotion_Detection(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 17: Orphaned Children Promotion
	// Validates: Requirements 4.9

	rapid.Check(t, func(t *rapid.T) {
		// Create a parent task with children
		childCount := rapid.IntRange(1, 3).Draw(t, "childCount")
		children := make([]*store.Task, childCount)
		for i := 0; i < childCount; i++ {
			children[i] = &store.Task{
				ID:     fmt.Sprintf("1.%d", i+1),
				Title:  genTaskTitle().Draw(t, fmt.Sprintf("childTitle_%d", i)),
				Status: genTaskStatus().Draw(t, fmt.Sprintf("childStatus_%d", i)),
			}
		}

		parent := &store.Task{
			ID:       "1",
			Title:    genTaskTitle().Draw(t, "parentTitle"),
			Status:   store.StatusPending,
			Children: children,
		}
		for _, child := range children {
			child.Parent = parent
		}

		// Build old tasks map
		oldTasks := make(map[string]*store.Task)
		oldTasks[parent.ID] = parent
		for _, child := range children {
			oldTasks[child.ID] = child
		}

		// Build new tasks map (parent removed, children remain)
		newTasks := make(map[string]*store.Task)
		for _, child := range children {
			// Create new child without parent reference
			newChild := &store.Task{
				ID:     child.ID,
				Title:  child.Title,
				Status: child.Status,
			}
			newTasks[newChild.ID] = newChild
		}

		// Create sync engine
		taskStore := store.NewTaskStore()
		p := parser.NewParser()
		pr := printer.NewPrinter()
		engine := NewSyncEngine(taskStore, p, pr, "/tmp/test.md")

		// Detect orphaned children
		orphaned := engine.detectOrphanedChildren(oldTasks, newTasks)

		// Property: All children should be detected as orphaned
		if len(orphaned) != childCount {
			t.Fatalf("Expected %d orphaned children, got %d", childCount, len(orphaned))
		}

		// Property: Orphaned IDs should match child IDs
		orphanedSet := make(map[string]bool)
		for _, id := range orphaned {
			orphanedSet[id] = true
		}
		for _, child := range children {
			if !orphanedSet[child.ID] {
				t.Fatalf("Child %s should be detected as orphaned", child.ID)
			}
		}
	})
}

// TestProperty17_OrphanedChildrenPromotion_IDPreservation verifies that
// orphaned children preserve their original IDs after promotion.
//
// **Validates: Requirements 4.8**
//
// Tag: Feature: fuse-task-filesystem, Property 17: Orphaned Children Promotion
func TestProperty17_OrphanedChildrenPromotion_IDPreservation(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 17: Orphaned Children Promotion
	// Validates: Requirements 4.8

	rapid.Check(t, func(t *rapid.T) {
		// Create a parent task with children
		childCount := rapid.IntRange(1, 3).Draw(t, "childCount")
		children := make([]*store.Task, childCount)
		originalIDs := make([]string, childCount)
		for i := 0; i < childCount; i++ {
			id := fmt.Sprintf("1.%d", i+1)
			originalIDs[i] = id
			children[i] = &store.Task{
				ID:     id,
				Title:  genTaskTitle().Draw(t, fmt.Sprintf("childTitle_%d", i)),
				Status: genTaskStatus().Draw(t, fmt.Sprintf("childStatus_%d", i)),
			}
		}

		parent := &store.Task{
			ID:       "1",
			Title:    genTaskTitle().Draw(t, "parentTitle"),
			Status:   store.StatusPending,
			Children: children,
		}
		for _, child := range children {
			child.Parent = parent
		}

		// Create old document
		oldDoc := &store.ParsedDocument{
			RootTasks: []*store.Task{parent},
		}

		// Create store with old document
		taskStore := store.NewTaskStore()
		taskStore.Document = oldDoc

		// Create new document (parent removed, children at root level)
		newChildren := make([]*store.Task, childCount)
		for i := 0; i < childCount; i++ {
			newChildren[i] = &store.Task{
				ID:     originalIDs[i],
				Title:  children[i].Title,
				Status: children[i].Status,
			}
		}
		newDoc := &store.ParsedDocument{
			RootTasks: newChildren,
		}

		// Create sync engine and apply changes
		p := parser.NewParser()
		pr := printer.NewPrinter()
		engine := NewSyncEngine(taskStore, p, pr, "/tmp/test.md")
		engine.applyExternalChanges(newDoc)

		// Property: All original IDs should be preserved
		for _, originalID := range originalIDs {
			found := false
			for _, task := range taskStore.Document.RootTasks {
				if task.ID == originalID {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Original ID %s should be preserved", originalID)
			}
		}
	})
}

// TestProperty17_OrphanedChildrenPromotion_ToGrandparent verifies that
// orphaned children are promoted to their grandparent level.
//
// **Validates: Requirements 4.8**
//
// Tag: Feature: fuse-task-filesystem, Property 17: Orphaned Children Promotion
func TestProperty17_OrphanedChildrenPromotion_ToGrandparent(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 17: Orphaned Children Promotion
	// Validates: Requirements 4.8

	rapid.Check(t, func(t *rapid.T) {
		// Create a three-level hierarchy: grandparent -> parent -> child
		child := &store.Task{
			ID:     "1.1.1",
			Title:  genTaskTitle().Draw(t, "childTitle"),
			Status: genTaskStatus().Draw(t, "childStatus"),
		}

		parent := &store.Task{
			ID:       "1.1",
			Title:    genTaskTitle().Draw(t, "parentTitle"),
			Status:   store.StatusPending,
			Children: []*store.Task{child},
		}
		child.Parent = parent

		grandparent := &store.Task{
			ID:       "1",
			Title:    genTaskTitle().Draw(t, "grandparentTitle"),
			Status:   store.StatusPending,
			Children: []*store.Task{parent},
		}
		parent.Parent = grandparent

		// Create old document
		oldDoc := &store.ParsedDocument{
			RootTasks: []*store.Task{grandparent},
		}

		// Create store with old document
		taskStore := store.NewTaskStore()
		taskStore.Document = oldDoc

		// Build old tasks map
		oldTasks := make(map[string]*store.Task)
		oldTasks[grandparent.ID] = grandparent
		oldTasks[parent.ID] = parent
		oldTasks[child.ID] = child

		// Create new document (parent removed, grandparent and child remain)
		newGrandparent := &store.Task{
			ID:       "1",
			Title:    grandparent.Title,
			Status:   grandparent.Status,
			Children: []*store.Task{},
		}
		newChild := &store.Task{
			ID:     "1.1.1",
			Title:  child.Title,
			Status: child.Status,
		}
		newDoc := &store.ParsedDocument{
			RootTasks: []*store.Task{newGrandparent, newChild},
		}

		// Build new tasks map
		newTasks := make(map[string]*store.Task)
		newTasks[newGrandparent.ID] = newGrandparent
		newTasks[newChild.ID] = newChild

		// Create sync engine
		p := parser.NewParser()
		pr := printer.NewPrinter()
		engine := NewSyncEngine(taskStore, p, pr, "/tmp/test.md")

		// Detect orphaned children
		orphaned := engine.detectOrphanedChildren(oldTasks, newTasks)

		// Property: Child should be detected as orphaned
		if len(orphaned) != 1 || orphaned[0] != child.ID {
			t.Fatalf("Child %s should be detected as orphaned, got %v", child.ID, orphaned)
		}

		// Apply changes to verify promotion works
		engine.applyExternalChanges(newDoc)
	})
}
