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

package seclib

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"regexp"
	"strings"

	"yunion.io/x/pkg/errors"
)

func CleanCertificate(cert string) string {
	re := regexp.MustCompile("---(.*)CERTIFICATE(.*)---")
	cert = re.ReplaceAllString(cert, "")
	cert = strings.Trim(cert, " \n")
	// cert = strings.Replace(cert, "\n", "", -1)
	return cert
}

func DecodePrivateKey(keyString []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyString)
	privKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		return privKey.(*rsa.PrivateKey), nil
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return priv, nil
	}
	return nil, errors.Wrap(errors.ErrInvalidFormat, "not a valid private key")
}
