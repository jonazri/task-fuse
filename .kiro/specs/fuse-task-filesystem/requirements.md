# Requirements: FUSE Task Filesystem

## Overview

This document specifies the requirements for a FUSE-based task manager filesystem that enables multiple AI agents to work on Kiro PRD tasks.md files concurrently without race conditions. The filesystem provides bidirectional synchronization between a tasks.md file and a virtual filesystem where tasks are represented as individual files organized by status directories. Atomic filesystem operations ensure race-free task claiming when multiple agents attempt to work on the same task simultaneously.

## Glossary

- **FUSE_Filesystem**: The Filesystem in Userspace implementation that exposes tasks as files and directories
- **Task_File**: A read-only virtual file representing a leaf task (task with no sub-tasks)
- **Leaf_Task**: A task that has no sub-tasks and is represented as a file in the filesystem
- **Task_Directory**: A virtual directory representing a parent task that contains sub-tasks
- **Parent_Task**: A task that contains one or more sub-tasks and is represented as a directory
- **Status_Directory**: A top-level directory representing a task status (pending, queued, doing, done, failed)
- **Index_File**: A read-only `index.md` file in each directory listing all tasks and their descriptions
- **Tasks_MD_Parser**: Component that parses the tasks.md markdown format into task objects
- **Pretty_Printer**: Component that serializes task objects back to markdown format
- **Sync_Engine**: Component responsible for bidirectional synchronization between filesystem and tasks.md
- **Task_Claim**: The atomic operation of moving a task from pending to doing status
- **Task_ID**: A dot-separated sequence of positive integers identifying a task (e.g., `1`, `1.2`, `1.2.3`)

## Task ID Format Specification

Task IDs follow a strict format to ensure consistent parsing and hierarchy representation:

- **Format**: Dot-separated positive integers (e.g., `1`, `1.2`, `1.2.3`, `10.5.2`)
- **Regex**: `^[1-9][0-9]*(\.[1-9][0-9]*)*$`
- **No leading zeros**: `01.1` or `1.01` are invalid
- **Hierarchy implied**: ID `1.2.3` implies ancestors `1.2` and `1` should exist (but gaps are tolerated)
- **Gaps allowed**: `1.1`, `1.3` without `1.2` is valid (tasks may be deleted)
- **Maximum depth**: 10 levels (e.g., `1.2.3.4.5.6.7.8.9.10` is the deepest valid ID)
- **Uniqueness**: Each task ID must be unique within the document

## Indentation Specification

Indentation determines task hierarchy (parent-child relationships):

- **Flexible spacing**: 2-4 spaces = 1 indent level
- **Tab handling**: 1 tab = 1 indent level (internally converted to 2 spaces)
- **Mixed tabs/spaces**: Tabs are converted to 2 spaces first, then total spaces are counted
- **Hierarchy determination**: A task is a child of the nearest preceding task with less indentation
- **Root tasks**: Tasks with zero indentation are root-level tasks

**Examples:**
```
- [ ] 1. Root task              (indent level 0)
  - [ ] 1.1 Child task          (indent level 1, 2 spaces)
    - [ ] 1.1.1 Grandchild      (indent level 2, 4 spaces)
   - [ ] 1.1.2 Also grandchild  (indent level 2, 3 spaces - still valid)
- [ ] 2. Another root task      (indent level 0)
```

**Edge cases:**
- Inconsistent indentation (mixing 2 and 4 spaces) is tolerated; hierarchy is determined by relative comparison
- A decrease in indentation closes the current subtree and starts a sibling or ancestor

## Filename Slugification Rules

When generating filenames from task titles, the following algorithm SHALL be applied:

1. Convert all characters to lowercase ASCII (non-ASCII characters are transliterated or removed)
2. Replace all whitespace sequences with a single underscore (`_`)
3. Remove all characters except `[a-z0-9_-]`
4. Collapse multiple consecutive underscores into one
5. Trim leading and trailing underscores
6. Truncate to maximum 50 characters (at word boundary if possible)
7. If result is empty, use the literal string `task`
8. Final filename format: `{task_id}.{slug}.md` (e.g., `1.2.implement_parser.md`)

