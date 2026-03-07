package zip

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"mime"
	"os"
	"path"
	"strings"

	virefs "github.com/lin-snow/VireFS"
)

// FS is a read-only virefs.FS backed by a zip archive.
// Put, Delete and Access always return virefs.ErrNotSupported.
type FS struct {
	r      *zip.Reader
	closer io.Closer
	index  map[string]*zip.File
}

// compile-time interface check
var _ virefs.FS = (*FS)(nil)

// OpenFS opens a zip file at path and returns a read-only FS.
// The caller must call Close when done.
func OpenFS(path string) (*FS, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	zr, err := zip.NewReader(f, info.Size())
	if err != nil {
		f.Close()
		return nil, err
	}
	return &FS{
		r:      zr,
		closer: f,
		index:  buildIndex(zr),
	}, nil
}

// NewFS creates a read-only FS from an io.ReaderAt.
// The caller is responsible for the lifetime of ra.
func NewFS(ra io.ReaderAt, size int64) (*FS, error) {
	zr, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, err
	}
	return &FS{
		r:     zr,
		index: buildIndex(zr),
	}, nil
}

// NewFSFromBytes creates a read-only FS from in-memory bytes.
func NewFSFromBytes(data []byte) (*FS, error) {
	return NewFS(bytes.NewReader(data), int64(len(data)))
}

// Close releases the underlying file handle if one was opened by OpenFS.
func (z *FS) Close() error {
	if z.closer != nil {
		return z.closer.Close()
	}
	return nil
}

// Get implements virefs.FS.
func (z *FS) Get(_ context.Context, key string) (io.ReadCloser, error) {
	cleaned, err := virefs.CleanKey(key)
	if err != nil {
		return nil, &virefs.OpError{Op: "Get", Key: key, Err: err}
	}
	f, ok := z.index[cleaned]
	if !ok {
		return nil, &virefs.OpError{Op: "Get", Key: key, Err: virefs.ErrNotFound}
	}
	rc, err := f.Open()
	if err != nil {
		return nil, &virefs.OpError{Op: "Get", Key: key, Err: err}
	}
	return rc, nil
}

// Put implements virefs.FS. Always returns ErrNotSupported.
func (z *FS) Put(_ context.Context, key string, _ io.Reader, _ ...virefs.PutOption) error {
	return &virefs.OpError{Op: "Put", Key: key, Err: virefs.ErrNotSupported}
}

// Delete implements virefs.FS. Always returns ErrNotSupported.
func (z *FS) Delete(_ context.Context, key string) error {
	return &virefs.OpError{Op: "Delete", Key: key, Err: virefs.ErrNotSupported}
}

// List implements virefs.FS.
func (z *FS) List(_ context.Context, prefix string) (*virefs.ListResult, error) {
	cleanedPrefix, err := virefs.CleanKey(prefix)
	if err != nil {
		return nil, &virefs.OpError{Op: "List", Key: prefix, Err: err}
	}

	dirSeen := make(map[string]struct{})
	result := &virefs.ListResult{}
	for k, f := range z.index {
		var rest string
		switch {
		case cleanedPrefix == "":
			rest = k
		case strings.HasPrefix(k, cleanedPrefix+"/"):
			rest = k[len(cleanedPrefix)+1:]
		default:
			continue
		}

		if idx := strings.Index(rest, "/"); idx >= 0 {
			dirName := rest[:idx]
			dirKey := dirName
			if cleanedPrefix != "" {
				dirKey = cleanedPrefix + "/" + dirName
			}
			if _, seen := dirSeen[dirKey]; !seen {
				dirSeen[dirKey] = struct{}{}
				result.Files = append(result.Files, virefs.FileInfo{
					Key:   dirKey,
					IsDir: true,
				})
			}
		} else {
			result.Files = append(result.Files, fileInfoFromZip(k, f))
		}
	}
	return result, nil
}

// Stat implements virefs.FS.
func (z *FS) Stat(_ context.Context, key string) (*virefs.FileInfo, error) {
	cleaned, err := virefs.CleanKey(key)
	if err != nil {
		return nil, &virefs.OpError{Op: "Stat", Key: key, Err: err}
	}
	f, ok := z.index[cleaned]
	if !ok {
		return nil, &virefs.OpError{Op: "Stat", Key: key, Err: virefs.ErrNotFound}
	}
	fi := fileInfoFromZip(cleaned, f)
	return &fi, nil
}

// Access implements virefs.FS. Always returns ErrNotSupported.
func (z *FS) Access(_ context.Context, key string) (*virefs.AccessInfo, error) {
	return nil, &virefs.OpError{Op: "Access", Key: key, Err: virefs.ErrNotSupported}
}

// buildIndex creates a key -> *zip.File map with normalised keys.
func buildIndex(zr *zip.Reader) map[string]*zip.File {
	idx := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		key, err := virefs.CleanKey(f.Name)
		if err != nil || key == "" {
			continue
		}
		idx[key] = f
	}
	return idx
}

func fileInfoFromZip(key string, f *zip.File) virefs.FileInfo {
	return virefs.FileInfo{
		Key:          key,
		Size:         int64(f.UncompressedSize64),
		LastModified: f.Modified,
		IsDir:        f.FileInfo().IsDir(),
		ContentType:  mime.TypeByExtension(path.Ext(key)),
	}
}
