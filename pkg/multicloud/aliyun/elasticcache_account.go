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

package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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
	switch self.AccountStatus {
	case "Unavailable":
		return api.ELASTIC_CACHE_ACCOUNT_STATUS_UNAVAILABLE
	case "Available":
		return api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE
	default:
		return self.AccountStatus
	}
}

func (self *SElasticcacheAccount) Refresh() error {
	iaccount, err := self.cacheDB.GetICloudElasticcacheAccountByName(self.GetName())
	if err != nil {
		return err
	}

	err = jsonutils.Update(self, iaccount.(*SElasticcacheAccount))
	if err != nil {
		return err
	}

	return nil
}

func (self *SElasticcacheAccount) GetAccountType() string {
	if self.AccountName == self.cacheDB.InstanceID {
		return api.ELASTIC_CACHE_ACCOUNT_TYPE_ADMIN
	}

	switch self.AccountType {
	case "Normal":
		return api.ELASTIC_CACHE_ACCOUNT_TYPE_NORMAL
	case "Super":
		return api.ELASTIC_CACHE_ACCOUNT_TYPE_ADMIN
	default:
		return self.AccountType
	}
}

func (self *SElasticcacheAccount) GetAccountPrivilege() string {
	if len(self.DatabasePrivileges.DatabasePrivilege) == 0 {
		return ""
	}

	privilege := self.DatabasePrivileges.DatabasePrivilege[0].AccountPrivilege
	switch privilege {
	case "RoleReadOnly":
		return api.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_READ
	case "RoleReadWrite":
		return api.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_WRITE
	case "RoleRepl":
		return api.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_REPL
	default:
		return privilege
	}
}

// https://help.aliyun.com/document_detail/95941.html?spm=a2c4g.11186623.6.746.523555d8D8Whxq
// https://help.aliyun.com/document_detail/98531.html?spm=5176.11065259.1996646101.searchclickresult.4df474c38Sc2SO
func (self *SElasticcacheAccount) ResetPassword(input cloudprovider.SCloudElasticCacheAccountResetPasswordInput) error {
	params := make(map[string]string)
	params["InstanceId"] = self.cacheDB.GetId()
	params["AccountName"] = self.GetName()
	params["AccountPassword"] = input.NewPassword

	err := DoAction(self.cacheDB.region.kvsRequest, "ResetAccountPassword", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcacheAccount.ResetPassword")
	}

	if input.NoPasswordAccess != nil {
		return cloudprovider.ErrNotSupported
	}

	return nil
}

func (self *SElasticcacheAccount) UpdateAccount(input cloudprovider.SCloudElasticCacheAccountUpdateInput) error {
	if input.Password != nil {
		inputPassword := cloudprovider.SCloudElasticCacheAccountResetPasswordInput{}
		inputPassword.NewPassword = *input.Password
		inputPassword.NoPasswordAccess = input.NoPasswordAccess
		err := self.ResetPassword(inputPassword)
		if err != nil {
			return err
		}
	}

	if input.Description != nil {
		err := self.ModifyAccountDescription(*input.Description)
		if err != nil {
			return err
		}
	}

	if input.AccountPrivilege != nil {
		err := self.GrantAccountPrivilege(*input.AccountPrivilege)
		if err != nil {
			return err
		}
	}

	return nil
}

// https://help.aliyun.com/document_detail/96020.html?spm=a2c4g.11186623.6.747.5c6f1c717weYEX
func (self *SElasticcacheAccount) ModifyAccountDescription(desc string) error {
	params := make(map[string]string)
	params["InstanceId"] = self.cacheDB.GetId()
	params["AccountName"] = self.GetName()
	params["AccountDescription"] = desc

	err := DoAction(self.cacheDB.region.kvsRequest, "ModifyAccountDescription", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcacheAccount.ModifyAccountDescription")
	}

	return nil
}

// https://help.aliyun.com/document_detail/95897.html?spm=a2c4g.11186623.6.745.28576cf9IqM44R
func (self *SElasticcacheAccount) GrantAccountPrivilege(privilege string) error {
	params := make(map[string]string)
	params["InstanceId"] = self.cacheDB.GetId()
	params["AccountName"] = self.GetName()
	params["AccountPrivilege"] = privilege

	err := DoAction(self.cacheDB.region.kvsRequest, "GrantAccountPrivilege", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcacheAccount.GrantAccountPrivilege")
	}

	return nil
}

// https://help.aliyun.com/document_detail/95988.html?spm=a2c4g.11186623.6.743.cb291c71D1Hlmu
func (self *SElasticcacheAccount) Delete() error {
	params := make(map[string]string)
	params["InstanceId"] = self.cacheDB.GetId()
	params["AccountName"] = self.GetName()

	err := DoAction(self.cacheDB.region.kvsRequest, "DeleteAccount", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcacheAccount.Delete")
	}

	return nil
}
