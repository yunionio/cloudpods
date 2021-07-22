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

package qcloud

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElasticcacheAccount struct {
	multicloud.SElasticcacheAccountBase
	multicloud.QcloudTags

	cacheDB *SElasticcache

	InstanceID     string   `json:"InstanceId"`
	AccountName    string   `json:"AccountName"`
	Remark         string   `json:"Remark"`
	Privilege      string   `json:"Privilege"`
	ReadonlyPolicy []string `json:"ReadonlyPolicy"`
	Status         int      `json:"Status"`
	IsEmulate      bool     `json:"is_emulate"`
}

func (self *SElasticcacheAccount) GetId() string {
	return fmt.Sprintf("%s/%s", self.InstanceID, self.AccountName)
}

func (self *SElasticcacheAccount) GetName() string {
	return self.AccountName
}

func (self *SElasticcacheAccount) IsEmulated() bool {
	return self.IsEmulate
}

func (self *SElasticcacheAccount) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheAccount) GetStatus() string {
	switch self.Status {
	case 1:
		return api.ELASTIC_CACHE_ACCOUNT_STATUS_CREATING
	case 2:
		return api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE
	case 4:
		return api.ELASTIC_CACHE_ACCOUNT_STATUS_DELETED
	default:
		return api.ELASTIC_CACHE_ACCOUNT_STATUS_UNAVAILABLE
	}
}

func (self *SElasticcacheAccount) GetAccountType() string {
	if strings.ToLower(self.AccountName) == "root" {
		return api.ELASTIC_CACHE_ACCOUNT_TYPE_ADMIN
	} else {
		return api.ELASTIC_CACHE_ACCOUNT_TYPE_NORMAL
	}
}

func (self *SElasticcacheAccount) GetAccountPrivilege() string {
	switch self.Privilege {
	case "r":
		return api.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_READ
	case "w":
		return api.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_WRITE
	case "rw":
		return api.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_WRITE
	default:
		return self.Privilege
	}
}

func (self *SElasticcacheAccount) Refresh() error {
	account, err := self.cacheDB.GetICloudElasticcacheAccount(self.GetGlobalId())
	if err != nil {
		return errors.Wrap(err, "GetICloudElasticcacheAccount")
	}

	err = jsonutils.Update(self, account)
	if err != nil {
		return errors.Wrap(err, "Update")
	}

	return nil
}

// https://cloud.tencent.com/document/product/239/38925
func (self *SElasticcacheAccount) Delete() error {
	params := map[string]string{}
	params["InstanceId"] = self.cacheDB.GetId()
	params["AccountName"] = self.AccountName
	_, err := self.cacheDB.region.redisRequest("DeleteInstanceAccount", params)
	if err != nil {
		return errors.Wrap(err, "DeleteInstanceAccount")
	}

	return nil
}

// https://cloud.tencent.com/document/product/239/20014
func (self *SElasticcacheAccount) ResetPassword(input cloudprovider.SCloudElasticCacheAccountResetPasswordInput) error {
	params := map[string]string{}
	params["InstanceId"] = self.cacheDB.GetId()
	if input.NoPasswordAccess != nil && *input.NoPasswordAccess {
		params["NoAuth"] = "true"
	} else {
		if len(input.NewPassword) > 0 {
			params["Password"] = input.NewPassword
		} else {
			return nil
		}
	}

	_, err := self.cacheDB.region.redisRequest("ResetPassword", params)
	if err != nil {
		return errors.Wrap(err, "ResetPassword")
	}

	return nil
}

// https://cloud.tencent.com/document/product/239/38923
func (self *SElasticcacheAccount) UpdateAccount(input cloudprovider.SCloudElasticCacheAccountUpdateInput) error {
	params := map[string]string{}
	params["InstanceId"] = self.cacheDB.GetId()
	params["AccountName"] = self.AccountName
	if input.NoPasswordAccess != nil && *input.NoPasswordAccess {
		params["NoAuth"] = "true"
	} else {
		if input.Password != nil && len(*input.Password) > 0 {
			params["AccountPassword"] = *input.Password
		}
	}

	if input.Description != nil && len(*input.Description) > 0 {
		params["Remark"] = *input.Description
	}

	if input.AccountPrivilege != nil && len(*input.AccountPrivilege) > 0 {
		params["Privilege"] = *input.AccountPrivilege
		params["ReadonlyPolicy.0"] = "master"
	}

	_, err := self.cacheDB.region.redisRequest("ModifyInstanceAccount", params)
	if err != nil {
		return errors.Wrap(err, "ModifyInstanceAccount")
	}

	return nil
}