## Task File Content Format

When reading a task file, the content SHALL be formatted as:

```markdown
# {task_id}. {title}

**Status:** {status}

{description lines, preserved exactly as in tasks.md}
```

Example:
```markdown
# 1.2. Implement the parser

**Status:** pending

This task involves creating the markdown parser.
- Handle checkbox syntax
- Extract task IDs
```

## Requirements

### Requirement 1: Parse Tasks.md Format

**User Story:** As a system integrator, I want the filesystem to parse tasks.md files, so that existing PRD task lists can be exposed through the filesystem.

#### Acceptance Criteria

1. WHEN a tasks.md file is loaded, THE Tasks_MD_Parser SHALL parse markdown checkbox syntax `- [ ]` as pending status
2. WHEN a tasks.md file is loaded, THE Tasks_MD_Parser SHALL parse markdown checkbox syntax `- [~]` as queued status
3. WHEN a tasks.md file is loaded, THE Tasks_MD_Parser SHALL parse markdown checkbox syntax `- [-]` as doing status
4. WHEN a tasks.md file is loaded, THE Tasks_MD_Parser SHALL parse markdown checkbox syntax `- [x]` as done status
5. WHEN a tasks.md file is loaded, THE Tasks_MD_Parser SHALL parse markdown checkbox syntax `- [!]` as failed status
5. WHEN a task line contains a numeric identifier (e.g., `1.1`, `2.3.1`), THE Tasks_MD_Parser SHALL extract it as the task ID
6. WHEN a task line has checkbox syntax but lacks a numeric identifier, THE Tasks_MD_Parser SHALL ignore it (not parse it as a task)
7. WHEN a task has nested sub-tasks based on indentation, THE Tasks_MD_Parser SHALL build a hierarchical tree structure
8. WHEN a task contains additional metadata or bullet points, THE Tasks_MD_Parser SHALL preserve all content as task description
9. THE Pretty_Printer SHALL serialize task objects back to valid tasks.md markdown format with correct checkbox syntax
10. FOR ALL valid task objects, parsing then printing then parsing SHALL produce an equivalent object (round-trip property)
11. IF a task ID does not match the Task ID Format Specification (e.g., `01.1`, `1.2.a`, `1..2`), THE Tasks_MD_Parser SHALL treat the line as non-task content and preserve it unchanged
12. IF a task has an orphaned ID (e.g., `1.2.3` exists but `1.2` does not), THE Tasks_MD_Parser SHALL attach it to the nearest valid ancestor by ID prefix, or treat as a root task if no ancestor exists

### Requirement 2: Expose Tasks as Files and Directories

**User Story:** As an AI agent, I want tasks exposed as files and directories in status directories, so that I can discover and interact with tasks using standard filesystem operations.

#### Acceptance Criteria

1. WHEN the FUSE_Filesystem mounts, THE FUSE_Filesystem SHALL create status directories: `pending/`, `queued/`, `doing/`, `done/`, `failed/`
2. WHEN a task has no sub-tasks (leaf task), THE FUSE_Filesystem SHALL expose it as a file
3. WHEN a task has sub-tasks (parent task), THE FUSE_Filesystem SHALL expose it as a directory containing its sub-tasks
4. THE FUSE_Filesystem SHALL preserve the hierarchical nesting of tasks (e.g., task `1.2.3` appears as `pending/1.parent/1.2.child/1.2.3.leaf.md`)
5. WHEN a task has pending status, THE FUSE_Filesystem SHALL place it in the `pending/` directory
6. WHEN a task has queued status, THE FUSE_Filesystem SHALL place it in the `queued/` directory
7. WHEN a task has doing status, THE FUSE_Filesystem SHALL place it in the `doing/` directory
8. WHEN a task has done status, THE FUSE_Filesystem SHALL place it in the `done/` directory
9. WHEN a task has failed status, THE FUSE_Filesystem SHALL place it in the `failed/` directory
10. WHEN reading a task file, THE FUSE_Filesystem SHALL return the task content formatted according to the Task File Content Format specification
11. THE FUSE_Filesystem SHALL generate filenames according to the Filename Slugification Rules specification
12. THE FUSE_Filesystem SHALL maintain an internal mapping between filenames and their corresponding task entries in tasks.md

