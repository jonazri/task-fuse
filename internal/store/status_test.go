package store

import (
	"testing"
)

// TestDeriveParentStatus_NilTask tests that nil task returns pending status.
func TestDeriveParentStatus_NilTask(t *testing.T) {
	result := DeriveParentStatus(nil)
	if result != StatusPending {
		t.Errorf("DeriveParentStatus(nil) = %q, want %q", result, StatusPending)
	}
}

// TestDeriveParentStatus_LeafTask tests that leaf tasks return their own status.
func TestDeriveParentStatus_LeafTask(t *testing.T) {
	tests := []struct {
		name     string
		status   TaskStatus
		expected TaskStatus
	}{
		{
			name:     "pending leaf",
			status:   StatusPending,
			expected: StatusPending,
		},
		{
			name:     "queued leaf",
			status:   StatusQueued,
			expected: StatusQueued,
		},
		{
			name:     "doing leaf",
			status:   StatusDoing,
			expected: StatusDoing,
		},
		{
			name:     "done leaf",
			status:   StatusDone,
			expected: StatusDone,
		},
		{
			name:     "failed leaf",
			status:   StatusFailed,
			expected: StatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				ID:       "1",
				Title:    "Test Task",
				Status:   tt.status,
				Children: nil, // Leaf task - no children
			}
			result := DeriveParentStatus(task)
			if result != tt.expected {
				t.Errorf("DeriveParentStatus(leaf with status %q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

// TestDeriveParentStatus_EmptyChildren tests that task with empty children slice returns its own status.
func TestDeriveParentStatus_EmptyChildren(t *testing.T) {
	task := &Task{
		ID:       "1",
		Title:    "Test Task",
		Status:   StatusDoing,
		Children: []*Task{}, // Empty children slice
	}
	result := DeriveParentStatus(task)
	if result != StatusDoing {
		t.Errorf("DeriveParentStatus(task with empty children) = %q, want %q", result, StatusDoing)
	}
}

// TestDeriveParentStatus_AllChildrenDone tests that parent with all done children is done.
func TestDeriveParentStatus_AllChildrenDone(t *testing.T) {
	task := &Task{
		ID:     "1",
		Title:  "Parent Task",
		Status: StatusPending, // Parent's own status is ignored
		Children: []*Task{
			{ID: "1.1", Title: "Child 1", Status: StatusDone},
			{ID: "1.2", Title: "Child 2", Status: StatusDone},
			{ID: "1.3", Title: "Child 3", Status: StatusDone},
		},
	}
	result := DeriveParentStatus(task)
	if result != StatusDone {
		t.Errorf("DeriveParentStatus(all children done) = %q, want %q", result, StatusDone)
	}
}

// TestDeriveParentStatus_Precedence_Doing tests that doing has highest precedence.
func TestDeriveParentStatus_Precedence_Doing(t *testing.T) {
	tests := []struct {
		name     string
		children []*Task
	}{
		{
			name: "doing with done",
			children: []*Task{
				{ID: "1.1", Status: StatusDoing},
				{ID: "1.2", Status: StatusDone},
			},
		},
		{
			name: "doing with failed",
			children: []*Task{
				{ID: "1.1", Status: StatusDoing},
				{ID: "1.2", Status: StatusFailed},
			},
		},
		{
			name: "doing with pending",
			children: []*Task{
				{ID: "1.1", Status: StatusDoing},
				{ID: "1.2", Status: StatusPending},
			},
		},
		{
			name: "doing with queued",
			children: []*Task{
				{ID: "1.1", Status: StatusDoing},
				{ID: "1.2", Status: StatusQueued},
			},
		},
		{
			name: "doing with all statuses",
			children: []*Task{
				{ID: "1.1", Status: StatusDoing},
				{ID: "1.2", Status: StatusFailed},
				{ID: "1.3", Status: StatusPending},
				{ID: "1.4", Status: StatusQueued},
				{ID: "1.5", Status: StatusDone},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				ID:       "1",
				Title:    "Parent Task",
				Children: tt.children,
			}
			result := DeriveParentStatus(task)
			if result != StatusDoing {
				t.Errorf("DeriveParentStatus(%s) = %q, want %q", tt.name, result, StatusDoing)
			}
		})
	}
}

// TestDeriveParentStatus_Precedence_Failed tests that failed has second highest precedence.
func TestDeriveParentStatus_Precedence_Failed(t *testing.T) {
	tests := []struct {
		name     string
		children []*Task
	}{
		{
			name: "failed with done",
			children: []*Task{
				{ID: "1.1", Status: StatusFailed},
				{ID: "1.2", Status: StatusDone},
			},
		},
		{
			name: "failed with pending",
			children: []*Task{
				{ID: "1.1", Status: StatusFailed},
				{ID: "1.2", Status: StatusPending},
			},
		},
		{
			name: "failed with queued",
			children: []*Task{
				{ID: "1.1", Status: StatusFailed},
				{ID: "1.2", Status: StatusQueued},
			},
		},
		{
			name: "failed with pending, queued, done",
			children: []*Task{
				{ID: "1.1", Status: StatusFailed},
				{ID: "1.2", Status: StatusPending},
				{ID: "1.3", Status: StatusQueued},
				{ID: "1.4", Status: StatusDone},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				ID:       "1",
				Title:    "Parent Task",
				Children: tt.children,
			}
			result := DeriveParentStatus(task)
			if result != StatusFailed {
				t.Errorf("DeriveParentStatus(%s) = %q, want %q", tt.name, result, StatusFailed)
			}
		})
	}
}

// TestDeriveParentStatus_Precedence_Pending tests that pending has third highest precedence.
func TestDeriveParentStatus_Precedence_Pending(t *testing.T) {
	tests := []struct {
		name     string
		children []*Task
	}{
		{
			name: "pending with done",
			children: []*Task{
				{ID: "1.1", Status: StatusPending},
				{ID: "1.2", Status: StatusDone},
			},
		},
		{
			name: "pending with queued",
			children: []*Task{
				{ID: "1.1", Status: StatusPending},
				{ID: "1.2", Status: StatusQueued},
			},
		},
		{
			name: "pending with queued and done",
			children: []*Task{
				{ID: "1.1", Status: StatusPending},
				{ID: "1.2", Status: StatusQueued},
				{ID: "1.3", Status: StatusDone},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				ID:       "1",
				Title:    "Parent Task",
				Children: tt.children,
			}
			result := DeriveParentStatus(task)
			if result != StatusPending {
				t.Errorf("DeriveParentStatus(%s) = %q, want %q", tt.name, result, StatusPending)
			}
		})
	}
}

// TestDeriveParentStatus_Precedence_Queued tests that queued has fourth highest precedence.
func TestDeriveParentStatus_Precedence_Queued(t *testing.T) {
	tests := []struct {
		name     string
		children []*Task
	}{
		{
			name: "queued with done",
			children: []*Task{
				{ID: "1.1", Status: StatusQueued},
				{ID: "1.2", Status: StatusDone},
			},
		},
		{
			name: "queued only",
			children: []*Task{
				{ID: "1.1", Status: StatusQueued},
			},
		},
		{
			name: "multiple queued with done",
			children: []*Task{
				{ID: "1.1", Status: StatusQueued},
				{ID: "1.2", Status: StatusQueued},
				{ID: "1.3", Status: StatusDone},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				ID:       "1",
				Title:    "Parent Task",
				Children: tt.children,
			}
			result := DeriveParentStatus(task)
			if result != StatusQueued {
				t.Errorf("DeriveParentStatus(%s) = %q, want %q", tt.name, result, StatusQueued)
			}
		})
	}
}

