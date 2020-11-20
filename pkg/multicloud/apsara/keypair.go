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

package apsara

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aokoli/goutils"
	"golang.org/x/crypto/ssh"

	"yunion.io/x/log"
)

type SKeypair struct {
	KeyPairFingerPrint string
	KeyPairName        string
}

func (self *SRegion) GetKeypairs(finger string, name string, offset int, limit int) ([]SKeypair, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)
	if len(finger) > 0 {
		params["KeyPairFingerPrint"] = finger
	}
	if len(name) > 0 {
		params["KeyPairName"] = name
	}

	body, err := self.ecsRequest("DescribeKeyPairs", params)
	if err != nil {
		log.Errorf("GetKeypairs fail %s", err)
		return nil, 0, err
	}

	keypairs := make([]SKeypair, 0)
	err = body.Unmarshal(&keypairs, "KeyPairs", "KeyPair")
	if err != nil {
		log.Errorf("Unmarshal keypair fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return keypairs, int(total), nil
}

func (self *SRegion) ImportKeypair(name string, pubKey string) (*SKeypair, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PublicKeyBody"] = pubKey
	params["KeyPairName"] = name

	body, err := self.ecsRequest("ImportKeyPair", params)
	if err != nil {
		log.Errorf("ImportKeypair fail %s", err)
		return nil, err
	}

	log.Debugf("%s", body)
	keypair := SKeypair{}
	err = body.Unmarshal(&keypair)
	if err != nil {
		log.Errorf("Unmarshall keypair fail %s", err)
		return nil, err
	}
	return &keypair, nil
}

func (self *SRegion) AttachKeypair(instanceId string, name string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["KeyPairName"] = name
	instances, _ := json.Marshal(&[...]string{instanceId})
	params["InstanceIds"] = string(instances)
	_, err := self.ecsRequest("AttachKeyPair", params)
	if err != nil {
		log.Errorf("AttachKeyPair fail %s", err)
		return err
	}

	return nil
}

func (self *SRegion) DetachKeyPair(instanceId string, name string) error {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["KeyPairName"] = name
	instances, _ := json.Marshal(&[...]string{instanceId})
	params["InstanceIds"] = string(instances)
	_, err := self.ecsRequest("DetachKeyPair", params)
	if err != nil {
		log.Errorf("DetachKeyPair fail %s", err)
		return err
	}

	return nil
}

func (self *SRegion) lookUpApsaraKeypair(publicKey string) (string, error) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("publicKey error %s", err)
	}

	fingerprint := strings.Replace(ssh.FingerprintLegacyMD5(pk), ":", "", -1)
	ks, total, err := self.GetKeypairs(fingerprint, "*", 0, 1)
	if total < 1 {
		return "", fmt.Errorf("keypair not found %s", err)
	} else {
		return ks[0].KeyPairName, nil
	}
}

func (self *SRegion) importApsaraKeypair(publicKey string) (string, error) {
	prefix, e := goutils.RandomAlphabetic(6)
	if e != nil {
		return "", fmt.Errorf("publicKey error %s", e)
	}

	name := prefix + strconv.FormatInt(time.Now().Unix(), 10)
	if k, e := self.ImportKeypair(name, publicKey); e != nil {
		return "", fmt.Errorf("keypair import error %s", e)
	} else {
		return k.KeyPairName, nil
	}
}

func (self *SRegion) syncKeypair(publicKey string) (string, error) {
	name, e := self.lookUpApsaraKeypair(publicKey)
	if e == nil {
		return name, nil
	}
	return self.importApsaraKeypair(publicKey)
}
