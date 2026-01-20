package parser

import (
	"testing"

	"task-fuse/internal/store"
)

// =============================================================================
// Parse Function Tests
// =============================================================================

// TestParse_EmptyContent tests parsing empty content
func TestParse_EmptyContent(t *testing.T) {
	p := NewParser()

	result := p.Parse("")

	if result == nil {
		t.Fatal("Parse should return a non-nil result")
	}
	if result.Document == nil {
		t.Fatal("Parse should return a non-nil document")
	}
	if len(result.Document.Preamble) != 0 {
		t.Errorf("Empty content should have no preamble, got %d lines", len(result.Document.Preamble))
	}
	if len(result.Document.RootTasks) != 0 {
		t.Errorf("Empty content should have no tasks, got %d tasks", len(result.Document.RootTasks))
	}
	if len(result.Document.Epilogue) != 0 {
		t.Errorf("Empty content should have no epilogue, got %d lines", len(result.Document.Epilogue))
	}
	if len(result.Errors) != 0 {
		t.Errorf("Empty content should have no errors, got %d errors", len(result.Errors))
	}
}

// TestParse_PreambleOnly tests parsing content with only preamble (no tasks)
func TestParse_PreambleOnly(t *testing.T) {
	p := NewParser()

	content := `# Project Tasks

This is the introduction.
Some more text here.`

	result := p.Parse(content)

	if result.Document == nil {
		t.Fatal("Parse should return a non-nil document")
	}
	if len(result.Document.Preamble) != 4 {
		t.Errorf("Expected 4 preamble lines, got %d", len(result.Document.Preamble))
	}
	if len(result.Document.RootTasks) != 0 {
		t.Errorf("Expected no tasks, got %d", len(result.Document.RootTasks))
	}
}

// TestParse_SingleTask tests parsing a single task
func TestParse_SingleTask(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 First task`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 1 {
		t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
	}

	task := result.Document.RootTasks[0]
	if task.ID != "1" {
		t.Errorf("Expected task ID '1', got '%s'", task.ID)
	}
	if task.Title != "First task" {
		t.Errorf("Expected title 'First task', got '%s'", task.Title)
	}
	if task.Status != store.StatusPending {
		t.Errorf("Expected status pending, got %s", task.Status)
	}
}

// TestParse_MultipleRootTasks tests parsing multiple root-level tasks
func TestParse_MultipleRootTasks(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 First task
- [x] 2 Second task
- [-] 3 Third task`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 3 {
		t.Fatalf("Expected 3 root tasks, got %d", len(result.Document.RootTasks))
	}

	// Check first task
	if result.Document.RootTasks[0].ID != "1" {
		t.Errorf("First task ID should be '1', got '%s'", result.Document.RootTasks[0].ID)
	}
	if result.Document.RootTasks[0].Status != store.StatusPending {
		t.Errorf("First task status should be pending, got %s", result.Document.RootTasks[0].Status)
	}

	// Check second task
	if result.Document.RootTasks[1].ID != "2" {
		t.Errorf("Second task ID should be '2', got '%s'", result.Document.RootTasks[1].ID)
	}
	if result.Document.RootTasks[1].Status != store.StatusDone {
		t.Errorf("Second task status should be done, got %s", result.Document.RootTasks[1].Status)
	}

	// Check third task
	if result.Document.RootTasks[2].ID != "3" {
		t.Errorf("Third task ID should be '3', got '%s'", result.Document.RootTasks[2].ID)
	}
	if result.Document.RootTasks[2].Status != store.StatusDoing {
		t.Errorf("Third task status should be doing, got %s", result.Document.RootTasks[2].Status)
	}
}

