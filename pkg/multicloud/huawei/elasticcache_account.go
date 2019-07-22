package huawei

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElasticcacheAccount struct {
	multicloud.SElasticcacheAccountBase

	cacheDB *SElasticcache
}

func (self *SElasticcacheAccount) GetId() string {
	return fmt.Sprintf("%s/%s", self.cacheDB.InstanceID, self.cacheDB.AccessUser)
}

func (self *SElasticcacheAccount) GetName() string {
	return self.cacheDB.AccessUser
}

func (self *SElasticcacheAccount) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheAccount) GetStatus() string {
	return ""
}

func (self *SElasticcacheAccount) GetAccountType() string {
	return "admin"
}

func (self *SElasticcacheAccount) GetAccountPrivilege() string {
	return "write"
}
