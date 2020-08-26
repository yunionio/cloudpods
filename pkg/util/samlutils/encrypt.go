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

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

func (saml *SSAMLInstance) parseKeys() error {
	privData, err := ioutil.ReadFile(saml.privateKeyFile)
	if err != nil {
		return errors.Wrapf(err, "ioutil.ReadFile %s", saml.privateKeyFile)
	}
	saml.privateKey, err = seclib2.DecodePrivateKey(privData)
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
			saml.certString = seclib2.CleanCertificate(string(pem.EncodeToMemory(block)))
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
		var shaAlg hash.Hash
		hashAlg := key.EncryptionMethod.DigestMethod.Algorithm
		switch hashAlg {
		case "http://www.w3.org/2000/09/xmldsig#sha1":
			shaAlg = sha1.New()
		default:
			return nil, errors.Wrapf(httperrors.ErrUnsupportedProtocol, "unsupported digest algorithm %s", hashAlg)
		}
		plaintext, err := rsa.DecryptOAEP(shaAlg, rand.Reader, privateKey, cipher, nil)
		if err != nil {
			return nil, errors.Wrap(err, "rsa.DecryptOAEP")
		}
		return plaintext, nil
	default:
		return nil, errors.Wrapf(httperrors.ErrUnsupportedProtocol, "unsupported encryption algorithm %s", encAlg)
	}
}

func (data EncryptedData) decryptData(privateKey *rsa.PrivateKey) ([]byte, error) {
	cipher, err := base64.StdEncoding.DecodeString(data.CipherData.CipherValue.Value)
	if err != nil {
		return nil, errors.Wrap(err, "base64.StdEncoding.DecodeString")
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
		return nil, errors.Wrapf(httperrors.ErrUnsupportedProtocol, "unsupported encryption algorithm %s", encAlg)
	}
}

func decryptAesCbc(key []byte, secret []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "aes.NewCipher")
	}

	decrypter := cipher.NewCBCDecrypter(c, secret[0:aes.BlockSize])

	data := make([]byte, len(secret)-aes.BlockSize)
	copy(data, secret[aes.BlockSize:])

	decrypter.CryptBlocks(data, data)

	return data, nil
}
