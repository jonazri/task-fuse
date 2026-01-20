// Package fuse provides the FUSE filesystem implementation that exposes
// tasks as files and directories organized by status.
package fuse

import (
	"context"
	"hash/fnv"
	"os"
	"strings"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"task-fuse/internal/store"
)

// EROFS is the Read-only file system error code.
// This is used for Setattr operations since the filesystem is read-only
// except for status transitions via rename.
const EROFS = fuse.Errno(30)

// StatusDirs contains all valid status directory names.
// These are the top-level directories under the mount root.
var StatusDirs = []store.TaskStatus{
	store.StatusPending,
	store.StatusQueued,
	store.StatusDoing,
	store.StatusDone,
	store.StatusFailed,
}

// statusDirSet is a set for quick lookup of valid status directory names.
var statusDirSet = map[string]store.TaskStatus{
	"pending": store.StatusPending,
	"queued":  store.StatusQueued,
	"doing":   store.StatusDoing,
	"done":    store.StatusDone,
	"failed":  store.StatusFailed,
}

// Ensure FuseFS implements the fs.FS interface.
var _ fs.FS = (*FuseFS)(nil)

// FuseFS implements the FUSE filesystem that exposes tasks as files
// and directories organized by status.
//
// The filesystem structure is:
//
//	/                           → Root (list status dirs)
//	/index.md                   → Root index file
//	/pending                    → Status directory
//	/pending/index.md           → Status index file
//	/pending/1.task.md          → Leaf task file
//	/pending/2.parent/          → Parent task directory
//	/pending/2.parent/index.md  → Parent task index
//	/pending/2.parent/2.1.child.md → Nested leaf task
type FuseFS struct {
	// store is a reference to the TaskStore that holds all task data.
	// The store provides concurrent access support via read-write locks.
	store *store.TaskStore

	// syncEngine is a reference to the SyncEngine for bidirectional
	// synchronization between the filesystem and tasks.md.
	// This is stored as interface{} since the sync package is not yet implemented.
	// It will be properly typed when the SyncEngine is implemented in Tasks 10-12.
	syncEngine interface{}

	// mountPoint is the path where the filesystem is mounted.
	mountPoint string
}

// NewFuseFS creates a new FuseFS instance with the given TaskStore and mount point.
// The SyncEngine can be set later via SetSyncEngine() if not available at creation time.
//
// Parameters:
//   - store: The TaskStore containing all task data (required)
//   - mountPoint: The path where the filesystem will be mounted (required)
//
// Returns a new FuseFS instance ready to be mounted.
func NewFuseFS(store *store.TaskStore, mountPoint string) *FuseFS {
	return &FuseFS{
		store:      store,
		syncEngine: nil,
		mountPoint: mountPoint,
	}
}

// SetSyncEngine sets the SyncEngine for bidirectional synchronization.
// This can be called after NewFuseFS() if the SyncEngine is not available
// at filesystem creation time.
// The engine parameter is interface{} since the sync package is not yet implemented.
func (f *FuseFS) SetSyncEngine(engine interface{}) {
	f.syncEngine = engine
}

// SyncEngine returns the SyncEngine associated with this filesystem.
// Returns nil if no SyncEngine has been set.
func (f *FuseFS) SyncEngine() interface{} {
	return f.syncEngine
}

// Store returns the TaskStore associated with this filesystem.
// This is useful for operations that need direct access to the store.
func (f *FuseFS) Store() *store.TaskStore {
	return f.store
}

// MountPoint returns the path where the filesystem is mounted.
func (f *FuseFS) MountPoint() string {
	return f.mountPoint
}

// Root returns the root directory node of the filesystem.
// This implements the fs.FS interface from bazil.org/fuse/fs.
//
// The root directory contains:
//   - index.md: Root index file with status counts
//   - pending/: Status directory for pending tasks
//   - queued/: Status directory for queued tasks
//   - doing/: Status directory for tasks in progress
//   - done/: Status directory for completed tasks
//   - failed/: Status directory for failed tasks
func (f *FuseFS) Root() (fs.Node, error) {
	return &Dir{
		fs:   f,
		path: "/",
	}, nil
}

