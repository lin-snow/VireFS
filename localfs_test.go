package virefs

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalFS_PutGetDeleteStat(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir)
	ctx := context.Background()

	// Put
	if err := fs.Put(ctx, "hello.txt", strings.NewReader("world")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Get
	rc, err := fs.Get(ctx, "hello.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "world" {
		t.Fatalf("Get content = %q, want %q", data, "world")
	}

	// Stat
	info, err := fs.Stat(ctx, "hello.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size != 5 {
		t.Fatalf("Stat size = %d, want 5", info.Size)
	}

	// Delete
	if err := fs.Delete(ctx, "hello.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Get after delete → ErrNotFound
	_, err = fs.Get(ctx, "hello.txt")
	if err == nil {
		t.Fatal("Get after delete should fail")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete error = %v, want ErrNotFound", err)
	}
}

func TestLocalFS_List(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir)
	ctx := context.Background()

	_ = fs.Put(ctx, "a.txt", strings.NewReader("a"))
	_ = fs.Put(ctx, "sub/b.txt", strings.NewReader("b"))

	result, err := fs.List(ctx, "")
	if err != nil {
		t.Fatalf("List root: %v", err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("List root got %d entries, want 2", len(result.Files))
	}

	result, err = fs.List(ctx, "sub")
	if err != nil {
		t.Fatalf("List sub: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("List sub got %d entries, want 1", len(result.Files))
	}
	if result.Files[0].Key != "sub/b.txt" {
		t.Fatalf("List sub key = %q, want %q", result.Files[0].Key, "sub/b.txt")
	}
}

func TestLocalFS_NestedPut(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir)
	ctx := context.Background()

	if err := fs.Put(ctx, "a/b/c/d.txt", strings.NewReader("deep")); err != nil {
		t.Fatalf("nested Put: %v", err)
	}
	rc, err := fs.Get(ctx, "a/b/c/d.txt")
	if err != nil {
		t.Fatalf("nested Get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "deep" {
		t.Fatalf("nested Get content = %q, want %q", data, "deep")
	}
}

func TestLocalFS_WithKeyFunc(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir, WithLocalKeyFunc(func(key string) string {
		return "transformed/" + key
	}))
	ctx := context.Background()

	if err := fs.Put(ctx, "note.txt", strings.NewReader("hello")); err != nil {
		t.Fatalf("Put with KeyFunc: %v", err)
	}

	rc, err := fs.Get(ctx, "note.txt")
	if err != nil {
		t.Fatalf("Get with KeyFunc: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "hello" {
		t.Fatalf("Get content = %q, want %q", data, "hello")
	}

	plain := NewLocalFS(dir)
	rc, err = plain.Get(ctx, "transformed/note.txt")
	if err != nil {
		t.Fatalf("plain Get transformed path: %v", err)
	}
	data, _ = io.ReadAll(rc)
	rc.Close()
	if string(data) != "hello" {
		t.Fatalf("plain Get content = %q, want %q", data, "hello")
	}
}

func TestLocalFS_Access(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir)
	ctx := context.Background()

	_ = fs.Put(ctx, "doc/readme.txt", strings.NewReader("hello"))

	info, err := fs.Access(ctx, "doc/readme.txt")
	if err != nil {
		t.Fatalf("Access: %v", err)
	}
	if info.Path == "" {
		t.Fatal("Access.Path should be non-empty for LocalFS")
	}
	if info.URL != "" {
		t.Fatal("Access.URL should be empty for LocalFS")
	}
	if !strings.HasSuffix(info.Path, "doc/readme.txt") {
		t.Fatalf("Access.Path = %q, want suffix doc/readme.txt", info.Path)
	}
}

func TestLocalFS_AccessWithKeyFunc(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir, WithLocalKeyFunc(func(key string) string {
		return "v2/" + key
	}))
	ctx := context.Background()

	info, err := fs.Access(ctx, "file.txt")
	if err != nil {
		t.Fatalf("Access with KeyFunc: %v", err)
	}
	if !strings.HasSuffix(info.Path, "v2/file.txt") {
		t.Fatalf("Access.Path = %q, want suffix v2/file.txt", info.Path)
	}
}

