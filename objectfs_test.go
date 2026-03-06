package virefs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// fakeS3 is an in-memory S3 implementation for testing.
type fakeS3 struct {
	objects map[string][]byte
}

func newFakeS3() *fakeS3 {
	return &fakeS3{objects: make(map[string][]byte)}
}

func (f *fakeS3) PutObject(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	data, err := io.ReadAll(in.Body)
	if err != nil {
		return nil, err
	}
	f.objects[aws.ToString(in.Key)] = data
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	key := aws.ToString(in.Key)
	data, ok := f.objects[key]
	if !ok {
		return nil, &types.NoSuchKey{Message: aws.String("no such key: " + key)}
	}
	return &s3.GetObjectOutput{
		Body:          io.NopCloser(bytes.NewReader(data)),
		ContentLength: aws.Int64(int64(len(data))),
	}, nil
}

func (f *fakeS3) DeleteObject(_ context.Context, in *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	delete(f.objects, aws.ToString(in.Key))
	return &s3.DeleteObjectOutput{}, nil
}

func (f *fakeS3) HeadObject(_ context.Context, in *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	key := aws.ToString(in.Key)
	data, ok := f.objects[key]
	if !ok {
		return nil, &types.NotFound{Message: aws.String("not found: " + key)}
	}
	now := time.Now()
	return &s3.HeadObjectOutput{
		ContentLength: aws.Int64(int64(len(data))),
		LastModified:  &now,
	}, nil
}

func (f *fakeS3) ListObjectsV2(_ context.Context, in *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	prefix := aws.ToString(in.Prefix)
	var contents []types.Object
	for k, v := range f.objects {
		if strings.HasPrefix(k, prefix) {
			now := time.Now()
			contents = append(contents, types.Object{
				Key:          aws.String(k),
				Size:         aws.Int64(int64(len(v))),
				LastModified: &now,
			})
		}
	}
	return &s3.ListObjectsV2Output{
		Contents:    contents,
		IsTruncated: aws.Bool(false),
	}, nil
}

func TestObjectFS_PutGetDeleteStat(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "test-bucket")
	ctx := context.Background()

	if err := fs.Put(ctx, "doc.txt", strings.NewReader("hello")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	rc, err := fs.Get(ctx, "doc.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "hello" {
		t.Fatalf("Get content = %q, want %q", data, "hello")
	}

	info, err := fs.Stat(ctx, "doc.txt")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size != 5 {
		t.Fatalf("Stat size = %d, want 5", info.Size)
	}

	if err := fs.Delete(ctx, "doc.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = fs.Get(ctx, "doc.txt")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete error = %v, want ErrNotFound", err)
	}
}

func TestObjectFS_BasePrefix(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket", WithPrefix("data/"))
	ctx := context.Background()

	_ = fs.Put(ctx, "a.txt", strings.NewReader("a"))

	if _, ok := fake.objects["data/a.txt"]; !ok {
		t.Fatal("expected object at data/a.txt in fake store")
	}

	rc, err := fs.Get(ctx, "a.txt")
	if err != nil {
		t.Fatalf("Get with prefix: %v", err)
	}
	d, _ := io.ReadAll(rc)
	rc.Close()
	if string(d) != "a" {
		t.Fatalf("Get content = %q, want %q", d, "a")
	}
}

func TestObjectFS_List(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket", WithPrefix("pfx/"))
	ctx := context.Background()

	_ = fs.Put(ctx, "dir/x.txt", strings.NewReader("x"))
	_ = fs.Put(ctx, "dir/y.txt", strings.NewReader("y"))
	_ = fs.Put(ctx, "other.txt", strings.NewReader("o"))

	result, err := fs.List(ctx, "dir")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Files) != 2 {
		t.Fatalf("List got %d entries, want 2", len(result.Files))
	}
}

