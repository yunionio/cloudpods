package qcloud

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aokoli/goutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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

	body, err := self.cvmRequest("DescribeKeyPairs", params)
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
	total, _ := body.Int("TotalCount")
	return keypairs, int(total), nil
}

func (self *SRegion) ImportKeypair(name string, pubKey string) (*SKeypair, error) {
	params := map[string]string{}
	params["PublicKey"] = pubKey
	params["ProjectId"] = "0"
	params["KeyName"] = name

	body, err := self.cvmRequest("ImportKeyPair", params)
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
	_, err := self.cvmRequest("AssociateInstancesKeyPairs", params)
	return err
}

func (self *SRegion) DetachKeyPair(instanceId string, keypairId string) error {
	params := make(map[string]string)
	params["InstanceIds.0"] = instanceId
	params["KeyIds.0"] = keypairId
	_, err := self.cvmRequest("DisassociateInstancesKeyPairs", params)
	return err
}

func (self *SRegion) CreateKeyPair(name string) (*SKeypair, error) {
	params := make(map[string]string)
	params["KeyName"] = name
	params["ProjectId"] = "0"
	body, err := self.cvmRequest("CreateKeyPair", params)
	keypair := SKeypair{}
	err = body.Unmarshal(&keypair, "KeyPair")
	if err != nil {
		return nil, err
	}
	return &keypair, err
}

func (self *SRegion) lookUpAliyunKeypair(publicKey string) (string, error) {
	keypairs, _, err := self.GetKeypairs("", []string{}, 0, 0)
	if err != nil {
		return "", err
	}

	for i := 0; i < len(keypairs); i++ {
		if keypairs[i].PublicKey == publicKey {
			return keypairs[i].KeyId, nil
		}
	}
	return "", cloudprovider.ErrNotFound
}

func (self *SRegion) syncKeypair(publicKey string) (string, error) {
	keypairId, e := self.lookUpAliyunKeypair(publicKey)
	if e == nil {
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
