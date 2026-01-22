# Performance Analysis Report

## Executive Summary

This report identifies performance anti-patterns, inefficient algorithms, and potential bottlenecks in the task-fuse codebase. Issues are categorized by severity and include specific file locations and recommended fixes.

---

## Critical Issues

### 1. Write Lock Held During Expensive Path Rebuild

**Location:** `internal/fuse/rename.go:102-136`

**Problem:** The rename handler acquires a write lock and holds it while calling `PathManager.OnStatusChange()`, which recursively rebuilds paths for the entire subtree.

```go
d.fs.store.Lock()  // Line 102 - Write lock acquired
// ... validation ...
srcEntry.Task.Status = dstStatus
pm := store.NewPathManager(d.fs.store)
pm.OnStatusChange(srcEntry.Task)  // Line 132 - EXPENSIVE under lock!
d.fs.store.Unlock()  // Line 135
```

**Impact:**
- `OnStatusChange()` (in `internal/store/paths.go:143-160`) finds the root ancestor and calls `RebuildPaths()` recursively
- `RebuildPaths()` (lines 54-100) performs filename generation, collision resolution, and map updates for every node
- All concurrent reads are blocked during this operation
- For deep task hierarchies (up to 10 levels), this creates significant lock contention

**Recommendation:**
- Consider computing new paths outside the lock, then apply changes atomically
- Use copy-on-write for path maps to reduce lock duration
- Pre-compute status derivation before acquiring the write lock

---

### 2. N+1 Status Derivation Pattern

**Location:** `internal/fuse/index.go:164-204`

**Problem:** Status is derived repeatedly for each task during index generation, with no memoization:

```go
// getStatusCounts() - lines 164-186
for _, task := range ig.store.Document.RootTasks {
    status := ig.deriveStatus(task)  // Called for EACH root task
    counts[status]++
}

// getTasksByStatus() - lines 188-204
for _, task := range ig.store.Document.RootTasks {
    derivedStatus := ig.deriveStatus(task)  // Called AGAIN for each task
    if derivedStatus == status {
        tasks = append(tasks, task)
    }
}
```

**Impact:**
- `DeriveParentStatus()` (`internal/store/status.go:41-96`) recursively traverses all children
- For a tree with n tasks, each index read triggers O(n) recursive calls
- When `GenerateRootIndex()` calls both `getStatusCounts()` and `GenerateStatusIndex()` for each status, this becomes O(n * s) where s = 5 statuses
- Same subtrees are traversed multiple times redundantly

**Recommendation:**
- Cache derived status during index generation
- Compute all statuses in a single tree traversal
- Store derived status in `PathEntry` when paths are rebuilt

---

### 3. O(n) Directory Listing Instead of O(1)

**Location:** `internal/fuse/fs.go:362-407`

**Problem:** `readDirWithTasks()` iterates through ALL paths to find direct children:

```go
func (d *Dir) readDirWithTasks() ([]fuse.Dirent, error) {
    entries := make([]fuse.Dirent, 0)  // No capacity hint
    // ...
    prefix := d.path + "/"
    for path, entry := range d.fs.store.TasksByPath {  // Iterates ALL tasks
        if !strings.HasPrefix(path, prefix) {
            continue
        }
        filename := path[len(prefix):]
        if strings.Contains(filename, "/") {
            continue
        }
        // ... append entry
    }
}
```

**Impact:**
- Every `readdir` system call scans the entire path map
- O(n) where n = total tasks, instead of O(k) where k = direct children
- String operations (`HasPrefix`, `Contains`) for every path

**Recommendation:**
- Maintain a secondary index mapping parent paths to child entries
- Pre-allocate slice with estimated capacity: `make([]fuse.Dirent, 0, estimatedChildren)`

---

## High Severity Issues

### 4. Inefficient `itoa()` Implementation

**Location:** `internal/store/paths.go:406-417`

**Problem:** Custom integer-to-string conversion with O(n²) complexity:

```go
func itoa(n int) string {
    var digits []byte
    for n > 0 {
        digits = append([]byte{byte('0' + n%10)}, digits...)  // PREPEND - O(n) each!
        n /= 10
    }
    return string(digits)
}
```

**Impact:**
- Each digit prepend creates a new slice and copies all existing digits
- For a 4-digit number: 4 allocations, 1+2+3+4 = 10 copies
- Called in `ResolveCollision()` for filename uniqueness

