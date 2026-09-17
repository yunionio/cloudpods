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

package samlutils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"hash"
	"io/ioutil"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/pkg/util/seclib"
)

func (saml *SSAMLInstance) parseKeys() error {
	privData, err := ioutil.ReadFile(saml.privateKeyFile)
	if err != nil {
		return errors.Wrapf(err, "ioutil.ReadFile %s", saml.privateKeyFile)
	}
	saml.privateKey, err = seclib.DecodePrivateKey(privData)
	if err != nil {
		return errors.Wrap(err, "decodePrivateKey")
	}

	certData, err := ioutil.ReadFile(saml.certFile)
	if err != nil {
		return errors.Wrapf(err, "ioutil.Readfile %s", saml.certFile)
	}

	var block *pem.Block
	saml.certs = make([]*x509.Certificate, 0)
	first := true
	for {
		block, certData = pem.Decode(certData)
		if block == nil {
			break
		}
		if first {
			first = false
			saml.certString = seclib.CleanCertificate(string(pem.EncodeToMemory(block)))
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return errors.Wrap(err, "x509.ParseCertificate")
		}
		saml.certs = append(saml.certs, cert)
	}

	return nil
}

func (key EncryptedKey) decryptKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	cipher, err := base64.StdEncoding.DecodeString(key.CipherData.CipherValue.Value)
	if err != nil {
		return nil, errors.Wrap(err, "base64.StdEncoding.DecodeString")
	}
	encAlg := key.EncryptionMethod.Algorithm
	switch encAlg {
	case "http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p":
		if key.EncryptionMethod.DigestMethod == nil {
			return nil, errors.Wrap(errors.ErrInvalidFormat, "missing DigestMethod")
		}
		var shaAlg hash.Hash
		hashAlg := key.EncryptionMethod.DigestMethod.Algorithm
		switch hashAlg {
		case "http://www.w3.org/2000/09/xmldsig#sha1":
			shaAlg = sha1.New()
		default:
			return nil, errors.Wrapf(errors.ErrUnsupportedProtocol, "unsupported digest algorithm %s", hashAlg)
		}
		plaintext, err := rsa.DecryptOAEP(shaAlg, rand.Reader, privateKey, cipher, nil)
		if err != nil {
			return nil, errors.Wrap(err, "rsa.DecryptOAEP")
		}
		return plaintext, nil
	default:
		return nil, errors.Wrapf(errors.ErrUnsupportedProtocol, "unsupported encryption algorithm %s", encAlg)
	}
}

func (data EncryptedData) decryptData(privateKey *rsa.PrivateKey) ([]byte, error) {
	cipher, err := base64.StdEncoding.DecodeString(data.CipherData.CipherValue.Value)
	if err != nil {
		return nil, errors.Wrap(err, "base64.StdEncoding.DecodeString")
	}
	if data.KeyInfo.EncryptedKey == nil {
		return nil, errors.Wrap(errors.ErrInvalidFormat, "missing KeyInfo.EncryptedKey")
	}
	key, err := data.KeyInfo.EncryptedKey.decryptKey(privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "KeyInfo.EncryptedKey.decryptKey")
	}
	encAlg := data.EncryptionMethod.Algorithm
	switch encAlg {
	case "http://www.w3.org/2001/04/xmlenc#aes128-cbc", "http://www.w3.org/2001/04/xmlenc#aes192-cbc", "http://www.w3.org/2001/04/xmlenc#aes256-cbc":
		return decryptAesCbc(key, cipher)
	default:
		return nil, errors.Wrapf(errors.ErrUnsupportedProtocol, "unsupported encryption algorithm %s", encAlg)
	}
}

// stripPKCS7Padding removes the padding that XML Encryption appends to the
// last block of a CBC payload.
//
// Data that does not carry valid padding is returned unchanged rather than
// reported, so that this cannot be used to tell one payload from another.
func stripPKCS7Padding(data []byte, blockSize int) []byte {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return data
	}
	n := int(data[len(data)-1])
	if n == 0 || n > blockSize || n > len(data) {
		return data
	}
	for _, b := range data[len(data)-n:] {
		if int(b) != n {
			return data
		}
	}
	return data[:len(data)-n]
}

func decryptAesCbc(key []byte, secret []byte) ([]byte, error) {
	// The payload is an IV followed by whole ciphertext blocks. Anything
	// shorter, or not a whole number of blocks, cannot be decrypted.
	if len(secret) < 2*aes.BlockSize || len(secret)%aes.BlockSize != 0 {
		return nil, errors.Wrapf(errors.ErrInvalidFormat,
			"ciphertext of %d bytes is not a whole number of %d byte blocks preceded by an IV",
			len(secret), aes.BlockSize)
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "aes.NewCipher")
	}

	decrypter := cipher.NewCBCDecrypter(c, secret[0:aes.BlockSize])

	data := make([]byte, len(secret)-aes.BlockSize)
	copy(data, secret[aes.BlockSize:])

	decrypter.CryptBlocks(data, data)

	return stripPKCS7Padding(data, aes.BlockSize), nil
}