func TestObjectFS_WithKeyFunc(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket", WithPrefix("base/"), WithObjectKeyFunc(func(key string) string {
		return "2026/03/06/" + key
	}))
	ctx := context.Background()

	_ = fs.Put(ctx, "photo.jpg", strings.NewReader("img"))

	wantKey := "base/2026/03/06/photo.jpg"
	if _, ok := fake.objects[wantKey]; !ok {
		keys := make([]string, 0, len(fake.objects))
		for k := range fake.objects {
			keys = append(keys, k)
		}
		t.Fatalf("expected object at %q, got keys %v", wantKey, keys)
	}

	rc, err := fs.Get(ctx, "photo.jpg")
	if err != nil {
		t.Fatalf("Get with KeyFunc: %v", err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "img" {
		t.Fatalf("Get content = %q, want %q", data, "img")
	}
}

func TestObjectFS_NotFound(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket")
	ctx := context.Background()

	_, err := fs.Get(ctx, "nope.txt")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing key error = %v, want ErrNotFound", err)
	}

	_, err = fs.Stat(ctx, "nope.txt")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Stat missing key error = %v, want ErrNotFound", err)
	}
}

// fakePresign implements PresignAPI for testing.
type fakePresign struct{}

func (fp *fakePresign) PresignGetObject(_ context.Context, in *s3.GetObjectInput, opts ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	var po s3.PresignOptions
	for _, fn := range opts {
		fn(&po)
	}
	return &v4.PresignedHTTPRequest{
		URL:          fmt.Sprintf("https://s3.example.com/%s/%s?expires=%s", aws.ToString(in.Bucket), aws.ToString(in.Key), po.Expires),
		Method:       "GET",
		SignedHeader: http.Header{"Host": []string{"s3.example.com"}},
	}, nil
}

func (fp *fakePresign) PresignPutObject(_ context.Context, in *s3.PutObjectInput, opts ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	var po s3.PresignOptions
	for _, fn := range opts {
		fn(&po)
	}
	return &v4.PresignedHTTPRequest{
		URL:          fmt.Sprintf("https://s3.example.com/%s/%s?expires=%s", aws.ToString(in.Bucket), aws.ToString(in.Key), po.Expires),
		Method:       "PUT",
		SignedHeader: http.Header{"Host": []string{"s3.example.com"}},
	}, nil
}

func TestObjectFS_PresignGet(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket", WithPrefix("pfx/"), WithPresignClient(&fakePresign{}))
	ctx := context.Background()

	req, err := fs.PresignGet(ctx, "report.pdf", 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	if req.Method != "GET" {
		t.Fatalf("PresignGet method = %q, want GET", req.Method)
	}
	if !strings.Contains(req.URL, "pfx/report.pdf") {
		t.Fatalf("PresignGet URL should contain prefixed key, got %q", req.URL)
	}
	if !strings.Contains(req.URL, "15m0s") {
		t.Fatalf("PresignGet URL should contain expiry, got %q", req.URL)
	}
}

func TestObjectFS_PresignPut(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket", WithPresignClient(&fakePresign{}))
	ctx := context.Background()

	req, err := fs.PresignPut(ctx, "upload.zip", 30*time.Minute)
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	if req.Method != "PUT" {
		t.Fatalf("PresignPut method = %q, want PUT", req.Method)
	}
	if !strings.Contains(req.URL, "upload.zip") {
		t.Fatalf("PresignPut URL should contain key, got %q", req.URL)
	}
}

func TestObjectFS_PresignWithKeyFunc(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket", WithPrefix("data/"),
		WithObjectKeyFunc(func(key string) string { return "v2/" + key }),
		WithPresignClient(&fakePresign{}),
	)
	ctx := context.Background()

	req, err := fs.PresignGet(ctx, "file.txt", 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet with KeyFunc: %v", err)
	}
	if !strings.Contains(req.URL, "data/v2/file.txt") {
		t.Fatalf("PresignGet URL should contain transformed key, got %q", req.URL)
	}
}

func TestObjectFS_PresignWithoutClient(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket")
	ctx := context.Background()

	_, err := fs.PresignGet(ctx, "file.txt", 5*time.Minute)
	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("PresignGet without client error = %v, want ErrNotSupported", err)
	}

	_, err = fs.PresignPut(ctx, "file.txt", 5*time.Minute)
	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("PresignPut without client error = %v, want ErrNotSupported", err)
	}
}

