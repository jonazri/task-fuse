// Package store provides property-based tests for filename generation.
//
// Feature: fuse-task-filesystem
// Property 5: Filename Generation Consistency
//
// These tests use the rapid library for property-based testing to verify that
// filename generation follows the specified rules and is deterministic.
package store

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Property 5: Filename Generation Consistency
// =============================================================================
//
// Property Statement:
// *For any* task, the generated filename SHALL follow the pattern
// `{id}.{slugified_title}.md` (all valid tasks have numeric IDs per Requirement 1.6).
// The same task SHALL always generate the same filename.
//
// **Validates: Requirements 2.10, 2.11**
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency

// =============================================================================
// Generators
// =============================================================================

// genValidTaskID generates a valid task ID (e.g., "1", "1.2", "1.2.3").
// Task IDs are dot-separated positive integers with no leading zeros.
// Maximum depth is 10 levels per the specification.
func genValidTaskID() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate 1-10 levels deep (max depth per spec)
		depth := rapid.IntRange(1, 10).Draw(t, "depth")
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			// Each segment is a positive integer (1-999)
			// No leading zeros allowed
			num := rapid.IntRange(1, 999).Draw(t, fmt.Sprintf("segment_%d", i))
			segments[i] = fmt.Sprintf("%d", num)
		}
		return strings.Join(segments, ".")
	})
}

// genTaskTitle generates various task titles for testing.
// This includes ASCII, non-ASCII, special characters, empty, and long titles.
func genTaskTitle() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		titleType := rapid.IntRange(0, 7).Draw(t, "titleType")
		switch titleType {
		case 0:
			// Simple ASCII title with words
			wordCount := rapid.IntRange(1, 10).Draw(t, "wordCount")
			words := make([]string, wordCount)
			for i := 0; i < wordCount; i++ {
				word := rapid.StringMatching(`[a-zA-Z]+`).Draw(t, fmt.Sprintf("word_%d", i))
				if len(word) > 15 {
					word = word[:15]
				}
				if len(word) == 0 {
					word = "task"
				}
				words[i] = word
			}
			return strings.Join(words, " ")
		case 1:
			// Title with numbers
			return fmt.Sprintf("Task %d implementation", rapid.IntRange(1, 1000).Draw(t, "num"))
		case 2:
			// Title with special characters
			base := rapid.StringMatching(`[a-zA-Z]+`).Draw(t, "base")
			if len(base) > 10 {
				base = base[:10]
			}
			if len(base) == 0 {
				base = "task"
			}
			specials := []string{"!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "[", "]"}
			special := specials[rapid.IntRange(0, len(specials)-1).Draw(t, "specialIdx")]
			return fmt.Sprintf("%s %s test", base, special)
		case 3:
			// Title with non-ASCII characters (accented)
			accented := []string{"café", "résumé", "naïve", "über", "señor", "Zürich"}
			return accented[rapid.IntRange(0, len(accented)-1).Draw(t, "accentedIdx")]
		case 4:
			// Empty title
			return ""
		case 5:
			// Very long title (exceeds 50 char slug limit)
			wordCount := rapid.IntRange(15, 25).Draw(t, "longWordCount")
			words := make([]string, wordCount)
			for i := 0; i < wordCount; i++ {
				word := rapid.StringMatching(`[a-zA-Z]+`).Draw(t, fmt.Sprintf("longWord_%d", i))
				if len(word) > 10 {
					word = word[:10]
				}
				if len(word) == 0 {
					word = "word"
				}
				words[i] = word
			}
			return strings.Join(words, " ")
		case 6:
			// Title with hyphens and underscores (allowed chars)
			return fmt.Sprintf("my-task_name-%d", rapid.IntRange(1, 100).Draw(t, "hyphenNum"))
		case 7:
			// Title with mixed whitespace
			base := rapid.StringMatching(`[a-zA-Z]+`).Draw(t, "wsBase")
			if len(base) > 10 {
				base = base[:10]
			}
			if len(base) == 0 {
				base = "task"
			}
			return fmt.Sprintf("  %s   test  ", base)
		default:
			return "default task"
		}
	})
}

// genTask generates a complete Task object with valid ID and various titles.
func genTask() *rapid.Generator[*Task] {
	return rapid.Custom(func(t *rapid.T) *Task {
		id := genValidTaskID().Draw(t, "taskID")
		title := genTaskTitle().Draw(t, "taskTitle")
		status := []TaskStatus{StatusPending, StatusQueued, StatusDoing, StatusDone, StatusFailed}
		statusIdx := rapid.IntRange(0, len(status)-1).Draw(t, "statusIdx")

		return &Task{
			ID:     id,
			Title:  title,
			Status: status[statusIdx],
		}
	})
}

// =============================================================================
// Property Tests
// =============================================================================

// validSlugCharsRegex matches only valid slug characters [a-z0-9_-]
var validSlugCharsRegex = regexp.MustCompile(`^[a-z0-9_-]*$`)

