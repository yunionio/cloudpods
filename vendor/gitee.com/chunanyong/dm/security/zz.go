/*
 * Copyright (c) 2000-2018, 达梦数据库有限公司.
 * All rights reserved.
 */

package security

type Cipher interface {
	Encrypt(plaintext []byte, genDigest bool) []byte
	Decrypt(ciphertext []byte, checkDigest bool) ([]byte, error)
}
