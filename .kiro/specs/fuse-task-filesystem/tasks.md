# Tasks: FUSE Task Filesystem

## Task 1: Project Setup and Core Data Structures

- [x] 1. Project Setup and Core Data Structures
  - [x] 1.1 Initialize Go module and project structure
    - Create `go.mod` with module name `task-fuse`
    - Create directory structure: `cmd/task-fuse/`, `internal/parser/`, `internal/printer/`, `internal/store/`, `internal/fuse/`, `internal/sync/`
    - Add dependencies: `bazil.org/fuse`, `github.com/fsnotify/fsnotify`
    - **Validates: Design - Implementation Language: Go**
  - [x] 1.2 Implement core data types in `internal/store/types.go`
    - Implement `TaskStatus` type with constants: `StatusPending`, `StatusQueued`, `StatusDoing`, `StatusDone`, `StatusFailed`
    - Implement `FileLine` struct with `LineNumber`, `RawContent`, `IndentLevel`
    - Implement `NonTaskContent` struct embedding `FileLine`
    - Implement `Task` struct with `FileLine`, `ID`, `Title`, `Status`, `Description`, `Children`, `Parent`
    - Implement `ParsedDocument` struct with `Preamble`, `RootTasks`, `Epilogue`
    - **Validates: Design - Task Data Model**
  - [x] 1.3 Implement TaskStore with concurrent access support in `internal/store/store.go`
    - Implement `PathEntry` struct with `Task`, `Status`, `FullPath`
    - Implement `TaskStore` struct with `Document`, `TasksByPath`, `TasksByID`, `PathByTask`, `RWLock`
    - Implement `NewTaskStore()` constructor
    - Implement read/write lock acquisition methods
    - **Validates: Design - TaskStore, Requirements 5.1, 5.4, 5.5**

## Task 2: Tasks.md Parser Implementation

- [x] 2. Tasks.md Parser Implementation
  - [x] 2.1 Implement checkbox parsing in `internal/parser/parser.go`
    - Implement `ParseCheckbox(line string) (TaskStatus, bool)` to extract status from `- [ ]`, `- [ ]`, `- [-]`, `- [x]`, `- [!]`
    - Return `false` for lines without valid checkbox syntax
    - **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5**
  - [x] 2.2 Implement task ID extraction and validation
    - Implement `ValidateTaskID(id string) bool` with regex `^[1-9][0-9]*(\.[1-9][0-9]*)*$`
    - Enforce maximum depth of 10 levels
    - Reject leading zeros (e.g., `01.1` is invalid)
    - Implement `ExtractTaskID(line string) (string, bool)` to extract numeric ID from task line
    - **Validates: Requirements 1.5, 1.6, 1.11, Design - Task ID Format**
  - [x] 2.3 Implement indentation handling
    - Implement `ComputeIndentLevel(line string) int` converting tabs to 2 spaces, then dividing by 2
    - Support flexible 2-4 spaces per indent level
    - **Validates: Design - Indentation Specification, Indent Level Computation**
  - [x] 2.4 Implement hierarchy building
    - Implement `BuildHierarchy(flatTasks []*Task) []*Task` using indent-based parent-child relationships
    - Track stack of (indent_level, task) pairs for hierarchy construction
    - **Validates: Requirements 1.7**
  - [x] 2.5 Implement orphaned ID resolution
    - Implement `FindAncestorByIDPrefix(id string, tasks []*Task) *Task`
    - Attach orphaned tasks to nearest valid ancestor by ID prefix match
    - Treat as root task if no ancestor found
    - **Validates: Requirements 1.12**
  - [x] 2.6 Implement full parser
    - Implement `Parse(content string) *ParseResult` combining all parsing logic
    - Collect preamble lines before first task
    - Attach description lines to tasks based on indentation
    - Preserve non-task content (lines without checkbox or without valid ID)
    - Collect epilogue lines after last task
    - **Validates: Requirements 1.6, 1.8, 1.11, Design - Content Attachment Rules**
  - [x] 2.7 Write property-based test for checkbox status mapping
    - Generate random task lines with valid checkboxes
    - Verify correct status mapping for all checkbox types
    - **Validates: Property 2 - Checkbox Status Mapping**
  - [x] 2.8 Write property-based test for invalid task ID handling
    - Generate task lines with invalid ID formats (leading zeros, non-numeric segments)
    - Verify they are preserved as non-task content
    - **Validates: Property 15 - Invalid Task ID Handling**
  - [x] 2.9 Write property-based test for orphaned ID resolution
    - Generate tasks with orphaned IDs (missing parents)
    - Verify attachment to nearest valid ancestor
    - **Validates: Property 16 - Orphaned Task ID Resolution**

