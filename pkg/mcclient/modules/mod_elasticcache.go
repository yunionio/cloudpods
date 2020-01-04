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

package modules

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

var (
	ElasticCache          ElasticCacheManager
	ElasticCacheAccount   ElasticCacheAccountManager
	ElasticCacheBackup    modulebase.ResourceManager
	ElasticCacheAcl       modulebase.ResourceManager
	ElasticCacheParameter modulebase.ResourceManager
)

type ElasticCacheManager struct {
	modulebase.ResourceManager
}

type ElasticCacheAccountManager struct {
	modulebase.ResourceManager
}

func init() {
	ElasticCache = ElasticCacheManager{NewComputeManager("elasticcache", "elasticcaches",
		[]string{"ID", "Name", "Cloudregion_Id", "Status", "InstanceType", "CapacityMB", "Engine", "EngineVersion"},
		[]string{})}

	ElasticCacheAccount = ElasticCacheAccountManager{NewComputeManager("elasticcacheaccount", "elasticcacheaccounts",
		[]string{},
		[]string{})}

	ElasticCacheBackup = NewComputeManager("elasticcachebackup", "elasticcachebackups",
		[]string{},
		[]string{})

	ElasticCacheAcl = NewComputeManager("elasticcacheacl", "elasticcacheacls",
		[]string{},
		[]string{})

	ElasticCacheParameter = NewComputeManager("elasticcacheparameter", "elasticcacheparameters",
		[]string{},
		[]string{})

	registerCompute(&ElasticCache)
	registerCompute(&ElasticCacheAccount)
	registerCompute(&ElasticCacheAcl)
	registerCompute(&ElasticCacheBackup)
	registerCompute(&ElasticCacheParameter)
}

func (self *ElasticCacheManager) GetLoginInfo(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.GetSpecific(s, id, "login-info", params)
}

func (self *ElasticCacheAccountManager) GetLoginInfo(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return self.GetSpecific(s, id, "login-info", params)
}
