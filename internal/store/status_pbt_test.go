// Package store provides property-based tests for parent status derivation.
//
// Feature: fuse-task-filesystem
// Property 10: Parent Status Derivation
//
// These tests use the rapid library for property-based testing to verify that
// parent task status is correctly derived from children following the precedence:
// doing > failed > pending > queued > done.
package store

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Property 10: Parent Status Derivation
// =============================================================================
//
// Property Statement:
// *For any* parent task, its status SHALL be derived from its children following
// the precedence: doing (if any child is doing) > failed (if any child is failed)
// > pending (if any child is pending) > queued (if any child is queued) > done
// (if all children are done).
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation

// =============================================================================
// Generators
// =============================================================================

// allStatuses contains all valid task statuses for generation.
var allStatuses = []TaskStatus{StatusPending, StatusQueued, StatusDoing, StatusDone, StatusFailed}

// statusPrecedence maps each status to its precedence level (higher = takes precedence).
// doing (4) > failed (3) > pending (2) > queued (1) > done (0)
var statusPrecedence = map[TaskStatus]int{
	StatusDone:    0,
	StatusQueued:  1,
	StatusPending: 2,
	StatusFailed:  3,
	StatusDoing:   4,
}

// genTaskStatus generates a random valid task status.
func genTaskStatus() *rapid.Generator[TaskStatus] {
	return rapid.Custom(func(t *rapid.T) TaskStatus {
		idx := rapid.IntRange(0, len(allStatuses)-1).Draw(t, "statusIdx")
		return allStatuses[idx]
	})
}

// genChildStatuses generates a slice of random child statuses (1-10 children).
func genChildStatuses() *rapid.Generator[[]TaskStatus] {
	return rapid.Custom(func(t *rapid.T) []TaskStatus {
		count := rapid.IntRange(1, 10).Draw(t, "childCount")
		statuses := make([]TaskStatus, count)
		for i := 0; i < count; i++ {
			statuses[i] = genTaskStatus().Draw(t, fmt.Sprintf("childStatus_%d", i))
		}
		return statuses
	})
}

// genParentTaskWithChildren generates a parent task with the given child statuses.
func genParentTaskWithChildren(id string, childStatuses []TaskStatus) *Task {
	children := make([]*Task, len(childStatuses))
	for i, status := range childStatuses {
		childID := fmt.Sprintf("%s.%d", id, i+1)
		children[i] = &Task{
			ID:       childID,
			Title:    fmt.Sprintf("Child Task %s", childID),
			Status:   status,
			Children: nil, // Leaf children
		}
	}
	parent := &Task{
		ID:       id,
		Title:    fmt.Sprintf("Parent Task %s", id),
		Status:   StatusPending, // Parent's own status is ignored
		Children: children,
	}
	// Set parent references
	for _, child := range children {
		child.Parent = parent
	}
	return parent
}

// genTaskTree generates a random task tree with varying depths (1-4 levels).
func genTaskTree() *rapid.Generator[*Task] {
	return rapid.Custom(func(t *rapid.T) *Task {
		depth := rapid.IntRange(1, 4).Draw(t, "treeDepth")
		return genTaskTreeAtDepth("1", depth, t)
	})
}

// genTaskTreeAtDepth recursively generates a task tree at the specified depth.
func genTaskTreeAtDepth(id string, remainingDepth int, t *rapid.T) *Task {
	if remainingDepth <= 1 {
		// Generate a leaf task
		status := genTaskStatus().Draw(t, fmt.Sprintf("leafStatus_%s", id))
		return &Task{
			ID:       id,
			Title:    fmt.Sprintf("Leaf %s", id),
			Status:   status,
			Children: nil,
		}
	}

	// Generate a parent task with children
	childCount := rapid.IntRange(1, 5).Draw(t, fmt.Sprintf("childCount_%s", id))
	children := make([]*Task, childCount)
	for i := 0; i < childCount; i++ {
		childID := fmt.Sprintf("%s.%d", id, i+1)
		// Randomly decide if child is a leaf or another parent
		isLeaf := rapid.Bool().Draw(t, fmt.Sprintf("isLeaf_%s", childID))
		if isLeaf || remainingDepth <= 2 {
			status := genTaskStatus().Draw(t, fmt.Sprintf("childStatus_%s", childID))
			children[i] = &Task{
				ID:       childID,
				Title:    fmt.Sprintf("Child %s", childID),
				Status:   status,
				Children: nil,
			}
		} else {
			children[i] = genTaskTreeAtDepth(childID, remainingDepth-1, t)
		}
	}

	parent := &Task{
		ID:       id,
		Title:    fmt.Sprintf("Parent %s", id),
		Status:   StatusPending, // Parent's own status is ignored
		Children: children,
	}
	// Set parent references
	for _, child := range children {
		child.Parent = parent
	}
	return parent
}

