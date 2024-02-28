/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package security

import "math/big"

type DhKey struct {
	x     *big.Int
	y     *big.Int
	group *dhGroup
}

func newPublicKey(s []byte) *DhKey {
	key := new(DhKey)
	key.y = new(big.Int).SetBytes(s)
	return key
}

func (dk *DhKey) GetX() *big.Int {
	x := new(big.Int)
	x.Set(dk.x)
	return x
}

func (dk *DhKey) GetY() *big.Int {
	y := new(big.Int)
	y.Set(dk.y)
	return y
}

func (dk *DhKey) GetYBytes() []byte {
	if dk.y == nil {
		return nil
	}
	if dk.group != nil {
		blen := (dk.group.p.BitLen() + 7) / 8
		ret := make([]byte, blen)
		copyWithLeftPad(ret, dk.y.Bytes())
		return ret
	}
	return dk.y.Bytes()
}

func (dk *DhKey) GetYString() string {
	if dk.y == nil {
		return ""
	}
	return dk.y.String()
}

func (dk *DhKey) IsPrivateKey() bool {
	return dk.x != nil
}

func copyWithLeftPad(dest, src []byte) {
	numPaddingBytes := len(dest) - len(src)
	for i := 0; i < numPaddingBytes; i++ {
		dest[i] = 0
	}
	copy(dest[:numPaddingBytes], src)
}
