package printer

import (
	"strings"
	"testing"

	"task-fuse/internal/store"
)

// =============================================================================
// StatusToCheckbox Tests
// =============================================================================

func TestStatusToCheckbox_AllStatuses(t *testing.T) {
	// This test explicitly validates Requirements 1.9
	// by testing the exact status to checkbox mapping
	p := NewPrinter()

	tests := []struct {
		name     string
		status   store.TaskStatus
		expected string
	}{
		{
			name:     "pending status",
			status:   store.StatusPending,
			expected: "- [ ]",
		},
		{
			name:     "queued status",
			status:   store.StatusQueued,
			expected: "- [~]",
		},
		{
			name:     "doing status",
			status:   store.StatusDoing,
			expected: "- [-]",
		},
		{
			name:     "done status",
			status:   store.StatusDone,
			expected: "- [x]",
		},
		{
			name:     "failed status",
			status:   store.StatusFailed,
			expected: "- [!]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.StatusToCheckbox(tt.status)
			if got != tt.expected {
				t.Errorf("StatusToCheckbox(%q) = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

func TestStatusToCheckbox_UnknownStatus(t *testing.T) {
	p := NewPrinter()

	// Unknown status should default to pending checkbox
	unknownStatus := store.TaskStatus("unknown")
	got := p.StatusToCheckbox(unknownStatus)
	expected := "- [ ]"

	if got != expected {
		t.Errorf("StatusToCheckbox(%q) = %q, want %q (default to pending)", unknownStatus, got, expected)
	}
}

func TestStatusToCheckbox_EmptyStatus(t *testing.T) {
	p := NewPrinter()

	// Empty status should default to pending checkbox
	emptyStatus := store.TaskStatus("")
	got := p.StatusToCheckbox(emptyStatus)
	expected := "- [ ]"

	if got != expected {
		t.Errorf("StatusToCheckbox(%q) = %q, want %q (default to pending)", emptyStatus, got, expected)
	}
}

// TestStatusToCheckbox_RequirementsValidation explicitly validates Requirements 1.9
// Validates: Requirements 1.9 - Pretty_Printer SHALL serialize task objects back to valid tasks.md markdown format with correct checkbox syntax
func TestStatusToCheckbox_RequirementsValidation(t *testing.T) {
	p := NewPrinter()

	// Requirement 1.9: Correct checkbox syntax for each status
	// Based on the Checkbox Syntax Mapping from the design document:
	// | Status  | Checkbox |
	// |---------|----------|
	// | Pending | `- [ ]`  |
	// | Queued  | `- [~]`  |
	// | Doing   | `- [-]`  |
	// | Done    | `- [x]`  |
	// | Failed  | `- [!]`  |

	t.Run("Requirement 1.9 - pending checkbox", func(t *testing.T) {
		if got := p.StatusToCheckbox(store.StatusPending); got != "- [ ]" {
			t.Errorf("StatusPending should produce '- [ ]', got %q", got)
		}
	})

	t.Run("Requirement 1.9 - queued checkbox", func(t *testing.T) {
		if got := p.StatusToCheckbox(store.StatusQueued); got != "- [~]" {
			t.Errorf("StatusQueued should produce '- [~]', got %q", got)
		}
	})

	t.Run("Requirement 1.9 - doing checkbox", func(t *testing.T) {
		if got := p.StatusToCheckbox(store.StatusDoing); got != "- [-]" {
			t.Errorf("StatusDoing should produce '- [-]', got %q", got)
		}
	})

	t.Run("Requirement 1.9 - done checkbox", func(t *testing.T) {
		if got := p.StatusToCheckbox(store.StatusDone); got != "- [x]" {
			t.Errorf("StatusDone should produce '- [x]', got %q", got)
		}
	})

	t.Run("Requirement 1.9 - failed checkbox", func(t *testing.T) {
		if got := p.StatusToCheckbox(store.StatusFailed); got != "- [!]" {
			t.Errorf("StatusFailed should produce '- [!]', got %q", got)
		}
	})
}

// TestStatusToCheckbox_CheckboxFormat verifies the exact format of checkbox strings
func TestStatusToCheckbox_CheckboxFormat(t *testing.T) {
	p := NewPrinter()

	// All checkboxes should:
	// 1. Start with "- ["
	// 2. Have exactly one character inside brackets
	// 3. End with "]"
	// 4. Be exactly 5 characters long

	statuses := []store.TaskStatus{
		store.StatusPending,
		store.StatusQueued,
		store.StatusDoing,
		store.StatusDone,
		store.StatusFailed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			checkbox := p.StatusToCheckbox(status)

			// Check length
			if len(checkbox) != 5 {
				t.Errorf("StatusToCheckbox(%q) = %q, expected length 5, got %d", status, checkbox, len(checkbox))
			}

			// Check prefix
			if checkbox[:3] != "- [" {
				t.Errorf("StatusToCheckbox(%q) = %q, should start with '- ['", status, checkbox)
			}

			// Check suffix
			if checkbox[4:] != "]" {
				t.Errorf("StatusToCheckbox(%q) = %q, should end with ']'", status, checkbox)
			}
		})
	}
}


// =============================================================================
// FormatTask Tests
// =============================================================================

// TestFormatTask_SimpleTask tests formatting a simple task without children or description
func TestFormatTask_SimpleTask(t *testing.T) {
	p := NewPrinter()

	task := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  1,
			RawContent:  "- [ ] 1. Simple task",
			IndentLevel: 0,
		},
		ID:          "1",
		Title:       "Simple task",
		Status:      store.StatusPending,
		Description: []store.FileLine{},
		Children:    []*store.Task{},
	}

	got := p.FormatTask(task)
	expected := "- [ ] 1. Simple task\n"

	if got != expected {
		t.Errorf("FormatTask() = %q, want %q", got, expected)
	}
}

// TestFormatTask_StatusChanged tests that only the checkbox character changes when status changes
func TestFormatTask_StatusChanged(t *testing.T) {
	p := NewPrinter()

	tests := []struct {
		name       string
		rawContent string
		newStatus  store.TaskStatus
		expected   string
	}{
		{
			name:       "pending to doing",
			rawContent: "- [ ] 1.1 Task title",
			newStatus:  store.StatusDoing,
			expected:   "- [-] 1.1 Task title\n",
		},
		{
			name:       "pending to done",
			rawContent: "- [ ] 2. Another task",
			newStatus:  store.StatusDone,
			expected:   "- [x] 2. Another task\n",
		},
		{
			name:       "doing to failed",
			rawContent: "- [-] 3.1.2 Deep task",
			newStatus:  store.StatusFailed,
			expected:   "- [!] 3.1.2 Deep task\n",
		},
		{
			name:       "done to pending",
			rawContent: "- [x] 4. Completed task",
			newStatus:  store.StatusPending,
			expected:   "- [ ] 4. Completed task\n",
		},
		{
			name:       "pending to queued",
			rawContent: "- [ ] 5. Queue me",
			newStatus:  store.StatusQueued,
			expected:   "- [~] 5. Queue me\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &store.Task{
				FileLine: store.FileLine{
					LineNumber:  1,
					RawContent:  tt.rawContent,
					IndentLevel: 0,
				},
				ID:          "1",
				Title:       "Task",
				Status:      tt.newStatus,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			}

			got := p.FormatTask(task)
			if got != tt.expected {
				t.Errorf("FormatTask() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestFormatTask_PreservesIndentation tests that original indentation is preserved
func TestFormatTask_PreservesIndentation(t *testing.T) {
	p := NewPrinter()

	tests := []struct {
		name       string
		rawContent string
		expected   string
	}{
		{
			name:       "no indentation",
			rawContent: "- [ ] 1. Root task",
			expected:   "- [ ] 1. Root task\n",
		},
		{
			name:       "2 spaces indentation",
			rawContent: "  - [ ] 1.1 Child task",
			expected:   "  - [ ] 1.1 Child task\n",
		},
		{
			name:       "4 spaces indentation",
			rawContent: "    - [ ] 1.1.1 Grandchild task",
			expected:   "    - [ ] 1.1.1 Grandchild task\n",
		},
		{
			name:       "tab indentation",
			rawContent: "\t- [ ] 1.1 Tab indented",
			expected:   "\t- [ ] 1.1 Tab indented\n",
		},
		{
			name:       "mixed indentation",
			rawContent: "  \t- [ ] 1.1.1 Mixed indent",
			expected:   "  \t- [ ] 1.1.1 Mixed indent\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &store.Task{
				FileLine: store.FileLine{
					LineNumber:  1,
					RawContent:  tt.rawContent,
					IndentLevel: 0,
				},
				ID:          "1",
				Title:       "Task",
				Status:      store.StatusPending,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			}

			got := p.FormatTask(task)
			if got != tt.expected {
				t.Errorf("FormatTask() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestFormatTask_WithDescription tests formatting a task with description lines
func TestFormatTask_WithDescription(t *testing.T) {
	p := NewPrinter()

	task := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  1,
			RawContent:  "- [ ] 1. Task with description",
			IndentLevel: 0,
		},
		ID:     "1",
		Title:  "Task with description",
		Status: store.StatusPending,
		Description: []store.FileLine{
			{LineNumber: 2, RawContent: "  This is the first description line", IndentLevel: 1},
			{LineNumber: 3, RawContent: "  - A bullet point", IndentLevel: 1},
			{LineNumber: 4, RawContent: "  - Another bullet", IndentLevel: 1},
		},
		Children: []*store.Task{},
	}

	got := p.FormatTask(task)
	expected := "- [ ] 1. Task with description\n" +
		"  This is the first description line\n" +
		"  - A bullet point\n" +
		"  - Another bullet\n"

	if got != expected {
		t.Errorf("FormatTask() = %q, want %q", got, expected)
	}
}

// TestFormatTask_WithChildren tests recursive formatting of children
func TestFormatTask_WithChildren(t *testing.T) {
	p := NewPrinter()

	child1 := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  2,
			RawContent:  "  - [x] 1.1 First child",
			IndentLevel: 1,
		},
		ID:          "1.1",
		Title:       "First child",
		Status:      store.StatusDone,
		Description: []store.FileLine{},
		Children:    []*store.Task{},
	}

	child2 := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  3,
			RawContent:  "  - [-] 1.2 Second child",
			IndentLevel: 1,
		},
		ID:          "1.2",
		Title:       "Second child",
		Status:      store.StatusDoing,
		Description: []store.FileLine{},
		Children:    []*store.Task{},
	}

	parent := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  1,
			RawContent:  "- [ ] 1. Parent task",
			IndentLevel: 0,
		},
		ID:          "1",
		Title:       "Parent task",
		Status:      store.StatusPending,
		Description: []store.FileLine{},
		Children:    []*store.Task{child1, child2},
	}

	child1.Parent = parent
	child2.Parent = parent

	got := p.FormatTask(parent)
	expected := "- [ ] 1. Parent task\n" +
		"  - [x] 1.1 First child\n" +
		"  - [-] 1.2 Second child\n"

	if got != expected {
		t.Errorf("FormatTask() = %q, want %q", got, expected)
	}
}

// TestFormatTask_NestedHierarchy tests deeply nested task hierarchy
func TestFormatTask_NestedHierarchy(t *testing.T) {
	p := NewPrinter()

	grandchild := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  3,
			RawContent:  "    - [ ] 1.1.1 Grandchild",
			IndentLevel: 2,
		},
		ID:          "1.1.1",
		Title:       "Grandchild",
		Status:      store.StatusPending,
		Description: []store.FileLine{},
		Children:    []*store.Task{},
	}

	child := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  2,
			RawContent:  "  - [-] 1.1 Child",
			IndentLevel: 1,
		},
		ID:          "1.1",
		Title:       "Child",
		Status:      store.StatusDoing,
		Description: []store.FileLine{},
		Children:    []*store.Task{grandchild},
	}

	parent := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  1,
			RawContent:  "- [ ] 1. Parent",
			IndentLevel: 0,
		},
		ID:          "1",
		Title:       "Parent",
		Status:      store.StatusPending,
		Description: []store.FileLine{},
		Children:    []*store.Task{child},
	}

	grandchild.Parent = child
	child.Parent = parent

	got := p.FormatTask(parent)
	expected := "- [ ] 1. Parent\n" +
		"  - [-] 1.1 Child\n" +
		"    - [ ] 1.1.1 Grandchild\n"

	if got != expected {
		t.Errorf("FormatTask() = %q, want %q", got, expected)
	}
}

