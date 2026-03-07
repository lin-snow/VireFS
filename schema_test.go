package virefs

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestSchema_RouteByExt(t *testing.T) {
	s := NewSchema(
		RouteByExt("images/", ".jpg", ".jpeg", ".png"),
		RouteByExt("docs/", ".pdf", ".docx"),
	)

	tests := []struct {
		key  string
		want string
	}{
		{"cat.jpg", "images/cat.jpg"},
		{"photo.jpeg", "images/photo.jpeg"},
		{"icon.png", "images/icon.png"},
		{"report.pdf", "docs/report.pdf"},
		{"essay.docx", "docs/essay.docx"},
		{"readme.txt", "readme.txt"},
	}
	for _, tt := range tests {
		got := s.Resolve(tt.key)
		if got != tt.want {
			t.Errorf("Resolve(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestSchema_CaseInsensitive(t *testing.T) {
	s := NewSchema(RouteByExt("images/", ".jpg"))

	tests := []string{"photo.JPG", "photo.Jpg", "photo.jPg"}
	for _, key := range tests {
		got := s.Resolve(key)
		want := "images/" + key
		if got != want {
			t.Errorf("Resolve(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestSchema_ExtWithoutDot(t *testing.T) {
	s := NewSchema(RouteByExt("images/", "jpg", "png"))

	if got := s.Resolve("a.jpg"); got != "images/a.jpg" {
		t.Fatalf("Resolve without dot prefix = %q", got)
	}
}

func TestSchema_DefaultRoute(t *testing.T) {
	s := NewSchema(
		RouteByExt("images/", ".jpg"),
		DefaultRoute("misc/"),
	)

	if got := s.Resolve("cat.jpg"); got != "images/cat.jpg" {
		t.Fatalf("matched route = %q, want images/cat.jpg", got)
	}
	if got := s.Resolve("readme.txt"); got != "misc/readme.txt" {
		t.Fatalf("default route = %q, want misc/readme.txt", got)
	}
}

func TestSchema_NoDefault_Passthrough(t *testing.T) {
	s := NewSchema(RouteByExt("images/", ".jpg"))

	if got := s.Resolve("data.csv"); got != "data.csv" {
		t.Fatalf("no default should passthrough, got %q", got)
	}
}

func TestSchema_RouteByFunc(t *testing.T) {
	s := NewSchema(
		RouteByFunc("archives/", func(key string) bool {
			return strings.HasSuffix(key, ".tar.gz") || strings.HasSuffix(key, ".zip")
		}),
		DefaultRoute("other/"),
	)

	if got := s.Resolve("backup.tar.gz"); got != "archives/backup.tar.gz" {
		t.Fatalf("func route = %q", got)
	}
	if got := s.Resolve("pkg.zip"); got != "archives/pkg.zip" {
		t.Fatalf("func route zip = %q", got)
	}
	if got := s.Resolve("notes.txt"); got != "other/notes.txt" {
		t.Fatalf("default = %q", got)
	}
}

func TestSchema_FirstMatchWins(t *testing.T) {
	s := NewSchema(
		RouteByExt("special/", ".jpg"),
		RouteByExt("images/", ".jpg", ".png"),
	)

	if got := s.Resolve("x.jpg"); got != "special/x.jpg" {
		t.Fatalf("first match should win, got %q", got)
	}
}

func TestSchema_IntegrationLocalFS(t *testing.T) {
	schema := NewSchema(
		RouteByExt("images/", ".jpg", ".png"),
		RouteByExt("docs/", ".pdf"),
		DefaultRoute("other/"),
	)

	dir := t.TempDir()
	fs := mustNewLocalFS(t, dir, WithLocalKeyFunc(schema.Resolve))
	ctx := context.Background()

	_ = fs.Put(ctx, "cat.jpg", strings.NewReader("img"))
	_ = fs.Put(ctx, "report.pdf", strings.NewReader("pdf"))
	_ = fs.Put(ctx, "data.csv", strings.NewReader("csv"))

	rc, err := fs.Get(ctx, "cat.jpg")
	if err != nil {
		t.Fatalf("Get cat.jpg: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "img" {
		t.Fatalf("content = %q", data)
	}

	info, _ := fs.Access(ctx, "cat.jpg")
	if !strings.Contains(info.Path, "images/cat.jpg") {
		t.Fatalf("Access path should contain images/cat.jpg, got %q", info.Path)
	}

	info, _ = fs.Access(ctx, "report.pdf")
	if !strings.Contains(info.Path, "docs/report.pdf") {
		t.Fatalf("Access path should contain docs/report.pdf, got %q", info.Path)
	}

	info, _ = fs.Access(ctx, "data.csv")
	if !strings.Contains(info.Path, "other/data.csv") {
		t.Fatalf("Access path should contain other/data.csv, got %q", info.Path)
	}
}

func TestSchema_IntegrationObjectFS(t *testing.T) {
	schema := NewSchema(
		RouteByExt("img/", ".png"),
		DefaultRoute("files/"),
	)

	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket",
		WithPrefix("uploads/"),
		WithObjectKeyFunc(schema.Resolve),
	)
	ctx := context.Background()

	_ = fs.Put(ctx, "logo.png", strings.NewReader("png"))
	_ = fs.Put(ctx, "readme.md", strings.NewReader("md"))

	if _, ok := fake.objects["uploads/img/logo.png"]; !ok {
		t.Fatal("expected uploads/img/logo.png in fake store")
	}
	if _, ok := fake.objects["uploads/files/readme.md"]; !ok {
		t.Fatal("expected uploads/files/readme.md in fake store")
	}

	rc, err := fs.Get(ctx, "logo.png")
	if err != nil {
		t.Fatalf("Get logo.png: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "png" {
		t.Fatalf("content = %q", data)
	}
}
