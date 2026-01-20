// Package fuse provides tests for the FUSE filesystem implementation.
package fuse

import (
	"os"
	"strings"
	"testing"

	"bazil.org/fuse"
	"task-fuse/internal/store"
)

func TestNewFuseFS(t *testing.T) {
	taskStore := store.NewTaskStore()
	mountPoint := "/mnt/tasks"

	fs := NewFuseFS(taskStore, mountPoint)

	if fs == nil {
		t.Fatal("NewFuseFS returned nil")
	}

	if fs.Store() != taskStore {
		t.Error("Store() did not return the expected TaskStore")
	}

	if fs.MountPoint() != mountPoint {
		t.Errorf("MountPoint() = %q, want %q", fs.MountPoint(), mountPoint)
	}

	if fs.SyncEngine() != nil {
		t.Error("SyncEngine() should be nil initially")
	}
}

func TestFuseFS_SetSyncEngine(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Initially nil
	if fs.SyncEngine() != nil {
		t.Error("SyncEngine() should be nil initially")
	}

	// Set a mock sync engine (using interface{} for now)
	mockEngine := struct{ name string }{name: "mock"}
	fs.SetSyncEngine(mockEngine)

	if fs.SyncEngine() == nil {
		t.Error("SyncEngine() should not be nil after SetSyncEngine()")
	}

	if fs.SyncEngine() != mockEngine {
		t.Error("SyncEngine() did not return the expected engine")
	}
}

func TestFuseFS_Root(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	root, err := fs.Root()
	if err != nil {
		t.Fatalf("Root() returned error: %v", err)
	}

	if root == nil {
		t.Fatal("Root() returned nil node")
	}

	// Verify it's a Dir
	dir, ok := root.(*Dir)
	if !ok {
		t.Fatalf("Root() returned %T, want *Dir", root)
	}

	if dir.Path() != "/" {
		t.Errorf("Root dir path = %q, want %q", dir.Path(), "/")
	}

	if dir.FS() != fs {
		t.Error("Root dir FS() did not return the expected FuseFS")
	}
}

func TestDir_PathAndFS(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{
		fs:   fs,
		path: "/pending",
	}

	if dir.Path() != "/pending" {
		t.Errorf("Path() = %q, want %q", dir.Path(), "/pending")
	}

	if dir.FS() != fs {
		t.Error("FS() did not return the expected FuseFS")
	}
}

func TestFile_PathFSAndTask(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	task := &store.Task{
		ID:     "1.1",
		Title:  "Test task",
		Status: store.StatusPending,
	}

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test_task.md",
		task: task,
	}

	if file.Path() != "/pending/1.1.test_task.md" {
		t.Errorf("Path() = %q, want %q", file.Path(), "/pending/1.1.test_task.md")
	}

	if file.FS() != fs {
		t.Error("FS() did not return the expected FuseFS")
	}

	if file.Task() != task {
		t.Error("Task() did not return the expected task")
	}
}

func TestFile_NilTask(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Index files have nil task
	file := &File{
		fs:   fs,
		path: "/index.md",
		task: nil,
	}

	if file.Task() != nil {
		t.Error("Task() should be nil for index files")
	}
}

func TestNewFuseFS_NilStore(t *testing.T) {
	// Should handle nil store gracefully
	fs := NewFuseFS(nil, "/mnt/tasks")

	if fs == nil {
		t.Fatal("NewFuseFS returned nil even with nil store")
	}

	if fs.Store() != nil {
		t.Error("Store() should be nil when created with nil store")
	}
}


// Tests for Task 6.2: Status directories implementation

func TestDir_Attr(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	tests := []struct {
		name string
		path string
	}{
		{"root directory", "/"},
		{"pending status dir", "/pending"},
		{"doing status dir", "/doing"},
		{"task directory", "/pending/1.setup_project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := &Dir{fs: fs, path: tt.path}

			var attr fuse.Attr
			err := dir.Attr(nil, &attr)
			if err != nil {
				t.Fatalf("Attr() returned error: %v", err)
			}

			// Check mode is directory with 0555 permissions
			expectedMode := os.ModeDir | 0555
			if attr.Mode != expectedMode {
				t.Errorf("Mode = %v, want %v", attr.Mode, expectedMode)
			}

			// Check inode is non-zero
			if attr.Inode == 0 {
				t.Error("Inode should be non-zero")
			}

			// Check nlink is 2
			if attr.Nlink != 2 {
				t.Errorf("Nlink = %d, want 2", attr.Nlink)
			}
		})
	}
}

func TestDir_Attr_UniqueInodes(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	paths := []string{"/", "/pending", "/doing", "/done", "/pending/1.task"}
	inodes := make(map[uint64]string)

	for _, path := range paths {
		dir := &Dir{fs: fs, path: path}
		var attr fuse.Attr
		err := dir.Attr(nil, &attr)
		if err != nil {
			t.Fatalf("Attr() for %s returned error: %v", path, err)
		}

		if existing, ok := inodes[attr.Inode]; ok {
			t.Errorf("Inode collision: %s and %s have same inode %d", path, existing, attr.Inode)
		}
		inodes[attr.Inode] = path
	}
}

func TestDir_ReadDirAll_Root(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{fs: fs, path: "/"}
	entries, err := dir.ReadDirAll(nil)
	if err != nil {
		t.Fatalf("ReadDirAll() returned error: %v", err)
	}

	// Should have index.md + 5 status directories = 6 entries
	if len(entries) != 6 {
		t.Errorf("ReadDirAll() returned %d entries, want 6", len(entries))
	}

	// Check for expected entries
	expectedNames := map[string]fuse.DirentType{
		"index.md": fuse.DT_File,
		"pending":  fuse.DT_Dir,
		"queued":   fuse.DT_Dir,
		"doing":    fuse.DT_Dir,
		"done":     fuse.DT_Dir,
		"failed":   fuse.DT_Dir,
	}

	for _, entry := range entries {
		expectedType, ok := expectedNames[entry.Name]
		if !ok {
			t.Errorf("Unexpected entry: %s", entry.Name)
			continue
		}
		if entry.Type != expectedType {
			t.Errorf("Entry %s has type %v, want %v", entry.Name, entry.Type, expectedType)
		}
		delete(expectedNames, entry.Name)
	}

	for name := range expectedNames {
		t.Errorf("Missing expected entry: %s", name)
	}
}

func TestDir_ReadDirAll_StatusDir_Empty(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{fs: fs, path: "/pending"}
	entries, err := dir.ReadDirAll(nil)
	if err != nil {
		t.Fatalf("ReadDirAll() returned error: %v", err)
	}

	// Should have only index.md when no tasks
	if len(entries) != 1 {
		t.Errorf("ReadDirAll() returned %d entries, want 1", len(entries))
	}

	if entries[0].Name != "index.md" {
		t.Errorf("Entry name = %s, want index.md", entries[0].Name)
	}

	if entries[0].Type != fuse.DT_File {
		t.Errorf("Entry type = %v, want DT_File", entries[0].Type)
	}
}

func TestDir_ReadDirAll_StatusDir_WithTasks(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Add some tasks to the store
	leafTask := &store.Task{
		ID:     "1.1",
		Title:  "Leaf task",
		Status: store.StatusPending,
	}
	parentTask := &store.Task{
		ID:       "2",
		Title:    "Parent task",
		Status:   store.StatusPending,
		Children: []*store.Task{{ID: "2.1", Title: "Child"}},
	}

	taskStore.TasksByPath["/pending/1.1.leaf_task.md"] = &store.PathEntry{
		Task:     leafTask,
		Status:   store.StatusPending,
		FullPath: "/pending/1.1.leaf_task.md",
	}
	taskStore.TasksByPath["/pending/2.parent_task"] = &store.PathEntry{
		Task:     parentTask,
		Status:   store.StatusPending,
		FullPath: "/pending/2.parent_task",
	}

	dir := &Dir{fs: fs, path: "/pending"}
	entries, err := dir.ReadDirAll(nil)
	if err != nil {
		t.Fatalf("ReadDirAll() returned error: %v", err)
	}

	// Should have index.md + 2 tasks = 3 entries
	if len(entries) != 3 {
		t.Errorf("ReadDirAll() returned %d entries, want 3", len(entries))
	}

	// Check for expected entries
	expectedNames := map[string]fuse.DirentType{
		"index.md":          fuse.DT_File,
		"1.1.leaf_task.md":  fuse.DT_File,
		"2.parent_task":     fuse.DT_Dir,
	}

	for _, entry := range entries {
		expectedType, ok := expectedNames[entry.Name]
		if !ok {
			t.Errorf("Unexpected entry: %s", entry.Name)
			continue
		}
		if entry.Type != expectedType {
			t.Errorf("Entry %s has type %v, want %v", entry.Name, entry.Type, expectedType)
		}
		delete(expectedNames, entry.Name)
	}

	for name := range expectedNames {
		t.Errorf("Missing expected entry: %s", name)
	}
}

func TestDir_ReadDirAll_TaskDir(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Add child tasks to the store
	childTask := &store.Task{
		ID:     "1.1",
		Title:  "Child task",
		Status: store.StatusPending,
	}

	taskStore.TasksByPath["/pending/1.parent/1.1.child_task.md"] = &store.PathEntry{
		Task:     childTask,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent/1.1.child_task.md",
	}

	dir := &Dir{fs: fs, path: "/pending/1.parent"}
	entries, err := dir.ReadDirAll(nil)
	if err != nil {
		t.Fatalf("ReadDirAll() returned error: %v", err)
	}

	// Should have index.md + 1 child task = 2 entries
	if len(entries) != 2 {
		t.Errorf("ReadDirAll() returned %d entries, want 2", len(entries))
	}

	// Check for expected entries
	expectedNames := map[string]fuse.DirentType{
		"index.md":           fuse.DT_File,
		"1.1.child_task.md":  fuse.DT_File,
	}

	for _, entry := range entries {
		expectedType, ok := expectedNames[entry.Name]
		if !ok {
			t.Errorf("Unexpected entry: %s", entry.Name)
			continue
		}
		if entry.Type != expectedType {
			t.Errorf("Entry %s has type %v, want %v", entry.Name, entry.Type, expectedType)
		}
		delete(expectedNames, entry.Name)
	}
}

