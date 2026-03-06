package zip

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"strings"

	virefs "github.com/lin-snow/VireFS"
)

// ZipFS is a read-only virefs.FS backed by a zip archive.
// Put, Delete and Access always return virefs.ErrNotSupported.
type ZipFS struct {
	r      *zip.Reader
	closer io.Closer
	index  map[string]*zip.File
}

// compile-time interface check
var _ virefs.FS = (*ZipFS)(nil)

// OpenFS opens a zip file at path and returns a read-only FS.
// The caller must call Close when done.
func OpenFS(path string) (*ZipFS, error) {
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
	return &ZipFS{
		r:      zr,
		closer: f,
		index:  buildIndex(zr),
	}, nil
}

// NewFS creates a read-only FS from an io.ReaderAt.
// The caller is responsible for the lifetime of ra.
func NewFS(ra io.ReaderAt, size int64) (*ZipFS, error) {
	zr, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, err
	}
	return &ZipFS{
		r:     zr,
		index: buildIndex(zr),
	}, nil
}

// NewFSFromBytes creates a read-only FS from in-memory bytes.
func NewFSFromBytes(data []byte) (*ZipFS, error) {
	return NewFS(bytes.NewReader(data), int64(len(data)))
}

// Close releases the underlying file handle if one was opened by OpenFS.
func (z *ZipFS) Close() error {
	if z.closer != nil {
		return z.closer.Close()
	}
	return nil
}

func (z *ZipFS) Get(_ context.Context, key string) (io.ReadCloser, error) {
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

func (z *ZipFS) Put(_ context.Context, key string, _ io.Reader, _ ...virefs.PutOption) error {
	return &virefs.OpError{Op: "Put", Key: key, Err: virefs.ErrNotSupported}
}

func (z *ZipFS) Delete(_ context.Context, key string) error {
	return &virefs.OpError{Op: "Delete", Key: key, Err: virefs.ErrNotSupported}
}

func (z *ZipFS) List(_ context.Context, prefix string) (*virefs.ListResult, error) {
	cleanedPrefix, err := virefs.CleanKey(prefix)
	if err != nil {
		return nil, &virefs.OpError{Op: "List", Key: prefix, Err: err}
	}
	result := &virefs.ListResult{}
	for k, f := range z.index {
		if cleanedPrefix != "" && !strings.HasPrefix(k, cleanedPrefix+"/") && k != cleanedPrefix {
			continue
		}
		result.Files = append(result.Files, fileInfoFromZip(k, f))
	}
	return result, nil
}

func (z *ZipFS) Stat(_ context.Context, key string) (*virefs.FileInfo, error) {
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

func (z *ZipFS) Access(_ context.Context, key string) (*virefs.AccessInfo, error) {
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
	}
}