**Recommendation:**
- Use `strconv.Itoa()` which is highly optimized
- Or build digits in reverse and reverse once at the end

---

### 5. Multiple Regex Passes in Slugify

**Location:** `internal/store/paths.go:208-242`

**Problem:** Five sequential regex/string operations per slug:

```go
func Slugify(title string) string {
    slug := transliterateToASCII(title)                        // Unicode transform
    slug = strings.ToLower(slug)                               // Case conversion
    slug = whitespaceRegex.ReplaceAllString(slug, "_")         // Regex pass 1
    slug = invalidCharsRegex.ReplaceAllString(slug, "")        // Regex pass 2
    slug = multipleUnderscoresRegex.ReplaceAllString(slug, "_") // Regex pass 3
    slug = strings.Trim(slug, "_")                             // Trim
    slug = truncateAtWordBoundary(slug, MaxSlugLength)         // String search
    return slug
}
```

**Impact:**
- Called for every task during `GenerateFilename()` and `GenerateDirname()`
- Each regex pass creates a new string allocation
- `transliterateToASCII()` uses Unicode normalization (expensive)

**Recommendation:**
- Combine operations into a single-pass byte buffer transformation
- Cache slugs for identical titles
- Consider using a character-by-character loop instead of regex for simple substitutions

---

### 6. Recursive Status Derivation Without Memoization

**Location:** `internal/store/status.go:41-96`

**Problem:** `DeriveParentStatus()` recursively traverses children with no caching:

```go
func DeriveParentStatus(task *Task) TaskStatus {
    if len(task.Children) == 0 {
        return task.Status
    }
    for _, child := range task.Children {
        childStatus := DeriveParentStatus(child)  // Recursive call
        // ... process status
    }
    // ... return derived status
}
```

**Impact:**
- Called multiple times for the same subtrees during:
  - Path rebuilding (`RebuildPaths` calls `DeriveStatus` for each child)
  - Index generation (multiple calls per root task)
- For a tree with depth d and branching factor b, complexity is O(b^d) per call

**Recommendation:**
- Add memoization with a status cache cleared on task status changes
- Store derived status in `PathEntry` structure
- Compute status bottom-up once during path rebuilding

---

### 7. Duplicate Map Construction in Parser

**Location:** `internal/parser/parser.go:250-281` and `542-578`

**Problem:** The same task ID map is built twice during parsing:

```go
// In FindAncestorByIDPrefix() - line 256
taskByID := make(map[string]*store.Task)
for _, task := range tasks {
    taskByID[task.ID] = task
}

// In resolveOrphanedTasks() - line 544
taskByID := make(map[string]*store.Task)
for _, task := range tasks {
    taskByID[task.ID] = task
}
```

**Impact:**
- Two O(n) map constructions during every parse
- Unnecessary memory allocations

**Recommendation:**
- Build the map once and pass it as a parameter
- Or maintain the map as part of the parse state

---

## Medium Severity Issues

### 8. Sibling Filename Collection Inefficiency

**Location:** `internal/store/paths.go:103-134`

**Problem:** For each task, all siblings are iterated with map lookups and string operations:

```go
for _, sibling := range siblings {
    if sibling == task {
        continue
    }
    siblingPath := pm.store.PathByTask[sibling]  // Map lookup
    if siblingPath != "" && strings.HasPrefix(siblingPath, parentPath+"/") {  // String alloc + compare
        filename := siblingPath[len(parentPath)+1:]
        if !strings.Contains(filename, "/") {  // String scan
            existing[filename] = true
        }
    }
}
```

**Impact:**
- O(siblings) map lookups per task
- String concatenation `parentPath+"/"` for each sibling
- Called during `RebuildPaths()` for every task in the tree

**Recommendation:**
- Pre-compute the prefix once before the loop
- Consider maintaining a sibling filename set during path building
- Use string slicing instead of concatenation where possible

---

### 9. String Concatenation in generateTaskContent

**Location:** `internal/fuse/fs.go:589-598`

**Problem:** Uses `+=` string concatenation instead of `strings.Builder`:

