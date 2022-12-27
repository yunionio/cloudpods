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

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	commonOptions "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/keystone/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SQuotaManager struct {
	quotas.SQuotaBaseManager
}

var (
	IdentityQuota               SIdentityQuota
	IdentityQuotaManager        *SQuotaManager
	IdentityUsageManager        *SQuotaManager
	IdentityPendingUsageManager *SQuotaManager
)

func init() {
	IdentityQuota = SIdentityQuota{}

	IdentityUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(IdentityQuota,
			rbacscope.ScopeDomain,
			"identity_quota_usage_tbl",
			"identity_quota_usage",
			"identity_quota_usages",
		),
	}
	IdentityPendingUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(IdentityQuota,
			rbacscope.ScopeDomain,
			"identity_quota_pending_usage_tbl",
			"identity_quota_pending_usage",
			"identity_quota_pending_usages",
		),
	}
	IdentityQuotaManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaBaseManager(IdentityQuota,
			rbacscope.ScopeDomain,
			"identity_quota_tbl",
			IdentityPendingUsageManager,
			IdentityUsageManager,
			"identity_quota",
			"identity_quotas",
		),
	}
	quotas.Register(IdentityQuotaManager)
}

type SIdentityQuota struct {
	quotas.SQuotaBase

	quotas.SBaseDomainQuotaKeys

	User    int `default:"-1" allow_zero:"true" json:"user"`
	Group   int `default:"-1" allow_zero:"true" json:"group"`
	Project int `default:"-1" allow_zero:"true" json:"project"`
	Role    int `default:"-1" allow_zero:"true" json:"role"`
	Policy  int `default:"-1" allow_zero:"true" json:"policy"`
}

func (self *SIdentityQuota) GetKeys() quotas.IQuotaKeys {
	return self.SBaseDomainQuotaKeys
}

func (self *SIdentityQuota) SetKeys(keys quotas.IQuotaKeys) {
	self.SBaseDomainQuotaKeys = keys.(quotas.SBaseDomainQuotaKeys)
}

func (self *SIdentityQuota) FetchSystemQuota() {
	base := 0
	switch options.Options.DefaultQuotaValue {
	case commonOptions.DefaultQuotaUnlimit:
		base = -1
	case commonOptions.DefaultQuotaZero:
		base = 0
	case commonOptions.DefaultQuotaDefault:
		base = 1
	}
	defaultValue := func(def int) int {
		if base < 0 {
			return -1
		} else {
			return def * base
		}
	}
	self.User = defaultValue(options.Options.DefaultUserQuota)
	self.Group = defaultValue(options.Options.DefaultGroupQuota)
	self.Project = defaultValue(options.Options.DefaultProjectQuota)
	self.Role = defaultValue(options.Options.DefaultRoleQuota)
	self.Policy = defaultValue(options.Options.DefaultPolicyQuota)
}

func (self *SIdentityQuota) FetchUsage(ctx context.Context) error {
	keys := self.SBaseDomainQuotaKeys

	scope := keys.Scope()
	ownerId := keys.OwnerId()

	self.User = UserManager.totalCount(scope, ownerId)
	self.Group = GroupManager.totalCount(scope, ownerId)
	self.Project = ProjectManager.totalCount(scope, ownerId)
	self.Role = RoleManager.totalCount(scope, ownerId)
	self.Policy = PolicyManager.totalCount(scope, ownerId)

	return nil
}

func (self *SIdentityQuota) ResetNegative() {
	if self.User < 0 {
		self.User = 0
	}
	if self.Group < 0 {
		self.Group = 0
	}
	if self.Project < 0 {
		self.Project = 0
	}
	if self.Role < 0 {
		self.Role = 0
	}
	if self.Policy < 0 {
		self.Policy = 0
	}
}

func (self *SIdentityQuota) IsEmpty() bool {
	if self.User > 0 {
		return false
	}
	if self.Group > 0 {
		return false
	}
	if self.Project > 0 {
		return false
	}
	if self.Role > 0 {
		return false
	}
	if self.Policy > 0 {
		return false
	}
	return true
}

func (self *SIdentityQuota) Add(quota quotas.IQuota) {
	squota := quota.(*SIdentityQuota)
	self.User = self.User + quotas.NonNegative(squota.User)
	self.Group = self.Group + quotas.NonNegative(squota.Group)
	self.Project = self.Project + quotas.NonNegative(squota.Project)
	self.Role = self.Role + quotas.NonNegative(squota.Role)
	self.Policy = self.Policy + quotas.NonNegative(squota.Policy)
}

func (self *SIdentityQuota) Sub(quota quotas.IQuota) {
	squota := quota.(*SIdentityQuota)
	self.User = quotas.NonNegative(self.User - squota.User)
	self.Group = quotas.NonNegative(self.Group - squota.Group)
	self.Project = quotas.NonNegative(self.Project - squota.Project)
	self.Role = quotas.NonNegative(self.Role - squota.Role)
	self.Policy = quotas.NonNegative(self.Policy - squota.Policy)
}