### Requirement 3: Read-Only Task Files with Index

**User Story:** As a system operator, I want task files to be read-only with index files for discovery, so that task content is protected and easily browsable.

#### Acceptance Criteria

1. WHEN an agent attempts to write to a task file, THE FUSE_Filesystem SHALL return EPERM (Operation not permitted)
2. WHEN an agent attempts to create a new file in a status directory, THE FUSE_Filesystem SHALL return EPERM
3. WHEN an agent attempts to delete a task file directly, THE FUSE_Filesystem SHALL return EPERM
4. THE FUSE_Filesystem SHALL create a read-only `index.md` file in each status directory
5. THE FUSE_Filesystem SHALL create a read-only `index.md` file in each task directory (parent task)
6. THE FUSE_Filesystem SHALL create a read-only `index.md` file at the mount root listing all status directories and overall task counts
7. WHEN reading an `index.md` file in a status directory, THE FUSE_Filesystem SHALL return a markdown list of all tasks and task directories within it.
8. WHEN reading an `index.md` file in a task directory, THE FUSE_Filesystem SHALL return the parent task's description, followed by a markdown list of its sub-tasks.
9. THE FUSE_Filesystem SHALL generate index.md content on-demand at read time to ensure it always reflects current state

### Requirement 4: Bidirectional Synchronization

**User Story:** As a system operator, I want changes to sync bidirectionally, so that the filesystem and tasks.md stay consistent.

#### Acceptance Criteria

1. WHEN a task file is moved between status directories, THE Sync_Engine SHALL update the corresponding task status in tasks.md
2. WHEN the tasks.md file is modified externally, THE Sync_Engine SHALL update the filesystem to reflect the changes
3. WHEN synchronizing changes, THE Sync_Engine SHALL preserve task content integrity including descriptions
4. WHEN a sync conflict occurs (filesystem operation and external file modification affect the same task), THE Sync_Engine SHALL resolve as follows: filesystem operations always take precedence, and the external modification is discarded for that task
5. THE Sync_Engine SHALL detect external tasks.md modifications using file watching or polling
6. WHEN a filesystem operation is in progress, THE Sync_Engine SHALL queue external modifications and apply them after the operation completes
7. THE `Sync_Engine` SHALL automatically manage the location (and therefore status) of a parent `Task_Directory` based on the status of its children, following a strict precedence:
    - `doing/`: If **any** child task is in `doing/`.
    - `failed/`: If **no** child tasks are in `doing/` and **at least one** child task is in `failed/`.
    - `pending/`: If **no** child tasks are in `doing/` or `failed/`, and **at least one** child task is in `pending/`.
    - `queued/`: If **no** child tasks are in `doing/`, `failed/`, or `pending/`, and **at least one** child task is in `queued/`.
    - `done/`: If **all** child tasks are in `done/`.
8. WHEN an external modification removes a parent task but leaves its children, THE Sync_Engine SHALL promote orphaned children to the removed parent's parent (or to root level if the removed parent was a root task), preserving their original IDs
9. THE Sync_Engine SHALL log a warning when orphaned children are detected and promoted

### Requirement 5: Atomic Task Claiming via Filesystem Semantics

**User Story:** As an AI agent coordinator, I want atomic task claiming operations, so that multiple agents cannot claim the same task simultaneously.

#### Acceptance Criteria

1. WHEN an agent moves a task file from `pending/` to `doing/`, THE FUSE_Filesystem SHALL perform the operation atomically using internal locking
2. WHEN two agents attempt to move the same task file simultaneously, THE FUSE_Filesystem SHALL succeed for exactly one agent and fail for the other
3. WHEN a task move operation fails due to concurrent access, THE FUSE_Filesystem SHALL return ENOENT (No such file or directory). The agent SHOULD re-list the directory and select a different task rather than retrying the same move.
4. WHEN a task is successfully claimed, THE FUSE_Filesystem SHALL immediately reflect the new status in directory listings
5. WHEN checking task availability via stat or readdir, THE FUSE_Filesystem SHALL return current state without stale data