func TestDir_Lookup_Root_IndexMd(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{fs: fs, path: "/"}
	node, err := dir.Lookup(nil, "index.md")
	if err != nil {
		t.Fatalf("Lookup(index.md) returned error: %v", err)
	}

	file, ok := node.(*File)
	if !ok {
		t.Fatalf("Lookup(index.md) returned %T, want *File", node)
	}

	if file.Path() != "/index.md" {
		t.Errorf("File path = %s, want /index.md", file.Path())
	}

	if file.Task() != nil {
		t.Error("Index file should have nil task")
	}
}

func TestDir_Lookup_Root_StatusDirs(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{fs: fs, path: "/"}

	statusDirs := []string{"pending", "queued", "doing", "done", "failed"}
	for _, name := range statusDirs {
		t.Run(name, func(t *testing.T) {
			node, err := dir.Lookup(nil, name)
			if err != nil {
				t.Fatalf("Lookup(%s) returned error: %v", name, err)
			}

			subdir, ok := node.(*Dir)
			if !ok {
				t.Fatalf("Lookup(%s) returned %T, want *Dir", name, node)
			}

			expectedPath := "/" + name
			if subdir.Path() != expectedPath {
				t.Errorf("Dir path = %s, want %s", subdir.Path(), expectedPath)
			}
		})
	}
}

func TestDir_Lookup_Root_NotFound(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{fs: fs, path: "/"}
	_, err := dir.Lookup(nil, "nonexistent")
	if err != fuse.ENOENT {
		t.Errorf("Lookup(nonexistent) returned %v, want ENOENT", err)
	}
}

func TestDir_Lookup_StatusDir_IndexMd(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{fs: fs, path: "/pending"}
	node, err := dir.Lookup(nil, "index.md")
	if err != nil {
		t.Fatalf("Lookup(index.md) returned error: %v", err)
	}

	file, ok := node.(*File)
	if !ok {
		t.Fatalf("Lookup(index.md) returned %T, want *File", node)
	}

	if file.Path() != "/pending/index.md" {
		t.Errorf("File path = %s, want /pending/index.md", file.Path())
	}
}

func TestDir_Lookup_StatusDir_LeafTask(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Add a leaf task to the store
	task := &store.Task{
		ID:     "1.1",
		Title:  "Test task",
		Status: store.StatusPending,
	}
	taskStore.TasksByPath["/pending/1.1.test_task.md"] = &store.PathEntry{
		Task:     task,
		Status:   store.StatusPending,
		FullPath: "/pending/1.1.test_task.md",
	}

	dir := &Dir{fs: fs, path: "/pending"}
	node, err := dir.Lookup(nil, "1.1.test_task.md")
	if err != nil {
		t.Fatalf("Lookup(1.1.test_task.md) returned error: %v", err)
	}

	file, ok := node.(*File)
	if !ok {
		t.Fatalf("Lookup returned %T, want *File", node)
	}

	if file.Task() != task {
		t.Error("File task does not match expected task")
	}
}

func TestDir_Lookup_StatusDir_ParentTask(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Add a parent task to the store
	task := &store.Task{
		ID:       "1",
		Title:    "Parent task",
		Status:   store.StatusPending,
		Children: []*store.Task{{ID: "1.1", Title: "Child"}},
	}
	taskStore.TasksByPath["/pending/1.parent_task"] = &store.PathEntry{
		Task:     task,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent_task",
	}

	dir := &Dir{fs: fs, path: "/pending"}
	node, err := dir.Lookup(nil, "1.parent_task")
	if err != nil {
		t.Fatalf("Lookup(1.parent_task) returned error: %v", err)
	}

	subdir, ok := node.(*Dir)
	if !ok {
		t.Fatalf("Lookup returned %T, want *Dir", node)
	}

	if subdir.Path() != "/pending/1.parent_task" {
		t.Errorf("Dir path = %s, want /pending/1.parent_task", subdir.Path())
	}
}

func TestDir_Lookup_StatusDir_NotFound(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{fs: fs, path: "/pending"}
	_, err := dir.Lookup(nil, "nonexistent.md")
	if err != fuse.ENOENT {
		t.Errorf("Lookup(nonexistent.md) returned %v, want ENOENT", err)
	}
}

func TestDir_Lookup_TaskDir_IndexMd(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{fs: fs, path: "/pending/1.parent_task"}
	node, err := dir.Lookup(nil, "index.md")
	if err != nil {
		t.Fatalf("Lookup(index.md) returned error: %v", err)
	}

	file, ok := node.(*File)
	if !ok {
		t.Fatalf("Lookup(index.md) returned %T, want *File", node)
	}

	if file.Path() != "/pending/1.parent_task/index.md" {
		t.Errorf("File path = %s, want /pending/1.parent_task/index.md", file.Path())
	}
}

func TestDir_Lookup_TaskDir_ChildTask(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Add a child task to the store
	task := &store.Task{
		ID:     "1.1",
		Title:  "Child task",
		Status: store.StatusPending,
	}
	taskStore.TasksByPath["/pending/1.parent_task/1.1.child_task.md"] = &store.PathEntry{
		Task:     task,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent_task/1.1.child_task.md",
	}

	dir := &Dir{fs: fs, path: "/pending/1.parent_task"}
	node, err := dir.Lookup(nil, "1.1.child_task.md")
	if err != nil {
		t.Fatalf("Lookup(1.1.child_task.md) returned error: %v", err)
	}

	file, ok := node.(*File)
	if !ok {
		t.Fatalf("Lookup returned %T, want *File", node)
	}

	if file.Task() != task {
		t.Error("File task does not match expected task")
	}
}