## Task 3: Pretty Printer Implementation

- [x] 3. Pretty Printer Implementation
  - [x] 3.1 Implement status-to-checkbox conversion in `internal/printer/printer.go`
    - Implement `StatusToCheckbox(status TaskStatus) string` returning `- [ ]`, `- [ ]`, `- [-]`, `- [x]`, `- [!]`
    - **Validates: Requirements 1.9**
  - [x] 3.2 Implement task serialization
    - Implement `FormatTask(task *Task) string` preserving original indentation and content
    - Only modify checkbox characters when status changes
    - Recursively format children
    - **Validates: Requirements 1.9**
  - [x] 3.3 Implement full document serialization
    - Implement `Serialize(doc *ParsedDocument) string`
    - Output preamble, root tasks with children, and epilogue in original order
    - Preserve all non-task content exactly as parsed
    - **Validates: Requirements 1.9, 1.10**
  - [x] 3.4 Write property-based test for parser round-trip consistency
    - Generate random valid task trees
    - Serialize with Pretty_Printer, parse with Parser, compare
    - **Validates: Property 1 - Parser Round-Trip Consistency**
  - [x] 3.5 Write property-based test for malformed line preservation
    - Generate files with malformed lines (invalid checkboxes, no IDs)
    - Verify lines preserved through round-trip
    - **Validates: Property 13 - Malformed Line Preservation**

## Task 4: Filename Generation and Path Management

- [x] 4. Filename Generation and Path Management
  - [x] 4.1 Implement filename slugification in `internal/store/paths.go`
    - Convert to lowercase ASCII
    - Replace whitespace with underscores
    - Keep only `[a-z0-9_-]`
    - Collapse multiple underscores, trim edges
    - Truncate to 50 characters at word boundary
    - Use `task` if result is empty
    - Format: `{id}.{slug}.md`
    - **Validates: Requirements 2.10, Design - Filename Slugification Rules**
  - [x] 4.2 Implement collision detection and resolution
    - Detect filename collisions within directories
    - Append numeric suffix (`-2`, `-3`, etc.) before `.md` extension
    - **Validates: Requirements 7.6**
  - [x] 4.3 Implement PathManager for path tracking
    - Implement `GenerateFilename(task *Task) string`
    - Implement `RebuildPaths(task *Task, newStatus TaskStatus, parentPath string)`
    - Implement `OnStatusChange(task *Task)` to rebuild entire subtree paths
    - Update all indexes (`TasksByPath`, `TasksByID`, `PathByTask`)
    - **Validates: Design - Path Management**
  - [x] 4.4 Write property-based test for filename generation consistency
    - Generate random tasks
    - Verify filename follows rules and is deterministic
    - **Validates: Property 5 - Filename Generation Consistency**
  - [x] 4.5 Write property-based test for collision resolution
    - Generate tasks with colliding names
    - Verify unique suffixes applied
    - **Validates: Property 14 - Filename Collision Resolution**

## Task 5: Parent Status Derivation

- [x] 5. Parent Status Derivation
  - [x] 5.1 Implement parent status derivation in `internal/store/status.go`
    - Implement `DeriveParentStatus(parent *Task) TaskStatus`
    - Follow precedence: doing > failed > pending > queued > done
    - Recursively derive for nested parents
    - **Validates: Requirements 4.7**
  - [x] 5.2 Implement status propagation on child changes
    - When any child status changes, recalculate all ancestor statuses
    - Update paths for affected subtrees
    - **Validates: Requirements 4.7**
  - [x] 5.3 Write property-based test for parent status derivation
    - Generate random child status combinations
    - Verify parent status follows precedence rules
    - **Validates: Property 10 - Parent Status Derivation**

## Task 6: FUSE Filesystem - Directory Structure

