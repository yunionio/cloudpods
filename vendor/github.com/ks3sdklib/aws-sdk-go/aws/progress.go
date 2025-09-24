package aws

import (
	"bytes"
	"io"
	"os"
	"strings"
)

type ProgressFunc func(increment, completed, total int64)

type teeReader struct {
	reader  io.Reader
	writer  io.Writer
	tracker *readerTracker
}

type readerTracker struct {
	completedBytes int64
	totalBytes     int64
	progressFunc   ProgressFunc
}

// TeeReader returns a Reader that writes to w what it reads from r.
// All reads from r performed through it are matched with
// corresponding writes to w.  There is no internal buffering -
// to write must complete before the read completes.
// Any error encountered while writing is reported as a read error.
func TeeReader(reader io.Reader, writer io.Writer, totalBytes int64, progressFunc ProgressFunc) io.ReadCloser {
	return &teeReader{
		reader: reader,
		writer: writer,
		tracker: &readerTracker{
			completedBytes: 0,
			totalBytes:     totalBytes,
			progressFunc:   progressFunc,
		},
	}
}

func (t *teeReader) Read(p []byte) (n int, err error) {
	n, err = t.reader.Read(p)

	// Read encountered error
	if err != nil && err != io.EOF {
		return
	}

	if n > 0 {
		// update completedBytes
		t.tracker.completedBytes += int64(n)

		if t.tracker.progressFunc != nil {
			// report progress
			t.tracker.progressFunc(int64(n), t.tracker.completedBytes, t.tracker.totalBytes)
		}
		// CRC
		if t.writer != nil {
			if n, err := t.writer.Write(p[:n]); err != nil {
				return n, err
			}
		}
	}

	return
}

func (t *teeReader) Close() error {
	if rc, ok := t.reader.(io.ReadCloser); ok {
		return rc.Close()
	}
	return nil
}

// GetReaderLen returns the length of the reader
func GetReaderLen(reader io.Reader) int64 {
	var contentLength int64
	switch v := reader.(type) {
	case *bytes.Buffer:
		contentLength = int64(v.Len())
	case *bytes.Reader:
		contentLength = int64(v.Len())
	case *strings.Reader:
		contentLength = int64(v.Len())
	case *os.File:
		fileInfo, err := v.Stat()
		if err != nil {
			contentLength = 0
		} else {
			contentLength = fileInfo.Size()
		}
	default:
		contentLength = 0
	}
	return contentLength
}
