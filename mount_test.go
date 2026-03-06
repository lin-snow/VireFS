package virefs

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestMountTable_Routing(t *testing.T) {
	dir := t.TempDir()
	local := NewLocalFS(dir)
	fake := newFakeS3()
	obj := NewObjectFS(fake, "bucket", "")

	mt := NewMountTable()
	if err := mt.Mount("local", local); err != nil {
		t.Fatal(err)
	}
	if err := mt.Mount("s3", obj); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Put via mount table
	_ = mt.Put(ctx, "local/greet.txt", strings.NewReader("hi"))
	_ = mt.Put(ctx, "s3/data.bin", strings.NewReader("01"))

	// Get from local
	rc, err := mt.Get(ctx, "local/greet.txt")
	if err != nil {
		t.Fatalf("Get local: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "hi" {
		t.Fatalf("Get local = %q, want %q", data, "hi")
	}

	// Get from s3
	rc, err = mt.Get(ctx, "s3/data.bin")
	if err != nil {
		t.Fatalf("Get s3: %v", err)
	}
	data, _ = io.ReadAll(rc)
	rc.Close()
	if string(data) != "01" {
		t.Fatalf("Get s3 = %q, want %q", data, "01")
	}
}

func TestMountTable_UnmountedPrefix(t *testing.T) {
	mt := NewMountTable()
	ctx := context.Background()
	_, err := mt.Get(ctx, "unknown/file.txt")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("unmounted prefix error = %v, want ErrNotFound", err)
	}
}

func TestMountTable_ListRoot(t *testing.T) {
	mt := NewMountTable()
	_ = mt.Mount("a", NewLocalFS(t.TempDir()))
	_ = mt.Mount("b", NewLocalFS(t.TempDir()))

	result, err := mt.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List root: %v", err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("List root got %d, want 2", len(result.Files))
	}
}

func TestMountTable_InvalidPrefix(t *testing.T) {
	mt := NewMountTable()
	if err := mt.Mount("a/b", NewLocalFS(t.TempDir())); err == nil {
		t.Fatal("mount with slash should fail")
	}
	if err := mt.Mount("", NewLocalFS(t.TempDir())); err == nil {
		t.Fatal("mount with empty should fail")
	}
}
