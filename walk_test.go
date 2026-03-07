package virefs

import (
	"context"
	"sort"
	"strings"
	"testing"
)

func TestWalk(t *testing.T) {
	dir := t.TempDir()
	fs := mustNewLocalFS(t, dir)
	ctx := context.Background()

	_ = fs.Put(ctx, "a.txt", strings.NewReader("a"))
	_ = fs.Put(ctx, "sub/b.txt", strings.NewReader("b"))
	_ = fs.Put(ctx, "sub/deep/c.txt", strings.NewReader("c"))

	var visited []string
	err := Walk(ctx, fs, "", func(key string, info FileInfo, err error) error {
		if err != nil {
			return err
		}
		visited = append(visited, key)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	sort.Strings(visited)
	want := []string{"a.txt", "sub", "sub/b.txt", "sub/deep", "sub/deep/c.txt"}
	if len(visited) != len(want) {
		t.Fatalf("Walk visited %v, want %v", visited, want)
	}
	for i, v := range visited {
		if v != want[i] {
			t.Fatalf("Walk visited[%d] = %q, want %q", i, v, want[i])
		}
	}
}

func TestWalk_SkipDir(t *testing.T) {
	dir := t.TempDir()
	fs := mustNewLocalFS(t, dir)
	ctx := context.Background()

	_ = fs.Put(ctx, "a.txt", strings.NewReader("a"))
	_ = fs.Put(ctx, "skip/b.txt", strings.NewReader("b"))
	_ = fs.Put(ctx, "keep/c.txt", strings.NewReader("c"))

	var visited []string
	err := Walk(ctx, fs, "", func(key string, info FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir && key == "skip" {
			return ErrSkipDir
		}
		visited = append(visited, key)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	sort.Strings(visited)
	for _, v := range visited {
		if strings.HasPrefix(v, "skip/") {
			t.Fatalf("Walk should have skipped 'skip/' directory, but visited %q", v)
		}
	}
	found := false
	for _, v := range visited {
		if v == "keep/c.txt" {
			found = true
		}
	}
	if !found {
		t.Fatal("Walk should have visited keep/c.txt")
	}
}

func TestWalk_ContextCancelled(t *testing.T) {
	dir := t.TempDir()
	fs := mustNewLocalFS(t, dir)
	ctx := context.Background()

	_ = fs.Put(ctx, "a.txt", strings.NewReader("a"))
	_ = fs.Put(ctx, "b.txt", strings.NewReader("b"))

	ctx2, cancel := context.WithCancel(ctx)
	cancel()

	err := Walk(ctx2, fs, "", func(key string, info FileInfo, err error) error {
		if err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		t.Fatal("Walk with cancelled context should fail")
	}
}