func (self *SIdentityQuota) Allocable(request quotas.IQuota) int {
	squota := request.(*SIdentityQuota)
	cnt := -1
	if self.User >= 0 && squota.User > 0 && (cnt < 0 || cnt > self.User/squota.User) {
		cnt = self.User / squota.User
	}
	if self.Group >= 0 && squota.Group > 0 && (cnt < 0 || cnt > self.Group/squota.Group) {
		cnt = self.Group / squota.Group
	}
	if self.Project >= 0 && squota.Project > 0 && (cnt < 0 || cnt > self.Project/squota.Project) {
		cnt = self.Project / squota.Project
	}
	if self.Role >= 0 && squota.Role > 0 && (cnt < 0 || cnt > self.Role/squota.Role) {
		cnt = self.Role / squota.Role
	}
	if self.Policy >= 0 && squota.Policy > 0 && (cnt < 0 || cnt > self.Policy/squota.Policy) {
		cnt = self.Policy / squota.Policy
	}
	return cnt
}

func (self *SIdentityQuota) Update(quota quotas.IQuota) {
	squota := quota.(*SIdentityQuota)
	if squota.User > 0 {
		self.User = squota.User
	}
	if squota.Group > 0 {
		self.Group = squota.Group
	}
	if squota.Project > 0 {
		self.Project = squota.Project
	}
	if squota.Role > 0 {
		self.Role = squota.Role
	}
	if squota.Policy > 0 {
		self.Policy = squota.Policy
	}

}

func (used *SIdentityQuota) Exceed(request quotas.IQuota, quota quotas.IQuota) error {
	err := quotas.NewOutOfQuotaError()
	sreq := request.(*SIdentityQuota)
	squota := quota.(*SIdentityQuota)
	if quotas.Exceed(used.User, sreq.User, squota.User) {
		err.Add(used, "user", squota.User, used.User, sreq.User)
	}
	if quotas.Exceed(used.Group, sreq.Group, squota.Group) {
		err.Add(used, "group", squota.Group, used.Group, sreq.Group)
	}
	if quotas.Exceed(used.Project, sreq.Project, squota.Project) {
		err.Add(used, "project", squota.Project, used.Project, sreq.Project)
	}
	if quotas.Exceed(used.Role, sreq.Role, squota.Role) {
		err.Add(used, "role", squota.Role, used.Role, sreq.Role)
	}
	if quotas.Exceed(used.Policy, sreq.Policy, squota.Policy) {
		err.Add(used, "policy", squota.Policy, used.Policy, sreq.Policy)
	}
	if err.IsError() {
		return err
	} else {
		return nil
	}
}

func (self *SIdentityQuota) ToJSON(prefix string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewInt(int64(self.User)), quotas.KeyName(prefix, "user"))
	ret.Add(jsonutils.NewInt(int64(self.Group)), quotas.KeyName(prefix, "group"))
	ret.Add(jsonutils.NewInt(int64(self.Project)), quotas.KeyName(prefix, "project"))
	ret.Add(jsonutils.NewInt(int64(self.Role)), quotas.KeyName(prefix, "role"))
	ret.Add(jsonutils.NewInt(int64(self.Policy)), quotas.KeyName(prefix, "policy"))
	return ret
}

func (manager *SQuotaManager) FetchIdNames(ctx context.Context, idMap map[string]map[string]string) (map[string]map[string]string, error) {
	for field := range idMap {
		switch field {
		case "domain_id":
			fieldIdMap, err := db.FetchIdNameMap(DomainManager, idMap[field])
			if err != nil {
				return nil, errors.Wrap(err, "db.FetchIdNameMap")
			}
			idMap[field] = fieldIdMap
		}
	}
	return idMap, nil
}

func (manager *SQuotaManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchDomainInfo(ctx, data)
}

///////////////////////////////////////////////////
// for swagger API documentation

// 域的认证配额详情
type SIdentityQuotaDetail struct {
	SIdentityQuota

	quotas.SBaseDomainQuotaDetailKeys
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/identity_quotas/domains/{domainId}
// +onecloud:swagger-gen-route-tag=identity_quota
// +onecloud:swagger-gen-param-path=domainId
// +onecloud:swagger-gen-param-path=指定域的ID
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=identity_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取指定域的认证配额
func GetIdentityQuota(query quotas.SBaseQuotaQueryInput) *SIdentityQuotaDetail {
	return nil
}

// +onecloud:swagger-gen-route-method=GET
// +onecloud:swagger-gen-route-path=/identity_quotas/domains
// +onecloud:swagger-gen-route-tag=identity_quota
// +onecloud:swagger-gen-param-query-index=0
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=identity_quotas
// +onecloud:swagger-gen-resp-body-list

// 获取所有域的域配额
func ListIdentityQuotas(query quotas.SBaseQuotaQueryInput) *SIdentityQuotaDetail {
	return nil
}

// 设置域的认证配额输入参数
type SetIdentityQuotaInput struct {
	quotas.SBaseQuotaSetInput

	SIdentityQuota
}

// +onecloud:swagger-gen-route-method=POST
// +onecloud:swagger-gen-route-path=/identity_quotas/domains/{domainId}
// +onecloud:swagger-gen-route-tag=identity_quota
// +onecloud:swagger-gen-param-path=domainId
// +onecloud:swagger-gen-param-path=指定域的ID
// +onecloud:swagger-gen-param-body-index=0
// +onecloud:swagger-gen-param-body-key=identity_quotas
// +onecloud:swagger-gen-resp-index=0
// +onecloud:swagger-gen-resp-body-key=identity_quotas
// +onecloud:swagger-gen-resp-body-list

// 设置域的认证配额
func SetIdentityQuotas(input SetIdentityQuotaInput) *SIdentityQuotaDetail {
	return nil
}
