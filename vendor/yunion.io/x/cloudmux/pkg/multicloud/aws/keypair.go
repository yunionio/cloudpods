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

package aws

import (
	"bytes"
	"crypto/md5"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/aokoli/goutils"
	"golang.org/x/crypto/ssh"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type SKeypair struct {
	KeyPairFingerPrint string    `xml:"keyFingerprint"`
	KeyName            string    `xml:"keyName"`
	KeyPairId          string    `xml:"keyPairId"`
	KeyType            string    `xml:"keyType"`
	CreateTime         time.Time `xml:"createTime"`
	PublicKey          string    `xml:"publicKey"`
}

// 只支持计算Openssh ras 格式公钥转换成DER格式后的MD5。
func md5Fingerprint(publickey string) (string, error) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publickey))
	if err != nil {
		return "", fmt.Errorf("publicKey error %s", err)
	}

	der := []byte{}
	cryptoPub, ok := pk.(ssh.CryptoPublicKey)
	if !ok {
		return "", fmt.Errorf("public key trans to crypto public key failed")
	}

	switch pk.Type() {
	case ssh.KeyAlgoRSA:
		pubKey := cryptoPub.CryptoPublicKey()
		rsaPK, ok := pubKey.(*rsa.PublicKey)
		if !ok {
			return "", fmt.Errorf("crypto public key trans to ras publickey failed")
		}
		der, err = x509.MarshalPKIXPublicKey(rsaPK)
		if err != nil {
			return "", fmt.Errorf("MarshalPKIXPublicKey ras publickey failed")
		}
	default:
		return "", fmt.Errorf("unsupport public key format.Only ssh-rsa supported")
	}

	var ret bytes.Buffer
	fp := md5.Sum(der)
	for i, b := range fp {
		ret.WriteString(fmt.Sprintf("%02x", b))
		if i < len(fp)-1 {
			ret.WriteString(":")
		}
	}

	return ret.String(), nil
}

func (self *SRegion) GetKeypairs(finger string, name string) ([]SKeypair, error) {
	params := map[string]string{
		"IncludePublicKey": "true",
	}
	if len(finger) > 0 {
		params["Filter.1.Name"] = "fingerprint"
		params["Filter.1.Value.1"] = finger
	}

	if len(name) > 0 {
		params["KeyName"] = name
	}

	ret := struct {
		KeySet []SKeypair `xml:"keySet>item"`
	}{}
	err := self.ec2Request("DescribeKeyPairs", params, &ret)
	return ret.KeySet, err
}

// Aws貌似不支持ssh-dss格式密钥
func (self *SRegion) ImportKeypair(name string, pubKey string) (*SKeypair, error) {
	params := map[string]string{
		"KeyName":           name,
		"PublicKeyMaterial": base64.StdEncoding.EncodeToString([]byte(pubKey)),
	}
	ret := &SKeypair{}
	return ret, self.ec2Request("ImportKeyPair", params, ret)
}

func (self *SRegion) AttachKeypair(instanceId string, keypairName string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) DetachKeyPair(instanceId string, keypairName string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) lookUpAwsKeypair(publicKey string) (string, error) {
	// https://docs.amazonaws.cn/AWSEC2/latest/UserGuide/ec2-key-pairs.html
	fingerprint, err := md5Fingerprint(publicKey)
	if err != nil {
		return "", err
	}

	keyparis, err := self.GetKeypairs(fingerprint, "")
	if err != nil {
		return "", errors.Wrapf(err, "GetKeypairs")
	}
	if len(keyparis) > 0 {
		return keyparis[0].KeyName, nil
	}
	return "", errors.Wrapf(cloudprovider.ErrNotFound, publicKey)
}

func (self *SRegion) importAwsKeypair(publicKey string) (string, error) {
	prefix, e := goutils.RandomAlphabetic(6)
	if e != nil {
		return "", fmt.Errorf("publicKey error %s", e)
	}

	name := prefix + strconv.FormatInt(time.Now().Unix(), 10)
	if k, e := self.ImportKeypair(name, publicKey); e != nil {
		return "", fmt.Errorf("keypair import error %s", e)
	} else {
		return k.KeyName, nil
	}
}

func (self *SRegion) SyncKeypair(publicKey string) (string, error) {
	name, e := self.lookUpAwsKeypair(publicKey)
	if e == nil {
		return name, nil
	}
	return self.importAwsKeypair(publicKey)
}
