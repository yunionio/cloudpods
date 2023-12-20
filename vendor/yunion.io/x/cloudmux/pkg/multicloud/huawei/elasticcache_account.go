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

package huawei

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElasticcacheAccount struct {
	multicloud.SElasticcacheAccountBase
	HuaweiTags

	cacheDB *SElasticcache
}

func (self *SElasticcacheAccount) GetId() string {
	return fmt.Sprintf("%s/root", self.cacheDB.InstanceID)
}

func (self *SElasticcacheAccount) GetName() string {
	if len(self.cacheDB.AccessUser) > 0 {
		return self.cacheDB.AccessUser
	}

	return "root"
}

func (self *SElasticcacheAccount) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcacheAccount) GetStatus() string {
	return api.ELASTIC_CACHE_ACCOUNT_STATUS_AVAILABLE
}

func (self *SElasticcacheAccount) GetAccountType() string {
	return api.ELASTIC_CACHE_ACCOUNT_TYPE_ADMIN
}

func (self *SElasticcacheAccount) GetAccountPrivilege() string {
	return api.ELASTIC_CACHE_ACCOUNT_PRIVILEGE_WRITE
}

func (self *SElasticcacheAccount) ResetPassword(input cloudprovider.SCloudElasticCacheAccountResetPasswordInput) error {
	if input.OldPassword == nil {
		return fmt.Errorf("elasticcacheAccount.ResetPassword.input OldPassword should not be empty")
	}
	params := map[string]interface{}{
		"old_password": *input.OldPassword,
		"new_password": input.NewPassword,
	}
	_, err := self.cacheDB.region.put(SERVICE_DCS, fmt.Sprintf("instances/%s/password", self.cacheDB.InstanceID), params)
	return err
}

func (self *SElasticcacheAccount) UpdateAccount(input cloudprovider.SCloudElasticCacheAccountUpdateInput) error {
	if input.Password != nil {
		inputPassword := cloudprovider.SCloudElasticCacheAccountResetPasswordInput{}
		inputPassword.NewPassword = *input.Password
		inputPassword.OldPassword = input.OldPassword
		inputPassword.NoPasswordAccess = input.NoPasswordAccess
		return self.ResetPassword(inputPassword)
	}

	return nil
}

func (self *SElasticcacheAccount) Delete() error {
	return cloudprovider.ErrNotSupported
}
