package virefs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalOption configures a LocalFS instance.
type LocalOption func(*LocalFS)

// WithDirPerm sets the permission bits used when auto-creating directories.
// Default is 0755.
func WithDirPerm(perm os.FileMode) LocalOption {
	return func(l *LocalFS) { l.dirPerm = perm }
}

// WithCreateRoot creates the root directory if it does not already exist.
func WithCreateRoot() LocalOption {
	return func(l *LocalFS) { l.createRoot = true }
}

// WithLocalKeyFunc sets a KeyFunc that transforms every key after CleanKey
// and before the key is mapped to a local path.
func WithLocalKeyFunc(fn KeyFunc) LocalOption {
	return func(l *LocalFS) { l.keyFunc = fn }
}

// WithAtomicWrite enables atomic writes: Put writes to a temporary file in
// the same directory and then renames it to the target path. This prevents
// data corruption from concurrent writes to the same key.
func WithAtomicWrite() LocalOption {
	return func(l *LocalFS) { l.atomicWrite = true }
}

// LocalFS implements FS backed by a local directory.
type LocalFS struct {
	root        string
	dirPerm     os.FileMode
	createRoot  bool
	keyFunc     KeyFunc
	atomicWrite bool
}

// NewLocalFS creates a LocalFS rooted at the given directory.
func NewLocalFS(root string, opts ...LocalOption) *LocalFS {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	l := &LocalFS{
		root:    abs,
		dirPerm: 0o755,
	}
	for _, o := range opts {
		o(l)
	}
	if l.createRoot {
		_ = os.MkdirAll(l.root, l.dirPerm)
	}
	return l
}

// fullPath resolves a cleaned key to an absolute local path and ensures it
// stays within root (preventing symlink escapes).
func (l *LocalFS) fullPath(key string) (string, error) {
	cleaned, err := CleanKey(key)
	if err != nil {
		return "", err
	}
	if l.keyFunc != nil {
		cleaned = l.keyFunc(cleaned)
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

func (l *LocalFS) Put(_ context.Context, key string, r io.Reader, _ ...PutOption) error {
	p, err := l.fullPath(key)
	if err != nil {
		return &OpError{Op: "Put", Key: key, Err: err}
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, l.dirPerm); err != nil {
		return &OpError{Op: "Put", Key: key, Err: err}
	}

	if l.atomicWrite {
		return l.putAtomic(p, dir, key, r)
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

func (l *LocalFS) putAtomic(target, dir, key string, r io.Reader) error {
	tmp, err := os.CreateTemp(dir, ".virefs-tmp-*")
	if err != nil {
		return &OpError{Op: "Put", Key: key, Err: err}
	}
	tmpName := tmp.Name()

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return &OpError{Op: "Put", Key: key, Err: err}
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return &OpError{Op: "Put", Key: key, Err: err}
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
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

func (l *LocalFS) Access(_ context.Context, key string) (*AccessInfo, error) {
	p, err := l.fullPath(key)
	if err != nil {
		return nil, &OpError{Op: "Access", Key: key, Err: err}
	}
	return &AccessInfo{Path: p}, nil
}

// Copy implements Copier for same-backend file copy.
func (l *LocalFS) Copy(_ context.Context, srcKey, dstKey string) error {
	srcPath, err := l.fullPath(srcKey)
	if err != nil {
		return &OpError{Op: "Copy", Key: srcKey, Err: err}
	}
	dstPath, err := l.fullPath(dstKey)
	if err != nil {
		return &OpError{Op: "Copy", Key: dstKey, Err: err}
	}
	sf, err := os.Open(srcPath)
	if err != nil {
		return &OpError{Op: "Copy", Key: srcKey, Err: mapOSError(err)}
	}
	defer sf.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), l.dirPerm); err != nil {
		return &OpError{Op: "Copy", Key: dstKey, Err: err}
	}
	df, err := os.Create(dstPath)
	if err != nil {
		return &OpError{Op: "Copy", Key: dstKey, Err: err}
	}
	defer df.Close()
	if _, err := io.Copy(df, sf); err != nil {
		return &OpError{Op: "Copy", Key: dstKey, Err: err}
	}
	return nil
}

// mapOSError converts common os errors to virefs sentinel errors.
func mapOSError(err error) error {
	if os.IsNotExist(err) {
		return ErrNotFound
	}
	return err
}
