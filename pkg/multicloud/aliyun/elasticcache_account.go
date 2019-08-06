package aliyun

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud"
)

// https://help.aliyun.com/document_detail/95802.html?spm=a2c4g.11186623.6.746.1d4b302ayCuzXB
type SElasticcacheAccount struct {
	multicloud.SElasticcacheAccountBase

	cacheDB *SElasticcache

	AccountStatus      string             `json:"AccountStatus"`
	DatabasePrivileges DatabasePrivileges `json:"DatabasePrivileges"`
	InstanceID         string             `json:"InstanceId"`
	AccountName        string             `json:"AccountName"`
	PrivExceeded       string             `json:"PrivExceeded"`
	AccountType        string             `json:"AccountType"`
}

type DatabasePrivileges struct {
	DatabasePrivilege []DatabasePrivilege `json:"DatabasePrivilege"`
}

type DatabasePrivilege struct {
	AccountPrivilege string `json:"AccountPrivilege"`
}

func (self *SElasticcacheAccount) GetId() string {
	return fmt.Sprintf("%s/%s", self.InstanceID, self.AccountName)
}

func (self *SElasticcacheAccount) GetName() string {
	return self.AccountName
}

func (self *SElasticcacheAccount) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheAccount) GetStatus() string {
	return self.AccountStatus
}

func (self *SElasticcacheAccount) GetAccountType() string {
	return self.AccountType
}

func (self *SElasticcacheAccount) GetAccountPrivilege() string {
	if len(self.DatabasePrivileges.DatabasePrivilege) == 0 {
		return ""
	}

	return self.DatabasePrivileges.DatabasePrivilege[0].AccountPrivilege
}
