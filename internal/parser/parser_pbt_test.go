// Package parser provides property-based tests for the Tasks_MD_Parser component.
//
// Feature: fuse-task-filesystem
// Property 2: Checkbox Status Mapping
//
// These tests use the rapid library for property-based testing to verify that
// checkbox syntax is correctly mapped to task statuses across all valid inputs.
package parser

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"task-fuse/internal/printer"
	"task-fuse/internal/store"
)

// =============================================================================
// Property 2: Checkbox Status Mapping
// =============================================================================
//
// Property Statement:
// *For any* task line with a valid checkbox syntax (`- [ ]`, `- [~]`, `- [-]`,
// `- [x]`, `- [!]`), the Tasks_MD_Parser SHALL map it to the corresponding
// status (pending, queued, doing, done, failed respectively).
//
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5**
//
// Tag: Feature: fuse-task-filesystem, Property 2: Checkbox Status Mapping

// checkboxMapping defines the mapping between checkbox characters and their
// expected TaskStatus values as specified in the requirements.
var checkboxMapping = map[string]store.TaskStatus{
	" ": store.StatusPending, // Requirement 1.1: - [ ] → pending
	"~": store.StatusQueued,  // Requirement 1.2: - [~] → queued
	"-": store.StatusDoing,   // Requirement 1.3: - [-] → doing
	"x": store.StatusDone,    // Requirement 1.4: - [x] → done
	"!": store.StatusFailed,  // Requirement 1.5: - [!] → failed
}

// validCheckboxChars contains all valid checkbox characters.
var validCheckboxChars = []string{" ", "~", "-", "x", "!"}

// genCheckboxChar generates a random valid checkbox character.
func genCheckboxChar() *rapid.Generator[string] {
	return rapid.SampledFrom(validCheckboxChars)
}

// genIndentation generates random leading whitespace (spaces and/or tabs).
// This tests that checkbox parsing works with various indentation levels.
func genIndentation() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate 0-10 indent units
		indentUnits := rapid.IntRange(0, 10).Draw(t, "indentUnits")
		var indent strings.Builder
		for i := 0; i < indentUnits; i++ {
			// Each unit is either 2 spaces or 1 tab
			if rapid.Bool().Draw(t, fmt.Sprintf("useTab_%d", i)) {
				indent.WriteString("\t")
			} else {
				indent.WriteString("  ")
			}
		}
		return indent.String()
	})
}

// genTaskID generates a valid task ID (e.g., "1", "1.2", "1.2.3").
func genTaskID() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate 1-5 levels deep
		depth := rapid.IntRange(1, 5).Draw(t, "depth")
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			// Each segment is a positive integer (1-99)
			num := rapid.IntRange(1, 99).Draw(t, fmt.Sprintf("segment_%d", i))
			segments[i] = fmt.Sprintf("%d", num)
		}
		return strings.Join(segments, ".")
	})
}

// genTaskTitle generates a random task title.
func genTaskTitle() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate a title with 1-10 words
		wordCount := rapid.IntRange(1, 10).Draw(t, "wordCount")
		words := make([]string, wordCount)
		for i := 0; i < wordCount; i++ {
			// Each word is 1-15 alphanumeric characters
			wordLen := rapid.IntRange(1, 15).Draw(t, fmt.Sprintf("wordLen_%d", i))
			// Use StringMatching with a simple alphanumeric pattern
			word := rapid.StringMatching(`[a-zA-Z0-9]+`).Draw(t, fmt.Sprintf("word_%d", i))
			// Truncate to desired length if needed
			if len(word) > wordLen {
				word = word[:wordLen]
			}
			if len(word) == 0 {
				word = "task"
			}
			words[i] = word
		}
		return strings.Join(words, " ")
	})
}

// genValidTaskLine generates a complete valid task line with checkbox syntax.
// Format: {indent}- [{checkbox}] {taskID} {title}
func genValidTaskLine() *rapid.Generator[struct {
	Line           string
	CheckboxChar   string
	ExpectedStatus store.TaskStatus
}] {
	return rapid.Custom(func(t *rapid.T) struct {
		Line           string
		CheckboxChar   string
		ExpectedStatus store.TaskStatus
	} {
		indent := genIndentation().Draw(t, "indent")
		checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
		taskID := genTaskID().Draw(t, "taskID")
		title := genTaskTitle().Draw(t, "title")

		line := fmt.Sprintf("%s- [%s] %s %s", indent, checkboxChar, taskID, title)
		expectedStatus := checkboxMapping[checkboxChar]

		return struct {
			Line           string
			CheckboxChar   string
			ExpectedStatus store.TaskStatus
		}{
			Line:           line,
			CheckboxChar:   checkboxChar,
			ExpectedStatus: expectedStatus,
		}
	})
}

// TestProperty2_CheckboxStatusMapping is the main property-based test for
// Property 2: Checkbox Status Mapping.
//
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5**
//
// This test generates random task lines with valid checkbox syntax and verifies
// that ParseCheckbox correctly maps each checkbox character to its corresponding
// TaskStatus value.
//
// Tag: Feature: fuse-task-filesystem, Property 2: Checkbox Status Mapping
func TestProperty2_CheckboxStatusMapping(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random valid task line
		testCase := genValidTaskLine().Draw(t, "taskLine")

		// Parse the checkbox from the line
		gotStatus, gotOk := p.ParseCheckbox(testCase.Line)

		// Property assertion 1: ParseCheckbox should successfully parse valid checkbox syntax
		if !gotOk {
			t.Fatalf("ParseCheckbox(%q) returned ok=false, expected ok=true for valid checkbox syntax", testCase.Line)
		}

		// Property assertion 2: The parsed status should match the expected status
		// based on the checkbox character
		if gotStatus != testCase.ExpectedStatus {
			t.Fatalf("ParseCheckbox(%q) returned status=%q, expected status=%q for checkbox char '%s'",
				testCase.Line, gotStatus, testCase.ExpectedStatus, testCase.CheckboxChar)
		}
	})
}

// TestProperty2_CheckboxStatusMapping_AllCheckboxTypes tests that each specific
// checkbox type maps to its correct status across many random variations.
//
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5**
//
// This test runs separate property checks for each checkbox type to ensure
// comprehensive coverage of all status mappings.
//
// Tag: Feature: fuse-task-filesystem, Property 2: Checkbox Status Mapping
func TestProperty2_CheckboxStatusMapping_AllCheckboxTypes(t *testing.T) {
	p := NewParser()

	// Test each checkbox type separately
	for checkboxChar, expectedStatus := range checkboxMapping {
		checkboxChar := checkboxChar // capture for closure
		expectedStatus := expectedStatus

		t.Run(fmt.Sprintf("checkbox_%s_maps_to_%s", checkboxChar, expectedStatus), func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				// Generate random indentation, task ID, and title
				indent := genIndentation().Draw(t, "indent")
				taskID := genTaskID().Draw(t, "taskID")
				title := genTaskTitle().Draw(t, "title")

				// Construct the task line with the specific checkbox character
				line := fmt.Sprintf("%s- [%s] %s %s", indent, checkboxChar, taskID, title)

				// Parse the checkbox
				gotStatus, gotOk := p.ParseCheckbox(line)

				// Verify parsing succeeded
				if !gotOk {
					t.Fatalf("ParseCheckbox(%q) returned ok=false for valid checkbox '[%s]'", line, checkboxChar)
				}

				// Verify correct status mapping
				if gotStatus != expectedStatus {
					t.Fatalf("ParseCheckbox(%q) returned status=%q, expected %q for checkbox '[%s]'",
						line, gotStatus, expectedStatus, checkboxChar)
				}
			})
		})
	}
}

// TestProperty2_CheckboxStatusMapping_WithVariousContent tests checkbox parsing
// with various content after the checkbox (including edge cases).
//
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5**
//
// Tag: Feature: fuse-task-filesystem, Property 2: Checkbox Status Mapping
func TestProperty2_CheckboxStatusMapping_WithVariousContent(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate checkbox character
		checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
		expectedStatus := checkboxMapping[checkboxChar]

		// Generate various content types after the checkbox
		contentType := rapid.IntRange(0, 4).Draw(t, "contentType")
		var content string
		switch contentType {
		case 0:
			// Just a task ID
			content = genTaskID().Draw(t, "taskID") + " " + genTaskTitle().Draw(t, "title")
		case 1:
			// Task ID with special characters in title
			taskID := genTaskID().Draw(t, "taskID")
			content = fmt.Sprintf("%s Task with @#$%% special chars", taskID)
		case 2:
			// Task ID with numbers in title
			taskID := genTaskID().Draw(t, "taskID")
			content = fmt.Sprintf("%s Task 123 with numbers", taskID)
		case 3:
			// Minimal content (just task ID and short title)
			taskID := genTaskID().Draw(t, "taskID")
			content = fmt.Sprintf("%s T", taskID)
		case 4:
			// No task ID (just title) - checkbox should still parse
			content = genTaskTitle().Draw(t, "title")
		}

		// Generate indentation
		indent := genIndentation().Draw(t, "indent")

		// Construct the line
		line := fmt.Sprintf("%s- [%s] %s", indent, checkboxChar, content)

		// Parse the checkbox
		gotStatus, gotOk := p.ParseCheckbox(line)

		// Verify parsing succeeded (checkbox syntax is valid regardless of content)
		if !gotOk {
			t.Fatalf("ParseCheckbox(%q) returned ok=false for valid checkbox '[%s]'", line, checkboxChar)
		}

		// Verify correct status mapping
		if gotStatus != expectedStatus {
			t.Fatalf("ParseCheckbox(%q) returned status=%q, expected %q for checkbox '[%s]'",
				line, gotStatus, expectedStatus, checkboxChar)
		}
	})
}

// TestProperty2_CheckboxStatusMapping_MinimalLines tests checkbox parsing with
// minimal valid lines (checkbox only, no content after).
//
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5**
//
// Tag: Feature: fuse-task-filesystem, Property 2: Checkbox Status Mapping
func TestProperty2_CheckboxStatusMapping_MinimalLines(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate checkbox character
		checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
		expectedStatus := checkboxMapping[checkboxChar]

		// Generate indentation
		indent := genIndentation().Draw(t, "indent")

		// Construct minimal line (just checkbox, no content)
		line := fmt.Sprintf("%s- [%s]", indent, checkboxChar)

		// Parse the checkbox
		gotStatus, gotOk := p.ParseCheckbox(line)

		// Verify parsing succeeded
		if !gotOk {
			t.Fatalf("ParseCheckbox(%q) returned ok=false for valid minimal checkbox '[%s]'", line, checkboxChar)
		}

		// Verify correct status mapping
		if gotStatus != expectedStatus {
			t.Fatalf("ParseCheckbox(%q) returned status=%q, expected %q for checkbox '[%s]'",
				line, gotStatus, expectedStatus, checkboxChar)
		}
	})
}

// TestProperty2_CheckboxStatusMapping_Deterministic verifies that parsing the
// same line always produces the same result (determinism property).
//
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5**
//
// Tag: Feature: fuse-task-filesystem, Property 2: Checkbox Status Mapping
func TestProperty2_CheckboxStatusMapping_Deterministic(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random valid task line
		testCase := genValidTaskLine().Draw(t, "taskLine")

		// Parse the same line multiple times
		status1, ok1 := p.ParseCheckbox(testCase.Line)
		status2, ok2 := p.ParseCheckbox(testCase.Line)
		status3, ok3 := p.ParseCheckbox(testCase.Line)

		// All results should be identical
		if ok1 != ok2 || ok2 != ok3 {
			t.Fatalf("ParseCheckbox(%q) returned inconsistent ok values: %v, %v, %v",
				testCase.Line, ok1, ok2, ok3)
		}

		if status1 != status2 || status2 != status3 {
			t.Fatalf("ParseCheckbox(%q) returned inconsistent status values: %q, %q, %q",
				testCase.Line, status1, status2, status3)
		}
	})
}

