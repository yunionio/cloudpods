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
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"strings"

	"yunion.io/x/pkg/util/seclib"
)

const (
	certBeginString = "BEGIN CERTIFICATE"
)

// MergeCaCertFiles concatenates cert and ca file to form a chain, write it to
// a tmpfile then return the path
//
// Callers are responsible for removing the returned tmpfile
func MergeCaCertFiles(cafile string, certfile string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "cerfile.*.crt")
	if err != nil {
		return "", fmt.Errorf("fail to open tempfile for ca cerfile: %s", err)
	}
	defer tmpfile.Close()

	cont, err := ioutil.ReadFile(certfile)
	if err != nil {
		return "", fmt.Errorf("fail to read certfile %s", err)
	}
	offset := strings.Index(string(cont), certBeginString)
	if offset < 0 {
		return "", fmt.Errorf("invalid certfile, no BEGIN CERTIFICATE found")
	}
	for offset > 0 && cont[offset-1] == '-' {
		offset -= 1
	}
	tmpfile.Write(cont[offset:])
	cont, err = ioutil.ReadFile(cafile)
	if err != nil {
		return "", fmt.Errorf("fail to read cafile %s", err)
	}
	tmpfile.Write(cont)

	return tmpfile.Name(), nil
}

func CleanCertificate(cert string) string {
	return seclib.CleanCertificate(cert)
}

func DecodePrivateKey(keyString []byte) (*rsa.PrivateKey, error) {
	return seclib.DecodePrivateKey(keyString)
}
