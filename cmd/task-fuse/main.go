// Package main provides the entry point for the task-fuse CLI application.
// task-fuse is a FUSE-based task manager filesystem that enables multiple AI agents
// to work on Kiro PRD tasks.md files concurrently without race conditions.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	fuseImpl "task-fuse/internal/fuse"
	"task-fuse/internal/logging"
	"task-fuse/internal/parser"
	"task-fuse/internal/printer"
	"task-fuse/internal/store"
	"task-fuse/internal/sync"
)

// Version information
const (
	Version = "0.1.0"
)

// Exit codes per design specification
const (
	ExitSuccess     = 0
	ExitError       = 1
	ExitMountFailed = 2
	ExitParseError  = 3
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(ExitError)
	}

	command := os.Args[1]
	switch command {
	case "mount":
		os.Exit(runMount(os.Args[2:]))
	case "unmount":
		os.Exit(runUnmount(os.Args[2:]))
	case "status":
		os.Exit(runStatus(os.Args[2:]))
	case "version":
		fmt.Printf("task-fuse version %s\n", Version)
		os.Exit(ExitSuccess)
	case "help", "-h", "--help":
		printUsage()
		os.Exit(ExitSuccess)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(ExitError)
	}
}

func printUsage() {
	fmt.Println(`task-fuse - FUSE Task Filesystem

Usage:
  task-fuse <command> [options] [arguments]

Commands:
  mount     Mount a tasks.md file as a FUSE filesystem
  unmount   Unmount a previously mounted filesystem
  status    Show status of mounted filesystems
  version   Show version information
  help      Show this help message

Use "task-fuse <command> --help" for more information about a command.`)
}

