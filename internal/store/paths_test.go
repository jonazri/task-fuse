package store

import (
	"strings"
	"testing"
)

// TestSlugify_BasicConversion tests basic slugification rules.
func TestSlugify_BasicConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple title",
			input:    "Implement the Parser",
			expected: "implement_the_parser",
		},
		{
			name:     "with special characters",
			input:    "Add UTF-8 Support!!!",
			expected: "add_utf-8_support",
		},
		{
			name:     "trim spaces",
			input:    "   Trim   Spaces   ",
			expected: "trim_spaces",
		},
		{
			name:     "uppercase and lowercase",
			input:    "UPPERCASE lowercase",
			expected: "uppercase_lowercase",
		},
		{
			name:     "special chars only",
			input:    "Special @#$% chars",
			expected: "special_chars",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "task",
		},
		{
			name:     "only special characters",
			input:    "@#$%^&*()",
			expected: "task",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n   ",
			expected: "task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSlugify_LowercaseConversion tests that all characters are converted to lowercase.
func TestSlugify_LowercaseConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "all uppercase",
			input:    "ALL UPPERCASE",
			expected: "all_uppercase",
		},
		{
			name:     "mixed case",
			input:    "MiXeD CaSe TiTlE",
			expected: "mixed_case_title",
		},
		{
			name:     "camelCase",
			input:    "camelCaseTitle",
			expected: "camelcasetitle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSlugify_WhitespaceHandling tests whitespace replacement with underscores.
func TestSlugify_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single space",
			input:    "hello world",
			expected: "hello_world",
		},
		{
			name:     "multiple spaces",
			input:    "hello    world",
			expected: "hello_world",
		},
		{
			name:     "tabs",
			input:    "hello\tworld",
			expected: "hello_world",
		},
		{
			name:     "newlines",
			input:    "hello\nworld",
			expected: "hello_world",
		},
		{
			name:     "mixed whitespace",
			input:    "hello \t\n world",
			expected: "hello_world",
		},
		{
			name:     "leading whitespace",
			input:    "   hello",
			expected: "hello",
		},
		{
			name:     "trailing whitespace",
			input:    "hello   ",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSlugify_AllowedCharacters tests that only [a-z0-9_-] are kept.
func TestSlugify_AllowedCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "numbers preserved",
			input:    "task123",
			expected: "task123",
		},
		{
			name:     "hyphens preserved",
			input:    "task-name",
			expected: "task-name",
		},
		{
			name:     "underscores preserved",
			input:    "task_name",
			expected: "task_name",
		},
		{
			name:     "punctuation removed",
			input:    "hello, world!",
			expected: "hello_world",
		},
		{
			name:     "brackets removed",
			input:    "task [important]",
			expected: "task_important",
		},
		{
			name:     "parentheses removed",
			input:    "task (urgent)",
			expected: "task_urgent",
		},
		{
			name:     "quotes removed",
			input:    `task "quoted"`,
			expected: "task_quoted",
		},
		{
			name:     "at symbol removed",
			input:    "task@work",
			expected: "taskwork",
		},
		{
			name:     "hash removed",
			input:    "task#1",
			expected: "task1",
		},
		{
			name:     "dollar removed",
			input:    "task$money",
			expected: "taskmoney",
		},
		{
			name:     "percent removed",
			input:    "task%complete",
			expected: "taskcomplete",
		},
		{
			name:     "ampersand removed",
			input:    "task&work",
			expected: "taskwork",
		},
		{
			name:     "asterisk removed",
			input:    "task*important",
			expected: "taskimportant",
		},
		{
			name:     "plus removed",
			input:    "task+more",
			expected: "taskmore",
		},
		{
			name:     "equals removed",
			input:    "task=value",
			expected: "taskvalue",
		},
		{
			name:     "colon removed",
			input:    "task:name",
			expected: "taskname",
		},
		{
			name:     "semicolon removed",
			input:    "task;name",
			expected: "taskname",
		},
		{
			name:     "slash removed",
			input:    "task/name",
			expected: "taskname",
		},
		{
			name:     "backslash removed",
			input:    "task\\name",
			expected: "taskname",
		},
		{
			name:     "pipe removed",
			input:    "task|name",
			expected: "taskname",
		},
		{
			name:     "caret removed",
			input:    "task^name",
			expected: "taskname",
		},
		{
			name:     "tilde removed",
			input:    "task~name",
			expected: "taskname",
		},
		{
			name:     "backtick removed",
			input:    "task`name",
			expected: "taskname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSlugify_UnderscoreCollapsing tests that multiple underscores are collapsed.
func TestSlugify_UnderscoreCollapsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "double underscore",
			input:    "hello__world",
			expected: "hello_world",
		},
		{
			name:     "triple underscore",
			input:    "hello___world",
			expected: "hello_world",
		},
		{
			name:     "many underscores",
			input:    "hello______world",
			expected: "hello_world",
		},
		{
			name:     "underscores from special chars",
			input:    "hello @#$ world",
			expected: "hello_world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSlugify_EdgeTrimming tests that leading/trailing underscores are trimmed.
func TestSlugify_EdgeTrimming(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "leading underscore",
			input:    "_hello",
			expected: "hello",
		},
		{
			name:     "trailing underscore",
			input:    "hello_",
			expected: "hello",
		},
		{
			name:     "both edges",
			input:    "_hello_",
			expected: "hello",
		},
		{
			name:     "multiple leading",
			input:    "___hello",
			expected: "hello",
		},
		{
			name:     "multiple trailing",
			input:    "hello___",
			expected: "hello",
		},
		{
			name:     "from special chars at edges",
			input:    "!!!hello!!!",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSlugify_Truncation tests truncation to 50 characters at word boundary.
func TestSlugify_Truncation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "exactly 50 chars",
			input:    "12345678901234567890123456789012345678901234567890",
			expected: "12345678901234567890123456789012345678901234567890",
		},
		{
			name:     "51 chars no underscore",
			input:    "123456789012345678901234567890123456789012345678901",
			expected: "12345678901234567890123456789012345678901234567890",
		},
		{
			name:     "long title with words - truncate at 50 chars",
			input:    "A very long title that exceeds the maximum allowed length for filenames",
			expected: "a_very_long_title_that_exceeds_the_maximum_allowed",
		},
		{
			name:     "long title truncate at word boundary within 10 chars",
			input:    "this is a very long title that should be truncated at a word boundary here",
			expected: "this_is_a_very_long_title_that_should_be_truncated",
		},
		{
			name:     "under 50 chars",
			input:    "short title",
			expected: "short_title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
			if len(result) > MaxSlugLength {
				t.Errorf("Slugify(%q) length = %d, want <= %d", tt.input, len(result), MaxSlugLength)
			}
		})
	}
}

