package virefs_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	virefs "github.com/lin-snow/VireFS"
)

func ExampleNewLocalFS() {
	dir, _ := os.MkdirTemp("", "virefs-example-*")
	defer os.RemoveAll(dir)

	fs, err := virefs.NewLocalFS(dir)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	if err := fs.Put(ctx, "hello.txt", strings.NewReader("world")); err != nil {
		log.Fatal(err)
	}
	rc, err := fs.Get(ctx, "hello.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	fmt.Println(string(data))
	// Output: world
}

func ExampleNewLocalFS_withKeyFunc() {
	dir, _ := os.MkdirTemp("", "virefs-example-*")
	defer os.RemoveAll(dir)

	fs, err := virefs.NewLocalFS(dir, virefs.WithLocalKeyFunc(func(key string) string {
		return "v2/" + key
	}))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	_ = fs.Put(ctx, "note.txt", strings.NewReader("hello"))

	info, err := fs.Access(ctx, "note.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(strings.Contains(info.Path, "v2/note.txt"))
	// Output: true
}

func ExampleCopy() {
	dir, _ := os.MkdirTemp("", "virefs-example-*")
	defer os.RemoveAll(dir)

	fs, err := virefs.NewLocalFS(dir)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	_ = fs.Put(ctx, "src.txt", strings.NewReader("data"))

	if err := virefs.Copy(ctx, fs, "src.txt", fs, "dst.txt"); err != nil {
		log.Fatal(err)
	}

	rc, _ := fs.Get(ctx, "dst.txt")
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	fmt.Println(string(data))
	// Output: data
}

func ExampleWalk() {
	dir, _ := os.MkdirTemp("", "virefs-example-*")
	defer os.RemoveAll(dir)

	fs, err := virefs.NewLocalFS(dir)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	_ = fs.Put(ctx, "a.txt", strings.NewReader("a"))
	_ = fs.Put(ctx, "sub/b.txt", strings.NewReader("b"))

	var count int
	_ = virefs.Walk(ctx, fs, "", func(key string, info virefs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir {
			count++
		}
		return nil
	})
	fmt.Println(count)
	// Output: 2
}

func ExampleNewSchema() {
	schema := virefs.NewSchema(
		virefs.RouteByExt("images/", ".jpg", ".png"),
		virefs.RouteByExt("docs/", ".pdf"),
		virefs.DefaultRoute("other/"),
	)

	fmt.Println(schema.Resolve("cat.jpg"))
	fmt.Println(schema.Resolve("report.pdf"))
	fmt.Println(schema.Resolve("readme.txt"))
	// Output:
	// images/cat.jpg
	// docs/report.pdf
	// other/readme.txt
}

func ExampleNewMountTable() {
	dir1, _ := os.MkdirTemp("", "virefs-example-*")
	defer os.RemoveAll(dir1)
	dir2, _ := os.MkdirTemp("", "virefs-example-*")
	defer os.RemoveAll(dir2)

	fs1, _ := virefs.NewLocalFS(dir1)
	fs2, _ := virefs.NewLocalFS(dir2)

	mt := virefs.NewMountTable()
	_ = mt.Mount("data", fs1)
	_ = mt.Mount("cache", fs2)

	ctx := context.Background()
	_ = mt.Put(ctx, "data/file.txt", strings.NewReader("hello"))

	rc, _ := mt.Get(ctx, "data/file.txt")
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	fmt.Println(string(data))
	// Output: hello
}
