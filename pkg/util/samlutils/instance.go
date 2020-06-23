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

	"yunion.io/x/pkg/errors"
)

type SSAMLInstance struct {
	entityID string

	privateKeyFile string
	certFile       string
	certString     string

	privateKey *rsa.PrivateKey

	certs []*x509.Certificate
}

func NewSAMLInstance(entityID string, cert, key string) (*SSAMLInstance, error) {
	saml := SSAMLInstance{
		privateKeyFile: key,
		certFile:       cert,
		entityID:       entityID,
	}
	err := saml.parseKeys()
	if err != nil {
		return nil, errors.Wrap(err, "saml.parseKeys")
	}
	return &saml, nil
}

func (saml *SSAMLInstance) GetEntityId() string {
	return saml.entityID
}

func (saml *SSAMLInstance) GetCertString() string {
	return "\n" + saml.certString + "\n"
}

func (saml *SSAMLInstance) SignXML(xmlstr string) (string, error) {
	return SignXML(xmlstr, saml.privateKey)
}

func (saml *SSAMLInstance) SetEntityId(id string) {
	saml.entityID = id
}