// TestSlugify_EmptyResult tests that empty results become "task".
func TestSlugify_EmptyResult(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "only spaces",
			input: "     ",
		},
		{
			name:  "only special chars",
			input: "@#$%^&*()",
		},
		{
			name:  "only underscores",
			input: "___",
		},
		{
			name:  "only non-ASCII",
			input: "日本語",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != "task" {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, "task")
			}
		})
	}
}

// TestSlugify_NonASCII tests handling of non-ASCII characters.
func TestSlugify_NonASCII(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "accented characters",
			input:    "café résumé",
			expected: "cafe_resume",
		},
		{
			name:     "german umlaut",
			input:    "über",
			expected: "uber",
		},
		{
			name:     "spanish tilde",
			input:    "señor",
			expected: "senor",
		},
		{
			name:     "mixed ASCII and non-ASCII",
			input:    "hello wörld",
			expected: "hello_world",
		},
		{
			name:     "emoji removed",
			input:    "task 🚀 launch",
			expected: "task_launch",
		},
		{
			name:     "chinese characters removed",
			input:    "task 任务",
			expected: "task",
		},
		{
			name:     "cyrillic removed",
			input:    "task задача",
			expected: "task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSlugify_DesignExamples tests the exact examples from the design document.
// Note: The design document shows 49 chars for the long title example, but
// the requirement says "maximum 50 characters", so we truncate to 50.
func TestSlugify_DesignExamples(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Implement the Parser",
			input:    "Implement the Parser",
			expected: "implement_the_parser",
		},
		{
			name:     "Add UTF-8 Support!!!",
			input:    "Add UTF-8 Support!!!",
			expected: "add_utf-8_support",
		},
		{
			name:     "Trim Spaces",
			input:    "   Trim   Spaces   ",
			expected: "trim_spaces",
		},
		{
			name:     "UPPERCASE lowercase",
			input:    "UPPERCASE lowercase",
			expected: "uppercase_lowercase",
		},
		{
			name:     "Special chars",
			input:    "Special @#$% chars",
			expected: "special_chars",
		},
		{
			name:     "empty",
			input:    "",
			expected: "task",
		},
		{
			name:     "very long title",
			input:    "A very long title that exceeds the maximum allowed length for filenames",
			expected: "a_very_long_title_that_exceeds_the_maximum_allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			if result != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSlugify_Deterministic tests that slugification is deterministic.
func TestSlugify_Deterministic(t *testing.T) {
	inputs := []string{
		"Hello World",
		"Test Task 123",
		"Special @#$% chars",
		"   Spaces   ",
		"UPPERCASE",
		"café résumé",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			result1 := Slugify(input)
			result2 := Slugify(input)
			if result1 != result2 {
				t.Errorf("Slugify(%q) is not deterministic: %q != %q", input, result1, result2)
			}
		})
	}
}