// TestParse_NestedTasks tests parsing nested tasks with hierarchy
func TestParse_NestedTasks(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 Parent task
  - [ ] 1.1 Child task
    - [ ] 1.1.1 Grandchild task
  - [ ] 1.2 Another child`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 1 {
		t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
	}

	parent := result.Document.RootTasks[0]
	if parent.ID != "1" {
		t.Errorf("Parent task ID should be '1', got '%s'", parent.ID)
	}
	if len(parent.Children) != 2 {
		t.Fatalf("Parent should have 2 children, got %d", len(parent.Children))
	}

	// Check first child
	child1 := parent.Children[0]
	if child1.ID != "1.1" {
		t.Errorf("First child ID should be '1.1', got '%s'", child1.ID)
	}
	if child1.Parent != parent {
		t.Error("First child's parent should be the parent task")
	}
	if len(child1.Children) != 1 {
		t.Fatalf("First child should have 1 grandchild, got %d", len(child1.Children))
	}

	// Check grandchild
	grandchild := child1.Children[0]
	if grandchild.ID != "1.1.1" {
		t.Errorf("Grandchild ID should be '1.1.1', got '%s'", grandchild.ID)
	}
	if grandchild.Parent != child1 {
		t.Error("Grandchild's parent should be the first child")
	}

	// Check second child
	child2 := parent.Children[1]
	if child2.ID != "1.2" {
		t.Errorf("Second child ID should be '1.2', got '%s'", child2.ID)
	}
	if child2.Parent != parent {
		t.Error("Second child's parent should be the parent task")
	}
}

// TestParse_PreambleAndTasks tests parsing with preamble before tasks
func TestParse_PreambleAndTasks(t *testing.T) {
	p := NewParser()

	content := `# Project Tasks

This is the introduction.

- [ ] 1 First task
- [ ] 2 Second task`

	result := p.Parse(content)

	// Check preamble
	if len(result.Document.Preamble) != 4 {
		t.Errorf("Expected 4 preamble lines, got %d", len(result.Document.Preamble))
	}
	if result.Document.Preamble[0].RawContent != "# Project Tasks" {
		t.Errorf("First preamble line should be heading, got '%s'", result.Document.Preamble[0].RawContent)
	}

	// Check tasks
	if len(result.Document.RootTasks) != 2 {
		t.Errorf("Expected 2 root tasks, got %d", len(result.Document.RootTasks))
	}
}

// TestParse_TaskWithDescription tests parsing tasks with description lines
func TestParse_TaskWithDescription(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 Setup project
  Initialize the project structure
  - More description
  - And another line`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 1 {
		t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
	}

	task := result.Document.RootTasks[0]
	if task.ID != "1" {
		t.Errorf("Task ID should be '1', got '%s'", task.ID)
	}
	if task.Title != "Setup project" {
		t.Errorf("Task title should be 'Setup project', got '%s'", task.Title)
	}
	if len(task.Description) != 3 {
		t.Errorf("Task should have 3 description lines, got %d", len(task.Description))
	}
	if task.Description[0].RawContent != "  Initialize the project structure" {
		t.Errorf("First description line incorrect, got '%s'", task.Description[0].RawContent)
	}
}

// TestParse_NoIDCheckboxLines tests that checkbox lines without valid IDs are treated as non-task content
// Validates: Requirements 1.6, 1.11, Design - Content Attachment Rules (Rule 3)
func TestParse_NoIDCheckboxLines(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 Valid task
  - [ ] No ID here
  - [x] Done task without ID`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 1 {
		t.Fatalf("Expected 1 root task (only the one with valid ID), got %d", len(result.Document.RootTasks))
	}

	task := result.Document.RootTasks[0]
	if task.ID != "1" {
		t.Errorf("Task ID should be '1', got '%s'", task.ID)
	}

	// The no-ID checkbox lines should be attached as description
	if len(task.Description) != 2 {
		t.Errorf("Task should have 2 description lines (the no-ID checkbox lines), got %d", len(task.Description))
	}
}

// TestParse_InvalidIDFormats tests that invalid ID formats are treated as non-task content
// Validates: Requirements 1.11
func TestParse_InvalidIDFormats(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 Valid task
  - [ ] 01.1 Invalid (leading zero)
  - [ ] 1.2 Valid subtask
  - [ ] 1.2.a Invalid (non-numeric)`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 1 {
		t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
	}

	parent := result.Document.RootTasks[0]
	if parent.ID != "1" {
		t.Errorf("Parent task ID should be '1', got '%s'", parent.ID)
	}

	// Should have one valid child (1.2)
	if len(parent.Children) != 1 {
		t.Fatalf("Parent should have 1 valid child, got %d", len(parent.Children))
	}

	child := parent.Children[0]
	if child.ID != "1.2" {
		t.Errorf("Child task ID should be '1.2', got '%s'", child.ID)
	}

	// The invalid ID line should be in parent's description
	if len(parent.Description) != 1 {
		t.Errorf("Parent should have 1 description line (the invalid ID line), got %d", len(parent.Description))
	}

	// The invalid ID line after 1.2 should be in child's description
	if len(child.Description) != 1 {
		t.Errorf("Child should have 1 description line (the invalid ID line), got %d", len(child.Description))
	}
}

