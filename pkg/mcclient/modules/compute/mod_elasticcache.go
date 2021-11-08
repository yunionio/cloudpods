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

package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	ElasticCache          ElasticCacheManager
	ElasticCacheAccount   ElasticCacheAccountManager
	ElasticCacheBackup    modulebase.ResourceManager
	ElasticCacheAcl       modulebase.ResourceManager
	ElasticCacheParameter modulebase.ResourceManager
	ElasticCacheSecgroup  modulebase.JointResourceManager
)

type ElasticCacheManager struct {
	modulebase.ResourceManager
}

type ElasticCacheAccountManager struct {
	modulebase.ResourceManager
}

func init() {
	ElasticCache = ElasticCacheManager{modules.NewComputeManager("elasticcache", "elasticcaches",
		[]string{"ID", "Name", "Cloudregion_Id", "Status", "InstanceType", "CapacityMB", "Engine", "EngineVersion"},
		[]string{})}

	ElasticCacheAccount = ElasticCacheAccountManager{modules.NewComputeManager("elasticcacheaccount", "elasticcacheaccounts",
		[]string{},
		[]string{})}

	ElasticCacheBackup = modules.NewComputeManager("elasticcachebackup", "elasticcachebackups",
		[]string{},
		[]string{})

	ElasticCacheAcl = modules.NewComputeManager("elasticcacheacl", "elasticcacheacls",
		[]string{},
		[]string{})

	ElasticCacheParameter = modules.NewComputeManager("elasticcacheparameter", "elasticcacheparameters",
		[]string{},
		[]string{})

	ElasticCacheSecgroup = modules.NewJointComputeManager("elasticcachesecgroup",
		"elasticcachesecgroups",
		[]string{},
		[]string{},
		&ElasticCache,
		&SecGroups)

	modules.RegisterCompute(&ElasticCache)
	modules.RegisterCompute(&ElasticCacheAccount)
	modules.RegisterCompute(&ElasticCacheAcl)
	modules.RegisterCompute(&ElasticCacheBackup)
	modules.RegisterCompute(&ElasticCacheParameter)
	modules.RegisterCompute(&ElasticCacheSecgroup)
}

func (self *ElasticCacheManager) GetLoginInfo(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, e := self.GetSpecific(s, id, "login-info", params)
	if e != nil {
		return nil, e
	}

	ret := jsonutils.NewDict()
	username, _ := data.GetString("username")
	ret.Set("username", jsonutils.NewString(username))

	account_id, _ := data.GetString("account_id")
	password, _ := data.GetString("password")
	if len(password) == 0 {
		return nil, httperrors.NewNotFoundError("No password found")
	}

	passwd, e := utils.DescryptAESBase64(account_id, password)
	if e != nil {
		return nil, e
	}
	ret.Set("password", jsonutils.NewString(passwd))
	return ret, nil
}

func (self *ElasticCacheAccountManager) GetLoginInfo(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, e := self.GetSpecific(s, id, "login-info", params)
	if e != nil {
		return nil, e
	}

	ret := jsonutils.NewDict()
	username, _ := data.GetString("username")
	ret.Set("username", jsonutils.NewString(username))

	password, _ := data.GetString("password")
	if len(password) == 0 {
		return nil, httperrors.NewNotFoundError("No password found")
	}

	passwd, e := utils.DescryptAESBase64(id, password)
	if e != nil {
		return nil, e
	}
	ret.Set("password", jsonutils.NewString(passwd))
	return ret, nil
}
