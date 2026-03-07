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
	ErrNotFound     = errors.New("virefs: not found")
	ErrInvalidKey   = errors.New("virefs: invalid key")
	ErrAlreadyExist = errors.New("virefs: already exists")
	ErrNotSupported = errors.New("virefs: operation not supported")
	ErrPermission   = errors.New("virefs: permission denied")
)

// FileInfo describes a single object / file stored in a FS.
type FileInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	IsDir        bool
	ContentType  string // MIME type, e.g. "image/jpeg"; may be empty
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
	// Use PutOption to set ContentType, Metadata, etc.
	Put(ctx context.Context, key string, r io.Reader, opts ...PutOption) error

	// Delete removes the object addressed by key.
	// Behaviour when the key does not exist is backend-specific:
	// LocalFS returns ErrNotFound; ObjectFS silently succeeds (S3 idempotent delete).
	Delete(ctx context.Context, key string) error

	// List returns immediate children (files and sub-directories) under prefix.
	// Sub-directories are returned as FileInfo with IsDir == true.
	// Pass an empty prefix to list the root level.
	List(ctx context.Context, prefix string) (*ListResult, error)

	// Stat returns metadata for a single key.
	// Returns ErrNotFound if the key does not exist.
	Stat(ctx context.Context, key string) (*FileInfo, error)

	// Access returns backend-specific access information for the given key.
	// LocalFS returns AccessInfo.Path (absolute file path).
	// ObjectFS returns AccessInfo.URL (presigned or public URL).
	Access(ctx context.Context, key string) (*AccessInfo, error)

	// Exists reports whether a key exists in the backend.
	// It returns (false, nil) when the key is not found, rather than an error.
	// Backends may implement this more efficiently than Stat (e.g. S3 HeadObject).
	Exists(ctx context.Context, key string) (bool, error)
}

// PutOption configures a Put operation.
type PutOption func(*PutConfig)

// PutConfig holds optional parameters for Put.
type PutConfig struct {
	ContentType string
	Metadata    map[string]string
}

// BuildPutConfig applies all PutOptions and returns the resulting config.
func BuildPutConfig(opts []PutOption) PutConfig {
	var c PutConfig
	for _, o := range opts {
		o(&c)
	}
	return c
}

// WithContentType sets the MIME type of the content being uploaded.
// ObjectFS passes it to S3 PutObject; LocalFS ignores it.
func WithContentType(ct string) PutOption {
	return func(c *PutConfig) { c.ContentType = ct }
}

// WithMetadata attaches custom key-value metadata to the uploaded object.
// ObjectFS passes it as S3 user metadata; LocalFS ignores it.
func WithMetadata(m map[string]string) PutOption {
	return func(c *PutConfig) { c.Metadata = m }
}

// Exists is a convenience helper that delegates to fs.Exists.
// It is kept for backward compatibility with code that calls the
// package-level function instead of the interface method.
func Exists(ctx context.Context, fs FS, key string) (bool, error) {
	return fs.Exists(ctx, key)
}

// Copier is an optional interface for efficient same-backend copies.
// Use a type assertion to check: if c, ok := fs.(Copier); ok { ... }
type Copier interface {
	Copy(ctx context.Context, srcKey, dstKey string) error
}

// BatchDeleter is an optional interface for efficient bulk deletion.
// ObjectFS implements this using S3 DeleteObjects.
// Use a type assertion to check: if bd, ok := fs.(BatchDeleter); ok { ... }
type BatchDeleter interface {
	BatchDelete(ctx context.Context, keys []string) error
}

// BatchDelete deletes multiple keys from fsys. If fsys implements
// BatchDeleter, the native bulk operation is used. Otherwise it falls
// back to calling Delete for each key individually, returning the first
// error encountered.
func BatchDelete(ctx context.Context, fsys FS, keys []string) error {
	if bd, ok := fsys.(BatchDeleter); ok {
		return bd.BatchDelete(ctx, keys)
	}
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fsys.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// Copy copies a file from src to dst. If src and dst are the same FS instance
// and it implements Copier, the native (efficient) copy is used. Otherwise it
// falls back to Get + Put.
func Copy(ctx context.Context, src FS, srcKey string, dst FS, dstKey string, opts ...PutOption) error {
	if src == dst {
		if c, ok := src.(Copier); ok {
			return c.Copy(ctx, srcKey, dstKey)
		}
	}
	rc, err := src.Get(ctx, srcKey)
	if err != nil {
		return fmt.Errorf("copy: get %q: %w", srcKey, err)
	}
	defer rc.Close()
	return dst.Put(ctx, dstKey, rc, opts...)
}

// AccessInfo describes how to access a file from outside the FS abstraction.
// At least one of Path or URL will be non-empty; both may be set
// simultaneously (e.g. LocalFS with an AccessFunc that adds an HTTP URL).
type AccessInfo struct {
	// Path is the absolute local file path (set by LocalFS).
	Path string
	// URL is a directly accessible URL (set by ObjectFS — presigned or public,
	// or by LocalFS when an AccessFunc is configured).
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