- [x] 6. FUSE Filesystem - Directory Structure
  - [x] 6.1 Implement root filesystem node in `internal/fuse/fs.go`
    - Implement `FuseFS` struct with `store`, `syncEngine`, `mountPoint`
    - Implement `Root() (fs.Node, error)` returning root directory
    - **Validates: Design - FUSE_Filesystem Interface**
  - [x] 6.2 Implement status directories
    - Create `pending/`, `queued/`, `doing/`, `done/`, `failed/` directories at mount root
    - Implement `Dir` struct with `Attr`, `Lookup`, `ReadDirAll`
    - **Validates: Requirements 2.1**
  - [x] 6.3 Implement task file nodes
    - Implement `File` struct for leaf tasks
    - Implement `Attr` returning read-only permissions
    - Implement `Read` returning task content per Task File Content Format
    - **Validates: Requirements 2.2, 2.9**
  - [x] 6.4 Implement task directory nodes for parent tasks
    - Expose parent tasks as directories containing sub-tasks
    - Preserve hierarchical nesting (e.g., `pending/1.parent/1.2.child/1.2.3.leaf.md`)
    - **Validates: Requirements 2.3, 2.4**
  - [x] 6.5 Implement status-based task placement
    - Place tasks in correct status directory based on their status
    - Children always appear under parent directory regardless of individual status
    - **Validates: Requirements 2.5, 2.6, 2.7, 2.8, Design - Example 1**
  - [x] 6.6 Write property-based test for status directory placement
    - Generate tasks with random statuses
    - Verify correct directory placement
    - **Validates: Property 4 - Status Directory Placement**
  - [x] 6.7 Write property-based test for hierarchy preservation
    - Generate random nested task structures
    - Verify parent-child relationships preserved in filesystem
    - **Validates: Property 3 - Task Hierarchy Preservation**

## Task 7: FUSE Filesystem - Index Files

- [x] 7. FUSE Filesystem - Index Files
  - [x] 7.1 Implement IndexGenerator in `internal/fuse/index.go`
    - Implement `GenerateRootIndex()` with status counts
    - Implement `GenerateStatusIndex(status TaskStatus)` listing tasks in status directory
    - Implement `GenerateTaskIndex(task *Task)` with parent description and sub-task list
    - Implement `GenerateTaskFileContent(task *Task)` per Task File Content Format
    - **Validates: Requirements 3.4, 3.5, 3.6, 3.7, 3.8**
  - [x] 7.2 Implement on-demand index generation
    - Generate index.md content at read time (not cached)
    - Ensure index always reflects current state
    - **Validates: Requirements 3.9**
  - [x] 7.3 Write property-based test for index file completeness
    - Generate random task structures
    - Verify index.md contains all items in each directory
    - **Validates: Property 7 - Index File Completeness**

## Task 8: FUSE Filesystem - Read-Only Enforcement

- [x] 8. FUSE Filesystem - Read-Only Enforcement
  - [x] 8.1 Implement write rejection
    - Implement `Write` returning EPERM for task files
    - Implement `Write` returning EPERM for index files
    - **Validates: Requirements 3.1**
  - [x] 8.2 Implement create rejection
    - Implement `Create` returning EPERM for all directories
    - **Validates: Requirements 3.2**
  - [x] 8.3 Implement delete rejection
    - Implement `Remove` returning EPERM for all files
    - **Validates: Requirements 3.3**
  - [x] 8.4 Implement attribute modification rejection
    - Implement `Setattr` returning EROFS
    - Implement `Mkdir` returning EPERM
    - **Validates: Design - All mutating operations return EPERM or EROFS**
  - [x] 8.5 Write property-based test for read-only enforcement
    - Generate random file operations on task files
    - Verify write/create/delete return EPERM
    - **Validates: Property 6 - Read-Only Enforcement**

## Task 9: FUSE Filesystem - Atomic Move Operations