// TestProperty2_CheckboxStatusMapping_BijectiveMapping verifies that the mapping
// between checkbox characters and statuses is bijective (one-to-one and onto).
//
// **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5**
//
// Tag: Feature: fuse-task-filesystem, Property 2: Checkbox Status Mapping
func TestProperty2_CheckboxStatusMapping_BijectiveMapping(t *testing.T) {
	p := NewParser()

	// Verify that each checkbox character maps to a unique status
	statusToCheckbox := make(map[store.TaskStatus]string)
	for checkboxChar, status := range checkboxMapping {
		if existingChar, exists := statusToCheckbox[status]; exists {
			t.Fatalf("Status %q is mapped from multiple checkbox chars: '%s' and '%s'",
				status, existingChar, checkboxChar)
		}
		statusToCheckbox[status] = checkboxChar
	}

	// Verify all 5 statuses are covered
	expectedStatuses := []store.TaskStatus{
		store.StatusPending,
		store.StatusQueued,
		store.StatusDoing,
		store.StatusDone,
		store.StatusFailed,
	}

	for _, status := range expectedStatuses {
		if _, exists := statusToCheckbox[status]; !exists {
			t.Fatalf("Status %q is not mapped from any checkbox character", status)
		}
	}

	// Property test: verify the mapping holds for random lines
	rapid.Check(t, func(t *rapid.T) {
		checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
		expectedStatus := checkboxMapping[checkboxChar]

		// Create a simple line with this checkbox
		line := fmt.Sprintf("- [%s] 1 Test", checkboxChar)

		gotStatus, gotOk := p.ParseCheckbox(line)
		if !gotOk {
			t.Fatalf("ParseCheckbox(%q) failed for valid checkbox", line)
		}

		// Verify the mapping is correct
		if gotStatus != expectedStatus {
			t.Fatalf("Checkbox '%s' should map to %q but got %q", checkboxChar, expectedStatus, gotStatus)
		}

		// Verify no other checkbox maps to this status (bijective)
		for otherChar, otherStatus := range checkboxMapping {
			if otherChar != checkboxChar && otherStatus == gotStatus {
				t.Fatalf("Multiple checkboxes map to same status: '%s' and '%s' both map to %q",
					checkboxChar, otherChar, gotStatus)
			}
		}
	})
}


// =============================================================================
// Property 15: Invalid Task ID Handling
// =============================================================================
//
// Property Statement:
// *For any* line with checkbox syntax but an invalid task ID format (e.g.,
// leading zeros, non-numeric segments), the Tasks_MD_Parser SHALL treat it as
// non-task content and preserve it unchanged.
//
// **Validates: Requirements 1.11**
//
// Tag: Feature: fuse-task-filesystem, Property 15: Invalid Task ID Handling

// InvalidIDType represents different categories of invalid task IDs.
type InvalidIDType int

const (
	InvalidIDLeadingZeroFirst  InvalidIDType = iota // e.g., "01.1" - leading zero in first segment
	InvalidIDLeadingZeroLater                       // e.g., "1.01" - leading zero in later segment
	InvalidIDNonNumericSegment                      // e.g., "1.2.a" - non-numeric segment
	InvalidIDEmptySegment                           // e.g., "1..2" - empty segment (double dot)
	InvalidIDZeroSegment                            // e.g., "0" or "1.0" - zero is not a positive integer
	InvalidIDLeadingDot                             // e.g., ".1.2" - leading dot
	InvalidIDOnlyDots                               // e.g., "..." - only dots
	InvalidIDMixedAlphaNumeric                      // e.g., "1a.2" - mixed alphanumeric in segment
)

// Note: Trailing dot (e.g., "1.2.") is NOT included as an invalid type because
// the parser's ExtractTaskID regex intentionally allows an optional trailing dot
// and extracts the valid ID portion (e.g., "1.2." becomes "1.2").

