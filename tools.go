//go:build tools
// +build tools

// Package tools declares tool dependencies that are not directly imported
// but are required for the project. This is a common Go idiom to ensure
// dependencies are tracked in go.mod even when not directly imported.
package tools

import (
	// FUSE filesystem library for exposing tasks as files
	_ "bazil.org/fuse"
	_ "bazil.org/fuse/fs"

	// File system notification library for watching tasks.md changes
	_ "github.com/fsnotify/fsnotify"
)