- [x] 9. FUSE Filesystem - Atomic Move Operations
  - [x] 9.1 Implement rename handler with locking in `internal/fuse/rename.go`
    - Acquire write lock before processing
    - Validate source path exists (return ENOENT if not)
    - Release lock after operation
    - **Validates: Requirements 5.1, 5.2**
  - [x] 9.2 Implement move validation
    - Reject same-directory renames (EPERM)
    - Reject filename changes (EPERM)
    - Reject parent directory moves (EPERM)
    - **Validates: Requirements 6.4, Design - Rename Validation Rules**
  - [x] 9.3 Implement status transition validation
    - Allow: pending→queued/doing, queued→doing/pending, doing→done/pending/failed, done→doing, failed→pending/doing
    - Reject all other transitions with EPERM
    - **Validates: Requirements 6.1, 6.2**
  - [x] 9.4 Implement concurrent move handling
    - Return ENOENT when task was already moved by another agent
    - Ensure immediate visibility of successful moves
    - **Validates: Requirements 5.3, 5.4, 5.5**
  - [x] 9.5 Write property-based test for atomic move exclusivity
    - Simulate concurrent move attempts
    - Verify exactly one succeeds, others get ENOENT
    - **Validates: Property 11 - Atomic Move Exclusivity**
  - [x] 9.6 Write property-based test for status transition enforcement
    - Generate all possible transitions
    - Verify valid succeed, invalid fail with EPERM
    - **Validates: Property 12 - Status Transition Enforcement**

## Task 10: Sync Engine - File Watching

- [x] 10. Sync Engine - File Watching
  - [x] 10.1 Implement file watcher in `internal/sync/sync.go`
    - Use `fsnotify` for file change detection
    - Implement `StartWatching()` and `StopWatching()`
    - Debounce rapid changes (100ms window)
    - **Validates: Requirements 4.5**
  - [x] 10.2 Implement polling fallback
    - Fall back to polling (1 second interval) if file watching fails
    - Configurable via `--sync-interval` CLI option
    - **Validates: Design - Graceful Degradation**
  - [x] 10.3 Implement change queuing during operations
    - Queue external modifications during filesystem operations
    - Process queued changes after operation completes
    - **Validates: Requirements 4.6**

## Task 11: Sync Engine - Bidirectional Sync

- [x] 11. Sync Engine - Bidirectional Sync
  - [x] 11.1 Implement sync-to-file
    - Serialize task tree via Pretty_Printer
    - Write to temporary file, then atomic rename
    - **Validates: Requirements 4.1**
  - [x] 11.2 Implement sync-from-file
    - Parse new file content
    - Diff against current task store
    - Apply non-conflicting changes
    - **Validates: Requirements 4.2**
  - [x] 11.3 Implement conflict resolution
    - Filesystem operations take precedence over external modifications
    - Discard external changes for conflicting tasks
    - **Validates: Requirements 4.4**
  - [x] 11.4 Implement content integrity preservation
    - Preserve task descriptions through sync operations
    - Only modify checkbox syntax for status changes
    - **Validates: Requirements 4.3**
  - [x] 11.5 Write property-based test for sync content integrity
    - Generate random tasks with descriptions
    - Perform sync operations
    - Verify content preserved
    - **Validates: Property 8 - Sync Content Integrity**
  - [x] 11.6 Write property-based test for filesystem precedence in conflicts
    - Generate concurrent filesystem and external changes
    - Verify filesystem wins
    - **Validates: Property 9 - Filesystem Precedence in Conflicts**

## Task 12: Sync Engine - Orphaned Task Handling

- [x] 12. Sync Engine - Orphaned Task Handling
  - [x] 12.1 Implement orphaned children detection
    - Detect when external edit removes parent but leaves children
    - Log warning when orphaned children detected
    - **Validates: Requirements 4.9**
  - [x] 12.2 Implement orphaned children promotion
    - Promote orphaned children to removed parent's parent level
    - Preserve original task IDs (no renumbering)
    - **Validates: Requirements 4.8**
  - [x] 12.3 Write property-based test for orphaned children promotion
    - Simulate external removal of parent tasks
    - Verify children promoted correctly with preserved IDs
    - **Validates: Property 17 - Orphaned Children Promotion**

## Task 13: Error Handling

- [x] 13. Error Handling
  - [x] 13.1 Implement missing file handling
    - Return EIO when tasks.md is missing
    - **Validates: Requirements 7.1**
  - [x] 13.2 Implement permission error handling
    - Return EACCES when tasks.md is unreadable
    - **Validates: Requirements 7.2**
  - [x] 13.3 Implement graceful shutdown
    - Flush pending changes on unmount
    - Wait for in-progress operations to complete
    - **Validates: Requirements 7.3**
  - [x] 13.4 Implement malformed content handling
    - Preserve malformed lines as-is
    - Skip parsing them as tasks
    - **Validates: Requirements 7.4**
  - [x] 13.5 Implement stale reference handling
    - Return ENOENT for tasks that no longer exist in tasks.md
    - **Validates: Requirements 7.5**

