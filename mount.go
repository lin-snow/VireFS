package virefs

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
)

// MountTable routes key-based operations to the correct FS by prefix.
//
// A key like "local/docs/a.txt" is split into mount prefix "local" and
// sub-key "docs/a.txt", then forwarded to the FS mounted at "local".
type MountTable struct {
	mu     sync.RWMutex
	mounts map[string]FS
}

// NewMountTable returns an empty MountTable.
func NewMountTable() *MountTable {
	return &MountTable{mounts: make(map[string]FS)}
}

// Mount registers fs under the given prefix.
// Prefix must be a single path segment with no slashes.
func (mt *MountTable) Mount(prefix string, fs FS) error {
	if prefix == "" || strings.Contains(prefix, "/") {
		return fmt.Errorf("%w: mount prefix must be a single non-empty segment, got %q", ErrInvalidKey, prefix)
	}
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.mounts[prefix] = fs
	return nil
}

// Unmount removes the FS registered under prefix.
func (mt *MountTable) Unmount(prefix string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	delete(mt.mounts, prefix)
}

// resolve splits a full key into the target FS and the sub-key.
func (mt *MountTable) resolve(fullKey string) (FS, string, error) {
	cleaned, err := CleanKey(fullKey)
	if err != nil {
		return nil, "", err
	}
	if cleaned == "" {
		return nil, "", fmt.Errorf("%w: empty key after cleaning", ErrInvalidKey)
	}

	prefix, subKey, _ := strings.Cut(cleaned, "/")

	mt.mu.RLock()
	fs, ok := mt.mounts[prefix]
	mt.mu.RUnlock()

	if !ok {
		return nil, "", fmt.Errorf("%w: no filesystem mounted at %q", ErrNotFound, prefix)
	}
	return fs, subKey, nil
}

// Get implements FS.
func (mt *MountTable) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fs, sub, err := mt.resolve(key)
	if err != nil {
		return nil, err
	}
	return fs.Get(ctx, sub)
}

// Put implements FS.
func (mt *MountTable) Put(ctx context.Context, key string, r io.Reader) error {
	fs, sub, err := mt.resolve(key)
	if err != nil {
		return err
	}
	return fs.Put(ctx, sub, r)
}

// Delete implements FS.
func (mt *MountTable) Delete(ctx context.Context, key string) error {
	fs, sub, err := mt.resolve(key)
	if err != nil {
		return err
	}
	return fs.Delete(ctx, sub)
}

// List implements FS.
// If prefix resolves to a mount point, the sub-prefix is forwarded.
// If prefix is empty it lists top-level mount points as virtual directories.
func (mt *MountTable) List(ctx context.Context, prefix string) (*ListResult, error) {
	if prefix == "" {
		mt.mu.RLock()
		defer mt.mu.RUnlock()
		result := &ListResult{}
		for name := range mt.mounts {
			result.Files = append(result.Files, FileInfo{Key: name, IsDir: true})
		}
		return result, nil
	}
	fs, sub, err := mt.resolve(prefix)
	if err != nil {
		return nil, err
	}
	return fs.List(ctx, sub)
}

// Stat implements FS.
func (mt *MountTable) Stat(ctx context.Context, key string) (*FileInfo, error) {
	fs, sub, err := mt.resolve(key)
	if err != nil {
		return nil, err
	}
	return fs.Stat(ctx, sub)
}
