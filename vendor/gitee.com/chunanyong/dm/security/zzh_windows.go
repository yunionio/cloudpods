/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package security

import (
	"syscall"
)

var (
	dmCipherEncryptDLL          *syscall.LazyDLL
	cipherGetCountProc          *syscall.LazyProc
	cipherGetInfoProc           *syscall.LazyProc
	cipherEncryptInitProc       *syscall.LazyProc
	cipherGetCipherTextSizeProc *syscall.LazyProc
	cipherEncryptProc           *syscall.LazyProc
	cipherCleanupProc           *syscall.LazyProc
	cipherDecryptInitProc       *syscall.LazyProc
	cipherDecryptProc           *syscall.LazyProc
)

func initThirdPartCipher(cipherPath string) error {
	dmCipherEncryptDLL = syscall.NewLazyDLL(cipherPath)
	if err := dmCipherEncryptDLL.Load(); err != nil {
		return err
	}
	cipherGetCountProc = dmCipherEncryptDLL.NewProc("cipher_get_count")
	cipherGetInfoProc = dmCipherEncryptDLL.NewProc("cipher_get_info")
	cipherEncryptInitProc = dmCipherEncryptDLL.NewProc("cipher_encrypt_init")
	cipherGetCipherTextSizeProc = dmCipherEncryptDLL.NewProc("cipher_get_cipher_text_size")
	cipherEncryptProc = dmCipherEncryptDLL.NewProc("cipher_encrypt")
	cipherCleanupProc = dmCipherEncryptDLL.NewProc("cipher_cleanup")
	cipherDecryptInitProc = dmCipherEncryptDLL.NewProc("cipher_decrypt_init")
	cipherDecryptProc = dmCipherEncryptDLL.NewProc("cipher_decrypt")
	return nil
}

func cipherGetCount() int {
	ret, _, _ := cipherGetCountProc.Call()
	return int(ret)
}

func cipherGetInfo(seqno, cipherId, cipherName, _type, blkSize, khSIze uintptr) {
	ret, _, _ := cipherGetInfoProc.Call(seqno, cipherId, cipherName, _type, blkSize, khSIze)
	if ret == 0 {
		panic("ThirdPartyCipher: call cipher_get_info failed")
	}
}

func cipherEncryptInit(cipherId, key, keySize, cipherPara uintptr) {
	ret, _, _ := cipherEncryptInitProc.Call(cipherId, key, keySize, cipherPara)
	if ret == 0 {
		panic("ThirdPartyCipher: call cipher_encrypt_init failed")
	}
}

func cipherGetCipherTextSize(cipherId, cipherPara, plainTextSize uintptr) uintptr {
	ciphertextLen, _, _ := cipherGetCipherTextSizeProc.Call(cipherId, cipherPara, plainTextSize)
	return ciphertextLen
}

func cipherEncrypt(cipherId, cipherPara, plainText, plainTextSize, cipherText, cipherTextBufSize uintptr) uintptr {
	ret, _, _ := cipherEncryptProc.Call(cipherId, cipherPara, plainText, plainTextSize, cipherText, cipherTextBufSize)
	return ret
}

func cipherClean(cipherId, cipherPara uintptr) {
	_, _, _ = cipherCleanupProc.Call(cipherId, cipherPara)
}

func cipherDecryptInit(cipherId, key, keySize, cipherPara uintptr) {
	ret, _, _ := cipherDecryptInitProc.Call(cipherId, key, keySize, cipherPara)
	if ret == 0 {
		panic("ThirdPartyCipher: call cipher_decrypt_init failed")
	}
}

func cipherDecrypt(cipherId, cipherPara, cipherText, cipherTextSize, plainText, plainTextBufSize uintptr) uintptr {
	ret, _, _ := cipherDecryptProc.Call(cipherId, cipherPara, cipherText, cipherTextSize, plainText, plainTextBufSize)
	return ret
}
