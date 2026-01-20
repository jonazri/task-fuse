package parser

import (
	"testing"

	"task-fuse/internal/store"
)

func TestParseCheckbox_ValidCheckboxes(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		line     string
		wantStatus store.TaskStatus
		wantOk   bool
	}{
		// Pending checkbox - [ ]
		{
			name:       "pending checkbox basic",
			line:       "- [ ] Task title",
			wantStatus: store.StatusPending,
			wantOk:     true,
		},
		{
			name:       "pending checkbox with ID",
			line:       "- [ ] 1.1 Create structure",
			wantStatus: store.StatusPending,
			wantOk:     true,
		},
		{
			name:       "pending checkbox no title",
			line:       "- [ ]",
			wantStatus: store.StatusPending,
			wantOk:     true,
		},

		// Queued checkbox - [~]
		{
			name:       "queued checkbox basic",
			line:       "- [~] Task title",
			wantStatus: store.StatusQueued,
			wantOk:     true,
		},
		{
			name:       "queued checkbox with ID",
			line:       "- [~] 1.2 Queued for execution",
			wantStatus: store.StatusQueued,
			wantOk:     true,
		},

		// Doing checkbox - [-]
		{
			name:       "doing checkbox basic",
			line:       "- [-] Task title",
			wantStatus: store.StatusDoing,
			wantOk:     true,
		},
		{
			name:       "doing checkbox with ID",
			line:       "- [-] 1.3 Configure build",
			wantStatus: store.StatusDoing,
			wantOk:     true,
		},

		// Done checkbox - [x]
		{
			name:       "done checkbox basic",
			line:       "- [x] Task title",
			wantStatus: store.StatusDone,
			wantOk:     true,
		},
		{
			name:       "done checkbox with ID",
			line:       "- [x] 1.4 Setup complete",
			wantStatus: store.StatusDone,
			wantOk:     true,
		},

		// Failed checkbox - [!]
		{
			name:       "failed checkbox basic",
			line:       "- [!] Task title",
			wantStatus: store.StatusFailed,
			wantOk:     true,
		},
		{
			name:       "failed checkbox with ID",
			line:       "- [!] 1.5 Broken feature",
			wantStatus: store.StatusFailed,
			wantOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotOk := p.ParseCheckbox(tt.line)
			if gotOk != tt.wantOk {
				t.Errorf("ParseCheckbox() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("ParseCheckbox() status = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}

func TestParseCheckbox_WithLeadingWhitespace(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		line     string
		wantStatus store.TaskStatus
		wantOk   bool
	}{
		{
			name:       "two spaces indent",
			line:       "  - [ ] 1.1 Child task",
			wantStatus: store.StatusPending,
			wantOk:     true,
		},
		{
			name:       "four spaces indent",
			line:       "    - [x] 1.1.1 Grandchild",
			wantStatus: store.StatusDone,
			wantOk:     true,
		},
		{
			name:       "tab indent",
			line:       "\t- [-] 2.1 Another task",
			wantStatus: store.StatusDoing,
			wantOk:     true,
		},
		{
			name:       "mixed spaces and tabs",
			line:       "  \t- [~] 3.1 Mixed indent",
			wantStatus: store.StatusQueued,
			wantOk:     true,
		},
		{
			name:       "many spaces",
			line:       "        - [!] 4.1 Deep task",
			wantStatus: store.StatusFailed,
			wantOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotOk := p.ParseCheckbox(tt.line)
			if gotOk != tt.wantOk {
				t.Errorf("ParseCheckbox() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("ParseCheckbox() status = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}

func TestParseCheckbox_InvalidLines(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name string
		line string
	}{
		{
			name: "empty line",
			line: "",
		},
		{
			name: "plain text",
			line: "This is just text",
		},
		{
			name: "heading",
			line: "# Project Tasks",
		},
		{
			name: "bullet without checkbox",
			line: "- Just a bullet point",
		},
		{
			name: "numbered list",
			line: "1. Numbered item",
		},
		{
			name: "wrong bracket type",
			line: "- ( ) Task with parens",
		},
		{
			name: "missing space after dash",
			line: "-[ ] No space after dash",
		},
		{
			name: "missing space before bracket",
			line: "- [] Empty brackets",
		},
		{
			name: "invalid checkbox character",
			line: "- [?] Unknown status",
		},
		{
			name: "uppercase X",
			line: "- [X] Uppercase X",
		},
		{
			name: "multiple characters in checkbox",
			line: "- [xx] Multiple chars",
		},
		{
			name: "asterisk bullet",
			line: "* [ ] Asterisk bullet",
		},
		{
			name: "plus bullet",
			line: "+ [ ] Plus bullet",
		},
		{
			name: "checkbox in middle of line",
			line: "Some text - [ ] checkbox",
		},
		{
			name: "description line",
			line: "  Description text without checkbox",
		},
		{
			name: "code block",
			line: "```",
		},
		{
			name: "link syntax",
			line: "[link](url)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotOk := p.ParseCheckbox(tt.line)
			if gotOk {
				t.Errorf("ParseCheckbox() ok = true, want false for line: %q", tt.line)
			}
			if gotStatus != "" {
				t.Errorf("ParseCheckbox() status = %v, want empty string for invalid line", gotStatus)
			}
		})
	}
}

func TestParseCheckbox_EdgeCases(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name       string
		line       string
		wantStatus store.TaskStatus
		wantOk     bool
	}{
		{
			name:       "checkbox with deep task ID",
			line:       "- [x] 10.5.2 Deep task",
			wantStatus: store.StatusDone,
			wantOk:     true,
		},
		{
			name:       "checkbox with no ID (valid checkbox, no ID)",
			line:       "- [ ] No number here",
			wantStatus: store.StatusPending,
			wantOk:     true,
		},
		{
			name:       "checkbox followed by special chars",
			line:       "- [x] Task with @#$% special chars",
			wantStatus: store.StatusDone,
			wantOk:     true,
		},
		{
			name:       "checkbox with unicode",
			line:       "- [ ] Task with émojis 🎉",
			wantStatus: store.StatusPending,
			wantOk:     true,
		},
		{
			name:       "checkbox with trailing whitespace",
			line:       "- [-] Task with trailing space   ",
			wantStatus: store.StatusDoing,
			wantOk:     true,
		},
		{
			name:       "checkbox only with trailing space",
			line:       "- [~] ",
			wantStatus: store.StatusQueued,
			wantOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotOk := p.ParseCheckbox(tt.line)
			if gotOk != tt.wantOk {
				t.Errorf("ParseCheckbox() ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("ParseCheckbox() status = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}

func TestParseCheckbox_AllStatusMappings(t *testing.T) {
	// This test explicitly validates Requirements 1.1, 1.2, 1.3, 1.4, 1.5
	// by testing the exact checkbox syntax to status mapping
	p := NewParser()

	// Requirement 1.1: - [ ] → pending
	status, ok := p.ParseCheckbox("- [ ] Task")
	if !ok || status != store.StatusPending {
		t.Errorf("Requirement 1.1: - [ ] should map to pending, got %v, ok=%v", status, ok)
	}

	// Requirement 1.2: - [~] → queued
	status, ok = p.ParseCheckbox("- [~] Task")
	if !ok || status != store.StatusQueued {
		t.Errorf("Requirement 1.2: - [~] should map to queued, got %v, ok=%v", status, ok)
	}

	// Requirement 1.3: - [-] → doing
	status, ok = p.ParseCheckbox("- [-] Task")
	if !ok || status != store.StatusDoing {
		t.Errorf("Requirement 1.3: - [-] should map to doing, got %v, ok=%v", status, ok)
	}

	// Requirement 1.4: - [x] → done
	status, ok = p.ParseCheckbox("- [x] Task")
	if !ok || status != store.StatusDone {
		t.Errorf("Requirement 1.4: - [x] should map to done, got %v, ok=%v", status, ok)
	}

	// Requirement 1.5: - [!] → failed
	status, ok = p.ParseCheckbox("- [!] Task")
	if !ok || status != store.StatusFailed {
		t.Errorf("Requirement 1.5: - [!] should map to failed, got %v, ok=%v", status, ok)
	}
}


// =============================================================================
// ValidateTaskID Tests
// =============================================================================

func TestValidateTaskID_ValidIDs(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name string
		id   string
	}{
		// Single-level IDs
		{name: "single digit", id: "1"},
		{name: "double digit", id: "10"},
		{name: "triple digit", id: "100"},
		{name: "large number", id: "999"},

		// Two-level IDs
		{name: "two levels basic", id: "1.1"},
		{name: "two levels mixed", id: "1.10"},
		{name: "two levels large", id: "10.5"},
		{name: "two levels both large", id: "99.99"},

		// Three-level IDs
		{name: "three levels basic", id: "1.2.3"},
		{name: "three levels mixed", id: "10.5.2"},
		{name: "three levels large", id: "100.200.300"},

		// Deep IDs (up to max depth of 10)
		{name: "four levels", id: "1.2.3.4"},
		{name: "five levels", id: "1.2.3.4.5"},
		{name: "six levels", id: "1.2.3.4.5.6"},
		{name: "seven levels", id: "1.2.3.4.5.6.7"},
		{name: "eight levels", id: "1.2.3.4.5.6.7.8"},
		{name: "nine levels", id: "1.2.3.4.5.6.7.8.9"},
		{name: "ten levels (max)", id: "1.2.3.4.5.6.7.8.9.10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !p.ValidateTaskID(tt.id) {
				t.Errorf("ValidateTaskID(%q) = false, want true", tt.id)
			}
		})
	}
}

func TestValidateTaskID_InvalidIDs(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name   string
		id     string
		reason string
	}{
		// Empty and whitespace
		{name: "empty string", id: "", reason: "empty ID"},
		{name: "whitespace only", id: "   ", reason: "whitespace only"},

		// Leading zeros
		{name: "leading zero single", id: "01", reason: "leading zero"},
		{name: "leading zero first segment", id: "01.1", reason: "leading zero in first segment"},
		{name: "leading zero second segment", id: "1.01", reason: "leading zero in second segment"},
		{name: "leading zero deep", id: "1.2.03", reason: "leading zero in third segment"},
		{name: "multiple leading zeros", id: "001.002", reason: "multiple leading zeros"},

		// Zero as segment
		{name: "zero only", id: "0", reason: "zero is not a positive integer"},
		{name: "zero first segment", id: "0.1", reason: "zero in first segment"},
		{name: "zero second segment", id: "1.0", reason: "zero in second segment"},
		{name: "zero middle segment", id: "1.0.2", reason: "zero in middle segment"},

		// Non-numeric segments
		{name: "letter segment", id: "1.a", reason: "non-numeric segment"},
		{name: "letter in middle", id: "1.2.a", reason: "non-numeric segment"},
		{name: "mixed alphanumeric", id: "1.2a", reason: "mixed alphanumeric"},
		{name: "letter first", id: "a.1", reason: "letter in first segment"},
		{name: "special chars", id: "1.2#3", reason: "special characters"},

		// Empty segments
		{name: "double dot", id: "1..2", reason: "empty segment"},
		{name: "leading dot", id: ".1.2", reason: "leading dot"},
		{name: "trailing dot", id: "1.2.", reason: "trailing dot"},
		{name: "multiple empty segments", id: "1...2", reason: "multiple empty segments"},

		// Exceeds max depth (11 levels)
		{name: "eleven levels", id: "1.2.3.4.5.6.7.8.9.10.11", reason: "exceeds max depth"},
		{name: "twelve levels", id: "1.2.3.4.5.6.7.8.9.10.11.12", reason: "exceeds max depth"},

		// Other invalid formats
		{name: "negative number", id: "-1", reason: "negative number"},
		{name: "negative segment", id: "1.-2", reason: "negative segment"},
		{name: "spaces in ID", id: "1 2", reason: "spaces in ID"},
		{name: "spaces around dot", id: "1. 2", reason: "spaces around dot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if p.ValidateTaskID(tt.id) {
				t.Errorf("ValidateTaskID(%q) = true, want false (reason: %s)", tt.id, tt.reason)
			}
		})
	}
}

func TestValidateTaskID_MaxDepthBoundary(t *testing.T) {
	p := NewParser()

	// Test exactly at max depth (10 levels) - should be valid
	tenLevels := "1.2.3.4.5.6.7.8.9.10"
	if !p.ValidateTaskID(tenLevels) {
		t.Errorf("ValidateTaskID(%q) = false, want true (exactly at max depth)", tenLevels)
	}

	// Test one over max depth (11 levels) - should be invalid
	elevenLevels := "1.2.3.4.5.6.7.8.9.10.11"
	if p.ValidateTaskID(elevenLevels) {
		t.Errorf("ValidateTaskID(%q) = true, want false (exceeds max depth)", elevenLevels)
	}
}

// =============================================================================
// ExtractTaskID Tests
// =============================================================================

func TestExtractTaskID_ValidExtractions(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		line     string
		wantID   string
		wantOk   bool
	}{
		// Basic extractions
		{
			name:   "simple ID with title",
			line:   "1.1 Task title",
			wantID: "1.1",
			wantOk: true,
		},
		{
			name:   "deep ID with title",
			line:   "10.5.2 Deep task",
			wantID: "10.5.2",
			wantOk: true,
		},
		{
			name:   "single level ID",
			line:   "1 First task",
			wantID: "1",
			wantOk: true,
		},
		{
			name:   "ID with trailing dot",
			line:   "1. Task with dot",
			wantID: "1",
			wantOk: true,
		},
		{
			name:   "two level ID with trailing dot",
			line:   "1.2. Task with trailing dot",
			wantID: "1.2",
			wantOk: true,
		},

		// Various title formats
		{
			name:   "title with special chars",
			line:   "1.1 Task with @#$% special chars",
			wantID: "1.1",
			wantOk: true,
		},
		{
			name:   "title with unicode",
			line:   "2.1 Task with émojis 🎉",
			wantID: "2.1",
			wantOk: true,
		},
		{
			name:   "title with numbers",
			line:   "3.1 Task 123 with numbers",
			wantID: "3.1",
			wantOk: true,
		},
		{
			name:   "title starting with number",
			line:   "4.1 123 starts with number",
			wantID: "4.1",
			wantOk: true,
		},

		// Large IDs
		{
			name:   "large numbers",
			line:   "100.200.300 Large numbers",
			wantID: "100.200.300",
			wantOk: true,
		},
		{
			name:   "max depth ID",
			line:   "1.2.3.4.5.6.7.8.9.10 Max depth task",
			wantID: "1.2.3.4.5.6.7.8.9.10",
			wantOk: true,
		},

		// Multiple spaces after ID
		{
			name:   "multiple spaces after ID",
			line:   "1.1   Multiple spaces",
			wantID: "1.1",
			wantOk: true,
		},
		{
			name:   "tab after ID",
			line:   "1.1\tTab after ID",
			wantID: "1.1",
			wantOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOk := p.ExtractTaskID(tt.line)
			if gotOk != tt.wantOk {
				t.Errorf("ExtractTaskID(%q) ok = %v, want %v", tt.line, gotOk, tt.wantOk)
			}
			if gotID != tt.wantID {
				t.Errorf("ExtractTaskID(%q) id = %q, want %q", tt.line, gotID, tt.wantID)
			}
		})
	}
}