func TestObjectFS_AccessWithPresign(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket", WithPrefix("pfx/"), WithPresignClient(&fakePresign{}))
	ctx := context.Background()

	info, err := fs.Access(ctx, "report.pdf")
	if err != nil {
		t.Fatalf("Access with presign: %v", err)
	}
	if info.URL == "" {
		t.Fatal("Access.URL should be non-empty")
	}
	if info.Path != "" {
		t.Fatal("Access.Path should be empty for ObjectFS")
	}
	if !strings.Contains(info.URL, "pfx/report.pdf") {
		t.Fatalf("Access.URL should contain prefixed key, got %q", info.URL)
	}
}

func TestObjectFS_AccessWithBaseURL(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket", WithPrefix("data/"), WithBaseURL("https://cdn.example.com"))
	ctx := context.Background()

	info, err := fs.Access(ctx, "img/logo.png")
	if err != nil {
		t.Fatalf("Access with base URL: %v", err)
	}
	want := "https://cdn.example.com/data/img/logo.png"
	if info.URL != want {
		t.Fatalf("Access.URL = %q, want %q", info.URL, want)
	}
}

func TestObjectFS_AccessPresignPriority(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket",
		WithPresignClient(&fakePresign{}),
		WithBaseURL("https://cdn.example.com"),
	)
	ctx := context.Background()

	info, err := fs.Access(ctx, "file.txt")
	if err != nil {
		t.Fatalf("Access: %v", err)
	}
	if strings.HasPrefix(info.URL, "https://cdn.example.com") {
		t.Fatalf("presign client should take priority over base URL, got %q", info.URL)
	}
}

func TestObjectFS_AccessNotConfigured(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket")
	ctx := context.Background()

	_, err := fs.Access(ctx, "file.txt")
	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("Access without config error = %v, want ErrNotSupported", err)
	}
}

func TestObjectFS_AccessFunc_CDN(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket", WithPrefix("assets/"),
		WithAccessFunc(func(key string) *AccessInfo {
			return &AccessInfo{URL: "https://cdn.example.com/" + key}
		}),
	)
	ctx := context.Background()

	info, err := fs.Access(ctx, "img/logo.png")
	if err != nil {
		t.Fatalf("Access with AccessFunc: %v", err)
	}
	want := "https://cdn.example.com/assets/img/logo.png"
	if info.URL != want {
		t.Fatalf("Access.URL = %q, want %q", info.URL, want)
	}
}

func TestObjectFS_AccessFunc_PriorityOverPresign(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket",
		WithPresignClient(&fakePresign{}),
		WithBaseURL("https://s3.example.com"),
		WithAccessFunc(func(key string) *AccessInfo {
			return &AccessInfo{URL: "https://cdn.fast.io/" + key}
		}),
	)
	ctx := context.Background()

	info, err := fs.Access(ctx, "file.txt")
	if err != nil {
		t.Fatalf("Access: %v", err)
	}
	if !strings.HasPrefix(info.URL, "https://cdn.fast.io/") {
		t.Fatalf("AccessFunc should take priority over presign and baseURL, got %q", info.URL)
	}
}

func TestObjectFS_AccessFunc_WithKeyFunc(t *testing.T) {
	fake := newFakeS3()
	fs := NewObjectFS(fake, "bucket", WithPrefix("data/"),
		WithObjectKeyFunc(func(key string) string { return "v2/" + key }),
		WithAccessFunc(func(key string) *AccessInfo {
			return &AccessInfo{URL: "https://cdn.example.com/" + key}
		}),
	)
	ctx := context.Background()

	info, err := fs.Access(ctx, "config.yaml")
	if err != nil {
		t.Fatalf("Access: %v", err)
	}
	want := "https://cdn.example.com/data/v2/config.yaml"
	if info.URL != want {
		t.Fatalf("AccessFunc should receive full s3 key, got URL %q, want %q", info.URL, want)
	}
}
