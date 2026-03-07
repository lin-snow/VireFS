// Package virefs provides a unified file system abstraction over local
// directories and S3-compatible object stores.
//
// All operations are key-based: a key is a forward-slash separated path
// such as "photos/2026/cat.jpg". Keys are automatically normalised by
// [CleanKey] — leading/trailing slashes are trimmed, duplicate slashes
// collapsed, and ".." traversals rejected.
//
// # Core interface
//
// [FS] is the minimal interface every storage backend implements.
// It provides Get, Put, Delete, List, Stat, and Access operations.
//
// Two built-in backends are included:
//   - [LocalFS] — backed by a local directory on disk.
//   - [ObjectFS] — backed by any S3-compatible object store (AWS S3,
//     MinIO, Cloudflare R2, etc.).
//
// # Optional capabilities
//
// Some backends support additional operations exposed through optional
// interfaces. Use type assertions to check:
//   - [Copier] — efficient same-backend copy (LocalFS, ObjectFS).
//   - [Presigner] — presigned upload/download URLs (ObjectFS).
//   - [BatchDeleter] — bulk deletion (ObjectFS via S3 DeleteObjects).
//
// # Composition
//
// [MountTable] routes operations to different backends by key prefix,
// allowing a single FS handle to span multiple storage backends.
//
// [Schema] provides declarative key routing by file extension or custom
// match functions, and plugs into any backend via [KeyFunc].
//
// # Hooks
//
// [WithHooks] wraps any FS with optional interceptors ([Hooks]) for
// Get, Put, Stat and Delete — no need to implement all six FS methods
// just to add behaviour to one. The returned hookFS deliberately does
// not forward optional interfaces (Copier, Presigner, BatchDeleter)
// so that all data operations pass through the hooks.
//
// # Helpers
//
// Package-level functions [Copy], [BatchDelete], [Exists], and [Walk]
// work with any FS implementation.
package virefs