// genInvalidTaskID generates an invalid task ID based on the specified type.
func genInvalidTaskID(idType InvalidIDType) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		switch idType {
		case InvalidIDLeadingZeroFirst:
			// Generate "0X.Y.Z" where X is 0-9, Y and Z are valid segments
			firstDigit := rapid.IntRange(0, 9).Draw(t, "firstDigit")
			depth := rapid.IntRange(1, 3).Draw(t, "depth")
			segments := []string{fmt.Sprintf("0%d", firstDigit)}
			for i := 1; i < depth; i++ {
				num := rapid.IntRange(1, 99).Draw(t, fmt.Sprintf("segment_%d", i))
				segments = append(segments, fmt.Sprintf("%d", num))
			}
			return strings.Join(segments, ".")

		case InvalidIDLeadingZeroLater:
			// Generate "X.0Y.Z" where X is valid, 0Y has leading zero
			firstNum := rapid.IntRange(1, 99).Draw(t, "firstNum")
			secondDigit := rapid.IntRange(0, 9).Draw(t, "secondDigit")
			depth := rapid.IntRange(2, 4).Draw(t, "depth")
			segments := []string{fmt.Sprintf("%d", firstNum), fmt.Sprintf("0%d", secondDigit)}
			for i := 2; i < depth; i++ {
				num := rapid.IntRange(1, 99).Draw(t, fmt.Sprintf("segment_%d", i))
				segments = append(segments, fmt.Sprintf("%d", num))
			}
			return strings.Join(segments, ".")

		case InvalidIDNonNumericSegment:
			// Generate "X.Y.abc" where X and Y are valid, last is non-numeric
			depth := rapid.IntRange(1, 3).Draw(t, "depth")
			segments := make([]string, depth)
			for i := 0; i < depth-1; i++ {
				num := rapid.IntRange(1, 99).Draw(t, fmt.Sprintf("segment_%d", i))
				segments[i] = fmt.Sprintf("%d", num)
			}
			// Last segment is non-numeric
			nonNumeric := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]*`).Draw(t, "nonNumeric")
			if len(nonNumeric) > 5 {
				nonNumeric = nonNumeric[:5]
			}
			if len(nonNumeric) == 0 {
				nonNumeric = "a"
			}
			segments[depth-1] = nonNumeric
			return strings.Join(segments, ".")

		case InvalidIDEmptySegment:
			// Generate "X..Y" with empty segment (double dot)
			firstNum := rapid.IntRange(1, 99).Draw(t, "firstNum")
			secondNum := rapid.IntRange(1, 99).Draw(t, "secondNum")
			// Insert double dot
			return fmt.Sprintf("%d..%d", firstNum, secondNum)

		case InvalidIDZeroSegment:
			// Generate "0" or "X.0" where a segment is exactly zero
			useZeroOnly := rapid.Bool().Draw(t, "useZeroOnly")
			if useZeroOnly {
				return "0"
			}
			firstNum := rapid.IntRange(1, 99).Draw(t, "firstNum")
			return fmt.Sprintf("%d.0", firstNum)

		case InvalidIDLeadingDot:
			// Generate ".X.Y" with leading dot
			depth := rapid.IntRange(1, 3).Draw(t, "depth")
			segments := make([]string, depth)
			for i := 0; i < depth; i++ {
				num := rapid.IntRange(1, 99).Draw(t, fmt.Sprintf("segment_%d", i))
				segments[i] = fmt.Sprintf("%d", num)
			}
			return "." + strings.Join(segments, ".")

		case InvalidIDOnlyDots:
			// Generate "..." or ".." - only dots
			dotCount := rapid.IntRange(1, 5).Draw(t, "dotCount")
			return strings.Repeat(".", dotCount)

		case InvalidIDMixedAlphaNumeric:
			// Generate "1a.2" where a segment has mixed alphanumeric
			firstPart := rapid.IntRange(1, 9).Draw(t, "firstPart")
			letter := rapid.StringMatching(`[a-zA-Z]`).Draw(t, "letter")
			if len(letter) == 0 {
				letter = "a"
			}
			secondNum := rapid.IntRange(1, 99).Draw(t, "secondNum")
			return fmt.Sprintf("%d%s.%d", firstPart, letter, secondNum)

		default:
			return "invalid"
		}
	})
}

// genAnyInvalidTaskID generates any type of invalid task ID.
func genAnyInvalidTaskID() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		idType := InvalidIDType(rapid.IntRange(0, int(InvalidIDMixedAlphaNumeric)).Draw(t, "idType"))
		return genInvalidTaskID(idType).Draw(t, "invalidID")
	})
}

// genInvalidTaskLine generates a task line with checkbox syntax but an invalid task ID.
// Format: {indent}- [{checkbox}] {invalidID} {title}
func genInvalidTaskLine() *rapid.Generator[struct {
	Line      string
	InvalidID string
}] {
	return rapid.Custom(func(t *rapid.T) struct {
		Line      string
		InvalidID string
	} {
		indent := genIndentation().Draw(t, "indent")
		checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
		invalidID := genAnyInvalidTaskID().Draw(t, "invalidID")
		title := genTaskTitle().Draw(t, "title")

		line := fmt.Sprintf("%s- [%s] %s %s", indent, checkboxChar, invalidID, title)

		return struct {
			Line      string
			InvalidID string
		}{
			Line:      line,
			InvalidID: invalidID,
		}
	})
}

// TestProperty15_InvalidTaskIDHandling_ValidateTaskID tests that ValidateTaskID
// correctly rejects invalid task ID formats.
//
// **Validates: Requirements 1.11**
//
// Tag: Feature: fuse-task-filesystem, Property 15: Invalid Task ID Handling
func TestProperty15_InvalidTaskIDHandling_ValidateTaskID(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate an invalid task ID
		invalidID := genAnyInvalidTaskID().Draw(t, "invalidID")

		// ValidateTaskID should return false for invalid IDs
		isValid := p.ValidateTaskID(invalidID)

		if isValid {
			t.Fatalf("ValidateTaskID(%q) returned true, expected false for invalid ID format", invalidID)
		}
	})
}

// TestProperty15_InvalidTaskIDHandling_ExtractTaskID tests that ExtractTaskID
// returns false for lines with invalid task ID formats.
//
// **Validates: Requirements 1.11**
//
// Tag: Feature: fuse-task-filesystem, Property 15: Invalid Task ID Handling
func TestProperty15_InvalidTaskIDHandling_ExtractTaskID(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate an invalid task line
		testCase := genInvalidTaskLine().Draw(t, "taskLine")

		// Extract the content after the checkbox
		checkboxEnd := strings.Index(testCase.Line, "] ")
		var afterCheckbox string
		if checkboxEnd != -1 && checkboxEnd+2 < len(testCase.Line) {
			afterCheckbox = testCase.Line[checkboxEnd+2:]
		} else {
			afterCheckbox = ""
		}

		// ExtractTaskID should return false for invalid IDs
		_, ok := p.ExtractTaskID(afterCheckbox)

		if ok {
			t.Fatalf("ExtractTaskID(%q) returned ok=true, expected ok=false for invalid ID format in line %q",
				afterCheckbox, testCase.Line)
		}
	})
}

// TestProperty15_InvalidTaskIDHandling_ParsePreservesAsNonTask tests that Parse
// treats lines with invalid task IDs as non-task content.
//
// **Validates: Requirements 1.11**
//
// Tag: Feature: fuse-task-filesystem, Property 15: Invalid Task ID Handling
func TestProperty15_InvalidTaskIDHandling_ParsePreservesAsNonTask(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid task line first (to have a task to attach description to)
		validIndent := genIndentation().Draw(t, "validIndent")
		validCheckbox := genCheckboxChar().Draw(t, "validCheckbox")
		validID := genTaskID().Draw(t, "validID")
		validTitle := genTaskTitle().Draw(t, "validTitle")
		validLine := fmt.Sprintf("%s- [%s] %s %s", validIndent, validCheckbox, validID, validTitle)

		// Generate an invalid task line (indented more than the valid task)
		invalidIndent := validIndent + "  " // More indented
		invalidCheckbox := genCheckboxChar().Draw(t, "invalidCheckbox")
		invalidID := genAnyInvalidTaskID().Draw(t, "invalidID")
		invalidTitle := genTaskTitle().Draw(t, "invalidTitle")
		invalidLine := fmt.Sprintf("%s- [%s] %s %s", invalidIndent, invalidCheckbox, invalidID, invalidTitle)

		// Create content with valid task followed by invalid task line
		content := validLine + "\n" + invalidLine

		// Parse the content
		result := p.Parse(content)

		// Should have exactly one task (the valid one)
		if len(result.Document.RootTasks) != 1 {
			t.Fatalf("Parse(%q) produced %d root tasks, expected 1 (invalid ID line should not be a task)",
				content, len(result.Document.RootTasks))
		}

		// The invalid line should be preserved in the task's description
		task := result.Document.RootTasks[0]
		if task.ID != validID {
			t.Fatalf("Parse(%q) produced task with ID %q, expected %q", content, task.ID, validID)
		}

		// Check that the invalid line is in the description
		foundInvalidLine := false
		for _, descLine := range task.Description {
			if descLine.RawContent == invalidLine {
				foundInvalidLine = true
				break
			}
		}

		if !foundInvalidLine {
			t.Fatalf("Parse(%q) did not preserve invalid ID line %q in task description. Description: %v",
				content, invalidLine, task.Description)
		}
	})
}

// TestProperty15_InvalidTaskIDHandling_AllInvalidTypes tests each specific type
// of invalid task ID to ensure comprehensive coverage.
//
// **Validates: Requirements 1.11**
//
// Tag: Feature: fuse-task-filesystem, Property 15: Invalid Task ID Handling
func TestProperty15_InvalidTaskIDHandling_AllInvalidTypes(t *testing.T) {
	p := NewParser()

	invalidTypes := []struct {
		name   string
		idType InvalidIDType
	}{
		{"leading_zero_first", InvalidIDLeadingZeroFirst},
		{"leading_zero_later", InvalidIDLeadingZeroLater},
		{"non_numeric_segment", InvalidIDNonNumericSegment},
		{"empty_segment", InvalidIDEmptySegment},
		{"zero_segment", InvalidIDZeroSegment},
		{"leading_dot", InvalidIDLeadingDot},
		{"only_dots", InvalidIDOnlyDots},
		{"mixed_alphanumeric", InvalidIDMixedAlphaNumeric},
	}

	for _, tc := range invalidTypes {
		tc := tc // capture for closure
		t.Run(tc.name, func(t *testing.T) {
			rapid.Check(t, func(rt *rapid.T) {
				// Generate an invalid ID of this specific type
				invalidID := genInvalidTaskID(tc.idType).Draw(rt, "invalidID")

				// ValidateTaskID should return false
				if p.ValidateTaskID(invalidID) {
					t.Fatalf("ValidateTaskID(%q) returned true for invalid ID type %s", invalidID, tc.name)
				}

				// Create a task line with this invalid ID
				indent := genIndentation().Draw(rt, "indent")
				checkboxChar := genCheckboxChar().Draw(rt, "checkboxChar")
				title := genTaskTitle().Draw(rt, "title")
				line := fmt.Sprintf("%s- [%s] %s %s", indent, checkboxChar, invalidID, title)

				// Extract content after checkbox
				checkboxEnd := strings.Index(line, "] ")
				var afterCheckbox string
				if checkboxEnd != -1 && checkboxEnd+2 < len(line) {
					afterCheckbox = line[checkboxEnd+2:]
				}

				// ExtractTaskID should return false
				_, ok := p.ExtractTaskID(afterCheckbox)
				if ok {
					t.Fatalf("ExtractTaskID(%q) returned ok=true for invalid ID type %s", afterCheckbox, tc.name)
				}
			})
		})
	}
}

// TestProperty15_InvalidTaskIDHandling_PreservedUnchanged tests that invalid task
// ID lines are preserved exactly as they appear in the original content.
//
// **Validates: Requirements 1.11**
//
// Tag: Feature: fuse-task-filesystem, Property 15: Invalid Task ID Handling
func TestProperty15_InvalidTaskIDHandling_PreservedUnchanged(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a preamble line
		preambleLine := "# Task List"

		// Generate a valid task
		validCheckbox := genCheckboxChar().Draw(t, "validCheckbox")
		validID := genTaskID().Draw(t, "validID")
		validTitle := genTaskTitle().Draw(t, "validTitle")
		validLine := fmt.Sprintf("- [%s] %s %s", validCheckbox, validID, validTitle)

		// Generate an invalid task line (as description)
		invalidCheckbox := genCheckboxChar().Draw(t, "invalidCheckbox")
		invalidID := genAnyInvalidTaskID().Draw(t, "invalidID")
		invalidTitle := genTaskTitle().Draw(t, "invalidTitle")
		invalidLine := fmt.Sprintf("  - [%s] %s %s", invalidCheckbox, invalidID, invalidTitle)

		// Create content
		content := preambleLine + "\n\n" + validLine + "\n" + invalidLine

		// Parse the content
		result := p.Parse(content)

		// Find the invalid line in the parsed result
		// It should be in the task's description, preserved exactly
		if len(result.Document.RootTasks) != 1 {
			t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
		}

		task := result.Document.RootTasks[0]
		foundExactMatch := false
		for _, descLine := range task.Description {
			if descLine.RawContent == invalidLine {
				foundExactMatch = true
				break
			}
		}

		if !foundExactMatch {
			t.Fatalf("Invalid ID line was not preserved exactly. Expected %q in description, got: %v",
				invalidLine, task.Description)
		}
	})
}

// TestProperty15_InvalidTaskIDHandling_InPreamble tests that invalid task ID lines
// appearing before any valid task are preserved in the preamble.
//
// **Validates: Requirements 1.11**
//
// Tag: Feature: fuse-task-filesystem, Property 15: Invalid Task ID Handling
func TestProperty15_InvalidTaskIDHandling_InPreamble(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate an invalid task line (will be in preamble since no valid task before it)
		invalidCheckbox := genCheckboxChar().Draw(t, "invalidCheckbox")
		invalidID := genAnyInvalidTaskID().Draw(t, "invalidID")
		invalidTitle := genTaskTitle().Draw(t, "invalidTitle")
		invalidLine := fmt.Sprintf("- [%s] %s %s", invalidCheckbox, invalidID, invalidTitle)

		// Generate a valid task line after
		validCheckbox := genCheckboxChar().Draw(t, "validCheckbox")
		validID := genTaskID().Draw(t, "validID")
		validTitle := genTaskTitle().Draw(t, "validTitle")
		validLine := fmt.Sprintf("- [%s] %s %s", validCheckbox, validID, validTitle)

		// Create content with invalid line first
		content := invalidLine + "\n" + validLine

		// Parse the content
		result := p.Parse(content)

		// The invalid line should be in the preamble
		foundInPreamble := false
		for _, preambleLine := range result.Document.Preamble {
			if preambleLine.RawContent == invalidLine {
				foundInPreamble = true
				break
			}
		}

		if !foundInPreamble {
			t.Fatalf("Invalid ID line %q was not found in preamble. Preamble: %v",
				invalidLine, result.Document.Preamble)
		}

		// Should still have the valid task
		if len(result.Document.RootTasks) != 1 {
			t.Fatalf("Expected 1 root task, got %d", len(result.Document.RootTasks))
		}

		if result.Document.RootTasks[0].ID != validID {
			t.Fatalf("Expected task ID %q, got %q", validID, result.Document.RootTasks[0].ID)
		}
	})
}

// TestProperty15_InvalidTaskIDHandling_Deterministic verifies that parsing lines
// with invalid task IDs produces consistent results.
//
// **Validates: Requirements 1.11**
//
// Tag: Feature: fuse-task-filesystem, Property 15: Invalid Task ID Handling
func TestProperty15_InvalidTaskIDHandling_Deterministic(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate an invalid task ID
		invalidID := genAnyInvalidTaskID().Draw(t, "invalidID")

		// Validate multiple times - should always return false
		result1 := p.ValidateTaskID(invalidID)
		result2 := p.ValidateTaskID(invalidID)
		result3 := p.ValidateTaskID(invalidID)

		if result1 != result2 || result2 != result3 {
			t.Fatalf("ValidateTaskID(%q) returned inconsistent results: %v, %v, %v",
				invalidID, result1, result2, result3)
		}

		if result1 {
			t.Fatalf("ValidateTaskID(%q) returned true for invalid ID", invalidID)
		}
	})
}


// =============================================================================
// Property 16: Orphaned Task ID Resolution
// =============================================================================
//
// Property Statement:
// *For any* task with an orphaned ID (where the implied parent does not exist),
// the Tasks_MD_Parser SHALL attach it to the nearest valid ancestor by ID prefix
// match, or treat it as a root task if no ancestor exists.
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution

// genOrphanedTaskScenario generates a scenario with orphaned task IDs.
// It creates a set of tasks where some tasks have IDs that imply missing parents.
type OrphanedTaskScenario struct {
	// ExistingTasks are tasks that exist (with valid IDs)
	ExistingTasks []struct {
		ID    string
		Title string
	}
	// OrphanedTask is the task with an orphaned ID
	OrphanedTask struct {
		ID    string
		Title string
	}
	// ExpectedAncestorID is the ID of the expected ancestor (empty if should be root)
	ExpectedAncestorID string
}

// genOrphanedTaskScenario generates a scenario with an orphaned task.
func genOrphanedTaskScenario() *rapid.Generator[OrphanedTaskScenario] {
	return rapid.Custom(func(t *rapid.T) OrphanedTaskScenario {
		scenario := OrphanedTaskScenario{}

		// Generate a base ID depth (1-4 levels)
		baseDepth := rapid.IntRange(1, 4).Draw(t, "baseDepth")

		// Generate the base segments
		baseSegments := make([]string, baseDepth)
		for i := 0; i < baseDepth; i++ {
			num := rapid.IntRange(1, 20).Draw(t, fmt.Sprintf("baseSegment_%d", i))
			baseSegments[i] = fmt.Sprintf("%d", num)
		}

		// Create the orphaned task ID by adding 1-3 more levels
		orphanedExtraLevels := rapid.IntRange(1, 3).Draw(t, "orphanedExtraLevels")
		orphanedSegments := make([]string, baseDepth+orphanedExtraLevels)
		copy(orphanedSegments, baseSegments)
		for i := baseDepth; i < baseDepth+orphanedExtraLevels; i++ {
			num := rapid.IntRange(1, 20).Draw(t, fmt.Sprintf("orphanedSegment_%d", i))
			orphanedSegments[i] = fmt.Sprintf("%d", num)
		}
		scenario.OrphanedTask.ID = strings.Join(orphanedSegments, ".")
		scenario.OrphanedTask.Title = genTaskTitle().Draw(t, "orphanedTitle")

		// Decide which ancestors exist (at least one must be missing to create orphan)
		// We'll randomly decide which prefixes exist
		existingPrefixes := make(map[string]bool)

		// Determine which ancestors exist
		// At least one intermediate ancestor must be missing
		missingIndex := rapid.IntRange(baseDepth, baseDepth+orphanedExtraLevels-1).Draw(t, "missingIndex")

		for i := 0; i < baseDepth+orphanedExtraLevels-1; i++ {
			prefix := strings.Join(orphanedSegments[:i+1], ".")
			if i < missingIndex {
				// This ancestor exists
				existingPrefixes[prefix] = true
			}
			// Ancestors at or after missingIndex don't exist
		}

		// Create existing tasks from the prefixes
		for prefix := range existingPrefixes {
			title := genTaskTitle().Draw(t, fmt.Sprintf("title_%s", prefix))
			scenario.ExistingTasks = append(scenario.ExistingTasks, struct {
				ID    string
				Title string
			}{
				ID:    prefix,
				Title: title,
			})
		}

		// Determine expected ancestor
		// Search from longest to shortest prefix (excluding the orphaned task's own ID)
		for i := len(orphanedSegments) - 2; i >= 0; i-- {
			prefix := strings.Join(orphanedSegments[:i+1], ".")
			if existingPrefixes[prefix] {
				scenario.ExpectedAncestorID = prefix
				break
			}
		}
		// If no ancestor found, ExpectedAncestorID remains empty (root task)

		return scenario
	})
}

// TestProperty16_OrphanedIDResolution_GetIDPrefixes tests that GetIDPrefixes
// correctly extracts all prefix segments from a task ID.
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution
func TestProperty16_OrphanedIDResolution_GetIDPrefixes(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid task ID
		taskID := genTaskID().Draw(t, "taskID")

		// Get prefixes
		prefixes := p.GetIDPrefixes(taskID)

		// Property 1: Number of prefixes should equal number of segments
		segments := strings.Split(taskID, ".")
		if len(prefixes) != len(segments) {
			t.Fatalf("GetIDPrefixes(%q) returned %d prefixes, expected %d (number of segments)",
				taskID, len(prefixes), len(segments))
		}

		// Property 2: Last prefix should be the full ID
		if prefixes[len(prefixes)-1] != taskID {
			t.Fatalf("GetIDPrefixes(%q) last prefix is %q, expected %q",
				taskID, prefixes[len(prefixes)-1], taskID)
		}

		// Property 3: First prefix should be the first segment
		if prefixes[0] != segments[0] {
			t.Fatalf("GetIDPrefixes(%q) first prefix is %q, expected %q",
				taskID, prefixes[0], segments[0])
		}

		// Property 4: Each prefix should be a valid prefix of the next
		for i := 0; i < len(prefixes)-1; i++ {
			if !strings.HasPrefix(prefixes[i+1], prefixes[i]+".") {
				t.Fatalf("GetIDPrefixes(%q) prefix %q is not a prefix of %q",
					taskID, prefixes[i], prefixes[i+1])
			}
		}

		// Property 5: Each prefix should be a valid task ID
		for _, prefix := range prefixes {
			if !p.ValidateTaskID(prefix) {
				t.Fatalf("GetIDPrefixes(%q) returned invalid prefix %q", taskID, prefix)
			}
		}
	})
}

// TestProperty16_OrphanedIDResolution_FindAncestorByIDPrefix tests that
// FindAncestorByIDPrefix correctly finds the nearest valid ancestor.
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution
func TestProperty16_OrphanedIDResolution_FindAncestorByIDPrefix(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate an orphaned task scenario
		scenario := genOrphanedTaskScenario().Draw(t, "scenario")

		// Create task objects for existing tasks
		var tasks []*store.Task
		for _, existing := range scenario.ExistingTasks {
			task := &store.Task{
				ID:    existing.ID,
				Title: existing.Title,
			}
			tasks = append(tasks, task)
		}

		// Find ancestor for the orphaned task
		ancestor := p.FindAncestorByIDPrefix(scenario.OrphanedTask.ID, tasks)

		// Verify the result
		if scenario.ExpectedAncestorID == "" {
			// Should be nil (no ancestor found, treat as root)
			if ancestor != nil {
				t.Fatalf("FindAncestorByIDPrefix(%q) returned ancestor %q, expected nil (root task)",
					scenario.OrphanedTask.ID, ancestor.ID)
			}
		} else {
			// Should find the expected ancestor
			if ancestor == nil {
				t.Fatalf("FindAncestorByIDPrefix(%q) returned nil, expected ancestor %q",
					scenario.OrphanedTask.ID, scenario.ExpectedAncestorID)
			}
			if ancestor.ID != scenario.ExpectedAncestorID {
				t.Fatalf("FindAncestorByIDPrefix(%q) returned ancestor %q, expected %q",
					scenario.OrphanedTask.ID, ancestor.ID, scenario.ExpectedAncestorID)
			}
		}
	})
}

// TestProperty16_OrphanedIDResolution_NearestAncestor tests that FindAncestorByIDPrefix
// always returns the NEAREST valid ancestor (longest matching prefix).
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution
func TestProperty16_OrphanedIDResolution_NearestAncestor(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a deep task ID (3-6 levels)
		depth := rapid.IntRange(3, 6).Draw(t, "depth")
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			num := rapid.IntRange(1, 20).Draw(t, fmt.Sprintf("segment_%d", i))
			segments[i] = fmt.Sprintf("%d", num)
		}
		orphanedID := strings.Join(segments, ".")

		// Create multiple ancestors at different levels
		var tasks []*store.Task
		var existingPrefixes []string

		// Randomly decide which ancestors exist (at least 2 for meaningful test)
		numAncestors := rapid.IntRange(2, depth-1).Draw(t, "numAncestors")
		ancestorIndices := make([]int, 0, numAncestors)

		// Select random indices for ancestors
		for i := 0; i < depth-1; i++ {
			if rapid.Bool().Draw(t, fmt.Sprintf("includeAncestor_%d", i)) && len(ancestorIndices) < numAncestors {
				ancestorIndices = append(ancestorIndices, i)
			}
		}

		// Ensure we have at least one ancestor
		if len(ancestorIndices) == 0 {
			ancestorIndices = append(ancestorIndices, 0)
		}

		// Create tasks for selected ancestors
		for _, idx := range ancestorIndices {
			prefix := strings.Join(segments[:idx+1], ".")
			existingPrefixes = append(existingPrefixes, prefix)
			task := &store.Task{
				ID:    prefix,
				Title: fmt.Sprintf("Task %s", prefix),
			}
			tasks = append(tasks, task)
		}

		// Find ancestor
		ancestor := p.FindAncestorByIDPrefix(orphanedID, tasks)

		if ancestor == nil {
			// This should only happen if no ancestors exist
			if len(tasks) > 0 {
				t.Fatalf("FindAncestorByIDPrefix(%q) returned nil but ancestors exist: %v",
					orphanedID, existingPrefixes)
			}
		} else {
			// Verify it's the nearest (longest prefix) ancestor
			// Find the expected nearest ancestor
			var expectedNearest string
			for i := depth - 2; i >= 0; i-- {
				prefix := strings.Join(segments[:i+1], ".")
				for _, existing := range existingPrefixes {
					if existing == prefix {
						expectedNearest = prefix
						break
					}
				}
				if expectedNearest != "" {
					break
				}
			}

			if ancestor.ID != expectedNearest {
				t.Fatalf("FindAncestorByIDPrefix(%q) returned %q, expected nearest ancestor %q (existing: %v)",
					orphanedID, ancestor.ID, expectedNearest, existingPrefixes)
			}
		}
	})
}

// TestProperty16_OrphanedIDResolution_NoAncestorTreatedAsRoot tests that when
// no ancestor exists, the orphaned task is treated as a root task.
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution
func TestProperty16_OrphanedIDResolution_NoAncestorTreatedAsRoot(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate an orphaned task ID with multiple levels
		depth := rapid.IntRange(2, 5).Draw(t, "depth")
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			num := rapid.IntRange(1, 20).Draw(t, fmt.Sprintf("segment_%d", i))
			segments[i] = fmt.Sprintf("%d", num)
		}
		orphanedID := strings.Join(segments, ".")

		// Create tasks that are NOT ancestors of the orphaned task
		// (different first segment)
		var tasks []*store.Task
		differentFirstSegment := rapid.IntRange(100, 200).Draw(t, "differentFirst")
		for i := 0; i < 3; i++ {
			task := &store.Task{
				ID:    fmt.Sprintf("%d.%d", differentFirstSegment, i+1),
				Title: fmt.Sprintf("Unrelated task %d", i+1),
			}
			tasks = append(tasks, task)
		}

		// Find ancestor - should return nil (no matching ancestor)
		ancestor := p.FindAncestorByIDPrefix(orphanedID, tasks)

		if ancestor != nil {
			t.Fatalf("FindAncestorByIDPrefix(%q) returned %q, expected nil (no ancestor exists)",
				orphanedID, ancestor.ID)
		}
	})
}

// TestProperty16_OrphanedIDResolution_EmptyTaskList tests that FindAncestorByIDPrefix
// handles empty task lists correctly.
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution
func TestProperty16_OrphanedIDResolution_EmptyTaskList(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate any valid task ID
		taskID := genTaskID().Draw(t, "taskID")

		// Find ancestor with empty task list
		ancestor := p.FindAncestorByIDPrefix(taskID, []*store.Task{})

		if ancestor != nil {
			t.Fatalf("FindAncestorByIDPrefix(%q, []) returned %q, expected nil",
				taskID, ancestor.ID)
		}

		// Also test with nil task list
		ancestor = p.FindAncestorByIDPrefix(taskID, nil)

		if ancestor != nil {
			t.Fatalf("FindAncestorByIDPrefix(%q, nil) returned %q, expected nil",
				taskID, ancestor.ID)
		}
	})
}

// TestProperty16_OrphanedIDResolution_SingleLevelID tests that single-level IDs
// (e.g., "1") have no ancestors.
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution
func TestProperty16_OrphanedIDResolution_SingleLevelID(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a single-level ID
		num := rapid.IntRange(1, 100).Draw(t, "num")
		singleLevelID := fmt.Sprintf("%d", num)

		// Create some tasks (none can be ancestors of a single-level ID)
		var tasks []*store.Task
		for i := 0; i < 5; i++ {
			task := &store.Task{
				ID:    fmt.Sprintf("%d.%d", num, i+1),
				Title: fmt.Sprintf("Child task %d", i+1),
			}
			tasks = append(tasks, task)
		}

		// Find ancestor - should return nil (single-level IDs have no ancestors)
		ancestor := p.FindAncestorByIDPrefix(singleLevelID, tasks)

		if ancestor != nil {
			t.Fatalf("FindAncestorByIDPrefix(%q) returned %q, expected nil (single-level IDs have no ancestors)",
				singleLevelID, ancestor.ID)
		}
	})
}

// TestProperty16_OrphanedIDResolution_Deterministic tests that FindAncestorByIDPrefix
// produces consistent results for the same inputs.
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution
func TestProperty16_OrphanedIDResolution_Deterministic(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate an orphaned task scenario
		scenario := genOrphanedTaskScenario().Draw(t, "scenario")

		// Create task objects
		var tasks []*store.Task
		for _, existing := range scenario.ExistingTasks {
			task := &store.Task{
				ID:    existing.ID,
				Title: existing.Title,
			}
			tasks = append(tasks, task)
		}

		// Call FindAncestorByIDPrefix multiple times
		result1 := p.FindAncestorByIDPrefix(scenario.OrphanedTask.ID, tasks)
		result2 := p.FindAncestorByIDPrefix(scenario.OrphanedTask.ID, tasks)
		result3 := p.FindAncestorByIDPrefix(scenario.OrphanedTask.ID, tasks)

		// All results should be identical
		getID := func(task *store.Task) string {
			if task == nil {
				return "<nil>"
			}
			return task.ID
		}

		if getID(result1) != getID(result2) || getID(result2) != getID(result3) {
			t.Fatalf("FindAncestorByIDPrefix(%q) returned inconsistent results: %s, %s, %s",
				scenario.OrphanedTask.ID, getID(result1), getID(result2), getID(result3))
		}
	})
}

// TestProperty16_OrphanedIDResolution_AncestorIsValidPrefix tests that the returned
// ancestor's ID is always a valid prefix of the orphaned task's ID.
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution
func TestProperty16_OrphanedIDResolution_AncestorIsValidPrefix(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate an orphaned task scenario
		scenario := genOrphanedTaskScenario().Draw(t, "scenario")

		// Create task objects
		var tasks []*store.Task
		for _, existing := range scenario.ExistingTasks {
			task := &store.Task{
				ID:    existing.ID,
				Title: existing.Title,
			}
			tasks = append(tasks, task)
		}

		// Find ancestor
		ancestor := p.FindAncestorByIDPrefix(scenario.OrphanedTask.ID, tasks)

		if ancestor != nil {
			// Verify the ancestor's ID is a valid prefix of the orphaned task's ID
			if !strings.HasPrefix(scenario.OrphanedTask.ID, ancestor.ID+".") {
				t.Fatalf("FindAncestorByIDPrefix(%q) returned ancestor %q which is not a prefix",
					scenario.OrphanedTask.ID, ancestor.ID)
			}

			// Verify the ancestor's ID is shorter than the orphaned task's ID
			if len(ancestor.ID) >= len(scenario.OrphanedTask.ID) {
				t.Fatalf("FindAncestorByIDPrefix(%q) returned ancestor %q which is not shorter",
					scenario.OrphanedTask.ID, ancestor.ID)
			}
		}
	})
}

// TestProperty16_OrphanedIDResolution_ExampleFromDesign tests the specific example
// from the design document: task "1.2.3" exists but "1.2" does not.
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution
func TestProperty16_OrphanedIDResolution_ExampleFromDesign(t *testing.T) {
	p := NewParser()

	// Create the scenario from the design document:
	// Task "1.2.3" exists but "1.2" does not
	// Task "1" exists
	// Expected: "1.2.3" should be attached to "1"

	tasks := []*store.Task{
		{ID: "1", Title: "Parent task"},
		// Note: "1.2" does NOT exist
	}

	// Find ancestor for "1.2.3"
	ancestor := p.FindAncestorByIDPrefix("1.2.3", tasks)

	if ancestor == nil {
		t.Fatalf("FindAncestorByIDPrefix(\"1.2.3\") returned nil, expected task \"1\"")
	}

	if ancestor.ID != "1" {
		t.Fatalf("FindAncestorByIDPrefix(\"1.2.3\") returned %q, expected \"1\"", ancestor.ID)
	}
}

// TestProperty16_OrphanedIDResolution_MultipleGaps tests scenarios with multiple
// missing ancestors in the hierarchy.
//
// **Validates: Requirements 1.12**
//
// Tag: Feature: fuse-task-filesystem, Property 16: Orphaned Task ID Resolution
func TestProperty16_OrphanedIDResolution_MultipleGaps(t *testing.T) {
	p := NewParser()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a deep ID (5-7 levels)
		depth := rapid.IntRange(5, 7).Draw(t, "depth")
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			num := rapid.IntRange(1, 20).Draw(t, fmt.Sprintf("segment_%d", i))
			segments[i] = fmt.Sprintf("%d", num)
		}
		orphanedID := strings.Join(segments, ".")

		// Create ancestors with gaps (only some levels exist)
		var tasks []*store.Task
		var existingLevels []int

		// Randomly select which levels exist (ensure at least one)
		for i := 0; i < depth-1; i++ {
			if rapid.Bool().Draw(t, fmt.Sprintf("level_%d_exists", i)) {
				existingLevels = append(existingLevels, i)
				prefix := strings.Join(segments[:i+1], ".")
				tasks = append(tasks, &store.Task{
					ID:    prefix,
					Title: fmt.Sprintf("Task at level %d", i),
				})
			}
		}

		// Find ancestor
		ancestor := p.FindAncestorByIDPrefix(orphanedID, tasks)

		if len(existingLevels) == 0 {
			// No ancestors exist - should return nil
			if ancestor != nil {
				t.Fatalf("FindAncestorByIDPrefix(%q) returned %q, expected nil (no ancestors)",
					orphanedID, ancestor.ID)
			}
		} else {
			// Should return the nearest (highest level) existing ancestor
			if ancestor == nil {
				t.Fatalf("FindAncestorByIDPrefix(%q) returned nil, but ancestors exist at levels %v",
					orphanedID, existingLevels)
			}

			// Find the expected nearest ancestor (highest level that exists)
			expectedLevel := existingLevels[len(existingLevels)-1]
			expectedID := strings.Join(segments[:expectedLevel+1], ".")

			if ancestor.ID != expectedID {
				t.Fatalf("FindAncestorByIDPrefix(%q) returned %q, expected nearest ancestor %q",
					orphanedID, ancestor.ID, expectedID)
			}
		}
	})
}


// =============================================================================
// Property 1: Parser Round-Trip Consistency
// =============================================================================
//
// Property Statement:
// *For any* valid task tree, serializing it with Pretty_Printer and then parsing
// the result with Tasks_MD_Parser SHALL produce an equivalent task tree.
//
// **Validates: Requirements 1.8, 1.9, 1.10**
//
// Tag: Feature: fuse-task-filesystem, Property 1: Parser Round-Trip Consistency

// allStatuses contains all valid task statuses for generation.
var allStatuses = []store.TaskStatus{
	store.StatusPending,
	store.StatusQueued,
	store.StatusDoing,
	store.StatusDone,
	store.StatusFailed,
}

// genStatus generates a random valid task status.
func genStatus() *rapid.Generator[store.TaskStatus] {
	return rapid.SampledFrom(allStatuses)
}

// genPreambleLine generates a random non-task preamble line.
func genPreambleLine() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		lineType := rapid.IntRange(0, 4).Draw(t, "lineType")
		switch lineType {
		case 0:
			// Heading
			level := rapid.IntRange(1, 3).Draw(t, "headingLevel")
			title := genTaskTitle().Draw(t, "headingTitle")
			return strings.Repeat("#", level) + " " + title
		case 1:
			// Empty line
			return ""
		case 2:
			// Plain text
			return genTaskTitle().Draw(t, "plainText")
		case 3:
			// Comment-like line
			return "<!-- " + genTaskTitle().Draw(t, "comment") + " -->"
		default:
			// Bullet point without checkbox
			return "* " + genTaskTitle().Draw(t, "bullet")
		}
	})
}

// genDescriptionLine generates a random description line for a task.
func genDescriptionLine(baseIndent int) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Description lines should be more indented than the task
		extraIndent := rapid.IntRange(1, 2).Draw(t, "extraIndent")
		totalIndent := (baseIndent + extraIndent) * 2 // Convert to spaces
		indent := strings.Repeat(" ", totalIndent)

		lineType := rapid.IntRange(0, 3).Draw(t, "descLineType")
		switch lineType {
		case 0:
			// Plain text description
			return indent + genTaskTitle().Draw(t, "descText")
		case 1:
			// Bullet point
			return indent + "- " + genTaskTitle().Draw(t, "descBullet")
		case 2:
			// Numbered item
			num := rapid.IntRange(1, 10).Draw(t, "descNum")
			return indent + fmt.Sprintf("%d. ", num) + genTaskTitle().Draw(t, "descNumbered")
		default:
			// Code or special content
			return indent + "`" + genTaskTitle().Draw(t, "descCode") + "`"
		}
	})
}

// genValidTaskIDAtDepth generates a valid task ID at a specific depth.
func genValidTaskIDAtDepth(depth int) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			num := rapid.IntRange(1, 20).Draw(t, fmt.Sprintf("idSegment_%d", i))
			segments[i] = fmt.Sprintf("%d", num)
		}
		return strings.Join(segments, ".")
	})
}

// statusToCheckboxChar returns the checkbox character for a status.
func statusToCheckboxChar(status store.TaskStatus) string {
	switch status {
	case store.StatusPending:
		return " "
	case store.StatusQueued:
		return "~"
	case store.StatusDoing:
		return "-"
	case store.StatusDone:
		return "x"
	case store.StatusFailed:
		return "!"
	default:
		return " "
	}
}

// GeneratedTask represents a task generated for testing.
type GeneratedTask struct {
	ID          string
	Title       string
	Status      store.TaskStatus
	IndentLevel int
	Description []string
	Children    []*GeneratedTask
}

// genTaskTree generates a random valid task tree with proper hierarchy.
func genTaskTree(maxDepth int, parentID string, indentLevel int) *rapid.Generator[[]*GeneratedTask] {
	return rapid.Custom(func(t *rapid.T) []*GeneratedTask {
		if maxDepth <= 0 {
			return nil
		}

		// Generate 0-4 tasks at this level
		numTasks := rapid.IntRange(0, 4).Draw(t, "numTasks")
		if numTasks == 0 {
			return nil
		}

		tasks := make([]*GeneratedTask, numTasks)
		for i := 0; i < numTasks; i++ {
			// Generate task ID
			var taskID string
			if parentID == "" {
				taskID = fmt.Sprintf("%d", i+1)
			} else {
				taskID = fmt.Sprintf("%s.%d", parentID, i+1)
			}

			// Generate task properties
			title := genTaskTitle().Draw(t, fmt.Sprintf("title_%s", taskID))
			status := genStatus().Draw(t, fmt.Sprintf("status_%s", taskID))

			// Generate 0-3 description lines
			numDesc := rapid.IntRange(0, 3).Draw(t, fmt.Sprintf("numDesc_%s", taskID))
			desc := make([]string, numDesc)
			for j := 0; j < numDesc; j++ {
				desc[j] = genDescriptionLine(indentLevel).Draw(t, fmt.Sprintf("desc_%s_%d", taskID, j))
			}

			// Recursively generate children (with reduced probability at deeper levels)
			var children []*GeneratedTask
			if maxDepth > 1 && rapid.Float64Range(0, 1).Draw(t, fmt.Sprintf("hasChildren_%s", taskID)) < 0.5 {
				children = genTaskTree(maxDepth-1, taskID, indentLevel+1).Draw(t, fmt.Sprintf("children_%s", taskID))
			}

			tasks[i] = &GeneratedTask{
				ID:          taskID,
				Title:       title,
				Status:      status,
				IndentLevel: indentLevel,
				Description: desc,
				Children:    children,
			}
		}
		return tasks
	})
}

// serializeGeneratedTask converts a GeneratedTask to markdown lines.
func serializeGeneratedTask(task *GeneratedTask) []string {
	var lines []string

	// Create the task line with proper indentation
	indent := strings.Repeat("  ", task.IndentLevel)
	checkboxChar := statusToCheckboxChar(task.Status)
	taskLine := fmt.Sprintf("%s- [%s] %s %s", indent, checkboxChar, task.ID, task.Title)
	lines = append(lines, taskLine)

	// Add description lines
	lines = append(lines, task.Description...)

	// Recursively add children
	for _, child := range task.Children {
		lines = append(lines, serializeGeneratedTask(child)...)
	}

	return lines
}

// GeneratedDocument represents a complete generated document for testing.
type GeneratedDocument struct {
	Preamble  []string
	Tasks     []*GeneratedTask
	Epilogue  []string
}

// genDocument generates a random valid ParsedDocument structure.
func genDocument() *rapid.Generator[GeneratedDocument] {
	return rapid.Custom(func(t *rapid.T) GeneratedDocument {
		doc := GeneratedDocument{}

		// Generate 0-3 preamble lines
		numPreamble := rapid.IntRange(0, 3).Draw(t, "numPreamble")
		for i := 0; i < numPreamble; i++ {
			doc.Preamble = append(doc.Preamble, genPreambleLine().Draw(t, fmt.Sprintf("preamble_%d", i)))
		}

		// Generate task tree (1-3 levels deep, starting at root)
		maxDepth := rapid.IntRange(1, 3).Draw(t, "maxDepth")
		doc.Tasks = genTaskTree(maxDepth, "", 0).Draw(t, "tasks")

		// Generate 0-2 epilogue lines
		numEpilogue := rapid.IntRange(0, 2).Draw(t, "numEpilogue")
		for i := 0; i < numEpilogue; i++ {
			doc.Epilogue = append(doc.Epilogue, genPreambleLine().Draw(t, fmt.Sprintf("epilogue_%d", i)))
		}

		return doc
	})
}

// serializeDocument converts a GeneratedDocument to markdown content.
func serializeDocument(doc GeneratedDocument) string {
	var lines []string

	// Add preamble
	lines = append(lines, doc.Preamble...)

	// Add tasks
	for _, task := range doc.Tasks {
		lines = append(lines, serializeGeneratedTask(task)...)
	}

	// Add epilogue
	lines = append(lines, doc.Epilogue...)

	return strings.Join(lines, "\n")
}

// countParsedTasks recursively counts all tasks in a parsed tree.
func countParsedTasks(tasks []*store.Task) int {
	count := len(tasks)
	for _, task := range tasks {
		count += countParsedTasks(task.Children)
	}
	return count
}

// TestProperty1_ParserRoundTripConsistency is the main property-based test for
// Property 1: Parser Round-Trip Consistency.
//
// **Validates: Requirements 1.8, 1.9, 1.10**
//
// This test generates random valid task trees, serializes them with Pretty_Printer,
// parses the result with Parser, and compares for equivalence.
//
// Tag: Feature: fuse-task-filesystem, Property 1: Parser Round-Trip Consistency
func TestProperty1_ParserRoundTripConsistency(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 1: Parser Round-Trip Consistency
	// Validates: Requirements 1.8, 1.9, 1.10

	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Step 1: Generate a random valid document
		genDoc := genDocument().Draw(t, "document")

		// Skip if no tasks generated (edge case)
		if len(genDoc.Tasks) == 0 {
			return
		}

		// Step 2: Serialize the generated document to markdown
		originalContent := serializeDocument(genDoc)

		// Step 3: Parse the original content
		parseResult1 := p.Parse(originalContent)
		if parseResult1.Document == nil {
			t.Fatalf("First parse returned nil document for content:\n%s", originalContent)
		}

		// Step 4: Serialize the parsed document using Pretty_Printer
		serialized := pr.Serialize(parseResult1.Document)

		// Step 5: Parse the serialized content again
		parseResult2 := p.Parse(serialized)
		if parseResult2.Document == nil {
			t.Fatalf("Second parse returned nil document for serialized content:\n%s", serialized)
		}

		// Step 6: Compare the two parsed documents for equivalence
		// Compare task counts
		count1 := countParsedTasks(parseResult1.Document.RootTasks)
		count2 := countParsedTasks(parseResult2.Document.RootTasks)
		if count1 != count2 {
			t.Fatalf("Task count mismatch after round-trip: first=%d, second=%d\nOriginal:\n%s\nSerialized:\n%s",
				count1, count2, originalContent, serialized)
		}

		// Compare task structure
		if err := compareParsedTasks(parseResult1.Document.RootTasks, parseResult2.Document.RootTasks); err != "" {
			t.Fatalf("Task structure mismatch after round-trip: %s\nOriginal:\n%s\nSerialized:\n%s",
				err, originalContent, serialized)
		}
	})
}

// compareParsedTasks compares two parsed task trees for equivalence.
func compareParsedTasks(tasks1, tasks2 []*store.Task) string {
	if len(tasks1) != len(tasks2) {
		return fmt.Sprintf("root task count mismatch: %d vs %d", len(tasks1), len(tasks2))
	}

	for i := range tasks1 {
		if err := compareParsedTask(tasks1[i], tasks2[i]); err != "" {
			return err
		}
	}
	return ""
}

// compareParsedTask compares two parsed tasks for equivalence.
func compareParsedTask(task1, task2 *store.Task) string {
	// Compare ID
	if task1.ID != task2.ID {
		return fmt.Sprintf("ID mismatch: %q vs %q", task1.ID, task2.ID)
	}

	// Compare Title
	if task1.Title != task2.Title {
		return fmt.Sprintf("Title mismatch for task %s: %q vs %q", task1.ID, task1.Title, task2.Title)
	}

	// Compare Status
	if task1.Status != task2.Status {
		return fmt.Sprintf("Status mismatch for task %s: %q vs %q", task1.ID, task1.Status, task2.Status)
	}

	// Compare Description count
	if len(task1.Description) != len(task2.Description) {
		return fmt.Sprintf("Description count mismatch for task %s: %d vs %d",
			task1.ID, len(task1.Description), len(task2.Description))
	}

	// Compare description content
	for j := range task1.Description {
		if task1.Description[j].RawContent != task2.Description[j].RawContent {
			return fmt.Sprintf("Description mismatch for task %s line %d: %q vs %q",
				task1.ID, j, task1.Description[j].RawContent, task2.Description[j].RawContent)
		}
	}

	// Compare Children count
	if len(task1.Children) != len(task2.Children) {
		return fmt.Sprintf("Children count mismatch for task %s: %d vs %d",
			task1.ID, len(task1.Children), len(task2.Children))
	}

	// Compare Children recursively
	return compareParsedTasks(task1.Children, task2.Children)
}

// TestProperty1_ParserRoundTripConsistency_AllStatuses tests round-trip consistency
// for tasks with all possible status types.
//
// **Validates: Requirements 1.8, 1.9, 1.10**
//
// Tag: Feature: fuse-task-filesystem, Property 1: Parser Round-Trip Consistency
func TestProperty1_ParserRoundTripConsistency_AllStatuses(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	for _, status := range allStatuses {
		status := status // capture for closure
		t.Run(fmt.Sprintf("status_%s", status), func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				// Generate a task with the specific status
				taskID := genTaskID().Draw(t, "taskID")
				title := genTaskTitle().Draw(t, "title")

				// Create content with this status
				checkboxChar := statusToCheckboxChar(status)
				content := fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title)

				// Parse
				result1 := p.Parse(content)
				if len(result1.Document.RootTasks) != 1 {
					t.Fatalf("Expected 1 task, got %d", len(result1.Document.RootTasks))
				}

				// Verify status was parsed correctly
				if result1.Document.RootTasks[0].Status != status {
					t.Fatalf("Status mismatch: expected %q, got %q",
						status, result1.Document.RootTasks[0].Status)
				}

				// Serialize and parse again
				serialized := pr.Serialize(result1.Document)
				result2 := p.Parse(serialized)

				// Verify status preserved
				if len(result2.Document.RootTasks) != 1 {
					t.Fatalf("After round-trip: expected 1 task, got %d", len(result2.Document.RootTasks))
				}
				if result2.Document.RootTasks[0].Status != status {
					t.Fatalf("Status not preserved: expected %q, got %q",
						status, result2.Document.RootTasks[0].Status)
				}
			})
		})
	}
}

// TestProperty1_ParserRoundTripConsistency_NestedTasks tests round-trip consistency
// for nested task hierarchies.
//
// **Validates: Requirements 1.8, 1.9, 1.10**
//
// Tag: Feature: fuse-task-filesystem, Property 1: Parser Round-Trip Consistency
func TestProperty1_ParserRoundTripConsistency_NestedTasks(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a nested task structure
		depth := rapid.IntRange(2, 4).Draw(t, "depth")

		var lines []string

		// Build nested structure
		currentID := ""
		for level := 0; level < depth; level++ {
			indent := strings.Repeat("  ", level)
			num := rapid.IntRange(1, 5).Draw(t, fmt.Sprintf("num_%d", level))

			if currentID == "" {
				currentID = fmt.Sprintf("%d", num)
			} else {
				currentID = fmt.Sprintf("%s.%d", currentID, num)
			}

			title := genTaskTitle().Draw(t, fmt.Sprintf("title_%d", level))
			status := genStatus().Draw(t, fmt.Sprintf("status_%d", level))
			checkboxChar := statusToCheckboxChar(status)

			line := fmt.Sprintf("%s- [%s] %s %s", indent, checkboxChar, currentID, title)
			lines = append(lines, line)
		}

		content := strings.Join(lines, "\n")

		// Parse
		result1 := p.Parse(content)

		// Serialize
		serialized := pr.Serialize(result1.Document)

		// Parse again
		result2 := p.Parse(serialized)

		// Verify structure preserved
		// Count total tasks
		count1 := countParsedTasks(result1.Document.RootTasks)
		count2 := countParsedTasks(result2.Document.RootTasks)

		if count1 != count2 {
			t.Fatalf("Task count mismatch: %d vs %d\nOriginal:\n%s\nSerialized:\n%s",
				count1, count2, content, serialized)
		}

		if count1 != depth {
			t.Fatalf("Expected %d tasks, got %d", depth, count1)
		}
	})
}

// TestProperty1_ParserRoundTripConsistency_WithDescriptions tests round-trip
// consistency for tasks with description lines.
//
// **Validates: Requirements 1.8, 1.9, 1.10**
//
// Tag: Feature: fuse-task-filesystem, Property 1: Parser Round-Trip Consistency
func TestProperty1_ParserRoundTripConsistency_WithDescriptions(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a task with descriptions
		taskID := genTaskID().Draw(t, "taskID")
		title := genTaskTitle().Draw(t, "title")
		status := genStatus().Draw(t, "status")
		checkboxChar := statusToCheckboxChar(status)

		// Generate 1-5 description lines
		numDesc := rapid.IntRange(1, 5).Draw(t, "numDesc")
		var lines []string
		lines = append(lines, fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title))

		for i := 0; i < numDesc; i++ {
			descLine := genDescriptionLine(0).Draw(t, fmt.Sprintf("desc_%d", i))
			lines = append(lines, descLine)
		}

		content := strings.Join(lines, "\n")

		// Parse
		result1 := p.Parse(content)
		if len(result1.Document.RootTasks) != 1 {
			t.Fatalf("Expected 1 task, got %d", len(result1.Document.RootTasks))
		}

		task1 := result1.Document.RootTasks[0]
		if len(task1.Description) != numDesc {
			t.Fatalf("Expected %d description lines, got %d", numDesc, len(task1.Description))
		}

		// Serialize
		serialized := pr.Serialize(result1.Document)

		// Parse again
		result2 := p.Parse(serialized)
		if len(result2.Document.RootTasks) != 1 {
			t.Fatalf("After round-trip: expected 1 task, got %d", len(result2.Document.RootTasks))
		}

		task2 := result2.Document.RootTasks[0]

		// Verify descriptions preserved
		if len(task2.Description) != len(task1.Description) {
			t.Fatalf("Description count mismatch: %d vs %d",
				len(task1.Description), len(task2.Description))
		}

		for i := range task1.Description {
			if task1.Description[i].RawContent != task2.Description[i].RawContent {
				t.Fatalf("Description line %d mismatch: %q vs %q",
					i, task1.Description[i].RawContent, task2.Description[i].RawContent)
			}
		}
	})
}

// TestProperty1_ParserRoundTripConsistency_WithPreambleAndEpilogue tests round-trip
// consistency for documents with preamble and epilogue content.
//
// **Validates: Requirements 1.8, 1.9, 1.10**
//
// Tag: Feature: fuse-task-filesystem, Property 1: Parser Round-Trip Consistency
func TestProperty1_ParserRoundTripConsistency_WithPreambleAndEpilogue(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate preamble
		numPreamble := rapid.IntRange(1, 3).Draw(t, "numPreamble")
		var preambleLines []string
		for i := 0; i < numPreamble; i++ {
			preambleLines = append(preambleLines, genPreambleLine().Draw(t, fmt.Sprintf("preamble_%d", i)))
		}

		// Generate a task
		taskID := genTaskID().Draw(t, "taskID")
		title := genTaskTitle().Draw(t, "title")
		status := genStatus().Draw(t, "status")
		checkboxChar := statusToCheckboxChar(status)
		taskLine := fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title)

		// Combine content
		var lines []string
		lines = append(lines, preambleLines...)
		lines = append(lines, taskLine)

		content := strings.Join(lines, "\n")

		// Parse
		result1 := p.Parse(content)

		// Verify preamble preserved
		if len(result1.Document.Preamble) != numPreamble {
			t.Fatalf("Expected %d preamble lines, got %d",
				numPreamble, len(result1.Document.Preamble))
		}

		// Serialize
		serialized := pr.Serialize(result1.Document)

		// Parse again
		result2 := p.Parse(serialized)

		// Verify preamble still preserved
		if len(result2.Document.Preamble) != len(result1.Document.Preamble) {
			t.Fatalf("Preamble count mismatch: %d vs %d",
				len(result1.Document.Preamble), len(result2.Document.Preamble))
		}

		for i := range result1.Document.Preamble {
			if result1.Document.Preamble[i].RawContent != result2.Document.Preamble[i].RawContent {
				t.Fatalf("Preamble line %d mismatch: %q vs %q",
					i, result1.Document.Preamble[i].RawContent, result2.Document.Preamble[i].RawContent)
			}
		}
	})
}

// TestProperty1_ParserRoundTripConsistency_Deterministic verifies that multiple
// round-trips produce consistent results.
//
// **Validates: Requirements 1.8, 1.9, 1.10**
//
// Tag: Feature: fuse-task-filesystem, Property 1: Parser Round-Trip Consistency
func TestProperty1_ParserRoundTripConsistency_Deterministic(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a document
		genDoc := genDocument().Draw(t, "document")
		if len(genDoc.Tasks) == 0 {
			return
		}

		originalContent := serializeDocument(genDoc)

		// Perform multiple round-trips
		content := originalContent
		for i := 0; i < 3; i++ {
			result := p.Parse(content)
			content = pr.Serialize(result.Document)
		}

		// Parse final result
		finalResult := p.Parse(content)

		// Parse original for comparison
		originalResult := p.Parse(originalContent)

		// Compare task counts
		originalCount := countParsedTasks(originalResult.Document.RootTasks)
		finalCount := countParsedTasks(finalResult.Document.RootTasks)

		if originalCount != finalCount {
			t.Fatalf("Task count changed after multiple round-trips: %d -> %d",
				originalCount, finalCount)
		}

		// Compare structure
		if err := compareParsedTasks(originalResult.Document.RootTasks, finalResult.Document.RootTasks); err != "" {
			t.Fatalf("Structure changed after multiple round-trips: %s", err)
		}
	})
}

// TestProperty1_ParserRoundTripConsistency_IDPreservation verifies that task IDs
// are preserved exactly through round-trips.
//
// **Validates: Requirements 1.8, 1.9, 1.10**
//
// Tag: Feature: fuse-task-filesystem, Property 1: Parser Round-Trip Consistency
func TestProperty1_ParserRoundTripConsistency_IDPreservation(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate various valid task IDs
		depth := rapid.IntRange(1, 5).Draw(t, "depth")
		taskID := genValidTaskIDAtDepth(depth).Draw(t, "taskID")
		title := genTaskTitle().Draw(t, "title")
		status := genStatus().Draw(t, "status")
		checkboxChar := statusToCheckboxChar(status)

		content := fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title)

		// Parse
		result1 := p.Parse(content)
		if len(result1.Document.RootTasks) != 1 {
			t.Fatalf("Expected 1 task, got %d", len(result1.Document.RootTasks))
		}

		// Verify ID parsed correctly
		if result1.Document.RootTasks[0].ID != taskID {
			t.Fatalf("ID not parsed correctly: expected %q, got %q",
				taskID, result1.Document.RootTasks[0].ID)
		}

		// Serialize and parse again
		serialized := pr.Serialize(result1.Document)
		result2 := p.Parse(serialized)

		// Verify ID preserved
		if len(result2.Document.RootTasks) != 1 {
			t.Fatalf("After round-trip: expected 1 task, got %d", len(result2.Document.RootTasks))
		}

		if result2.Document.RootTasks[0].ID != taskID {
			t.Fatalf("ID not preserved: expected %q, got %q",
				taskID, result2.Document.RootTasks[0].ID)
		}
	})
}

// TestProperty1_ParserRoundTripConsistency_TitlePreservation verifies that task
// titles are preserved exactly through round-trips.
//
// **Validates: Requirements 1.8, 1.9, 1.10**
//
// Tag: Feature: fuse-task-filesystem, Property 1: Parser Round-Trip Consistency
func TestProperty1_ParserRoundTripConsistency_TitlePreservation(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		taskID := genTaskID().Draw(t, "taskID")
		title := genTaskTitle().Draw(t, "title")
		status := genStatus().Draw(t, "status")
		checkboxChar := statusToCheckboxChar(status)

		content := fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title)

		// Parse
		result1 := p.Parse(content)
		if len(result1.Document.RootTasks) != 1 {
			t.Fatalf("Expected 1 task, got %d", len(result1.Document.RootTasks))
		}

		// Verify title parsed correctly
		if result1.Document.RootTasks[0].Title != title {
			t.Fatalf("Title not parsed correctly: expected %q, got %q",
				title, result1.Document.RootTasks[0].Title)
		}

		// Serialize and parse again
		serialized := pr.Serialize(result1.Document)
		result2 := p.Parse(serialized)

		// Verify title preserved
		if len(result2.Document.RootTasks) != 1 {
			t.Fatalf("After round-trip: expected 1 task, got %d", len(result2.Document.RootTasks))
		}

		if result2.Document.RootTasks[0].Title != title {
			t.Fatalf("Title not preserved: expected %q, got %q",
				title, result2.Document.RootTasks[0].Title)
		}
	})
}


// =============================================================================
// Property 13: Malformed Line Preservation
// =============================================================================
//
// Property Statement:
// *For any* line in tasks.md that does not match valid checkbox syntax, the
// Tasks_MD_Parser SHALL preserve it unchanged in the output when serialized
// by Pretty_Printer.
//
// **Validates: Requirements 7.4**
//
// Tag: Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation

// MalformedLineType represents different categories of malformed lines.
type MalformedLineType int

const (
	// MalformedNoID: Lines with checkbox syntax but NO numeric ID
	// e.g., "- [ ] No number here"
	MalformedNoID MalformedLineType = iota

	// MalformedInvalidCheckboxChar: Lines with invalid checkbox characters
	// e.g., "- [?] 1.1 Task"
	MalformedInvalidCheckboxChar

	// MalformedLeadingZeroID: Lines with leading zeros in ID
	// e.g., "- [ ] 01.1 Task"
	MalformedLeadingZeroID

	// MalformedNonNumericID: Lines with non-numeric segments in ID
	// e.g., "- [ ] 1.2.a Task"
	MalformedNonNumericID

	// MalformedEmptySegmentID: Lines with empty segments in ID
	// e.g., "- [ ] 1..2 Task"
	MalformedEmptySegmentID

	// MalformedMissingDash: Lines missing the leading dash
	// e.g., "[ ] 1.1 Task"
	MalformedMissingDash

	// MalformedMissingBrackets: Lines missing brackets
	// e.g., "- 1.1 Task"
	MalformedMissingBrackets

	// MalformedWrongBracketType: Lines with wrong bracket type
	// e.g., "- (x) 1.1 Task"
	MalformedWrongBracketType

	// NonTaskHeading: Regular heading lines
	// e.g., "# Heading"
	NonTaskHeading

	// NonTaskPlainText: Plain text lines
	// e.g., "Some text"
	NonTaskPlainText

	// NonTaskEmptyLine: Empty lines
	// e.g., ""
	NonTaskEmptyLine

	// NonTaskBulletPoint: Bullet points without checkbox
	// e.g., "* Item"
	NonTaskBulletPoint
)

// genMalformedLine generates a malformed line based on the specified type.
func genMalformedLine(lineType MalformedLineType) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		indent := genIndentation().Draw(t, "indent")
		title := genTaskTitle().Draw(t, "title")

		switch lineType {
		case MalformedNoID:
			// Checkbox syntax but NO numeric ID
			checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
			return fmt.Sprintf("%s- [%s] %s", indent, checkboxChar, title)

		case MalformedInvalidCheckboxChar:
			// Invalid checkbox characters like ?, *, +, etc.
			invalidChars := []string{"?", "*", "+", "o", "O", "X", ".", "/", "\\", "0", "1"}
			invalidChar := rapid.SampledFrom(invalidChars).Draw(t, "invalidChar")
			taskID := genTaskID().Draw(t, "taskID")
			return fmt.Sprintf("%s- [%s] %s %s", indent, invalidChar, taskID, title)

		case MalformedLeadingZeroID:
			// Leading zeros in ID: "01.1" or "1.01"
			invalidID := genInvalidTaskID(InvalidIDLeadingZeroFirst).Draw(t, "invalidID")
			checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
			return fmt.Sprintf("%s- [%s] %s %s", indent, checkboxChar, invalidID, title)

		case MalformedNonNumericID:
			// Non-numeric segments: "1.2.a"
			invalidID := genInvalidTaskID(InvalidIDNonNumericSegment).Draw(t, "invalidID")
			checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
			return fmt.Sprintf("%s- [%s] %s %s", indent, checkboxChar, invalidID, title)

		case MalformedEmptySegmentID:
			// Empty segments: "1..2"
			invalidID := genInvalidTaskID(InvalidIDEmptySegment).Draw(t, "invalidID")
			checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
			return fmt.Sprintf("%s- [%s] %s %s", indent, checkboxChar, invalidID, title)

		case MalformedMissingDash:
			// Missing dash: "[ ] 1.1 Task"
			checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
			taskID := genTaskID().Draw(t, "taskID")
			return fmt.Sprintf("%s[%s] %s %s", indent, checkboxChar, taskID, title)

		case MalformedMissingBrackets:
			// Missing brackets: "- 1.1 Task"
			taskID := genTaskID().Draw(t, "taskID")
			return fmt.Sprintf("%s- %s %s", indent, taskID, title)

		case MalformedWrongBracketType:
			// Wrong bracket type: "- (x) 1.1 Task" or "- {x} 1.1 Task"
			bracketTypes := []struct{ open, close string }{
				{"(", ")"}, {"{", "}"}, {"<", ">"}, {"[", ")"}, {"(", "]"},
			}
			brackets := rapid.SampledFrom(bracketTypes).Draw(t, "brackets")
			checkboxChar := genCheckboxChar().Draw(t, "checkboxChar")
			taskID := genTaskID().Draw(t, "taskID")
			return fmt.Sprintf("%s- %s%s%s %s %s", indent, brackets.open, checkboxChar, brackets.close, taskID, title)

		case NonTaskHeading:
			// Heading: "# Heading"
			level := rapid.IntRange(1, 4).Draw(t, "headingLevel")
			return fmt.Sprintf("%s%s %s", indent, strings.Repeat("#", level), title)

		case NonTaskPlainText:
			// Plain text
			return fmt.Sprintf("%s%s", indent, title)

		case NonTaskEmptyLine:
			// Empty line (may have whitespace)
			if rapid.Bool().Draw(t, "hasWhitespace") {
				return indent
			}
			return ""

		case NonTaskBulletPoint:
			// Bullet point without checkbox: "* Item" or "+ Item"
			bulletChars := []string{"*", "+"}
			bullet := rapid.SampledFrom(bulletChars).Draw(t, "bullet")
			return fmt.Sprintf("%s%s %s", indent, bullet, title)

		default:
			return title
		}
	})
}

// genAnyMalformedLine generates any type of malformed line.
func genAnyMalformedLine() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		lineType := MalformedLineType(rapid.IntRange(0, int(NonTaskBulletPoint)).Draw(t, "lineType"))
		return genMalformedLine(lineType).Draw(t, "malformedLine")
	})
}

// MalformedDocumentScenario represents a document with malformed lines.
type MalformedDocumentScenario struct {
	// Content is the full document content
	Content string
	// MalformedLines are the malformed lines that should be preserved
	MalformedLines []string
	// MalformedLinePositions maps line content to its position (preamble, description, epilogue)
	MalformedLinePositions map[string]string
}

// genMalformedDocumentScenario generates a document with a mix of valid tasks
// and malformed lines in various positions.
func genMalformedDocumentScenario() *rapid.Generator[MalformedDocumentScenario] {
	return rapid.Custom(func(t *rapid.T) MalformedDocumentScenario {
		scenario := MalformedDocumentScenario{
			MalformedLinePositions: make(map[string]string),
		}
		var lines []string

		// Generate 0-3 malformed lines in preamble
		numPreambleMalformed := rapid.IntRange(0, 3).Draw(t, "numPreambleMalformed")
		for i := 0; i < numPreambleMalformed; i++ {
			malformedLine := genAnyMalformedLine().Draw(t, fmt.Sprintf("preambleMalformed_%d", i))
			lines = append(lines, malformedLine)
			scenario.MalformedLines = append(scenario.MalformedLines, malformedLine)
			scenario.MalformedLinePositions[malformedLine] = "preamble"
		}

		// Generate 1-3 valid tasks with malformed lines in descriptions
		numTasks := rapid.IntRange(1, 3).Draw(t, "numTasks")
		for i := 0; i < numTasks; i++ {
			// Generate valid task
			taskID := fmt.Sprintf("%d", i+1)
			title := genTaskTitle().Draw(t, fmt.Sprintf("taskTitle_%d", i))
			status := genStatus().Draw(t, fmt.Sprintf("taskStatus_%d", i))
			checkboxChar := statusToCheckboxChar(status)
			taskLine := fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title)
			lines = append(lines, taskLine)

			// Generate 0-2 malformed lines as description
			numDescMalformed := rapid.IntRange(0, 2).Draw(t, fmt.Sprintf("numDescMalformed_%d", i))
			for j := 0; j < numDescMalformed; j++ {
				// Description lines need to be indented
				malformedLine := "  " + genAnyMalformedLine().Draw(t, fmt.Sprintf("descMalformed_%d_%d", i, j))
				lines = append(lines, malformedLine)
				scenario.MalformedLines = append(scenario.MalformedLines, malformedLine)
				scenario.MalformedLinePositions[malformedLine] = fmt.Sprintf("description_task_%d", i+1)
			}
		}

		// Generate 0-2 malformed lines in epilogue
		numEpilogueMalformed := rapid.IntRange(0, 2).Draw(t, "numEpilogueMalformed")
		for i := 0; i < numEpilogueMalformed; i++ {
			malformedLine := genAnyMalformedLine().Draw(t, fmt.Sprintf("epilogueMalformed_%d", i))
			lines = append(lines, malformedLine)
			scenario.MalformedLines = append(scenario.MalformedLines, malformedLine)
			scenario.MalformedLinePositions[malformedLine] = "epilogue"
		}

		scenario.Content = strings.Join(lines, "\n")
		return scenario
	})
}

// TestProperty13_MalformedLinePreservation is the main property-based test for
// Property 13: Malformed Line Preservation.
//
// **Validates: Requirements 7.4**
//
// This test generates documents with malformed lines (invalid checkboxes, no IDs,
// etc.) and verifies that these lines are preserved unchanged through the
// parse → serialize → parse round-trip.
//
// Tag: Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation
func TestProperty13_MalformedLinePreservation(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation
	// Validates: Requirements 7.4

	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a document with malformed lines
		scenario := genMalformedDocumentScenario().Draw(t, "scenario")

		// Skip if no malformed lines generated
		if len(scenario.MalformedLines) == 0 {
			return
		}

		// Step 1: Parse the original content
		parseResult1 := p.Parse(scenario.Content)
		if parseResult1.Document == nil {
			t.Fatalf("First parse returned nil document for content:\n%s", scenario.Content)
		}

		// Step 2: Serialize the parsed document
		serialized := pr.Serialize(parseResult1.Document)

		// Step 3: Parse the serialized content again
		parseResult2 := p.Parse(serialized)
		if parseResult2.Document == nil {
			t.Fatalf("Second parse returned nil document for serialized content:\n%s", serialized)
		}

		// Step 4: Serialize again for final verification
		finalSerialized := pr.Serialize(parseResult2.Document)

		// Step 5: Verify all malformed lines are preserved in the final output
		for _, malformedLine := range scenario.MalformedLines {
			// Check if the malformed line appears in the final serialized output
			if !strings.Contains(finalSerialized, malformedLine) {
				t.Fatalf("Malformed line not preserved through round-trip.\n"+
					"Line: %q\n"+
					"Position: %s\n"+
					"Original content:\n%s\n"+
					"First serialization:\n%s\n"+
					"Final serialization:\n%s",
					malformedLine,
					scenario.MalformedLinePositions[malformedLine],
					scenario.Content,
					serialized,
					finalSerialized)
			}
		}
	})
}

// TestProperty13_MalformedLinePreservation_AllTypes tests each specific type
// of malformed line to ensure comprehensive coverage.
//
// **Validates: Requirements 7.4**
//
// Tag: Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation
func TestProperty13_MalformedLinePreservation_AllTypes(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	malformedTypes := []struct {
		name     string
		lineType MalformedLineType
	}{
		{"no_id", MalformedNoID},
		{"invalid_checkbox_char", MalformedInvalidCheckboxChar},
		{"leading_zero_id", MalformedLeadingZeroID},
		{"non_numeric_id", MalformedNonNumericID},
		{"empty_segment_id", MalformedEmptySegmentID},
		{"missing_dash", MalformedMissingDash},
		{"missing_brackets", MalformedMissingBrackets},
		{"wrong_bracket_type", MalformedWrongBracketType},
		{"heading", NonTaskHeading},
		{"plain_text", NonTaskPlainText},
		{"empty_line", NonTaskEmptyLine},
		{"bullet_point", NonTaskBulletPoint},
	}

	for _, tc := range malformedTypes {
		tc := tc // capture for closure
		t.Run(tc.name, func(t *testing.T) {
			rapid.Check(t, func(rt *rapid.T) {
				// Generate a malformed line of this specific type
				malformedLine := genMalformedLine(tc.lineType).Draw(rt, "malformedLine")

				// Generate a valid task to provide context
				taskID := genTaskID().Draw(rt, "taskID")
				title := genTaskTitle().Draw(rt, "title")
				status := genStatus().Draw(rt, "status")
				checkboxChar := statusToCheckboxChar(status)
				validTaskLine := fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title)

				// Create content with valid task followed by malformed line as description
				indentedMalformed := "  " + malformedLine
				content := validTaskLine + "\n" + indentedMalformed

				// Parse
				result1 := p.Parse(content)

				// Serialize
				serialized := pr.Serialize(result1.Document)

				// Parse again
				result2 := p.Parse(serialized)

				// Serialize again
				finalSerialized := pr.Serialize(result2.Document)

				// Verify the malformed line is preserved
				if !strings.Contains(finalSerialized, indentedMalformed) {
					t.Fatalf("Malformed line type %s not preserved.\n"+
						"Line: %q\n"+
						"Original:\n%s\n"+
						"Final:\n%s",
						tc.name, indentedMalformed, content, finalSerialized)
				}
			})
		})
	}
}

// TestProperty13_MalformedLinePreservation_InPreamble tests that malformed lines
// in the preamble (before any valid task) are preserved.
//
// **Validates: Requirements 7.4**
//
// Tag: Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation
func TestProperty13_MalformedLinePreservation_InPreamble(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate 1-3 malformed lines for preamble
		numMalformed := rapid.IntRange(1, 3).Draw(t, "numMalformed")
		var malformedLines []string
		for i := 0; i < numMalformed; i++ {
			malformedLine := genAnyMalformedLine().Draw(t, fmt.Sprintf("malformed_%d", i))
			malformedLines = append(malformedLines, malformedLine)
		}

		// Generate a valid task
		taskID := genTaskID().Draw(t, "taskID")
		title := genTaskTitle().Draw(t, "title")
		status := genStatus().Draw(t, "status")
		checkboxChar := statusToCheckboxChar(status)
		validTaskLine := fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title)

		// Create content with malformed lines in preamble
		var lines []string
		lines = append(lines, malformedLines...)
		lines = append(lines, validTaskLine)
		content := strings.Join(lines, "\n")

		// Parse → Serialize → Parse → Serialize
		result1 := p.Parse(content)
		serialized := pr.Serialize(result1.Document)
		result2 := p.Parse(serialized)
		finalSerialized := pr.Serialize(result2.Document)

		// Verify all malformed lines are preserved in preamble
		for _, malformedLine := range malformedLines {
			if !strings.Contains(finalSerialized, malformedLine) {
				t.Fatalf("Malformed preamble line not preserved: %q\n"+
					"Original:\n%s\n"+
					"Final:\n%s",
					malformedLine, content, finalSerialized)
			}
		}

		// Verify preamble count is preserved
		if len(result2.Document.Preamble) != len(result1.Document.Preamble) {
			t.Fatalf("Preamble count changed: %d -> %d",
				len(result1.Document.Preamble), len(result2.Document.Preamble))
		}
	})
}

// TestProperty13_MalformedLinePreservation_InDescription tests that malformed lines
// in task descriptions are preserved.
//
// **Validates: Requirements 7.4**
//
// Tag: Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation
func TestProperty13_MalformedLinePreservation_InDescription(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid task
		taskID := genTaskID().Draw(t, "taskID")
		title := genTaskTitle().Draw(t, "title")
		status := genStatus().Draw(t, "status")
		checkboxChar := statusToCheckboxChar(status)
		validTaskLine := fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title)

		// Generate 1-3 malformed lines for description (indented)
		numMalformed := rapid.IntRange(1, 3).Draw(t, "numMalformed")
		var malformedLines []string
		var lines []string
		lines = append(lines, validTaskLine)

		for i := 0; i < numMalformed; i++ {
			malformedLine := "  " + genAnyMalformedLine().Draw(t, fmt.Sprintf("malformed_%d", i))
			malformedLines = append(malformedLines, malformedLine)
			lines = append(lines, malformedLine)
		}

		content := strings.Join(lines, "\n")

		// Parse → Serialize → Parse → Serialize
		result1 := p.Parse(content)
		serialized := pr.Serialize(result1.Document)
		result2 := p.Parse(serialized)
		finalSerialized := pr.Serialize(result2.Document)

		// Verify all malformed lines are preserved in description
		for _, malformedLine := range malformedLines {
			if !strings.Contains(finalSerialized, malformedLine) {
				t.Fatalf("Malformed description line not preserved: %q\n"+
					"Original:\n%s\n"+
					"Final:\n%s",
					malformedLine, content, finalSerialized)
			}
		}

		// Verify description count is preserved
		if len(result1.Document.RootTasks) > 0 && len(result2.Document.RootTasks) > 0 {
			if len(result2.Document.RootTasks[0].Description) != len(result1.Document.RootTasks[0].Description) {
				t.Fatalf("Description count changed: %d -> %d",
					len(result1.Document.RootTasks[0].Description),
					len(result2.Document.RootTasks[0].Description))
			}
		}
	})
}

// TestProperty13_MalformedLinePreservation_InEpilogue tests that malformed lines
// in the epilogue (after all tasks) are preserved.
//
// **Validates: Requirements 7.4**
//
// Tag: Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation
func TestProperty13_MalformedLinePreservation_InEpilogue(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid task
		taskID := genTaskID().Draw(t, "taskID")
		title := genTaskTitle().Draw(t, "title")
		status := genStatus().Draw(t, "status")
		checkboxChar := statusToCheckboxChar(status)
		validTaskLine := fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title)

		// Generate 1-3 malformed lines for epilogue
		numMalformed := rapid.IntRange(1, 3).Draw(t, "numMalformed")
		var malformedLines []string
		var lines []string
		lines = append(lines, validTaskLine)

		for i := 0; i < numMalformed; i++ {
			malformedLine := genAnyMalformedLine().Draw(t, fmt.Sprintf("malformed_%d", i))
			malformedLines = append(malformedLines, malformedLine)
			lines = append(lines, malformedLine)
		}

		content := strings.Join(lines, "\n")

		// Parse → Serialize → Parse → Serialize
		result1 := p.Parse(content)
		serialized := pr.Serialize(result1.Document)
		result2 := p.Parse(serialized)
		finalSerialized := pr.Serialize(result2.Document)

		// Verify all malformed lines are preserved
		for _, malformedLine := range malformedLines {
			if !strings.Contains(finalSerialized, malformedLine) {
				t.Fatalf("Malformed epilogue line not preserved: %q\n"+
					"Original:\n%s\n"+
					"Final:\n%s",
					malformedLine, content, finalSerialized)
			}
		}
	})
}

// genGuaranteedMalformedLine generates a line that is guaranteed to NOT be parsed
// as a valid task. This excludes MalformedNoID which could accidentally generate
// a line that looks like a valid task if the title starts with a number.
func genGuaranteedMalformedLine() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Only use malformed types that are guaranteed to not be valid tasks
		// Exclude: MalformedNoID (could accidentally be valid if title starts with number)
		guaranteedMalformedTypes := []MalformedLineType{
			MalformedInvalidCheckboxChar,
			MalformedLeadingZeroID,
			MalformedNonNumericID,
			MalformedEmptySegmentID,
			MalformedMissingDash,
			MalformedMissingBrackets,
			MalformedWrongBracketType,
			NonTaskHeading,
			NonTaskPlainText,
			NonTaskEmptyLine,
			NonTaskBulletPoint,
		}
		lineType := rapid.SampledFrom(guaranteedMalformedTypes).Draw(t, "lineType")
		return genMalformedLine(lineType).Draw(t, "malformedLine")
	})
}

// TestProperty13_MalformedLinePreservation_ByteForByte tests that malformed lines
// are preserved byte-for-byte (exact content match).
//
// **Validates: Requirements 7.4**
//
// Tag: Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation
func TestProperty13_MalformedLinePreservation_ByteForByte(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a malformed line that is guaranteed to not be a valid task
		malformedLine := genGuaranteedMalformedLine().Draw(t, "malformedLine")

		// Skip empty lines for this test (they're tested elsewhere)
		if strings.TrimSpace(malformedLine) == "" {
			return
		}

		// Generate a valid task with a unique ID that won't conflict
		taskID := fmt.Sprintf("%d", rapid.IntRange(100, 999).Draw(t, "taskIDNum"))
		title := "ValidTask" + genTaskTitle().Draw(t, "title")
		status := genStatus().Draw(t, "status")
		checkboxChar := statusToCheckboxChar(status)
		validTaskLine := fmt.Sprintf("- [%s] %s %s", checkboxChar, taskID, title)

		// Create content with malformed line as description
		indentedMalformed := "  " + malformedLine
		content := validTaskLine + "\n" + indentedMalformed

		// Parse
		result := p.Parse(content)

		// Verify we have at least one task
		if len(result.Document.RootTasks) == 0 {
			t.Fatalf("Expected at least 1 task, got 0")
		}

		// The malformed line should be in the first task's description
		// (since it's indented under the valid task)
		task := result.Document.RootTasks[0]
		foundExactMatch := false
		for _, descLine := range task.Description {
			if descLine.RawContent == indentedMalformed {
				foundExactMatch = true
				break
			}
		}

		if !foundExactMatch {
			t.Fatalf("Malformed line not found byte-for-byte in description.\n"+
				"Expected: %q\n"+
				"Description lines: %v\n"+
				"Content:\n%s",
				indentedMalformed, task.Description, content)
		}

		// Serialize and verify byte-for-byte preservation
		serialized := pr.Serialize(result.Document)

		// Split serialized content into lines and find the malformed line
		serializedLines := strings.Split(serialized, "\n")
		foundInSerialized := false
		for _, line := range serializedLines {
			if line == indentedMalformed {
				foundInSerialized = true
				break
			}
		}

		if !foundInSerialized {
			t.Fatalf("Malformed line not preserved byte-for-byte in serialized output.\n"+
				"Expected: %q\n"+
				"Serialized:\n%s",
				indentedMalformed, serialized)
		}
	})
}

// TestProperty13_MalformedLinePreservation_MultipleRoundTrips tests that malformed
// lines are preserved through multiple parse-serialize cycles.
//
// **Validates: Requirements 7.4**
//
// Tag: Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation
func TestProperty13_MalformedLinePreservation_MultipleRoundTrips(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a document with malformed lines
		scenario := genMalformedDocumentScenario().Draw(t, "scenario")

		// Skip if no malformed lines generated
		if len(scenario.MalformedLines) == 0 {
			return
		}

		// Perform multiple round-trips (3 iterations)
		content := scenario.Content
		for i := 0; i < 3; i++ {
			result := p.Parse(content)
			content = pr.Serialize(result.Document)
		}

		// Verify all malformed lines are still preserved after multiple round-trips
		for _, malformedLine := range scenario.MalformedLines {
			if !strings.Contains(content, malformedLine) {
				t.Fatalf("Malformed line not preserved after 3 round-trips.\n"+
					"Line: %q\n"+
					"Original:\n%s\n"+
					"After 3 round-trips:\n%s",
					malformedLine, scenario.Content, content)
			}
		}
	})
}

// TestProperty13_MalformedLinePreservation_Deterministic tests that parsing and
// serializing malformed lines produces consistent results.
//
// **Validates: Requirements 7.4**
//
// Tag: Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation
func TestProperty13_MalformedLinePreservation_Deterministic(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a document with malformed lines
		scenario := genMalformedDocumentScenario().Draw(t, "scenario")

		// Skip if no malformed lines generated
		if len(scenario.MalformedLines) == 0 {
			return
		}

		// Parse and serialize multiple times
		result1 := p.Parse(scenario.Content)
		serialized1 := pr.Serialize(result1.Document)

		result2 := p.Parse(scenario.Content)
		serialized2 := pr.Serialize(result2.Document)

		result3 := p.Parse(scenario.Content)
		serialized3 := pr.Serialize(result3.Document)

		// All serializations should be identical
		if serialized1 != serialized2 || serialized2 != serialized3 {
			t.Fatalf("Non-deterministic serialization of malformed lines.\n"+
				"Original:\n%s\n"+
				"Serialized 1:\n%s\n"+
				"Serialized 2:\n%s\n"+
				"Serialized 3:\n%s",
				scenario.Content, serialized1, serialized2, serialized3)
		}
	})
}

// TestProperty13_MalformedLinePreservation_SpecificExamples tests specific examples
// of malformed lines from the design document.
//
// **Validates: Requirements 7.4**
//
// Tag: Feature: fuse-task-filesystem, Property 13: Malformed Line Preservation
func TestProperty13_MalformedLinePreservation_SpecificExamples(t *testing.T) {
	p := NewParser()
	pr := printer.NewPrinter()

	// Test cases from the design document and task description
	testCases := []struct {
		name         string
		malformedLine string
		description  string
	}{
		{
			name:         "no_id_checkbox",
			malformedLine: "- [ ] No number here",
			description:  "Checkbox syntax but NO numeric ID",
		},
		{
			name:         "invalid_checkbox_char",
			malformedLine: "- [?] 1.1 Task",
			description:  "Invalid checkbox character",
		},
		{
			name:         "leading_zero_id",
			malformedLine: "- [ ] 01.1 Task",
			description:  "Leading zero in ID",
		},
		{
			name:         "non_numeric_segment",
			malformedLine: "- [ ] 1.2.a Task",
			description:  "Non-numeric segment in ID",
		},
		{
			name:         "empty_segment",
			malformedLine: "- [ ] 1..2 Task",
			description:  "Empty segment in ID",
		},
		{
			name:         "missing_dash",
			malformedLine: "[ ] 1.1 Task",
			description:  "Missing dash",
		},
		{
			name:         "missing_brackets",
			malformedLine: "- 1.1 Task",
			description:  "Missing brackets",
		},
		{
			name:         "wrong_bracket_type",
			malformedLine: "- (x) 1.1 Task",
			description:  "Wrong bracket type",
		},
		{
			name:         "heading",
			malformedLine: "# Heading",
			description:  "Regular heading",
		},
		{
			name:         "plain_text",
			malformedLine: "Some text",
			description:  "Plain text",
		},
		{
			name:         "empty_line",
			malformedLine: "",
			description:  "Empty line",
		},
		{
			name:         "bullet_point",
			malformedLine: "* Item",
			description:  "Bullet point without checkbox",
		},
	}

	for _, tc := range testCases {
		tc := tc // capture for closure
		t.Run(tc.name, func(t *testing.T) {
			// Create content with a valid task and the malformed line as description
			validTaskLine := "- [ ] 1 Valid task"
			indentedMalformed := "  " + tc.malformedLine
			content := validTaskLine + "\n" + indentedMalformed

			// Parse → Serialize → Parse → Serialize
			result1 := p.Parse(content)
			serialized := pr.Serialize(result1.Document)
			result2 := p.Parse(serialized)
			finalSerialized := pr.Serialize(result2.Document)

			// Verify the malformed line is preserved
			if !strings.Contains(finalSerialized, indentedMalformed) {
				t.Fatalf("Malformed line not preserved: %s\n"+
					"Line: %q\n"+
					"Original:\n%s\n"+
					"Final:\n%s",
					tc.description, indentedMalformed, content, finalSerialized)
			}
		})
	}
}
