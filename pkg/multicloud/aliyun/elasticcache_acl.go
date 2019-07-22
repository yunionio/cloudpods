package aliyun

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElasticcacheAcl struct {
	multicloud.SElasticcacheAclBase

	cacheDB *SElasticcache

	SecurityIPList           string `json:"SecurityIpList"`
	SecurityIPGroupAttribute string `json:"SecurityIpGroupAttribute"`
	SecurityIPGroupName      string `json:"SecurityIpGroupName"`
}

func (self *SElasticcacheAcl) GetId() string {
	return fmt.Sprintf("%s/%s", self.cacheDB.GetId(), self.SecurityIPGroupName)
}

func (self *SElasticcacheAcl) GetName() string {
	return self.SecurityIPGroupName
}

func (self *SElasticcacheAcl) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheAcl) GetStatus() string {
	return ""
}

func (self *SElasticcacheAcl) GetIpList() string {
	return self.SecurityIPList
}