func TestExtractTaskID_InvalidExtractions(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name   string
		line   string
		reason string
	}{
		// No ID present
		{
			name:   "no number",
			line:   "No number here",
			reason: "no numeric ID",
		},
		{
			name:   "empty line",
			line:   "",
			reason: "empty line",
		},
		{
			name:   "only whitespace",
			line:   "   ",
			reason: "only whitespace",
		},
		{
			name:   "text only",
			line:   "Just some text",
			reason: "no ID",
		},

		// Leading zeros (invalid format)
		{
			name:   "leading zero",
			line:   "01.1 Leading zero",
			reason: "leading zero in first segment",
		},
		{
			name:   "leading zero second segment",
			line:   "1.01 Leading zero second",
			reason: "leading zero in second segment",
		},
		{
			name:   "leading zero deep",
			line:   "1.2.03 Leading zero deep",
			reason: "leading zero in third segment",
		},

		// Zero segments (invalid)
		{
			name:   "zero only",
			line:   "0 Zero task",
			reason: "zero is not positive",
		},
		{
			name:   "zero segment",
			line:   "1.0 Zero segment",
			reason: "zero segment",
		},

		// Non-numeric segments
		{
			name:   "letter segment",
			line:   "1.2.a Non-numeric",
			reason: "non-numeric segment",
		},
		{
			name:   "mixed alphanumeric",
			line:   "1.2a Mixed",
			reason: "mixed alphanumeric",
		},

		// Empty segments
		{
			name:   "double dot",
			line:   "1..2 Empty segment",
			reason: "empty segment",
		},

		// Exceeds max depth
		{
			name:   "exceeds max depth",
			line:   "1.2.3.4.5.6.7.8.9.10.11 Too deep",
			reason: "exceeds max depth",
		},

		// ID without space/title
		{
			name:   "ID only no space",
			line:   "1.1",
			reason: "no space after ID",
		},
		{
			name:   "ID with dot only",
			line:   "1.",
			reason: "no title after ID",
		},

		// Number not at start
		{
			name:   "number in middle",
			line:   "Task 1.1 in middle",
			reason: "ID not at start",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOk := p.ExtractTaskID(tt.line)
			if gotOk {
				t.Errorf("ExtractTaskID(%q) ok = true, want false (reason: %s), got ID=%q", tt.line, tt.reason, gotID)
			}
			if gotID != "" {
				t.Errorf("ExtractTaskID(%q) id = %q, want empty string", tt.line, gotID)
			}
		})
	}
}