// TestFormatTask_WithDescriptionAndChildren tests task with both description and children
func TestFormatTask_WithDescriptionAndChildren(t *testing.T) {
	p := NewPrinter()

	child := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  4,
			RawContent:  "  - [x] 1.1 Child task",
			IndentLevel: 1,
		},
		ID:          "1.1",
		Title:       "Child task",
		Status:      store.StatusDone,
		Description: []store.FileLine{},
		Children:    []*store.Task{},
	}

	parent := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  1,
			RawContent:  "- [ ] 1. Parent with description",
			IndentLevel: 0,
		},
		ID:     "1",
		Title:  "Parent with description",
		Status: store.StatusPending,
		Description: []store.FileLine{
			{LineNumber: 2, RawContent: "  Description line 1", IndentLevel: 1},
			{LineNumber: 3, RawContent: "  Description line 2", IndentLevel: 1},
		},
		Children: []*store.Task{child},
	}

	child.Parent = parent

	got := p.FormatTask(parent)
	expected := "- [ ] 1. Parent with description\n" +
		"  Description line 1\n" +
		"  Description line 2\n" +
		"  - [x] 1.1 Child task\n"

	if got != expected {
		t.Errorf("FormatTask() = %q, want %q", got, expected)
	}
}

// TestFormatTask_NilTask tests that nil task returns empty string
func TestFormatTask_NilTask(t *testing.T) {
	p := NewPrinter()

	got := p.FormatTask(nil)
	if got != "" {
		t.Errorf("FormatTask(nil) = %q, want empty string", got)
	}
}

