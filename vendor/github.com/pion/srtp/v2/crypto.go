package srtp

import (
	"crypto/cipher"
)

// xorBytes computes the exclusive-or of src1 and src2 and stores it in dst.
// It returns the number of bytes written.
func xorBytes(dst, src1, src2 []byte) int {
	n := len(src1)
	if len(src2) < n {
		n = len(src2)
	}
	if len(dst) < n {
		n = len(dst)
	}

	for i := 0; i < n; i++ {
		dst[i] = src1[i] ^ src2[i]
	}

	return n
}

// incrementCTR increments a big-endian integer of arbitrary size.
func incrementCTR(ctr []byte) {
	for i := len(ctr) - 1; i >= 0; i-- {
		ctr[i]++
		if ctr[i] != 0 {
			break
		}
	}
}

// xorBytesCTR performs CTR encryption and decryption.
// It is equivalent to cipher.NewCTR followed by XORKeyStream.
func xorBytesCTR(block cipher.Block, iv []byte, dst, src []byte) error {
	if len(iv) != block.BlockSize() {
		return errBadIVLength
	}

	ctr := make([]byte, len(iv))
	copy(ctr, iv)
	bs := block.BlockSize()
	stream := make([]byte, bs)

	i := 0
	for i < len(src) {
		block.Encrypt(stream, ctr)
		incrementCTR(ctr)
		n := xorBytes(dst[i:], src[i:], stream)
		if n == 0 {
			break
		}
		i += n
	}
	return nil
}