func TestExtractTaskID_ExamplesFromDesign(t *testing.T) {
	// Test the exact examples from the design document
	p := NewParser()

	tests := []struct {
		name     string
		line     string
		wantID   string
		wantOk   bool
	}{
		// Valid examples from design
		{
			name:   "design example 1",
			line:   "1.1 Task title",
			wantID: "1.1",
			wantOk: true,
		},
		{
			name:   "design example 2",
			line:   "10.5.2 Deep task",
			wantID: "10.5.2",
			wantOk: true,
		},

		// Invalid examples from design
		{
			name:   "design invalid - leading zero",
			line:   "01.1 Leading zero",
			wantID: "",
			wantOk: false,
		},
		{
			name:   "design invalid - non-numeric",
			line:   "1.2.a Non-numeric",
			wantID: "",
			wantOk: false,
		},
		{
			name:   "design invalid - empty segment",
			line:   "1..2 Empty seg",
			wantID: "",
			wantOk: false,
		},
		{
			name:   "design invalid - no ID",
			line:   "No number here",
			wantID: "",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOk := p.ExtractTaskID(tt.line)
			if gotOk != tt.wantOk {
				t.Errorf("ExtractTaskID(%q) ok = %v, want %v", tt.line, gotOk, tt.wantOk)
			}
			if gotID != tt.wantID {
				t.Errorf("ExtractTaskID(%q) id = %q, want %q", tt.line, gotID, tt.wantID)
			}
		})
	}
}

// TestValidateTaskID_RequirementsValidation explicitly validates the requirements
// Validates: Requirements 1.5, 1.6, 1.11, Design - Task ID Format
func TestValidateTaskID_RequirementsValidation(t *testing.T) {
	p := NewParser()

	// Requirement 1.5/1.6: Task IDs are numeric identifiers like "1.1", "2.3.1"
	t.Run("Requirement 1.5/1.6 - numeric identifiers", func(t *testing.T) {
		validIDs := []string{"1", "1.1", "2.3.1", "10.5.2"}
		for _, id := range validIDs {
			if !p.ValidateTaskID(id) {
				t.Errorf("ValidateTaskID(%q) should be valid per Requirement 1.5/1.6", id)
			}
		}
	})

	// Requirement 1.11: Invalid formats like "01.1", "1.2.a", "1..2" should be rejected
	t.Run("Requirement 1.11 - invalid formats rejected", func(t *testing.T) {
		invalidIDs := []string{"01.1", "1.01", "1.2.a", "1..2", "a.1", "1.0"}
		for _, id := range invalidIDs {
			if p.ValidateTaskID(id) {
				t.Errorf("ValidateTaskID(%q) should be invalid per Requirement 1.11", id)
			}
		}
	})

	// Design - Task ID Format: Max depth of 10 levels
	t.Run("Design - max depth 10 levels", func(t *testing.T) {
		// 10 levels should be valid
		if !p.ValidateTaskID("1.2.3.4.5.6.7.8.9.10") {
			t.Error("10 levels should be valid per Design spec")
		}
		// 11 levels should be invalid
		if p.ValidateTaskID("1.2.3.4.5.6.7.8.9.10.11") {
			t.Error("11 levels should be invalid per Design spec")
		}
	})

	// Design - Task ID Format: No leading zeros
	t.Run("Design - no leading zeros", func(t *testing.T) {
		leadingZeroIDs := []string{"01", "01.1", "1.01", "001.002"}
		for _, id := range leadingZeroIDs {
			if p.ValidateTaskID(id) {
				t.Errorf("ValidateTaskID(%q) should reject leading zeros per Design spec", id)
			}
		}
	})
}


// =============================================================================
// ComputeIndentLevel Tests
// =============================================================================

func TestComputeIndentLevel_NoIndentation(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name string
		line string
		want int
	}{
		{
			name: "empty line",
			line: "",
			want: 0,
		},
		{
			name: "no leading whitespace",
			line: "- [ ] 1. Root task",
			want: 0,
		},
		{
			name: "text only",
			line: "Some text without indent",
			want: 0,
		},
		{
			name: "single space (less than 2)",
			line: " - [ ] 1. Task",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.ComputeIndentLevel(tt.line)
			if got != tt.want {
				t.Errorf("ComputeIndentLevel(%q) = %d, want %d", tt.line, got, tt.want)
			}
		})
	}
}

func TestComputeIndentLevel_SpacesOnly(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name   string
		line   string
		want   int
		reason string
	}{
		// 2 spaces = 1 level
		{
			name:   "2 spaces",
			line:   "  - [ ] 1.1 Child task",
			want:   1,
			reason: "2 spaces / 2 = 1",
		},
		// 3 spaces = 1 level (floor(3/2) = 1)
		{
			name:   "3 spaces",
			line:   "   - [ ] 1.1.2 Also grandchild",
			want:   1,
			reason: "floor(3/2) = 1",
		},
		// 4 spaces = 2 levels
		{
			name:   "4 spaces",
			line:   "    - [ ] 1.1.1 Grandchild",
			want:   2,
			reason: "4 spaces / 2 = 2",
		},
		// 5 spaces = 2 levels (floor(5/2) = 2)
		{
			name:   "5 spaces",
			line:   "     - [ ] Task",
			want:   2,
			reason: "floor(5/2) = 2",
		},
		// 6 spaces = 3 levels
		{
			name:   "6 spaces",
			line:   "      - [ ] Task",
			want:   3,
			reason: "6 spaces / 2 = 3",
		},
		// 7 spaces = 3 levels (floor(7/2) = 3)
		{
			name:   "7 spaces",
			line:   "       - [ ] Task",
			want:   3,
			reason: "floor(7/2) = 3",
		},
		// 8 spaces = 4 levels
		{
			name:   "8 spaces",
			line:   "        - [ ] Task",
			want:   4,
			reason: "8 spaces / 2 = 4",
		},
		// Deep indentation
		{
			name:   "20 spaces",
			line:   "                    - [ ] Deep task",
			want:   10,
			reason: "20 spaces / 2 = 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.ComputeIndentLevel(tt.line)
			if got != tt.want {
				t.Errorf("ComputeIndentLevel(%q) = %d, want %d (%s)", tt.line, got, tt.want, tt.reason)
			}
		})
	}
}