// =============================================================================
// Helper Functions
// =============================================================================

// expectedParentStatus calculates the expected parent status based on child statuses
// using the precedence rules: doing > failed > pending > queued > done.
func expectedParentStatus(childStatuses []TaskStatus) TaskStatus {
	if len(childStatuses) == 0 {
		return StatusPending // Default for no children
	}

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

	// Apply precedence: doing > failed > pending > queued > done
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

// collectLeafStatuses recursively collects all leaf task statuses from a task tree.
func collectLeafStatuses(task *Task) []TaskStatus {
	if task == nil {
		return nil
	}
	if len(task.Children) == 0 {
		return []TaskStatus{task.Status}
	}
	var statuses []TaskStatus
	for _, child := range task.Children {
		statuses = append(statuses, collectLeafStatuses(child)...)
	}
	return statuses
}

// highestPrecedenceStatus returns the status with highest precedence from a slice.
func highestPrecedenceStatus(statuses []TaskStatus) TaskStatus {
	if len(statuses) == 0 {
		return StatusPending
	}

	highest := statuses[0]
	for _, s := range statuses[1:] {
		if statusPrecedence[s] > statusPrecedence[highest] {
			highest = s
		}
	}
	return highest
}

// =============================================================================
// Property Tests
// =============================================================================

// TestProperty10_ParentStatusDerivation is the main property-based test for
// Property 10: Parent Status Derivation.
//
// **Validates: Requirements 4.7**
//
// This test generates random child status combinations and verifies that
// the parent status follows the precedence rules.
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate random child statuses
		childStatuses := genChildStatuses().Draw(t, "childStatuses")

		// Create parent task with these children
		parent := genParentTaskWithChildren("1", childStatuses)

		// Derive parent status
		derivedStatus := DeriveParentStatus(parent)

		// Calculate expected status
		expected := expectedParentStatus(childStatuses)

		// Property: Derived status should match expected based on precedence rules
		if derivedStatus != expected {
			t.Fatalf("DeriveParentStatus with children %v = %q, expected %q",
				childStatuses, derivedStatus, expected)
		}
	})
}

// TestProperty10_ParentStatusDerivation_PrecedenceOrder verifies the exact
// precedence order: doing > failed > pending > queued > done.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_PrecedenceOrder(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate two different statuses
		status1 := genTaskStatus().Draw(t, "status1")
		status2 := genTaskStatus().Draw(t, "status2")

		// Create parent with both statuses as children
		parent := genParentTaskWithChildren("1", []TaskStatus{status1, status2})

		// Derive parent status
		derivedStatus := DeriveParentStatus(parent)

		// Property: Derived status should be the one with higher precedence
		expectedStatus := highestPrecedenceStatus([]TaskStatus{status1, status2})
		if derivedStatus != expectedStatus {
			t.Fatalf("DeriveParentStatus with children [%s, %s] = %q, expected %q (higher precedence)",
				status1, status2, derivedStatus, expectedStatus)
		}
	})
}

