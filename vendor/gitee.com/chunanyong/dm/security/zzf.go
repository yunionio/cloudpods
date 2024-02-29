/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package security

import (
	"crypto/md5"
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

type ThirdPartCipher struct {
	encryptType int    // 外部加密算法id
	encryptName string // 外部加密算法名称
	hashType    int
	key         []byte
	cipherCount int // 外部加密算法个数
	//innerId		int // 外部加密算法内部id
	blockSize int // 分组块大小
	khSize    int // key/hash大小
}

func NewThirdPartCipher(encryptType int, key []byte, cipherPath string, hashType int) (ThirdPartCipher, error) {
	var tpc = ThirdPartCipher{
		encryptType: encryptType,
		key:         key,
		hashType:    hashType,
		cipherCount: -1,
	}
	var err error
	err = initThirdPartCipher(cipherPath)
	if err != nil {
		return tpc, err
	}
	tpc.getCount()
	if err = tpc.getInfo(); err != nil {
		return tpc, err
	}
	return tpc, nil
}

func (tpc *ThirdPartCipher) getCount() int {
	if tpc.cipherCount == -1 {
		tpc.cipherCount = cipherGetCount()
	}
	return tpc.cipherCount
}

func (tpc *ThirdPartCipher) getInfo() error {
	var cipher_id, ty, blk_size, kh_size int
	//var strptr, _ = syscall.UTF16PtrFromString(tpc.encryptName)
	var strptr *uint16 = new(uint16)
	for i := 1; i <= tpc.getCount(); i++ {
		cipherGetInfo(uintptr(i), uintptr(unsafe.Pointer(&cipher_id)), uintptr(unsafe.Pointer(&strptr)),
			uintptr(unsafe.Pointer(&ty)), uintptr(unsafe.Pointer(&blk_size)), uintptr(unsafe.Pointer(&kh_size)))
		if tpc.encryptType == cipher_id {
			tpc.blockSize = blk_size
			tpc.khSize = kh_size
			tpc.encryptName = string(uintptr2bytes(uintptr(unsafe.Pointer(strptr))))
			return nil
		}
	}
	return fmt.Errorf("ThirdPartyCipher: cipher id:%d not found", tpc.encryptType)
}

func (tpc ThirdPartCipher) Encrypt(plaintext []byte, genDigest bool) []byte {
	var tmp_para uintptr
	cipherEncryptInit(uintptr(tpc.encryptType), uintptr(unsafe.Pointer(&tpc.key[0])), uintptr(len(tpc.key)), tmp_para)

	ciphertextLen := cipherGetCipherTextSize(uintptr(tpc.encryptType), tmp_para, uintptr(len(plaintext)))

	ciphertext := make([]byte, ciphertextLen)
	ret := cipherEncrypt(uintptr(tpc.encryptType), tmp_para, uintptr(unsafe.Pointer(&plaintext[0])), uintptr(len(plaintext)),
		uintptr(unsafe.Pointer(&ciphertext[0])), uintptr(len(ciphertext)))
	ciphertext = ciphertext[:ret]

	cipherClean(uintptr(tpc.encryptType), tmp_para)
	// md5摘要
	if genDigest {
		digest := md5.Sum(plaintext)
		encrypt := ciphertext
		ciphertext = make([]byte, len(encrypt)+len(digest))
		copy(ciphertext[:len(encrypt)], encrypt)
		copy(ciphertext[len(encrypt):], digest[:])
	}
	return ciphertext
}

func (tpc ThirdPartCipher) Decrypt(ciphertext []byte, checkDigest bool) ([]byte, error) {
	var ret []byte
	if checkDigest {
		var digest = ciphertext[len(ciphertext)-MD5_DIGEST_SIZE:]
		ret = ciphertext[:len(ciphertext)-MD5_DIGEST_SIZE]
		ret = tpc.decrypt(ret)
		var msgDigest = md5.Sum(ret)
		if !reflect.DeepEqual(msgDigest[:], digest) {
			return nil, errors.New("Decrypt failed/Digest not match\n")
		}
	} else {
		ret = tpc.decrypt(ciphertext)
	}
	return ret, nil
}

func (tpc ThirdPartCipher) decrypt(ciphertext []byte) []byte {
	var tmp_para uintptr

	cipherDecryptInit(uintptr(tpc.encryptType), uintptr(unsafe.Pointer(&tpc.key[0])), uintptr(len(tpc.key)), tmp_para)

	plaintext := make([]byte, len(ciphertext))
	ret := cipherDecrypt(uintptr(tpc.encryptType), tmp_para, uintptr(unsafe.Pointer(&ciphertext[0])), uintptr(len(ciphertext)),
		uintptr(unsafe.Pointer(&plaintext[0])), uintptr(len(plaintext)))
	plaintext = plaintext[:ret]

	cipherClean(uintptr(tpc.encryptType), tmp_para)
	return plaintext
}

func addBufSize(buf []byte, newCap int) []byte {
	newBuf := make([]byte, newCap)
	copy(newBuf, buf)
	return newBuf
}

func uintptr2bytes(p uintptr) []byte {
	buf := make([]byte, 64)
	i := 0
	for b := (*byte)(unsafe.Pointer(p)); *b != 0; i++ {
		if i > cap(buf) {
			buf = addBufSize(buf, i*2)
		}
		buf[i] = *b
		// byte占1字节
		p++
		b = (*byte)(unsafe.Pointer(p))
	}
	return buf[:i]
}