// TestFormatTask_StatusChangePreservesContent tests that status change only affects checkbox
func TestFormatTask_StatusChangePreservesContent(t *testing.T) {
	p := NewPrinter()

	// Original task was pending, now it's done
	task := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  1,
			RawContent:  "  - [ ] 1.2.3 Task with special chars: @#$%^&*()",
			IndentLevel: 1,
		},
		ID:          "1.2.3",
		Title:       "Task with special chars: @#$%^&*()",
		Status:      store.StatusDone, // Changed from pending to done
		Description: []store.FileLine{},
		Children:    []*store.Task{},
	}

	got := p.FormatTask(task)
	expected := "  - [x] 1.2.3 Task with special chars: @#$%^&*()\n"

	if got != expected {
		t.Errorf("FormatTask() = %q, want %q", got, expected)
	}
}

// TestFormatTask_AllStatusTransitions tests all possible status values
func TestFormatTask_AllStatusTransitions(t *testing.T) {
	p := NewPrinter()

	statuses := []struct {
		status   store.TaskStatus
		checkbox string
	}{
		{store.StatusPending, " "},
		{store.StatusQueued, "~"},
		{store.StatusDoing, "-"},
		{store.StatusDone, "x"},
		{store.StatusFailed, "!"},
	}

	for _, s := range statuses {
		t.Run(string(s.status), func(t *testing.T) {
			task := &store.Task{
				FileLine: store.FileLine{
					LineNumber:  1,
					RawContent:  "- [x] 1. Test task", // Original was done
					IndentLevel: 0,
				},
				ID:          "1",
				Title:       "Test task",
				Status:      s.status,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			}

			got := p.FormatTask(task)
			expectedCheckbox := "- [" + s.checkbox + "]"
			if !strings.Contains(got, expectedCheckbox) {
				t.Errorf("FormatTask() = %q, should contain checkbox %q", got, expectedCheckbox)
			}
		})
	}
}