// TestSlugify_MaxLengthProperty tests that output never exceeds MaxSlugLength.
func TestSlugify_MaxLengthProperty(t *testing.T) {
	// Test with various long inputs
	longInputs := []string{
		strings.Repeat("a", 100),
		strings.Repeat("ab ", 50),
		strings.Repeat("test ", 20),
		"A very long title that exceeds the maximum allowed length for filenames and should be truncated properly",
	}

	for _, input := range longInputs {
		t.Run("long_input", func(t *testing.T) {
			result := Slugify(input)
			if len(result) > MaxSlugLength {
				t.Errorf("Slugify(%q...) length = %d, want <= %d", input[:20], len(result), MaxSlugLength)
			}
		})
	}
}

// TestSlugify_ValidCharactersProperty tests that output only contains valid characters.
func TestSlugify_ValidCharactersProperty(t *testing.T) {
	inputs := []string{
		"Hello World!",
		"Test@Task#123",
		"Special $%^& chars",
		"café résumé naïve",
		"Mixed 123 Numbers",
		"hyphen-test_underscore",
	}

	validChars := "abcdefghijklmnopqrstuvwxyz0123456789_-"

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			result := Slugify(input)
			for _, ch := range result {
				if !strings.ContainsRune(validChars, ch) {
					t.Errorf("Slugify(%q) = %q contains invalid character %q", input, result, string(ch))
				}
			}
		})
	}
}

// TestGenerateFilename tests the GenerateFilename function.
func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name     string
		task     *Task
		expected string
	}{
		{
			name: "simple task",
			task: &Task{
				ID:    "1.1",
				Title: "Implement the Parser",
			},
			expected: "1.1.implement_the_parser.md",
		},
		{
			name: "task with special chars",
			task: &Task{
				ID:    "1.2",
				Title: "Add UTF-8 Support!!!",
			},
			expected: "1.2.add_utf-8_support.md",
		},
		{
			name: "task with spaces",
			task: &Task{
				ID:    "1.3",
				Title: "   Trim   Spaces   ",
			},
			expected: "1.3.trim_spaces.md",
		},
		{
			name: "task with uppercase",
			task: &Task{
				ID:    "1.4",
				Title: "UPPERCASE lowercase",
			},
			expected: "1.4.uppercase_lowercase.md",
		},
		{
			name: "task with special chars only",
			task: &Task{
				ID:    "1.5",
				Title: "Special @#$% chars",
			},
			expected: "1.5.special_chars.md",
		},
		{
			name: "task with empty title",
			task: &Task{
				ID:    "1.6",
				Title: "",
			},
			expected: "1.6.task.md",
		},
		{
			name: "task with long title",
			task: &Task{
				ID:    "1.7",
				Title: "A very long title that exceeds the maximum allowed length for filenames",
			},
			expected: "1.7.a_very_long_title_that_exceeds_the_maximum_allowed.md",
		},
		{
			name: "deep nested task",
			task: &Task{
				ID:    "1.2.3.4",
				Title: "Deep Task",
			},
			expected: "1.2.3.4.deep_task.md",
		},
		{
			name: "root task",
			task: &Task{
				ID:    "1",
				Title: "Root Task",
			},
			expected: "1.root_task.md",
		},
		{
			name:     "nil task",
			task:     nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateFilename(tt.task)
			if result != tt.expected {
				t.Errorf("GenerateFilename(%v) = %q, want %q", tt.task, result, tt.expected)
			}
		})
	}
}

