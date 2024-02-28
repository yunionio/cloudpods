/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package security

import "plugin"

var (
	dmCipherEncryptSo           *plugin.Plugin
	cipherGetCountProc          plugin.Symbol
	cipherGetInfoProc           plugin.Symbol
	cipherEncryptInitProc       plugin.Symbol
	cipherGetCipherTextSizeProc plugin.Symbol
	cipherEncryptProc           plugin.Symbol
	cipherCleanupProc           plugin.Symbol
	cipherDecryptInitProc       plugin.Symbol
	cipherDecryptProc           plugin.Symbol
)

func initThirdPartCipher(cipherPath string) (err error) {
	if dmCipherEncryptSo, err = plugin.Open(cipherPath); err != nil {
		return err
	}
	if cipherGetCountProc, err = dmCipherEncryptSo.Lookup("cipher_get_count"); err != nil {
		return err
	}
	if cipherGetInfoProc, err = dmCipherEncryptSo.Lookup("cipher_get_info"); err != nil {
		return err
	}
	if cipherEncryptInitProc, err = dmCipherEncryptSo.Lookup("cipher_encrypt_init"); err != nil {
		return err
	}
	if cipherGetCipherTextSizeProc, err = dmCipherEncryptSo.Lookup("cipher_get_cipher_text_size"); err != nil {
		return err
	}
	if cipherEncryptProc, err = dmCipherEncryptSo.Lookup("cipher_encrypt"); err != nil {
		return err
	}
	if cipherCleanupProc, err = dmCipherEncryptSo.Lookup("cipher_cleanup"); err != nil {
		return err
	}
	if cipherDecryptInitProc, err = dmCipherEncryptSo.Lookup("cipher_decrypt_init"); err != nil {
		return err
	}
	if cipherDecryptProc, err = dmCipherEncryptSo.Lookup("cipher_decrypt"); err != nil {
		return err
	}
	return nil
}

func cipherGetCount() int {
	ret := cipherGetCountProc.(func() interface{})()
	return ret.(int)
}

func cipherGetInfo(seqno, cipherId, cipherName, _type, blkSize, khSIze uintptr) {
	ret := cipherGetInfoProc.(func(uintptr, uintptr, uintptr, uintptr, uintptr, uintptr) interface{})(seqno, cipherId, cipherName, _type, blkSize, khSIze)
	if ret.(int) == 0 {
		panic("ThirdPartyCipher: call cipher_get_info failed")
	}
}

func cipherEncryptInit(cipherId, key, keySize, cipherPara uintptr) {
	ret := cipherEncryptInitProc.(func(uintptr, uintptr, uintptr, uintptr) interface{})(cipherId, key, keySize, cipherPara)
	if ret.(int) == 0 {
		panic("ThirdPartyCipher: call cipher_encrypt_init failed")
	}
}

func cipherGetCipherTextSize(cipherId, cipherPara, plainTextSize uintptr) uintptr {
	ciphertextLen := cipherGetCipherTextSizeProc.(func(uintptr, uintptr, uintptr) interface{})(cipherId, cipherPara, plainTextSize)
	return ciphertextLen.(uintptr)
}

func cipherEncrypt(cipherId, cipherPara, plainText, plainTextSize, cipherText, cipherTextBufSize uintptr) uintptr {
	ret := cipherEncryptProc.(func(uintptr, uintptr, uintptr, uintptr, uintptr, uintptr) interface{})(cipherId, cipherPara, plainText, plainTextSize, cipherText, cipherTextBufSize)
	return ret.(uintptr)
}

func cipherClean(cipherId, cipherPara uintptr) {
	cipherEncryptProc.(func(uintptr, uintptr))(cipherId, cipherPara)
}

func cipherDecryptInit(cipherId, key, keySize, cipherPara uintptr) {
	ret := cipherDecryptInitProc.(func(uintptr, uintptr, uintptr, uintptr) interface{})(cipherId, key, keySize, cipherPara)
	if ret.(int) == 0 {
		panic("ThirdPartyCipher: call cipher_decrypt_init failed")
	}
}

func cipherDecrypt(cipherId, cipherPara, cipherText, cipherTextSize, plainText, plainTextBufSize uintptr) uintptr {
	ret := cipherDecryptProc.(func(uintptr, uintptr, uintptr, uintptr, uintptr, uintptr) interface{})(cipherId, cipherPara, cipherText, cipherTextSize, plainText, plainTextBufSize)
	return ret.(uintptr)
}