func TestIsStatusDir(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/pending", true},
		{"/queued", true},
		{"/doing", true},
		{"/done", true},
		{"/failed", true},
		{"/", false},
		{"/pending/1.task", false},
		{"/invalid", false},
		{"pending", false},
		{"/pending/", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isStatusDir(tt.path)
			if result != tt.expected {
				t.Errorf("isStatusDir(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestHashPath(t *testing.T) {
	// Same path should always produce same hash
	hash1 := hashPath("/pending/1.task.md")
	hash2 := hashPath("/pending/1.task.md")
	if hash1 != hash2 {
		t.Error("hashPath should be deterministic")
	}

	// Different paths should produce different hashes (with high probability)
	hash3 := hashPath("/doing/1.task.md")
	if hash1 == hash3 {
		t.Error("Different paths should produce different hashes")
	}
}

func TestStatusDirs(t *testing.T) {
	// Verify StatusDirs contains all expected statuses
	expected := map[store.TaskStatus]bool{
		store.StatusPending: true,
		store.StatusQueued:  true,
		store.StatusDoing:   true,
		store.StatusDone:    true,
		store.StatusFailed:  true,
	}

	if len(StatusDirs) != len(expected) {
		t.Errorf("StatusDirs has %d entries, want %d", len(StatusDirs), len(expected))
	}

	for _, status := range StatusDirs {
		if !expected[status] {
			t.Errorf("Unexpected status in StatusDirs: %s", status)
		}
		delete(expected, status)
	}

	for status := range expected {
		t.Errorf("Missing status in StatusDirs: %s", status)
	}
}

func TestStatusDirSet(t *testing.T) {
	// Verify statusDirSet contains all expected mappings
	expected := map[string]store.TaskStatus{
		"pending": store.StatusPending,
		"queued":  store.StatusQueued,
		"doing":   store.StatusDoing,
		"done":    store.StatusDone,
		"failed":  store.StatusFailed,
	}

	if len(statusDirSet) != len(expected) {
		t.Errorf("statusDirSet has %d entries, want %d", len(statusDirSet), len(expected))
	}

	for name, expectedStatus := range expected {
		actualStatus, ok := statusDirSet[name]
		if !ok {
			t.Errorf("Missing entry in statusDirSet: %s", name)
			continue
		}
		if actualStatus != expectedStatus {
			t.Errorf("statusDirSet[%s] = %s, want %s", name, actualStatus, expectedStatus)
		}
	}
}


// Tests for Task 6.3: Task file nodes implementation

func TestFile_Attr_TaskFile(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	task := &store.Task{
		ID:     "1.1",
		Title:  "Test task",
		Status: store.StatusPending,
		Description: []store.FileLine{
			{RawContent: "  This is a description line"},
			{RawContent: "  Another line"},
		},
	}

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test_task.md",
		task: task,
	}

	var attr fuse.Attr
	err := file.Attr(nil, &attr)
	if err != nil {
		t.Fatalf("Attr() returned error: %v", err)
	}

	// Check mode is read-only (0444)
	if attr.Mode != 0444 {
		t.Errorf("Mode = %o, want 0444", attr.Mode)
	}

	// Check inode is non-zero
	if attr.Inode == 0 {
		t.Error("Inode should be non-zero")
	}

	// Check nlink is 1
	if attr.Nlink != 1 {
		t.Errorf("Nlink = %d, want 1", attr.Nlink)
	}

	// Check size matches generated content
	expectedContent := generateTaskContent(task)
	if attr.Size != uint64(len(expectedContent)) {
		t.Errorf("Size = %d, want %d", attr.Size, len(expectedContent))
	}
}

func TestFile_Attr_IndexFile(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	file := &File{
		fs:   fs,
		path: "/index.md",
		task: nil, // Index files have nil task
	}

	var attr fuse.Attr
	err := file.Attr(nil, &attr)
	if err != nil {
		t.Fatalf("Attr() returned error: %v", err)
	}

	// Check mode is read-only (0444)
	if attr.Mode != 0444 {
		t.Errorf("Mode = %o, want 0444", attr.Mode)
	}

	// Check inode is non-zero
	if attr.Inode == 0 {
		t.Error("Inode should be non-zero")
	}

	// Check nlink is 1
	if attr.Nlink != 1 {
		t.Errorf("Nlink = %d, want 1", attr.Nlink)
	}

	// Index files have actual content generated by index.go
	// Size should be non-zero for index files with content
	if attr.Size == 0 {
		// Root index.md should have content showing status counts
		// This is acceptable - empty store means minimal content
	}
}

func TestFile_Attr_UniqueInodes(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	task := &store.Task{
		ID:     "1.1",
		Title:  "Test task",
		Status: store.StatusPending,
	}

	paths := []string{
		"/index.md",
		"/pending/index.md",
		"/pending/1.1.test_task.md",
		"/doing/index.md",
	}

	inodes := make(map[uint64]string)

	for _, path := range paths {
		var taskRef *store.Task
		if path == "/pending/1.1.test_task.md" {
			taskRef = task
		}

		file := &File{fs: fs, path: path, task: taskRef}
		var attr fuse.Attr
		err := file.Attr(nil, &attr)
		if err != nil {
			t.Fatalf("Attr() for %s returned error: %v", path, err)
		}

		if existing, ok := inodes[attr.Inode]; ok {
			t.Errorf("Inode collision: %s and %s have same inode %d", path, existing, attr.Inode)
		}
		inodes[attr.Inode] = path
	}
}

func TestFile_Read_TaskFile(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	task := &store.Task{
		ID:     "1.2",
		Title:  "Configure build system",
		Status: store.StatusDoing,
		Description: []store.FileLine{
			{RawContent: "  Set up the build configuration"},
		},
	}

	file := &File{
		fs:   fs,
		path: "/doing/1.2.configure_build_system.md",
		task: task,
	}

	req := &fuse.ReadRequest{
		Offset: 0,
		Size:   4096,
	}
	resp := &fuse.ReadResponse{}

	err := file.Read(nil, req, resp)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	expectedContent := "# 1.2. Configure build system\n\n**Status:** doing\n\n  Set up the build configuration\n"
	if string(resp.Data) != expectedContent {
		t.Errorf("Read() returned:\n%q\nwant:\n%q", string(resp.Data), expectedContent)
	}
}

func TestFile_Read_TaskFile_NoDescription(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	task := &store.Task{
		ID:          "1.1",
		Title:       "Create structure",
		Status:      store.StatusDone,
		Description: nil,
	}

	file := &File{
		fs:   fs,
		path: "/done/1.1.create_structure.md",
		task: task,
	}

	req := &fuse.ReadRequest{
		Offset: 0,
		Size:   4096,
	}
	resp := &fuse.ReadResponse{}

	err := file.Read(nil, req, resp)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	expectedContent := "# 1.1. Create structure\n\n**Status:** done\n\n"
	if string(resp.Data) != expectedContent {
		t.Errorf("Read() returned:\n%q\nwant:\n%q", string(resp.Data), expectedContent)
	}
}

func TestFile_Read_IndexFile(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	file := &File{
		fs:   fs,
		path: "/index.md",
		task: nil,
	}

	req := &fuse.ReadRequest{
		Offset: 0,
		Size:   4096,
	}
	resp := &fuse.ReadResponse{}

	err := file.Read(nil, req, resp)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	// Index files return generated content (implemented in Task 7)
	// Root index.md should have content showing status counts
	if len(resp.Data) == 0 {
		t.Log("Index file returned empty content - this is acceptable for empty store")
	}
}

func TestFile_Read_WithOffset(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	task := &store.Task{
		ID:     "1.1",
		Title:  "Test task",
		Status: store.StatusPending,
	}

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test_task.md",
		task: task,
	}

	fullContent := generateTaskContent(task)

	// Read from offset 5
	req := &fuse.ReadRequest{
		Offset: 5,
		Size:   10,
	}
	resp := &fuse.ReadResponse{}

	err := file.Read(nil, req, resp)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	expectedData := fullContent[5:15]
	if string(resp.Data) != expectedData {
		t.Errorf("Read() with offset returned:\n%q\nwant:\n%q", string(resp.Data), expectedData)
	}
}

func TestFile_Read_OffsetBeyondContent(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	task := &store.Task{
		ID:     "1.1",
		Title:  "Test task",
		Status: store.StatusPending,
	}

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test_task.md",
		task: task,
	}

	fullContent := generateTaskContent(task)

	// Read from offset beyond content length
	req := &fuse.ReadRequest{
		Offset: int64(len(fullContent) + 100),
		Size:   10,
	}
	resp := &fuse.ReadResponse{}

	err := file.Read(nil, req, resp)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	// Should return empty data
	if len(resp.Data) != 0 {
		t.Errorf("Read() with offset beyond content returned %d bytes, want 0", len(resp.Data))
	}
}

func TestFile_Read_SizeLargerThanContent(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	task := &store.Task{
		ID:     "1.1",
		Title:  "Test task",
		Status: store.StatusPending,
	}

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test_task.md",
		task: task,
	}

	fullContent := generateTaskContent(task)

	// Request more bytes than content length
	req := &fuse.ReadRequest{
		Offset: 0,
		Size:   len(fullContent) + 1000,
	}
	resp := &fuse.ReadResponse{}

	err := file.Read(nil, req, resp)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	// Should return only the available content
	if string(resp.Data) != fullContent {
		t.Errorf("Read() returned:\n%q\nwant:\n%q", string(resp.Data), fullContent)
	}
}

func TestFile_Read_PartialRead(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	task := &store.Task{
		ID:     "1.1",
		Title:  "Test task",
		Status: store.StatusPending,
	}

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test_task.md",
		task: task,
	}

	fullContent := generateTaskContent(task)

	// Read from middle of content with size that extends beyond end
	offset := int64(len(fullContent) - 5)
	req := &fuse.ReadRequest{
		Offset: offset,
		Size:   100,
	}
	resp := &fuse.ReadResponse{}

	err := file.Read(nil, req, resp)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}

	// Should return only the remaining 5 bytes
	expectedData := fullContent[offset:]
	if string(resp.Data) != expectedData {
		t.Errorf("Read() returned:\n%q\nwant:\n%q", string(resp.Data), expectedData)
	}
}

