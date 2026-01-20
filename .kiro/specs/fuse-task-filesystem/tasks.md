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

- [ ] 2. Tasks.md Parser Implementation
  - [x] 2.1 Implement checkbox parsing in `internal/parser/parser.go`
    - Implement `ParseCheckbox(line string) (TaskStatus, bool)` to extract status from `- [ ]`, `- [~]`, `- [-]`, `- [x]`, `- [!]`
    - Return `false` for lines without valid checkbox syntax
    - **Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5**
  - [-] 2.2 Implement task ID extraction and validation
    - Implement `ValidateTaskID(id string) bool` with regex `^[1-9][0-9]*(\.[1-9][0-9]*)*$`
    - Enforce maximum depth of 10 levels
    - Reject leading zeros (e.g., `01.1` is invalid)
    - Implement `ExtractTaskID(line string) (string, bool)` to extract numeric ID from task line
    - **Validates: Requirements 1.5, 1.6, 1.11, Design - Task ID Format**
  - [~] 2.3 Implement indentation handling
    - Implement `ComputeIndentLevel(line string) int` converting tabs to 2 spaces, then dividing by 2
    - Support flexible 2-4 spaces per indent level
    - **Validates: Design - Indentation Specification, Indent Level Computation**
  - [~] 2.4 Implement hierarchy building
    - Implement `BuildHierarchy(flatTasks []*Task) []*Task` using indent-based parent-child relationships
    - Track stack of (indent_level, task) pairs for hierarchy construction
    - **Validates: Requirements 1.7**
  - [~] 2.5 Implement orphaned ID resolution
    - Implement `FindAncestorByIDPrefix(id string, tasks []*Task) *Task`
    - Attach orphaned tasks to nearest valid ancestor by ID prefix match
    - Treat as root task if no ancestor found
    - **Validates: Requirements 1.12**
  - [~] 2.6 Implement full parser
    - Implement `Parse(content string) *ParseResult` combining all parsing logic
    - Collect preamble lines before first task
    - Attach description lines to tasks based on indentation
    - Preserve non-task content (lines without checkbox or without valid ID)
    - Collect epilogue lines after last task
    - **Validates: Requirements 1.6, 1.8, 1.11, Design - Content Attachment Rules**
  - [~] 2.7 Write property-based test for checkbox status mapping
    - Generate random task lines with valid checkboxes
    - Verify correct status mapping for all checkbox types
    - **Validates: Property 2 - Checkbox Status Mapping**
  - [~] 2.8 Write property-based test for invalid task ID handling
    - Generate task lines with invalid ID formats (leading zeros, non-numeric segments)
    - Verify they are preserved as non-task content
    - **Validates: Property 15 - Invalid Task ID Handling**
  - [~] 2.9 Write property-based test for orphaned ID resolution
    - Generate tasks with orphaned IDs (missing parents)
    - Verify attachment to nearest valid ancestor
    - **Validates: Property 16 - Orphaned Task ID Resolution**

## Task 3: Pretty Printer Implementation

- [ ] 3. Pretty Printer Implementation
  - [~] 3.1 Implement status-to-checkbox conversion in `internal/printer/printer.go`
    - Implement `StatusToCheckbox(status TaskStatus) string` returning `- [ ]`, `- [~]`, `- [-]`, `- [x]`, `- [!]`
    - **Validates: Requirements 1.9**
  - [~] 3.2 Implement task serialization
    - Implement `FormatTask(task *Task) string` preserving original indentation and content
    - Only modify checkbox characters when status changes
    - Recursively format children
    - **Validates: Requirements 1.9**
  - [~] 3.3 Implement full document serialization
    - Implement `Serialize(doc *ParsedDocument) string`
    - Output preamble, root tasks with children, and epilogue in original order
    - Preserve all non-task content exactly as parsed
    - **Validates: Requirements 1.9, 1.10**
  - [~] 3.4 Write property-based test for parser round-trip consistency
    - Generate random valid task trees
    - Serialize with Pretty_Printer, parse with Parser, compare
    - **Validates: Property 1 - Parser Round-Trip Consistency**
  - [~] 3.5 Write property-based test for malformed line preservation
    - Generate files with malformed lines (invalid checkboxes, no IDs)
    - Verify lines preserved through round-trip
    - **Validates: Property 13 - Malformed Line Preservation**

## Task 4: Filename Generation and Path Management

