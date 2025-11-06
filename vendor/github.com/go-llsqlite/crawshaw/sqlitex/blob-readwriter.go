package sqlitex

import (
	"errors"
	"io"

	sqlite "github.com/go-llsqlite/crawshaw"
)

// This wraps a sqlite.Blob with positional state provide extra io implementations.
func NewBlobSeeker(blob *sqlite.Blob) *blobSeeker {
	return &blobSeeker{
		size:     blob.Size(),
		blob:     blob,
		ReaderAt: blob,
		WriterAt: blob,
	}
}

var _ interface {
	io.ReadWriteSeeker
} = (*blobSeeker)(nil)

type blobSeeker struct {
	io.ReaderAt
	io.WriterAt
	blob *sqlite.Blob
	off  int64
	size int64
}

func (blob *blobSeeker) Read(p []byte) (n int, err error) {
	if blob.off >= blob.size {
		return 0, io.EOF
	}
	if rem := blob.size - blob.off; int64(len(p)) > rem {
		p = p[:rem]
	}
	n, err = blob.ReadAt(p, blob.off)
	blob.off += int64(n)
	return n, err
}

func (blob *blobSeeker) Write(p []byte) (n int, err error) {
	if rem := blob.size - blob.off; int64(len(p)) > rem {
		return 0, io.ErrShortWrite
	}
	n, err = blob.WriteAt(p, blob.off)
	blob.off += int64(n)
	return n, err
}

func (blob *blobSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		// use offset directly
	case io.SeekCurrent:
		offset += blob.off
	case io.SeekEnd:
		offset += blob.size
	}
	if offset < 0 {
		return blob.off, errors.New("seek to offset < 0")
	}
	blob.off = offset
	return offset, nil
}
