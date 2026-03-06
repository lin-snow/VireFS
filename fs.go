package virefs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Sentinel errors shared across all FS implementations.
var (
	ErrNotFound       = errors.New("virefs: not found")
	ErrInvalidKey     = errors.New("virefs: invalid key")
	ErrAlreadyExist   = errors.New("virefs: already exists")
	ErrNotSupported   = errors.New("virefs: operation not supported")
)

// FileInfo describes a single object / file stored in a FS.
type FileInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	IsDir        bool
}

// ListResult is returned by FS.List.
type ListResult struct {
	Files []FileInfo
}

// FS is the minimal interface every storage backend must implement.
// All keys use forward-slash separated paths with no leading slash.
type FS interface {
	// Get returns a ReadCloser for the content addressed by key.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Put writes content from r under the given key.
	// If the key already exists its content is overwritten.
	Put(ctx context.Context, key string, r io.Reader) error

	// Delete removes the object addressed by key.
	// Returns ErrNotFound if the key does not exist.
	Delete(ctx context.Context, key string) error

	// List returns objects whose keys start with prefix.
	// Pass an empty prefix to list everything under the root.
	List(ctx context.Context, prefix string) (*ListResult, error)

	// Stat returns metadata for a single key.
	// Returns ErrNotFound if the key does not exist.
	Stat(ctx context.Context, key string) (*FileInfo, error)

	// Access returns backend-specific access information for the given key.
	// LocalFS returns AccessInfo.Path (absolute file path).
	// ObjectFS returns AccessInfo.URL (presigned or public URL).
	Access(ctx context.Context, key string) (*AccessInfo, error)
}

// AccessInfo describes how to access a file from outside the FS abstraction.
// Exactly one of Path or URL will be non-empty.
type AccessInfo struct {
	// Path is the absolute local file path (set by LocalFS).
	Path string
	// URL is a directly accessible URL (set by ObjectFS — presigned or public).
	URL string
}

// PresignedRequest holds a presigned HTTP request returned by Presigner.
// It deliberately avoids exposing AWS SDK types so callers don't need a
// direct dependency on aws-sdk-go-v2.
type PresignedRequest struct {
	URL    string
	Method string
	Header http.Header
}

// Presigner is an optional interface that FS implementations may support.
// Use a type assertion to check: if p, ok := fs.(Presigner); ok { ... }
type Presigner interface {
	// PresignGet returns a presigned URL for downloading the given key.
	PresignGet(ctx context.Context, key string, expires time.Duration) (*PresignedRequest, error)

	// PresignPut returns a presigned URL for uploading to the given key.
	PresignPut(ctx context.Context, key string, expires time.Duration) (*PresignedRequest, error)
}

// KeyFunc transforms a cleaned key before it reaches the storage backend.
// It is called after CleanKey, so the input is already normalised (no "..",
// no leading/trailing slashes). The returned string is used as-is.
type KeyFunc func(key string) string

// AccessFunc builds an AccessInfo for a fully resolved storage key
// (after CleanKey + KeyFunc + basePrefix). Use it to implement custom URL
// schemes such as CDN domains or per-file-type routing.
type AccessFunc func(key string) *AccessInfo

// OpError wraps a backend error with operation context.
type OpError struct {
	Op  string // e.g. "Get", "Put"
	Key string
	Err error
}

func (e *OpError) Error() string {
	return fmt.Sprintf("virefs %s %q: %v", e.Op, e.Key, e.Err)
}

func (e *OpError) Unwrap() error { return e.Err }