// Dir represents a directory node in the FUSE filesystem.
// This can be:
//   - The root directory (path="/")
//   - A status directory (path="/pending", "/doing", etc.)
//   - A parent task directory (path="/pending/1.parent_task")
type Dir struct {
	// fs is a reference to the parent FuseFS instance.
	fs *FuseFS

	// path is the full path of this directory from the mount root.
	// Examples: "/", "/pending", "/pending/1.setup_project"
	path string
}

// Path returns the full path of this directory.
func (d *Dir) Path() string {
	return d.path
}

// FS returns the parent FuseFS instance.
func (d *Dir) FS() *FuseFS {
	return d.fs
}

// Attr sets the attributes for this directory.
// Directories have read and execute permissions (0555).
//
// Implements fs.Node interface.
// Validates: Requirements 2.1
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	// Set directory mode with read and execute permissions
	a.Mode = os.ModeDir | 0555

	// Generate a unique inode based on the path hash
	a.Inode = hashPath(d.path)

	// Standard link count for directories
	a.Nlink = 2

	return nil
}

// hashPath generates a unique inode number from a path string.
// Uses FNV-1a hash for good distribution.
func hashPath(path string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(path))
	return h.Sum64()
}

// Lookup looks up a child node by name within this directory.
//
// For root directory ("/"):
//   - If name is "index.md", return a File node for root index
//   - If name is one of "pending", "queued", "doing", "done", "failed", return a Dir node
//   - Otherwise return fuse.ENOENT
//
// For status directories ("/pending", etc.):
//   - If name is "index.md", return a File node for status index
//   - Look up tasks in the store that are in this status directory
//   - If name matches a task filename, return appropriate node (Dir for parent tasks, File for leaf tasks)
//   - Otherwise return fuse.ENOENT
//
// For task directories ("/pending/1.parent_task"):
//   - If name is "index.md", return a File node for task index
//   - Look up child tasks
//   - If name matches a child task filename, return appropriate node
//   - Otherwise return fuse.ENOENT
//
// Implements fs.NodeStringLookuper interface.
// Validates: Requirements 2.1, 2.2, 2.3, 2.4
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	// Handle index.md for any directory
	if name == "index.md" {
		indexPath := d.path + "/" + name
		if d.path == "/" {
			indexPath = "/" + name
		}
		return &File{
			fs:   d.fs,
			path: indexPath,
			task: nil, // index.md files have no associated task
		}, nil
	}

	// Root directory lookup
	if d.path == "/" {
		return d.lookupInRoot(name)
	}

	// Status directory lookup (e.g., "/pending")
	if isStatusDir(d.path) {
		return d.lookupInStatusDir(name)
	}

	// Task directory lookup (e.g., "/pending/1.parent_task")
	return d.lookupInTaskDir(name)
}

// lookupInRoot handles lookup in the root directory.
// Returns status directories by name.
func (d *Dir) lookupInRoot(name string) (fs.Node, error) {
	// Check if name is a valid status directory
	if _, ok := statusDirSet[name]; ok {
		return &Dir{
			fs:   d.fs,
			path: "/" + name,
		}, nil
	}

	return nil, fuse.ENOENT
}

// lookupInDir handles lookup in any directory containing tasks.
// Returns task files or directories that are direct children of this directory.
// Used for both status directories and task directories.
func (d *Dir) lookupInDir(name string) (fs.Node, error) {
	// Acquire read lock on the store
	d.fs.store.RLock()
	defer d.fs.store.RUnlock()

	// Build the full path for the requested item
	fullPath := d.path + "/" + name

	// Look up in the store's TasksByPath
	entry := d.fs.store.TasksByPath[fullPath]
	if entry == nil {
		return nil, fuse.ENOENT
	}

	// Return Dir for parent tasks (have children), File for leaf tasks
	if len(entry.Task.Children) > 0 {
		return &Dir{
			fs:   d.fs,
			path: fullPath,
		}, nil
	}

	return &File{
		fs:   d.fs,
		path: fullPath,
		task: entry.Task,
	}, nil
}

