package virefs

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestWithHooks_WrapGet(t *testing.T) {
	dir := t.TempDir()
	inner := mustNewLocalFS(t, dir)
	ctx := context.Background()

	_ = inner.Put(ctx, "file.txt", strings.NewReader("hello"))

	var gotKey string
	hfs := WithHooks(inner, Hooks{
		WrapGet: func(key string, rc io.ReadCloser) io.ReadCloser {
			gotKey = key
			return io.NopCloser(io.MultiReader(strings.NewReader("PREFIX:"), rc))
		},
	})

	rc, err := hfs.Get(ctx, "file.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()

	if gotKey != "file.txt" {
		t.Fatalf("WrapGet key = %q, want %q", gotKey, "file.txt")
	}
	if string(data) != "PREFIX:hello" {
		t.Fatalf("Get content = %q, want %q", data, "PREFIX:hello")
	}
}

func TestWithHooks_WrapPut(t *testing.T) {
	dir := t.TempDir()
	inner := mustNewLocalFS(t, dir)
	ctx := context.Background()

	var gotKey string
	hfs := WithHooks(inner, Hooks{
		WrapPut: func(key string, r io.Reader) io.Reader {
			gotKey = key
			return io.MultiReader(strings.NewReader("HEADER:"), r)
		},
	})

	err := hfs.Put(ctx, "out.txt", strings.NewReader("body"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	if gotKey != "out.txt" {
		t.Fatalf("WrapPut key = %q, want %q", gotKey, "out.txt")
	}

	rc, err := inner.Get(ctx, "out.txt")
	if err != nil {
		t.Fatalf("Get from inner: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "HEADER:body" {
		t.Fatalf("stored content = %q, want %q", data, "HEADER:body")
	}
}

func TestWithHooks_AfterStat(t *testing.T) {
	dir := t.TempDir()
	inner := mustNewLocalFS(t, dir)
	ctx := context.Background()

	_ = inner.Put(ctx, "note.txt", strings.NewReader("content"))

	hfs := WithHooks(inner, Hooks{
		AfterStat: func(key string, info *FileInfo) {
			info.ContentType = "custom/overridden"
		},
	})

	info, err := hfs.Stat(ctx, "note.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.ContentType != "custom/overridden" {
		t.Fatalf("ContentType = %q, want %q", info.ContentType, "custom/overridden")
	}
}

func TestWithHooks_OnDelete(t *testing.T) {
	dir := t.TempDir()
	inner := mustNewLocalFS(t, dir)
	ctx := context.Background()

	_ = inner.Put(ctx, "del.txt", strings.NewReader("bye"))

	var deletedKey string
	hfs := WithHooks(inner, Hooks{
		OnDelete: func(key string) {
			deletedKey = key
		},
	})

	if err := hfs.Delete(ctx, "del.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deletedKey != "del.txt" {
		t.Fatalf("OnDelete key = %q, want %q", deletedKey, "del.txt")
	}
}

func TestWithHooks_OnDelete_NotCalledOnError(t *testing.T) {
	dir := t.TempDir()
	inner := mustNewLocalFS(t, dir)
	ctx := context.Background()

	called := false
	hfs := WithHooks(inner, Hooks{
		OnDelete: func(key string) { called = true },
	})

	_ = hfs.Delete(ctx, "nonexistent.txt")
	if called {
		t.Fatal("OnDelete should not be called when Delete fails")
	}
}

func TestWithHooks_Passthrough(t *testing.T) {
	dir := t.TempDir()
	inner := mustNewLocalFS(t, dir)
	ctx := context.Background()

	_ = inner.Put(ctx, "a.txt", strings.NewReader("a"))
	_ = inner.Put(ctx, "sub/b.txt", strings.NewReader("b"))

	hfs := WithHooks(inner, Hooks{})

	result, err := hfs.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("List got %d entries, want 2", len(result.Files))
	}

	info, err := hfs.Access(ctx, "a.txt")
	if err != nil {
		t.Fatalf("Access: %v", err)
	}
	if info.Path == "" {
		t.Fatal("Access.Path should be non-empty")
	}
}

func TestWithHooks_NilHooks(t *testing.T) {
	dir := t.TempDir()
	inner := mustNewLocalFS(t, dir)
	ctx := context.Background()

	hfs := WithHooks(inner, Hooks{})

	if err := hfs.Put(ctx, "test.txt", strings.NewReader("data")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	rc, err := hfs.Get(ctx, "test.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "data" {
		t.Fatalf("Get content = %q, want %q", data, "data")
	}

	info, err := hfs.Stat(ctx, "test.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size != 4 {
		t.Fatalf("Stat size = %d, want 4", info.Size)
	}

	if err := hfs.Delete(ctx, "test.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestWithHooks_Unwrap(t *testing.T) {
	dir := t.TempDir()
	inner := mustNewLocalFS(t, dir)

	hfs := WithHooks(inner, Hooks{})

	if hfs.Unwrap() != inner {
		t.Fatal("Unwrap should return the original FS")
	}
}

func TestWithHooks_MultipleHooks(t *testing.T) {
	dir := t.TempDir()
	inner := mustNewLocalFS(t, dir)
	ctx := context.Background()

	var buf bytes.Buffer
	hfs := WithHooks(inner, Hooks{
		WrapPut: func(key string, r io.Reader) io.Reader {
			return io.TeeReader(r, &buf)
		},
		OnDelete: func(key string) {
			buf.Reset()
		},
	})

	_ = hfs.Put(ctx, "traced.txt", strings.NewReader("captured"))
	if buf.String() != "captured" {
		t.Fatalf("TeeReader captured = %q, want %q", buf.String(), "captured")
	}

	_ = hfs.Delete(ctx, "traced.txt")
	if buf.Len() != 0 {
		t.Fatal("OnDelete should have reset the buffer")
	}
}