func TestComputeIndentLevel_TabsOnly(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name   string
		line   string
		want   int
		reason string
	}{
		// 1 tab = 2 spaces = 1 level
		{
			name:   "1 tab",
			line:   "\t- [ ] 1.1 Child task",
			want:   1,
			reason: "1 tab = 2 spaces, 2/2 = 1",
		},
		// 2 tabs = 4 spaces = 2 levels
		{
			name:   "2 tabs",
			line:   "\t\t- [ ] 1.1.1 Grandchild",
			want:   2,
			reason: "2 tabs = 4 spaces, 4/2 = 2",
		},
		// 3 tabs = 6 spaces = 3 levels
		{
			name:   "3 tabs",
			line:   "\t\t\t- [ ] Task",
			want:   3,
			reason: "3 tabs = 6 spaces, 6/2 = 3",
		},
		// 4 tabs = 8 spaces = 4 levels
		{
			name:   "4 tabs",
			line:   "\t\t\t\t- [ ] Task",
			want:   4,
			reason: "4 tabs = 8 spaces, 8/2 = 4",
		},
		// 5 tabs = 10 spaces = 5 levels
		{
			name:   "5 tabs",
			line:   "\t\t\t\t\t- [ ] Task",
			want:   5,
			reason: "5 tabs = 10 spaces, 10/2 = 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.ComputeIndentLevel(tt.line)
			if got != tt.want {
				t.Errorf("ComputeIndentLevel(%q) = %d, want %d (%s)", tt.line, got, tt.want, tt.reason)
			}
		})
	}
}

func TestComputeIndentLevel_MixedTabsAndSpaces(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name   string
		line   string
		want   int
		reason string
	}{
		// 1 tab + 2 spaces = 4 spaces = 2 levels
		{
			name:   "tab then 2 spaces",
			line:   "\t  - [ ] Task",
			want:   2,
			reason: "1 tab (2) + 2 spaces = 4, 4/2 = 2",
		},
		// 2 spaces + 1 tab = 4 spaces = 2 levels
		{
			name:   "2 spaces then tab",
			line:   "  \t- [ ] Task",
			want:   2,
			reason: "2 spaces + 1 tab (2) = 4, 4/2 = 2",
		},
		// 1 tab + 1 space = 3 spaces = 1 level (floor)
		{
			name:   "tab then 1 space",
			line:   "\t - [ ] Task",
			want:   1,
			reason: "1 tab (2) + 1 space = 3, floor(3/2) = 1",
		},
		// 1 space + 1 tab = 3 spaces = 1 level (floor)
		{
			name:   "1 space then tab",
			line:   " \t- [ ] Task",
			want:   1,
			reason: "1 space + 1 tab (2) = 3, floor(3/2) = 1",
		},
		// 2 tabs + 2 spaces = 6 spaces = 3 levels
		{
			name:   "2 tabs then 2 spaces",
			line:   "\t\t  - [ ] Task",
			want:   3,
			reason: "2 tabs (4) + 2 spaces = 6, 6/2 = 3",
		},
		// 2 spaces + 2 tabs = 6 spaces = 3 levels
		{
			name:   "2 spaces then 2 tabs",
			line:   "  \t\t- [ ] Task",
			want:   3,
			reason: "2 spaces + 2 tabs (4) = 6, 6/2 = 3",
		},
		// Alternating: space, tab, space, tab = 1 + 2 + 1 + 2 = 6 = 3 levels
		{
			name:   "alternating space tab space tab",
			line:   " \t \t- [ ] Task",
			want:   3,
			reason: "1 + 2 + 1 + 2 = 6, 6/2 = 3",
		},
		// Tab, space, tab = 2 + 1 + 2 = 5 = 2 levels (floor)
		{
			name:   "tab space tab",
			line:   "\t \t- [ ] Task",
			want:   2,
			reason: "2 + 1 + 2 = 5, floor(5/2) = 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.ComputeIndentLevel(tt.line)
			if got != tt.want {
				t.Errorf("ComputeIndentLevel(%q) = %d, want %d (%s)", tt.line, got, tt.want, tt.reason)
			}
		})
	}
}

func TestComputeIndentLevel_ExamplesFromRequirements(t *testing.T) {
	// Test the exact examples from the requirements document
	p := NewParser()

	tests := []struct {
		name   string
		line   string
		want   int
		reason string
	}{
		{
			name:   "root task (indent level 0)",
			line:   "- [ ] 1. Root task",
			want:   0,
			reason: "no indentation",
		},
		{
			name:   "child task (indent level 1, 2 spaces)",
			line:   "  - [ ] 1.1 Child task",
			want:   1,
			reason: "2 spaces = 1 level",
		},
		{
			name:   "grandchild (indent level 2, 4 spaces)",
			line:   "    - [ ] 1.1.1 Grandchild",
			want:   2,
			reason: "4 spaces = 2 levels",
		},
		{
			name:   "also grandchild (indent level 2, 3 spaces - still valid)",
			line:   "   - [ ] 1.1.2 Also grandchild",
			want:   1,
			reason: "3 spaces = floor(3/2) = 1 level (flexible spacing)",
		},
		{
			name:   "another root task (indent level 0)",
			line:   "- [ ] 2. Another root task",
			want:   0,
			reason: "no indentation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.ComputeIndentLevel(tt.line)
			if got != tt.want {
				t.Errorf("ComputeIndentLevel(%q) = %d, want %d (%s)", tt.line, got, tt.want, tt.reason)
			}
		})
	}
}

func TestComputeIndentLevel_EdgeCases(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name   string
		line   string
		want   int
		reason string
	}{
		// Only whitespace
		{
			name:   "only spaces",
			line:   "    ",
			want:   2,
			reason: "4 spaces / 2 = 2",
		},
		{
			name:   "only tabs",
			line:   "\t\t",
			want:   2,
			reason: "2 tabs = 4 spaces, 4/2 = 2",
		},
		// Whitespace followed by non-whitespace
		{
			name:   "spaces then text",
			line:   "  text",
			want:   1,
			reason: "2 spaces / 2 = 1",
		},
		// Description lines (indented text without checkbox)
		{
			name:   "description line 2 spaces",
			line:   "  Description text",
			want:   1,
			reason: "2 spaces / 2 = 1",
		},
		{
			name:   "description line 4 spaces",
			line:   "    More description",
			want:   2,
			reason: "4 spaces / 2 = 2",
		},
		// Very deep indentation
		{
			name:   "very deep spaces",
			line:   "                              - [ ] Very deep",
			want:   15,
			reason: "30 spaces / 2 = 15",
		},
		// Unicode content after whitespace
		{
			name:   "unicode after indent",
			line:   "  - [ ] 1.1 Task with émojis 🎉",
			want:   1,
			reason: "2 spaces / 2 = 1",
		},
		// Trailing whitespace (should not affect indent)
		{
			name:   "trailing whitespace",
			line:   "  - [ ] Task   ",
			want:   1,
			reason: "only leading whitespace counts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.ComputeIndentLevel(tt.line)
			if got != tt.want {
				t.Errorf("ComputeIndentLevel(%q) = %d, want %d (%s)", tt.line, got, tt.want, tt.reason)
			}
		})
	}
}