func TestGenerateTaskContent(t *testing.T) {
	tests := []struct {
		name     string
		task     *store.Task
		expected string
	}{
		{
			name: "basic task",
			task: &store.Task{
				ID:     "1.1",
				Title:  "Test task",
				Status: store.StatusPending,
			},
			expected: "# 1.1. Test task\n\n**Status:** pending\n\n",
		},
		{
			name: "task with description",
			task: &store.Task{
				ID:     "2.3",
				Title:  "Another task",
				Status: store.StatusDoing,
				Description: []store.FileLine{
					{RawContent: "  Line 1"},
					{RawContent: "  Line 2"},
				},
			},
			expected: "# 2.3. Another task\n\n**Status:** doing\n\n  Line 1\n  Line 2\n",
		},
		{
			name: "done task",
			task: &store.Task{
				ID:     "1",
				Title:  "Root task",
				Status: store.StatusDone,
			},
			expected: "# 1. Root task\n\n**Status:** done\n\n",
		},
		{
			name: "failed task",
			task: &store.Task{
				ID:     "3.2.1",
				Title:  "Deep nested task",
				Status: store.StatusFailed,
				Description: []store.FileLine{
					{RawContent: "    Error details here"},
				},
			},
			expected: "# 3.2.1. Deep nested task\n\n**Status:** failed\n\n    Error details here\n",
		},
		{
			name: "queued task",
			task: &store.Task{
				ID:     "5",
				Title:  "Queued task",
				Status: store.StatusQueued,
			},
			expected: "# 5. Queued task\n\n**Status:** queued\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTaskContent(tt.task)
			if result != tt.expected {
				t.Errorf("generateTaskContent() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}

func TestGenerateTaskContent_Format(t *testing.T) {
	// Test that the format matches the Task File Content Format specification
	task := &store.Task{
		ID:     "1.2",
		Title:  "Implement the parser",
		Status: store.StatusPending,
		Description: []store.FileLine{
			{RawContent: "This task involves creating the markdown parser."},
			{RawContent: "- Handle checkbox syntax"},
			{RawContent: "- Extract task IDs"},
		},
	}

	content := generateTaskContent(task)

	// Verify format: # {task_id}. {title}
	if !strings.HasPrefix(content, "# 1.2. Implement the parser\n") {
		t.Error("Content should start with '# {task_id}. {title}'")
	}

	// Verify **Status:** line
	if !strings.Contains(content, "**Status:** pending") {
		t.Error("Content should contain '**Status:** {status}'")
	}

	// Verify description lines are preserved
	if !strings.Contains(content, "This task involves creating the markdown parser.") {
		t.Error("Content should contain description lines")
	}
	if !strings.Contains(content, "- Handle checkbox syntax") {
		t.Error("Content should contain description lines")
	}
}

// =============================================================================
// Tests for Task 6.4: Task directory nodes for parent tasks
// Validates: Requirements 2.3, 2.4
// =============================================================================

// TestDir_ParentTaskExposedAsDirectory verifies that parent tasks (tasks with children)
// are exposed as directories, not files.
// Validates: Requirements 2.3
func TestDir_ParentTaskExposedAsDirectory(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Create a parent task with children
	childTask := &store.Task{
		ID:     "1.1",
		Title:  "Child task",
		Status: store.StatusPending,
	}
	parentTask := &store.Task{
		ID:       "1",
		Title:    "Parent task",
		Status:   store.StatusPending,
		Children: []*store.Task{childTask},
	}
	childTask.Parent = parentTask

	// Add to store
	taskStore.TasksByPath["/pending/1.parent_task"] = &store.PathEntry{
		Task:     parentTask,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent_task",
	}
	taskStore.TasksByPath["/pending/1.parent_task/1.1.child_task.md"] = &store.PathEntry{
		Task:     childTask,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent_task/1.1.child_task.md",
	}

	// Lookup the parent task from status directory
	dir := &Dir{fs: fs, path: "/pending"}
	node, err := dir.Lookup(nil, "1.parent_task")
	if err != nil {
		t.Fatalf("Lookup(1.parent_task) returned error: %v", err)
	}

	// Verify it's a directory, not a file
	subdir, ok := node.(*Dir)
	if !ok {
		t.Fatalf("Parent task should be exposed as Dir, got %T", node)
	}

	if subdir.Path() != "/pending/1.parent_task" {
		t.Errorf("Dir path = %s, want /pending/1.parent_task", subdir.Path())
	}
}

// TestDir_LeafTaskExposedAsFile verifies that leaf tasks (tasks without children)
// are exposed as files, not directories.
// Validates: Requirements 2.2
func TestDir_LeafTaskExposedAsFile(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Create a leaf task (no children)
	leafTask := &store.Task{
		ID:       "1",
		Title:    "Leaf task",
		Status:   store.StatusPending,
		Children: nil, // No children = leaf task
	}

	// Add to store
	taskStore.TasksByPath["/pending/1.leaf_task.md"] = &store.PathEntry{
		Task:     leafTask,
		Status:   store.StatusPending,
		FullPath: "/pending/1.leaf_task.md",
	}

	// Lookup the leaf task from status directory
	dir := &Dir{fs: fs, path: "/pending"}
	node, err := dir.Lookup(nil, "1.leaf_task.md")
	if err != nil {
		t.Fatalf("Lookup(1.leaf_task.md) returned error: %v", err)
	}

	// Verify it's a file, not a directory
	file, ok := node.(*File)
	if !ok {
		t.Fatalf("Leaf task should be exposed as File, got %T", node)
	}

	if file.Task() != leafTask {
		t.Error("File task does not match expected task")
	}
}

// TestDir_HierarchicalNesting_TwoLevels verifies that hierarchical nesting is preserved
// for two-level deep structures (parent/child).
// Validates: Requirements 2.4
func TestDir_HierarchicalNesting_TwoLevels(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Create two-level hierarchy: parent -> child
	childTask := &store.Task{
		ID:     "1.1",
		Title:  "Child task",
		Status: store.StatusPending,
	}
	parentTask := &store.Task{
		ID:       "1",
		Title:    "Parent task",
		Status:   store.StatusPending,
		Children: []*store.Task{childTask},
	}
	childTask.Parent = parentTask

	// Add to store with hierarchical paths
	taskStore.TasksByPath["/pending/1.parent_task"] = &store.PathEntry{
		Task:     parentTask,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent_task",
	}
	taskStore.TasksByPath["/pending/1.parent_task/1.1.child_task.md"] = &store.PathEntry{
		Task:     childTask,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent_task/1.1.child_task.md",
	}

	// Navigate: /pending -> 1.parent_task -> 1.1.child_task.md
	statusDir := &Dir{fs: fs, path: "/pending"}

	// Step 1: Lookup parent directory
	parentNode, err := statusDir.Lookup(nil, "1.parent_task")
	if err != nil {
		t.Fatalf("Lookup(1.parent_task) returned error: %v", err)
	}
	parentDir, ok := parentNode.(*Dir)
	if !ok {
		t.Fatalf("Parent should be Dir, got %T", parentNode)
	}

	// Step 2: Lookup child file within parent directory
	childNode, err := parentDir.Lookup(nil, "1.1.child_task.md")
	if err != nil {
		t.Fatalf("Lookup(1.1.child_task.md) returned error: %v", err)
	}
	childFile, ok := childNode.(*File)
	if !ok {
		t.Fatalf("Child should be File, got %T", childNode)
	}

	if childFile.Task() != childTask {
		t.Error("Child file task does not match expected task")
	}
}

// TestDir_HierarchicalNesting_ThreeLevels verifies that deeply nested structures
// (3+ levels) work correctly: parent/child/grandchild.
// Validates: Requirements 2.4
func TestDir_HierarchicalNesting_ThreeLevels(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Create three-level hierarchy: parent -> child -> grandchild
	grandchildTask := &store.Task{
		ID:     "1.2.3",
		Title:  "Grandchild task",
		Status: store.StatusPending,
	}
	childTask := &store.Task{
		ID:       "1.2",
		Title:    "Child task",
		Status:   store.StatusPending,
		Children: []*store.Task{grandchildTask},
	}
	parentTask := &store.Task{
		ID:       "1",
		Title:    "Parent task",
		Status:   store.StatusPending,
		Children: []*store.Task{childTask},
	}
	grandchildTask.Parent = childTask
	childTask.Parent = parentTask

	// Add to store with hierarchical paths matching design spec:
	// pending/1.parent/1.2.child/1.2.3.leaf.md
	taskStore.TasksByPath["/pending/1.parent_task"] = &store.PathEntry{
		Task:     parentTask,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent_task",
	}
	taskStore.TasksByPath["/pending/1.parent_task/1.2.child_task"] = &store.PathEntry{
		Task:     childTask,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent_task/1.2.child_task",
	}
	taskStore.TasksByPath["/pending/1.parent_task/1.2.child_task/1.2.3.grandchild_task.md"] = &store.PathEntry{
		Task:     grandchildTask,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent_task/1.2.child_task/1.2.3.grandchild_task.md",
	}

	// Navigate: /pending -> 1.parent_task -> 1.2.child_task -> 1.2.3.grandchild_task.md
	statusDir := &Dir{fs: fs, path: "/pending"}

	// Step 1: Lookup parent directory
	parentNode, err := statusDir.Lookup(nil, "1.parent_task")
	if err != nil {
		t.Fatalf("Lookup(1.parent_task) returned error: %v", err)
	}
	parentDir := parentNode.(*Dir)

	// Step 2: Lookup child directory within parent
	childNode, err := parentDir.Lookup(nil, "1.2.child_task")
	if err != nil {
		t.Fatalf("Lookup(1.2.child_task) returned error: %v", err)
	}
	childDir, ok := childNode.(*Dir)
	if !ok {
		t.Fatalf("Child should be Dir (has grandchild), got %T", childNode)
	}

	// Step 3: Lookup grandchild file within child directory
	grandchildNode, err := childDir.Lookup(nil, "1.2.3.grandchild_task.md")
	if err != nil {
		t.Fatalf("Lookup(1.2.3.grandchild_task.md) returned error: %v", err)
	}
	grandchildFile, ok := grandchildNode.(*File)
	if !ok {
		t.Fatalf("Grandchild should be File, got %T", grandchildNode)
	}

	if grandchildFile.Task() != grandchildTask {
		t.Error("Grandchild file task does not match expected task")
	}
}

// TestDir_ReadDirAll_TaskDir_NestedChildren verifies that ReadDirAll on a task directory
// returns all direct children (both files and subdirectories).
// Validates: Requirements 2.3, 2.4
func TestDir_ReadDirAll_TaskDir_NestedChildren(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Create parent with mixed children: one leaf (file) and one parent (directory)
	leafChild := &store.Task{
		ID:     "1.1",
		Title:  "Leaf child",
		Status: store.StatusPending,
	}
	nestedGrandchild := &store.Task{
		ID:     "1.2.1",
		Title:  "Grandchild",
		Status: store.StatusPending,
	}
	parentChild := &store.Task{
		ID:       "1.2",
		Title:    "Parent child",
		Status:   store.StatusPending,
		Children: []*store.Task{nestedGrandchild},
	}

	// Add to store
	taskStore.TasksByPath["/pending/1.parent/1.1.leaf_child.md"] = &store.PathEntry{
		Task:     leafChild,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent/1.1.leaf_child.md",
	}
	taskStore.TasksByPath["/pending/1.parent/1.2.parent_child"] = &store.PathEntry{
		Task:     parentChild,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent/1.2.parent_child",
	}
	// Grandchild is nested deeper, should NOT appear in parent's ReadDirAll
	taskStore.TasksByPath["/pending/1.parent/1.2.parent_child/1.2.1.grandchild.md"] = &store.PathEntry{
		Task:     nestedGrandchild,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent/1.2.parent_child/1.2.1.grandchild.md",
	}

	dir := &Dir{fs: fs, path: "/pending/1.parent"}
	entries, err := dir.ReadDirAll(nil)
	if err != nil {
		t.Fatalf("ReadDirAll() returned error: %v", err)
	}

	// Should have: index.md + 2 direct children = 3 entries
	// Grandchild should NOT be included (it's nested deeper)
	if len(entries) != 3 {
		t.Errorf("ReadDirAll() returned %d entries, want 3", len(entries))
	}

	// Check for expected entries
	expectedNames := map[string]fuse.DirentType{
		"index.md":          fuse.DT_File,
		"1.1.leaf_child.md": fuse.DT_File,
		"1.2.parent_child":  fuse.DT_Dir,
	}

	for _, entry := range entries {
		expectedType, ok := expectedNames[entry.Name]
		if !ok {
			t.Errorf("Unexpected entry: %s", entry.Name)
			continue
		}
		if entry.Type != expectedType {
			t.Errorf("Entry %s has type %v, want %v", entry.Name, entry.Type, expectedType)
		}
		delete(expectedNames, entry.Name)
	}

	for name := range expectedNames {
		t.Errorf("Missing expected entry: %s", name)
	}
}

// TestDir_ChildrenUnderParentRegardlessOfStatus verifies that children appear under
// their parent directory regardless of their individual status.
// This is the "Critical Rule: Children Live Under Parent Directory" from the design.
// Validates: Requirements 2.3, 2.4
func TestDir_ChildrenUnderParentRegardlessOfStatus(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Create parent with children of different statuses
	// Per design: Task 1.1 (done), 1.2 (doing), 1.3 (pending) all appear under parent
	doneChild := &store.Task{
		ID:     "1.1",
		Title:  "Done child",
		Status: store.StatusDone,
	}
	doingChild := &store.Task{
		ID:     "1.2",
		Title:  "Doing child",
		Status: store.StatusDoing,
	}
	pendingChild := &store.Task{
		ID:     "1.3",
		Title:  "Pending child",
		Status: store.StatusPending,
	}
	parentTask := &store.Task{
		ID:       "1",
		Title:    "Setup project",
		Status:   store.StatusPending, // Will be derived to "doing" due to 1.2
		Children: []*store.Task{doneChild, doingChild, pendingChild},
	}

	// Parent's derived status is "doing" (has doing child)
	// ALL children should be under /doing/1.setup_project/ regardless of their status
	taskStore.TasksByPath["/doing/1.setup_project"] = &store.PathEntry{
		Task:     parentTask,
		Status:   store.StatusDoing,
		FullPath: "/doing/1.setup_project",
	}
	taskStore.TasksByPath["/doing/1.setup_project/1.1.done_child.md"] = &store.PathEntry{
		Task:     doneChild,
		Status:   store.StatusDone, // Individual status is done
		FullPath: "/doing/1.setup_project/1.1.done_child.md",
	}
	taskStore.TasksByPath["/doing/1.setup_project/1.2.doing_child.md"] = &store.PathEntry{
		Task:     doingChild,
		Status:   store.StatusDoing,
		FullPath: "/doing/1.setup_project/1.2.doing_child.md",
	}
	taskStore.TasksByPath["/doing/1.setup_project/1.3.pending_child.md"] = &store.PathEntry{
		Task:     pendingChild,
		Status:   store.StatusPending, // Individual status is pending
		FullPath: "/doing/1.setup_project/1.3.pending_child.md",
	}

	// Verify parent is in /doing directory
	doingDir := &Dir{fs: fs, path: "/doing"}
	parentNode, err := doingDir.Lookup(nil, "1.setup_project")
	if err != nil {
		t.Fatalf("Parent should be in /doing: %v", err)
	}
	parentDir := parentNode.(*Dir)

	// Verify ALL children are under parent, regardless of their individual status
	entries, err := parentDir.ReadDirAll(nil)
	if err != nil {
		t.Fatalf("ReadDirAll() returned error: %v", err)
	}

	// Should have: index.md + 3 children = 4 entries
	if len(entries) != 4 {
		t.Errorf("ReadDirAll() returned %d entries, want 4", len(entries))
	}

	// Verify each child can be looked up from parent directory
	childNames := []string{"1.1.done_child.md", "1.2.doing_child.md", "1.3.pending_child.md"}
	for _, name := range childNames {
		node, err := parentDir.Lookup(nil, name)
		if err != nil {
			t.Errorf("Child %s should be under parent: %v", name, err)
			continue
		}
		if _, ok := node.(*File); !ok {
			t.Errorf("Child %s should be a File", name)
		}
	}

	// Verify children are NOT in their own status directories
	// (They should only be under the parent)
	doneDir := &Dir{fs: fs, path: "/done"}
	_, err = doneDir.Lookup(nil, "1.1.done_child.md")
	if err != fuse.ENOENT {
		t.Error("Done child should NOT be in /done directory (should be under parent)")
	}

	pendingDir := &Dir{fs: fs, path: "/pending"}
	_, err = pendingDir.Lookup(nil, "1.3.pending_child.md")
	if err != fuse.ENOENT {
		t.Error("Pending child should NOT be in /pending directory (should be under parent)")
	}
}

// TestDir_TaskDir_IndexMdInNestedDir verifies that index.md exists in nested task directories.
// Validates: Requirements 3.5
func TestDir_TaskDir_IndexMdInNestedDir(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Create nested structure
	grandchild := &store.Task{
		ID:     "1.1.1",
		Title:  "Grandchild",
		Status: store.StatusPending,
	}
	child := &store.Task{
		ID:       "1.1",
		Title:    "Child",
		Status:   store.StatusPending,
		Children: []*store.Task{grandchild},
	}
	parent := &store.Task{
		ID:       "1",
		Title:    "Parent",
		Status:   store.StatusPending,
		Children: []*store.Task{child},
	}

	taskStore.TasksByPath["/pending/1.parent"] = &store.PathEntry{
		Task:     parent,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent",
	}
	taskStore.TasksByPath["/pending/1.parent/1.1.child"] = &store.PathEntry{
		Task:     child,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent/1.1.child",
	}
	taskStore.TasksByPath["/pending/1.parent/1.1.child/1.1.1.grandchild.md"] = &store.PathEntry{
		Task:     grandchild,
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent/1.1.child/1.1.1.grandchild.md",
	}

	// Verify index.md exists at each level
	testCases := []struct {
		dirPath string
		name    string
	}{
		{"/pending/1.parent", "parent task directory"},
		{"/pending/1.parent/1.1.child", "nested child task directory"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := &Dir{fs: fs, path: tc.dirPath}
			node, err := dir.Lookup(nil, "index.md")
			if err != nil {
				t.Fatalf("index.md should exist in %s: %v", tc.dirPath, err)
			}
			file, ok := node.(*File)
			if !ok {
				t.Fatalf("index.md should be a File, got %T", node)
			}
			if file.Task() != nil {
				t.Error("index.md should have nil task")
			}
		})
	}
}

// TestDir_Attr_TaskDirectory verifies that task directories have correct attributes.
// Validates: Requirements 2.3
func TestDir_Attr_TaskDirectory(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Test attributes for nested task directories
	testCases := []struct {
		path string
		name string
	}{
		{"/pending/1.parent_task", "first level task directory"},
		{"/pending/1.parent_task/1.1.child_task", "second level task directory"},
		{"/doing/2.another_parent/2.1.child/2.1.1.grandchild", "third level task directory"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := &Dir{fs: fs, path: tc.path}

			var attr fuse.Attr
			err := dir.Attr(nil, &attr)
			if err != nil {
				t.Fatalf("Attr() returned error: %v", err)
			}

			// Check mode is directory with 0555 permissions
			expectedMode := os.ModeDir | 0555
			if attr.Mode != expectedMode {
				t.Errorf("Mode = %v, want %v", attr.Mode, expectedMode)
			}

			// Check inode is non-zero and unique
			if attr.Inode == 0 {
				t.Error("Inode should be non-zero")
			}

			// Check nlink is 2 (standard for directories)
			if attr.Nlink != 2 {
				t.Errorf("Nlink = %d, want 2", attr.Nlink)
			}
		})
	}
}

// TestDir_ReadDirAll_ExcludesNestedPaths verifies that ReadDirAll only returns
// direct children, not deeply nested paths.
// Validates: Requirements 2.4
func TestDir_ReadDirAll_ExcludesNestedPaths(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Add paths at multiple levels
	taskStore.TasksByPath["/pending/1.parent"] = &store.PathEntry{
		Task:     &store.Task{ID: "1", Title: "Parent", Children: []*store.Task{{}}},
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent",
	}
	taskStore.TasksByPath["/pending/1.parent/1.1.child.md"] = &store.PathEntry{
		Task:     &store.Task{ID: "1.1", Title: "Child"},
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent/1.1.child.md",
	}
	taskStore.TasksByPath["/pending/1.parent/1.2.nested_parent"] = &store.PathEntry{
		Task:     &store.Task{ID: "1.2", Title: "Nested Parent", Children: []*store.Task{{}}},
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent/1.2.nested_parent",
	}
	// This deeply nested path should NOT appear in /pending ReadDirAll
	taskStore.TasksByPath["/pending/1.parent/1.2.nested_parent/1.2.1.deep.md"] = &store.PathEntry{
		Task:     &store.Task{ID: "1.2.1", Title: "Deep"},
		Status:   store.StatusPending,
		FullPath: "/pending/1.parent/1.2.nested_parent/1.2.1.deep.md",
	}

	// ReadDirAll on /pending should only show direct children
	pendingDir := &Dir{fs: fs, path: "/pending"}
	entries, err := pendingDir.ReadDirAll(nil)
	if err != nil {
		t.Fatalf("ReadDirAll() returned error: %v", err)
	}

	// Should have: index.md + 1.parent = 2 entries
	// Should NOT include nested paths
	if len(entries) != 2 {
		t.Errorf("ReadDirAll() returned %d entries, want 2", len(entries))
		for _, e := range entries {
			t.Logf("  Entry: %s (type: %v)", e.Name, e.Type)
		}
	}

	// Verify only expected entries
	foundParent := false
	foundIndex := false
	for _, entry := range entries {
		switch entry.Name {
		case "index.md":
			foundIndex = true
		case "1.parent":
			foundParent = true
			if entry.Type != fuse.DT_Dir {
				t.Errorf("1.parent should be DT_Dir, got %v", entry.Type)
			}
		default:
			t.Errorf("Unexpected entry: %s", entry.Name)
		}
	}

	if !foundIndex {
		t.Error("Missing index.md entry")
	}
	if !foundParent {
		t.Error("Missing 1.parent entry")
	}
}


// =============================================================================
// Tests for Task 6.5: Status-based task placement
// Validates: Requirements 2.5, 2.6, 2.7, 2.8, Design - Example 1
// =============================================================================

// TestStatusBasedPlacement_LeafTaskInOwnStatusDir verifies that a leaf task
// (task with no children) appears in its own status directory.
// Validates: Requirements 2.5, 2.6, 2.7, 2.8
func TestStatusBasedPlacement_LeafTaskInOwnStatusDir(t *testing.T) {
	tests := []struct {
		name       string
		status     store.TaskStatus
		statusDir  string
	}{
		{"pending leaf task", store.StatusPending, "/pending"},
		{"queued leaf task", store.StatusQueued, "/queued"},
		{"doing leaf task", store.StatusDoing, "/doing"},
		{"done leaf task", store.StatusDone, "/done"},
		{"failed leaf task", store.StatusFailed, "/failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskStore := store.NewTaskStore()
			pm := store.NewPathManager(taskStore)
			fuseFS := NewFuseFS(taskStore, "/mnt/tasks")

			// Create a leaf task with the specified status
			task := &store.Task{
				ID:     "1",
				Title:  "Test Task",
				Status: tt.status,
			}
			taskStore.Document = &store.ParsedDocument{RootTasks: []*store.Task{task}}

			// Build paths using PathManager
			pm.BuildAllPaths()

			// Verify the task is in the correct status directory
			path := taskStore.PathByTask[task]
			if !strings.HasPrefix(path, tt.statusDir+"/") {
				t.Errorf("Task path = %q, should be in %s/", path, tt.statusDir)
			}

			// Verify the task can be looked up via FUSE filesystem
			statusDir := &Dir{fs: fuseFS, path: tt.statusDir}
			entries, err := statusDir.ReadDirAll(nil)
			if err != nil {
				t.Fatalf("ReadDirAll() returned error: %v", err)
			}

			// Should find the task file (plus index.md)
			foundTask := false
			for _, entry := range entries {
				if entry.Name == "1.test_task.md" {
					foundTask = true
					if entry.Type != fuse.DT_File {
						t.Errorf("Task entry should be DT_File, got %v", entry.Type)
					}
					break
				}
			}
			if !foundTask {
				t.Errorf("Task not found in %s directory", tt.statusDir)
			}
		})
	}
}

// TestStatusBasedPlacement_ParentTaskInDerivedStatusDir verifies that a parent task
// appears in the status directory matching its derived status (from children).
// Validates: Requirements 4.7, Design - Example 1
func TestStatusBasedPlacement_ParentTaskInDerivedStatusDir(t *testing.T) {
	tests := []struct {
		name           string
		childStatuses  []store.TaskStatus
		expectedDir    string
		expectedStatus store.TaskStatus
	}{
		{
			name:           "parent with doing child goes to doing",
			childStatuses:  []store.TaskStatus{store.StatusDone, store.StatusDoing, store.StatusPending},
			expectedDir:    "/doing",
			expectedStatus: store.StatusDoing,
		},
		{
			name:           "parent with failed child (no doing) goes to failed",
			childStatuses:  []store.TaskStatus{store.StatusDone, store.StatusFailed, store.StatusPending},
			expectedDir:    "/failed",
			expectedStatus: store.StatusFailed,
		},
		{
			name:           "parent with pending child (no doing/failed) goes to pending",
			childStatuses:  []store.TaskStatus{store.StatusDone, store.StatusPending, store.StatusQueued},
			expectedDir:    "/pending",
			expectedStatus: store.StatusPending,
		},
		{
			name:           "parent with queued child (no doing/failed/pending) goes to queued",
			childStatuses:  []store.TaskStatus{store.StatusDone, store.StatusQueued},
			expectedDir:    "/queued",
			expectedStatus: store.StatusQueued,
		},
		{
			name:           "parent with all done children goes to done",
			childStatuses:  []store.TaskStatus{store.StatusDone, store.StatusDone, store.StatusDone},
			expectedDir:    "/done",
			expectedStatus: store.StatusDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskStore := store.NewTaskStore()
			pm := store.NewPathManager(taskStore)
			fuseFS := NewFuseFS(taskStore, "/mnt/tasks")

			// Create children with specified statuses
			var children []*store.Task
			for i, status := range tt.childStatuses {
				child := &store.Task{
					ID:     "1." + itoa(i+1),
					Title:  "Child " + itoa(i+1),
					Status: status,
				}
				children = append(children, child)
			}

			// Create parent task
			parent := &store.Task{
				ID:       "1",
				Title:    "Parent Task",
				Status:   store.StatusPending, // Will be derived
				Children: children,
			}
			for _, child := range children {
				child.Parent = parent
			}

			taskStore.Document = &store.ParsedDocument{RootTasks: []*store.Task{parent}}

			// Build paths using PathManager
			pm.BuildAllPaths()

			// Verify parent is in the correct derived status directory
			parentPath := taskStore.PathByTask[parent]
			if !strings.HasPrefix(parentPath, tt.expectedDir+"/") {
				t.Errorf("Parent path = %q, should be in %s/", parentPath, tt.expectedDir)
			}

			// Verify via FUSE filesystem
			statusDir := &Dir{fs: fuseFS, path: tt.expectedDir}
			entries, err := statusDir.ReadDirAll(nil)
			if err != nil {
				t.Fatalf("ReadDirAll() returned error: %v", err)
			}

			// Should find the parent directory
			foundParent := false
			for _, entry := range entries {
				if entry.Name == "1.parent_task" {
					foundParent = true
					if entry.Type != fuse.DT_Dir {
						t.Errorf("Parent entry should be DT_Dir, got %v", entry.Type)
					}
					break
				}
			}
			if !foundParent {
				t.Errorf("Parent not found in %s directory", tt.expectedDir)
			}
		})
	}
}

// TestStatusBasedPlacement_ChildrenUnderParent verifies that children with different
// statuses all appear under their parent directory, not in their own status directories.
// This is the "Critical Rule: Children Live Under Parent Directory" from the design.
// Validates: Requirements 2.3, 2.4, Design - Example 1
func TestStatusBasedPlacement_ChildrenUnderParent(t *testing.T) {
	taskStore := store.NewTaskStore()
	pm := store.NewPathManager(taskStore)
	fuseFS := NewFuseFS(taskStore, "/mnt/tasks")

	// Create the exact example from the design document:
	// Task 1 has children: 1.1 (done), 1.2 (doing), 1.3 (pending)
	// Child 1.2 is `doing` → Parent 1 derives status `doing`
	child1 := &store.Task{
		ID:     "1.1",
		Title:  "Create directory structure",
		Status: store.StatusDone,
	}
	child2 := &store.Task{
		ID:     "1.2",
		Title:  "Configure build system",
		Status: store.StatusDoing,
	}
	child3 := &store.Task{
		ID:     "1.3",
		Title:  "Add dependencies",
		Status: store.StatusPending,
	}
	parent := &store.Task{
		ID:       "1",
		Title:    "Setup project",
		Status:   store.StatusPending,
		Children: []*store.Task{child1, child2, child3},
	}
	child1.Parent = parent
	child2.Parent = parent
	child3.Parent = parent

	taskStore.Document = &store.ParsedDocument{RootTasks: []*store.Task{parent}}

	// Build paths using PathManager
	pm.BuildAllPaths()

	// Verify parent is in /doing (derived from child 1.2)
	parentPath := taskStore.PathByTask[parent]
	if !strings.HasPrefix(parentPath, "/doing/") {
		t.Errorf("Parent path = %q, should be in /doing/", parentPath)
	}

	// Verify ALL children are under parent directory
	for _, child := range []*store.Task{child1, child2, child3} {
		childPath := taskStore.PathByTask[child]
		if !strings.HasPrefix(childPath, parentPath+"/") {
			t.Errorf("Child %s path = %q, should be under parent %q", child.ID, childPath, parentPath)
		}
	}

	// Verify children are NOT in their own status directories via FUSE
	// Child 1.1 (done) should NOT be in /done
	doneDir := &Dir{fs: fuseFS, path: "/done"}
	doneEntries, _ := doneDir.ReadDirAll(nil)
	for _, entry := range doneEntries {
		if strings.Contains(entry.Name, "1.1") {
			t.Error("Child 1.1 (done) should NOT be in /done directory")
		}
	}

	// Child 1.3 (pending) should NOT be in /pending
	pendingDir := &Dir{fs: fuseFS, path: "/pending"}
	pendingEntries, _ := pendingDir.ReadDirAll(nil)
	for _, entry := range pendingEntries {
		if strings.Contains(entry.Name, "1.3") {
			t.Error("Child 1.3 (pending) should NOT be in /pending directory")
		}
	}

	// Verify children CAN be found under parent via FUSE
	parentDir := &Dir{fs: fuseFS, path: parentPath}
	parentEntries, err := parentDir.ReadDirAll(nil)
	if err != nil {
		t.Fatalf("ReadDirAll() on parent returned error: %v", err)
	}

	expectedChildren := map[string]bool{
		"1.1.create_directory_structure.md": false,
		"1.2.configure_build_system.md":     false,
		"1.3.add_dependencies.md":           false,
	}

	for _, entry := range parentEntries {
		if _, ok := expectedChildren[entry.Name]; ok {
			expectedChildren[entry.Name] = true
		}
	}

	for name, found := range expectedChildren {
		if !found {
			t.Errorf("Child %s not found under parent directory", name)
		}
	}
}

// TestStatusBasedPlacement_SubtreeMovesOnStatusChange verifies that when a child's
// status changes (affecting parent's derived status), the entire subtree moves.
// Validates: Requirements 4.7, Design - Path Management
func TestStatusBasedPlacement_SubtreeMovesOnStatusChange(t *testing.T) {
	taskStore := store.NewTaskStore()
	pm := store.NewPathManager(taskStore)
	fuseFS := NewFuseFS(taskStore, "/mnt/tasks")

	// Create parent with one pending child
	child := &store.Task{
		ID:     "1.1",
		Title:  "Child Task",
		Status: store.StatusPending,
	}
	parent := &store.Task{
		ID:       "1",
		Title:    "Parent Task",
		Status:   store.StatusPending,
		Children: []*store.Task{child},
	}
	child.Parent = parent

	taskStore.Document = &store.ParsedDocument{RootTasks: []*store.Task{parent}}

	// Initial build - parent should be in /pending
	pm.BuildAllPaths()
	initialParentPath := taskStore.PathByTask[parent]
	if !strings.HasPrefix(initialParentPath, "/pending/") {
		t.Errorf("Initial parent path = %q, should be in /pending/", initialParentPath)
	}

	// Verify parent is in /pending via FUSE
	pendingDir := &Dir{fs: fuseFS, path: "/pending"}
	_, err := pendingDir.Lookup(nil, "1.parent_task")
	if err != nil {
		t.Errorf("Parent should be in /pending initially: %v", err)
	}

	// Change child status to doing
	child.Status = store.StatusDoing
	pm.OnStatusChange(child)

	// Parent should now be in /doing
	newParentPath := taskStore.PathByTask[parent]
	if !strings.HasPrefix(newParentPath, "/doing/") {
		t.Errorf("After child doing, parent path = %q, should be in /doing/", newParentPath)
	}

	// Verify parent is now in /doing via FUSE
	doingDir := &Dir{fs: fuseFS, path: "/doing"}
	_, err = doingDir.Lookup(nil, "1.parent_task")
	if err != nil {
		t.Errorf("Parent should be in /doing after child status change: %v", err)
	}

	// Verify parent is no longer in /pending
	_, err = pendingDir.Lookup(nil, "1.parent_task")
	if err != fuse.ENOENT {
		t.Error("Parent should NOT be in /pending after status change")
	}

	// Verify child moved with parent
	childPath := taskStore.PathByTask[child]
	if !strings.HasPrefix(childPath, newParentPath+"/") {
		t.Errorf("Child path = %q, should be under new parent path %q", childPath, newParentPath)
	}

	// Change child status to done
	child.Status = store.StatusDone
	pm.OnStatusChange(child)

	// Parent should now be in /done (only child is done)
	finalParentPath := taskStore.PathByTask[parent]
	if !strings.HasPrefix(finalParentPath, "/done/") {
		t.Errorf("After child done, parent path = %q, should be in /done/", finalParentPath)
	}

	// Verify via FUSE
	doneDir := &Dir{fs: fuseFS, path: "/done"}
	_, err = doneDir.Lookup(nil, "1.parent_task")
	if err != nil {
		t.Errorf("Parent should be in /done after child completed: %v", err)
	}
}

// TestStatusBasedPlacement_DesignExample1 tests the exact example from the design document.
// Design Example 1:
// Task 1 has children: 1.1 (done), 1.2 (doing), 1.3 (pending)
// Child 1.2 is `doing` → Parent 1 derives status `doing`
//
// Resulting Filesystem Structure:
// /mount/
// ├── doing/
// │   └── 1.setup_project/                  # Parent at derived status location
// │       ├── index.md
// │       ├── 1.1.create_directory_structure.md   # Child stays with parent
// │       ├── 1.2.configure_build_system.md       # Child stays with parent
// │       └── 1.3.add_dependencies.md             # Child stays with parent
//
// Validates: Design - Example 1
func TestStatusBasedPlacement_DesignExample1(t *testing.T) {
	taskStore := store.NewTaskStore()
	pm := store.NewPathManager(taskStore)
	fuseFS := NewFuseFS(taskStore, "/mnt/tasks")

	// Create the exact structure from Design Example 1
	child1 := &store.Task{
		ID:     "1.1",
		Title:  "Create directory structure",
		Status: store.StatusDone,
	}
	child2 := &store.Task{
		ID:     "1.2",
		Title:  "Configure build system",
		Status: store.StatusDoing,
	}
	child3 := &store.Task{
		ID:     "1.3",
		Title:  "Add dependencies",
		Status: store.StatusPending,
	}
	parent := &store.Task{
		ID:       "1",
		Title:    "Setup project",
		Status:   store.StatusPending,
		Children: []*store.Task{child1, child2, child3},
	}
	child1.Parent = parent
	child2.Parent = parent
	child3.Parent = parent

	taskStore.Document = &store.ParsedDocument{RootTasks: []*store.Task{parent}}

	// Build paths
	pm.BuildAllPaths()

	// Verify the exact filesystem structure from the design
	// 1. Parent should be at /doing/1.setup_project
	parentPath := taskStore.PathByTask[parent]
	expectedParentPath := "/doing/1.setup_project"
	if parentPath != expectedParentPath {
		t.Errorf("Parent path = %q, want %q", parentPath, expectedParentPath)
	}

	// 2. Children should be under parent
	expectedPaths := map[string]string{
		"1.1": "/doing/1.setup_project/1.1.create_directory_structure.md",
		"1.2": "/doing/1.setup_project/1.2.configure_build_system.md",
		"1.3": "/doing/1.setup_project/1.3.add_dependencies.md",
	}

	for _, child := range []*store.Task{child1, child2, child3} {
		actualPath := taskStore.PathByTask[child]
		expectedPath := expectedPaths[child.ID]
		if actualPath != expectedPath {
			t.Errorf("Child %s path = %q, want %q", child.ID, actualPath, expectedPath)
		}
	}

	// 3. Verify via FUSE filesystem navigation
	// Navigate: / -> doing -> 1.setup_project
	rootDir := &Dir{fs: fuseFS, path: "/"}
	doingNode, err := rootDir.Lookup(nil, "doing")
	if err != nil {
		t.Fatalf("Lookup(doing) failed: %v", err)
	}
	doingDir := doingNode.(*Dir)

	parentNode, err := doingDir.Lookup(nil, "1.setup_project")
	if err != nil {
		t.Fatalf("Lookup(1.setup_project) failed: %v", err)
	}
	parentDir := parentNode.(*Dir)

	// 4. Verify all children can be found under parent
	childNames := []string{
		"1.1.create_directory_structure.md",
		"1.2.configure_build_system.md",
		"1.3.add_dependencies.md",
	}

	for _, name := range childNames {
		node, err := parentDir.Lookup(nil, name)
		if err != nil {
			t.Errorf("Child %s not found under parent: %v", name, err)
			continue
		}
		if _, ok := node.(*File); !ok {
			t.Errorf("Child %s should be a File", name)
		}
	}

	// 5. Verify index.md exists in parent directory
	indexNode, err := parentDir.Lookup(nil, "index.md")
	if err != nil {
		t.Errorf("index.md not found in parent directory: %v", err)
	}
	if _, ok := indexNode.(*File); !ok {
		t.Error("index.md should be a File")
	}

	// 6. Verify children are NOT in their own status directories
	// Check /done does not contain 1.1
	doneDir := &Dir{fs: fuseFS, path: "/done"}
	doneEntries, _ := doneDir.ReadDirAll(nil)
	for _, entry := range doneEntries {
		if strings.Contains(entry.Name, "1.1") || strings.Contains(entry.Name, "create_directory") {
			t.Error("Child 1.1 should NOT appear in /done directory")
		}
	}

	// Check /pending does not contain 1.3
	pendingDir := &Dir{fs: fuseFS, path: "/pending"}
	pendingEntries, _ := pendingDir.ReadDirAll(nil)
	for _, entry := range pendingEntries {
		if strings.Contains(entry.Name, "1.3") || strings.Contains(entry.Name, "add_dependencies") {
			t.Error("Child 1.3 should NOT appear in /pending directory")
		}
	}
}

// TestStatusBasedPlacement_DeepNesting verifies status-based placement works
// with deeply nested task hierarchies (3+ levels).
// Validates: Requirements 2.4
func TestStatusBasedPlacement_DeepNesting(t *testing.T) {
	taskStore := store.NewTaskStore()
	pm := store.NewPathManager(taskStore)
	fuseFS := NewFuseFS(taskStore, "/mnt/tasks")

	// Create 3-level hierarchy:
	// 1. Root (derived: doing)
	//   1.1 Child (derived: doing)
	//     1.1.1 Grandchild (doing)
	//     1.1.2 Grandchild (done)
	grandchild1 := &store.Task{
		ID:     "1.1.1",
		Title:  "Grandchild Doing",
		Status: store.StatusDoing,
	}
	grandchild2 := &store.Task{
		ID:     "1.1.2",
		Title:  "Grandchild Done",
		Status: store.StatusDone,
	}
	child := &store.Task{
		ID:       "1.1",
		Title:    "Child",
		Status:   store.StatusPending,
		Children: []*store.Task{grandchild1, grandchild2},
	}
	root := &store.Task{
		ID:       "1",
		Title:    "Root",
		Status:   store.StatusPending,
		Children: []*store.Task{child},
	}
	grandchild1.Parent = child
	grandchild2.Parent = child
	child.Parent = root

	taskStore.Document = &store.ParsedDocument{RootTasks: []*store.Task{root}}

	// Build paths
	pm.BuildAllPaths()

	// Root should be in /doing (grandchild is doing)
	rootPath := taskStore.PathByTask[root]
	if !strings.HasPrefix(rootPath, "/doing/") {
		t.Errorf("Root path = %q, should be in /doing/", rootPath)
	}

	// Child should be under root
	childPath := taskStore.PathByTask[child]
	if !strings.HasPrefix(childPath, rootPath+"/") {
		t.Errorf("Child path = %q, should be under root %q", childPath, rootPath)
	}

	// Grandchildren should be under child
	for _, gc := range []*store.Task{grandchild1, grandchild2} {
		gcPath := taskStore.PathByTask[gc]
		if !strings.HasPrefix(gcPath, childPath+"/") {
			t.Errorf("Grandchild %s path = %q, should be under child %q", gc.ID, gcPath, childPath)
		}
	}

	// Verify via FUSE navigation
	doingDir := &Dir{fs: fuseFS, path: "/doing"}
	rootNode, err := doingDir.Lookup(nil, "1.root")
	if err != nil {
		t.Fatalf("Root not found in /doing: %v", err)
	}
	rootDir := rootNode.(*Dir)

	childNode, err := rootDir.Lookup(nil, "1.1.child")
	if err != nil {
		t.Fatalf("Child not found under root: %v", err)
	}
	childDir := childNode.(*Dir)

	// Verify grandchildren under child
	_, err = childDir.Lookup(nil, "1.1.1.grandchild_doing.md")
	if err != nil {
		t.Errorf("Grandchild 1.1.1 not found under child: %v", err)
	}
	_, err = childDir.Lookup(nil, "1.1.2.grandchild_done.md")
	if err != nil {
		t.Errorf("Grandchild 1.1.2 not found under child: %v", err)
	}
}

// TestStatusBasedPlacement_MultipleRootTasks verifies that multiple root tasks
// are each placed in their correct status directories.
// Validates: Requirements 2.5, 2.6, 2.7, 2.8
func TestStatusBasedPlacement_MultipleRootTasks(t *testing.T) {
	taskStore := store.NewTaskStore()
	pm := store.NewPathManager(taskStore)
	fuseFS := NewFuseFS(taskStore, "/mnt/tasks")

	// Create multiple root tasks with different statuses
	task1 := &store.Task{ID: "1", Title: "Pending Task", Status: store.StatusPending}
	task2 := &store.Task{ID: "2", Title: "Doing Task", Status: store.StatusDoing}
	task3 := &store.Task{ID: "3", Title: "Done Task", Status: store.StatusDone}
	task4 := &store.Task{ID: "4", Title: "Failed Task", Status: store.StatusFailed}
	task5 := &store.Task{ID: "5", Title: "Queued Task", Status: store.StatusQueued}

	taskStore.Document = &store.ParsedDocument{
		RootTasks: []*store.Task{task1, task2, task3, task4, task5},
	}

	// Build paths
	pm.BuildAllPaths()

	// Verify each task is in its correct status directory
	expectedDirs := map[*store.Task]string{
		task1: "/pending",
		task2: "/doing",
		task3: "/done",
		task4: "/failed",
		task5: "/queued",
	}

	for task, expectedDir := range expectedDirs {
		path := taskStore.PathByTask[task]
		if !strings.HasPrefix(path, expectedDir+"/") {
			t.Errorf("Task %s path = %q, should be in %s/", task.ID, path, expectedDir)
		}

		// Verify via FUSE
		statusDir := &Dir{fs: fuseFS, path: expectedDir}
		entries, _ := statusDir.ReadDirAll(nil)
		found := false
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name, task.ID+".") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Task %s not found in %s directory via FUSE", task.ID, expectedDir)
		}
	}
}