// filenamePatternRegex matches the expected filename pattern {id}.{slug}.md
// where id is dot-separated positive integers and slug is [a-z0-9_-]+
var filenamePatternRegex = regexp.MustCompile(`^[1-9][0-9]*(\.[1-9][0-9]*)*\.[a-z0-9_-]+\.md$`)

// TestProperty5_FilenameGenerationConsistency is the main property-based test for
// Property 5: Filename Generation Consistency.
//
// **Validates: Requirements 2.10, 2.11**
//
// This test generates random valid tasks and verifies that:
// 1. The filename follows the pattern {id}.{slug}.md
// 2. The same task always generates the same filename (deterministic)
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
func TestProperty5_FilenameGenerationConsistency(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
	// Validates: Requirements 2.10, 2.11

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random task
		task := genTask().Draw(t, "task")

		// Generate filename
		filename := GenerateFilename(task)

		// Property 1: Filename should not be empty for valid tasks
		if filename == "" {
			t.Fatalf("GenerateFilename returned empty string for task with ID=%q, Title=%q",
				task.ID, task.Title)
		}

		// Property 2: Filename should follow the pattern {id}.{slug}.md
		if !filenamePatternRegex.MatchString(filename) {
			t.Fatalf("GenerateFilename(%q, %q) = %q does not match pattern {id}.{slug}.md",
				task.ID, task.Title, filename)
		}

		// Property 3: Filename should start with the task ID
		if !strings.HasPrefix(filename, task.ID+".") {
			t.Fatalf("GenerateFilename(%q, %q) = %q does not start with task ID",
				task.ID, task.Title, filename)
		}

		// Property 4: Filename should end with .md
		if !strings.HasSuffix(filename, ".md") {
			t.Fatalf("GenerateFilename(%q, %q) = %q does not end with .md",
				task.ID, task.Title, filename)
		}

		// Property 5: Same task should always generate the same filename (deterministic)
		filename2 := GenerateFilename(task)
		if filename != filename2 {
			t.Fatalf("GenerateFilename is not deterministic: %q != %q for task ID=%q, Title=%q",
				filename, filename2, task.ID, task.Title)
		}
	})
}

// TestProperty5_FilenameGenerationConsistency_PatternFormat verifies the filename format:
// - Filename starts with task ID
// - Filename ends with .md
// - Slug portion only contains [a-z0-9_-]
//
// **Validates: Requirements 2.10, 2.11**
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
func TestProperty5_FilenameGenerationConsistency_PatternFormat(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
	// Validates: Requirements 2.10, 2.11

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random task
		task := genTask().Draw(t, "task")

		// Generate filename
		filename := GenerateFilename(task)

		// Property 1: Filename should start with task ID followed by a dot
		expectedPrefix := task.ID + "."
		if !strings.HasPrefix(filename, expectedPrefix) {
			t.Fatalf("Filename %q should start with %q", filename, expectedPrefix)
		}

		// Property 2: Filename should end with .md
		if !strings.HasSuffix(filename, ".md") {
			t.Fatalf("Filename %q should end with .md", filename)
		}

		// Property 3: Extract slug portion and verify it only contains valid characters
		// Filename format: {id}.{slug}.md
		// Remove the ID prefix and .md suffix to get the slug
		withoutID := strings.TrimPrefix(filename, expectedPrefix)
		slug := strings.TrimSuffix(withoutID, ".md")

		// Slug should only contain [a-z0-9_-]
		if !validSlugCharsRegex.MatchString(slug) {
			t.Fatalf("Slug %q contains invalid characters (only [a-z0-9_-] allowed)", slug)
		}

		// Property 4: Slug should not be empty (empty titles become "task")
		if slug == "" {
			t.Fatalf("Slug should not be empty for task ID=%q, Title=%q", task.ID, task.Title)
		}

		// Property 5: Slug should not have leading or trailing underscores
		if strings.HasPrefix(slug, "_") || strings.HasSuffix(slug, "_") {
			t.Fatalf("Slug %q should not have leading or trailing underscores", slug)
		}

		// Property 6: Slug should not have consecutive underscores
		if strings.Contains(slug, "__") {
			t.Fatalf("Slug %q should not have consecutive underscores", slug)
		}
	})
}

