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

package qcloud

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aokoli/goutils"
	"golang.org/x/crypto/ssh"

	"yunion.io/x/log"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SKeypair struct {
	AssociatedInstanceIds []string
	CreateTime            time.Time
	Description           string
	KeyId                 string
	KeyName               string
	PublicKey             string
}

func (self *SRegion) GetKeypairs(name string, keyIds []string, offset int, limit int) ([]SKeypair, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := map[string]string{}
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)

	if len(keyIds) > 0 {
		for i := 0; i < len(keyIds); i++ {
			params[fmt.Sprintf("KeyIds.%d", i)] = keyIds[i]
		}
	} else {
		if len(name) > 0 {
			params["Filters.0.Name"] = "key-name"
			params["Filters.0.Values.0"] = name
		}
	}

	body, err := self.cvmRequest("DescribeKeyPairs", params, true)
	if err != nil {
		log.Errorf("GetKeypairs fail %s", err)
		return nil, 0, err
	}

	keypairs := []SKeypair{}
	err = body.Unmarshal(&keypairs, "KeyPairSet")
	if err != nil {
		log.Errorf("Unmarshal keypair fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Float("TotalCount")
	return keypairs, int(total), nil
}

func (self *SRegion) ImportKeypair(name string, pubKey string) (*SKeypair, error) {
	params := map[string]string{}
	params["PublicKey"] = pubKey
	params["ProjectId"] = "0"
	params["KeyName"] = name

	body, err := self.cvmRequest("ImportKeyPair", params, true)
	if err != nil {
		log.Errorf("ImportKeypair fail %s", err)
		return nil, err
	}

	keypairID, err := body.GetString("KeyId")
	if err != nil {
		return nil, err
	}
	keypairs, total, err := self.GetKeypairs("", []string{keypairID}, 0, 1)
	if err != nil {
		return nil, err
	}
	if total != 1 {
		return nil, cloudprovider.ErrNotFound
	}
	return &keypairs[0], nil
}

func (self *SRegion) AttachKeypair(instanceId string, keypairId string) error {
	params := map[string]string{}
	params["InstanceIds.0"] = instanceId
	params["KeyIds.0"] = keypairId
	_, err := self.cvmRequest("AssociateInstancesKeyPairs", params, true)
	return err
}

func (self *SRegion) DetachKeyPair(instanceId string, keypairId string) error {
	params := make(map[string]string)
	params["InstanceIds.0"] = instanceId
	params["KeyIds.0"] = keypairId
	_, err := self.cvmRequest("DisassociateInstancesKeyPairs", params, true)
	return err
}

func (self *SRegion) CreateKeyPair(name string) (*SKeypair, error) {
	params := make(map[string]string)
	params["KeyName"] = name
	params["ProjectId"] = "0"
	body, err := self.cvmRequest("CreateKeyPair", params, true)
	keypair := SKeypair{}
	err = body.Unmarshal(&keypair, "KeyPair")
	if err != nil {
		return nil, err
	}
	return &keypair, err
}

func (self *SRegion) getKeypairs() ([]SKeypair, error) {
	keypairs := []SKeypair{}
	for {
		parts, total, err := self.GetKeypairs("", []string{}, 0, 50)
		if err != nil {
			log.Errorf("Get keypairs fail %v", err)
			return nil, err
		}
		keypairs = append(keypairs, parts...)
		if len(keypairs) >= total {
			break
		}
	}
	return keypairs, nil
}

func (self *SRegion) getFingerprint(publicKey string) (string, error) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("publicKey error %s", err)
	}
	return ssh.FingerprintLegacyMD5(pk), nil
}

func (self *SRegion) lookUpQcloudKeypair(publicKey string) (string, error) {
	keypairs, err := self.getKeypairs()
	if err != nil {
		return "", err
	}

	localFiger, err := self.getFingerprint(publicKey)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(keypairs); i++ {
		finger, err := self.getFingerprint(keypairs[i].PublicKey)
		if err != nil {
			continue
		}
		if finger == localFiger {
			return keypairs[i].KeyId, nil
		}
	}
	return "", cloudprovider.ErrNotFound
}

func (self *SRegion) syncKeypair(publicKey string) (string, error) {
	keypairId, err := self.lookUpQcloudKeypair(publicKey)
	if err == nil {
		return keypairId, nil
	}

	prefix, e := goutils.RandomAlphabetic(6)
	if e != nil {
		return "", fmt.Errorf("publicKey error %s", e)
	}

	name := prefix + strconv.FormatInt(time.Now().Unix(), 10)
	keypair, err := self.ImportKeypair(name, publicKey)
	if err != nil {
		return "", err
	}
	return keypair.KeyId, nil
}
