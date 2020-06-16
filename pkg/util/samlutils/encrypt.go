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
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

func decodePrivateKey(keyString []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyString)
	log.Debugf("pem.Decode privateKey data: type=%s header: %s", block.Type, block.Headers)
	privKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		return privKey.(*rsa.PrivateKey), nil
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return priv, nil
	}
	return nil, errors.Wrap(httperrors.ErrInvalidFormat, "not a valid private key")
}

func (saml *SSAMLInstance) parseKeys() error {
	privData, err := ioutil.ReadFile(saml.privateKeyFile)
	if err != nil {
		return errors.Wrapf(err, "ioutil.ReadFile %s", saml.privateKeyFile)
	}
	saml.privateKey, err = decodePrivateKey(privData)
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

			log.Debugf("cert: %s", saml.certString)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return errors.Wrap(err, "x509.ParseCertificate")
		}
		saml.certs = append(saml.certs, cert)
	}

	return nil
}