// TestProperty5_FilenameGenerationConsistency_MaxLength verifies length constraints:
// - Slug portion never exceeds 50 characters
// - Total filename length is reasonable
//
// **Validates: Requirements 2.10, 2.11**
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
func TestProperty5_FilenameGenerationConsistency_MaxLength(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
	// Validates: Requirements 2.10, 2.11

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random task
		task := genTask().Draw(t, "task")

		// Generate filename
		filename := GenerateFilename(task)

		// Extract slug portion
		// Filename format: {id}.{slug}.md
		expectedPrefix := task.ID + "."
		withoutID := strings.TrimPrefix(filename, expectedPrefix)
		slug := strings.TrimSuffix(withoutID, ".md")

		// Property 1: Slug should not exceed MaxSlugLength (50 characters)
		if len(slug) > MaxSlugLength {
			t.Fatalf("Slug %q has length %d, exceeds MaxSlugLength %d",
				slug, len(slug), MaxSlugLength)
		}

		// Property 2: Total filename length should be reasonable
		// Max ID length: 10 levels * 4 chars (999.) = 40 chars + slug (50) + .md (3) = 93 chars
		// But realistically, IDs are shorter, so we use a generous upper bound
		maxReasonableLength := 150
		if len(filename) > maxReasonableLength {
			t.Fatalf("Filename %q has length %d, exceeds reasonable maximum %d",
				filename, len(filename), maxReasonableLength)
		}

		// Property 3: Filename should have minimum length (at least "1.task.md" = 9 chars)
		minLength := len(task.ID) + 1 + 1 + 3 // id + "." + at least 1 char slug + ".md"
		if len(filename) < minLength {
			t.Fatalf("Filename %q has length %d, less than minimum expected %d",
				filename, len(filename), minLength)
		}
	})
}

// TestProperty5_FilenameGenerationConsistency_Deterministic verifies determinism:
// - Same task generates same filename on multiple calls
// - Order of calls doesn't affect result
//
// **Validates: Requirements 2.10, 2.11**
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
func TestProperty5_FilenameGenerationConsistency_Deterministic(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
	// Validates: Requirements 2.10, 2.11

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random task
		task := genTask().Draw(t, "task")

		// Generate filename multiple times
		filename1 := GenerateFilename(task)
		filename2 := GenerateFilename(task)
		filename3 := GenerateFilename(task)

		// Property 1: All calls should return the same result
		if filename1 != filename2 {
			t.Fatalf("GenerateFilename is not deterministic: call 1 returned %q, call 2 returned %q",
				filename1, filename2)
		}
		if filename2 != filename3 {
			t.Fatalf("GenerateFilename is not deterministic: call 2 returned %q, call 3 returned %q",
				filename2, filename3)
		}

		// Property 2: Creating a new task with same ID and Title should generate same filename
		task2 := &Task{
			ID:     task.ID,
			Title:  task.Title,
			Status: task.Status,
		}
		filename4 := GenerateFilename(task2)
		if filename1 != filename4 {
			t.Fatalf("GenerateFilename is not deterministic for equivalent tasks: %q != %q",
				filename1, filename4)
		}

		// Property 3: Status should not affect filename (only ID and Title matter)
		allStatuses := []TaskStatus{StatusPending, StatusQueued, StatusDoing, StatusDone, StatusFailed}
		for _, status := range allStatuses {
			taskWithStatus := &Task{
				ID:     task.ID,
				Title:  task.Title,
				Status: status,
			}
			filenameWithStatus := GenerateFilename(taskWithStatus)
			if filename1 != filenameWithStatus {
				t.Fatalf("Filename should not depend on status: %q (status=%s) != %q (status=%s)",
					filename1, task.Status, filenameWithStatus, status)
			}
		}
	})
}

// TestProperty5_FilenameGenerationConsistency_SlugifyRules verifies that the
// slugification rules are correctly applied:
// 1. Convert to lowercase
// 2. Replace whitespace with underscores
// 3. Remove invalid characters
// 4. Collapse multiple underscores
// 5. Trim leading/trailing underscores
// 6. Truncate to 50 chars
// 7. Use "task" if empty
//
// **Validates: Requirements 2.10, 2.11**
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
func TestProperty5_FilenameGenerationConsistency_SlugifyRules(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
	// Validates: Requirements 2.10, 2.11

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random title
		title := genTaskTitle().Draw(t, "title")

		// Slugify the title
		slug := Slugify(title)

		// Property 1: Slug should only contain lowercase letters, numbers, underscores, hyphens
		if !validSlugCharsRegex.MatchString(slug) {
			t.Fatalf("Slugify(%q) = %q contains invalid characters", title, slug)
		}

		// Property 2: Slug should not exceed MaxSlugLength
		if len(slug) > MaxSlugLength {
			t.Fatalf("Slugify(%q) = %q exceeds MaxSlugLength %d", title, slug, MaxSlugLength)
		}

		// Property 3: Slug should not be empty (empty input becomes "task")
		if slug == "" {
			t.Fatalf("Slugify(%q) returned empty string, expected at least 'task'", title)
		}

		// Property 4: Slug should not have leading or trailing underscores
		if strings.HasPrefix(slug, "_") {
			t.Fatalf("Slugify(%q) = %q has leading underscore", title, slug)
		}
		if strings.HasSuffix(slug, "_") {
			t.Fatalf("Slugify(%q) = %q has trailing underscore", title, slug)
		}

		// Property 5: Slug should not have consecutive underscores
		if strings.Contains(slug, "__") {
			t.Fatalf("Slugify(%q) = %q has consecutive underscores", title, slug)
		}

		// Property 6: Slug should be lowercase
		if slug != strings.ToLower(slug) {
			t.Fatalf("Slugify(%q) = %q is not lowercase", title, slug)
		}

		// Property 7: Slugify should be deterministic
		slug2 := Slugify(title)
		if slug != slug2 {
			t.Fatalf("Slugify is not deterministic: %q != %q for input %q", slug, slug2, title)
		}
	})
}

