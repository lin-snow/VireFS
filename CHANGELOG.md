# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `Walk` helper function for recursive traversal of any FS.
- `SkipDir` sentinel for skipping directories during Walk.
- `BatchDeleter` optional interface and `BatchDelete` helper function.
- `ObjectFS.BatchDelete` using S3 `DeleteObjects` for efficient bulk deletion.
- `MountTable` now implements `Copier` — same-mount copies use native backend copy.
- `ErrPermission` sentinel error for permission-denied cases.
- `doc.go` — package-level documentation for pkg.go.dev.
- Godoc testable examples (`ExampleNewLocalFS`, `ExampleCopy`, `ExampleWalk`, etc.).
- Benchmark tests for `CleanKey`, `Schema.Resolve`, and `LocalFS` operations.
- Test coverage for `MountTable.Unmount`, `MountTable.Delete`, `MountTable.Stat`,
  `MountTable.ConcurrentAccess`, `LocalFS.WithCreateRoot`, `LocalFS.WithDirPerm`,
  `LocalFS.DeleteNotFound`, `ObjectFS.ListPagination`, `ObjectFS.ListShallow`.

### Changed
- `NewLocalFS` now returns `(*LocalFS, error)` instead of `*LocalFS` — **breaking change**.
- `FS.Delete` contract clarified: behaviour on missing key is backend-specific.
- `FS.List` now returns only immediate children (shallow listing) across all backends.
- `ObjectFS.List` uses S3 `Delimiter` for shallow listing with `CommonPrefixes` as directories.
- `ZipFS.List` updated to shallow semantics matching the core interface.
- `LocalFS` methods now check `ctx.Err()` before performing I/O.
- `Pack` function uses named return + defer for reliable `zip.Writer` cleanup.
- `mapOSError` expanded to map `os.IsPermission` to `ErrPermission`.
- Compile-time interface checks added for `LocalFS` (`FS`, `Copier`).
- `S3API` interface now includes `DeleteObjects`.
- README examples updated with proper error handling.