```go
func generateTaskContent(task *store.Task) string {
    content := "# " + task.ID + ". " + task.Title + "\n\n"
    content += "**Status:** " + string(task.Status) + "\n\n"
    for _, line := range task.Description {
        content += line.RawContent + "\n"  // Allocation per line!
    }
    return content
}
```

**Impact:**
- Each `+=` creates a new string allocation
- For tasks with n description lines, n+2 allocations

**Recommendation:**
- Use `strings.Builder` as done in `internal/printer/printer.go`
- Pre-allocate builder capacity based on estimated content size

---

### 10. Missing Slice Pre-allocation

**Locations:** Multiple files

Examples:
- `internal/fuse/fs.go:363`: `entries := make([]fuse.Dirent, 0)`
- `internal/fuse/index.go:190`: `var tasks []*store.Task`
- `internal/sync/sync.go:472`: `var orphaned []string`

**Impact:**
- Slices grow dynamically with multiple reallocations
- Each reallocation copies all existing elements

**Recommendation:**
- Pre-allocate with estimated capacity: `make([]T, 0, estimatedSize)`
- Use `len()` of source collection when available

---

### 11. Sync Engine Mutex Lock Contention

**Location:** `internal/sync/sync.go:301-334`

**Problem:** Nested lock acquisition in timer callback:

```go
func (s *SyncEngine) handleFileChange() {
    s.mu.Lock()
    defer s.mu.Unlock()
    // ...
    s.debounceTimer = time.AfterFunc(s.debounceDuration, func() {
        s.mu.Lock()      // Nested lock in callback
        s.pendingSync = false
        inOp := s.inOperation
        s.mu.Unlock()

        if inOp {
            content, err := os.ReadFile(s.tasksFile)  // Blocking I/O
            // ...
        }
    })
}
```

**Impact:**
- Synchronous file read inside timer callback
- Lock held during file I/O preparation

**Recommendation:**
- Move file read outside the lock critical section
- Use non-blocking I/O patterns

---

## Low Severity Issues

### 12. Recursive String Building in Printer

**Location:** `internal/printer/printer.go:97-123`

**Problem:** Recursive `FormatTask()` creates intermediate strings:

```go
for _, child := range task.Children {
    result.WriteString(p.FormatTask(child))  // Recursive call returns string
}
```

**Impact:**
- Each recursive call allocates a string that's immediately copied
- For deep trees, many intermediate allocations

**Recommendation:**
- Pass `strings.Builder` as parameter to avoid intermediate strings
- Or use iterative approach with explicit stack

---

## Summary Table

| Severity | Issue | Location | Complexity Impact |
|----------|-------|----------|-------------------|
| **Critical** | Write lock during path rebuild | rename.go:102-136 | High contention |
| **Critical** | N+1 status derivation | index.go:164-204 | O(n²) |
| **Critical** | Full map scan for directory listing | fs.go:362-407 | O(n) vs O(k) |
| **High** | O(n²) itoa implementation | paths.go:406-417 | O(d²) per number |
| **High** | Multiple regex passes in Slugify | paths.go:208-242 | 5 passes per slug |
| **High** | No memoization in status derivation | status.go:41-96 | O(b^d) repeated |
| **High** | Duplicate map construction | parser.go:256,544 | 2x O(n) |
| **Medium** | Sibling collection inefficiency | paths.go:103-134 | O(siblings) lookups |
| **Medium** | String concatenation in content gen | fs.go:589-598 | O(lines) allocs |
| **Medium** | Missing slice pre-allocation | Multiple | Reallocation churn |
| **Medium** | Sync mutex contention | sync.go:301-334 | Lock + I/O |
| **Low** | Recursive string building | printer.go:97-123 | Intermediate allocs |

---

## Recommendations Priority

1. **Immediate** (Critical issues):
   - Add secondary index for parent-to-children mapping
   - Cache derived status in PathEntry
   - Reduce lock duration in rename handler

2. **Short-term** (High severity):
   - Replace custom `itoa()` with `strconv.Itoa()`
   - Add memoization to status derivation
   - Consolidate map building in parser

3. **Medium-term** (Medium severity):
   - Refactor Slugify to single-pass
   - Use strings.Builder consistently
   - Pre-allocate slices throughout

4. **Long-term** (Architecture):
   - Consider copy-on-write for path maps
   - Implement incremental path updates instead of full rebuild
   - Add metrics/profiling hooks for monitoring
