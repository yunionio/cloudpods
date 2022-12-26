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
	"yunion.io/x/pkg/util/rbacscope"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	commonOptions "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

var (
	ProjectQuota               SProjectQuota
	ProjectQuotaManager        *SQuotaManager
	ProjectUsageManager        *SQuotaManager
	ProjectPendingUsageManager *SQuotaManager
)

func init() {
	ProjectQuota = SProjectQuota{}

	ProjectUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(ProjectQuota,
			rbacscope.ScopeProject,
			"project_quota_usage_tbl",
			"project_quota_usage",
			"project_quota_usages",
		),
	}
	ProjectPendingUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(ProjectQuota,
			rbacscope.ScopeProject,
			"project_quota_pending_usage_tbl",
			"project_quota_pending_usage",
			"project_quota_pending_usages",
		),
	}
	ProjectQuotaManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaBaseManager(ProjectQuota,
			rbacscope.ScopeProject,
			"project_quota_tbl",
			ProjectPendingUsageManager,
			ProjectUsageManager,
			"project_quota",
			"project_quotas",
		),
	}
	quotas.Register(ProjectQuotaManager)
}

type SProjectQuota struct {
	quotas.SQuotaBase

	quotas.SBaseProjectQuotaKeys

	Secgroup int `default:"-1" allow_zero:"true" json:"secgroup"`
}

func (self *SProjectQuota) GetKeys() quotas.IQuotaKeys {
	return self.SBaseProjectQuotaKeys
}

func (self *SProjectQuota) SetKeys(keys quotas.IQuotaKeys) {
	self.SBaseProjectQuotaKeys = keys.(quotas.SBaseProjectQuotaKeys)
}

func (self *SProjectQuota) FetchSystemQuota() {
	keys := self.SBaseProjectQuotaKeys
	base := 0
	switch options.Options.DefaultQuotaValue {
	case commonOptions.DefaultQuotaUnlimit:
		base = -1
	case commonOptions.DefaultQuotaZero:
		base = 0
		if keys.Scope() == rbacscope.ScopeDomain { // domain level quota
			base = 10
		} else if keys.DomainId == identityapi.DEFAULT_DOMAIN_ID && keys.ProjectId == auth.AdminCredential().GetProjectId() {
			base = 1
		}
	case commonOptions.DefaultQuotaDefault:
		base = 1
		if keys.Scope() == rbacscope.ScopeDomain {
			base = 10
		}
	}
	defaultValue := func(def int) int {
		if base < 0 {
			return -1
		} else {
			return def * base
		}
	}
	self.Secgroup = defaultValue(options.Options.DefaultSecgroupQuota)
}

func (self *SProjectQuota) FetchUsage(ctx context.Context) error {
	regionKeys := self.SBaseProjectQuotaKeys

	scope := regionKeys.Scope()
	ownerId := regionKeys.OwnerId()

	self.Secgroup, _ = totalSecurityGroupCount(scope, ownerId)
	return nil
}

func (self *SProjectQuota) ResetNegative() {
	if self.Secgroup < 0 {
		self.Secgroup = 0
	}
}

func (self *SProjectQuota) IsEmpty() bool {
	if self.Secgroup > 0 {
		return false
	}
	return true
}

func (self *SProjectQuota) Add(quota quotas.IQuota) {
	squota := quota.(*SProjectQuota)
	self.Secgroup = self.Secgroup + quotas.NonNegative(squota.Secgroup)
}

func (self *SProjectQuota) Sub(quota quotas.IQuota) {
	squota := quota.(*SProjectQuota)
	self.Secgroup = nonNegative(self.Secgroup - squota.Secgroup)
}

func (self *SProjectQuota) Allocable(request quotas.IQuota) int {
	squota := request.(*SProjectQuota)
	cnt := -1
	if self.Secgroup >= 0 && squota.Secgroup > 0 && (cnt < 0 || cnt > self.Secgroup/squota.Secgroup) {
		cnt = self.Secgroup / squota.Secgroup
	}
	return cnt
}

func (self *SProjectQuota) Update(quota quotas.IQuota) {
	squota := quota.(*SProjectQuota)
	if squota.Secgroup > 0 {
		self.Secgroup = squota.Secgroup
	}
}

func (used *SProjectQuota) Exceed(request quotas.IQuota, quota quotas.IQuota) error {
	err := quotas.NewOutOfQuotaError()
	sreq := request.(*SProjectQuota)
	squota := quota.(*SProjectQuota)
	if quotas.Exceed(used.Secgroup, sreq.Secgroup, squota.Secgroup) {
		err.Add(used, "secgroup", squota.Secgroup, used.Secgroup, sreq.Secgroup)
	}
	if err.IsError() {
		return err
	} else {
		return nil
	}
}

func (self *SProjectQuota) ToJSON(prefix string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewInt(int64(self.Secgroup)), keyName(prefix, "secgroup"))
	return ret
}