// TestGenerateFilename_DesignExamples tests the exact examples from the design document.
// Note: The design document shows 49 chars for the long title example, but
// the requirement says "maximum 50 characters", so we truncate to 50.
func TestGenerateFilename_DesignExamples(t *testing.T) {
	tests := []struct {
		id       string
		title    string
		expected string
	}{
		{"1.1", "Implement the Parser", "1.1.implement_the_parser.md"},
		{"1.2", "Add UTF-8 Support!!!", "1.2.add_utf-8_support.md"},
		{"1.3", "   Trim   Spaces   ", "1.3.trim_spaces.md"},
		{"1.4", "UPPERCASE lowercase", "1.4.uppercase_lowercase.md"},
		{"1.5", "Special @#$% chars", "1.5.special_chars.md"},
		{"1.6", "", "1.6.task.md"},
		{"1.7", "A very long title that exceeds the maximum allowed length for filenames", "1.7.a_very_long_title_that_exceeds_the_maximum_allowed.md"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			task := &Task{
				ID:    tt.id,
				Title: tt.title,
			}
			result := GenerateFilename(task)
			if result != tt.expected {
				t.Errorf("GenerateFilename(id=%q, title=%q) = %q, want %q", tt.id, tt.title, result, tt.expected)
			}
		})
	}
}

// TestGenerateFilename_Format tests that the filename format is correct.
func TestGenerateFilename_Format(t *testing.T) {
	task := &Task{
		ID:    "2.3.4",
		Title: "Test Task",
	}

	result := GenerateFilename(task)

	// Should start with the ID
	if !strings.HasPrefix(result, task.ID+".") {
		t.Errorf("GenerateFilename should start with ID: got %q", result)
	}

	// Should end with .md
	if !strings.HasSuffix(result, ".md") {
		t.Errorf("GenerateFilename should end with .md: got %q", result)
	}

	// Should have format {id}.{slug}.md
	parts := strings.Split(result, ".")
	if len(parts) < 3 {
		t.Errorf("GenerateFilename should have format {id}.{slug}.md: got %q", result)
	}
}

// TestGenerateFilename_Deterministic tests that filename generation is deterministic.
func TestGenerateFilename_Deterministic(t *testing.T) {
	task := &Task{
		ID:    "1.1",
		Title: "Test Task",
	}

	result1 := GenerateFilename(task)
	result2 := GenerateFilename(task)

	if result1 != result2 {
		t.Errorf("GenerateFilename is not deterministic: %q != %q", result1, result2)
	}
}