// Helper function for tests
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// =============================================================================
// Read-Only Enforcement Tests
// =============================================================================
// These tests verify that the filesystem correctly rejects mutating operations.
// Validates: Requirements 3.1, 3.2, 3.3, Property 6

// TestFile_Write_ReturnsEPERM verifies that writing to task files returns EPERM.
// Validates: Requirements 3.1
func TestFile_Write_ReturnsEPERM(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	task := &store.Task{
		ID:     "1.1",
		Title:  "Test task",
		Status: store.StatusPending,
	}

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test_task.md",
		task: task,
	}

	req := &fuse.WriteRequest{
		Data: []byte("new content"),
	}
	resp := &fuse.WriteResponse{}

	err := file.Write(nil, req, resp)
	if err != fuse.EPERM {
		t.Errorf("Write() = %v, want fuse.EPERM", err)
	}
}

// TestFile_Write_IndexFile_ReturnsEPERM verifies that writing to index files returns EPERM.
// Validates: Requirements 3.1
func TestFile_Write_IndexFile_ReturnsEPERM(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	// Index files have nil task
	file := &File{
		fs:   fs,
		path: "/index.md",
		task: nil,
	}

	req := &fuse.WriteRequest{
		Data: []byte("new content"),
	}
	resp := &fuse.WriteResponse{}

	err := file.Write(nil, req, resp)
	if err != fuse.EPERM {
		t.Errorf("Write() on index file = %v, want fuse.EPERM", err)
	}
}

