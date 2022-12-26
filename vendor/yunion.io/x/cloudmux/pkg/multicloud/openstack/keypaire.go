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

package openstack

import (
	"fmt"
	"net/url"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rand"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SKeypair struct {
	Fingerprint string
	Name        string
	Type        string
	PublicKey   string
}

type SKeyPair struct {
	Keypair SKeypair
}

func (region *SRegion) GetKeypairs() ([]SKeyPair, error) {
	keypairs := []SKeyPair{}
	resource := "/os-keypairs"
	query := url.Values{}
	for {
		resp, err := region.ecsList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "ecsList")
		}
		part := struct {
			Keypairs      []SKeyPair
			KeypairsLinks SNextLinks
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		keypairs = append(keypairs, part.Keypairs...)
		marker := part.KeypairsLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return keypairs, nil
}

func (region *SRegion) CreateKeypair(name, publicKey, Type string) (*SKeyPair, error) {
	params := map[string]map[string]string{
		"keypair": {
			"name":       name,
			"public_key": publicKey,
		},
	}
	if len(Type) > 0 {
		params["keypair"]["type"] = Type
	}
	resp, err := region.ecsPost("/os-keypairs", params)
	if err != nil {
		return nil, errors.Wrap(err, "ecsPost")
	}
	keypair := &SKeyPair{}
	err = resp.Unmarshal(keypair)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return keypair, nil
}

func (region *SRegion) DeleteKeypair(name string) error {
	_, err := region.ecsDelete("/os-keypairs/" + name)
	return err
}

func (region *SRegion) GetKeypair(name string) (*SKeyPair, error) {
	resp, err := region.ecsGet("/os-keypairs/" + name)
	if err != nil {
		return nil, errors.Wrap(err, "ecsGet")
	}
	keypair := &SKeyPair{}
	err = resp.Unmarshal(keypair)
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return keypair, nil
}

func (region *SRegion) syncKeypair(namePrefix, publicKey string) (string, error) {
	keypairs, err := region.GetKeypairs()
	if err != nil {
		return "", err
	}

	for _, keypair := range keypairs {
		if keypair.Keypair.PublicKey == publicKey {
			return keypair.Keypair.Name, nil
		}
	}
	randomString := func(prefix string, length int) string {
		return fmt.Sprintf("%s-%s", prefix, rand.String(length))
	}
	for i := 1; i < 10; i++ {
		name := randomString(namePrefix, i)
		if _, err := region.GetKeypair(name); err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				keypair, err := region.CreateKeypair(name, publicKey, "ssh")
				if err != nil {
					return "", errors.Wrapf(err, "CreateKeypair")
				}
				return keypair.Keypair.Name, nil
			}
		}
	}
	return "", fmt.Errorf("failed to find uniq name for keypair")
}