// TestTruncateAtWordBoundary tests the truncation helper function.
func TestTruncateAtWordBoundary(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "no truncation needed",
			input:    "hello_world",
			maxLen:   50,
			expected: "hello_world",
		},
		{
			name:     "truncate at underscore within 10 chars",
			input:    "hello_world_test_more",
			maxLen:   15,
			expected: "hello_world",
		},
		{
			name:     "truncate no underscore",
			input:    "helloworld",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "truncate at exact boundary",
			input:    "hello_world",
			maxLen:   11,
			expected: "hello_world",
		},
		{
			name:     "underscore within 10 chars - truncate at word boundary",
			input:    "hello_worldtestmorechars",
			maxLen:   12,
			expected: "hello",
		},
		{
			name:     "no truncation when length equals maxLen",
			input:    "hello_world_",
			maxLen:   12,
			expected: "hello_world_",
		},
		{
			name:     "truncate and trim trailing underscore then word boundary",
			input:    "hello_world_test",
			maxLen:   12,
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateAtWordBoundary(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateAtWordBoundary(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestTransliterateToASCII tests the transliteration helper function.
func TestTransliterateToASCII(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain ASCII",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "accented e",
			input:    "café",
			expected: "cafe",
		},
		{
			name:     "accented a",
			input:    "naïve",
			expected: "naive",
		},
		{
			name:     "german umlaut",
			input:    "über",
			expected: "uber",
		},
		{
			name:     "spanish tilde",
			input:    "señor",
			expected: "senor",
		},
		{
			name:     "multiple accents",
			input:    "résumé",
			expected: "resume",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transliterateToASCII(tt.input)
			if result != tt.expected {
				t.Errorf("transliterateToASCII(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}


// TestResolveCollision_NoCollision tests that filenames without collisions are returned as-is.
func TestResolveCollision_NoCollision(t *testing.T) {
	tests := []struct {
		name              string
		filename          string
		existingFilenames map[string]bool
		expected          string
	}{
		{
			name:              "empty existing set",
			filename:          "1.1.task.md",
			existingFilenames: map[string]bool{},
			expected:          "1.1.task.md",
		},
		{
			name:     "no collision with other files",
			filename: "1.1.task.md",
			existingFilenames: map[string]bool{
				"1.2.other_task.md": true,
				"2.1.another.md":    true,
			},
			expected: "1.1.task.md",
		},
		{
			name:              "nil existing set treated as empty",
			filename:          "1.1.task.md",
			existingFilenames: nil,
			expected:          "1.1.task.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveCollision(tt.filename, tt.existingFilenames)
			if result != tt.expected {
				t.Errorf("ResolveCollision(%q, %v) = %q, want %q", tt.filename, tt.existingFilenames, result, tt.expected)
			}
		})
	}
}

// TestResolveCollision_SingleCollision tests that first collision gets -2 suffix.
func TestResolveCollision_SingleCollision(t *testing.T) {
	tests := []struct {
		name              string
		filename          string
		existingFilenames map[string]bool
		expected          string
	}{
		{
			name:     "first collision gets -2",
			filename: "1.1.implement_parser.md",
			existingFilenames: map[string]bool{
				"1.1.implement_parser.md": true,
			},
			expected: "1.1.implement_parser-2.md",
		},
		{
			name:     "collision with simple filename",
			filename: "1.task.md",
			existingFilenames: map[string]bool{
				"1.task.md": true,
			},
			expected: "1.task-2.md",
		},
		{
			name:     "collision with deep nested ID",
			filename: "1.2.3.4.deep_task.md",
			existingFilenames: map[string]bool{
				"1.2.3.4.deep_task.md": true,
			},
			expected: "1.2.3.4.deep_task-2.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveCollision(tt.filename, tt.existingFilenames)
			if result != tt.expected {
				t.Errorf("ResolveCollision(%q, %v) = %q, want %q", tt.filename, tt.existingFilenames, result, tt.expected)
			}
		})
	}
}

// TestResolveCollision_MultipleCollisions tests that subsequent collisions get -3, -4, etc.
func TestResolveCollision_MultipleCollisions(t *testing.T) {
	tests := []struct {
		name              string
		filename          string
		existingFilenames map[string]bool
		expected          string
	}{
		{
			name:     "second collision gets -3",
			filename: "1.1.implement_parser.md",
			existingFilenames: map[string]bool{
				"1.1.implement_parser.md":   true,
				"1.1.implement_parser-2.md": true,
			},
			expected: "1.1.implement_parser-3.md",
		},
		{
			name:     "third collision gets -4",
			filename: "1.1.implement_parser.md",
			existingFilenames: map[string]bool{
				"1.1.implement_parser.md":   true,
				"1.1.implement_parser-2.md": true,
				"1.1.implement_parser-3.md": true,
			},
			expected: "1.1.implement_parser-4.md",
		},
		{
			name:     "many collisions",
			filename: "1.1.task.md",
			existingFilenames: map[string]bool{
				"1.1.task.md":    true,
				"1.1.task-2.md":  true,
				"1.1.task-3.md":  true,
				"1.1.task-4.md":  true,
				"1.1.task-5.md":  true,
				"1.1.task-6.md":  true,
				"1.1.task-7.md":  true,
				"1.1.task-8.md":  true,
				"1.1.task-9.md":  true,
				"1.1.task-10.md": true,
			},
			expected: "1.1.task-11.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveCollision(tt.filename, tt.existingFilenames)
			if result != tt.expected {
				t.Errorf("ResolveCollision(%q, %v) = %q, want %q", tt.filename, tt.existingFilenames, result, tt.expected)
			}
		})
	}
}

// TestResolveCollision_GapsInSuffixes tests that gaps in suffix numbers are filled.
func TestResolveCollision_GapsInSuffixes(t *testing.T) {
	tests := []struct {
		name              string
		filename          string
		existingFilenames map[string]bool
		expected          string
	}{
		{
			name:     "gap at -2 is filled",
			filename: "1.1.task.md",
			existingFilenames: map[string]bool{
				"1.1.task.md":   true,
				"1.1.task-3.md": true, // -2 is missing
			},
			expected: "1.1.task-2.md",
		},
		{
			name:     "gap at -3 is filled",
			filename: "1.1.task.md",
			existingFilenames: map[string]bool{
				"1.1.task.md":   true,
				"1.1.task-2.md": true,
				"1.1.task-4.md": true, // -3 is missing
			},
			expected: "1.1.task-3.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveCollision(tt.filename, tt.existingFilenames)
			if result != tt.expected {
				t.Errorf("ResolveCollision(%q, %v) = %q, want %q", tt.filename, tt.existingFilenames, result, tt.expected)
			}
		})
	}
}

// TestResolveCollision_DesignExamples tests the exact examples from the design document.
func TestResolveCollision_DesignExamples(t *testing.T) {
	// Design document states:
	// - First occurrence: `1.1.implement_parser.md`
	// - Second occurrence: `1.1.implement_parser-2.md`
	// - Third occurrence: `1.1.implement_parser-3.md`

	// First occurrence - no collision
	existing1 := map[string]bool{}
	result1 := ResolveCollision("1.1.implement_parser.md", existing1)
	if result1 != "1.1.implement_parser.md" {
		t.Errorf("First occurrence: got %q, want %q", result1, "1.1.implement_parser.md")
	}

	// Second occurrence - collision with first
	existing2 := map[string]bool{
		"1.1.implement_parser.md": true,
	}
	result2 := ResolveCollision("1.1.implement_parser.md", existing2)
	if result2 != "1.1.implement_parser-2.md" {
		t.Errorf("Second occurrence: got %q, want %q", result2, "1.1.implement_parser-2.md")
	}

	// Third occurrence - collision with first and second
	existing3 := map[string]bool{
		"1.1.implement_parser.md":   true,
		"1.1.implement_parser-2.md": true,
	}
	result3 := ResolveCollision("1.1.implement_parser.md", existing3)
	if result3 != "1.1.implement_parser-3.md" {
		t.Errorf("Third occurrence: got %q, want %q", result3, "1.1.implement_parser-3.md")
	}
}

// TestResolveCollision_EdgeCases tests edge cases in collision resolution.
func TestResolveCollision_EdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		filename          string
		existingFilenames map[string]bool
		expected          string
	}{
		{
			name:     "filename without .md extension",
			filename: "1.1.task",
			existingFilenames: map[string]bool{
				"1.1.task": true,
			},
			expected: "1.1.task-2",
		},
		{
			name:     "filename with hyphen already in name",
			filename: "1.1.my-task.md",
			existingFilenames: map[string]bool{
				"1.1.my-task.md": true,
			},
			expected: "1.1.my-task-2.md",
		},
		{
			name:     "filename with underscore",
			filename: "1.1.my_task.md",
			existingFilenames: map[string]bool{
				"1.1.my_task.md": true,
			},
			expected: "1.1.my_task-2.md",
		},
		{
			name:     "filename with numbers in slug",
			filename: "1.1.task123.md",
			existingFilenames: map[string]bool{
				"1.1.task123.md": true,
			},
			expected: "1.1.task123-2.md",
		},
		{
			name:     "very short filename",
			filename: "1.a.md",
			existingFilenames: map[string]bool{
				"1.a.md": true,
			},
			expected: "1.a-2.md",
		},
		{
			name:     "filename that looks like it has suffix but doesn't",
			filename: "1.1.task-2.md",
			existingFilenames: map[string]bool{
				"1.1.task-2.md": true,
			},
			expected: "1.1.task-2-2.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveCollision(tt.filename, tt.existingFilenames)
			if result != tt.expected {
				t.Errorf("ResolveCollision(%q, %v) = %q, want %q", tt.filename, tt.existingFilenames, result, tt.expected)
			}
		})
	}
}