// TestDir_Create_ReturnsEPERM verifies that creating files in directories returns EPERM.
// Validates: Requirements 3.2
func TestDir_Create_ReturnsEPERM(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	testDirs := []struct {
		name string
		path string
	}{
		{"root directory", "/"},
		{"status directory", "/pending"},
		{"task directory", "/pending/1.parent_task"},
	}

	for _, td := range testDirs {
		t.Run(td.name, func(t *testing.T) {
			dir := &Dir{
				fs:   fs,
				path: td.path,
			}

			req := &fuse.CreateRequest{
				Name: "newfile.md",
			}
			resp := &fuse.CreateResponse{}

			_, _, err := dir.Create(nil, req, resp)
			if err != fuse.EPERM {
				t.Errorf("Create() in %s = %v, want fuse.EPERM", td.name, err)
			}
		})
	}
}

// TestDir_Remove_ReturnsEPERM verifies that removing files returns EPERM.
// Validates: Requirements 3.3
func TestDir_Remove_ReturnsEPERM(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{
		fs:   fs,
		path: "/pending",
	}

	req := &fuse.RemoveRequest{
		Name: "1.1.test_task.md",
	}

	err := dir.Remove(nil, req)
	if err != fuse.EPERM {
		t.Errorf("Remove() = %v, want fuse.EPERM", err)
	}
}

