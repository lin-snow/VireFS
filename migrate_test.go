package virefs

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestMigrate_Basic(t *testing.T) {
	src := mustNewLocalFS(t, t.TempDir())
	dst := mustNewLocalFS(t, t.TempDir())
	ctx := context.Background()

	_ = src.Put(ctx, "a.txt", strings.NewReader("aaa"))
	_ = src.Put(ctx, "sub/b.txt", strings.NewReader("bbb"))

	result, err := Migrate(ctx, src, "", dst, "", WithConflictPolicy(ConflictOverwrite))
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if result.Copied != 2 {
		t.Fatalf("Copied = %d, want 2", result.Copied)
	}
	if result.Total != 2 {
		t.Fatalf("Total = %d, want 2", result.Total)
	}

	rc, err := dst.Get(ctx, "a.txt")
	if err != nil {
		t.Fatalf("Get a.txt from dst: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "aaa" {
		t.Fatalf("a.txt = %q, want %q", data, "aaa")
	}

	rc, err = dst.Get(ctx, "sub/b.txt")
	if err != nil {
		t.Fatalf("Get sub/b.txt from dst: %v", err)
	}
	data, _ = io.ReadAll(rc)
	rc.Close()
	if string(data) != "bbb" {
		t.Fatalf("sub/b.txt = %q, want %q", data, "bbb")
	}
}

func TestMigrate_WithPrefix(t *testing.T) {
	src := mustNewLocalFS(t, t.TempDir())
	dst := mustNewLocalFS(t, t.TempDir())
	ctx := context.Background()

	_ = src.Put(ctx, "data/x.txt", strings.NewReader("x"))
	_ = src.Put(ctx, "data/y.txt", strings.NewReader("y"))
	_ = src.Put(ctx, "other.txt", strings.NewReader("o"))

	result, err := Migrate(ctx, src, "data", dst, "backup", WithConflictPolicy(ConflictOverwrite))
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if result.Copied != 2 {
		t.Fatalf("Copied = %d, want 2", result.Copied)
	}

	rc, err := dst.Get(ctx, "backup/x.txt")
	if err != nil {
		t.Fatalf("Get backup/x.txt: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "x" {
		t.Fatalf("backup/x.txt = %q, want %q", data, "x")
	}

	ok, _ := dst.Exists(ctx, "backup/other.txt")
	if ok {
		t.Fatal("other.txt should not be copied (outside srcPrefix)")
	}
}

func TestMigrate_ConflictError(t *testing.T) {
	src := mustNewLocalFS(t, t.TempDir())
	dst := mustNewLocalFS(t, t.TempDir())
	ctx := context.Background()

	_ = src.Put(ctx, "conflict.txt", strings.NewReader("new"))
	_ = dst.Put(ctx, "conflict.txt", strings.NewReader("old"))

	_, err := Migrate(ctx, src, "", dst, "", WithConflictPolicy(ConflictError))
	if err == nil {
		t.Fatal("Migrate should fail with ConflictError when destination exists")
	}
	if !errors.Is(err, ErrAlreadyExist) {
		t.Fatalf("error = %v, want ErrAlreadyExist", err)
	}
}

func TestMigrate_ConflictSkip(t *testing.T) {
	src := mustNewLocalFS(t, t.TempDir())
	dst := mustNewLocalFS(t, t.TempDir())
	ctx := context.Background()

	_ = src.Put(ctx, "keep.txt", strings.NewReader("new"))
	_ = src.Put(ctx, "fresh.txt", strings.NewReader("fresh"))
	_ = dst.Put(ctx, "keep.txt", strings.NewReader("old"))

	result, err := Migrate(ctx, src, "", dst, "", WithConflictPolicy(ConflictSkip))
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if result.Skipped != 1 {
		t.Fatalf("Skipped = %d, want 1", result.Skipped)
	}
	if result.Copied != 1 {
		t.Fatalf("Copied = %d, want 1", result.Copied)
	}

	rc, err := dst.Get(ctx, "keep.txt")
	if err != nil {
		t.Fatalf("Get keep.txt: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "old" {
		t.Fatalf("keep.txt should be unchanged, got %q", data)
	}
}

func TestMigrate_ConflictOverwrite(t *testing.T) {
	src := mustNewLocalFS(t, t.TempDir())
	dst := mustNewLocalFS(t, t.TempDir())
	ctx := context.Background()

	_ = src.Put(ctx, "file.txt", strings.NewReader("new-content"))
	_ = dst.Put(ctx, "file.txt", strings.NewReader("old-content"))

	result, err := Migrate(ctx, src, "", dst, "", WithConflictPolicy(ConflictOverwrite))
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if result.Copied != 1 {
		t.Fatalf("Copied = %d, want 1", result.Copied)
	}

	rc, err := dst.Get(ctx, "file.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "new-content" {
		t.Fatalf("file.txt = %q, want %q", data, "new-content")
	}
}

func TestMigrate_DryRun(t *testing.T) {
	src := mustNewLocalFS(t, t.TempDir())
	dst := mustNewLocalFS(t, t.TempDir())
	ctx := context.Background()

	_ = src.Put(ctx, "a.txt", strings.NewReader("a"))
	_ = src.Put(ctx, "b.txt", strings.NewReader("b"))

	result, err := Migrate(ctx, src, "", dst, "", WithDryRun(), WithConflictPolicy(ConflictOverwrite))
	if err != nil {
		t.Fatalf("Migrate DryRun: %v", err)
	}
	if result.Copied != 2 {
		t.Fatalf("DryRun Copied = %d, want 2", result.Copied)
	}

	ok, _ := dst.Exists(ctx, "a.txt")
	if ok {
		t.Fatal("DryRun should not actually copy files")
	}
}

func TestMigrate_ProgressFunc(t *testing.T) {
	src := mustNewLocalFS(t, t.TempDir())
	dst := mustNewLocalFS(t, t.TempDir())
	ctx := context.Background()

	_ = src.Put(ctx, "x.txt", strings.NewReader("x"))
	_ = src.Put(ctx, "y.txt", strings.NewReader("y"))

	var calls []MigrateProgress
	_, err := Migrate(ctx, src, "", dst, "",
		WithConflictPolicy(ConflictOverwrite),
		WithProgressFunc(func(p MigrateProgress) {
			calls = append(calls, p)
		}),
	)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("progress called %d times, want 2", len(calls))
	}
	last := calls[len(calls)-1]
	if last.Copied != 2 || last.Total != 2 {
		t.Fatalf("last progress = {Copied:%d Total:%d}, want {2 2}", last.Copied, last.Total)
	}
}

func TestMigrate_KeyFunc(t *testing.T) {
	src := mustNewLocalFS(t, t.TempDir())
	dst := mustNewLocalFS(t, t.TempDir())
	ctx := context.Background()

	_ = src.Put(ctx, "file.txt", strings.NewReader("data"))

	_, err := Migrate(ctx, src, "", dst, "",
		WithConflictPolicy(ConflictOverwrite),
		WithMigrateKeyFunc(func(key string) string {
			return "renamed_" + key
		}),
	)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	ok, _ := dst.Exists(ctx, "renamed_file.txt")
	if !ok {
		t.Fatal("destination should have renamed key")
	}

	ok, _ = dst.Exists(ctx, "file.txt")
	if ok {
		t.Fatal("original key should not exist at destination")
	}
}

func TestMigrate_CrossBackend(t *testing.T) {
	local := mustNewLocalFS(t, t.TempDir())
	fake := newFakeS3()
	obj := NewObjectFS(fake, "bucket")
	ctx := context.Background()

	_ = local.Put(ctx, "upload.txt", strings.NewReader("to-s3"))

	result, err := Migrate(ctx, local, "", obj, "imported", WithConflictPolicy(ConflictOverwrite))
	if err != nil {
		t.Fatalf("Migrate local->s3: %v", err)
	}
	if result.Copied != 1 {
		t.Fatalf("Copied = %d, want 1", result.Copied)
	}

	rc, err := obj.Get(ctx, "imported/upload.txt")
	if err != nil {
		t.Fatalf("Get from s3: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "to-s3" {
		t.Fatalf("content = %q, want %q", data, "to-s3")
	}
}

func TestMigrate_EmptySource(t *testing.T) {
	src := mustNewLocalFS(t, t.TempDir())
	dst := mustNewLocalFS(t, t.TempDir())
	ctx := context.Background()

	result, err := Migrate(ctx, src, "", dst, "")
	if err != nil {
		t.Fatalf("Migrate empty: %v", err)
	}
	if result.Copied != 0 || result.Total != 0 {
		t.Fatalf("result = %+v, want zero", result)
	}
}
