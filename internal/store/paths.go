// Package store provides path management utilities for the task filesystem.
package store

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// PathManager manages path generation and tracking for tasks in the filesystem.
// It handles filename generation, collision resolution, and path rebuilding
// when task statuses change.
//
// Important: PathManager methods do NOT acquire locks - the caller must hold
// the appropriate lock on the TaskStore before calling these methods.
type PathManager struct {
	store *TaskStore
}

// NewPathManager creates a new PathManager with the given TaskStore.
func NewPathManager(store *TaskStore) *PathManager {
	return &PathManager{
		store: store,
	}
}

// DeriveStatus derives the status for a task.
// For leaf tasks (no children): returns the task's own status.
// For parent tasks: derives status from children using precedence rules.
// Precedence: doing > failed > pending > queued > done
//
// Validates: Requirements 4.7
func (pm *PathManager) DeriveStatus(task *Task) TaskStatus {
	if task == nil {
		return StatusPending
	}

	// Leaf task: return its own status
	if len(task.Children) == 0 {
		return task.Status
	}

	// Parent task: derive from children
	// Precedence: doing > failed > pending > queued > done
	hasDoing := false
	hasFailed := false
	hasPending := false
	hasQueued := false
	allDone := true

	for _, child := range task.Children {
		childStatus := pm.DeriveStatus(child)
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

	// Apply precedence rules
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

	// Default to pending (shouldn't reach here with valid children)
	return StatusPending
}

// RebuildPaths rebuilds paths for a task and all its descendants.
// It generates filenames, resolves collisions with siblings, builds full paths,
// and updates all indexes in the store.
//
// Parameters:
//   - task: The task to rebuild paths for
//   - newStatus: The status to use for this task (may be derived for parents)
//   - parentPath: The parent directory path (e.g., "/pending" or "/doing/1.parent_task")
//
// Note: This method does NOT acquire locks - the caller must hold the write lock.
func (pm *PathManager) RebuildPaths(task *Task, newStatus TaskStatus, parentPath string) {
	if task == nil || pm.store == nil {
		return
	}

	// Generate filename for this task
	// Parent tasks (directories) don't have .md extension
	// Leaf tasks (files) have .md extension
	var filename string
	if len(task.Children) > 0 {
		// Parent task - directory name without .md
		filename = GenerateDirname(task)
	} else {
		// Leaf task - file name with .md
		filename = GenerateFilename(task)
	}

	// Resolve collisions with siblings at this level
	// Collect existing filenames at this directory level
	existingFilenames := pm.collectSiblingFilenames(task, parentPath)
	filename = ResolveCollision(filename, existingFilenames)

	// Build the full path
	newPath := parentPath + "/" + filename

	// Remove old path entry if it exists
	oldPath := pm.store.PathByTask[task]
	if oldPath != "" {
		delete(pm.store.TasksByPath, oldPath)
	}

	// Create and add new entry
	entry := &PathEntry{
		Task:     task,
		Status:   newStatus,
		FullPath: newPath,
	}
	pm.store.TasksByPath[newPath] = entry
	pm.store.TasksByID[task.ID] = entry
	pm.store.PathByTask[task] = newPath

	// Recursively update children (they move with parent)
	// Children appear under the parent's directory, not in separate status directories
	for _, child := range task.Children {
		childStatus := pm.DeriveStatus(child)
		pm.RebuildPaths(child, childStatus, newPath)
	}
}

// collectSiblingFilenames collects filenames of siblings at the same directory level.
// This is used for collision detection when generating filenames.
func (pm *PathManager) collectSiblingFilenames(task *Task, parentPath string) map[string]bool {
	existing := make(map[string]bool)

	// Get siblings (other children of the same parent, or other root tasks)
	var siblings []*Task
	if task.Parent != nil {
		siblings = task.Parent.Children
	} else if pm.store.Document != nil {
		siblings = pm.store.Document.RootTasks
	}

	// Collect filenames of siblings that have already been processed
	for _, sibling := range siblings {
		if sibling == task {
			continue // Skip self
		}
		// Check if sibling already has a path at this level
		siblingPath := pm.store.PathByTask[sibling]
		if siblingPath != "" && strings.HasPrefix(siblingPath, parentPath+"/") {
			// Extract just the filename from the path
			filename := siblingPath[len(parentPath)+1:]
			// Only include direct children (no further slashes)
			if !strings.Contains(filename, "/") {
				existing[filename] = true
			}
		}
	}

	return existing
}

// OnStatusChange is called when any task's status changes.
// It finds the root ancestor and rebuilds the entire subtree's paths.
// This ensures that when a child's status changes, the parent's derived
// status is recalculated and the entire subtree moves to the correct
// status directory.
//
// Note: This method does NOT acquire locks - the caller must hold the write lock.
func (pm *PathManager) OnStatusChange(task *Task) {
	if task == nil || pm.store == nil {
		return
	}

	// Find root ancestor
	root := task
	for root.Parent != nil {
		root = root.Parent
	}

	// Derive the root's status
	rootStatus := pm.DeriveStatus(root)

	// Rebuild paths for the entire subtree starting from root
	// The root goes into the status directory matching its derived status
	pm.RebuildPaths(root, rootStatus, "/"+string(rootStatus))
}

// BuildAllPaths builds paths for all tasks in the document.
// This should be called after initial parse or reload.
//
// Note: This method does NOT acquire locks - the caller must hold the write lock.
func (pm *PathManager) BuildAllPaths() {
	if pm.store == nil || pm.store.Document == nil {
		return
	}

	// Clear existing path indexes
	pm.store.TasksByPath = make(map[string]*PathEntry)
	pm.store.TasksByID = make(map[string]*PathEntry)
	pm.store.PathByTask = make(map[*Task]string)

	// Build paths for each root task
	for _, rootTask := range pm.store.Document.RootTasks {
		rootStatus := pm.DeriveStatus(rootTask)
		pm.RebuildPaths(rootTask, rootStatus, "/"+string(rootStatus))
	}
}

// MaxSlugLength is the maximum length for a slugified title.
const MaxSlugLength = 50

var (
	// whitespaceRegex matches one or more whitespace characters.
	whitespaceRegex = regexp.MustCompile(`\s+`)

	// invalidCharsRegex matches characters that are not allowed in slugs.
	// Only [a-z0-9_-] are allowed.
	invalidCharsRegex = regexp.MustCompile(`[^a-z0-9_-]`)

	// multipleUnderscoresRegex matches two or more consecutive underscores.
	multipleUnderscoresRegex = regexp.MustCompile(`_+`)
)

// Slugify converts a task title to a URL-safe slug for use in filenames.
// It follows the Filename Slugification Rules from the design document:
//
// 1. Convert all characters to lowercase ASCII (non-ASCII characters are transliterated or removed)
// 2. Replace all whitespace sequences with a single underscore (_)
// 3. Remove all characters except [a-z0-9_-]
// 4. Collapse multiple consecutive underscores into one
// 5. Trim leading and trailing underscores
// 6. Truncate to maximum 50 characters (at word boundary if possible)
// 7. If result is empty, use the literal string "task"
func Slugify(title string) string {
	if title == "" {
		return "task"
	}

	// Step 1: Transliterate non-ASCII characters to ASCII equivalents
	// This uses Unicode normalization (NFD) to decompose characters,
	// then removes non-spacing marks (accents, diacritics)
	slug := transliterateToASCII(title)

	// Convert to lowercase
	slug = strings.ToLower(slug)

	// Step 2: Replace all whitespace sequences with a single underscore
	slug = whitespaceRegex.ReplaceAllString(slug, "_")

	// Step 3: Remove all characters except [a-z0-9_-]
	slug = invalidCharsRegex.ReplaceAllString(slug, "")

	// Step 4: Collapse multiple consecutive underscores into one
	slug = multipleUnderscoresRegex.ReplaceAllString(slug, "_")

	// Step 5: Trim leading and trailing underscores
	slug = strings.Trim(slug, "_")

	// Step 6: Truncate to maximum 50 characters (at word boundary if possible)
	slug = truncateAtWordBoundary(slug, MaxSlugLength)

	// Step 7: If result is empty, use "task"
	if slug == "" {
		return "task"
	}

	return slug
}

// transliterateToASCII converts non-ASCII characters to their ASCII equivalents
// where possible, and removes characters that cannot be transliterated.
func transliterateToASCII(s string) string {
	// Use Unicode normalization to decompose characters (NFD form)
	// This separates base characters from combining marks (accents, diacritics)
	// Then we remove the combining marks to get ASCII-like characters
	t := transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)), // Remove non-spacing marks (accents)
		norm.NFC,
	)

	result, _, err := transform.String(t, s)
	if err != nil {
		// If transformation fails, return original string
		return s
	}

	// Remove any remaining non-ASCII characters
	var builder strings.Builder
	for _, r := range result {
		if r < 128 {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// truncateAtWordBoundary truncates a slug to maxLen characters.
// The truncation prefers word boundaries (underscores) when possible:
// - If the character at position maxLen is an underscore, truncate there and trim it
// - If truncating at maxLen would cut a word, and there's an underscore within
//   the last 10 characters, truncate at that underscore
// - Otherwise, truncate at exactly maxLen
func truncateAtWordBoundary(slug string, maxLen int) string {
	if len(slug) <= maxLen {
		return slug
	}

	truncated := slug[:maxLen]

	// If the truncated string ends with an underscore, trim it
	truncated = strings.TrimRight(truncated, "_")

	// Check if we cut in the middle of a word by looking at the next character
	// in the original string. If it's not an underscore, we cut a word.
	if len(slug) > maxLen && slug[maxLen] != '_' {
		// We cut in the middle of a word. Look for an underscore to truncate at,
		// but only if it's reasonably close (within last 10 chars of maxLen)
		lastUnderscore := strings.LastIndex(truncated, "_")

		// Only truncate at word boundary if the underscore is within the last 10 chars
		// This prevents truncating too aggressively (e.g., from 50 to 10 chars)
		if lastUnderscore > 0 && lastUnderscore >= maxLen-10 {
			return truncated[:lastUnderscore]
		}
	}

	return truncated
}

// GenerateFilename generates the filename for a task.
// Format: {id}.{slug}.md
// Example: "1.2.implement_parser.md"
//
// The task ID is used as-is (it's already validated by the parser).
// The title is slugified according to the Filename Slugification Rules.
func GenerateFilename(task *Task) string {
	if task == nil {
		return ""
	}

	slug := Slugify(task.Title)
	return task.ID + "." + slug + ".md"
}

// GenerateDirname generates the directory name for a parent task.
// Format: {id}.{slug} (no .md extension)
// Example: "1.setup_project"
//
// Parent tasks are represented as directories in the filesystem,
// so they don't have the .md extension that leaf task files have.
func GenerateDirname(task *Task) string {
	if task == nil {
		return ""
	}

	slug := Slugify(task.Title)
	return task.ID + "." + slug
}

// ResolveCollision resolves filename collisions by appending numeric suffixes.
// If the filename already exists in existingFilenames, it appends -2, -3, etc.
// before the .md extension until a unique filename is found.
//
// Example:
//   - "1.1.task.md" with no collision → "1.1.task.md"
//   - "1.1.task.md" with collision → "1.1.task-2.md"
//   - "1.1.task.md" with -2 also taken → "1.1.task-3.md"
//
// Validates: Requirements 7.6
func ResolveCollision(filename string, existingFilenames map[string]bool) string {
	// If no collision, return the filename as-is
	if !existingFilenames[filename] {
		return filename
	}

	// Extract base and extension
	base, ext := splitFilenameExtension(filename)

	// Start from suffix 2 and increment until we find a unique name
	suffix := 2
	for {
		candidate := base + "-" + itoa(suffix) + ext
		if !existingFilenames[candidate] {
			return candidate
		}
		suffix++
	}
}

// splitFilenameExtension splits a filename into base and extension.
// For "1.1.task.md", returns ("1.1.task", ".md").
// For "task", returns ("task", "").
func splitFilenameExtension(filename string) (base, ext string) {
	// Look for .md extension specifically (case-sensitive)
	if strings.HasSuffix(filename, ".md") {
		return filename[:len(filename)-3], ".md"
	}
	// No .md extension found, return filename as base with empty extension
	return filename, ""
}

// itoa converts an integer to a string without importing strconv.
// This is a simple implementation for small positive integers.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
