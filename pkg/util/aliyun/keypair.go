package aliyun

import (
	"encoding/json"
	"fmt"
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

func (self *SRegion) AttachKeypair(instanceId string, name string)  error {
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

func (self *SRegion) DetachKeyPair(instanceId string, name string)  error {
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
