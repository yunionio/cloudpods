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

package huawei

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aokoli/goutils"
	"golang.org/x/crypto/ssh"

	"yunion.io/x/pkg/errors"
)

type SKeypair struct {
	Keypair struct {
		Fingerprint string `json:"fingerprint"`
		Name        string `json:"name"`
		PublicKey   string `json:"public_key"`
	}
}

func (self *SRegion) getFingerprint(publicKey string) (string, error) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("publicKey error %s", err)
	}

	fingerprint := strings.Replace(ssh.FingerprintLegacyMD5(pk), ":", "", -1)
	return fingerprint, nil
}

func (self *SRegion) GetKeypairs() ([]SKeypair, error) {
	ret := []SKeypair{}
	query := url.Values{}
	resp, err := self.list(SERVICE_ECS, "os-keypairs", query)
	if err != nil {
		return nil, errors.Wrapf(err, "list os-keypairs")
	}
	err = resp.Unmarshal(&ret, "keypairs")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) lookUpKeypair(publicKey string) (string, error) {
	keypairs, err := self.GetKeypairs()
	if err != nil {
		return "", err
	}

	fingerprint, err := self.getFingerprint(publicKey)
	if err != nil {
		return "", err
	}

	for _, keypair := range keypairs {
		if keypair.Keypair.Fingerprint == fingerprint {
			return keypair.Keypair.Name, nil
		}
	}

	return "", fmt.Errorf("keypair not found %s", err)
}

// https://console.huaweicloud.com/apiexplorer/#/openapi/ECS/doc?api=NovaCreateKeypair
func (self *SRegion) ImportKeypair(name, publicKey string) (*SKeypair, error) {
	params := map[string]interface{}{
		"name":       name,
		"public_key": publicKey,
	}
	resp, err := self.post(SERVICE_ECS, "os-keypairs", map[string]interface{}{"keypair": params})
	if err != nil {
		return nil, errors.Wrapf(err, "create os-keypairs")
	}
	ret := &SKeypair{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SRegion) importKeypair(publicKey string) (string, error) {
	prefix, e := goutils.RandomAlphabetic(6)
	if e != nil {
		return "", fmt.Errorf("publicKey error %s", e)
	}

	name := prefix + strconv.FormatInt(time.Now().Unix(), 10)
	if k, e := self.ImportKeypair(name, publicKey); e != nil {
		return "", fmt.Errorf("keypair import error %s", e)
	} else {
		return k.Keypair.Name, nil
	}
}

func (self *SRegion) syncKeypair(publicKey string) (string, error) {
	name, e := self.lookUpKeypair(publicKey)
	if e == nil {
		return name, nil
	}
	return self.importKeypair(publicKey)
}