// TestDeriveParentStatus_NestedChildren tests recursive derivation for nested parents.
func TestDeriveParentStatus_NestedChildren(t *testing.T) {
	// Create a nested structure:
	// 1 (parent)
	//   1.1 (parent)
	//     1.1.1 (leaf, doing)
	//     1.1.2 (leaf, done)
	//   1.2 (leaf, done)
	// Expected: 1.1 derives to "doing", so 1 derives to "doing"

	task := &Task{
		ID:    "1",
		Title: "Root Parent",
		Children: []*Task{
			{
				ID:    "1.1",
				Title: "Nested Parent",
				Children: []*Task{
					{ID: "1.1.1", Title: "Grandchild 1", Status: StatusDoing},
					{ID: "1.1.2", Title: "Grandchild 2", Status: StatusDone},
				},
			},
			{
				ID:     "1.2",
				Title:  "Child 2",
				Status: StatusDone,
			},
		},
	}

	result := DeriveParentStatus(task)
	if result != StatusDoing {
		t.Errorf("DeriveParentStatus(nested with doing grandchild) = %q, want %q", result, StatusDoing)
	}
}

// TestDeriveParentStatus_DeeplyNested tests deeply nested task hierarchies.
func TestDeriveParentStatus_DeeplyNested(t *testing.T) {
	// Create a deeply nested structure:
	// 1 (parent)
	//   1.1 (parent)
	//     1.1.1 (parent)
	//       1.1.1.1 (leaf, failed)
	// Expected: all parents derive to "failed"

	task := &Task{
		ID:    "1",
		Title: "Level 1",
		Children: []*Task{
			{
				ID:    "1.1",
				Title: "Level 2",
				Children: []*Task{
					{
						ID:    "1.1.1",
						Title: "Level 3",
						Children: []*Task{
							{ID: "1.1.1.1", Title: "Level 4", Status: StatusFailed},
						},
					},
				},
			},
		},
	}

	result := DeriveParentStatus(task)
	if result != StatusFailed {
		t.Errorf("DeriveParentStatus(deeply nested with failed leaf) = %q, want %q", result, StatusFailed)
	}
}

