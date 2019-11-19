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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SElasticcacheAccount struct {
	multicloud.SElasticcacheAccountBase

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

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423031.html
// 未找到关闭密码的开放api， 不支持开启/关闭密码访问
// https://console.huaweicloud.com/dcs/rest/v2/41f6bfe48d7f4455b7754f7c1b11ae34/instances/26db46e2-c7d8-4b5e-bd36-b5278d2fe17c/password/reset
// new_password: "26db46e2!"
// no_password_access: false
func (self *SElasticcacheAccount) ResetPassword(input cloudprovider.SCloudElasticCacheAccountResetPasswordInput) error {
	if input.OldPassword == nil {
		return fmt.Errorf("elasticcacheAccount.ResetPassword.input OldPassword should not be empty")
	}

	params := jsonutils.NewDict()
	params.Set("old_password", jsonutils.NewString(*input.OldPassword))
	params.Set("new_password", jsonutils.NewString(input.NewPassword))
	err := DoUpdateWithSpec(self.cacheDB.region.ecsClient.Elasticcache.UpdateInContextWithSpec, self.cacheDB.GetId(), "password", params)
	if err != nil {
		return errors.Wrap(err, "elasticcacheAccount.ResetPassword")
	}

	return nil
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
