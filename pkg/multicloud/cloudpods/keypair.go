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

package cloudpods

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SKeypair struct {
	api.KeypairDetails
}

func (self *SRegion) GetKeypairs() ([]SKeypair, error) {
	keypairs := []SKeypair{}
	return keypairs, self.list(&modules.Keypairs, nil, &keypairs)
}

func (self *SRegion) CreateKeypair(name, publicKey string) (*SKeypair, error) {
	input := api.KeypairCreateInput{}
	input.GenerateName = fmt.Sprintf("keypair-for-%s", name)
	input.PublicKey = publicKey
	keypair := &SKeypair{}
	return keypair, self.create(&modules.Keypairs, input, keypair)
}

func (self *SRegion) syncKeypair(serverName, publicKey string) (string, error) {
	keypairs, err := self.GetKeypairs()
	if err != nil {
		return "", err
	}
	for _, keypair := range keypairs {
		if keypair.PublicKey == publicKey {
			return keypair.Id, nil
		}
	}
	keypair, err := self.CreateKeypair(serverName, publicKey)
	if err != nil {
		return "", errors.Wrapf(err, "CreateKeypair")
	}
	return keypair.Id, nil
}
