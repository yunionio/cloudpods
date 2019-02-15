package huawei

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aokoli/goutils"
	"golang.org/x/crypto/ssh"
	"yunion.io/x/jsonutils"
)

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212676.html
type SKeypair struct {
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
}

func (self *SRegion) getFingerprint(publicKey string) (string, error) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", fmt.Errorf("publicKey error %s", err)
	}

	fingerprint := strings.Replace(ssh.FingerprintLegacyMD5(pk), ":", "", -1)
	return fingerprint, nil
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212676.html
func (self *SRegion) GetKeypairs() ([]SKeypair, int, error) {
	keypairs := make([]SKeypair, 0)
	err := doListAll(self.ecsClient.Keypairs.List, nil, &keypairs)
	return keypairs, len(keypairs), err
}

func (self *SRegion) lookUpKeypair(publicKey string) (string, error) {
	keypairs, _, err := self.GetKeypairs()
	if err != nil {
		return "", err
	}

	fingerprint, err := self.getFingerprint(publicKey)
	if err != nil {
		return "", err
	}

	for _, keypair := range keypairs {
		if keypair.Fingerprint == fingerprint {
			return keypair.Name, nil
		}
	}

	return "", fmt.Errorf("keypair not found %s", err)
}

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212678.html
func (self *SRegion) ImportKeypair(name, publicKey string) (*SKeypair, error) {
	fingerprint, err := self.getFingerprint(publicKey)
	if err != nil {
		return nil, err
	}

	keypair := SKeypair{
		Name:        name,
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
	}

	keypairObj := jsonutils.Marshal(keypair)
	ret := SKeypair{}
	err = DoCreate(self.ecsClient.Keypairs.Create, keypairObj, &ret)
	return &ret, err
}

func (self *SRegion) importKeypair(publicKey string) (string, error) {
	prefix, e := goutils.RandomAlphabetic(6)
	if e != nil {
		return "", fmt.Errorf("publicKey error %s", e)
	}

	name := prefix + strconv.FormatInt(time.Now().Unix(), 10)
	if k, e := self.ImportKeypair(name, publicKey); e != nil {
		return "", fmt.Errorf("keypair import error %s", e)
	} else {
		return k.Name, nil
	}
}

func (self *SRegion) syncKeypair(publicKey string) (string, error) {
	name, e := self.lookUpKeypair(publicKey)
	if e == nil {
		return name, nil
	}
	return self.importKeypair(publicKey)
}
