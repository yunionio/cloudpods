package tos

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"strings"
)

var (
	ErrETagMissMatch = errors.New("tos: ETag miss match")
)

// ETagCheckReadCloser checks ETag on read EOF
type ETagCheckReadCloser struct {
	reader    io.Reader
	closer    io.Closer
	checksum  hash.Hash
	eTag      string
	requestID string
}

func NewETagCheckReadCloser(reader io.ReadCloser, eTag, requestID string) *ETagCheckReadCloser {
	checksum := md5.New()

	return &ETagCheckReadCloser{
		reader:    io.TeeReader(reader, checksum),
		closer:    reader,
		checksum:  checksum,
		eTag:      strings.Trim(eTag, `"`),
		requestID: requestID,
	}
}

func (ec *ETagCheckReadCloser) Read(p []byte) (n int, err error) {
	n, err = ec.reader.Read(p)
	if err == io.EOF && len(ec.eTag) > 0 {
		sum := ec.checksum.Sum(nil)
		if hexSum := hex.EncodeToString(sum); hexSum != ec.eTag {
			return n, &ChecksumError{
				RequestID:        ec.requestID,
				ExpectedChecksum: ec.eTag,
				ActualChecksum:   hexSum,
			}
		}
	}

	return n, err
}

func (ec *ETagCheckReadCloser) Close() error {
	return ec.closer.Close()
}
