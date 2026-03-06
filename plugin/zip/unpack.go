package zip

import (
	"archive/zip"
	"context"
	"io"

	virefs "github.com/lin-snow/VireFS"
)

// Unpack reads a zip archive from r and writes every file entry into dst
// under the given prefix. Directory entries are skipped. Each entry name is
// normalised via virefs.CleanKey before being joined with prefix.
func Unpack(ctx context.Context, r io.ReaderAt, size int64, dst virefs.FS, prefix string, opts ...virefs.PutOption) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return err
	}

	cleanedPrefix, err := virefs.CleanKey(prefix)
	if err != nil {
		return err
	}

	for _, f := range zr.File {
		if err := ctx.Err(); err != nil {
			return err
		}

		if f.FileInfo().IsDir() {
			continue
		}

		name, err := virefs.CleanKey(f.Name)
		if err != nil || name == "" {
			continue
		}

		dstKey := name
		if cleanedPrefix != "" {
			dstKey = cleanedPrefix + "/" + name
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		putErr := dst.Put(ctx, dstKey, rc, opts...)
		rc.Close()
		if putErr != nil {
			return putErr
		}
	}
	return nil
}
