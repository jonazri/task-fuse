// Package main provides the entry point for the task-fuse CLI application.
// task-fuse is a FUSE-based task manager filesystem that enables multiple AI agents
// to work on Kiro PRD tasks.md files concurrently without race conditions.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	fuseImpl "task-fuse/internal/fuse"
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
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: mountpoint does not exist: %s\n", mountPoint)
		return ExitMountFailed
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: mountpoint is not a directory: %s\n", mountPoint)
		return ExitMountFailed
	}

	// Log configuration (placeholder - would use proper logging library)
	_ = logLevel
	_ = logFormat
	_ = logFile

	// Parse the tasks.md file
	content, err := os.ReadFile(tasksFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading tasks.md: %v\n", err)
		return ExitParseError
	}

	p := parser.NewParser()
	result := p.Parse(string(content))
	if result == nil {
		fmt.Fprintln(os.Stderr, "Error: failed to parse tasks.md")
		return ExitParseError
	}

	// Create task store
	taskStore := store.NewTaskStore()
	taskStore.Document = result.Document

	// Build paths
	pm := store.NewPathManager(taskStore)
	pm.BuildAllPaths()

	// Create sync engine
	pr := printer.NewPrinter()
	syncEngine := sync.NewSyncEngine(taskStore, p, pr, tasksFile)
	syncEngine.SetPollInterval(time.Duration(*syncInterval) * time.Millisecond)

	// Create FUSE filesystem
	fuseFS := fuseImpl.NewFuseFS(taskStore, mountPoint)
	fuseFS.SetSyncEngine(syncEngine)

	// Mount the filesystem
	c, err := fuse.Mount(
		mountPoint,
		fuse.FSName("task-fuse"),
		fuse.Subtype("taskfs"),
		fuse.ReadOnly(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error mounting filesystem: %v\n", err)
		return ExitMountFailed
	}
	defer c.Close()

	// Start file watching
	if err := syncEngine.StartWatching(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to start file watching: %v\n", err)
	}

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run in foreground or background
	if *foreground {
		fmt.Printf("Mounted %s at %s (foreground mode)\n", tasksFile, mountPoint)
		fmt.Println("Press Ctrl+C to unmount")

		// Serve filesystem in a goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- fs.Serve(c, fuseFS)
		}()

		// Wait for signal or error
		select {
		case <-sigChan:
			fmt.Println("\nReceived signal, unmounting...")
		case err := <-errChan:
			if err != nil {
				fmt.Fprintf(os.Stderr, "Filesystem error: %v\n", err)
			}
		}

		// Graceful shutdown
		syncEngine.GracefulShutdown(5 * time.Second)
		fuse.Unmount(mountPoint)
	} else {
		// Serve filesystem
		fmt.Printf("Mounted %s at %s\n", tasksFile, mountPoint)
		if err := fs.Serve(c, fuseFS); err != nil {
			fmt.Fprintf(os.Stderr, "Filesystem error: %v\n", err)
			return ExitMountFailed
		}
	}

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

	// Unmount the filesystem
	if err := fuse.Unmount(mountPoint); err != nil {
		if *force {
			// Try force unmount (platform-specific)
			fmt.Fprintf(os.Stderr, "Warning: force unmount not fully implemented: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error unmounting: %v\n", err)
			return ExitError
		}
	}

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
  mountpoint    (Optional) Check specific mountpoint, or list all if omitted`)
	}

	if err := statusFlags.Parse(args); err != nil {
		return ExitError
	}

	// For now, just print a placeholder message
	// A full implementation would track mounted filesystems
	if statusFlags.NArg() > 0 {
		mountPoint := statusFlags.Arg(0)
		fmt.Printf("Status for %s: not implemented\n", mountPoint)
	} else {
		fmt.Println("No mounted filesystems (status tracking not implemented)")
	}

	return ExitSuccess
}