// runMount implements the mount command.
//
// Usage: task-fuse mount [options] <tasks.md> <mountpoint>
//
// Validates: Design - Mount Command
func runMount(args []string) int {
	mountFlags := flag.NewFlagSet("mount", flag.ExitOnError)
	foreground := mountFlags.Bool("foreground", false, "Run in foreground (default: daemonize)")
	mountFlags.BoolVar(foreground, "f", false, "Run in foreground (shorthand)")
	logLevel := mountFlags.String("log-level", "info", "Set log level: error, warn, info, debug")
	logFormat := mountFlags.String("log-format", "text", "Set log format: text, json")
	logFile := mountFlags.String("log-file", "", "Write logs to file instead of stderr")
	syncInterval := mountFlags.Int("sync-interval", 1000, "Polling interval in ms if file watching fails")

	mountFlags.Usage = func() {
		fmt.Println(`Usage: task-fuse mount [options] <tasks.md> <mountpoint>

Arguments:
  tasks.md      Path to the tasks.md file to mount
  mountpoint    Directory to mount the filesystem on (must exist and be empty)

Options:`)
		mountFlags.PrintDefaults()
	}

	if err := mountFlags.Parse(args); err != nil {
		return ExitError
	}

	if mountFlags.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "Error: tasks.md and mountpoint arguments are required")
		mountFlags.Usage()
		return ExitError
	}

	tasksFile := mountFlags.Arg(0)
	mountPoint := mountFlags.Arg(1)

	// Validate tasks.md exists
	if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: tasks.md file not found: %s\n", tasksFile)
		return ExitError
	}

	// Validate mountpoint exists and is a directory
	info, err := os.Stat(mountPoint)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: mountpoint does not exist: %s\n", mountPoint)
		} else {
			fmt.Fprintf(os.Stderr, "Error: cannot access mountpoint: %v\n", err)
		}
		return ExitMountFailed
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: mountpoint is not a directory: %s\n", mountPoint)
		return ExitMountFailed
	}

	// Configure logging
	logLevelParsed, err := logging.ParseLevel(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitError
	}

	logFormatParsed, err := logging.ParseFormat(*logFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitError
	}

	logger, logFileHandle, err := logging.NewLoggerFromFile(logLevelParsed, logFormatParsed, *logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return ExitError
	}
	if logFileHandle != nil {
		defer logFileHandle.Close()
	}

	// Set as default logger for the application
	logging.SetDefault(logger)

	// Create component-specific loggers
	mainLogger := logger.WithComponent("main")
	mainLogger.Info("starting task-fuse",
		"tasks_file", tasksFile,
		"mount_point", mountPoint,
		"log_level", *logLevel,
		"log_format", *logFormat,
	)

	// Parse the tasks.md file
	mainLogger.Debug("parsing tasks.md file")
	content, err := os.ReadFile(tasksFile)
	if err != nil {
		mainLogger.Error("failed to read tasks.md", "error", err)
		fmt.Fprintf(os.Stderr, "Error reading tasks.md: %v\n", err)
		return ExitParseError
	}

	p := parser.NewParser()
	result := p.Parse(string(content))
	if result == nil {
		mainLogger.Error("failed to parse tasks.md")
		fmt.Fprintln(os.Stderr, "Error: failed to parse tasks.md")
		return ExitParseError
	}
	mainLogger.Info("parsed tasks.md successfully",
		"root_tasks", len(result.Document.RootTasks),
	)

	// Create task store
	mainLogger.Debug("creating task store")
	taskStore := store.NewTaskStore()
	taskStore.Document = result.Document

	// Build paths
	pm := store.NewPathManager(taskStore)
	pm.BuildAllPaths()
	mainLogger.Debug("built task paths", "total_paths", len(taskStore.TasksByPath))

	// Create sync engine
	pr := printer.NewPrinter()
	syncEngine := sync.NewSyncEngine(taskStore, p, pr, tasksFile)
	syncEngine.SetPollInterval(time.Duration(*syncInterval) * time.Millisecond)
	mainLogger.Debug("created sync engine", "poll_interval_ms", *syncInterval)

	// Create FUSE filesystem
	mainLogger.Debug("creating FUSE filesystem")
	fuseFS := fuseImpl.NewFuseFS(taskStore, mountPoint)
	fuseFS.SetSyncEngine(syncEngine)

	// Mount the filesystem
	// Note: We don't use fuse.ReadOnly() because we need rename operations
	// for status transitions. Individual file writes are blocked at the
	// filesystem level (returning EPERM).
	mainLogger.Debug("mounting FUSE filesystem")
	c, err := fuse.Mount(
		mountPoint,
		fuse.FSName("task-fuse"),
		fuse.Subtype("taskfs"),
	)
	if err != nil {
		mainLogger.Error("failed to mount filesystem", "error", err)
		fmt.Fprintf(os.Stderr, "Error mounting filesystem: %v\n", err)
		return ExitMountFailed
	}
	defer c.Close()

	// Start file watching
	if err := syncEngine.StartWatching(); err != nil {
		mainLogger.Warn("failed to start file watching, falling back to polling",
			"error", err,
			"poll_interval_ms", *syncInterval,
		)
		fmt.Fprintf(os.Stderr, "Warning: failed to start file watching: %v\n", err)
	} else {
		mainLogger.Debug("file watching started")
	}

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run in foreground or background
	if *foreground {
		mainLogger.Info("running in foreground mode")
		fmt.Printf("Mounted %s at %s (foreground mode)\n", tasksFile, mountPoint)
		fmt.Println("Press Ctrl+C to unmount")

		// Serve filesystem in a goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- fs.Serve(c, fuseFS)
		}()

		// Wait for signal or error
		select {
		case sig := <-sigChan:
			mainLogger.Info("received signal, initiating shutdown", "signal", sig.String())
			fmt.Println("\nReceived signal, unmounting...")
		case err := <-errChan:
			if err != nil {
				mainLogger.Error("filesystem error", "error", err)
				fmt.Fprintf(os.Stderr, "Filesystem error: %v\n", err)
			}
		}

		// Graceful shutdown
		mainLogger.Info("performing graceful shutdown")
		syncEngine.GracefulShutdown(5 * time.Second)
		if err := fuse.Unmount(mountPoint); err != nil {
			mainLogger.Error("failed to unmount", "error", err)
		} else {
			mainLogger.Info("unmounted successfully")
		}
	} else {
		// Serve filesystem
		mainLogger.Info("running in background mode")
		fmt.Printf("Mounted %s at %s\n", tasksFile, mountPoint)
		if err := fs.Serve(c, fuseFS); err != nil {
			mainLogger.Error("filesystem error", "error", err)
			fmt.Fprintf(os.Stderr, "Filesystem error: %v\n", err)
			return ExitMountFailed
		}
	}

	mainLogger.Info("task-fuse shutdown complete")
	return ExitSuccess
}

// runUnmount implements the unmount command.
//
// Usage: task-fuse unmount [options] <mountpoint>
//
// Validates: Design - Unmount Command
func runUnmount(args []string) int {
	unmountFlags := flag.NewFlagSet("unmount", flag.ExitOnError)
	force := unmountFlags.Bool("force", false, "Force unmount even if busy")
	unmountFlags.BoolVar(force, "f", false, "Force unmount (shorthand)")
	timeout := unmountFlags.Int("timeout", 5, "Wait N seconds for pending operations")

	unmountFlags.Usage = func() {
		fmt.Println(`Usage: task-fuse unmount [options] <mountpoint>

Arguments:
  mountpoint    The mounted filesystem to unmount

Options:`)
		unmountFlags.PrintDefaults()
	}

	if err := unmountFlags.Parse(args); err != nil {
		return ExitError
	}

	if unmountFlags.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: mountpoint argument is required")
		unmountFlags.Usage()
		return ExitError
	}

	mountPoint := unmountFlags.Arg(0)
	_ = timeout // Would be used for graceful shutdown

	slog.Info("unmounting filesystem", "mount_point", mountPoint)

	// Unmount the filesystem
	if err := fuse.Unmount(mountPoint); err != nil {
		if *force {
			// Try force unmount (platform-specific)
			slog.Warn("force unmount not fully implemented", "error", err)
			fmt.Fprintf(os.Stderr, "Warning: force unmount not fully implemented: %v\n", err)
		} else {
			slog.Error("failed to unmount", "error", err)
			fmt.Fprintf(os.Stderr, "Error unmounting: %v\n", err)
			return ExitError
		}
	}

	slog.Info("unmounted successfully", "mount_point", mountPoint)
	fmt.Printf("Unmounted %s\n", mountPoint)
	return ExitSuccess
}

