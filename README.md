# VireFS

**VireFS** is a lightweight filesystem abstraction layer for Go.

It provides a unified interface to access different storage backends such as **local filesystems and object storage (e.g. S3)** through a single, consistent API.

The goal of VireFS is to make file operations **backend-agnostic**, allowing applications to switch or combine storage systems without changing business logic.

---

## Features

* Unified filesystem abstraction
* Multiple storage backends
* Simple and idiomatic Go API
* Easy backend switching (local ↔ object storage)
* Designed for cloud-native applications
* Extensible driver architecture

## Core Interface

```go
type FS interface {
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Put(ctx context.Context, key string, r io.Reader) error
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string) (*ListResult, error)
    Stat(ctx context.Context, key string) (*FileInfo, error)
}
```

### Key Convention

* Keys use `/` as separator — e.g. `docs/readme.txt`.
* No leading or trailing slashes needed; they are trimmed automatically.
* `..` traversal is rejected as invalid.

### Error Model

| Sentinel | Meaning |
|---|---|
| `ErrNotFound` | Key does not exist |
| `ErrInvalidKey` | Key contains illegal patterns (e.g. `..`) |
| `ErrAlreadyExist` | Resource already exists (reserved) |

All backend errors are wrapped in `*OpError{Op, Key, Err}` for easy debugging.

## Mount & Routing

```go
mt := virefs.NewMountTable()
mt.Mount("local", virefs.NewLocalFS("/data/files"))
mt.Mount("s3",    virefs.NewObjectFS(s3Client, "my-bucket", virefs.WithPrefix("prefix/")))

// Routed automatically by prefix:
mt.Get(ctx, "local/reports/q1.csv")   // → LocalFS("/data/files").Get("reports/q1.csv")
mt.Get(ctx, "s3/images/logo.png")     // → ObjectFS(bucket).Get("prefix/images/logo.png")
```

## Quick Start

### Local filesystem only

```go
fs := virefs.NewLocalFS("/tmp/mydata", virefs.WithCreateRoot())
_ = fs.Put(ctx, "hello.txt", strings.NewReader("world"))

rc, _ := fs.Get(ctx, "hello.txt")
defer rc.Close()
data, _ := io.ReadAll(rc)
fmt.Println(string(data)) // "world"
```

### Object storage (S3-compatible)

```go
cfg, _ := config.LoadDefaultConfig(ctx)
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.BaseEndpoint = aws.String("https://s3.example.com")
    o.UsePathStyle = true
})

fs := virefs.NewObjectFS(client, "my-bucket", virefs.WithPrefix("app/"))
_ = fs.Put(ctx, "data.json", strings.NewReader(`{"ok":true}`))
// writes to S3 key: "app/data.json"
```

### Key transformation (KeyFunc)

Use `WithLocalKeyFunc` / `WithObjectKeyFunc` to transform keys before they hit storage. The function receives a cleaned key (no `..`, no leading slashes) and returns the final key.

```go
fs := virefs.NewLocalFS("/data", virefs.WithLocalKeyFunc(func(key string) string {
    return time.Now().Format("2006/01/02") + "/" + key
}))
// Put("photo.jpg", ...) actually writes to /data/2026/03/06/photo.jpg

objFS := virefs.NewObjectFS(client, "bucket", virefs.WithObjectKeyFunc(func(key string) string {
    return "v2/" + key
}))
// Get("config.yaml") fetches S3 key "v2/config.yaml"
```

## License

See [LICENSE](LICENSE).
