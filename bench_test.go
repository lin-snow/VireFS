package virefs

import (
	"context"
	"strings"
	"testing"
)

func BenchmarkCleanKey(b *testing.B) {
	for b.Loop() {
		_, _ = CleanKey("a/b/c/d.txt")
	}
}

func BenchmarkCleanKey_Dirty(b *testing.B) {
	for b.Loop() {
		_, _ = CleanKey("//a/./b///c/d.txt/")
	}
}

func BenchmarkSchemaResolve(b *testing.B) {
	s := NewSchema(
		RouteByExt("images/", ".jpg", ".jpeg", ".png", ".gif", ".webp"),
		RouteByExt("docs/", ".pdf", ".doc", ".docx"),
		DefaultRoute("other/"),
	)
	for b.Loop() {
		s.Resolve("photo.jpg")
	}
}

func BenchmarkLocalFS_Put(b *testing.B) {
	dir := b.TempDir()
	fs, err := NewLocalFS(dir)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	data := strings.NewReader("benchmark-payload")

	for b.Loop() {
		data.Reset("benchmark-payload")
		_ = fs.Put(ctx, "bench.txt", data)
	}
}

func BenchmarkLocalFS_Get(b *testing.B) {
	dir := b.TempDir()
	fs, err := NewLocalFS(dir)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	_ = fs.Put(ctx, "bench.txt", strings.NewReader("benchmark-payload"))

	for b.Loop() {
		rc, err := fs.Get(ctx, "bench.txt")
		if err != nil {
			b.Fatal(err)
		}
		rc.Close()
	}
}

func BenchmarkLocalFS_Stat(b *testing.B) {
	dir := b.TempDir()
	fs, err := NewLocalFS(dir)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	_ = fs.Put(ctx, "bench.txt", strings.NewReader("benchmark-payload"))

	for b.Loop() {
		_, _ = fs.Stat(ctx, "bench.txt")
	}
}