// TestParse_AllStatusTypes tests parsing all checkbox status types
// Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5
func TestParse_AllStatusTypes(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 Pending task
- [~] 2 Queued task
- [-] 3 Doing task
- [x] 4 Done task
- [!] 5 Failed task`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 5 {
		t.Fatalf("Expected 5 root tasks, got %d", len(result.Document.RootTasks))
	}

	expectedStatuses := []store.TaskStatus{
		store.StatusPending,
		store.StatusQueued,
		store.StatusDoing,
		store.StatusDone,
		store.StatusFailed,
	}

	for i, task := range result.Document.RootTasks {
		if task.Status != expectedStatuses[i] {
			t.Errorf("Task %d status should be %s, got %s", i+1, expectedStatuses[i], task.Status)
		}
	}
}

// TestParse_DesignExample1 tests the exact example from the design document
// Validates: Design - Example 1: Basic Parsing and Filesystem Structure
func TestParse_DesignExample1(t *testing.T) {
	p := NewParser()

	content := `# Project Tasks

- [ ] 1. Setup project
  Initialize the project structure
  - [x] 1.1 Create directory structure
  - [-] 1.2 Configure build system
  - [ ] 1.3 Add dependencies
- [x] 2. Write documentation
  Document the API`

	result := p.Parse(content)

	// Check preamble
	if len(result.Document.Preamble) != 2 {
		t.Errorf("Expected 2 preamble lines, got %d", len(result.Document.Preamble))
	}

	// Check root tasks
	if len(result.Document.RootTasks) != 2 {
		t.Fatalf("Expected 2 root tasks, got %d", len(result.Document.RootTasks))
	}

	// Check task 1
	task1 := result.Document.RootTasks[0]
	if task1.ID != "1" {
		t.Errorf("First task ID should be '1', got '%s'", task1.ID)
	}
	if task1.Title != "Setup project" {
		t.Errorf("First task title should be 'Setup project', got '%s'", task1.Title)
	}
	if len(task1.Description) != 1 {
		t.Errorf("First task should have 1 description line, got %d", len(task1.Description))
	}
	if len(task1.Children) != 3 {
		t.Fatalf("First task should have 3 children, got %d", len(task1.Children))
	}

	// Check children of task 1
	if task1.Children[0].ID != "1.1" || task1.Children[0].Status != store.StatusDone {
		t.Errorf("Child 1.1 incorrect: ID=%s, Status=%s", task1.Children[0].ID, task1.Children[0].Status)
	}
	if task1.Children[1].ID != "1.2" || task1.Children[1].Status != store.StatusDoing {
		t.Errorf("Child 1.2 incorrect: ID=%s, Status=%s", task1.Children[1].ID, task1.Children[1].Status)
	}
	if task1.Children[2].ID != "1.3" || task1.Children[2].Status != store.StatusPending {
		t.Errorf("Child 1.3 incorrect: ID=%s, Status=%s", task1.Children[2].ID, task1.Children[2].Status)
	}

	// Check task 2
	task2 := result.Document.RootTasks[1]
	if task2.ID != "2" {
		t.Errorf("Second task ID should be '2', got '%s'", task2.ID)
	}
	if task2.Status != store.StatusDone {
		t.Errorf("Second task status should be done, got %s", task2.Status)
	}
	if len(task2.Description) != 1 {
		t.Errorf("Second task should have 1 description line, got %d", len(task2.Description))
	}
}

// TestParse_DesignExample7 tests the invalid task ID handling example from design
// Validates: Design - Example 7: Invalid Task ID Handling
func TestParse_DesignExample7(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1. Valid task
  - [ ] 01.1 Invalid (leading zero)
  - [ ] 1.2 Valid subtask
  - [ ] 1.2.a Invalid (non-numeric)
- [ ] 2. Another valid task`

	result := p.Parse(content)

	// Should have 2 root tasks (1 and 2)
	if len(result.Document.RootTasks) != 2 {
		t.Fatalf("Expected 2 root tasks, got %d", len(result.Document.RootTasks))
	}

	// Task 1 should have the invalid ID line as description
	task1 := result.Document.RootTasks[0]
	if task1.ID != "1" {
		t.Errorf("First task ID should be '1', got '%s'", task1.ID)
	}
	if len(task1.Description) != 1 {
		t.Errorf("Task 1 should have 1 description line (invalid ID line), got %d", len(task1.Description))
	}

	// Task 1 should have 1 valid child (1.2)
	if len(task1.Children) != 1 {
		t.Fatalf("Task 1 should have 1 child, got %d", len(task1.Children))
	}

	// Child 1.2 should have the invalid ID line as description
	child := task1.Children[0]
	if child.ID != "1.2" {
		t.Errorf("Child ID should be '1.2', got '%s'", child.ID)
	}
	if len(child.Description) != 1 {
		t.Errorf("Child 1.2 should have 1 description line (invalid ID line), got %d", len(child.Description))
	}

	// Task 2 should be valid
	task2 := result.Document.RootTasks[1]
	if task2.ID != "2" {
		t.Errorf("Second task ID should be '2', got '%s'", task2.ID)
	}
}