// =============================================================================
// Serialize Tests
// =============================================================================

// TestSerialize_NilDocument tests that nil document returns empty string
func TestSerialize_NilDocument(t *testing.T) {
	p := NewPrinter()

	got := p.Serialize(nil)
	if got != "" {
		t.Errorf("Serialize(nil) = %q, want empty string", got)
	}
}

// TestSerialize_EmptyDocument tests serializing an empty document
func TestSerialize_EmptyDocument(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble:  []store.NonTaskContent{},
		RootTasks: []*store.Task{},
		Epilogue:  []store.NonTaskContent{},
	}

	got := p.Serialize(doc)
	if got != "" {
		t.Errorf("Serialize(empty doc) = %q, want empty string", got)
	}
}

// TestSerialize_PreambleOnly tests serializing a document with only preamble
func TestSerialize_PreambleOnly(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{
			{FileLine: store.FileLine{LineNumber: 1, RawContent: "# Project Tasks", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 2, RawContent: "", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 3, RawContent: "Some intro text", IndentLevel: 0}},
		},
		RootTasks: []*store.Task{},
		Epilogue:  []store.NonTaskContent{},
	}

	got := p.Serialize(doc)
	expected := "# Project Tasks\n\nSome intro text"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_EpilogueOnly tests serializing a document with only epilogue