### Requirement 6: Support Task Status Workflow

**User Story:** As an AI agent, I want to transition tasks through the workflow, so that I can track task progress from pending to completion.

#### Acceptance Criteria

1. THE FUSE_Filesystem SHALL enforce the following valid status transitions for leaf tasks:
   - `pending/` → `queued/` (queue for execution)
   - `pending/` → `doing/` (claim)
   - `queued/` → `doing/` (claim from queue)
   - `queued/` → `pending/` (unqueue)
   - `doing/` → `done/` (complete)
   - `doing/` → `pending/` (unclaim)
   - `doing/` → `failed/` (fail)
   - `done/` → `doing/` (reopen)
   - `failed/` → `pending/` (queue reattempt)
   - `failed/` → `doing/` (immediate reattempt)
2. WHEN any transition not listed in 6.1 is attempted on a leaf task, THE FUSE_Filesystem SHALL return EPERM (Operation not permitted)
3. WHEN a leaf task's status changes via a `move` operation, THE Sync_Engine SHALL update the task's checkbox syntax in the `tasks.md` file
4. WHEN an agent attempts to move a parent `Task_Directory`, THE FUSE_Filesystem SHALL return EPERM (parent task status is derived from children per Requirement 4.7)

### Requirement 7: Handle Edge Cases and Errors

**User Story:** As a system operator, I want robust error handling, so that the filesystem remains stable under unexpected conditions.

#### Acceptance Criteria

1. IF the tasks.md file is missing, THEN THE FUSE_Filesystem SHALL return EIO on file operations until the file is created
2. IF the tasks.md file is unreadable due to permissions, THEN THE FUSE_Filesystem SHALL return EACCES
3. IF the filesystem is unmounted during an operation, THEN THE Sync_Engine SHALL flush pending changes to tasks.md before shutdown
4. IF the tasks.md file contains malformed checkbox syntax, THEN THE Tasks_MD_Parser SHALL preserve the line as-is and skip parsing it as a task
5. IF a task file is accessed that no longer exists in tasks.md, THEN THE FUSE_Filesystem SHALL return ENOENT
6. IF generating a filename results in a collision, THEN THE FUSE_Filesystem SHALL append a numeric suffix before the extension (e.g., `1.1.implement_parser.md` becomes `1.1.implement_parser-2.md`)


## Non-Requirements (Explicitly Out of Scope)

The following capabilities are explicitly NOT part of this specification:

1. **Task creation via filesystem**: New tasks cannot be created by creating files in the filesystem. Tasks must be added by editing tasks.md directly.

2. **Task deletion via filesystem**: Tasks cannot be deleted by removing files. Tasks must be removed by editing tasks.md directly.

3. **Task content editing via filesystem**: Task descriptions and titles cannot be modified through the filesystem. The filesystem is read-only except for status transitions via move operations.

4. **Multi-file support**: The system operates on a single tasks.md file. Managing multiple task files simultaneously is not supported.

5. **Remote/network filesystems**: The FUSE filesystem is designed for local mounting only. NFS, CIFS, or other network filesystem protocols are not supported.

6. **Symbolic links**: The filesystem does not support creating or following symbolic links.

7. **Extended attributes (xattrs)**: Extended file attributes are not supported.

8. **File permissions management**: All files have fixed permissions (read-only for task files, no write/execute). chmod operations are not supported.

9. **Real-time collaboration**: While multiple agents can work concurrently, there is no real-time notification system for task changes. Agents must poll or re-scan directories.

10. **Task dependencies**: The system does not model or enforce task dependencies (e.g., "task B requires task A to be done first").

11. **Due dates or scheduling**: Time-based task attributes are not part of the core specification.

12. **User/agent identity tracking**: The system does not track which agent claimed or completed a task.