// lookupInStatusDir handles lookup in a status directory.
// Returns task files or directories that are direct children of this status.
func (d *Dir) lookupInStatusDir(name string) (fs.Node, error) {
	return d.lookupInDir(name)
}

// lookupInTaskDir handles lookup in a task directory (parent task).
// Returns child task files or directories.
func (d *Dir) lookupInTaskDir(name string) (fs.Node, error) {
	return d.lookupInDir(name)
}

// isStatusDir checks if a path is a status directory (e.g., "/pending").
func isStatusDir(path string) bool {
	// Status directories are at the first level: "/pending", "/doing", etc.
	if !strings.HasPrefix(path, "/") {
		return false
	}

	// Remove leading slash and check if it's a valid status
	name := strings.TrimPrefix(path, "/")

	// Should not contain any more slashes (direct child of root)
	if strings.Contains(name, "/") {
		return false
	}

	_, ok := statusDirSet[name]
	return ok
}

// ReadDirAll returns all directory entries in this directory.
//
// For root directory:
//   - Return entries for: index.md, pending, queued, doing, done, failed
//
// For status directories:
//   - Return entry for index.md
//   - Return entries for all tasks in this status directory
//
// For task directories:
//   - Return entry for index.md
//   - Return entries for all child tasks
//
// Implements fs.HandleReadDirAller interface.
// Validates: Requirements 2.1
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	// Root directory
	if d.path == "/" {
		return d.readDirRoot()
	}

	// Status directory (e.g., "/pending")
	if isStatusDir(d.path) {
		return d.readDirStatus()
	}

	// Task directory (e.g., "/pending/1.parent_task")
	return d.readDirTask()
}

// readDirRoot returns directory entries for the root directory.
// Contains: index.md and all status directories.
func (d *Dir) readDirRoot() ([]fuse.Dirent, error) {
	entries := make([]fuse.Dirent, 0, len(StatusDirs)+1)

	// Add index.md file
	entries = append(entries, fuse.Dirent{
		Inode: hashPath(d.path + "/index.md"),
		Type:  fuse.DT_File,
		Name:  "index.md",
	})

	// Add status directories
	for _, status := range StatusDirs {
		entries = append(entries, fuse.Dirent{
			Inode: hashPath("/" + string(status)),
			Type:  fuse.DT_Dir,
			Name:  string(status),
		})
	}

	return entries, nil
}

// readDirWithTasks returns directory entries for any directory containing tasks.
// Contains: index.md and all direct child tasks.
// Used for both status directories and task directories.
func (d *Dir) readDirWithTasks() ([]fuse.Dirent, error) {
	entries := make([]fuse.Dirent, 0)

	// Add index.md file
	entries = append(entries, fuse.Dirent{
		Inode: hashPath(d.path + "/index.md"),
		Type:  fuse.DT_File,
		Name:  "index.md",
	})

	// Acquire read lock on the store
	d.fs.store.RLock()
	defer d.fs.store.RUnlock()

	// Find all tasks that are direct children of this directory
	// by iterating through TasksByPath and finding entries that match
	prefix := d.path + "/"
	for path, entry := range d.fs.store.TasksByPath {
		// Check if this path is a direct child of this directory
		if !strings.HasPrefix(path, prefix) {
			continue
		}

		// Extract the filename (everything after the prefix)
		filename := path[len(prefix):]

		// Skip if this is a nested path (contains more slashes)
		if strings.Contains(filename, "/") {
			continue
		}

		// Determine if this is a directory (parent task) or file (leaf task)
		entryType := fuse.DT_File
		if len(entry.Task.Children) > 0 {
			entryType = fuse.DT_Dir
		}

		entries = append(entries, fuse.Dirent{
			Inode: hashPath(path),
			Type:  entryType,
			Name:  filename,
		})
	}

	return entries, nil
}