// TestProperty10_ParentStatusDerivation_DoingTakesPrecedence verifies that
// doing status takes precedence over all other statuses.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_DoingTakesPrecedence(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate random child statuses
		childStatuses := genChildStatuses().Draw(t, "childStatuses")

		// Add at least one doing status
		childStatuses = append(childStatuses, StatusDoing)

		// Create parent task
		parent := genParentTaskWithChildren("1", childStatuses)

		// Derive parent status
		derivedStatus := DeriveParentStatus(parent)

		// Property: If any child is doing, parent should be doing
		if derivedStatus != StatusDoing {
			t.Fatalf("DeriveParentStatus with doing child = %q, expected %q",
				derivedStatus, StatusDoing)
		}
	})
}

// TestProperty10_ParentStatusDerivation_FailedPrecedence verifies that
// failed status takes precedence over pending, queued, and done.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_FailedPrecedence(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate statuses that don't include doing
		nonDoingStatuses := []TaskStatus{StatusPending, StatusQueued, StatusDone, StatusFailed}
		count := rapid.IntRange(1, 5).Draw(t, "count")
		childStatuses := make([]TaskStatus, count)
		for i := 0; i < count; i++ {
			idx := rapid.IntRange(0, len(nonDoingStatuses)-1).Draw(t, fmt.Sprintf("idx_%d", i))
			childStatuses[i] = nonDoingStatuses[idx]
		}

		// Add at least one failed status
		childStatuses = append(childStatuses, StatusFailed)

		// Create parent task
		parent := genParentTaskWithChildren("1", childStatuses)

		// Derive parent status
		derivedStatus := DeriveParentStatus(parent)

		// Property: If no doing but has failed, parent should be failed
		if derivedStatus != StatusFailed {
			t.Fatalf("DeriveParentStatus with failed child (no doing) = %q, expected %q",
				derivedStatus, StatusFailed)
		}
	})
}

// TestProperty10_ParentStatusDerivation_PendingPrecedence verifies that
// pending status takes precedence over queued and done.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_PendingPrecedence(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate statuses that don't include doing or failed
		lowerStatuses := []TaskStatus{StatusPending, StatusQueued, StatusDone}
		count := rapid.IntRange(1, 5).Draw(t, "count")
		childStatuses := make([]TaskStatus, count)
		for i := 0; i < count; i++ {
			idx := rapid.IntRange(0, len(lowerStatuses)-1).Draw(t, fmt.Sprintf("idx_%d", i))
			childStatuses[i] = lowerStatuses[idx]
		}

		// Add at least one pending status
		childStatuses = append(childStatuses, StatusPending)

		// Create parent task
		parent := genParentTaskWithChildren("1", childStatuses)

		// Derive parent status
		derivedStatus := DeriveParentStatus(parent)

		// Property: If no doing/failed but has pending, parent should be pending
		if derivedStatus != StatusPending {
			t.Fatalf("DeriveParentStatus with pending child (no doing/failed) = %q, expected %q",
				derivedStatus, StatusPending)
		}
	})
}

// TestProperty10_ParentStatusDerivation_QueuedPrecedence verifies that
// queued status takes precedence over done.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_QueuedPrecedence(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate statuses that are only queued or done
		lowestStatuses := []TaskStatus{StatusQueued, StatusDone}
		count := rapid.IntRange(1, 5).Draw(t, "count")
		childStatuses := make([]TaskStatus, count)
		for i := 0; i < count; i++ {
			idx := rapid.IntRange(0, len(lowestStatuses)-1).Draw(t, fmt.Sprintf("idx_%d", i))
			childStatuses[i] = lowestStatuses[idx]
		}

		// Add at least one queued status
		childStatuses = append(childStatuses, StatusQueued)

		// Create parent task
		parent := genParentTaskWithChildren("1", childStatuses)

		// Derive parent status
		derivedStatus := DeriveParentStatus(parent)

		// Property: If only queued/done and has queued, parent should be queued
		if derivedStatus != StatusQueued {
			t.Fatalf("DeriveParentStatus with queued child (only queued/done) = %q, expected %q",
				derivedStatus, StatusQueued)
		}
	})
}