// TestProperty5_FilenameGenerationConsistency_EmptyTitle verifies that empty
// titles result in the slug "task".
//
// **Validates: Requirements 2.10, 2.11**
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
func TestProperty5_FilenameGenerationConsistency_EmptyTitle(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
	// Validates: Requirements 2.10, 2.11

	rapid.Check(t, func(t *rapid.T) {
		// Generate a random task ID
		taskID := genValidTaskID().Draw(t, "taskID")

		// Create task with empty title
		task := &Task{
			ID:     taskID,
			Title:  "",
			Status: StatusPending,
		}

		// Generate filename
		filename := GenerateFilename(task)

		// Property: Filename should use "task" as the slug for empty titles
		expectedFilename := taskID + ".task.md"
		if filename != expectedFilename {
			t.Fatalf("GenerateFilename for empty title: got %q, expected %q", filename, expectedFilename)
		}
	})
}

// TestProperty5_FilenameGenerationConsistency_SpecialCharsOnly verifies that
// titles with only special characters result in the slug "task".
//
// **Validates: Requirements 2.10, 2.11**
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
func TestProperty5_FilenameGenerationConsistency_SpecialCharsOnly(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
	// Validates: Requirements 2.10, 2.11

	specialOnlyTitles := []string{
		"@#$%^&*()",
		"!!!",
		"...",
		"   ",
		"\t\n",
		"[]{}",
		"<>",
	}

	for _, title := range specialOnlyTitles {
		t.Run(fmt.Sprintf("title_%q", title), func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				taskID := genValidTaskID().Draw(t, "taskID")

				task := &Task{
					ID:     taskID,
					Title:  title,
					Status: StatusPending,
				}

				filename := GenerateFilename(task)
				expectedFilename := taskID + ".task.md"

				if filename != expectedFilename {
					t.Fatalf("GenerateFilename for special-chars-only title %q: got %q, expected %q",
						title, filename, expectedFilename)
				}
			})
		})
	}
}

// TestProperty5_FilenameGenerationConsistency_LongTitles verifies that long
// titles are properly truncated to ensure slug doesn't exceed 50 characters.
//
// **Validates: Requirements 2.10, 2.11**
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
func TestProperty5_FilenameGenerationConsistency_LongTitles(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
	// Validates: Requirements 2.10, 2.11

	rapid.Check(t, func(t *rapid.T) {
		// Generate a very long title (guaranteed to exceed 50 chars when slugified)
		wordCount := rapid.IntRange(20, 30).Draw(t, "wordCount")
		words := make([]string, wordCount)
		for i := 0; i < wordCount; i++ {
			word := rapid.StringMatching(`[a-zA-Z]+`).Draw(t, fmt.Sprintf("word_%d", i))
			if len(word) > 8 {
				word = word[:8]
			}
			if len(word) == 0 {
				word = "word"
			}
			words[i] = word
		}
		longTitle := strings.Join(words, " ")

		taskID := genValidTaskID().Draw(t, "taskID")
		task := &Task{
			ID:     taskID,
			Title:  longTitle,
			Status: StatusPending,
		}

		filename := GenerateFilename(task)

		// Extract slug
		expectedPrefix := taskID + "."
		withoutID := strings.TrimPrefix(filename, expectedPrefix)
		slug := strings.TrimSuffix(withoutID, ".md")

		// Property: Slug should not exceed MaxSlugLength even for very long titles
		if len(slug) > MaxSlugLength {
			t.Fatalf("Slug %q from long title has length %d, exceeds MaxSlugLength %d",
				slug, len(slug), MaxSlugLength)
		}

		// Property: Slug should still be valid
		if !validSlugCharsRegex.MatchString(slug) {
			t.Fatalf("Slug %q from long title contains invalid characters", slug)
		}
	})
}

// TestProperty5_FilenameGenerationConsistency_NilTask verifies that nil task
// returns empty string.
//
// **Validates: Requirements 2.10, 2.11**
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
func TestProperty5_FilenameGenerationConsistency_NilTask(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
	// Validates: Requirements 2.10, 2.11

	filename := GenerateFilename(nil)
	if filename != "" {
		t.Fatalf("GenerateFilename(nil) should return empty string, got %q", filename)
	}
}