// TestDeriveParentStatus_MixedNestedStatuses tests complex nested status combinations.
func TestDeriveParentStatus_MixedNestedStatuses(t *testing.T) {
	// Create a structure where nested parents have different derived statuses:
	// 1 (parent)
	//   1.1 (parent) -> derives to "pending" (has pending child)
	//     1.1.1 (leaf, pending)
	//     1.1.2 (leaf, done)
	//   1.2 (parent) -> derives to "done" (all children done)
	//     1.2.1 (leaf, done)
	//     1.2.2 (leaf, done)
	// Expected: 1 derives to "pending" (1.1 is pending)

	task := &Task{
		ID:    "1",
		Title: "Root",
		Children: []*Task{
			{
				ID:    "1.1",
				Title: "Parent with pending",
				Children: []*Task{
					{ID: "1.1.1", Title: "Pending child", Status: StatusPending},
					{ID: "1.1.2", Title: "Done child", Status: StatusDone},
				},
			},
			{
				ID:    "1.2",
				Title: "Parent all done",
				Children: []*Task{
					{ID: "1.2.1", Title: "Done child 1", Status: StatusDone},
					{ID: "1.2.2", Title: "Done child 2", Status: StatusDone},
				},
			},
		},
	}

	result := DeriveParentStatus(task)
	if result != StatusPending {
		t.Errorf("DeriveParentStatus(mixed nested) = %q, want %q", result, StatusPending)
	}
}

