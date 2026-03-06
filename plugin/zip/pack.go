package zip

import (
	"archive/zip"
	"context"
	"io"

	virefs "github.com/lin-snow/VireFS"
)

// Pack reads the listed keys from fsys and writes them into a zip archive
// streamed to w. Keys are used as entry names inside the archive after
// normalisation via virefs.CleanKey.
func Pack(ctx context.Context, fsys virefs.FS, keys []string, w io.Writer) error {
	zw := zip.NewWriter(w)

	for _, raw := range keys {
		if err := ctx.Err(); err != nil {
			zw.Close()
			return err
		}

		key, err := virefs.CleanKey(raw)
		if err != nil {
			zw.Close()
			return err
		}

		header := &zip.FileHeader{
			Name:   key,
			Method: zip.Deflate,
		}

		if info, err := fsys.Stat(ctx, key); err == nil {
			header.UncompressedSize64 = uint64(info.Size)
			header.Modified = info.LastModified
		}

		ew, err := zw.CreateHeader(header)
		if err != nil {
			zw.Close()
			return err
		}

		rc, err := fsys.Get(ctx, key)
		if err != nil {
			zw.Close()
			return err
		}

		_, copyErr := io.Copy(ew, rc)
		rc.Close()
		if copyErr != nil {
			zw.Close()
			return copyErr
		}
	}

	return zw.Close()
}
