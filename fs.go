package virefs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// Sentinel errors shared across all FS implementations.
var (
	ErrNotFound     = errors.New("virefs: not found")
	ErrInvalidKey   = errors.New("virefs: invalid key")
	ErrAlreadyExist = errors.New("virefs: already exists")
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
}

// KeyFunc transforms a cleaned key before it reaches the storage backend.
// It is called after CleanKey, so the input is already normalised (no "..",
// no leading/trailing slashes). The returned string is used as-is.
type KeyFunc func(key string) string

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