// TestProperty5_FilenameGenerationConsistency_IDVariations verifies that
// filename generation works correctly with various valid task ID formats.
//
// **Validates: Requirements 2.10, 2.11**
//
// Tag: Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
func TestProperty5_FilenameGenerationConsistency_IDVariations(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 5: Filename Generation Consistency
	// Validates: Requirements 2.10, 2.11

	rapid.Check(t, func(t *rapid.T) {
		// Generate various ID depths
		depth := rapid.IntRange(1, 10).Draw(t, "depth")
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			// Use various segment sizes
			num := rapid.IntRange(1, 999).Draw(t, fmt.Sprintf("segment_%d", i))
			segments[i] = fmt.Sprintf("%d", num)
		}
		taskID := strings.Join(segments, ".")

		title := genTaskTitle().Draw(t, "title")
		task := &Task{
			ID:     taskID,
			Title:  title,
			Status: StatusPending,
		}

		filename := GenerateFilename(task)

		// Property 1: Filename should start with the exact task ID
		if !strings.HasPrefix(filename, taskID+".") {
			t.Fatalf("Filename %q should start with task ID %q", filename, taskID)
		}

		// Property 2: Filename should match the expected pattern
		if !filenamePatternRegex.MatchString(filename) {
			t.Fatalf("Filename %q does not match expected pattern for ID %q", filename, taskID)
		}

		// Property 3: ID should be preserved exactly (no modification)
		parts := strings.SplitN(filename, ".", depth+2) // ID segments + slug + md
		reconstructedID := strings.Join(parts[:depth], ".")
		if reconstructedID != taskID {
			t.Fatalf("Task ID not preserved: expected %q, got %q in filename %q",
				taskID, reconstructedID, filename)
		}
	})
}


// =============================================================================
// Property 14: Filename Collision Resolution
// =============================================================================
//
// Property Statement:
// *For any* set of tasks that would generate the same filename, the FUSE_Filesystem
// SHALL append unique numeric suffixes (`-2`, `-3`, etc.) to ensure all filenames
// are unique within their directory.
//
// **Validates: Requirements 7.6**
//
// Tag: Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution

// =============================================================================
// Generators for Property 14
// =============================================================================

// genFilename generates a valid filename in format `{id}.{slug}.md`.
// This mimics the output of GenerateFilename for testing collision resolution.
func genFilename() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate a task ID (1-3 levels deep for simplicity)
		depth := rapid.IntRange(1, 3).Draw(t, "depth")
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			num := rapid.IntRange(1, 99).Draw(t, fmt.Sprintf("segment_%d", i))
			segments[i] = fmt.Sprintf("%d", num)
		}
		taskID := strings.Join(segments, ".")

		// Generate a slug (lowercase letters, numbers, underscores)
		slugType := rapid.IntRange(0, 3).Draw(t, "slugType")
		var slug string
		switch slugType {
		case 0:
			// Simple word slug
			slug = rapid.StringMatching(`[a-z]+`).Draw(t, "simpleSlug")
			if len(slug) > 20 {
				slug = slug[:20]
			}
			if len(slug) == 0 {
				slug = "task"
			}
		case 1:
			// Multi-word slug with underscores
			wordCount := rapid.IntRange(1, 4).Draw(t, "wordCount")
			words := make([]string, wordCount)
			for i := 0; i < wordCount; i++ {
				word := rapid.StringMatching(`[a-z]+`).Draw(t, fmt.Sprintf("word_%d", i))
				if len(word) > 10 {
					word = word[:10]
				}
				if len(word) == 0 {
					word = "word"
				}
				words[i] = word
			}
			slug = strings.Join(words, "_")
		case 2:
			// Slug with numbers
			base := rapid.StringMatching(`[a-z]+`).Draw(t, "numBase")
			if len(base) > 10 {
				base = base[:10]
			}
			if len(base) == 0 {
				base = "task"
			}
			num := rapid.IntRange(1, 100).Draw(t, "slugNum")
			slug = fmt.Sprintf("%s_%d", base, num)
		case 3:
			// Default slug
			slug = "task"
		}

		return taskID + "." + slug + ".md"
	})
}

// genExistingFilenames generates a set of existing filenames with potential collisions.
// It takes a base filename and generates a set that may or may not contain it.
func genExistingFilenames(baseFilename string) *rapid.Generator[map[string]bool] {
	return rapid.Custom(func(t *rapid.T) map[string]bool {
		existing := make(map[string]bool)

		// Decide how many existing filenames to generate
		count := rapid.IntRange(0, 10).Draw(t, "existingCount")

		for i := 0; i < count; i++ {
			filenameType := rapid.IntRange(0, 4).Draw(t, fmt.Sprintf("filenameType_%d", i))
			switch filenameType {
			case 0:
				// Add the base filename (creates collision)
				existing[baseFilename] = true
			case 1:
				// Add a collision variant (-2, -3, etc.)
				suffix := rapid.IntRange(2, 5).Draw(t, fmt.Sprintf("suffix_%d", i))
				base, ext := splitTestFilename(baseFilename)
				existing[fmt.Sprintf("%s-%d%s", base, suffix, ext)] = true
			case 2:
				// Add a completely different filename
				different := genFilename().Draw(t, fmt.Sprintf("different_%d", i))
				existing[different] = true
			case 3:
				// Add a similar filename (same ID, different slug)
				parts := strings.SplitN(baseFilename, ".", 2)
				if len(parts) >= 1 {
					newSlug := rapid.StringMatching(`[a-z]+`).Draw(t, fmt.Sprintf("newSlug_%d", i))
					if len(newSlug) > 10 {
						newSlug = newSlug[:10]
					}
					if len(newSlug) == 0 {
						newSlug = "other"
					}
					existing[parts[0]+"."+newSlug+".md"] = true
				}
			case 4:
				// Add nothing (sparse set)
			}
		}

		return existing
	})
}

