# PR Review Comments Documentation

## PR #1: FUSE Task Filesystem Implementation

**Review Date:** 2026-01-20
**Reviewer:** GitHub Copilot (copilot-pull-request-reviewer[bot])
**Commit:** cefc7b239b47a37ed29b0ff753b3795a27f64cb4

---

## Summary

The PR review covered 31 out of 39 changed files and generated **1 line-level comment** that requires attention.

---

## Line-Level Comments

### Comment 1: Directory Name Collision Resolution Bug

**File:** `internal/store/paths.go`
**Lines:** 421-426 (function `splitFilenameExtension`)
**Type:** Bug Fix
**Priority:** High

#### Issue Description

The `splitFilenameExtension` function incorrectly handles directory names that happen to end with `.md`. When `ResolveCollision` is called with a directory name like `1.md` (which has `.md` in the name but is a directory, not a file), the function incorrectly returns `1-2.md` instead of the expected `1.md-2`.

#### Root Cause

The function treats any filename ending in `.md` as having a file extension, even for directory names. This causes the collision suffix to be inserted before `.md` instead of appended at the end.

#### Evidence

A property-based test failure was recorded in:
```
internal/store/testdata/rapid/TestProperty14_FilenameCollisionResolution_DirectoryNames/
TestProperty14_FilenameCollisionResolution_DirectoryNames-20260120012346-47684.fail
```

The test shows:
- Input: `ResolveCollision("1.md")` (directory name)
- Actual output: `"1-2.md"`
- Expected output: `"1.md-2"`

#### Suggested Fix

The reviewer suggests adding a heuristic to distinguish between file names and directory-like names:

```go
// splitFilenameExtension splits a filename into base and extension.
// For "1.1.task.md", returns ("1.1.task", ".md").
// For "task", returns ("task", "").
func splitFilenameExtension(filename string) (base, ext string) {
	// Look for .md extension specifically (case-sensitive)
	if strings.HasSuffix(filename, ".md") {
		base := filename[:len(filename)-3]

		// Heuristic: names like "1.md" are used as directory names in some cases.
		// For such directory-like names, we must *not* treat ".md" as a file
		// extension; instead, collision suffixes are appended after the full name,
		// e.g. "1.md" -> "1.md-2".
		//
		// To avoid changing existing behavior for regular files like
		// "task.md" or "1.1.task.md", we only treat the name as having a
		// ".md" extension when the part before ".md" is not purely numeric.
		if !isAllDigits(base) {
			return base, ".md"
		}
	}
	// No .md extension found (or treated as directory-like), return filename
	// as base with empty extension.
	return filename, ""
}

// isAllDigits reports whether s consists solely of decimal digit characters.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
```

#### Affected Functionality

- **Property 14:** Filename Collision Resolution
- **Requirement 7.6:** Filename collision handling

---

## General Review Notes

The PR review summary noted the following areas were reviewed:

| File | Description |
|------|-------------|
| internal/sync/sync.go | Synchronization engine with file watching, debouncing, and bidirectional sync |
| internal/store/types.go | Core data structures for tasks, statuses, and parsed documents |
| internal/store/store.go | Task store with concurrent access support using read-write locks |
| internal/store/status.go | Parent status derivation with precedence rules |
| internal/store/paths.go | Path management, filename generation with slugification, and collision resolution |
| internal/printer/printer.go | Markdown serialization with checkbox status mapping |
| internal/store/testdata/rapid/*.fail | Property-based test failure data |
| internal/store/*_test.go | Unit and property-based tests for store functionality |
| internal/printer/printer_test.go | Unit tests for printer functionality |

---

## Action Items

| # | Task | File | Type | Status |
|---|------|------|------|--------|
| 1 | Fix `splitFilenameExtension` to handle directory names ending in `.md` | `internal/store/paths.go` | Bug Fix | Pending |

---

## Related Tasks

This documentation supports:
- **Task 17.1:** Fetch and review line-level comments from PR
- **Task 17.2:** Address each review comment
