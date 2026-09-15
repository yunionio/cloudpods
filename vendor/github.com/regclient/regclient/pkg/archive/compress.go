package archive

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"io"
)

// CompressType identifies the detected compression type
type CompressType int

const (
	// CompressNone detected no compression
	CompressNone CompressType = iota
	// CompressBzip2 compression
	CompressBzip2
	// CompressGzip compression
	CompressGzip
	// CompressXz compression
	CompressXz
)

// compressHeaders are used to detect the compression type
var compressHeaders = map[CompressType][]byte{
	CompressBzip2: []byte("\x42\x5A\x68"),
	CompressGzip:  []byte("\x1F\x8B\x08"),
	CompressXz:    []byte("\xFD\x37\x7A\x58\x5A\x00"),
}

func Compress(r io.Reader, oComp CompressType) (io.Reader, error) {
	br := bufio.NewReader(r)
	head, err := br.Peek(10)
	if err != nil {
		return br, err
	}
	rComp := DetectCompression(head)
	if rComp == oComp {
		return br, nil
	}
	switch oComp {
	case CompressGzip:
		switch rComp {
		case CompressNone:
			return compressGzip(br)
		case CompressBzip2:
			return compressGzip(bzip2.NewReader(br))
		}
	}
	// No other types currently supported
	return nil, ErrUnknownType
}

func compressGzip(src io.Reader) (io.Reader, error) {
	pipeR, pipeW := io.Pipe()
	go func() {
		defer pipeW.Close()
		gzipW := gzip.NewWriter(pipeW)
		defer gzipW.Close()
		_, _ = io.Copy(gzipW, src)
	}()
	return pipeR, nil
}

// Decompress extracts gzip and bzip streams
func Decompress(r io.Reader) (io.Reader, error) {
	// create bufio to peak on first few bytes
	br := bufio.NewReader(r)
	head, err := br.Peek(10)
	if err != nil {
		return br, err
	}

	// compare peaked data against known compression types
	switch DetectCompression(head) {
	case CompressBzip2:
		return bzip2.NewReader(br), nil
	case CompressGzip:
		return gzip.NewReader(br)
	case CompressXz:
		return br, ErrXzUnsupported
	default:
		return br, nil
	}
}

// DetectCompression identifies the compression type based on the first few bytes
func DetectCompression(head []byte) CompressType {
	for c, b := range compressHeaders {
		if bytes.HasPrefix(head, b) {
			return c
		}
	}
	return CompressNone
}

func (ct CompressType) String() string {
	switch ct {
	case CompressNone:
		return "none"
	case CompressBzip2:
		return "bzip2"
	case CompressGzip:
		return "gzip"
	case CompressXz:
		return "xz"
	}
	return "unknown"
}