// TestParse_LineNumbers tests that line numbers are correctly tracked
func TestParse_LineNumbers(t *testing.T) {
	p := NewParser()

	content := `# Heading
- [ ] 1 First task
  Description line
- [ ] 2 Second task`

	result := p.Parse(content)

	// Check preamble line number
	if result.Document.Preamble[0].LineNumber != 1 {
		t.Errorf("Preamble line number should be 1, got %d", result.Document.Preamble[0].LineNumber)
	}

	// Check first task line number
	if result.Document.RootTasks[0].LineNumber != 2 {
		t.Errorf("First task line number should be 2, got %d", result.Document.RootTasks[0].LineNumber)
	}

	// Check description line number
	if result.Document.RootTasks[0].Description[0].LineNumber != 3 {
		t.Errorf("Description line number should be 3, got %d", result.Document.RootTasks[0].Description[0].LineNumber)
	}

	// Check second task line number
	if result.Document.RootTasks[1].LineNumber != 4 {
		t.Errorf("Second task line number should be 4, got %d", result.Document.RootTasks[1].LineNumber)
	}
}

// TestParse_IndentLevels tests that indent levels are correctly computed
func TestParse_IndentLevels(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 Root task
  - [ ] 1.1 Child (2 spaces)
    - [ ] 1.1.1 Grandchild (4 spaces)
	- [ ] 1.2 Child with tab`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 1 {
		t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
	}

	root := result.Document.RootTasks[0]
	if root.IndentLevel != 0 {
		t.Errorf("Root task indent level should be 0, got %d", root.IndentLevel)
	}

	if len(root.Children) < 1 {
		t.Fatal("Root should have at least 1 child")
	}

	child := root.Children[0]
	if child.IndentLevel != 1 {
		t.Errorf("Child indent level should be 1, got %d", child.IndentLevel)
	}

	if len(child.Children) < 1 {
		t.Fatal("Child should have at least 1 grandchild")
	}

	grandchild := child.Children[0]
	if grandchild.IndentLevel != 2 {
		t.Errorf("Grandchild indent level should be 2, got %d", grandchild.IndentLevel)
	}
}

// TestParse_RawContentPreserved tests that raw content is preserved exactly
func TestParse_RawContentPreserved(t *testing.T) {
	p := NewParser()

	content := `# Project Tasks

- [ ] 1 First task
  Description with   extra   spaces
- [x] 2 Second task`

	result := p.Parse(content)

	// Check preamble raw content
	if result.Document.Preamble[0].RawContent != "# Project Tasks" {
		t.Errorf("Preamble raw content not preserved: '%s'", result.Document.Preamble[0].RawContent)
	}

	// Check task raw content
	if result.Document.RootTasks[0].RawContent != "- [ ] 1 First task" {
		t.Errorf("Task raw content not preserved: '%s'", result.Document.RootTasks[0].RawContent)
	}

	// Check description raw content (with extra spaces)
	if result.Document.RootTasks[0].Description[0].RawContent != "  Description with   extra   spaces" {
		t.Errorf("Description raw content not preserved: '%s'", result.Document.RootTasks[0].Description[0].RawContent)
	}
}

// TestParse_DeepNesting tests parsing deeply nested tasks
func TestParse_DeepNesting(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 Level 0
  - [ ] 1.1 Level 1
    - [ ] 1.1.1 Level 2
      - [ ] 1.1.1.1 Level 3
        - [ ] 1.1.1.1.1 Level 4`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 1 {
		t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
	}

	// Traverse down the hierarchy
	current := result.Document.RootTasks[0]
	expectedIDs := []string{"1", "1.1", "1.1.1", "1.1.1.1", "1.1.1.1.1"}

	for i, expectedID := range expectedIDs {
		if current.ID != expectedID {
			t.Errorf("Level %d task ID should be '%s', got '%s'", i, expectedID, current.ID)
		}
		if i < len(expectedIDs)-1 {
			if len(current.Children) != 1 {
				t.Fatalf("Level %d task should have 1 child, got %d", i, len(current.Children))
			}
			current = current.Children[0]
		}
	}
}

