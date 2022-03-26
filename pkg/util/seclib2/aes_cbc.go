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

package seclib2

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/tjfoc/gmsm/sm4"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/seclib"
)

type TSymEncAlg string

const (
	SYM_ENC_ALG_AES_256 = TSymEncAlg("aes-256")
	SYM_ENC_ALG_SM4_128 = TSymEncAlg("sm4")
)

type SSymEncAlg struct {
	name      TSymEncAlg
	blockSize int
	newCipher func(key []byte) (cipher.Block, error)
	keySize   int
}

var (
	AES_256 = SSymEncAlg{
		name:      SYM_ENC_ALG_AES_256,
		blockSize: aes.BlockSize,
		newCipher: aes.NewCipher,
		keySize:   32,
	}
	SM4_128 = SSymEncAlg{
		name:      SYM_ENC_ALG_SM4_128,
		blockSize: sm4.BlockSize,
		newCipher: sm4.NewCipher,
		keySize:   16,
	}
)

func Alg(alg TSymEncAlg) SSymEncAlg {
	switch alg {
	case SYM_ENC_ALG_SM4_128:
		return SM4_128
	case SYM_ENC_ALG_AES_256:
		return AES_256
	default:
		return AES_256
	}
}

// AES-256-CBC
// key: 32bytes=256bites
func (alg SSymEncAlg) CbcEncodeIV(content []byte, encryptionKey []byte, IV []byte) ([]byte, error) {
	bPlaintext, err := pkcs7pad(content, alg.blockSize)
	if err != nil {
		return nil, errors.Wrap(err, "pkcs7pad")
	}
	block, err := alg.newCipher(alg.normalizeKey(encryptionKey))
	if err != nil {
		return nil, errors.Wrap(err, "newCipher")
	}
	if len(IV) == 0 {
		IV, _ = GenerateRandomBytes(block.BlockSize())
	}
	ciphertext := make([]byte, block.BlockSize()+len(bPlaintext))
	copy(ciphertext, IV)
	mode := cipher.NewCBCEncrypter(block, IV)
	mode.CryptBlocks(ciphertext[block.BlockSize():], bPlaintext)
	return ciphertext, nil
}

func (alg SSymEncAlg) CbcEncode(content []byte, encryptionKey []byte) ([]byte, error) {
	return alg.CbcEncodeIV(content, encryptionKey, nil)
}

func (alg SSymEncAlg) normalizeKey(encKey []byte) []byte {
	for len(encKey) < alg.keySize {
		encKey = append(encKey, '0')
	}
	if len(encKey) > alg.keySize {
		encKey = encKey[:alg.keySize]
	}
	return encKey
}

func (alg SSymEncAlg) CbcDecode(cipherText []byte, encryptionKey []byte) ([]byte, error) {
	block, err := alg.newCipher(alg.normalizeKey(encryptionKey))
	if err != nil {
		return nil, errors.Wrap(err, "newCipher")
	}
	if len(cipherText) < block.BlockSize() {
		return nil, errors.Wrap(errors.ErrInvalidStatus, "not a encrypted text")
	}
	mode := cipher.NewCBCDecrypter(block, cipherText[:block.BlockSize()])
	cipherText = cipherText[block.BlockSize():]
	mode.CryptBlocks(cipherText, cipherText)
	return pkcs7strip(cipherText, alg.blockSize)
}

func (alg SSymEncAlg) CbcEncodeBase64(content []byte, encryptionKey []byte) (string, error) {
	bSecret, err := alg.CbcEncode(content, encryptionKey)
	if err != nil {
		return "", errors.Wrap(err, "Aes256CbcEncode")
	}
	secret := base64.StdEncoding.EncodeToString(bSecret)
	return secret, nil
}

func (alg SSymEncAlg) CbcDecodeBase64(cipherText string, encryptionKey []byte) ([]byte, error) {
	bCipher, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return nil, errors.Wrap(err, "base64.StdEncoding.DecodeString cipher")
	}
	return alg.CbcDecode(bCipher, encryptionKey)
}

func (alg SSymEncAlg) GenerateKey() string {
	return seclib.RandomPassword(alg.keySize)
}

func (alg SSymEncAlg) Name() TSymEncAlg {
	return alg.name
}

// pkcs7strip remove pkcs7 padding
func pkcs7strip(data []byte, blockSize int) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.Error("pkcs7: Data is empty")
	}
	if length%blockSize != 0 {
		return nil, errors.Error("pkcs7: Data is not block-aligned")
	}
	padLen := int(data[length-1])
	ref := bytes.Repeat([]byte{byte(padLen)}, padLen)
	if padLen > blockSize || padLen == 0 || !bytes.HasSuffix(data, ref) {
		return nil, errors.Error("pkcs7: Invalid padding")
	}
	return data[:length-padLen], nil
}

// pkcs7pad add pkcs7 padding
func pkcs7pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize < 0 || blockSize > 256 {
		return nil, errors.Error(fmt.Sprintf("pkcs7: Invalid block size %d", blockSize))
	} else {
		padLen := blockSize - len(data)%blockSize
		padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
		return append(data, padding...), nil
	}
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)

	if err != nil {
		return nil, err
	}

	return b, nil
}