## Task 14: CLI Implementation

- [x] 14. CLI Implementation
  - [x] 14.1 Implement main entry point in `cmd/task-fuse/main.go`
    - Parse command-line arguments
    - Dispatch to appropriate command handler
    - **Validates: Design - CLI Interface**
  - [x] 14.2 Implement mount command
    - Accept `<tasks.md>` and `<mountpoint>` arguments
    - Support `--foreground`, `--log-level`, `--log-format`, `--log-file`, `--sync-interval` options
    - Validate mountpoint exists and is empty
    - **Validates: Design - Mount Command**
  - [x] 14.3 Implement unmount command
    - Accept `<mountpoint>` argument
    - Support `--force` and `--timeout` options
    - Flush pending changes before unmount
    - **Validates: Design - Unmount Command**
  - [x] 14.4 Implement status command
    - Show mounted filesystem status
    - Display task counts by status
    - Show last sync time
    - **Validates: Design - Status Command**
  - [x] 14.5 Implement version command
    - Display version information
    - **Validates: Design - Commands**
  - [x] 14.6 Implement exit codes
    - Return 0 for success
    - Return 1 for general errors
    - Return 2 for mount failures
    - Return 3 for parse errors
    - **Validates: Design - Exit Codes**

## Task 15: Integration Testing

- [x] 15. Integration Testing
  - [x] 15.1 Write end-to-end workflow test
    - Mount filesystem, move tasks through workflow, verify tasks.md updates
    - Test complete lifecycle: pending → doing → done
    - **Validates: Design - Integration Tests**
  - [x] 15.2 Write multi-agent simulation test
    - Multiple goroutines competing for tasks
    - Verify no race conditions
    - Verify exactly one agent claims each task
    - **Validates: Design - Integration Tests, Requirements 5.2**
  - [x] 15.3 Write external modification test
    - Modify tasks.md while filesystem is mounted
    - Verify sync detects and applies changes
    - **Validates: Design - Integration Tests**
  - [x] 15.4 Write crash recovery test
    - Simulate process termination during operation
    - Verify consistent state on restart
    - **Validates: Design - Integration Tests**


## Task 16: Linux Remote Testing (FUSE Verification)

- [x] 16. Linux Remote Testing (FUSE Verification)
  - [x] 16.1 Set up remote test environment
    - SSH to Linux dev box
    - Verify Go is installed (install if needed)
    - Verify FUSE is available (`fusermount --version`)
    - Create working directory for test execution
    - **Validates: Requirements 5.1, 5.2 - FUSE operations require Linux**
  - [x] 16.2 Deploy code to remote environment
    - Copy project files to remote Linux box via rsync/scp
    - Run `go mod download` to fetch dependencies
    - Verify `bazil.org/fuse` compiles successfully on Linux
    - **Validates: Design - Implementation Language: Go**
  - [x] 16.3 Run FUSE unit tests on Linux
    - Execute `go test ./internal/fuse/...` on remote
    - Verify fs_test.go tests pass
    - Capture and review test output
    - **Validates: Task 6, Task 7, Task 8, Task 9 - FUSE filesystem tests**
  - [x] 16.4 Run FUSE integration tests on Linux
    - Execute `go test ./internal/integration/...` on remote
    - Verify end-to-end workflow test passes
    - Verify multi-agent simulation test passes
    - Verify external modification test passes
    - Verify crash recovery test passes
    - **Validates: Task 15 - Integration Testing**
  - [x] 16.5 Run full test suite on Linux
    - Execute `go test ./...` to run all tests
    - Verify all property-based tests pass
    - Document any test failures for investigation
    - **Validates: All Properties 1-17**
  - [x] 16.6 Manual FUSE mount verification
    - Build the binary: `go build ./cmd/task-fuse`
    - Create a test tasks.md file
    - Mount filesystem and verify directory structure
    - Test manual task moves between status directories
    - Verify tasks.md updates correctly
    - Unmount and verify clean shutdown
    - **Validates: Requirements 2.1-2.10, 4.1-4.9, 6.1-6.4**


## Task 17: PR Review Comments