func TestSerialize_EpilogueOnly(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble:  []store.NonTaskContent{},
		RootTasks: []*store.Task{},
		Epilogue: []store.NonTaskContent{
			{FileLine: store.FileLine{LineNumber: 1, RawContent: "", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 2, RawContent: "## Notes", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 3, RawContent: "Some notes here", IndentLevel: 0}},
		},
	}

	got := p.Serialize(doc)
	expected := "\n## Notes\nSome notes here"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_TasksOnly tests serializing a document with only tasks
func TestSerialize_TasksOnly(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{},
		RootTasks: []*store.Task{
			{
				FileLine: store.FileLine{
					LineNumber:  1,
					RawContent:  "- [ ] 1. First task",
					IndentLevel: 0,
				},
				ID:          "1",
				Title:       "First task",
				Status:      store.StatusPending,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
			{
				FileLine: store.FileLine{
					LineNumber:  2,
					RawContent:  "- [x] 2. Second task",
					IndentLevel: 0,
				},
				ID:          "2",
				Title:       "Second task",
				Status:      store.StatusDone,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
		},
		Epilogue: []store.NonTaskContent{},
	}

	got := p.Serialize(doc)
	expected := "- [ ] 1. First task\n- [x] 2. Second task"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_FullDocument tests serializing a complete document with preamble, tasks, and epilogue
// Validates: Requirements 1.9, 1.10
func TestSerialize_FullDocument(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{
			{FileLine: store.FileLine{LineNumber: 1, RawContent: "# Project Tasks", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 2, RawContent: "", IndentLevel: 0}},
		},
		RootTasks: []*store.Task{
			{
				FileLine: store.FileLine{
					LineNumber:  3,
					RawContent:  "- [ ] 1. Setup project",
					IndentLevel: 0,
				},
				ID:     "1",
				Title:  "Setup project",
				Status: store.StatusPending,
				Description: []store.FileLine{
					{LineNumber: 4, RawContent: "  Initialize the project", IndentLevel: 1},
				},
				Children: []*store.Task{},
			},
			{
				FileLine: store.FileLine{
					LineNumber:  5,
					RawContent:  "- [x] 2. Documentation",
					IndentLevel: 0,
				},
				ID:          "2",
				Title:       "Documentation",
				Status:      store.StatusDone,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
		},
		Epilogue: []store.NonTaskContent{
			{FileLine: store.FileLine{LineNumber: 6, RawContent: "", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 7, RawContent: "## Notes", IndentLevel: 0}},
		},
	}

	got := p.Serialize(doc)
	expected := "# Project Tasks\n\n- [ ] 1. Setup project\n  Initialize the project\n- [x] 2. Documentation\n\n## Notes"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_NestedTasks tests serializing tasks with nested children
func TestSerialize_NestedTasks(t *testing.T) {
	p := NewPrinter()

	grandchild := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  4,
			RawContent:  "    - [ ] 1.1.1 Grandchild task",
			IndentLevel: 2,
		},
		ID:          "1.1.1",
		Title:       "Grandchild task",
		Status:      store.StatusPending,
		Description: []store.FileLine{},
		Children:    []*store.Task{},
	}

	child := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  3,
			RawContent:  "  - [-] 1.1 Child task",
			IndentLevel: 1,
		},
		ID:          "1.1",
		Title:       "Child task",
		Status:      store.StatusDoing,
		Description: []store.FileLine{},
		Children:    []*store.Task{grandchild},
	}

	parent := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  2,
			RawContent:  "- [ ] 1. Parent task",
			IndentLevel: 0,
		},
		ID:          "1",
		Title:       "Parent task",
		Status:      store.StatusPending,
		Description: []store.FileLine{},
		Children:    []*store.Task{child},
	}

	grandchild.Parent = child
	child.Parent = parent

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{
			{FileLine: store.FileLine{LineNumber: 1, RawContent: "# Tasks", IndentLevel: 0}},
		},
		RootTasks: []*store.Task{parent},
		Epilogue:  []store.NonTaskContent{},
	}

	got := p.Serialize(doc)
	expected := "# Tasks\n- [ ] 1. Parent task\n  - [-] 1.1 Child task\n    - [ ] 1.1.1 Grandchild task"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_PreservesNonTaskContent tests that non-task content is preserved exactly
