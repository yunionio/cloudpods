/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package security

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rc4"
	"errors"
	"reflect"
)

type SymmCipher struct {
	encryptCipher interface{} //cipher.BlockMode | cipher.Stream
	decryptCipher interface{} //cipher.BlockMode | cipher.Stream
	key           []byte
	block         cipher.Block // 分组加密算法
	algorithmType int
	workMode      int
	needPadding   bool
}

func NewSymmCipher(algorithmID int, key []byte) (SymmCipher, error) {
	var sc SymmCipher
	var err error
	sc.key = key
	sc.algorithmType = algorithmID & ALGO_MASK
	sc.workMode = algorithmID & WORK_MODE_MASK
	switch sc.algorithmType {
	case AES128:
		if sc.block, err = aes.NewCipher(key[:16]); err != nil {
			return sc, err
		}
	case AES192:
		if sc.block, err = aes.NewCipher(key[:24]); err != nil {
			return sc, err
		}
	case AES256:
		if sc.block, err = aes.NewCipher(key[:32]); err != nil {
			return sc, err
		}
	case DES:
		if sc.block, err = des.NewCipher(key[:8]); err != nil {
			return sc, err
		}
	case DES3:
		var tripleDESKey []byte
		tripleDESKey = append(tripleDESKey, key[:16]...)
		tripleDESKey = append(tripleDESKey, key[:8]...)
		if sc.block, err = des.NewTripleDESCipher(tripleDESKey); err != nil {
			return sc, err
		}
	case RC4:
		if sc.encryptCipher, err = rc4.NewCipher(key[:16]); err != nil {
			return sc, err
		}
		if sc.decryptCipher, err = rc4.NewCipher(key[:16]); err != nil {
			return sc, err
		}
		return sc, nil
	default:
		return sc, errors.New("invalidCipher")
	}
	blockSize := sc.block.BlockSize()
	if sc.encryptCipher, err = sc.getEncrypter(sc.workMode, sc.block, defaultIV[:blockSize]); err != nil {
		return sc, err
	}
	if sc.decryptCipher, err = sc.getDecrypter(sc.workMode, sc.block, defaultIV[:blockSize]); err != nil {
		return sc, err
	}
	return sc, nil
}

func (sc SymmCipher) Encrypt(plaintext []byte, genDigest bool) []byte {
	// 执行过加密后,IV值变了,需要重新初始化encryptCipher对象(因为没有类似resetIV的方法)
	if sc.algorithmType != RC4 {
		sc.encryptCipher, _ = sc.getEncrypter(sc.workMode, sc.block, defaultIV[:sc.block.BlockSize()])
	} else {
		sc.encryptCipher, _ = rc4.NewCipher(sc.key[:16])
	}
	// 填充
	var paddingtext = make([]byte, len(plaintext))
	copy(paddingtext, plaintext)
	if sc.needPadding {
		paddingtext = pkcs5Padding(paddingtext)
	}

	ret := make([]byte, len(paddingtext))

	if v, ok := sc.encryptCipher.(cipher.Stream); ok {
		v.XORKeyStream(ret, paddingtext)
	} else if v, ok := sc.encryptCipher.(cipher.BlockMode); ok {
		v.CryptBlocks(ret, paddingtext)
	}

	// md5摘要
	if genDigest {
		digest := md5.Sum(plaintext)
		encrypt := ret
		ret = make([]byte, len(encrypt)+len(digest))
		copy(ret[:len(encrypt)], encrypt)
		copy(ret[len(encrypt):], digest[:])
	}
	return ret
}

func (sc SymmCipher) Decrypt(ciphertext []byte, checkDigest bool) ([]byte, error) {
	// 执行过解密后,IV值变了,需要重新初始化decryptCipher对象(因为没有类似resetIV的方法)
	if sc.algorithmType != RC4 {
		sc.decryptCipher, _ = sc.getDecrypter(sc.workMode, sc.block, defaultIV[:sc.block.BlockSize()])
	} else {
		sc.decryptCipher, _ = rc4.NewCipher(sc.key[:16])
	}
	var ret []byte
	if checkDigest {
		var digest = ciphertext[len(ciphertext)-MD5_DIGEST_SIZE:]
		ret = ciphertext[:len(ciphertext)-MD5_DIGEST_SIZE]
		ret = sc.decrypt(ret)
		var msgDigest = md5.Sum(ret)
		if !reflect.DeepEqual(msgDigest[:], digest) {
			return nil, errors.New("Decrypt failed/Digest not match\n")
		}
	} else {
		ret = sc.decrypt(ciphertext)
	}
	return ret, nil
}

func (sc SymmCipher) decrypt(ciphertext []byte) []byte {
	ret := make([]byte, len(ciphertext))
	if v, ok := sc.decryptCipher.(cipher.Stream); ok {
		v.XORKeyStream(ret, ciphertext)
	} else if v, ok := sc.decryptCipher.(cipher.BlockMode); ok {
		v.CryptBlocks(ret, ciphertext)
	}
	// 去除填充
	if sc.needPadding {
		ret = pkcs5UnPadding(ret)
	}
	return ret
}

func (sc *SymmCipher) getEncrypter(workMode int, block cipher.Block, iv []byte) (ret interface{}, err error) {
	switch workMode {
	case ECB_MODE:
		ret = NewECBEncrypter(block)
		sc.needPadding = true
	case CBC_MODE:
		ret = cipher.NewCBCEncrypter(block, iv)
		sc.needPadding = true
	case CFB_MODE:
		ret = cipher.NewCFBEncrypter(block, iv)
		sc.needPadding = false
	case OFB_MODE:
		ret = cipher.NewOFB(block, iv)
		sc.needPadding = false
	default:
		err = errors.New("invalidCipherMode")
	}
	return
}

func (sc *SymmCipher) getDecrypter(workMode int, block cipher.Block, iv []byte) (ret interface{}, err error) {
	switch workMode {
	case ECB_MODE:
		ret = NewECBDecrypter(block)
		sc.needPadding = true
	case CBC_MODE:
		ret = cipher.NewCBCDecrypter(block, iv)
		sc.needPadding = true
	case CFB_MODE:
		ret = cipher.NewCFBDecrypter(block, iv)
		sc.needPadding = false
	case OFB_MODE:
		ret = cipher.NewOFB(block, iv)
		sc.needPadding = false
	default:
		err = errors.New("invalidCipherMode")
	}
	return
}

// 补码
func pkcs77Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// 去码
func pkcs7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:length-unpadding]
}

// 补码
func pkcs5Padding(ciphertext []byte) []byte {
	return pkcs77Padding(ciphertext, 8)
}

// 去码
func pkcs5UnPadding(ciphertext []byte) []byte {
	return pkcs7UnPadding(ciphertext)
}