// TestParse_NoIDCheckboxInPreamble tests checkbox without ID in preamble
func TestParse_NoIDCheckboxInPreamble(t *testing.T) {
	p := NewParser()

	content := `# Project Tasks

- [ ] Checkbox without ID in preamble
- [x] Another checkbox without ID

- [ ] 1 First valid task`

	result := p.Parse(content)

	// The no-ID checkboxes should be in preamble
	if len(result.Document.Preamble) != 5 {
		t.Errorf("Expected 5 preamble lines (including no-ID checkboxes), got %d", len(result.Document.Preamble))
	}

	// Should have 1 valid task
	if len(result.Document.RootTasks) != 1 {
		t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
	}

	if result.Document.RootTasks[0].ID != "1" {
		t.Errorf("Task ID should be '1', got '%s'", result.Document.RootTasks[0].ID)
	}
}

// TestParse_MixedContent tests parsing with mixed content types
func TestParse_MixedContent(t *testing.T) {
	p := NewParser()

	content := `# Project Tasks

Introduction text.

- [ ] 1 First task
  Task description
  - Bullet point in description
  - [ ] 1.1 Valid subtask
    Subtask description
  - [ ] No ID checkbox (description)
- [ ] 2 Second task`

	result := p.Parse(content)

	// Check preamble
	if len(result.Document.Preamble) != 4 {
		t.Errorf("Expected 4 preamble lines, got %d", len(result.Document.Preamble))
	}

	// Check root tasks
	if len(result.Document.RootTasks) != 2 {
		t.Fatalf("Expected 2 root tasks, got %d", len(result.Document.RootTasks))
	}

	// Check first task
	task1 := result.Document.RootTasks[0]
	if task1.ID != "1" {
		t.Errorf("First task ID should be '1', got '%s'", task1.ID)
	}

	// First task should have description lines
	if len(task1.Description) < 2 {
		t.Errorf("First task should have at least 2 description lines, got %d", len(task1.Description))
	}

	// First task should have 1 valid child
	if len(task1.Children) != 1 {
		t.Fatalf("First task should have 1 child, got %d", len(task1.Children))
	}

	// Check subtask
	subtask := task1.Children[0]
	if subtask.ID != "1.1" {
		t.Errorf("Subtask ID should be '1.1', got '%s'", subtask.ID)
	}
}

