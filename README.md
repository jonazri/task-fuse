# task-fuse

A FUSE filesystem that exposes `tasks.md` files as a virtual filesystem, enabling multiple AI agents to work on tasks concurrently without race conditions.

## How It Works

Tasks are organized into status directories (`pending/`, `doing/`, `done/`, `failed/`). Agents claim tasks by moving files between directories using standard `mv` commands. FUSE's atomic rename semantics prevent race conditions when multiple agents try to claim the same task.

```
/mount/
├── pending/
│   └── 1.2.implement_parser.md
├── doing/
│   └── 1.1.setup_project.md
├── done/
│   └── 1.3.write_tests.md
└── failed/
    └── index.md
```

## Usage

```bash
# Mount a tasks.md file
task-fuse mount tasks.md /mnt/tasks

# Claim a task (atomic operation)
mv /mnt/tasks/pending/1.2.implement_parser.md /mnt/tasks/doing/

# Complete a task
mv /mnt/tasks/doing/1.2.implement_parser.md /mnt/tasks/done/

# Unmount
task-fuse unmount /mnt/tasks
```

## Task Format

Tasks in `tasks.md` use checkbox syntax with numeric IDs:

```markdown
- [ ] 1. Setup project
  - [x] 1.1 Create structure
  - [-] 1.2 Configure build
- [ ] 2. Implement feature
```

| Checkbox | Status  |
|----------|---------|
| `- [ ]`  | pending |
| `- [-]`  | doing   |
| `- [x]`  | done    |
| `- [!]`  | failed  |

## Building

```bash
go build -o task-fuse ./cmd/task-fuse
```

## License

MIT
