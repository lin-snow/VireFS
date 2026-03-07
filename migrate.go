package virefs

import (
	"context"
	"fmt"
	"path"
	"strings"
)

// ConflictPolicy controls how Migrate behaves when a destination key
// already exists.
type ConflictPolicy int

const (
	// ConflictError causes Migrate to return an error on the first conflict.
	ConflictError ConflictPolicy = iota
	// ConflictSkip silently skips keys that already exist at the destination.
	ConflictSkip
	// ConflictOverwrite overwrites existing keys at the destination.
	ConflictOverwrite
)

// MigrateProgress is passed to the progress callback after each file is
// processed (copied or skipped).
type MigrateProgress struct {
	Key     string // source key being processed
	Copied  int    // cumulative files copied so far
	Skipped int    // cumulative files skipped so far
	Total   int    // cumulative files scanned so far (excluding directories)
}

// MigrateResult summarises a completed Migrate operation.
type MigrateResult struct {
	Copied  int
	Skipped int
	Total   int
}

// MigrateOption configures a Migrate operation.
type MigrateOption func(*migrateConfig)

type migrateConfig struct {
	conflict ConflictPolicy
	dryRun   bool
	progress func(MigrateProgress)
	keyFunc  func(srcKey string) string
}

// WithConflictPolicy sets the conflict resolution strategy.
// Default is ConflictError.
func WithConflictPolicy(p ConflictPolicy) MigrateOption {
	return func(c *migrateConfig) { c.conflict = p }
}

// WithDryRun makes Migrate walk and check conflicts without actually
// copying any data. The returned MigrateResult still reports what
// would have been copied or skipped.
func WithDryRun() MigrateOption {
	return func(c *migrateConfig) { c.dryRun = true }
}

// WithProgressFunc registers a callback invoked after each file is
// processed. The callback is called synchronously from Migrate.
func WithProgressFunc(fn func(MigrateProgress)) MigrateOption {
	return func(c *migrateConfig) { c.progress = fn }
}

// WithMigrateKeyFunc sets a function that transforms source keys into
// destination keys. The function receives the relative source key
// (with srcPrefix stripped) and returns the relative destination key.
// If nil, the source relative key is used as-is.
func WithMigrateKeyFunc(fn func(srcKey string) string) MigrateOption {
	return func(c *migrateConfig) { c.keyFunc = fn }
}

// Migrate recursively copies files from src (under srcPrefix) to dst
// (under dstPrefix). It uses Walk to enumerate source files and Copy
// to transfer each one.
//
// Migrate supports conflict policies, dry-run mode, progress callbacks,
// and key transformation. See the With* options for details.
func Migrate(ctx context.Context, src FS, srcPrefix string, dst FS, dstPrefix string, opts ...MigrateOption) (*MigrateResult, error) {
	cfg := &migrateConfig{}
	for _, o := range opts {
		o(cfg)
	}

	result := &MigrateResult{}

	err := Walk(ctx, src, srcPrefix, func(key string, info FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir {
			return nil
		}

		result.Total++

		relKey := stripPrefix(key, srcPrefix)
		if cfg.keyFunc != nil {
			relKey = cfg.keyFunc(relKey)
		}

		var dstKey string
		if dstPrefix != "" {
			dstKey = path.Join(dstPrefix, relKey)
		} else {
			dstKey = relKey
		}

		if cfg.conflict != ConflictOverwrite {
			exists, err := dst.Exists(ctx, dstKey)
			if err != nil {
				return fmt.Errorf("migrate: check exists %q: %w", dstKey, err)
			}
			if exists {
				switch cfg.conflict {
				case ConflictError:
					return fmt.Errorf("migrate: %w: destination key %q already exists", ErrAlreadyExist, dstKey)
				case ConflictSkip:
					result.Skipped++
					if cfg.progress != nil {
						cfg.progress(MigrateProgress{
							Key: key, Copied: result.Copied,
							Skipped: result.Skipped, Total: result.Total,
						})
					}
					return nil
				}
			}
		}

		if !cfg.dryRun {
			if err := Copy(ctx, src, key, dst, dstKey); err != nil {
				return fmt.Errorf("migrate: copy %q -> %q: %w", key, dstKey, err)
			}
		}

		result.Copied++
		if cfg.progress != nil {
			cfg.progress(MigrateProgress{
				Key: key, Copied: result.Copied,
				Skipped: result.Skipped, Total: result.Total,
			})
		}
		return nil
	})

	if err != nil {
		return result, err
	}
	return result, nil
}

func stripPrefix(key, prefix string) string {
	if prefix == "" {
		return key
	}
	trimmed := strings.TrimPrefix(key, prefix)
	trimmed = strings.TrimPrefix(trimmed, "/")
	return trimmed
}
