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

package ctyun

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aokoli/goutils"
	"golang.org/x/crypto/ssh"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

// GET http://ctyun-api-url/apiproxy/v3/querySSH
// POST http://ctyun-api-url/apiproxy/v3/deleteSSH
type SKeypair struct {
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
}

func (self *SRegion) getFingerprint(publicKey string) (string, error) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("publicKey error %s", err)
	}

	return ssh.FingerprintSHA256(pk), nil
}

// GET http://ctyun-api-url/apiproxy/v3/querySSH
func (self *SRegion) GetKeypairs() ([]SKeypair, int, error) {
	params := map[string]string{
		"regionId": self.GetId(),
	}

	resp, err := self.client.DoGet("/apiproxy/v3/querySSH", params)
	if err != nil {
		return nil, 0, errors.Wrap(err, "SRegion.GetKeypairs.DoGet")
	}

	keypairs := []jsonutils.JSONObject{}
	err = resp.Unmarshal(&keypairs, "returnObj", "keypairs")
	if err != nil {
		return nil, 0, errors.Wrap(err, "SRegion.GetKeypairs.Unmarshal")
	}

	ret := []SKeypair{}
	for i := range keypairs {
		k, err := keypairs[i].Get("keypair")
		if err != nil {
			return nil, 0, errors.Wrap(err, "SRegion.GetKeypairs")
		}

		keypair := SKeypair{}
		err = k.Unmarshal(&keypair)
		if err != nil {
			return nil, 0, errors.Wrap(err, "SRegion.GetKeypairs.Unmarshal")
		}

		ret = append(ret, keypair)
	}

	return ret, len(ret), nil
}

func (self *SRegion) GetKeypair(name string) (*SKeypair, error) {
	keypairs, _, err := self.GetKeypairs()
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetKeypair.GetKeypairs")
	}

	for i := range keypairs {
		if keypairs[i].Name == name {
			return &keypairs[i], nil
		}
	}

	return nil, errors.Wrap(errors.ErrNotFound, "SRegion.GetKeypair")
}

func (self *SRegion) lookUpKeypair(publicKey string) (string, error) {
	keypairs, _, err := self.GetKeypairs()
	if err != nil {
		return "", err
	}

	fingerprint, err := self.getFingerprint(publicKey)
	if err != nil {
		return "", err
	}

	for _, keypair := range keypairs {
		if keypair.Fingerprint == fingerprint {
			return keypair.Name, nil
		}
	}

	return "", fmt.Errorf("keypair not found %s", err)
}

// POST http://ctyun-api-url/apiproxy/v3/createSSH
func (self *SRegion) ImportKeypair(name, publicKey string) (*SKeypair, error) {
	params := map[string]jsonutils.JSONObject{
		"regionId":  jsonutils.NewString(self.GetId()),
		"name":      jsonutils.NewString(name),
		"publicKey": jsonutils.NewString(publicKey),
	}

	_, err := self.client.DoPost("/apiproxy/v3/createSSH", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.ImportKeypair.DoPost")
	}

	return self.GetKeypair(name)
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
		return k.Name, nil
	}
}

func (self *SRegion) syncKeypair(publicKey string) (string, error) {
	name, e := self.lookUpKeypair(publicKey)
	if e == nil {
		return name, nil
	}
	return self.importKeypair(publicKey)
}
