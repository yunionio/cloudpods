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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
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

func toAESKey(k []byte) []byte {
	if len(k) > 32 {
		return k[0:32]
	} else {
		for len(k) < 32 {
			k = append(k, '$')
		}
		return k
	}
}

func decryptAES(k, secret []byte) ([]byte, error) {
	block, err := aes.NewCipher(toAESKey(k))
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
	block, err := aes.NewCipher(toAESKey(k))
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