// TestProperty10_ParentStatusDerivation_AllDone verifies that
// if all children are done, parent is done.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_AllDone(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate 1-10 done children
		count := rapid.IntRange(1, 10).Draw(t, "count")
		childStatuses := make([]TaskStatus, count)
		for i := 0; i < count; i++ {
			childStatuses[i] = StatusDone
		}

		// Create parent task
		parent := genParentTaskWithChildren("1", childStatuses)

		// Derive parent status
		derivedStatus := DeriveParentStatus(parent)

		// Property: If all children are done, parent should be done
		if derivedStatus != StatusDone {
			t.Fatalf("DeriveParentStatus with all done children = %q, expected %q",
				derivedStatus, StatusDone)
		}
	})
}

// TestProperty10_ParentStatusDerivation_RecursiveDerivation verifies that
// nested parent tasks derive status from their children recursively.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_RecursiveDerivation(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random task tree with depth 2-4
		tree := genTaskTree().Draw(t, "taskTree")

		// Derive status
		derivedStatus := DeriveParentStatus(tree)

		// Collect all leaf statuses
		leafStatuses := collectLeafStatuses(tree)

		// Calculate expected status from leaf statuses
		expected := expectedParentStatus(leafStatuses)

		// Property: Derived status should match expected based on all leaf statuses
		if derivedStatus != expected {
			t.Fatalf("DeriveParentStatus for tree with leaf statuses %v = %q, expected %q",
				leafStatuses, derivedStatus, expected)
		}
	})
}

// TestProperty10_ParentStatusDerivation_SingleChildDominance verifies that
// a single child with higher-precedence status determines parent status.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_SingleChildDominance(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate many done children
		doneCount := rapid.IntRange(5, 10).Draw(t, "doneCount")
		childStatuses := make([]TaskStatus, doneCount)
		for i := 0; i < doneCount; i++ {
			childStatuses[i] = StatusDone
		}

		// Add one child with a higher-precedence status
		dominantStatus := genTaskStatus().Draw(t, "dominantStatus")
		childStatuses = append(childStatuses, dominantStatus)

		// Create parent task
		parent := genParentTaskWithChildren("1", childStatuses)

		// Derive parent status
		derivedStatus := DeriveParentStatus(parent)

		// Property: Single dominant status should determine parent status
		expected := expectedParentStatus(childStatuses)
		if derivedStatus != expected {
			t.Fatalf("DeriveParentStatus with dominant %s among done = %q, expected %q",
				dominantStatus, derivedStatus, expected)
		}
	})
}

// TestProperty10_ParentStatusDerivation_LeafTaskPassthrough verifies that
// leaf tasks return their own status.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_LeafTaskPassthrough(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate a leaf task with random status
		status := genTaskStatus().Draw(t, "leafStatus")
		leaf := &Task{
			ID:       "1",
			Title:    "Leaf Task",
			Status:   status,
			Children: nil, // No children = leaf task
		}

		// Derive status
		derivedStatus := DeriveParentStatus(leaf)

		// Property: Leaf task should return its own status
		if derivedStatus != status {
			t.Fatalf("DeriveParentStatus for leaf with status %q = %q, expected same status",
				status, derivedStatus)
		}
	})
}

// TestProperty10_ParentStatusDerivation_ParentOwnStatusIgnored verifies that
// a parent's own status field is ignored when deriving status.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_ParentOwnStatusIgnored(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate child statuses
		childStatuses := genChildStatuses().Draw(t, "childStatuses")

		// Generate a random parent status (should be ignored)
		parentOwnStatus := genTaskStatus().Draw(t, "parentOwnStatus")

		// Create parent with explicit status
		children := make([]*Task, len(childStatuses))
		for i, status := range childStatuses {
			children[i] = &Task{
				ID:     fmt.Sprintf("1.%d", i+1),
				Title:  fmt.Sprintf("Child %d", i+1),
				Status: status,
			}
		}
		parent := &Task{
			ID:       "1",
			Title:    "Parent",
			Status:   parentOwnStatus, // This should be ignored
			Children: children,
		}

		// Derive status
		derivedStatus := DeriveParentStatus(parent)

		// Calculate expected from children only
		expected := expectedParentStatus(childStatuses)

		// Property: Parent's own status should be ignored
		if derivedStatus != expected {
			t.Fatalf("DeriveParentStatus (parent status=%s, children=%v) = %q, expected %q (parent status should be ignored)",
				parentOwnStatus, childStatuses, derivedStatus, expected)
		}
	})
}