// TestDeriveParentStatus_SingleChild tests parent with single child.
func TestDeriveParentStatus_SingleChild(t *testing.T) {
	tests := []struct {
		name        string
		childStatus TaskStatus
		expected    TaskStatus
	}{
		{
			name:        "single pending child",
			childStatus: StatusPending,
			expected:    StatusPending,
		},
		{
			name:        "single queued child",
			childStatus: StatusQueued,
			expected:    StatusQueued,
		},
		{
			name:        "single doing child",
			childStatus: StatusDoing,
			expected:    StatusDoing,
		},
		{
			name:        "single done child",
			childStatus: StatusDone,
			expected:    StatusDone,
		},
		{
			name:        "single failed child",
			childStatus: StatusFailed,
			expected:    StatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{
				ID:    "1",
				Title: "Parent",
				Children: []*Task{
					{ID: "1.1", Title: "Only Child", Status: tt.childStatus},
				},
			}
			result := DeriveParentStatus(task)
			if result != tt.expected {
				t.Errorf("DeriveParentStatus(%s) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

// TestDeriveParentStatus_DesignExample tests the example from the design document.
// From design.md Example 1:
// Task 1 has children: 1.1 (done), 1.2 (doing), 1.3 (pending)
// Child 1.2 is `doing` → Parent 1 derives status `doing`
func TestDeriveParentStatus_DesignExample(t *testing.T) {
	task := &Task{
		ID:    "1",
		Title: "Setup project",
		Children: []*Task{
			{ID: "1.1", Title: "Create directory structure", Status: StatusDone},
			{ID: "1.2", Title: "Configure build system", Status: StatusDoing},
			{ID: "1.3", Title: "Add dependencies", Status: StatusPending},
		},
	}

	result := DeriveParentStatus(task)
	if result != StatusDoing {
		t.Errorf("DeriveParentStatus(design example) = %q, want %q", result, StatusDoing)
	}
}

// TestDeriveParentStatus_ParentOwnStatusIgnored tests that parent's own status is ignored.
func TestDeriveParentStatus_ParentOwnStatusIgnored(t *testing.T) {
	// Parent has status "done" but children are not all done
	// The parent's own status should be ignored
	task := &Task{
		ID:     "1",
		Title:  "Parent",
		Status: StatusDone, // This should be ignored
		Children: []*Task{
			{ID: "1.1", Title: "Child 1", Status: StatusPending},
			{ID: "1.2", Title: "Child 2", Status: StatusDone},
		},
	}

	result := DeriveParentStatus(task)
	if result != StatusPending {
		t.Errorf("DeriveParentStatus(parent status should be ignored) = %q, want %q", result, StatusPending)
	}
}

// TestDeriveParentStatus_AllStatusCombinations tests all possible two-child status combinations.
func TestDeriveParentStatus_AllStatusCombinations(t *testing.T) {
	statuses := []TaskStatus{StatusPending, StatusQueued, StatusDoing, StatusDone, StatusFailed}

	// Expected results based on precedence: doing > failed > pending > queued > done
	expectedResult := func(s1, s2 TaskStatus) TaskStatus {
		if s1 == StatusDoing || s2 == StatusDoing {
			return StatusDoing
		}
		if s1 == StatusFailed || s2 == StatusFailed {
			return StatusFailed
		}
		if s1 == StatusPending || s2 == StatusPending {
			return StatusPending
		}
		if s1 == StatusQueued || s2 == StatusQueued {
			return StatusQueued
		}
		return StatusDone
	}

	for _, s1 := range statuses {
		for _, s2 := range statuses {
			t.Run(string(s1)+"_"+string(s2), func(t *testing.T) {
				task := &Task{
					ID:    "1",
					Title: "Parent",
					Children: []*Task{
						{ID: "1.1", Title: "Child 1", Status: s1},
						{ID: "1.2", Title: "Child 2", Status: s2},
					},
				}
				expected := expectedResult(s1, s2)
				result := DeriveParentStatus(task)
				if result != expected {
					t.Errorf("DeriveParentStatus(children: %q, %q) = %q, want %q", s1, s2, result, expected)
				}
			})
		}
	}
}

// TestDeriveParentStatus_ManyChildren tests parent with many children.
func TestDeriveParentStatus_ManyChildren(t *testing.T) {
	// Create a parent with 10 done children and 1 doing child
	children := make([]*Task, 11)
	for i := 0; i < 10; i++ {
		children[i] = &Task{
			ID:     "1." + string(rune('1'+i)),
			Title:  "Done Child",
			Status: StatusDone,
		}
	}
	children[10] = &Task{
		ID:     "1.11",
		Title:  "Doing Child",
		Status: StatusDoing,
	}

	task := &Task{
		ID:       "1",
		Title:    "Parent with many children",
		Children: children,
	}

	result := DeriveParentStatus(task)
	if result != StatusDoing {
		t.Errorf("DeriveParentStatus(many children with one doing) = %q, want %q", result, StatusDoing)
	}
}


// =============================================================================
// PropagateStatusChange Tests
// =============================================================================

// TestPropagateStatusChange_NilInputs tests that nil inputs are handled gracefully.
func TestPropagateStatusChange_NilInputs(t *testing.T) {
	// Should not panic with nil inputs
	PropagateStatusChange(nil, nil, nil)

	store := NewTaskStore()
	pm := NewPathManager(store)

	// Should not panic with nil task
	PropagateStatusChange(nil, store, pm)

	task := &Task{ID: "1", Title: "Test", Status: StatusPending}

	// Should not panic with nil store
	PropagateStatusChange(task, nil, pm)

	// Should not panic with nil PathManager
	PropagateStatusChange(task, store, nil)
}

// TestPropagateStatusChange_LeafTask tests propagation for a leaf task.
func TestPropagateStatusChange_LeafTask(t *testing.T) {
	store := NewTaskStore()
	store.Document = &ParsedDocument{
		RootTasks: []*Task{
			{ID: "1", Title: "Leaf Task", Status: StatusPending},
		},
	}
	pm := NewPathManager(store)

	// Build initial paths
	pm.BuildAllPaths()

	// Verify initial path
	task := store.Document.RootTasks[0]
	initialPath := store.PathByTask[task]
	if initialPath != "/pending/1.leaf_task.md" {
		t.Errorf("Initial path = %q, want %q", initialPath, "/pending/1.leaf_task.md")
	}

	// Change status and propagate
	task.Status = StatusDoing
	PropagateStatusChange(task, store, pm)

	// Verify path updated
	newPath := store.PathByTask[task]
	if newPath != "/doing/1.leaf_task.md" {
		t.Errorf("After propagation path = %q, want %q", newPath, "/doing/1.leaf_task.md")
	}

	// Verify indexes updated
	if _, exists := store.TasksByPath["/pending/1.leaf_task.md"]; exists {
		t.Error("Old path should be removed from TasksByPath")
	}
	if _, exists := store.TasksByPath["/doing/1.leaf_task.md"]; !exists {
		t.Error("New path should be in TasksByPath")
	}
}

// TestPropagateStatusChange_ChildAffectsParent tests that child status change affects parent.
func TestPropagateStatusChange_ChildAffectsParent(t *testing.T) {
	// Create parent with children
	child1 := &Task{ID: "1.1", Title: "Child 1", Status: StatusDone}
	child2 := &Task{ID: "1.2", Title: "Child 2", Status: StatusDone}
	parent := &Task{
		ID:       "1",
		Title:    "Parent Task",
		Status:   StatusPending, // Own status ignored for parents
		Children: []*Task{child1, child2},
	}
	child1.Parent = parent
	child2.Parent = parent

	store := NewTaskStore()
	store.Document = &ParsedDocument{
		RootTasks: []*Task{parent},
	}
	pm := NewPathManager(store)

	// Build initial paths - parent should be in /done (all children done)
	pm.BuildAllPaths()

	parentPath := store.PathByTask[parent]
	if parentPath != "/done/1.parent_task" {
		t.Errorf("Initial parent path = %q, want %q", parentPath, "/done/1.parent_task")
	}

	// Change child1 status to doing
	child1.Status = StatusDoing
	PropagateStatusChange(child1, store, pm)

	// Parent should now be in /doing (has doing child)
	newParentPath := store.PathByTask[parent]
	if newParentPath != "/doing/1.parent_task" {
		t.Errorf("After child change, parent path = %q, want %q", newParentPath, "/doing/1.parent_task")
	}

	// Children should be under the new parent path
	child1Path := store.PathByTask[child1]
	expectedChild1Path := "/doing/1.parent_task/1.1.child_1.md"
	if child1Path != expectedChild1Path {
		t.Errorf("Child1 path = %q, want %q", child1Path, expectedChild1Path)
	}

	child2Path := store.PathByTask[child2]
	expectedChild2Path := "/doing/1.parent_task/1.2.child_2.md"
	if child2Path != expectedChild2Path {
		t.Errorf("Child2 path = %q, want %q", child2Path, expectedChild2Path)
	}
}

// TestPropagateStatusChange_GrandchildAffectsAncestors tests deep propagation.
func TestPropagateStatusChange_GrandchildAffectsAncestors(t *testing.T) {
	// Create a 3-level hierarchy
	grandchild := &Task{ID: "1.1.1", Title: "Grandchild", Status: StatusDone}
	child := &Task{
		ID:       "1.1",
		Title:    "Child",
		Status:   StatusPending,
		Children: []*Task{grandchild},
	}
	grandchild.Parent = child
	parent := &Task{
		ID:       "1",
		Title:    "Parent",
		Status:   StatusPending,
		Children: []*Task{child},
	}
	child.Parent = parent

	store := NewTaskStore()
	store.Document = &ParsedDocument{
		RootTasks: []*Task{parent},
	}
	pm := NewPathManager(store)

	// Build initial paths - all should be in /done (grandchild is done)
	pm.BuildAllPaths()

	parentPath := store.PathByTask[parent]
	if parentPath != "/done/1.parent" {
		t.Errorf("Initial parent path = %q, want %q", parentPath, "/done/1.parent")
	}

	// Change grandchild status to failed
	grandchild.Status = StatusFailed
	PropagateStatusChange(grandchild, store, pm)

	// All ancestors should now be in /failed
	newParentPath := store.PathByTask[parent]
	if newParentPath != "/failed/1.parent" {
		t.Errorf("After grandchild change, parent path = %q, want %q", newParentPath, "/failed/1.parent")
	}

	childPath := store.PathByTask[child]
	expectedChildPath := "/failed/1.parent/1.1.child"
	if childPath != expectedChildPath {
		t.Errorf("Child path = %q, want %q", childPath, expectedChildPath)
	}

	grandchildPath := store.PathByTask[grandchild]
	expectedGrandchildPath := "/failed/1.parent/1.1.child/1.1.1.grandchild.md"
	if grandchildPath != expectedGrandchildPath {
		t.Errorf("Grandchild path = %q, want %q", grandchildPath, expectedGrandchildPath)
	}
}

// TestPropagateStatusChange_PrecedenceRules tests that precedence rules are applied.
func TestPropagateStatusChange_PrecedenceRules(t *testing.T) {
	tests := []struct {
		name           string
		childStatuses  []TaskStatus
		changeIndex    int
		newStatus      TaskStatus
		expectedStatus TaskStatus
	}{
		{
			name:           "doing takes precedence over done",
			childStatuses:  []TaskStatus{StatusDone, StatusDone},
			changeIndex:    0,
			newStatus:      StatusDoing,
			expectedStatus: StatusDoing,
		},
		{
			name:           "failed takes precedence over pending",
			childStatuses:  []TaskStatus{StatusPending, StatusDone},
			changeIndex:    1,
			newStatus:      StatusFailed,
			expectedStatus: StatusFailed,
		},
		{
			name:           "pending takes precedence over queued",
			childStatuses:  []TaskStatus{StatusQueued, StatusDone},
			changeIndex:    1,
			newStatus:      StatusPending,
			expectedStatus: StatusPending,
		},
		{
			name:           "queued takes precedence over done",
			childStatuses:  []TaskStatus{StatusDone, StatusDone},
			changeIndex:    0,
			newStatus:      StatusQueued,
			expectedStatus: StatusQueued,
		},
		{
			name:           "all done results in done",
			childStatuses:  []TaskStatus{StatusDoing, StatusDone},
			changeIndex:    0,
			newStatus:      StatusDone,
			expectedStatus: StatusDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create children
			children := make([]*Task, len(tt.childStatuses))
			for i, status := range tt.childStatuses {
				children[i] = &Task{
					ID:     "1." + itoa(i+1),
					Title:  "Child " + itoa(i+1),
					Status: status,
				}
			}

			parent := &Task{
				ID:       "1",
				Title:    "Parent",
				Children: children,
			}
			for _, child := range children {
				child.Parent = parent
			}

			store := NewTaskStore()
			store.Document = &ParsedDocument{
				RootTasks: []*Task{parent},
			}
			pm := NewPathManager(store)
			pm.BuildAllPaths()

			// Change the specified child's status
			children[tt.changeIndex].Status = tt.newStatus
			PropagateStatusChange(children[tt.changeIndex], store, pm)

			// Verify parent is in the expected status directory
			parentPath := store.PathByTask[parent]
			expectedPathPrefix := "/" + string(tt.expectedStatus) + "/1.parent"
			if parentPath != expectedPathPrefix {
				t.Errorf("Parent path = %q, want %q", parentPath, expectedPathPrefix)
			}
		})
	}
}

// TestPropagateStatusChange_MultipleRootTasks tests propagation with multiple root tasks.
func TestPropagateStatusChange_MultipleRootTasks(t *testing.T) {
	// Create two independent root tasks
	task1Child := &Task{ID: "1.1", Title: "Task 1 Child", Status: StatusDone}
	task1 := &Task{
		ID:       "1",
		Title:    "Task 1",
		Children: []*Task{task1Child},
	}
	task1Child.Parent = task1

	task2 := &Task{ID: "2", Title: "Task 2", Status: StatusPending}

	store := NewTaskStore()
	store.Document = &ParsedDocument{
		RootTasks: []*Task{task1, task2},
	}
	pm := NewPathManager(store)
	pm.BuildAllPaths()

	// Verify initial paths
	task1Path := store.PathByTask[task1]
	if task1Path != "/done/1.task_1" {
		t.Errorf("Initial task1 path = %q, want %q", task1Path, "/done/1.task_1")
	}
	task2Path := store.PathByTask[task2]
	if task2Path != "/pending/2.task_2.md" {
		t.Errorf("Initial task2 path = %q, want %q", task2Path, "/pending/2.task_2.md")
	}

	// Change task1's child status
	task1Child.Status = StatusDoing
	PropagateStatusChange(task1Child, store, pm)

	// Task1 should move to /doing
	newTask1Path := store.PathByTask[task1]
	if newTask1Path != "/doing/1.task_1" {
		t.Errorf("After change, task1 path = %q, want %q", newTask1Path, "/doing/1.task_1")
	}

	// Task2 should be unaffected
	newTask2Path := store.PathByTask[task2]
	if newTask2Path != "/pending/2.task_2.md" {
		t.Errorf("Task2 path should be unchanged, got %q, want %q", newTask2Path, "/pending/2.task_2.md")
	}
}

// TestPropagateStatusChange_IndexesUpdated tests that all indexes are properly updated.
func TestPropagateStatusChange_IndexesUpdated(t *testing.T) {
	task := &Task{ID: "1", Title: "Test Task", Status: StatusPending}

	store := NewTaskStore()
	store.Document = &ParsedDocument{
		RootTasks: []*Task{task},
	}
	pm := NewPathManager(store)
	pm.BuildAllPaths()

	// Verify initial indexes
	oldPath := "/pending/1.test_task.md"
	if store.PathByTask[task] != oldPath {
		t.Errorf("Initial PathByTask = %q, want %q", store.PathByTask[task], oldPath)
	}
	if store.TasksByPath[oldPath] == nil {
		t.Error("Initial TasksByPath should contain old path")
	}
	if store.TasksByID["1"] == nil {
		t.Error("Initial TasksByID should contain task ID")
	}

	// Change status and propagate
	task.Status = StatusDone
	PropagateStatusChange(task, store, pm)

	// Verify indexes updated
	newPath := "/done/1.test_task.md"
	if store.PathByTask[task] != newPath {
		t.Errorf("After propagation PathByTask = %q, want %q", store.PathByTask[task], newPath)
	}
	if store.TasksByPath[oldPath] != nil {
		t.Error("Old path should be removed from TasksByPath")
	}
	if store.TasksByPath[newPath] == nil {
		t.Error("New path should be in TasksByPath")
	}
	if store.TasksByID["1"] == nil {
		t.Error("TasksByID should still contain task ID")
	}
	if store.TasksByID["1"].FullPath != newPath {
		t.Errorf("TasksByID entry path = %q, want %q", store.TasksByID["1"].FullPath, newPath)
	}
}

// TestPropagateStatusChange_DesignExample tests the example from the design document.
// From design.md: When child 1.2 changes to "doing", parent 1 should derive to "doing"
func TestPropagateStatusChange_DesignExample(t *testing.T) {
	// Create the structure from design.md Example 1:
	// - [ ] 1. Setup project
	//   - [x] 1.1 Create directory structure
	//   - [-] 1.2 Configure build system
	//   - [ ] 1.3 Add dependencies
	child1 := &Task{ID: "1.1", Title: "Create directory structure", Status: StatusDone}
	child2 := &Task{ID: "1.2", Title: "Configure build system", Status: StatusPending}
	child3 := &Task{ID: "1.3", Title: "Add dependencies", Status: StatusPending}
	parent := &Task{
		ID:       "1",
		Title:    "Setup project",
		Children: []*Task{child1, child2, child3},
	}
	child1.Parent = parent
	child2.Parent = parent
	child3.Parent = parent

	store := NewTaskStore()
	store.Document = &ParsedDocument{
		RootTasks: []*Task{parent},
	}
	pm := NewPathManager(store)
	pm.BuildAllPaths()

	// Initially parent should be in /pending (has pending children)
	initialPath := store.PathByTask[parent]
	if initialPath != "/pending/1.setup_project" {
		t.Errorf("Initial parent path = %q, want %q", initialPath, "/pending/1.setup_project")
	}

	// Change child2 to doing (simulating claiming the task)
	child2.Status = StatusDoing
	PropagateStatusChange(child2, store, pm)

	// Parent should now be in /doing
	newPath := store.PathByTask[parent]
	if newPath != "/doing/1.setup_project" {
		t.Errorf("After child2 doing, parent path = %q, want %q", newPath, "/doing/1.setup_project")
	}

	// All children should be under the new parent path
	child1Path := store.PathByTask[child1]
	if child1Path != "/doing/1.setup_project/1.1.create_directory_structure.md" {
		t.Errorf("Child1 path = %q, want %q", child1Path, "/doing/1.setup_project/1.1.create_directory_structure.md")
	}
}