// splitTestFilename splits a filename into base and extension for testing.
func splitTestFilename(filename string) (base, ext string) {
	if strings.HasSuffix(filename, ".md") {
		return filename[:len(filename)-3], ".md"
	}
	return filename, ""
}

// =============================================================================
// Property Tests for Property 14
// =============================================================================

// TestProperty14_FilenameCollisionResolution is the main property-based test for
// Property 14: Filename Collision Resolution.
//
// **Validates: Requirements 7.6**
//
// This test generates random filenames and existing filename sets, then verifies
// that ResolveCollision always produces a unique filename.
//
// Tag: Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
func TestProperty14_FilenameCollisionResolution(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
	// Validates: Requirements 7.6

	rapid.Check(t, func(t *rapid.T) {
		// Generate a base filename
		baseFilename := genFilename().Draw(t, "baseFilename")

		// Generate a set of existing filenames (may or may not contain baseFilename)
		existingFilenames := genExistingFilenames(baseFilename).Draw(t, "existingFilenames")

		// Call ResolveCollision
		result := ResolveCollision(baseFilename, existingFilenames)

		// Property 1: Result should not be empty
		if result == "" {
			t.Fatalf("ResolveCollision returned empty string for filename %q", baseFilename)
		}

		// Property 2: Result should be unique (not in existing set)
		if existingFilenames[result] {
			t.Fatalf("ResolveCollision(%q) = %q is not unique (exists in set)",
				baseFilename, result)
		}

		// Property 3: Result should end with .md
		if !strings.HasSuffix(result, ".md") {
			t.Fatalf("ResolveCollision(%q) = %q does not end with .md",
				baseFilename, result)
		}

		// Property 4: If no collision existed, result should equal input
		if !existingFilenames[baseFilename] {
			if result != baseFilename {
				t.Fatalf("ResolveCollision(%q) = %q but no collision existed, expected unchanged",
					baseFilename, result)
			}
		}
	})
}

// TestProperty14_FilenameCollisionResolution_SuffixFormat verifies the suffix format:
// - First collision gets `-2` suffix
// - Subsequent collisions get `-3`, `-4`, etc.
// - Suffix is inserted before `.md` extension
//
// **Validates: Requirements 7.6**
//
// Tag: Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
func TestProperty14_FilenameCollisionResolution_SuffixFormat(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
	// Validates: Requirements 7.6

	rapid.Check(t, func(t *rapid.T) {
		// Generate a base filename
		baseFilename := genFilename().Draw(t, "baseFilename")

		// Create an existing set with the base filename (guaranteed collision)
		existingFilenames := map[string]bool{
			baseFilename: true,
		}

		// Call ResolveCollision
		result := ResolveCollision(baseFilename, existingFilenames)

		// Property 1: First collision should get -2 suffix
		base, ext := splitTestFilename(baseFilename)
		expectedFirst := base + "-2" + ext
		if result != expectedFirst {
			t.Fatalf("First collision: ResolveCollision(%q) = %q, expected %q",
				baseFilename, result, expectedFirst)
		}

		// Property 2: Add -2 to existing, next should be -3
		existingFilenames[expectedFirst] = true
		result2 := ResolveCollision(baseFilename, existingFilenames)
		expectedSecond := base + "-3" + ext
		if result2 != expectedSecond {
			t.Fatalf("Second collision: ResolveCollision(%q) = %q, expected %q",
				baseFilename, result2, expectedSecond)
		}

		// Property 3: Add -3 to existing, next should be -4
		existingFilenames[expectedSecond] = true
		result3 := ResolveCollision(baseFilename, existingFilenames)
		expectedThird := base + "-4" + ext
		if result3 != expectedThird {
			t.Fatalf("Third collision: ResolveCollision(%q) = %q, expected %q",
				baseFilename, result3, expectedThird)
		}

		// Property 4: Suffix should be inserted before .md extension
		if !strings.HasSuffix(result, ".md") {
			t.Fatalf("Suffix not inserted before .md: %q", result)
		}
		if !strings.Contains(result, "-2.md") {
			t.Fatalf("Suffix format incorrect: %q should contain '-2.md'", result)
		}
	})
}