func TestLocalFS_TraversalRejected(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir)
	ctx := context.Background()

	_, err := fs.Get(ctx, "../../etc/passwd")
	if err == nil {
		t.Fatal("traversal should be rejected")
	}
	if !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("traversal error = %v, want ErrInvalidKey", err)
	}
}

func TestLocalFS_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir, WithAtomicWrite())
	ctx := context.Background()

	if err := fs.Put(ctx, "atomic.txt", strings.NewReader("safe")); err != nil {
		t.Fatalf("AtomicWrite Put: %v", err)
	}
	rc, err := fs.Get(ctx, "atomic.txt")
	if err != nil {
		t.Fatalf("AtomicWrite Get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "safe" {
		t.Fatalf("AtomicWrite content = %q, want %q", data, "safe")
	}
}

func TestLocalFS_AtomicWriteNested(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir, WithAtomicWrite())
	ctx := context.Background()

	if err := fs.Put(ctx, "a/b/c.txt", strings.NewReader("deep")); err != nil {
		t.Fatalf("AtomicWrite nested Put: %v", err)
	}
	rc, err := fs.Get(ctx, "a/b/c.txt")
	if err != nil {
		t.Fatalf("AtomicWrite nested Get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "deep" {
		t.Fatalf("AtomicWrite nested content = %q, want %q", data, "deep")
	}
}

func TestLocalFS_Copy(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir)
	ctx := context.Background()

	_ = fs.Put(ctx, "original.txt", strings.NewReader("data"))

	if err := fs.Copy(ctx, "original.txt", "copied.txt"); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	rc, err := fs.Get(ctx, "copied.txt")
	if err != nil {
		t.Fatalf("Get copied: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "data" {
		t.Fatalf("Copy content = %q, want %q", data, "data")
	}

	rc, err = fs.Get(ctx, "original.txt")
	if err != nil {
		t.Fatalf("original should still exist: %v", err)
	}
	rc.Close()
}

func TestLocalFS_CopyNested(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir)
	ctx := context.Background()

	_ = fs.Put(ctx, "src/file.txt", strings.NewReader("nested"))

	if err := fs.Copy(ctx, "src/file.txt", "dst/dir/file.txt"); err != nil {
		t.Fatalf("Copy nested: %v", err)
	}

	rc, err := fs.Get(ctx, "dst/dir/file.txt")
	if err != nil {
		t.Fatalf("Get nested copy: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "nested" {
		t.Fatalf("Nested copy content = %q, want %q", data, "nested")
	}
}

func TestLocalFS_Exists(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir)
	ctx := context.Background()

	_ = fs.Put(ctx, "exists.txt", strings.NewReader("yes"))

	ok, err := Exists(ctx, fs, "exists.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !ok {
		t.Fatal("Exists should return true")
	}

	ok, err = Exists(ctx, fs, "nope.txt")
	if err != nil {
		t.Fatalf("Exists missing: %v", err)
	}
	if ok {
		t.Fatal("Exists should return false for missing key")
	}
}

func TestLocalFS_CopyHelper_SameBackend(t *testing.T) {
	dir := t.TempDir()
	fs := NewLocalFS(dir)
	ctx := context.Background()

	_ = fs.Put(ctx, "a.txt", strings.NewReader("hello"))

	if err := Copy(ctx, fs, "a.txt", fs, "b.txt"); err != nil {
		t.Fatalf("Copy helper same backend: %v", err)
	}

	rc, err := fs.Get(ctx, "b.txt")
	if err != nil {
		t.Fatalf("Get b.txt: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "hello" {
		t.Fatalf("content = %q, want %q", data, "hello")
	}
}
