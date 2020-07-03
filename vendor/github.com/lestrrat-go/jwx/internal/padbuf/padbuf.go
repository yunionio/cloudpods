// Package padbuf implements a simple buffer that knows how to pad/unpad
// itself so that the buffer size aligns with an arbitrary block size.
package padbuf

import "errors"

type PadBuffer []byte

func (pb PadBuffer) Len() int {
	return len(pb)
}

func (pb PadBuffer) Pad(n int) PadBuffer {
	rem := n - pb.Len()%n
	if rem == 0 {
		return pb
	}

	newpb := pb.Resize(pb.Len() + rem)

	pad := make([]byte, rem)
	for i := 0; i < rem; i++ {
		pad[i] = byte(rem)
	}
	copy(newpb[pb.Len():], pad)

	return newpb
}

func (pb PadBuffer) Resize(newlen int) PadBuffer {
	if pb.Len() == newlen {
		return pb
	}

	buf := make([]byte, newlen)
	copy(buf, pb)
	return PadBuffer(buf)
}

func (pb PadBuffer) Unpad(n int) (PadBuffer, error) {
	rem := pb.Len() % n
	if rem != 0 {
		return pb, errors.New("buffer should be multiple block size")
	}

	last := pb[pb.Len()-1]

	count := 0
	for i := pb.Len() - 1; i >= 0; i-- {
		if pb[i] != last {
			break
		}
		count++
	}

	if count != int(last) {
		return pb, errors.New("invalid padding")
	}

	return pb[:pb.Len()-int(last)], nil
}
