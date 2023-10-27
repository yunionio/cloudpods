// Copyright 2023 Yunion
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

package volcengine

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aokoli/goutils"
	"golang.org/x/crypto/ssh"

	"yunion.io/x/pkg/errors"
)

type SKeypair struct {
	KeyPairFingerPrint string
	KeyPairName        string
}

func (region *SRegion) GetKeypairs(finger string, name string, limit int, token string) ([]SKeypair, string, error) {
	if limit > 500 || limit <= 0 {
		limit = 500
	}
	params := make(map[string]string)
	params["MaxResults"] = fmt.Sprintf("%d", limit)
	if len(token) > 0 {
		params["NextToken"] = token
	}
	if len(finger) > 0 {
		params["FingerPrint"] = finger
	}
	if len(name) > 0 {
		params["KeyPairName"] = name
	}

	body, err := region.ecsRequest("DescribeKeyPairs", params)
	if err != nil {
		return nil, "", errors.Wrapf(err, "GetKeypairs fail")
	}

	keypairs := make([]SKeypair, 0)
	err = body.Unmarshal(&keypairs, "KeyPairs")
	if err != nil {
		return nil, "", errors.Wrapf(err, "Unmarshal keypair fail")
	}
	nextToken, _ := body.GetString("NextToken")
	return keypairs, nextToken, nil
}

func (region *SRegion) ImportKeypair(name string, pubKey string) (*SKeypair, error) {
	params := make(map[string]string)
	params["PublicKey"] = pubKey
	params["KeyPairName"] = name

	body, err := region.ecsRequest("ImportKeyPair", params)
	if err != nil {
		return nil, errors.Wrapf(err, "ImportKeypair fail")
	}

	keypair := SKeypair{}
	err = body.Unmarshal(&keypair)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal keypair fail")
	}
	return &keypair, nil
}

func (region *SRegion) AttachKeypair(instanceId string, name string) error {
	params := make(map[string]string)
	params["KeyPairName"] = name
	params["InstanceIds.1"] = instanceId
	_, err := region.ecsRequest("AttachKeyPair", params)
	if err != nil {
		return errors.Wrapf(err, "AttachKeyPair fail")
	}

	return nil
}

func (region *SRegion) DetachKeyPair(instanceId string, name string) error {
	params := make(map[string]string)
	params["KeyPairName"] = name
	params["InstanceIds.1"] = instanceId
	_, err := region.ecsRequest("DetachKeyPair", params)
	if err != nil {
		return errors.Wrapf(err, "DetachKeyPair fail")
	}

	return nil
}

func (region *SRegion) lookUpVolcEngineKeypair(publicKey string) (string, error) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("publicKey error %s", err)
	}

	fingerprint := strings.Replace(ssh.FingerprintLegacyMD5(pk), ":", "", -1)
	ks, _, err := region.GetKeypairs(fingerprint, "*", 0, "")
	if len(ks) < 1 {
		return "", fmt.Errorf("keypair not found %s", err)
	} else {
		return ks[0].KeyPairName, nil
	}
}

func (region *SRegion) importVolcEngineKeypair(publicKey string) (string, error) {
	prefix, e := goutils.RandomAlphabetic(6)
	if e != nil {
		return "", fmt.Errorf("publicKey error %s", e)
	}

	name := prefix + strconv.FormatInt(time.Now().Unix(), 10)
	if k, e := region.ImportKeypair(name, publicKey); e != nil {
		return "", fmt.Errorf("keypair import error %s", e)
	} else {
		return k.KeyPairName, nil
	}
}

func (region *SRegion) syncKeypair(publicKey string) (string, error) {
	name, e := region.lookUpVolcEngineKeypair(publicKey)
	if e == nil {
		return name, nil
	}
	return region.importVolcEngineKeypair(publicKey)
}