// TestProperty10_ParentStatusDerivation_DeeplyNestedTree verifies status
// derivation for deeply nested task trees (up to 4 levels).
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_DeeplyNestedTree(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate a deeply nested tree
		tree := genTaskTree().Draw(t, "deepTree")

		// Derive status
		derivedStatus := DeriveParentStatus(tree)

		// Collect all leaf statuses
		leafStatuses := collectLeafStatuses(tree)

		// Calculate expected status
		expected := expectedParentStatus(leafStatuses)

		// Property: Derived status should match expected from all leaves
		if derivedStatus != expected {
			t.Fatalf("DeriveParentStatus for deep tree with leaves %v = %q, expected %q",
				leafStatuses, derivedStatus, expected)
		}
	})
}

// TestProperty10_ParentStatusDerivation_MixedNestedStatuses verifies that
// complex nested structures with mixed statuses derive correctly.
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_MixedNestedStatuses(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple subtrees with different dominant statuses
		subtreeCount := rapid.IntRange(2, 4).Draw(t, "subtreeCount")
		children := make([]*Task, subtreeCount)

		for i := 0; i < subtreeCount; i++ {
			// Each subtree has a dominant status
			dominantStatus := genTaskStatus().Draw(t, fmt.Sprintf("dominant_%d", i))
			leafCount := rapid.IntRange(1, 3).Draw(t, fmt.Sprintf("leafCount_%d", i))

			grandchildren := make([]*Task, leafCount)
			for j := 0; j < leafCount; j++ {
				// Mix of dominant status and done
				var status TaskStatus
				if j == 0 {
					status = dominantStatus
				} else {
					status = StatusDone
				}
				grandchildren[j] = &Task{
					ID:     fmt.Sprintf("1.%d.%d", i+1, j+1),
					Title:  fmt.Sprintf("Grandchild %d.%d", i+1, j+1),
					Status: status,
				}
			}

			children[i] = &Task{
				ID:       fmt.Sprintf("1.%d", i+1),
				Title:    fmt.Sprintf("Child %d", i+1),
				Children: grandchildren,
			}
		}

		parent := &Task{
			ID:       "1",
			Title:    "Root",
			Children: children,
		}

		// Derive status
		derivedStatus := DeriveParentStatus(parent)

		// Collect all leaf statuses
		leafStatuses := collectLeafStatuses(parent)

		// Calculate expected status
		expected := expectedParentStatus(leafStatuses)

		// Property: Derived status should match expected from all leaves
		if derivedStatus != expected {
			t.Fatalf("DeriveParentStatus for mixed nested tree = %q, expected %q (leaves: %v)",
				derivedStatus, expected, leafStatuses)
		}
	})
}

// TestProperty10_ParentStatusDerivation_NilTask verifies that nil task
// returns pending status (safe default).
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_NilTask(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	result := DeriveParentStatus(nil)
	if result != StatusPending {
		t.Fatalf("DeriveParentStatus(nil) = %q, expected %q", result, StatusPending)
	}
}

// TestProperty10_ParentStatusDerivation_EmptyChildren verifies that a task
// with empty children slice returns its own status (treated as leaf).
//
// **Validates: Requirements 4.7**
//
// Tag: Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
func TestProperty10_ParentStatusDerivation_EmptyChildren(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 10: Parent Status Derivation
	// Validates: Requirements 4.7

	rapid.Check(t, func(t *rapid.T) {
		status := genTaskStatus().Draw(t, "status")
		task := &Task{
			ID:       "1",
			Title:    "Task with empty children",
			Status:   status,
			Children: []*Task{}, // Empty slice, not nil
		}

		derivedStatus := DeriveParentStatus(task)

		// Property: Empty children = leaf task, returns own status
		if derivedStatus != status {
			t.Fatalf("DeriveParentStatus for task with empty children (status=%s) = %q, expected %q",
				status, derivedStatus, status)
		}
	})
}
