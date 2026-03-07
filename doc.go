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
// It provides Get, Put, Delete, List, Stat, Access, and Exists operations.
//
// Two built-in backends are included:
//   - [LocalFS] — backed by a local directory on disk.
//   - [ObjectFS] — backed by any S3-compatible object store (AWS S3,
//     MinIO, Cloudflare R2, etc.).
//
// # S3 client construction
//
// [S3Config] and [NewS3Client] simplify the creation of S3 clients with
// provider-aware defaults for AWS, MinIO, and Cloudflare R2. Use
// [NewObjectFSFromConfig] to create an ObjectFS (with presign support)
// in a single call.
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
// # Hooks and middleware
//
// [WithHooks] wraps any FS with optional interceptors ([Hooks]) for
// Get, Put, Stat and Delete — no need to implement all seven FS methods
// just to add behaviour to one. The returned hookFS deliberately does
// not forward optional interfaces (Copier, Presigner, BatchDeleter)
// so that all data operations pass through the hooks.
//
// For more complex scenarios (multiple layers, intercepting any method),
// use [Chain] with [Middleware] functions. Embed [BaseFS] in a custom
// struct to forward all methods and override only those you need.
//
// # Helpers
//
// Package-level functions [Copy], [BatchDelete], [Exists], [Walk], and
// [Migrate] work with any FS implementation. Migrate supports conflict
// policies ([ConflictSkip], [ConflictOverwrite], [ConflictError]),
// dry-run mode, and progress callbacks for bulk data migration.
package virefs