// TestProperty14_FilenameCollisionResolution_Uniqueness verifies uniqueness:
// - Result is always unique in the existing set
// - Multiple calls with same input produce same result (deterministic)
//
// **Validates: Requirements 7.6**
//
// Tag: Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
func TestProperty14_FilenameCollisionResolution_Uniqueness(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
	// Validates: Requirements 7.6

	rapid.Check(t, func(t *rapid.T) {
		// Generate a base filename
		baseFilename := genFilename().Draw(t, "baseFilename")

		// Generate a set of existing filenames with potential collisions
		existingFilenames := genExistingFilenames(baseFilename).Draw(t, "existingFilenames")

		// Call ResolveCollision multiple times
		result1 := ResolveCollision(baseFilename, existingFilenames)
		result2 := ResolveCollision(baseFilename, existingFilenames)
		result3 := ResolveCollision(baseFilename, existingFilenames)

		// Property 1: Result should always be unique (not in existing set)
		if existingFilenames[result1] {
			t.Fatalf("ResolveCollision result %q is not unique", result1)
		}

		// Property 2: Multiple calls should produce the same result (deterministic)
		if result1 != result2 {
			t.Fatalf("ResolveCollision is not deterministic: %q != %q", result1, result2)
		}
		if result2 != result3 {
			t.Fatalf("ResolveCollision is not deterministic: %q != %q", result2, result3)
		}

		// Property 3: If we add the result to existing and call again, we get a different result
		existingFilenames[result1] = true
		result4 := ResolveCollision(baseFilename, existingFilenames)
		if result4 == result1 {
			t.Fatalf("After adding result to existing, ResolveCollision should return different value: got %q again", result4)
		}
		if existingFilenames[result4] {
			t.Fatalf("New result %q is not unique after adding previous result", result4)
		}
	})
}

// TestProperty14_FilenameCollisionResolution_NoCollision verifies the no-collision case:
// - When no collision exists, filename is returned unchanged
//
// **Validates: Requirements 7.6**
//
// Tag: Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
func TestProperty14_FilenameCollisionResolution_NoCollision(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
	// Validates: Requirements 7.6

	rapid.Check(t, func(t *rapid.T) {
		// Generate a base filename
		baseFilename := genFilename().Draw(t, "baseFilename")

		// Generate a set of existing filenames that does NOT contain the base filename
		existingFilenames := make(map[string]bool)
		count := rapid.IntRange(0, 10).Draw(t, "existingCount")
		for i := 0; i < count; i++ {
			// Generate different filenames
			different := genFilename().Draw(t, fmt.Sprintf("different_%d", i))
			// Make sure it's not the same as baseFilename
			if different != baseFilename {
				existingFilenames[different] = true
			}
		}

		// Ensure baseFilename is not in the set
		delete(existingFilenames, baseFilename)

		// Call ResolveCollision
		result := ResolveCollision(baseFilename, existingFilenames)

		// Property: When no collision exists, filename should be returned unchanged
		if result != baseFilename {
			t.Fatalf("No collision case: ResolveCollision(%q) = %q, expected unchanged",
				baseFilename, result)
		}
	})
}

// TestProperty14_FilenameCollisionResolution_ManyCollisions verifies handling of many collisions:
// - Even with many existing collisions (-2, -3, ..., -N), a unique suffix is found
//
// **Validates: Requirements 7.6**
//
// Tag: Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
func TestProperty14_FilenameCollisionResolution_ManyCollisions(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
	// Validates: Requirements 7.6

	rapid.Check(t, func(t *rapid.T) {
		// Generate a base filename
		baseFilename := genFilename().Draw(t, "baseFilename")
		base, ext := splitTestFilename(baseFilename)

		// Create an existing set with many collision variants
		existingFilenames := map[string]bool{
			baseFilename: true,
		}

		// Add collision variants -2 through -N (random N between 2 and 20)
		maxSuffix := rapid.IntRange(2, 20).Draw(t, "maxSuffix")
		for i := 2; i <= maxSuffix; i++ {
			existingFilenames[fmt.Sprintf("%s-%d%s", base, i, ext)] = true
		}

		// Call ResolveCollision
		result := ResolveCollision(baseFilename, existingFilenames)

		// Property 1: Result should be unique
		if existingFilenames[result] {
			t.Fatalf("ResolveCollision(%q) = %q is not unique with %d existing collisions",
				baseFilename, result, maxSuffix)
		}

		// Property 2: Result should have the next available suffix
		expectedSuffix := maxSuffix + 1
		expectedResult := fmt.Sprintf("%s-%d%s", base, expectedSuffix, ext)
		if result != expectedResult {
			t.Fatalf("ResolveCollision(%q) = %q, expected %q (next available suffix)",
				baseFilename, result, expectedResult)
		}

		// Property 3: Result should still end with .md
		if !strings.HasSuffix(result, ".md") {
			t.Fatalf("Result %q does not end with .md", result)
		}
	})
}