- [ ] 4. Filename Generation and Path Management
  - [~] 4.1 Implement filename slugification in `internal/store/paths.go`
    - Convert to lowercase ASCII
    - Replace whitespace with underscores
    - Keep only `[a-z0-9_-]`
    - Collapse multiple underscores, trim edges
    - Truncate to 50 characters at word boundary
    - Use `task` if result is empty
    - Format: `{id}.{slug}.md`
    - **Validates: Requirements 2.10, Design - Filename Slugification Rules**
  - [~] 4.2 Implement collision detection and resolution
    - Detect filename collisions within directories
    - Append numeric suffix (`-2`, `-3`, etc.) before `.md` extension
    - **Validates: Requirements 7.6**
  - [~] 4.3 Implement PathManager for path tracking
    - Implement `GenerateFilename(task *Task) string`
    - Implement `RebuildPaths(task *Task, newStatus TaskStatus, parentPath string)`
    - Implement `OnStatusChange(task *Task)` to rebuild entire subtree paths
    - Update all indexes (`TasksByPath`, `TasksByID`, `PathByTask`)
    - **Validates: Design - Path Management**
  - [~] 4.4 Write property-based test for filename generation consistency
    - Generate random tasks
    - Verify filename follows rules and is deterministic
    - **Validates: Property 5 - Filename Generation Consistency**
  - [~] 4.5 Write property-based test for collision resolution
    - Generate tasks with colliding names
    - Verify unique suffixes applied
    - **Validates: Property 14 - Filename Collision Resolution**

## Task 5: Parent Status Derivation

- [ ] 5. Parent Status Derivation
  - [~] 5.1 Implement parent status derivation in `internal/store/status.go`
    - Implement `DeriveParentStatus(parent *Task) TaskStatus`
    - Follow precedence: doing > failed > pending > queued > done
    - Recursively derive for nested parents
    - **Validates: Requirements 4.7**
  - [~] 5.2 Implement status propagation on child changes
    - When any child status changes, recalculate all ancestor statuses
    - Update paths for affected subtrees
    - **Validates: Requirements 4.7**
  - [~] 5.3 Write property-based test for parent status derivation
    - Generate random child status combinations
    - Verify parent status follows precedence rules
    - **Validates: Property 10 - Parent Status Derivation**

## Task 6: FUSE Filesystem - Directory Structure

- [ ] 6. FUSE Filesystem - Directory Structure
  - [~] 6.1 Implement root filesystem node in `internal/fuse/fs.go`
    - Implement `FuseFS` struct with `store`, `syncEngine`, `mountPoint`
    - Implement `Root() (fs.Node, error)` returning root directory
    - **Validates: Design - FUSE_Filesystem Interface**
  - [~] 6.2 Implement status directories
    - Create `pending/`, `queued/`, `doing/`, `done/`, `failed/` directories at mount root
    - Implement `Dir` struct with `Attr`, `Lookup`, `ReadDirAll`
    - **Validates: Requirements 2.1**
  - [~] 6.3 Implement task file nodes
    - Implement `File` struct for leaf tasks
    - Implement `Attr` returning read-only permissions
    - Implement `Read` returning task content per Task File Content Format
    - **Validates: Requirements 2.2, 2.9**
  - [~] 6.4 Implement task directory nodes for parent tasks
    - Expose parent tasks as directories containing sub-tasks
    - Preserve hierarchical nesting (e.g., `pending/1.parent/1.2.child/1.2.3.leaf.md`)
    - **Validates: Requirements 2.3, 2.4**
  - [~] 6.5 Implement status-based task placement
    - Place tasks in correct status directory based on their status
    - Children always appear under parent directory regardless of individual status
    - **Validates: Requirements 2.5, 2.6, 2.7, 2.8, Design - Example 1**
  - [~] 6.6 Write property-based test for status directory placement
    - Generate tasks with random statuses
    - Verify correct directory placement
    - **Validates: Property 4 - Status Directory Placement**
  - [~] 6.7 Write property-based test for hierarchy preservation
    - Generate random nested task structures
    - Verify parent-child relationships preserved in filesystem
    - **Validates: Property 3 - Task Hierarchy Preservation**

## Task 7: FUSE Filesystem - Index Files

