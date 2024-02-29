/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

// go官方没有实现ecb加密模式
package security

import (
	"crypto/cipher"
)

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

func NewECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}

func (x *ecbEncrypter) BlockSize() int { return x.blockSize }

func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("dm/security: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("dm/security: output smaller than input")
	}
	if InexactOverlap(dst[:len(src)], src) {
		panic("dm/security: invalid buffer overlap")
	}
	for bs, be := 0, x.blockSize; bs < len(src); bs, be = bs+x.blockSize, be+x.blockSize {
		x.b.Encrypt(dst[bs:be], src[bs:be])
	}
}

type ecbDecrypter ecb

func NewECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}

func (x *ecbDecrypter) BlockSize() int { return x.blockSize }

func (x *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("dm/security: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("dm/security: output smaller than input")
	}
	if InexactOverlap(dst[:len(src)], src) {
		panic("dm/security: invalid buffer overlap")
	}
	for bs, be := 0, x.blockSize; bs < len(src); bs, be = bs+x.blockSize, be+x.blockSize {
		x.b.Decrypt(dst[bs:be], src[bs:be])
	}
}
