// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
)

// https://stackoverflow.com/questions/23897809/different-results-in-go-and-pycrypto-when-using-aes-cfb
// CFB stream with 8 bit segment size
// See http://csrc.nist.gov/publications/nistpubs/800-38a/sp800-38a.pdf
type cfb8 struct {
	b         cipher.Block
	blockSize int
	in        []byte
	out       []byte

	decrypt bool
}

func (x *cfb8) XORKeyStream(dst, src []byte) {
	for i := range src {
		x.b.Encrypt(x.out, x.in)
		copy(x.in[:x.blockSize-1], x.in[1:])
		if x.decrypt {
			x.in[x.blockSize-1] = src[i]
		}
		dst[i] = src[i] ^ x.out[0]
		if !x.decrypt {
			x.in[x.blockSize-1] = dst[i]
		}
	}
}

// NewCFB8Encrypter returns a Stream which encrypts with cipher feedback mode
// (segment size = 8), using the given Block. The iv must be the same length as
// the Block's block size.
func newCFB8Encrypter(block cipher.Block, iv []byte) cipher.Stream {
	return newCFB8(block, iv, false)
}

// NewCFB8Decrypter returns a Stream which decrypts with cipher feedback mode
// (segment size = 8), using the given Block. The iv must be the same length as
// the Block's block size.
func newCFB8Decrypter(block cipher.Block, iv []byte) cipher.Stream {
	return newCFB8(block, iv, true)
}

func newCFB8(block cipher.Block, iv []byte, decrypt bool) cipher.Stream {
	blockSize := block.BlockSize()
	if len(iv) != blockSize {
		// stack trace will indicate whether it was de or encryption
		panic("cipher.newCFB: IV length must equal block size")
	}
	x := &cfb8{
		b:         block,
		blockSize: blockSize,
		out:       make([]byte, blockSize),
		in:        make([]byte, blockSize),
		decrypt:   decrypt,
	}
	copy(x.in, iv)

	return x
}

func toAESKey(key string) []byte {
	k := []byte(key)
	if len(k) > 32 {
		return k[0:32]
	} else {
		for len(k) < 32 {
			k = append(k, '$')
		}
		return k
	}
}

func descryptAES(k, secret []byte) ([]byte, error) {
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	if len(secret) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	iv := secret[:aes.BlockSize]
	ciphertext := secret[aes.BlockSize:]
	stream := newCFB8Decrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}

func encryptAES(k, msg []byte) ([]byte, error) {
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	cipherText := make([]byte, aes.BlockSize+len(msg))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	stream := newCFB8Encrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], msg)
	return cipherText, nil
}

func DescryptAESBase64(key, secret string) (string, error) {
	s, e := base64.StdEncoding.DecodeString(secret)
	if e != nil {
		return "", e
	}
	k := toAESKey(key)
	result, e := descryptAES(k, s)
	if e != nil {
		return "", e
	}
	return string(result), nil
}

func DescryptAESBase64Url(key, secret string) (string, error) {
	s, e := base64.URLEncoding.DecodeString(secret)
	if e != nil {
		return "", e
	}
	k := toAESKey(key)
	result, e := descryptAES(k, s)
	if e != nil {
		return "", e
	}
	return string(result), nil
}

func EncryptAESBase64(key, msg string) (string, error) {
	k := toAESKey(key)
	result, err := encryptAES(k, []byte(msg))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(result), nil
}

func EncryptAESBase64Url(key, msg string) (string, error) {
	k := toAESKey(key)
	result, err := encryptAES(k, []byte(msg))
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(result), nil
}

// RSA 加密
func rsaEncrypt(publicKey, origData []byte) ([]byte, error) {
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, fmt.Errorf("public key error")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub := pubInterface.(*rsa.PublicKey)
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData)
}

// Base64封装的 RSA 密文解密
func RsaEncryptBase64(publicKey []byte, origData string) (string, error) {
	data := []byte(origData)
	s, e := rsaEncrypt(publicKey, data)
	if e != nil {
		return "", e
	}

	result := base64.StdEncoding.EncodeToString(s)
	return string(result), nil
}

// RSA 解密
func rsaDecrypt(privateKey, secret []byte) ([]byte, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, fmt.Errorf("private key error!")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, priv, secret)
}

// Base64封装的 RSA 密文解密
func RsaDecryptBase64(privateKey []byte, secret string) (string, error) {
	s, e := base64.StdEncoding.DecodeString(secret)
	if e != nil {
		return "", e
	}

	result, e := rsaDecrypt(privateKey, s)
	if e != nil {
		return "", e
	}

	return string(result), nil
}

// RSA 签名
func RsaSign(privateKey []byte, message string) (string, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return "", fmt.Errorf("private key error!")
	}
	priv, e := x509.ParsePKCS1PrivateKey(block.Bytes)
	if e != nil {
		return "", e
	}

	h := sha256.New()
	h.Write([]byte(message))
	d := h.Sum(nil)
	sign, e := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, d)
	return string(sign), e
}

// RSA 签名验证
func RsaUnsign(publicKey []byte, message, sign string) error {
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return fmt.Errorf("private key error!")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return err
	}
	pub := pubInterface.(*rsa.PublicKey)

	h := sha256.New()
	h.Write([]byte(message))
	d := h.Sum(nil)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, d, []byte(sign))
}