// TestComputeIndentLevel_DesignSpecValidation explicitly validates the design specification
// Validates: Design - Indentation Specification, Indent Level Computation
func TestComputeIndentLevel_DesignSpecValidation(t *testing.T) {
	p := NewParser()

	// Design spec: 1 tab = 2 spaces (internally converted)
	t.Run("Design spec - tab conversion", func(t *testing.T) {
		// 1 tab should equal 2 spaces
		tabLine := "\t- [ ] Task"
		spaceLine := "  - [ ] Task"
		tabLevel := p.ComputeIndentLevel(tabLine)
		spaceLevel := p.ComputeIndentLevel(spaceLine)
		if tabLevel != spaceLevel {
			t.Errorf("1 tab should equal 2 spaces: tab=%d, 2spaces=%d", tabLevel, spaceLevel)
		}
		if tabLevel != 1 {
			t.Errorf("1 tab should be indent level 1, got %d", tabLevel)
		}
	})

	// Design spec: 2-4 spaces = 1 indent level (flexible spacing)
	t.Run("Design spec - flexible spacing 2-4 spaces", func(t *testing.T) {
		// 2 spaces = 1 level
		if level := p.ComputeIndentLevel("  x"); level != 1 {
			t.Errorf("2 spaces should be level 1, got %d", level)
		}
		// 3 spaces = 1 level (floor(3/2) = 1)
		if level := p.ComputeIndentLevel("   x"); level != 1 {
			t.Errorf("3 spaces should be level 1 (floor), got %d", level)
		}
		// 4 spaces = 2 levels
		if level := p.ComputeIndentLevel("    x"); level != 2 {
			t.Errorf("4 spaces should be level 2, got %d", level)
		}
	})

	// Design spec: Mixed tabs/spaces - tabs converted first, then spaces counted
	t.Run("Design spec - mixed tabs and spaces", func(t *testing.T) {
		// 1 tab + 2 spaces = 2 + 2 = 4 spaces = 2 levels
		mixed := "\t  x"
		if level := p.ComputeIndentLevel(mixed); level != 2 {
			t.Errorf("1 tab + 2 spaces should be level 2, got %d", level)
		}

		// 2 spaces + 1 tab = 2 + 2 = 4 spaces = 2 levels
		mixed2 := "  \tx"
		if level := p.ComputeIndentLevel(mixed2); level != 2 {
			t.Errorf("2 spaces + 1 tab should be level 2, got %d", level)
		}
	})

	// Design spec: Result is floor(total_spaces / 2)
	t.Run("Design spec - floor division", func(t *testing.T) {
		// Odd number of spaces should use floor division
		// 1 space = floor(1/2) = 0
		if level := p.ComputeIndentLevel(" x"); level != 0 {
			t.Errorf("1 space should be level 0 (floor), got %d", level)
		}
		// 3 spaces = floor(3/2) = 1
		if level := p.ComputeIndentLevel("   x"); level != 1 {
			t.Errorf("3 spaces should be level 1 (floor), got %d", level)
		}
		// 5 spaces = floor(5/2) = 2
		if level := p.ComputeIndentLevel("     x"); level != 2 {
			t.Errorf("5 spaces should be level 2 (floor), got %d", level)
		}
		// 7 spaces = floor(7/2) = 3
		if level := p.ComputeIndentLevel("       x"); level != 3 {
			t.Errorf("7 spaces should be level 3 (floor), got %d", level)
		}
	})
}

// =============================================================================
// BuildHierarchy Tests
// =============================================================================

// Helper function to create a task with the given ID and indent level
func makeTask(id string, indentLevel int) *store.Task {
	return &store.Task{
		FileLine: store.FileLine{
			IndentLevel: indentLevel,
		},
		ID:     id,
		Title:  "Task " + id,
		Status: store.StatusPending,
	}
}

func TestBuildHierarchy_EmptyInput(t *testing.T) {
	p := NewParser()

	// Empty slice
	result := p.BuildHierarchy([]*store.Task{})
	if result != nil {
		t.Errorf("BuildHierarchy([]) = %v, want nil", result)
	}

	// Nil slice
	result = p.BuildHierarchy(nil)
	if result != nil {
		t.Errorf("BuildHierarchy(nil) = %v, want nil", result)
	}
}

func TestBuildHierarchy_SingleTask(t *testing.T) {
	p := NewParser()

	task := makeTask("1", 0)
	result := p.BuildHierarchy([]*store.Task{task})

	if len(result) != 1 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 1", len(result))
	}
	if result[0] != task {
		t.Error("BuildHierarchy() returned different task than input")
	}
	if task.Parent != nil {
		t.Error("Single root task should have nil parent")
	}
	if len(task.Children) != 0 {
		t.Error("Single task should have no children")
	}
}

func TestBuildHierarchy_MultipleRootTasks(t *testing.T) {
	p := NewParser()

	// All tasks at indent level 0 should be root tasks
	task1 := makeTask("1", 0)
	task2 := makeTask("2", 0)
	task3 := makeTask("3", 0)

	result := p.BuildHierarchy([]*store.Task{task1, task2, task3})

	if len(result) != 3 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 3", len(result))
	}

	// Verify all are root tasks with no parent
	for i, task := range result {
		if task.Parent != nil {
			t.Errorf("Root task %d should have nil parent", i+1)
		}
		if len(task.Children) != 0 {
			t.Errorf("Root task %d should have no children", i+1)
		}
	}
}

func TestBuildHierarchy_SimpleParentChild(t *testing.T) {
	p := NewParser()

	// Parent at level 0, child at level 1
	parent := makeTask("1", 0)
	child := makeTask("1.1", 1)

	result := p.BuildHierarchy([]*store.Task{parent, child})

	// Should have 1 root task
	if len(result) != 1 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 1", len(result))
	}

	// Parent should have 1 child
	if len(parent.Children) != 1 {
		t.Fatalf("Parent should have 1 child, got %d", len(parent.Children))
	}
	if parent.Children[0] != child {
		t.Error("Parent's child should be the child task")
	}

	// Child should have parent set
	if child.Parent != parent {
		t.Error("Child's parent should be the parent task")
	}
}

func TestBuildHierarchy_ThreeLevelHierarchy(t *testing.T) {
	p := NewParser()

	// Three levels: root -> child -> grandchild
	root := makeTask("1", 0)
	child := makeTask("1.1", 1)
	grandchild := makeTask("1.1.1", 2)

	result := p.BuildHierarchy([]*store.Task{root, child, grandchild})

	// Should have 1 root task
	if len(result) != 1 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 1", len(result))
	}

	// Root -> Child relationship
	if len(root.Children) != 1 || root.Children[0] != child {
		t.Error("Root should have child as its only child")
	}
	if child.Parent != root {
		t.Error("Child's parent should be root")
	}

	// Child -> Grandchild relationship
	if len(child.Children) != 1 || child.Children[0] != grandchild {
		t.Error("Child should have grandchild as its only child")
	}
	if grandchild.Parent != child {
		t.Error("Grandchild's parent should be child")
	}

	// Grandchild should have no children
	if len(grandchild.Children) != 0 {
		t.Error("Grandchild should have no children")
	}
}

func TestBuildHierarchy_MultipleSiblings(t *testing.T) {
	p := NewParser()

	// Parent with multiple children at same level
	parent := makeTask("1", 0)
	child1 := makeTask("1.1", 1)
	child2 := makeTask("1.2", 1)
	child3 := makeTask("1.3", 1)

	result := p.BuildHierarchy([]*store.Task{parent, child1, child2, child3})

	// Should have 1 root task
	if len(result) != 1 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 1", len(result))
	}

	// Parent should have 3 children
	if len(parent.Children) != 3 {
		t.Fatalf("Parent should have 3 children, got %d", len(parent.Children))
	}

	// All children should have parent set
	for i, child := range []*store.Task{child1, child2, child3} {
		if child.Parent != parent {
			t.Errorf("Child %d's parent should be the parent task", i+1)
		}
	}

	// Children should be in order
	if parent.Children[0] != child1 || parent.Children[1] != child2 || parent.Children[2] != child3 {
		t.Error("Children should be in order")
	}
}

