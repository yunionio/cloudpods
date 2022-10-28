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
	"fmt"
	"strconv"
	"time"

	"github.com/aokoli/goutils"
	"github.com/aws/aws-sdk-go/service/ec2"
	"golang.org/x/crypto/ssh"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SKeypair struct {
	KeyPairFingerPrint string
	KeyPairName        string
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
		rsaPK, ok := cryptoPub.CryptoPublicKey().(*rsa.PublicKey)
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

func (self *SRegion) GetKeypairs(finger string, name string, offset int, limit int) ([]SKeypair, int, error) {
	params := &ec2.DescribeKeyPairsInput{}
	filters := []*ec2.Filter{}
	if len(finger) > 0 {
		filters = AppendSingleValueFilter(filters, "fingerprint", finger)
	}

	if len(name) > 0 {
		params.SetKeyNames([]*string{&name})
	}

	if len(filters) > 0 {
		params.SetFilters(filters)
	}

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, 0, errors.Wrap(err, "getEc2Client")
	}
	ret, err := ec2Client.DescribeKeyPairs(params)
	if err != nil {
		return nil, 0, err
	}

	keypairs := []SKeypair{}
	for _, item := range ret.KeyPairs {
		if err := FillZero(item); err != nil {
			return nil, 0, err
		}

		keypairs = append(keypairs, SKeypair{*item.KeyFingerprint, *item.KeyName})
	}

	return keypairs, len(keypairs), nil
}

// Aws貌似不支持ssh-dss格式密钥
func (self *SRegion) ImportKeypair(name string, pubKey string) (*SKeypair, error) {
	params := &ec2.ImportKeyPairInput{}
	params.SetKeyName(name)
	params.SetPublicKeyMaterial([]byte(pubKey))
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}
	ret, err := ec2Client.ImportKeyPair(params)
	if err != nil {
		return nil, errors.Wrap(err, "ImportKeyPair")
	} else {
		return &SKeypair{StrVal(ret.KeyFingerprint), StrVal(ret.KeyName)}, nil
	}
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

	ks, total, err := self.GetKeypairs(fingerprint, "", 0, 1)
	if total < 1 {
		return "", fmt.Errorf("keypair not found %s", err)
	} else {
		return ks[0].KeyPairName, nil
	}
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
		return k.KeyPairName, nil
	}
}

func (self *SRegion) syncKeypair(publicKey string) (string, error) {
	name, e := self.lookUpAwsKeypair(publicKey)
	if e == nil {
		return name, nil
	}
	return self.importAwsKeypair(publicKey)
}