// readDirStatus returns directory entries for a status directory.
// Contains: index.md and all tasks with this status.
func (d *Dir) readDirStatus() ([]fuse.Dirent, error) {
	return d.readDirWithTasks()
}

// readDirTask returns directory entries for a task directory (parent task).
// Contains: index.md and all child tasks.
func (d *Dir) readDirTask() ([]fuse.Dirent, error) {
	return d.readDirWithTasks()
}

// File represents a file node in the FUSE filesystem.
// This can be:
//   - A leaf task file (e.g., "1.1.create_structure.md")
//   - An index.md file (in any directory)
type File struct {
	// fs is a reference to the parent FuseFS instance.
	fs *FuseFS

	// path is the full path of this file from the mount root.
	// Examples: "/index.md", "/pending/1.1.task_name.md"
	path string

	// task is a reference to the Task this file represents.
	// This is nil for index.md files.
	task *store.Task
}

// Path returns the full path of this file.
func (f *File) Path() string {
	return f.path
}

// FS returns the parent FuseFS instance.
func (f *File) FS() *FuseFS {
	return f.fs
}

// Task returns the Task this file represents.
// Returns nil for index.md files.
func (f *File) Task() *store.Task {
	return f.task
}

// Attr sets the attributes for this file.
// Files have read-only permissions (0444).
//
// For task files (task != nil):
//   - Size is the length of the generated task content
//
// For index.md files (task == nil):
//   - Size is the length of the generated index content
//
// Implements fs.Node interface.
// Validates: Requirements 2.2, 2.9, 3.4, 3.5, 3.6, 3.7, 3.8
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	// Set file mode with read-only permissions (0444)
	a.Mode = 0444

	// Generate a unique inode based on the path hash
	a.Inode = hashPath(f.path)

	// Standard link count for files
	a.Nlink = 1

	// Calculate file size based on content
	if f.task != nil {
		// Task file: size is the length of generated content
		content := generateTaskContent(f.task)
		a.Size = uint64(len(content))
	} else {
		// Index file: generate content to get size
		content := f.generateIndexContent()
		a.Size = uint64(len(content))
	}

	return nil
}

// Read reads data from this file.
//
// For task files (task != nil):
//   - Generate content using Task File Content Format:
//     # {task_id}. {title}
//     **Status:** {status}
//     {description lines}
//
// For index.md files (task == nil):
//   - Generate appropriate index content based on directory type
//
// Handles offset and size from the request:
//   - req.Offset: starting position in the file
//   - req.Size: maximum bytes to read
//
// Implements fs.HandleReader interface.
// Validates: Requirements 2.2, 2.9, 3.4, 3.5, 3.6, 3.7, 3.8, 3.9
func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	var content string

	if f.task != nil {
		// Task file: generate task content
		content = generateTaskContent(f.task)
	} else {
		// Index file: generate index content based on directory type
		content = f.generateIndexContent()
	}

	// Handle offset and size
	contentBytes := []byte(content)
	contentLen := int64(len(contentBytes))

	// If offset is beyond content length, return empty data
	if req.Offset >= contentLen {
		resp.Data = []byte{}
		return nil
	}

	// Calculate the end position
	end := req.Offset + int64(req.Size)
	if end > contentLen {
		end = contentLen
	}

	// Set response data to the appropriate slice
	resp.Data = contentBytes[req.Offset:end]

	return nil
}

// generateIndexContent generates the appropriate index content based on the file's path.
// - /index.md: Root index with status counts
// - /{status}/index.md: Status directory index with task list
// - /{status}/{task}/index.md: Task directory index with description and sub-tasks
//
// Validates: Requirements 3.4, 3.5, 3.6, 3.7, 3.8, 3.9
func (f *File) generateIndexContent() string {
	ig := NewIndexGenerator(f.fs.store)

	// Determine the type of index based on path
	// Path format: /index.md, /pending/index.md, /pending/1.task/index.md

	// Remove the /index.md suffix to get the directory path
	dirPath := strings.TrimSuffix(f.path, "/index.md")

	// Root index
	if dirPath == "" {
		return ig.GenerateRootIndex()
	}

	// Status directory index (e.g., /pending)
	if isStatusDir(dirPath) {
		statusName := strings.TrimPrefix(dirPath, "/")
		status, ok := statusDirSet[statusName]
		if ok {
			return ig.GenerateStatusIndex(status)
		}
	}

	// Task directory index - look up the task from the store
	f.fs.store.RLock()
	defer f.fs.store.RUnlock()

	entry := f.fs.store.TasksByPath[dirPath]
	if entry != nil && entry.Task != nil {
		return ig.GenerateTaskIndex(entry.Task)
	}

	// Fallback: empty content
	return ""
}

