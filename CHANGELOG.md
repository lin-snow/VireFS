# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [v0.1.4] - 2026-04-02

### Fixed

- Improve MinIO upload compatibility by setting
  `RequestChecksumCalculationWhenRequired` in `NewS3Client` when
  `ProviderMinIO` is used, avoiding `chunk too big` failures for larger
  uploads.

### Added

- Add a regression test ensuring MinIO clients created via `NewS3Client`
  default to `RequestChecksumCalculationWhenRequired`.

## [v0.1.3] - 2026-03-07

### Changed

- Bump minimum Go version to 1.26.0.

## [v0.1.2] - 2026-03-07

### Breaking Changes

- `FS` interface now includes `Exists(ctx, key) (bool, error)` method. All
  implementations must add this method. The package-level `Exists()` function
  is retained as a backward-compatible alias.

### Added

- **LocalFS AccessFunc**: `WithLocalAccessFunc(fn)` option allows LocalFS to
  return both a disk `Path` and an HTTP `URL` from `Access()`.
- **AccessInfo relaxation**: `Path` and `URL` may now both be non-empty
  simultaneously.
- **S3 client constructor**: `S3Config`, `Provider` (AWS/MinIO/R2),
  `NewS3Client()`, and `NewObjectFSFromConfig()` simplify S3 client creation
  with provider-aware defaults (path style, region).
- **Migrate tool**: `Migrate()` recursively copies files between any two FS
  backends with conflict policies (`ConflictError`, `ConflictSkip`,
  `ConflictOverwrite`), dry-run mode, progress callbacks, and key
  transformation.
- **Middleware chain**: `Middleware` type, `Chain()` function, and `BaseFS`
  embedding helper for composing multiple FS layers without manual nesting.

### Fixed

- `ObjectFS.Access` no longer returns `(nil, nil)` when `AccessFunc` returns
  nil; it falls back to presign/baseURL strategies.
- `NewS3Client` and `NewObjectFSFromConfig` now return a clear error when
  `cfg` is nil instead of panicking.
- `ObjectFS.List` and `ObjectFS.BatchDelete` now check `ctx.Err()` in
  pagination/batching loops for proper cancellation support.
- `MountTable` methods now check context cancellation at entry.
- Added missing `var _ FS = (*ObjectFS)(nil)` compile-time interface check.

## [v0.1.1] - 2026-03-06

### Changed

- Dependency updates (GitHub Actions).

## [v0.1.0] - 2026-03-05

### Added

- Initial release.
- `FS` interface with Get, Put, Delete, List, Stat, Access.
- `LocalFS` backend (local directory).
- `ObjectFS` backend (S3-compatible object store).
- `Copier`, `Presigner`, `BatchDeleter` optional interfaces.
- `MountTable` multi-backend routing.
- `Schema` declarative key routing.
- `WithHooks` operation interceptors.
- `Walk`, `Copy`, `BatchDelete`, `Exists` helper functions.
- `plugin/zip` read-only zip archive FS.

[v0.1.4]: https://github.com/lin-snow/VireFS/compare/v0.1.3...v0.1.4
[v0.1.3]: https://github.com/lin-snow/VireFS/compare/v0.1.2...v0.1.3
[v0.1.2]: https://github.com/lin-snow/VireFS/compare/v0.1.1...v0.1.2
[v0.1.1]: https://github.com/lin-snow/VireFS/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/lin-snow/VireFS/releases/tag/v0.1.0