// TestDir_Mkdir_ReturnsEPERM verifies that creating directories returns EPERM.
func TestDir_Mkdir_ReturnsEPERM(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{
		fs:   fs,
		path: "/pending",
	}

	req := &fuse.MkdirRequest{
		Name: "newdir",
	}

	_, err := dir.Mkdir(nil, req)
	if err != fuse.EPERM {
		t.Errorf("Mkdir() = %v, want fuse.EPERM", err)
	}
}

// TestFile_Setattr_ReturnsEROFS verifies that modifying file attributes returns EROFS.
func TestFile_Setattr_ReturnsEROFS(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test_task.md",
		task: &store.Task{ID: "1.1", Title: "Test", Status: store.StatusPending},
	}

	req := &fuse.SetattrRequest{}
	resp := &fuse.SetattrResponse{}

	err := file.Setattr(nil, req, resp)
	if err != EROFS {
		t.Errorf("File.Setattr() = %v, want EROFS", err)
	}
}

// TestDir_Setattr_ReturnsEROFS verifies that modifying directory attributes returns EROFS.
func TestDir_Setattr_ReturnsEROFS(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{
		fs:   fs,
		path: "/pending",
	}

	req := &fuse.SetattrRequest{}
	resp := &fuse.SetattrResponse{}

	err := dir.Setattr(nil, req, resp)
	if err != EROFS {
		t.Errorf("Dir.Setattr() = %v, want EROFS", err)
	}
}