// TestResolveCollision_Deterministic tests that collision resolution is deterministic.
func TestResolveCollision_Deterministic(t *testing.T) {
	filename := "1.1.task.md"
	existing := map[string]bool{
		"1.1.task.md":   true,
		"1.1.task-2.md": true,
	}

	result1 := ResolveCollision(filename, existing)
	result2 := ResolveCollision(filename, existing)

	if result1 != result2 {
		t.Errorf("ResolveCollision is not deterministic: %q != %q", result1, result2)
	}
}

// TestResolveCollision_ResultIsUnique tests that the result is always unique.
func TestResolveCollision_ResultIsUnique(t *testing.T) {
	filename := "1.1.task.md"
	existing := map[string]bool{
		"1.1.task.md":   true,
		"1.1.task-2.md": true,
		"1.1.task-3.md": true,
	}

	result := ResolveCollision(filename, existing)

	if existing[result] {
		t.Errorf("ResolveCollision returned existing filename: %q", result)
	}
}

// TestSplitFilenameExtension tests the helper function for splitting filename and extension.
func TestSplitFilenameExtension(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		expectedBase string
		expectedExt  string
	}{
		{
			name:         "standard .md file",
			filename:     "1.1.task.md",
			expectedBase: "1.1.task",
			expectedExt:  ".md",
		},
		{
			name:         "no extension",
			filename:     "1.1.task",
			expectedBase: "1.1.task",
			expectedExt:  "",
		},
		{
			name:         "multiple dots",
			filename:     "1.2.3.task.md",
			expectedBase: "1.2.3.task",
			expectedExt:  ".md",
		},
		{
			name:         "just .md",
			filename:     ".md",
			expectedBase: "",
			expectedExt:  ".md",
		},
		{
			name:         "empty string",
			filename:     "",
			expectedBase: "",
			expectedExt:  "",
		},
		{
			name:         "other extension",
			filename:     "file.txt",
			expectedBase: "file.txt",
			expectedExt:  "",
		},
		// Test cases for numeric directory names (PR review fix)
		// Directory names like "1.md" should NOT be treated as having .md extension
		{
			name:         "numeric directory name 1.md",
			filename:     "1.md",
			expectedBase: "1.md",
			expectedExt:  "",
		},
		{
			name:         "numeric directory name 123.md",
			filename:     "123.md",
			expectedBase: "123.md",
			expectedExt:  "",
		},
		{
			name:         "numeric directory name 42.md",
			filename:     "42.md",
			expectedBase: "42.md",
			expectedExt:  "",
		},
		// Non-numeric bases should still be treated as having .md extension
		{
			name:         "alphanumeric base task1.md",
			filename:     "task1.md",
			expectedBase: "task1",
			expectedExt:  ".md",
		},
		{
			name:         "alphanumeric base 1task.md",
			filename:     "1task.md",
			expectedBase: "1task",
			expectedExt:  ".md",
		},
		{
			name:         "dotted numeric ID 1.2.task.md",
			filename:     "1.2.task.md",
			expectedBase: "1.2.task",
			expectedExt:  ".md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, ext := splitFilenameExtension(tt.filename)
			if base != tt.expectedBase {
				t.Errorf("splitFilenameExtension(%q) base = %q, want %q", tt.filename, base, tt.expectedBase)
			}
			if ext != tt.expectedExt {
				t.Errorf("splitFilenameExtension(%q) ext = %q, want %q", tt.filename, ext, tt.expectedExt)
			}
		})
	}
}