- [ ] 7. FUSE Filesystem - Index Files
  - [~] 7.1 Implement IndexGenerator in `internal/fuse/index.go`
    - Implement `GenerateRootIndex()` with status counts
    - Implement `GenerateStatusIndex(status TaskStatus)` listing tasks in status directory
    - Implement `GenerateTaskIndex(task *Task)` with parent description and sub-task list
    - Implement `GenerateTaskFileContent(task *Task)` per Task File Content Format
    - **Validates: Requirements 3.4, 3.5, 3.6, 3.7, 3.8**
  - [~] 7.2 Implement on-demand index generation
    - Generate index.md content at read time (not cached)
    - Ensure index always reflects current state
    - **Validates: Requirements 3.9**
  - [~] 7.3 Write property-based test for index file completeness
    - Generate random task structures
    - Verify index.md contains all items in each directory
    - **Validates: Property 7 - Index File Completeness**

## Task 8: FUSE Filesystem - Read-Only Enforcement

- [ ] 8. FUSE Filesystem - Read-Only Enforcement
  - [~] 8.1 Implement write rejection
    - Implement `Write` returning EPERM for task files
    - Implement `Write` returning EPERM for index files
    - **Validates: Requirements 3.1**
  - [~] 8.2 Implement create rejection
    - Implement `Create` returning EPERM for all directories
    - **Validates: Requirements 3.2**
  - [~] 8.3 Implement delete rejection
    - Implement `Remove` returning EPERM for all files
    - **Validates: Requirements 3.3**
  - [~] 8.4 Implement attribute modification rejection
    - Implement `Setattr` returning EROFS
    - Implement `Mkdir` returning EPERM
    - **Validates: Design - All mutating operations return EPERM or EROFS**
  - [~] 8.5 Write property-based test for read-only enforcement
    - Generate random file operations on task files
    - Verify write/create/delete return EPERM
    - **Validates: Property 6 - Read-Only Enforcement**

## Task 9: FUSE Filesystem - Atomic Move Operations

- [ ] 9. FUSE Filesystem - Atomic Move Operations
  - [~] 9.1 Implement rename handler with locking in `internal/fuse/rename.go`
    - Acquire write lock before processing
    - Validate source path exists (return ENOENT if not)
    - Release lock after operation
    - **Validates: Requirements 5.1, 5.2**
  - [~] 9.2 Implement move validation
    - Reject same-directory renames (EPERM)
    - Reject filename changes (EPERM)
    - Reject parent directory moves (EPERM)
    - **Validates: Requirements 6.4, Design - Rename Validation Rules**
  - [~] 9.3 Implement status transition validation
    - Allow: pending→queued/doing, queued→doing/pending, doing→done/pending/failed, done→doing, failed→pending/doing
    - Reject all other transitions with EPERM
    - **Validates: Requirements 6.1, 6.2**
  - [~] 9.4 Implement concurrent move handling
    - Return ENOENT when task was already moved by another agent
    - Ensure immediate visibility of successful moves
    - **Validates: Requirements 5.3, 5.4, 5.5**
  - [~] 9.5 Write property-based test for atomic move exclusivity
    - Simulate concurrent move attempts
    - Verify exactly one succeeds, others get ENOENT
    - **Validates: Property 11 - Atomic Move Exclusivity**
  - [~] 9.6 Write property-based test for status transition enforcement
    - Generate all possible transitions
    - Verify valid succeed, invalid fail with EPERM
    - **Validates: Property 12 - Status Transition Enforcement**

## Task 10: Sync Engine - File Watching

- [ ] 10. Sync Engine - File Watching
  - [~] 10.1 Implement file watcher in `internal/sync/sync.go`
    - Use `fsnotify` for file change detection
    - Implement `StartWatching()` and `StopWatching()`
    - Debounce rapid changes (100ms window)
    - **Validates: Requirements 4.5**
  - [~] 10.2 Implement polling fallback
    - Fall back to polling (1 second interval) if file watching fails
    - Configurable via `--sync-interval` CLI option
    - **Validates: Design - Graceful Degradation**
  - [~] 10.3 Implement change queuing during operations
    - Queue external modifications during filesystem operations
    - Process queued changes after operation completes
    - **Validates: Requirements 4.6**

## Task 11: Sync Engine - Bidirectional Sync

