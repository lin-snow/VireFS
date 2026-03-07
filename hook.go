package virefs

import (
	"context"
	"io"
)

// Hooks defines optional interceptors for FS operations.
// A nil field means the corresponding operation is not intercepted.
//
// Use [WithHooks] to apply hooks to any FS implementation.
type Hooks struct {
	// WrapGet wraps the ReadCloser returned by a successful Get.
	// Use this for on-the-fly hashing, decryption, decompression, etc.
	WrapGet func(key string, rc io.ReadCloser) io.ReadCloser

	// WrapPut wraps the Reader before it reaches Put.
	// Use this for encryption, compression, byte counting, etc.
	WrapPut func(key string, r io.Reader) io.Reader

	// AfterStat is called after a successful Stat. The hook may mutate
	// the FileInfo in place (e.g. to enrich metadata).
	AfterStat func(key string, info *FileInfo)

	// OnDelete is called after a successful Delete.
	// Use this for audit logging, cache invalidation, etc.
	OnDelete func(key string)
}

// WithHooks returns a new FS that applies hooks to every operation on inner.
// Operations whose hook field is nil are forwarded to inner unchanged.
//
// The returned FS intentionally does NOT implement optional interfaces
// (Copier, Presigner, BatchDeleter) so that all data operations pass
// through the hooks. Call [hookFS.Unwrap] to access the inner FS directly.
func WithHooks(inner FS, hooks Hooks) *hookFS {
	return &hookFS{inner: inner, hooks: hooks}
}

// hookFS wraps an FS and applies user-defined hooks.
type hookFS struct {
	inner FS
	hooks Hooks
}

// Unwrap returns the underlying FS, bypassing all hooks.
func (h *hookFS) Unwrap() FS { return h.inner }

// Get implements FS. Applies WrapGet hook if set.
func (h *hookFS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	rc, err := h.inner.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if h.hooks.WrapGet != nil {
		rc = h.hooks.WrapGet(key, rc)
	}
	return rc, nil
}

// Put implements FS. Applies WrapPut hook if set.
func (h *hookFS) Put(ctx context.Context, key string, r io.Reader, opts ...PutOption) error {
	if h.hooks.WrapPut != nil {
		r = h.hooks.WrapPut(key, r)
	}
	return h.inner.Put(ctx, key, r, opts...)
}

// Delete implements FS. Calls OnDelete hook after successful deletion.
func (h *hookFS) Delete(ctx context.Context, key string) error {
	err := h.inner.Delete(ctx, key)
	if err != nil {
		return err
	}
	if h.hooks.OnDelete != nil {
		h.hooks.OnDelete(key)
	}
	return nil
}

// List implements FS.
func (h *hookFS) List(ctx context.Context, prefix string) (*ListResult, error) {
	return h.inner.List(ctx, prefix)
}

// Stat implements FS. Applies AfterStat hook if set.
func (h *hookFS) Stat(ctx context.Context, key string) (*FileInfo, error) {
	info, err := h.inner.Stat(ctx, key)
	if err != nil {
		return nil, err
	}
	if h.hooks.AfterStat != nil {
		h.hooks.AfterStat(key, info)
	}
	return info, nil
}

// Exists implements FS.
func (h *hookFS) Exists(ctx context.Context, key string) (bool, error) {
	return h.inner.Exists(ctx, key)
}

// Access implements FS.
func (h *hookFS) Access(ctx context.Context, key string) (*AccessInfo, error) {
	return h.inner.Access(ctx, key)
}

// Compile-time interface check.
var _ FS = (*hookFS)(nil)

// ---------------------------------------------------------------------------
// Middleware chain
// ---------------------------------------------------------------------------

// Middleware wraps an FS and returns a new FS with additional behaviour.
// Use [Chain] to compose multiple middlewares.
type Middleware func(FS) FS

// Chain applies middlewares to fs in declaration order. Each middleware
// wraps the result of the previous one, so the first middleware is closest
// to the base FS (innermost) and the last middleware is outermost
// (its methods are called first by the caller).
//
//	fs := virefs.Chain(baseFS,
//	    innerMiddleware,  // applied first, closest to baseFS
//	    outerMiddleware,  // applied last, outermost layer
//	)
func Chain(fs FS, mw ...Middleware) FS {
	for _, m := range mw {
		fs = m(fs)
	}
	return fs
}

// ---------------------------------------------------------------------------
// BaseFS — embedding helper for writing middlewares
// ---------------------------------------------------------------------------

// BaseFS forwards every FS method to Inner. Embed it in a custom struct
// and override only the methods you want to intercept:
//
//	type myFS struct {
//	    virefs.BaseFS
//	}
//
//	func (m *myFS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
//	    log.Println("get", key)
//	    return m.Inner.Get(ctx, key)
//	}
type BaseFS struct{ Inner FS }

// Get implements FS by forwarding to Inner.
func (b BaseFS) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return b.Inner.Get(ctx, key)
}

// Put implements FS by forwarding to Inner.
func (b BaseFS) Put(ctx context.Context, key string, r io.Reader, opts ...PutOption) error {
	return b.Inner.Put(ctx, key, r, opts...)
}

// Delete implements FS by forwarding to Inner.
func (b BaseFS) Delete(ctx context.Context, key string) error {
	return b.Inner.Delete(ctx, key)
}

// List implements FS by forwarding to Inner.
func (b BaseFS) List(ctx context.Context, prefix string) (*ListResult, error) {
	return b.Inner.List(ctx, prefix)
}

// Stat implements FS by forwarding to Inner.
func (b BaseFS) Stat(ctx context.Context, key string) (*FileInfo, error) {
	return b.Inner.Stat(ctx, key)
}

// Access implements FS by forwarding to Inner.
func (b BaseFS) Access(ctx context.Context, key string) (*AccessInfo, error) {
	return b.Inner.Access(ctx, key)
}

// Exists implements FS by forwarding to Inner.
func (b BaseFS) Exists(ctx context.Context, key string) (bool, error) {
	return b.Inner.Exists(ctx, key)
}

var _ FS = BaseFS{}