func TestBuildHierarchy_ComplexHierarchy(t *testing.T) {
	p := NewParser()

	// Complex hierarchy from requirements example:
	// - [ ] 1. Root task              (indent level 0) → root
	//   - [ ] 1.1 Child task          (indent level 1) → child of 1
	//     - [ ] 1.1.1 Grandchild      (indent level 2) → child of 1.1
	//    - [ ] 1.1.2 Also grandchild  (indent level 1) → child of 1 (3 spaces = level 1)
	// - [ ] 2. Another root task      (indent level 0) → root

	task1 := makeTask("1", 0)
	task1_1 := makeTask("1.1", 1)
	task1_1_1 := makeTask("1.1.1", 2)
	task1_1_2 := makeTask("1.1.2", 1) // Same level as 1.1, so sibling of 1.1
	task2 := makeTask("2", 0)

	result := p.BuildHierarchy([]*store.Task{task1, task1_1, task1_1_1, task1_1_2, task2})

	// Should have 2 root tasks
	if len(result) != 2 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 2", len(result))
	}
	if result[0] != task1 || result[1] != task2 {
		t.Error("Root tasks should be task1 and task2 in order")
	}

	// Task 1 should have 2 children: 1.1 and 1.1.2
	if len(task1.Children) != 2 {
		t.Fatalf("Task 1 should have 2 children, got %d", len(task1.Children))
	}
	if task1.Children[0] != task1_1 || task1.Children[1] != task1_1_2 {
		t.Error("Task 1's children should be 1.1 and 1.1.2 in order")
	}

	// Task 1.1 should have 1 child: 1.1.1
	if len(task1_1.Children) != 1 || task1_1.Children[0] != task1_1_1 {
		t.Error("Task 1.1 should have 1.1.1 as its only child")
	}

	// Task 1.1.2 should have no children (it's at level 1, sibling of 1.1)
	if len(task1_1_2.Children) != 0 {
		t.Error("Task 1.1.2 should have no children")
	}

	// Task 2 should have no children
	if len(task2.Children) != 0 {
		t.Error("Task 2 should have no children")
	}

	// Verify parent relationships
	if task1.Parent != nil {
		t.Error("Task 1 should have nil parent")
	}
	if task1_1.Parent != task1 {
		t.Error("Task 1.1's parent should be task 1")
	}
	if task1_1_1.Parent != task1_1 {
		t.Error("Task 1.1.1's parent should be task 1.1")
	}
	if task1_1_2.Parent != task1 {
		t.Error("Task 1.1.2's parent should be task 1 (same level as 1.1)")
	}
	if task2.Parent != nil {
		t.Error("Task 2 should have nil parent")
	}
}

func TestBuildHierarchy_DecreasingIndent(t *testing.T) {
	p := NewParser()

	// Test that decreasing indent correctly closes subtrees
	// Level 0 -> Level 2 -> Level 1 -> Level 0
	task1 := makeTask("1", 0)
	task1_1_1 := makeTask("1.1.1", 2) // Skips level 1, but still child of task1
	task1_2 := makeTask("1.2", 1)     // Back to level 1, sibling of 1.1.1's implied parent
	task2 := makeTask("2", 0)         // Back to root

	result := p.BuildHierarchy([]*store.Task{task1, task1_1_1, task1_2, task2})

	// Should have 2 root tasks
	if len(result) != 2 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 2", len(result))
	}

	// Task 1 should have 2 children: 1.1.1 (at level 2) and 1.2 (at level 1)
	if len(task1.Children) != 2 {
		t.Fatalf("Task 1 should have 2 children, got %d", len(task1.Children))
	}

	// Both should be direct children of task1 since there's no intermediate parent
	if task1_1_1.Parent != task1 {
		t.Error("Task 1.1.1's parent should be task 1 (no intermediate parent)")
	}
	if task1_2.Parent != task1 {
		t.Error("Task 1.2's parent should be task 1")
	}
}

func TestBuildHierarchy_DeepNesting(t *testing.T) {
	p := NewParser()

	// Test deep nesting (5 levels)
	tasks := []*store.Task{
		makeTask("1", 0),
		makeTask("1.1", 1),
		makeTask("1.1.1", 2),
		makeTask("1.1.1.1", 3),
		makeTask("1.1.1.1.1", 4),
	}

	result := p.BuildHierarchy(tasks)

	// Should have 1 root task
	if len(result) != 1 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 1", len(result))
	}

	// Verify chain of parent-child relationships
	current := tasks[0]
	for i := 1; i < len(tasks); i++ {
		if len(current.Children) != 1 {
			t.Errorf("Task at level %d should have 1 child", i-1)
		}
		if current.Children[0] != tasks[i] {
			t.Errorf("Task at level %d's child should be task at level %d", i-1, i)
		}
		if tasks[i].Parent != current {
			t.Errorf("Task at level %d's parent should be task at level %d", i, i-1)
		}
		current = tasks[i]
	}

	// Last task should have no children
	if len(tasks[len(tasks)-1].Children) != 0 {
		t.Error("Deepest task should have no children")
	}
}

func TestBuildHierarchy_SameIndentLevel(t *testing.T) {
	p := NewParser()

	// All tasks at same non-zero indent level
	// Without a parent at lower level, they should all be roots
	task1 := makeTask("1", 1)
	task2 := makeTask("2", 1)
	task3 := makeTask("3", 1)

	result := p.BuildHierarchy([]*store.Task{task1, task2, task3})

	// All should be root tasks since there's no task at level 0
	if len(result) != 3 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 3", len(result))
	}

	for _, task := range result {
		if task.Parent != nil {
			t.Errorf("Task %s should have nil parent", task.ID)
		}
	}
}

func TestBuildHierarchy_JumpInIndent(t *testing.T) {
	p := NewParser()

	// Test jumping from level 0 directly to level 3
	// The level 3 task should still be a child of level 0
	root := makeTask("1", 0)
	deep := makeTask("1.1.1.1", 3)

	result := p.BuildHierarchy([]*store.Task{root, deep})

	if len(result) != 1 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 1", len(result))
	}

	if len(root.Children) != 1 || root.Children[0] != deep {
		t.Error("Root should have deep task as direct child")
	}
	if deep.Parent != root {
		t.Error("Deep task's parent should be root")
	}
}

func TestBuildHierarchy_RequirementsExample(t *testing.T) {
	// Test the exact example from the requirements document
	// - [ ] 1. Root task              (indent level 0) → root
	//   - [ ] 1.1 Child task          (indent level 1) → child of 1
	//     - [ ] 1.1.1 Grandchild      (indent level 2) → child of 1.1
	//    - [ ] 1.1.2 Also grandchild  (indent level 2) → child of 1.1 (3 spaces = level 1, but still child of 1.1)
	// - [ ] 2. Another root task      (indent level 0) → root

	p := NewParser()

	// Note: The requirements example says 1.1.2 at "3 spaces = level 1" but says it's "child of 1.1"
	// This is actually inconsistent - if it's level 1, it would be a sibling of 1.1, not a child.
	// The design doc clarifies: hierarchy is determined by relative indent comparison.
	// If 1.1.2 is at level 2 (4 spaces), it would be child of 1.1.
	// If 1.1.2 is at level 1 (3 spaces = floor(3/2) = 1), it would be sibling of 1.1.

	// Let's test the case where 1.1.2 is at level 2 (making it child of 1.1)
	task1 := makeTask("1", 0)
	task1_1 := makeTask("1.1", 1)
	task1_1_1 := makeTask("1.1.1", 2)
	task1_1_2 := makeTask("1.1.2", 2) // Level 2, child of 1.1
	task2 := makeTask("2", 0)

	result := p.BuildHierarchy([]*store.Task{task1, task1_1, task1_1_1, task1_1_2, task2})

	// Should have 2 root tasks
	if len(result) != 2 {
		t.Fatalf("BuildHierarchy() returned %d root tasks, want 2", len(result))
	}

	// Task 1 should have 1 child: 1.1
	if len(task1.Children) != 1 || task1.Children[0] != task1_1 {
		t.Error("Task 1 should have 1.1 as its only child")
	}

	// Task 1.1 should have 2 children: 1.1.1 and 1.1.2
	if len(task1_1.Children) != 2 {
		t.Fatalf("Task 1.1 should have 2 children, got %d", len(task1_1.Children))
	}
	if task1_1.Children[0] != task1_1_1 || task1_1.Children[1] != task1_1_2 {
		t.Error("Task 1.1's children should be 1.1.1 and 1.1.2 in order")
	}

	// Verify parent relationships
	if task1_1_1.Parent != task1_1 {
		t.Error("Task 1.1.1's parent should be task 1.1")
	}
	if task1_1_2.Parent != task1_1 {
		t.Error("Task 1.1.2's parent should be task 1.1")
	}
}