- [ ] 11. Sync Engine - Bidirectional Sync
  - [~] 11.1 Implement sync-to-file
    - Serialize task tree via Pretty_Printer
    - Write to temporary file, then atomic rename
    - **Validates: Requirements 4.1**
  - [~] 11.2 Implement sync-from-file
    - Parse new file content
    - Diff against current task store
    - Apply non-conflicting changes
    - **Validates: Requirements 4.2**
  - [~] 11.3 Implement conflict resolution
    - Filesystem operations take precedence over external modifications
    - Discard external changes for conflicting tasks
    - **Validates: Requirements 4.4**
  - [~] 11.4 Implement content integrity preservation
    - Preserve task descriptions through sync operations
    - Only modify checkbox syntax for status changes
    - **Validates: Requirements 4.3**
  - [~] 11.5 Write property-based test for sync content integrity
    - Generate random tasks with descriptions
    - Perform sync operations
    - Verify content preserved
    - **Validates: Property 8 - Sync Content Integrity**
  - [~] 11.6 Write property-based test for filesystem precedence in conflicts
    - Generate concurrent filesystem and external changes
    - Verify filesystem wins
    - **Validates: Property 9 - Filesystem Precedence in Conflicts**

## Task 12: Sync Engine - Orphaned Task Handling

- [ ] 12. Sync Engine - Orphaned Task Handling
  - [~] 12.1 Implement orphaned children detection
    - Detect when external edit removes parent but leaves children
    - Log warning when orphaned children detected
    - **Validates: Requirements 4.9**
  - [~] 12.2 Implement orphaned children promotion
    - Promote orphaned children to removed parent's parent level
    - Preserve original task IDs (no renumbering)
    - **Validates: Requirements 4.8**
  - [~] 12.3 Write property-based test for orphaned children promotion
    - Simulate external removal of parent tasks
    - Verify children promoted correctly with preserved IDs
    - **Validates: Property 17 - Orphaned Children Promotion**

## Task 13: Error Handling

- [ ] 13. Error Handling
  - [~] 13.1 Implement missing file handling
    - Return EIO when tasks.md is missing
    - **Validates: Requirements 7.1**
  - [~] 13.2 Implement permission error handling
    - Return EACCES when tasks.md is unreadable
    - **Validates: Requirements 7.2**
  - [~] 13.3 Implement graceful shutdown
    - Flush pending changes on unmount
    - Wait for in-progress operations to complete
    - **Validates: Requirements 7.3**
  - [~] 13.4 Implement malformed content handling
    - Preserve malformed lines as-is
    - Skip parsing them as tasks
    - **Validates: Requirements 7.4**
  - [~] 13.5 Implement stale reference handling
    - Return ENOENT for tasks that no longer exist in tasks.md
    - **Validates: Requirements 7.5**

## Task 14: CLI Implementation

- [ ] 14. CLI Implementation
  - [~] 14.1 Implement main entry point in `cmd/task-fuse/main.go`
    - Parse command-line arguments
    - Dispatch to appropriate command handler
    - **Validates: Design - CLI Interface**
  - [~] 14.2 Implement mount command
    - Accept `<tasks.md>` and `<mountpoint>` arguments
    - Support `--foreground`, `--log-level`, `--log-format`, `--log-file`, `--sync-interval` options
    - Validate mountpoint exists and is empty
    - **Validates: Design - Mount Command**
  - [~] 14.3 Implement unmount command
    - Accept `<mountpoint>` argument
    - Support `--force` and `--timeout` options
    - Flush pending changes before unmount
    - **Validates: Design - Unmount Command**
  - [~] 14.4 Implement status command
    - Show mounted filesystem status
    - Display task counts by status
    - Show last sync time
    - **Validates: Design - Status Command**
  - [~] 14.5 Implement version command
    - Display version information
    - **Validates: Design - Commands**
  - [~] 14.6 Implement exit codes
    - Return 0 for success
    - Return 1 for general errors
    - Return 2 for mount failures
    - Return 3 for parse errors
    - **Validates: Design - Exit Codes**

## Task 15: Integration Testing

- [ ] 15. Integration Testing
  - [~] 15.1 Write end-to-end workflow test
    - Mount filesystem, move tasks through workflow, verify tasks.md updates
    - Test complete lifecycle: pending → doing → done
    - **Validates: Design - Integration Tests**
  - [~] 15.2 Write multi-agent simulation test
    - Multiple goroutines competing for tasks
    - Verify no race conditions
    - Verify exactly one agent claims each task
    - **Validates: Design - Integration Tests, Requirements 5.2**
  - [~] 15.3 Write external modification test
    - Modify tasks.md while filesystem is mounted
    - Verify sync detects and applies changes
    - **Validates: Design - Integration Tests**
  - [~] 15.4 Write crash recovery test
    - Simulate process termination during operation
    - Verify consistent state on restart
    - **Validates: Design - Integration Tests**