// TestItoa tests the integer to string conversion helper.
func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{2, "2"},
		{9, "9"},
		{10, "10"},
		{11, "11"},
		{99, "99"},
		{100, "100"},
		{123, "123"},
		{1000, "1000"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := itoa(tt.input)
			if result != tt.expected {
				t.Errorf("itoa(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsAllDigits tests the helper function for checking if a string is all digits.
func TestIsAllDigits(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "single digit",
			input:    "1",
			expected: true,
		},
		{
			name:     "multiple digits",
			input:    "123",
			expected: true,
		},
		{
			name:     "zero",
			input:    "0",
			expected: true,
		},
		{
			name:     "large number",
			input:    "9876543210",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "letters only",
			input:    "abc",
			expected: false,
		},
		{
			name:     "mixed alphanumeric",
			input:    "1a2b",
			expected: false,
		},
		{
			name:     "digit then letter",
			input:    "1a",
			expected: false,
		},
		{
			name:     "letter then digit",
			input:    "a1",
			expected: false,
		},
		{
			name:     "with dot",
			input:    "1.2",
			expected: false,
		},
		{
			name:     "with underscore",
			input:    "1_2",
			expected: false,
		},
		{
			name:     "with hyphen",
			input:    "1-2",
			expected: false,
		},
		{
			name:     "with space",
			input:    "1 2",
			expected: false,
		},
		{
			name:     "unicode digits",
			input:    "١٢٣", // Arabic-Indic digits
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAllDigits(tt.input)
			if result != tt.expected {
				t.Errorf("isAllDigits(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