// Validates: Requirements 1.9, 1.10
func TestSerialize_PreservesNonTaskContent(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{
			{FileLine: store.FileLine{LineNumber: 1, RawContent: "# Project Tasks", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 2, RawContent: "", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 3, RawContent: "This is some intro text with **markdown**.", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 4, RawContent: "", IndentLevel: 0}},
		},
		RootTasks: []*store.Task{
			{
				FileLine: store.FileLine{
					LineNumber:  5,
					RawContent:  "- [ ] 1. Task",
					IndentLevel: 0,
				},
				ID:          "1",
				Title:       "Task",
				Status:      store.StatusPending,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
		},
		Epilogue: []store.NonTaskContent{
			{FileLine: store.FileLine{LineNumber: 6, RawContent: "", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 7, RawContent: "---", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 8, RawContent: "Footer content", IndentLevel: 0}},
		},
	}

	got := p.Serialize(doc)
	expected := "# Project Tasks\n\nThis is some intro text with **markdown**.\n\n- [ ] 1. Task\n\n---\nFooter content"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_TaskWithDescription tests serializing a task with description lines
func TestSerialize_TaskWithDescription(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{},
		RootTasks: []*store.Task{
			{
				FileLine: store.FileLine{
					LineNumber:  1,
					RawContent:  "- [ ] 1. Task with description",
					IndentLevel: 0,
				},
				ID:     "1",
				Title:  "Task with description",
				Status: store.StatusPending,
				Description: []store.FileLine{
					{LineNumber: 2, RawContent: "  This is the description", IndentLevel: 1},
					{LineNumber: 3, RawContent: "  - Bullet point 1", IndentLevel: 1},
					{LineNumber: 4, RawContent: "  - Bullet point 2", IndentLevel: 1},
				},
				Children: []*store.Task{},
			},
		},
		Epilogue: []store.NonTaskContent{},
	}

	got := p.Serialize(doc)
	expected := "- [ ] 1. Task with description\n  This is the description\n  - Bullet point 1\n  - Bullet point 2"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_StatusChangesReflected tests that status changes are reflected in output
// Validates: Requirements 1.9
func TestSerialize_StatusChangesReflected(t *testing.T) {
	p := NewPrinter()

	// Task was originally pending (- [ ]) but status changed to done
	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{},
		RootTasks: []*store.Task{
			{
				FileLine: store.FileLine{
					LineNumber:  1,
					RawContent:  "- [ ] 1. Task that was pending",
					IndentLevel: 0,
				},
				ID:          "1",
				Title:       "Task that was pending",
				Status:      store.StatusDone, // Changed to done
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
		},
		Epilogue: []store.NonTaskContent{},
	}

	got := p.Serialize(doc)
	expected := "- [x] 1. Task that was pending" // Checkbox should be [x] now

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_MultipleRootTasks tests serializing multiple root tasks
func TestSerialize_MultipleRootTasks(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{},
		RootTasks: []*store.Task{
			{
				FileLine: store.FileLine{
					LineNumber:  1,
					RawContent:  "- [ ] 1. First root task",
					IndentLevel: 0,
				},
				ID:          "1",
				Title:       "First root task",
				Status:      store.StatusPending,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
			{
				FileLine: store.FileLine{
					LineNumber:  2,
					RawContent:  "- [-] 2. Second root task",
					IndentLevel: 0,
				},
				ID:          "2",
				Title:       "Second root task",
				Status:      store.StatusDoing,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
			{
				FileLine: store.FileLine{
					LineNumber:  3,
					RawContent:  "- [x] 3. Third root task",
					IndentLevel: 0,
				},
				ID:          "3",
				Title:       "Third root task",
				Status:      store.StatusDone,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
		},
		Epilogue: []store.NonTaskContent{},
	}

	got := p.Serialize(doc)
	expected := "- [ ] 1. First root task\n- [-] 2. Second root task\n- [x] 3. Third root task"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_ComplexDocument tests a complex document with all features
// Validates: Requirements 1.9, 1.10
func TestSerialize_ComplexDocument(t *testing.T) {
	p := NewPrinter()

	// Build a complex document structure
	child1 := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  5,
			RawContent:  "  - [x] 1.1 Create directory structure",
			IndentLevel: 1,
		},
		ID:          "1.1",
		Title:       "Create directory structure",
		Status:      store.StatusDone,
		Description: []store.FileLine{},
		Children:    []*store.Task{},
	}

	child2 := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  6,
			RawContent:  "  - [-] 1.2 Configure build system",
			IndentLevel: 1,
		},
		ID:          "1.2",
		Title:       "Configure build system",
		Status:      store.StatusDoing,
		Description: []store.FileLine{},
		Children:    []*store.Task{},
	}

	parent := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  3,
			RawContent:  "- [ ] 1. Setup project",
			IndentLevel: 0,
		},
		ID:     "1",
		Title:  "Setup project",
		Status: store.StatusPending,
		Description: []store.FileLine{
			{LineNumber: 4, RawContent: "  Initialize the project structure", IndentLevel: 1},
		},
		Children: []*store.Task{child1, child2},
	}

	child1.Parent = parent
	child2.Parent = parent

	task2 := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  7,
			RawContent:  "- [x] 2. Write documentation",
			IndentLevel: 0,
		},
		ID:     "2",
		Title:  "Write documentation",
		Status: store.StatusDone,
		Description: []store.FileLine{
			{LineNumber: 8, RawContent: "  Document the API", IndentLevel: 1},
		},
		Children: []*store.Task{},
	}

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{
			{FileLine: store.FileLine{LineNumber: 1, RawContent: "# Project Tasks", IndentLevel: 0}},
			{FileLine: store.FileLine{LineNumber: 2, RawContent: "", IndentLevel: 0}},
		},
		RootTasks: []*store.Task{parent, task2},
		Epilogue:  []store.NonTaskContent{},
	}

	got := p.Serialize(doc)
	expected := "# Project Tasks\n\n" +
		"- [ ] 1. Setup project\n" +
		"  Initialize the project structure\n" +
		"  - [x] 1.1 Create directory structure\n" +
		"  - [-] 1.2 Configure build system\n" +
		"- [x] 2. Write documentation\n" +
		"  Document the API"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_NoTrailingNewline tests that output has no trailing newline
