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

package usages

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/hashcache"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	usageCache = hashcache.NewCache(1024, time.Second*300) // 5 minutes, 1024 buckets cache
)

func getCacheKey(
	scope rbacscope.TRbacScope,
	userCred mcclient.IIdentityProvider,
	isOwner bool,
	rangeObjs []db.IStandaloneModel,
	hostTypes []string,
	providers []string,
	brands []string,
	cloudEnv string,
	includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) string {
	type RangeObject struct {
		Resource string `json:"resource"`
		Id       string `json:"id"`
	}
	type KeyStruct struct {
		Scope     rbacscope.TRbacScope `json:"scope"`
		Domain    string               `json:"domain"`
		Project   string               `json:"project"`
		IsOwner   bool                 `json:"is_owner"`
		Ranges    []RangeObject        `json:"ranges"`
		HostTypes []string             `json:"host_types"`
		Providers []string             `json:"providers"`
		Brands    []string             `json:"brands"`
		CloudEnv  string               `json:"cloud_env"`
		System    bool                 `json:"system"`

		PolicyResult rbacutils.SPolicyResult `json:"policy_result"`
	}
	key := KeyStruct{}
	key.Scope = scope
	switch scope {
	case rbacscope.ScopeSystem:
	case rbacscope.ScopeDomain:
		key.Domain = userCred.GetProjectDomainId()
	case rbacscope.ScopeProject:
		key.Project = userCred.GetProjectId()
	}
	if isOwner {
		key.IsOwner = true
	}
	for _, obj := range rangeObjs {
		robj := RangeObject{
			Resource: obj.Keyword(),
			Id:       obj.GetId(),
		}
		key.Ranges = append(key.Ranges, robj)
	}
	key.HostTypes = hostTypes
	key.Providers = providers
	key.Brands = brands
	key.CloudEnv = cloudEnv
	key.System = includeSystem
	key.PolicyResult = policyResult
	jsonObj := jsonutils.Marshal(key)
	return jsonObj.QueryString()
}
