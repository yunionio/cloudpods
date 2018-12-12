package aws

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aokoli/goutils"
	"github.com/aws/aws-sdk-go/service/ec2"
	"golang.org/x/crypto/ssh"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SKeypair struct {
	KeyPairFingerPrint string
	KeyPairName        string
}

func (self *SRegion) GetKeypairs(finger string, name string, offset int, limit int) ([]SKeypair, int, error) {
	ret, err := self.ec2Client.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{})
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

func (self *SRegion) ImportKeypair(name string, pubKey string) (*SKeypair, error) {
	params := &ec2.ImportKeyPairInput{}
	params.SetKeyName(name)
	params.SetPublicKeyMaterial([]byte(pubKey))
	ret, err := self.ec2Client.ImportKeyPair(params)
	if err != nil {
		return nil, err
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
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("publicKey error %s", err)
	}

	fingerprint := ssh.FingerprintLegacyMD5(pk)
	ks, total, err := self.GetKeypairs(fingerprint, "*", 0, 1)
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