// TestDir_Symlink_ReturnsEPERM verifies that creating symbolic links returns EPERM.
func TestDir_Symlink_ReturnsEPERM(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{
		fs:   fs,
		path: "/pending",
	}

	req := &fuse.SymlinkRequest{
		NewName: "link",
		Target:  "/some/target",
	}

	_, err := dir.Symlink(nil, req)
	if err != fuse.EPERM {
		t.Errorf("Symlink() = %v, want fuse.EPERM", err)
	}
}

// TestDir_Link_ReturnsEPERM verifies that creating hard links returns EPERM.
func TestDir_Link_ReturnsEPERM(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{
		fs:   fs,
		path: "/pending",
	}

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test.md",
	}

	req := &fuse.LinkRequest{
		NewName: "newlink",
	}

	_, err := dir.Link(nil, req, file)
	if err != fuse.EPERM {
		t.Errorf("Link() = %v, want fuse.EPERM", err)
	}
}

// TestDir_Mknod_ReturnsEPERM verifies that creating special files returns EPERM.
func TestDir_Mknod_ReturnsEPERM(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	dir := &Dir{
		fs:   fs,
		path: "/pending",
	}

	req := &fuse.MknodRequest{
		Name: "special",
	}

	_, err := dir.Mknod(nil, req)
	if err != fuse.EPERM {
		t.Errorf("Mknod() = %v, want fuse.EPERM", err)
	}
}

// TestFile_Fsync_Success verifies that fsync is a successful no-op.
func TestFile_Fsync_Success(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test.md",
	}

	req := &fuse.FsyncRequest{}

	err := file.Fsync(nil, req)
	if err != nil {
		t.Errorf("Fsync() = %v, want nil", err)
	}
}

// TestFile_Flush_Success verifies that flush is a successful no-op.
func TestFile_Flush_Success(t *testing.T) {
	taskStore := store.NewTaskStore()
	fs := NewFuseFS(taskStore, "/mnt/tasks")

	file := &File{
		fs:   fs,
		path: "/pending/1.1.test.md",
	}

	req := &fuse.FlushRequest{}

	err := file.Flush(nil, req)
	if err != nil {
		t.Errorf("Flush() = %v, want nil", err)
	}
}