// TestProperty14_FilenameCollisionResolution_GapsInSuffixes verifies handling of gaps:
// - If -2 exists but -3 doesn't, -3 should be used (not -4)
//
// **Validates: Requirements 7.6**
//
// Tag: Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
func TestProperty14_FilenameCollisionResolution_GapsInSuffixes(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
	// Validates: Requirements 7.6

	rapid.Check(t, func(t *rapid.T) {
		// Generate a base filename
		baseFilename := genFilename().Draw(t, "baseFilename")
		base, ext := splitTestFilename(baseFilename)

		// Create an existing set with gaps in suffixes
		// e.g., base, -2, -4, -5 exist but -3 doesn't
		existingFilenames := map[string]bool{
			baseFilename: true,
		}

		// Generate a random set of suffixes with gaps
		numSuffixes := rapid.IntRange(1, 5).Draw(t, "numSuffixes")
		usedSuffixes := make(map[int]bool)
		for i := 0; i < numSuffixes; i++ {
			suffix := rapid.IntRange(2, 10).Draw(t, fmt.Sprintf("suffix_%d", i))
			usedSuffixes[suffix] = true
			existingFilenames[fmt.Sprintf("%s-%d%s", base, suffix, ext)] = true
		}

		// Call ResolveCollision
		result := ResolveCollision(baseFilename, existingFilenames)

		// Property 1: Result should be unique
		if existingFilenames[result] {
			t.Fatalf("ResolveCollision(%q) = %q is not unique", baseFilename, result)
		}

		// Property 2: Result should use the first available suffix starting from 2
		// Find the first gap
		expectedSuffix := 2
		for usedSuffixes[expectedSuffix] {
			expectedSuffix++
		}
		expectedResult := fmt.Sprintf("%s-%d%s", base, expectedSuffix, ext)
		if result != expectedResult {
			t.Fatalf("ResolveCollision(%q) = %q, expected %q (first available suffix)",
				baseFilename, result, expectedResult)
		}
	})
}

// TestProperty14_FilenameCollisionResolution_DirectoryNames verifies collision resolution
// for directory names (without .md extension).
//
// **Validates: Requirements 7.6**
//
// Tag: Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
func TestProperty14_FilenameCollisionResolution_DirectoryNames(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
	// Validates: Requirements 7.6

	rapid.Check(t, func(t *rapid.T) {
		// Generate a directory name (no .md extension)
		depth := rapid.IntRange(1, 3).Draw(t, "depth")
		segments := make([]string, depth)
		for i := 0; i < depth; i++ {
			num := rapid.IntRange(1, 99).Draw(t, fmt.Sprintf("segment_%d", i))
			segments[i] = fmt.Sprintf("%d", num)
		}
		taskID := strings.Join(segments, ".")

		// Generate a slug that doesn't end in "md" to avoid ambiguity
		slug := rapid.StringMatching(`[a-ln-z][a-z]*`).Draw(t, "slug")
		if len(slug) > 15 {
			slug = slug[:15]
		}
		if len(slug) == 0 {
			slug = "parent"
		}

		dirname := taskID + "." + slug // No .md extension

		// Create existing set with the dirname
		existingFilenames := map[string]bool{
			dirname: true,
		}

		// Call ResolveCollision
		result := ResolveCollision(dirname, existingFilenames)

		// Property 1: Result should be unique
		if existingFilenames[result] {
			t.Fatalf("ResolveCollision(%q) = %q is not unique", dirname, result)
		}

		// Property 2: Result should have -2 suffix (no .md to insert before)
		expectedResult := dirname + "-2"
		if result != expectedResult {
			t.Fatalf("ResolveCollision(%q) = %q, expected %q for directory name",
				dirname, result, expectedResult)
		}

		// Property 3: Adding -2 should result in -3
		existingFilenames[result] = true
		result2 := ResolveCollision(dirname, existingFilenames)
		expectedResult2 := dirname + "-3"
		if result2 != expectedResult2 {
			t.Fatalf("Second collision for directory: ResolveCollision(%q) = %q, expected %q",
				dirname, result2, expectedResult2)
		}
	})
}

// TestProperty14_FilenameCollisionResolution_EmptyExistingSet verifies behavior with empty set:
// - Empty existing set means no collision, filename returned unchanged
//
// **Validates: Requirements 7.6**
//
// Tag: Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
func TestProperty14_FilenameCollisionResolution_EmptyExistingSet(t *testing.T) {
	// Feature: fuse-task-filesystem, Property 14: Filename Collision Resolution
	// Validates: Requirements 7.6

	rapid.Check(t, func(t *rapid.T) {
		// Generate a base filename
		baseFilename := genFilename().Draw(t, "baseFilename")

		// Empty existing set
		existingFilenames := map[string]bool{}

		// Call ResolveCollision
		result := ResolveCollision(baseFilename, existingFilenames)

		// Property: With empty set, filename should be returned unchanged
		if result != baseFilename {
			t.Fatalf("Empty set: ResolveCollision(%q) = %q, expected unchanged",
				baseFilename, result)
		}
	})
}
