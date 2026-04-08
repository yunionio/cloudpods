// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package tracer

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync/atomic"

	"github.com/tinylib/msgp/msgp"
)

// payload is a wrapper on top of the msgpack encoder which allows constructing an
// encoded array by pushing its entries sequentially, one at a time. It basically
// allows us to encode as we would with a stream, except that the contents of the stream
// can be read as a slice by the msgpack decoder at any time. It follows the guidelines
// from the msgpack array spec:
// https://github.com/msgpack/msgpack/blob/master/spec.md#array-format-family
//
// payload implements io.Reader and can be used with the decoder directly. To create
// a new payload use the newPayload method.
//
// payload is not safe for concurrent use, is meant to be used only once and eventually
// dismissed.
type payload struct {
	// header specifies the first few bytes in the msgpack stream
	// indicating the type of array (fixarray, array16 or array32)
	// and the number of items contained in the stream.
	header []byte

	// off specifies the current read position on the header.
	off int

	// count specifies the number of items in the stream.
	count uint32

	// buf holds the sequence of msgpack-encoded items.
	buf bytes.Buffer
}

var _ io.Reader = (*payload)(nil)

// newPayload returns a ready to use payload.
func newPayload() *payload {
	p := &payload{
		header: make([]byte, 8),
		off:    8,
	}
	return p
}

// push pushes a new item into the stream.
func (p *payload) push(t spanList) error {
	if err := msgp.Encode(&p.buf, t); err != nil {
		return err
	}
	atomic.AddUint32(&p.count, 1)
	p.updateHeader()
	return nil
}

// itemCount returns the number of items available in the srteam.
func (p *payload) itemCount() int {
	return int(atomic.LoadUint32(&p.count))
}

// size returns the payload size in bytes. After the first read the value becomes
// inaccurate by up to 8 bytes.
func (p *payload) size() int {
	return p.buf.Len() + len(p.header) - p.off
}

// reset should *not* be used. It is not implemented and is only here to serve
// as information on how to implement it in case the same payload object ever
// needs to be reused.
func (p *payload) reset() {
	// ⚠️  Warning!
	//
	// Resetting the payload for re-use requires the transport to wait for the
	// HTTP package to Close the request body before attempting to re-use it
	// again! This requires additional logic to be in place. See:
	//
	// • https://github.com/golang/go/blob/go1.16/src/net/http/client.go#L136-L138
	// • https://github.com/DataDog/dd-trace-go/pull/475
	// • https://github.com/DataDog/dd-trace-go/pull/549
	// • https://github.com/DataDog/dd-trace-go/pull/976
	//
	panic("not implemented")
}

// https://github.com/msgpack/msgpack/blob/master/spec.md#array-format-family
const (
	msgpackArrayFix byte = 144  // up to 15 items
	msgpackArray16       = 0xdc // up to 2^16-1 items, followed by size in 2 bytes
	msgpackArray32       = 0xdd // up to 2^32-1 items, followed by size in 4 bytes
)

// updateHeader updates the payload header based on the number of items currently
// present in the stream.
func (p *payload) updateHeader() {
	n := uint64(atomic.LoadUint32(&p.count))
	switch {
	case n <= 15:
		p.header[7] = msgpackArrayFix + byte(n)
		p.off = 7
	case n <= 1<<16-1:
		binary.BigEndian.PutUint64(p.header, n) // writes 2 bytes
		p.header[5] = msgpackArray16
		p.off = 5
	default: // n <= 1<<32-1
		binary.BigEndian.PutUint64(p.header, n) // writes 4 bytes
		p.header[3] = msgpackArray32
		p.off = 3
	}
}

// Close implements io.Closer
func (p *payload) Close() error {
	// Once the payload has been read, clear the buffer for garbage collection to avoid
	// a memory leak when references to this object may still be kept by faulty transport
	// implementations or the standard library. See dd-trace-go#976
	p.buf = bytes.Buffer{}
	return nil
}

// Read implements io.Reader. It reads from the msgpack-encoded stream.
func (p *payload) Read(b []byte) (n int, err error) {
	if p.off < len(p.header) {
		// reading header
		n = copy(b, p.header[p.off:])
		p.off += n
		return n, nil
	}
	return p.buf.Read(b)
}
