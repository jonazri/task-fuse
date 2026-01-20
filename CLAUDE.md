# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

task-fuse is a Go-based FUSE filesystem that exposes Kiro PRD `tasks.md` files as a virtual filesystem. It enables multiple AI agents to work on tasks concurrently without race conditions by using atomic filesystem operations (`mv`) for task status transitions.

**Current State:** Specifications complete, implementation not yet started.

## Build Commands

```bash
# Initialize (once project structure exists)
go mod tidy

# Build
go build -o task-fuse ./cmd/task-fuse

# Run tests
go test ./...

# Run single test
go test -run TestName ./internal/parser

# Run with verbose output
go test -v ./...
```

## Project Structure (Planned)

```
cmd/task-fuse/       # CLI entry point
internal/
  parser/            # tasks.md markdown parser
  printer/           # Markdown serialization
  store/             # TaskStore, path management, status derivation
  fuse/              # FUSE filesystem implementation
  sync/              # Bidirectional sync engine
```

## Architecture

### Core Components

1. **Tasks_MD_Parser** - Parses checkbox-based task lists from markdown into hierarchical task trees
2. **Pretty_Printer** - Serializes task objects back to markdown, preserving non-task content
3. **FUSE_Filesystem** - Exposes tasks as files in status directories (`pending/`, `doing/`, `done/`, `failed/`)
4. **Sync_Engine** - Maintains bidirectional sync between filesystem and tasks.md
5. **TaskStore** - In-memory task storage with `sync.RWMutex` for concurrent access

### Key Design Decisions

- **Filesystem wins conflicts**: When filesystem operations and external edits conflict, filesystem takes precedence
- **Children live under parents**: Child tasks always appear under their parent directory regardless of individual status
- **Parent status derived**: Parent task status is auto-derived from children (doing > failed > pending > done)
- **Read-only files**: Task content is read-only; only status transitions via `mv` are allowed
- **ENOENT for races**: When agents race to claim a task, losers get ENOENT (task was moved)

### Task ID Format

- Pattern: `^[1-9][0-9]*(\.[1-9][0-9]*)*$`
- Examples: `1`, `1.2`, `10.5.2`
- No leading zeros, max 10 levels deep
- Lines with checkboxes but no valid numeric ID are preserved as non-task content

### Checkbox Syntax

| Status  | Checkbox | Example |
|---------|----------|---------|
| Pending | `- [ ]`  | `- [ ] 1.1 Task` |
| Doing   | `- [-]`  | `- [-] 1.2 Task` |
| Done    | `- [x]`  | `- [x] 1.3 Task` |
| Failed  | `- [!]`  | `- [!] 1.4 Task` |

### Valid Status Transitions (Leaf Tasks Only)

- `pending` → `doing`
- `doing` → `done`, `pending`, `failed`
- `done` → `doing`
- `failed` → `pending`, `doing`

Parent directories cannot be moved (EPERM).

### Dependencies

- `bazil.org/fuse` - FUSE filesystem library
- `github.com/fsnotify/fsnotify` - File watching for sync

## Specification Documents

Detailed requirements, design, and tasks are in `.kiro/specs/fuse-task-filesystem/`:
- `requirements.md` - Acceptance criteria and specifications
- `design.md` - Architecture, data models, algorithms, properties
- `tasks.md` - Implementation task breakdown