func TestBuildHierarchy_PreservesTaskData(t *testing.T) {
	p := NewParser()

	// Verify that BuildHierarchy doesn't modify task data other than Parent/Children
	task := &store.Task{
		FileLine: store.FileLine{
			LineNumber:  42,
			RawContent:  "- [ ] 1. Test task",
			IndentLevel: 0,
		},
		ID:          "1",
		Title:       "Test task",
		Status:      store.StatusDoing,
		Description: []store.FileLine{{LineNumber: 43, RawContent: "  Description"}},
	}

	result := p.BuildHierarchy([]*store.Task{task})

	if len(result) != 1 {
		t.Fatal("Expected 1 root task")
	}

	// Verify original data is preserved
	if task.LineNumber != 42 {
		t.Error("LineNumber should be preserved")
	}
	if task.RawContent != "- [ ] 1. Test task" {
		t.Error("RawContent should be preserved")
	}
	if task.ID != "1" {
		t.Error("ID should be preserved")
	}
	if task.Title != "Test task" {
		t.Error("Title should be preserved")
	}
	if task.Status != store.StatusDoing {
		t.Error("Status should be preserved")
	}
	if len(task.Description) != 1 || task.Description[0].RawContent != "  Description" {
		t.Error("Description should be preserved")
	}
}

// TestBuildHierarchy_Requirement1_7 explicitly validates Requirement 1.7
// Validates: Requirements 1.7 - hierarchical tree structure based on indentation
func TestBuildHierarchy_Requirement1_7(t *testing.T) {
	p := NewParser()

	t.Run("Requirement 1.7 - nested sub-tasks build hierarchical tree", func(t *testing.T) {
		// Create a hierarchy based on indentation
		root := makeTask("1", 0)
		child := makeTask("1.1", 1)
		grandchild := makeTask("1.1.1", 2)

		result := p.BuildHierarchy([]*store.Task{root, child, grandchild})

		// Verify hierarchical tree structure
		if len(result) != 1 {
			t.Fatal("Should have 1 root task")
		}
		if result[0] != root {
			t.Error("Root should be the first task")
		}
		if len(root.Children) != 1 || root.Children[0] != child {
			t.Error("Root should have child as its child")
		}
		if len(child.Children) != 1 || child.Children[0] != grandchild {
			t.Error("Child should have grandchild as its child")
		}
	})

	t.Run("Requirement 1.7 - child of nearest preceding task with less indentation", func(t *testing.T) {
		// A task is a child of the nearest preceding task with less indentation
		task1 := makeTask("1", 0)
		task1_1 := makeTask("1.1", 1)
		task1_2 := makeTask("1.2", 1) // Should be sibling of 1.1, not child

		p.BuildHierarchy([]*store.Task{task1, task1_1, task1_2})

		// Both 1.1 and 1.2 should be children of 1
		if len(task1.Children) != 2 {
			t.Fatalf("Task 1 should have 2 children, got %d", len(task1.Children))
		}
		if task1_1.Parent != task1 || task1_2.Parent != task1 {
			t.Error("Both 1.1 and 1.2 should have task 1 as parent")
		}
	})

	t.Run("Requirement 1.7 - root tasks have zero indentation", func(t *testing.T) {
		task1 := makeTask("1", 0)
		task2 := makeTask("2", 0)

		result := p.BuildHierarchy([]*store.Task{task1, task2})

		if len(result) != 2 {
			t.Fatal("Should have 2 root tasks")
		}
		if task1.Parent != nil || task2.Parent != nil {
			t.Error("Root tasks should have nil parent")
		}
	})
}

// =============================================================================
// GetIDPrefixes Tests
// =============================================================================

func TestGetIDPrefixes_BasicCases(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		id       string
		expected []string
	}{
		{
			name:     "single level ID",
			id:       "1",
			expected: []string{"1"},
		},
		{
			name:     "two level ID",
			id:       "1.2",
			expected: []string{"1", "1.2"},
		},
		{
			name:     "three level ID",
			id:       "1.2.3",
			expected: []string{"1", "1.2", "1.2.3"},
		},
		{
			name:     "four level ID",
			id:       "1.2.3.4",
			expected: []string{"1", "1.2", "1.2.3", "1.2.3.4"},
		},
		{
			name:     "large numbers",
			id:       "10.5.2",
			expected: []string{"10", "10.5", "10.5.2"},
		},
		{
			name:     "max depth ID",
			id:       "1.2.3.4.5.6.7.8.9.10",
			expected: []string{"1", "1.2", "1.2.3", "1.2.3.4", "1.2.3.4.5", "1.2.3.4.5.6", "1.2.3.4.5.6.7", "1.2.3.4.5.6.7.8", "1.2.3.4.5.6.7.8.9", "1.2.3.4.5.6.7.8.9.10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.GetIDPrefixes(tt.id)
			if len(result) != len(tt.expected) {
				t.Errorf("GetIDPrefixes(%q) returned %d prefixes, want %d", tt.id, len(result), len(tt.expected))
				return
			}
			for i, prefix := range result {
				if prefix != tt.expected[i] {
					t.Errorf("GetIDPrefixes(%q)[%d] = %q, want %q", tt.id, i, prefix, tt.expected[i])
				}
			}
		})
	}
}

func TestGetIDPrefixes_EdgeCases(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		id       string
		expected []string
	}{
		{
			name:     "empty string",
			id:       "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.GetIDPrefixes(tt.id)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("GetIDPrefixes(%q) = %v, want nil", tt.id, result)
				}
			} else if len(result) != len(tt.expected) {
				t.Errorf("GetIDPrefixes(%q) returned %d prefixes, want %d", tt.id, len(result), len(tt.expected))
			}
		})
	}
}

// =============================================================================
// FindAncestorByIDPrefix Tests
// =============================================================================

func TestFindAncestorByIDPrefix_BasicCases(t *testing.T) {
	p := NewParser()

	t.Run("finds immediate parent", func(t *testing.T) {
		// Task 1.2.3 exists, 1.2 exists → should find 1.2
		task1 := makeTask("1", 0)
		task1_2 := makeTask("1.2", 1)
		tasks := []*store.Task{task1, task1_2}

		result := p.FindAncestorByIDPrefix("1.2.3", tasks)
		if result != task1_2 {
			t.Errorf("FindAncestorByIDPrefix(\"1.2.3\") should find task 1.2, got %v", result)
		}
	})

	t.Run("finds grandparent when parent missing", func(t *testing.T) {
		// Task 1.2.3 exists, 1.2 does NOT exist, 1 exists → should find 1
		task1 := makeTask("1", 0)
		tasks := []*store.Task{task1}

		result := p.FindAncestorByIDPrefix("1.2.3", tasks)
		if result != task1 {
			t.Errorf("FindAncestorByIDPrefix(\"1.2.3\") should find task 1, got %v", result)
		}
	})

	t.Run("returns nil when no ancestor exists", func(t *testing.T) {
		// Task 1.2.3 exists, neither 1.2 nor 1 exist → should return nil
		task2 := makeTask("2", 0)
		tasks := []*store.Task{task2}

		result := p.FindAncestorByIDPrefix("1.2.3", tasks)
		if result != nil {
			t.Errorf("FindAncestorByIDPrefix(\"1.2.3\") should return nil when no ancestor exists, got %v", result)
		}
	})

	t.Run("returns nil for root task ID", func(t *testing.T) {
		// Task 1 has no parent by definition
		task2 := makeTask("2", 0)
		tasks := []*store.Task{task2}

		result := p.FindAncestorByIDPrefix("1", tasks)
		if result != nil {
			t.Errorf("FindAncestorByIDPrefix(\"1\") should return nil for root task ID, got %v", result)
		}
	})
}