// generateTaskContent generates content for a leaf task file.
// Format follows Task File Content Format specification in requirements.md:
//
//	# {task_id}. {title}
//
//	**Status:** {status}
//
//	{description lines, preserved exactly as in tasks.md}
func generateTaskContent(task *store.Task) string {
	content := "# " + task.ID + ". " + task.Title + "\n\n"
	content += "**Status:** " + string(task.Status) + "\n\n"

	for _, line := range task.Description {
		content += line.RawContent + "\n"
	}

	return content
}


// =============================================================================
// Read-Only Enforcement Methods
// =============================================================================
// These methods implement read-only enforcement for the filesystem.
// All mutating operations return EPERM (Operation not permitted) or EROFS
// (Read-only file system) as appropriate.
//
// Validates: Requirements 3.1, 3.2, 3.3

// Write rejects write operations on task files and index files.
// Returns EPERM (Operation not permitted).
//
// Implements fs.HandleWriter interface.
// Validates: Requirements 3.1
func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	return fuse.EPERM
}

// Create rejects file creation in all directories.
// Returns EPERM (Operation not permitted).
//
// Implements fs.NodeCreater interface.
// Validates: Requirements 3.2
func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	return nil, nil, fuse.EPERM
}

// Remove rejects file and directory deletion.
// Returns EPERM (Operation not permitted).
//
// Implements fs.NodeRemover interface.
// Validates: Requirements 3.3
func (d *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	return fuse.EPERM
}

// Mkdir rejects directory creation.
// Returns EPERM (Operation not permitted).
//
// Implements fs.NodeMkdirer interface.
// Validates: Design - All mutating operations return EPERM or EROFS
func (d *Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	return nil, fuse.EPERM
}

// Setattr rejects attribute modification on files.
// Returns EROFS (Read-only file system).
//
// Implements fs.NodeSetattrer interface.
// Validates: Design - All mutating operations return EPERM or EROFS
func (f *File) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	return EROFS
}

// Setattr rejects attribute modification on directories.
// Returns EROFS (Read-only file system).
//
// Implements fs.NodeSetattrer interface.
// Validates: Design - All mutating operations return EPERM or EROFS
func (d *Dir) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	return EROFS
}

// Symlink rejects symbolic link creation.
// Returns EPERM (Operation not permitted).
//
// Implements fs.NodeSymlinker interface.
func (d *Dir) Symlink(ctx context.Context, req *fuse.SymlinkRequest) (fs.Node, error) {
	return nil, fuse.EPERM
}

// Link rejects hard link creation.
// Returns EPERM (Operation not permitted).
//
// Implements fs.NodeLinker interface.
func (d *Dir) Link(ctx context.Context, req *fuse.LinkRequest, old fs.Node) (fs.Node, error) {
	return nil, fuse.EPERM
}

// Mknod rejects special file creation.
// Returns EPERM (Operation not permitted).
//
// Implements fs.NodeMknoder interface.
func (d *Dir) Mknod(ctx context.Context, req *fuse.MknodRequest) (fs.Node, error) {
	return nil, fuse.EPERM
}

// Fsync is a no-op for read-only files.
// Returns nil (success) since there's nothing to sync.
//
// Implements fs.HandleFsyncer interface.
func (f *File) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	return nil
}

// Flush is a no-op for read-only files.
// Returns nil (success) since there's nothing to flush.
//
// Implements fs.HandleFlusher interface.
func (f *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	return nil
}
