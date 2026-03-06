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