func TestFindAncestorByIDPrefix_DesignExample(t *testing.T) {
	// Test the exact example from the design document:
	// If task 1.2.3 exists but 1.2 does not:
	// - Check if 1.2 exists → No
	// - Check if 1 exists → Yes
	// - Attach 1.2.3 as child of 1
	p := NewParser()

	task1 := makeTask("1", 0)
	task1_1 := makeTask("1.1", 1)
	// Note: task 1.2 is intentionally missing
	tasks := []*store.Task{task1, task1_1}

	result := p.FindAncestorByIDPrefix("1.2.3", tasks)
	if result != task1 {
		t.Errorf("Design example: FindAncestorByIDPrefix(\"1.2.3\") should find task 1, got %v", result)
	}
}

func TestFindAncestorByIDPrefix_DeepHierarchy(t *testing.T) {
	p := NewParser()

	t.Run("finds nearest ancestor in deep hierarchy", func(t *testing.T) {
		// Task 1.2.3.4.5 exists, only 1 and 1.2 exist
		task1 := makeTask("1", 0)
		task1_2 := makeTask("1.2", 1)
		tasks := []*store.Task{task1, task1_2}

		result := p.FindAncestorByIDPrefix("1.2.3.4.5", tasks)
		if result != task1_2 {
			t.Errorf("FindAncestorByIDPrefix(\"1.2.3.4.5\") should find nearest ancestor 1.2, got %v", result)
		}
	})

	t.Run("finds root when all intermediate ancestors missing", func(t *testing.T) {
		// Task 1.2.3.4.5 exists, only 1 exists
		task1 := makeTask("1", 0)
		tasks := []*store.Task{task1}

		result := p.FindAncestorByIDPrefix("1.2.3.4.5", tasks)
		if result != task1 {
			t.Errorf("FindAncestorByIDPrefix(\"1.2.3.4.5\") should find root 1, got %v", result)
		}
	})
}

func TestFindAncestorByIDPrefix_EdgeCases(t *testing.T) {
	p := NewParser()

	t.Run("empty ID returns nil", func(t *testing.T) {
		task1 := makeTask("1", 0)
		tasks := []*store.Task{task1}

		result := p.FindAncestorByIDPrefix("", tasks)
		if result != nil {
			t.Errorf("FindAncestorByIDPrefix(\"\") should return nil, got %v", result)
		}
	})

	t.Run("empty task list returns nil", func(t *testing.T) {
		result := p.FindAncestorByIDPrefix("1.2.3", []*store.Task{})
		if result != nil {
			t.Errorf("FindAncestorByIDPrefix with empty task list should return nil, got %v", result)
		}
	})

	t.Run("nil task list returns nil", func(t *testing.T) {
		result := p.FindAncestorByIDPrefix("1.2.3", nil)
		if result != nil {
			t.Errorf("FindAncestorByIDPrefix with nil task list should return nil, got %v", result)
		}
	})

	t.Run("task with nil in list is skipped", func(t *testing.T) {
		task1 := makeTask("1", 0)
		tasks := []*store.Task{nil, task1, nil}

		result := p.FindAncestorByIDPrefix("1.2.3", tasks)
		if result != task1 {
			t.Errorf("FindAncestorByIDPrefix should skip nil tasks and find 1, got %v", result)
		}
	})

	t.Run("task with empty ID is skipped", func(t *testing.T) {
		task1 := makeTask("1", 0)
		emptyIDTask := &store.Task{ID: ""}
		tasks := []*store.Task{emptyIDTask, task1}

		result := p.FindAncestorByIDPrefix("1.2.3", tasks)
		if result != task1 {
			t.Errorf("FindAncestorByIDPrefix should skip tasks with empty ID and find 1, got %v", result)
		}
	})
}

func TestFindAncestorByIDPrefix_DoesNotFindSelf(t *testing.T) {
	p := NewParser()

	// The function should not return the task itself as its own ancestor
	task1_2_3 := makeTask("1.2.3", 2)
	tasks := []*store.Task{task1_2_3}

	result := p.FindAncestorByIDPrefix("1.2.3", tasks)
	if result != nil {
		t.Errorf("FindAncestorByIDPrefix should not return the task itself as ancestor, got %v", result)
	}
}

func TestFindAncestorByIDPrefix_MultipleRoots(t *testing.T) {
	p := NewParser()

	// Multiple root tasks exist, orphaned task should find correct ancestor
	task1 := makeTask("1", 0)
	task2 := makeTask("2", 0)
	task3 := makeTask("3", 0)
	tasks := []*store.Task{task1, task2, task3}

	t.Run("finds correct root for 1.x.x", func(t *testing.T) {
		result := p.FindAncestorByIDPrefix("1.2.3", tasks)
		if result != task1 {
			t.Errorf("FindAncestorByIDPrefix(\"1.2.3\") should find task 1, got %v", result)
		}
	})

	t.Run("finds correct root for 2.x.x", func(t *testing.T) {
		result := p.FindAncestorByIDPrefix("2.5.7", tasks)
		if result != task2 {
			t.Errorf("FindAncestorByIDPrefix(\"2.5.7\") should find task 2, got %v", result)
		}
	})

	t.Run("finds correct root for 3.x.x", func(t *testing.T) {
		result := p.FindAncestorByIDPrefix("3.1", tasks)
		if result != task3 {
			t.Errorf("FindAncestorByIDPrefix(\"3.1\") should find task 3, got %v", result)
		}
	})
}

func TestFindAncestorByIDPrefix_LargeNumbers(t *testing.T) {
	p := NewParser()

	// Test with large task IDs
	task10 := makeTask("10", 0)
	task10_5 := makeTask("10.5", 1)
	tasks := []*store.Task{task10, task10_5}

	result := p.FindAncestorByIDPrefix("10.5.2", tasks)
	if result != task10_5 {
		t.Errorf("FindAncestorByIDPrefix(\"10.5.2\") should find task 10.5, got %v", result)
	}
}

// TestFindAncestorByIDPrefix_Requirement1_12 explicitly validates Requirement 1.12
// Validates: Requirements 1.12 - orphaned ID resolution
func TestFindAncestorByIDPrefix_Requirement1_12(t *testing.T) {
	p := NewParser()

	t.Run("Requirement 1.12 - attach to nearest valid ancestor by ID prefix", func(t *testing.T) {
		// Create a scenario where 1.2.3 exists but 1.2 does not
		task1 := makeTask("1", 0)
		task1_1 := makeTask("1.1", 1)
		// 1.2 is missing
		tasks := []*store.Task{task1, task1_1}

		// FindAncestorByIDPrefix should find task 1 as the nearest ancestor
		result := p.FindAncestorByIDPrefix("1.2.3", tasks)
		if result != task1 {
			t.Errorf("Requirement 1.12: orphaned task 1.2.3 should attach to nearest ancestor 1, got %v", result)
		}
	})

	t.Run("Requirement 1.12 - treat as root if no ancestor exists", func(t *testing.T) {
		// Create a scenario where 5.1.2 exists but neither 5.1 nor 5 exist
		task1 := makeTask("1", 0)
		task2 := makeTask("2", 0)
		tasks := []*store.Task{task1, task2}

		// FindAncestorByIDPrefix should return nil (treat as root)
		result := p.FindAncestorByIDPrefix("5.1.2", tasks)
		if result != nil {
			t.Errorf("Requirement 1.12: orphaned task 5.1.2 with no ancestor should be treated as root (nil), got %v", result)
		}
	})

	t.Run("Requirement 1.12 - finds immediate parent when it exists", func(t *testing.T) {
		// Normal case: 1.2 exists, so 1.2.3 should find 1.2
		task1 := makeTask("1", 0)
		task1_2 := makeTask("1.2", 1)
		tasks := []*store.Task{task1, task1_2}

		result := p.FindAncestorByIDPrefix("1.2.3", tasks)
		if result != task1_2 {
			t.Errorf("Requirement 1.12: task 1.2.3 should find immediate parent 1.2, got %v", result)
		}
	})
}
