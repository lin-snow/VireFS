package virefs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalFS implements FS backed by a local directory.
type LocalFS struct {
	root string // absolute path to the mount root
}

// NewLocalFS creates a LocalFS rooted at the given directory.
// The directory must already exist.
func NewLocalFS(root string) *LocalFS {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	return &LocalFS{root: abs}
}

// fullPath resolves a cleaned key to an absolute local path and ensures it
// stays within root (preventing symlink escapes).
func (l *LocalFS) fullPath(key string) (string, error) {
	cleaned, err := CleanKey(key)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(l.root, filepath.FromSlash(cleaned))
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	if !strings.HasPrefix(abs, l.root) {
		return "", fmt.Errorf("%w: resolved path escapes root", ErrInvalidKey)
	}
	return abs, nil
}

func (l *LocalFS) Get(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := l.fullPath(key)
	if err != nil {
		return nil, &OpError{Op: "Get", Key: key, Err: err}
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, &OpError{Op: "Get", Key: key, Err: mapOSError(err)}
	}
	return f, nil
}

func (l *LocalFS) Put(_ context.Context, key string, r io.Reader) error {
	p, err := l.fullPath(key)
	if err != nil {
		return &OpError{Op: "Put", Key: key, Err: err}
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return &OpError{Op: "Put", Key: key, Err: err}
	}
	f, err := os.Create(p)
	if err != nil {
		return &OpError{Op: "Put", Key: key, Err: err}
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return &OpError{Op: "Put", Key: key, Err: err}
	}
	return nil
}

func (l *LocalFS) Delete(_ context.Context, key string) error {
	p, err := l.fullPath(key)
	if err != nil {
		return &OpError{Op: "Delete", Key: key, Err: err}
	}
	if err := os.Remove(p); err != nil {
		return &OpError{Op: "Delete", Key: key, Err: mapOSError(err)}
	}
	return nil
}

func (l *LocalFS) List(_ context.Context, prefix string) (*ListResult, error) {
	cleanedPrefix, err := CleanKey(prefix)
	if err != nil {
		return nil, &OpError{Op: "List", Key: prefix, Err: err}
	}

	dir := l.root
	if cleanedPrefix != "" {
		dir = filepath.Join(l.root, filepath.FromSlash(cleanedPrefix))
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, &OpError{Op: "List", Key: prefix, Err: mapOSError(err)}
	}

	result := &ListResult{}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		key := e.Name()
		if cleanedPrefix != "" {
			key = cleanedPrefix + "/" + e.Name()
		}
		result.Files = append(result.Files, FileInfo{
			Key:          key,
			Size:         info.Size(),
			LastModified: info.ModTime(),
			IsDir:        e.IsDir(),
		})
	}
	return result, nil
}

func (l *LocalFS) Stat(_ context.Context, key string) (*FileInfo, error) {
	p, err := l.fullPath(key)
	if err != nil {
		return nil, &OpError{Op: "Stat", Key: key, Err: err}
	}
	info, err := os.Stat(p)
	if err != nil {
		return nil, &OpError{Op: "Stat", Key: key, Err: mapOSError(err)}
	}
	return &FileInfo{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
		IsDir:        info.IsDir(),
	}, nil
}

// mapOSError converts common os errors to virefs sentinel errors.
func mapOSError(err error) error {
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}
