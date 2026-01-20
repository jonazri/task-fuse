# task-fuse

A FUSE filesystem that exposes `tasks.md` files as a virtual filesystem, enabling multiple AI agents to work on tasks concurrently without race conditions.

**Platform: Linux only.** The FUSE library (`bazil.org/fuse`) does not support macOS or Windows.

## How It Works

Tasks are organized into status directories (`pending/`, `queued/`, `doing/`, `done/`, `failed/`). Agents claim tasks by moving files between directories using standard `mv` commands. FUSE's atomic rename semantics prevent race conditions when multiple agents try to claim the same task.

```
/mount/
├── index.md
├── pending/
│   ├── index.md
│   └── 1.2.implement_parser.md
├── queued/
│   └── index.md
├── doing/
│   ├── index.md
│   └── 1.1.setup_project.md
├── done/
│   ├── index.md
│   └── 1.3.write_tests.md
└── failed/
    └── index.md
```

## Installation

### Quick Install

```bash
git clone https://github.com/jonazri/task-fuse.git
cd task-fuse
./install.sh
```

The install script will:
1. Detect your Linux distro and package manager
2. Install Go (if missing or too old)
3. Install FUSE dependencies
4. Build and install to `/usr/local/bin`

### Manual Installation

#### Prerequisites

- Linux (required for FUSE support)
- Go 1.21 or later
- FUSE development libraries

```bash
# Debian/Ubuntu
sudo apt-get install fuse libfuse-dev

# Fedora/RHEL
sudo dnf install fuse fuse-devel

# Arch Linux
sudo pacman -S fuse2
```

#### Build from Source

```bash
git clone https://github.com/jonazri/task-fuse.git
cd task-fuse
go build -o task-fuse ./cmd/task-fuse

# Optional: install to PATH
sudo mv task-fuse /usr/local/bin/
```

## Usage

```bash
# Mount a tasks.md file
task-fuse mount tasks.md /mnt/tasks

# Run in foreground (for debugging)
task-fuse mount --foreground tasks.md /mnt/tasks

# Claim a task (atomic operation)
mv /mnt/tasks/pending/1.2.implement_parser.md /mnt/tasks/doing/

# Queue a task for later
mv /mnt/tasks/pending/1.3.write_docs.md /mnt/tasks/queued/

# Complete a task
mv /mnt/tasks/doing/1.2.implement_parser.md /mnt/tasks/done/

# Mark a task as failed
mv /mnt/tasks/doing/1.4.broken_feature.md /mnt/tasks/failed/

# Unmount
task-fuse unmount /mnt/tasks

# Check status
task-fuse status /mnt/tasks
```

### CLI Options

```
task-fuse mount [options] <tasks.md> <mountpoint>

Options:
  --foreground, -f    Run in foreground (default: daemonize)
  --log-level LEVEL   Set log level: error, warn, info, debug (default: info)
  --log-file PATH     Write logs to file instead of stderr
  --sync-interval MS  Polling interval in ms if file watching fails (default: 1000)
```

## Task Format

Tasks in `tasks.md` use checkbox syntax with numeric IDs:

```markdown
- [ ] 1. Setup project
  - [x] 1.1 Create structure
  - [-] 1.2 Configure build
  - [~] 1.3 Queued for review
- [ ] 2. Implement feature
```

| Checkbox | Status  |
|----------|---------|
| `- [ ]`  | pending |
| `- [~]`  | queued  |
| `- [-]`  | doing   |
| `- [x]`  | done    |
| `- [!]`  | failed  |

### Valid Status Transitions

- `pending` → `queued`, `doing`
- `queued` → `doing`, `pending`
- `doing` → `done`, `pending`, `failed`
- `done` → `doing`
- `failed` → `pending`, `doing`

Parent tasks (those with sub-tasks) cannot be moved directly; their status is derived from their children.

## License

MIT