func TestSerialize_NoTrailingNewline(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{
			{FileLine: store.FileLine{LineNumber: 1, RawContent: "# Tasks", IndentLevel: 0}},
		},
		RootTasks: []*store.Task{
			{
				FileLine: store.FileLine{
					LineNumber:  2,
					RawContent:  "- [ ] 1. Task",
					IndentLevel: 0,
				},
				ID:          "1",
				Title:       "Task",
				Status:      store.StatusPending,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
		},
		Epilogue: []store.NonTaskContent{},
	}

	got := p.Serialize(doc)

	if strings.HasSuffix(got, "\n") {
		t.Errorf("Serialize() output should not have trailing newline, got %q", got)
	}
}

// TestSerialize_EmptyPreambleAndEpilogue tests with empty preamble and epilogue
func TestSerialize_EmptyPreambleAndEpilogue(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{},
		RootTasks: []*store.Task{
			{
				FileLine: store.FileLine{
					LineNumber:  1,
					RawContent:  "- [ ] 1. Only task",
					IndentLevel: 0,
				},
				ID:          "1",
				Title:       "Only task",
				Status:      store.StatusPending,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
		},
		Epilogue: []store.NonTaskContent{},
	}

	got := p.Serialize(doc)
	expected := "- [ ] 1. Only task"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}