- [x] 17. Address PR Review Comments
  - [x] 17.1 Fetch and review line-level comments from PR
    - Use `gh api repos/jonazri/task-fuse/pulls/1/comments` to fetch review comments
    - Document each comment and the file/line it references
    - Categorize comments by type (bug fix, style, documentation, etc.)
  - [x] 17.2 Address each review comment
    - Make code changes as requested in comments
    - Run tests to verify changes don't break functionality
    - Document any comments that were intentionally not addressed with rationale
  - [x] 17.3 Verify all changes on Linux
    - Deploy updated code to Linux remote box
    - Run full test suite to ensure no regressions
    - **Validates: Task 16 - Linux Remote Testing**


## Task 18: Address TODOs and Placeholders

- [x] 18. Address TODOs and Placeholders
  - [x] 18.1 Implement proper logging infrastructure
    - Add a logging library (e.g., `log/slog` or `zerolog`)
    - Replace placeholder log configuration in `cmd/task-fuse/main.go:138`
    - Wire up `--log-level`, `--log-format`, `--log-file` CLI options
    - **File: cmd/task-fuse/main.go**
  - [x] 18.2 Add logging for sync errors
    - Replace `_ = err // TODO: Add proper logging` in `internal/sync/sync.go:258`
    - Log sync errors with appropriate severity level
    - **File: internal/sync/sync.go**
  - [x] 18.3 Add logging for orphaned children detection
    - Replace `// TODO: Add proper logging` in `internal/sync/sync.go:437`
    - Log warning when orphaned children are detected and promoted
    - Include task IDs in log message
    - **Validates: Requirements 4.9**
    - **File: internal/sync/sync.go**
  - [x] 18.4 Implement status command properly
    - Replace placeholder message in `cmd/task-fuse/main.go:302`
    - Track mounted filesystems (possibly via PID file or shared state)
    - Display actual task counts and sync status
    - **Validates: Design - Status Command**
    - **File: cmd/task-fuse/main.go**


## Task 19: Address Code Review Feedback

- [x] 19. Address Code Review Feedback (from tmp/codereview.md)
  - [x] 19.1 Consolidate triple-duplicated status derivation logic (HIGH)
    - Move canonical `DeriveStatus()` to `internal/store/status.go`
    - Update `PathManager.DeriveStatus()` in paths.go to delegate to it
    - Update `IndexGenerator.deriveStatus()` in index.go to delegate to it
    - Remove duplicate implementations
    - **Files: internal/store/paths.go, internal/store/status.go, internal/fuse/index.go**
  - [x] 19.2 Extract duplicate lookup methods (MEDIUM)
    - Create `lookupInDir(name string)` helper in fs.go
    - Refactor `lookupInStatusDir()` and `lookupInTaskDir()` to use helper
    - **File: internal/fuse/fs.go:239-297**
  - [x] 19.3 Extract duplicate readDir methods (MEDIUM)
    - Create helper for common iteration logic
    - Refactor `readDirStatus()` and `readDirTask()` to use helper
    - **File: internal/fuse/fs.go:374-467**
  - [x] 19.4 Refactor rename to use defer for lock release (MEDIUM)
    - Replace manual `Unlock()` calls with `defer d.fs.store.Unlock()`
    - Restructure validation to allow early returns without manual unlock
    - **File: internal/fuse/rename.go:88-134**
  - [x] 19.5 Remove ghost code - unused functions (LOW)
    - Remove `extractFilename()` function (never called)
    - Remove `isLeafTaskPath()` function (never called)
    - **File: internal/fuse/rename.go:169-177**
  - [x] 19.6 Remove ghost code - unused changeQueue channel (LOW)
    - Remove `changeQueue chan FileChange` from SyncEngine struct
    - Remove initialization in NewSyncEngine
    - **File: internal/sync/sync.go:47-48, 99**
  - [x] 19.7 Define EROFS constant (LOW)
    - Replace magic number `fuse.Errno(30)` with named constant
    - Apply to both File.Setattr and Dir.Setattr
    - **File: internal/fuse/fs.go:700, 709**
  - [x] 19.8 Handle temp file cleanup error (LOW)
    - Log error if temp file removal fails after atomic rename failure
    - **File: internal/sync/sync.go:653**
  - [x] 19.9 Add test coverage for read-only enforcement
    - Add explicit tests for Write, Create, Remove, Mkdir, Setattr methods
    - Verify EPERM/EROFS returns
    - **File: internal/fuse/fs_test.go**
