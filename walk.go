package virefs

import (
	"context"
	"errors"
	"path"
)

// ErrSkipDir is used as a return value from WalkFunc to indicate that the
// directory named in the call is to be skipped. It is not returned as
// an error by Walk itself.
var ErrSkipDir = errors.New("skip this directory")

// WalkFunc is the type of the function called by Walk to visit each file
// or directory. The key argument is the full key of the entry. If there
// was an error listing a directory, err will be non-nil and info may be
// zero-valued.
//
// If WalkFunc returns ErrSkipDir when invoked on a directory, Walk skips
// that directory's contents. If WalkFunc returns any other non-nil error,
// Walk stops entirely and returns that error.
type WalkFunc func(key string, info FileInfo, err error) error

// Walk recursively traverses the file tree rooted at prefix, calling fn
// for each file or directory (including prefix itself if it is a directory).
// It uses List internally and recurses into sub-directories (IsDir == true).
func Walk(ctx context.Context, fsys FS, prefix string, fn WalkFunc) error {
	result, err := fsys.List(ctx, prefix)
	if err != nil {
		return fn(prefix, FileInfo{}, err)
	}
	for _, fi := range result.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if fi.IsDir {
			if err := fn(fi.Key, fi, nil); err != nil {
				if errors.Is(err, ErrSkipDir) {
					continue
				}
				return err
			}
			subPrefix := fi.Key
			if prefix != "" && !hasPathPrefix(fi.Key, prefix) {
				subPrefix = path.Join(prefix, fi.Key)
			}
			if err := Walk(ctx, fsys, subPrefix, fn); err != nil {
				return err
			}
		} else {
			if err := fn(fi.Key, fi, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func hasPathPrefix(key, prefix string) bool {
	if prefix == "" {
		return true
	}
	return len(key) > len(prefix) && key[:len(prefix)] == prefix && key[len(prefix)] == '/'
}