// TestSerialize_AllStatusTypes tests serializing tasks with all status types
func TestSerialize_AllStatusTypes(t *testing.T) {
	p := NewPrinter()

	doc := &store.ParsedDocument{
		Preamble: []store.NonTaskContent{},
		RootTasks: []*store.Task{
			{
				FileLine:    store.FileLine{LineNumber: 1, RawContent: "- [ ] 1. Pending", IndentLevel: 0},
				ID:          "1",
				Title:       "Pending",
				Status:      store.StatusPending,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
			{
				FileLine:    store.FileLine{LineNumber: 2, RawContent: "- [~] 2. Queued", IndentLevel: 0},
				ID:          "2",
				Title:       "Queued",
				Status:      store.StatusQueued,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
			{
				FileLine:    store.FileLine{LineNumber: 3, RawContent: "- [-] 3. Doing", IndentLevel: 0},
				ID:          "3",
				Title:       "Doing",
				Status:      store.StatusDoing,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
			{
				FileLine:    store.FileLine{LineNumber: 4, RawContent: "- [x] 4. Done", IndentLevel: 0},
				ID:          "4",
				Title:       "Done",
				Status:      store.StatusDone,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
			{
				FileLine:    store.FileLine{LineNumber: 5, RawContent: "- [!] 5. Failed", IndentLevel: 0},
				ID:          "5",
				Title:       "Failed",
				Status:      store.StatusFailed,
				Description: []store.FileLine{},
				Children:    []*store.Task{},
			},
		},
		Epilogue: []store.NonTaskContent{},
	}

	got := p.Serialize(doc)
	expected := "- [ ] 1. Pending\n- [~] 2. Queued\n- [-] 3. Doing\n- [x] 4. Done\n- [!] 5. Failed"

	if got != expected {
		t.Errorf("Serialize() = %q, want %q", got, expected)
	}
}