// runStatus implements the status command.
//
// Usage: task-fuse status [mountpoint]
//
// Validates: Design - Status Command
func runStatus(args []string) int {
	statusFlags := flag.NewFlagSet("status", flag.ExitOnError)

	statusFlags.Usage = func() {
		fmt.Println(`Usage: task-fuse status [mountpoint]

Arguments:
  mountpoint    (Optional) Check specific mountpoint, or list all if omitted

Output:
  Shows status of task-fuse mounted filesystems including task counts.`)
	}

	if err := statusFlags.Parse(args); err != nil {
		return ExitError
	}

	if statusFlags.NArg() > 0 {
		// Check specific mountpoint
		mountPoint := statusFlags.Arg(0)
		return showMountPointStatus(mountPoint)
	}

	// List all task-fuse mounts by reading /proc/mounts (Linux) or /etc/mtab
	mounts := findTaskFuseMounts()
	if len(mounts) == 0 {
		fmt.Println("No task-fuse filesystems currently mounted.")
		return ExitSuccess
	}

	fmt.Println("Mounted task-fuse filesystems:")
	fmt.Println()
	for _, mp := range mounts {
		showMountPointStatus(mp)
		fmt.Println()
	}
	return ExitSuccess
}

// showMountPointStatus shows status for a specific mountpoint.
func showMountPointStatus(mountPoint string) int {
	// Check if mountpoint exists and is accessible
	info, err := os.Stat(mountPoint)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: mountpoint does not exist: %s\n", mountPoint)
		} else {
			fmt.Fprintf(os.Stderr, "Error: cannot access mountpoint: %v\n", err)
		}
		return ExitError
	}

	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: not a directory: %s\n", mountPoint)
		return ExitError
	}

	// Try to read the index.md file to verify it's a task-fuse mount
	indexPath := mountPoint + "/index.md"
	content, err := os.ReadFile(indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot read index.md (is this a task-fuse mount?): %v\n", err)
		return ExitError
	}

	fmt.Printf("Mountpoint: %s\n", mountPoint)

	// Parse task counts from index.md content
	// Format: "- pending/: N tasks"
	lines := string(content)
	fmt.Println("  Status:")
	if idx := findTaskCount(lines, "pending/"); idx >= 0 {
		fmt.Printf("    Pending: %d\n", idx)
	}
	if idx := findTaskCount(lines, "queued/"); idx >= 0 {
		fmt.Printf("    Queued:  %d\n", idx)
	}
	if idx := findTaskCount(lines, "doing/"); idx >= 0 {
		fmt.Printf("    Doing:   %d\n", idx)
	}
	if idx := findTaskCount(lines, "done/"); idx >= 0 {
		fmt.Printf("    Done:    %d\n", idx)
	}
	if idx := findTaskCount(lines, "failed/"); idx >= 0 {
		fmt.Printf("    Failed:  %d\n", idx)
	}

	return ExitSuccess
}

// findTaskCount extracts the task count from an index.md line like "- pending/: 5 tasks".
func findTaskCount(content, status string) int {
	// Look for pattern "- {status}: N tasks"
	pattern := "- " + status + ": "
	idx := 0
	for _, line := range splitLines(content) {
		if len(line) > len(pattern) && line[:len(pattern)] == pattern {
			// Extract number
			rest := line[len(pattern):]
			n := 0
			for _, ch := range rest {
				if ch >= '0' && ch <= '9' {
					n = n*10 + int(ch-'0')
				} else {
					break
				}
			}
			return n
		}
		idx++
	}
	return -1
}

// splitLines splits content into lines.
func splitLines(content string) []string {
	var lines []string
	start := 0
	for i, ch := range content {
		if ch == '\n' {
			lines = append(lines, content[start:i])
			start = i + 1
		}
	}
	if start < len(content) {
		lines = append(lines, content[start:])
	}
	return lines
}

// findTaskFuseMounts reads /proc/mounts to find task-fuse mountpoints.
// Returns an empty slice on non-Linux systems or if no mounts are found.
func findTaskFuseMounts() []string {
	var mounts []string

	// Try to read /proc/mounts (Linux)
	content, err := os.ReadFile("/proc/mounts")
	if err != nil {
		// Not on Linux or /proc not available
		return mounts
	}

	// Parse mount entries, looking for fuse.taskfs
	for _, line := range splitLines(string(content)) {
		fields := splitFields(line)
		if len(fields) >= 3 {
			// Format: device mountpoint fstype options...
			if fields[2] == "fuse.taskfs" {
				mounts = append(mounts, fields[1])
			}
		}
	}

	return mounts
}

// splitFields splits a line by whitespace into fields.
func splitFields(line string) []string {
	var fields []string
	start := -1
	for i, ch := range line {
		if ch == ' ' || ch == '\t' {
			if start >= 0 {
				fields = append(fields, line[start:i])
				start = -1
			}
		} else {
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		fields = append(fields, line[start:])
	}
	return fields
}
