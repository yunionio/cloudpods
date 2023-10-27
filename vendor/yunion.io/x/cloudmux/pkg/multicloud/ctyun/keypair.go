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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type SKeypair struct {
	Fingerprint string
	KeyPairName string
	KeyPairId   string
	PublicKey   string
}

func (self *SRegion) GetKeypairs(name string) ([]SKeypair, error) {
	pageNo := 1
	params := map[string]interface{}{
		"pageNo":   pageNo,
		"pageSize": 50,
	}
	if len(name) > 0 {
		params["queryContent"] = name
	}
	ret := []SKeypair{}
	for {
		resp, err := self.post(SERVICE_ECS, "/v4/ecs/keypair/details", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			ReturnObj struct {
				Results []SKeypair
			}
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		ret = append(ret, part.ReturnObj.Results...)
		if len(ret) >= part.TotalCount || len(part.ReturnObj.Results) == 0 {
			break
		}
		pageNo++
		params["pageNo"] = pageNo
	}
	return ret, nil
}

func (self *SRegion) lookUpKeypair(publicKey string) (*SKeypair, error) {
	keypairs, err := self.GetKeypairs("")
	if err != nil {
		return nil, err
	}
	for i := range keypairs {
		if keypairs[i].PublicKey == publicKey {
			return &keypairs[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "by public key")
}

func (self *SRegion) ImportKeypair(name, publicKey string) (*SKeypair, error) {
	params := map[string]interface{}{
		"keyPairName": name,
		"publicKey":   publicKey,
	}
	_, err := self.post(SERVICE_ECS, "/v4/ecs/keypair/import-keypair", params)
	if err != nil {
		return nil, err
	}
	keypairs, err := self.GetKeypairs(name)
	if err != nil {
		return nil, err
	}
	for i := range keypairs {
		if keypairs[i].KeyPairName == name {
			return &keypairs[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after create %s", name)
}

func (self *SRegion) importKeypair(publicKey string) (*SKeypair, error) {
	prefix, err := goutils.RandomAlphabetic(6)
	if err != nil {
		return nil, fmt.Errorf("publicKey error %s", err)
	}

	name := prefix + strconv.FormatInt(time.Now().Unix(), 10)
	return self.ImportKeypair(name, publicKey)
}

func (self *SRegion) syncKeypair(publicKey string) (*SKeypair, error) {
	key, err := self.lookUpKeypair(publicKey)
	if err == nil {
		return key, nil
	}
	return self.importKeypair(publicKey)
}
