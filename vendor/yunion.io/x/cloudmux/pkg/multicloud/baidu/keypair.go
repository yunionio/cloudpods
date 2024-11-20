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

package baidu

import "net/url"

type SKeypair struct {
	Name        string
	PublicKey   string
	FingerPrint string
	RegionId    string
	KeypairId   string
}

func (self *SRegion) GetKeypairs() ([]SKeypair, error) {
	params := url.Values{}
	ret := []SKeypair{}
	for {
		resp, err := self.bccList("v2/keypair", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Keypairs   []SKeypair
			NextMarker string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Keypairs...)
		if len(part.NextMarker) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return ret, nil
}

func (self *SRegion) SyncKeypair(name, publicKey string) (*SKeypair, error) {
	keypairs, err := self.GetKeypairs()
	if err != nil {
		return nil, err
	}
	for i := range keypairs {
		if keypairs[i].PublicKey == publicKey {
			return &keypairs[i], nil
		}
	}
	return self.CreateKeypair(name, publicKey)
}

func (self *SRegion) CreateKeypair(name, publicKey string) (*SKeypair, error) {
	body := map[string]interface{}{
		"name":      name,
		"publicKey": publicKey,
	}
	resp, err := self.bccPost("v2/keypair", nil, body)
	if err != nil {
		return nil, err
	}
	ret := &SKeypair{}
	err = resp.Unmarshal(ret, "keypair")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