// TestParse_TrailingDotInID tests parsing task IDs with trailing dots
func TestParse_TrailingDotInID(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1. Task with trailing dot
- [ ] 2.1. Another with trailing dot`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 2 {
		t.Fatalf("Expected 2 root tasks, got %d", len(result.Document.RootTasks))
	}

	// IDs should not include the trailing dot
	if result.Document.RootTasks[0].ID != "1" {
		t.Errorf("First task ID should be '1', got '%s'", result.Document.RootTasks[0].ID)
	}
	if result.Document.RootTasks[1].ID != "2.1" {
		t.Errorf("Second task ID should be '2.1', got '%s'", result.Document.RootTasks[1].ID)
	}
}

// TestParse_Requirement1_6 validates Requirement 1.6
// WHEN a task line has checkbox syntax but lacks a numeric identifier,
// THE Tasks_MD_Parser SHALL ignore it (not parse it as a task)
func TestParse_Requirement1_6(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 Valid task
- [ ] No ID here
- [x] Another no ID
- [ ] 2 Another valid task`

	result := p.Parse(content)

	// Should only have 2 valid tasks (1 and 2)
	if len(result.Document.RootTasks) != 2 {
		t.Fatalf("Requirement 1.6: Expected 2 valid tasks, got %d", len(result.Document.RootTasks))
	}

	if result.Document.RootTasks[0].ID != "1" {
		t.Errorf("First task ID should be '1', got '%s'", result.Document.RootTasks[0].ID)
	}
	if result.Document.RootTasks[1].ID != "2" {
		t.Errorf("Second task ID should be '2', got '%s'", result.Document.RootTasks[1].ID)
	}

	// The no-ID lines should be attached to task 1's description
	if len(result.Document.RootTasks[0].Description) != 2 {
		t.Errorf("Task 1 should have 2 description lines (the no-ID checkboxes), got %d", len(result.Document.RootTasks[0].Description))
	}
}

// TestParse_Requirement1_8 validates Requirement 1.8
// WHEN a task contains additional metadata or bullet points,
// THE Tasks_MD_Parser SHALL preserve all content as task description
func TestParse_Requirement1_8(t *testing.T) {
	p := NewParser()

	content := `- [ ] 1 Task with metadata
  **Status:** In Progress
  - Bullet point 1
  - Bullet point 2
  Some additional text`

	result := p.Parse(content)

	if len(result.Document.RootTasks) != 1 {
		t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
	}

	task := result.Document.RootTasks[0]
	if len(task.Description) != 4 {
		t.Errorf("Requirement 1.8: Task should have 4 description lines, got %d", len(task.Description))
	}

	// Verify content is preserved
	expectedLines := []string{
		"  **Status:** In Progress",
		"  - Bullet point 1",
		"  - Bullet point 2",
		"  Some additional text",
	}

	for i, expected := range expectedLines {
		if i < len(task.Description) && task.Description[i].RawContent != expected {
			t.Errorf("Description line %d should be '%s', got '%s'", i, expected, task.Description[i].RawContent)
		}
	}
}

// TestParse_Requirement1_11 validates Requirement 1.11
// IF a task ID does not match the Task ID Format Specification,
// THE Tasks_MD_Parser SHALL treat the line as non-task content and preserve it unchanged
func TestParse_Requirement1_11(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name    string
		content string
		reason  string
	}{
		{
			name:    "leading zero",
			content: "- [ ] 1 Valid\n  - [ ] 01.1 Invalid leading zero",
			reason:  "leading zero in ID",
		},
		{
			name:    "non-numeric segment",
			content: "- [ ] 1 Valid\n  - [ ] 1.2.a Invalid non-numeric",
			reason:  "non-numeric segment in ID",
		},
		{
			name:    "empty segment",
			content: "- [ ] 1 Valid\n  - [ ] 1..2 Invalid empty segment",
			reason:  "empty segment in ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.content)

			// Should have 1 valid task
			if len(result.Document.RootTasks) != 1 {
				t.Fatalf("Expected 1 valid task, got %d", len(result.Document.RootTasks))
			}

			// The invalid ID line should be preserved as description
			if len(result.Document.RootTasks[0].Description) != 1 {
				t.Errorf("Requirement 1.11: Invalid ID line should be preserved as description (%s)", tt.reason)
			}
		})
	}
}
