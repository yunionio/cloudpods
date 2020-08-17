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

	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	commonOptions "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	DomainQuota               SDomainQuota
	DomainQuotaManager        *SQuotaManager
	DomainUsageManager        *SQuotaManager
	DomainPendingUsageManager *SQuotaManager
)

func init() {
	DomainQuota = SDomainQuota{}

	DomainUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(DomainQuota,
			rbacutils.ScopeDomain,
			"domain_quota_usage_tbl",
			"domain_quota_usage",
			"domain_quota_usages",
		),
	}
	DomainPendingUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(DomainQuota,
			rbacutils.ScopeDomain,
			"domain_quota_pending_usage_tbl",
			"domain_quota_pending_usage",
			"domain_quota_pending_usages",
		),
	}
	DomainQuotaManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaBaseManager(DomainQuota,
			rbacutils.ScopeDomain,
			"domain_quota_tbl",
			DomainPendingUsageManager,
			DomainUsageManager,
			"domain_quota",
			"domain_quotas",
		),
	}
	quotas.Register(DomainQuotaManager)
}

type SDomainQuota struct {
	quotas.SQuotaBase

	quotas.SBaseDomainQuotaKeys

	Cloudaccount int `default:"-1" allow_zero:"true" json:"cloudaccount"`

	Globalvpc int `default:"-1" allow_zero:"true" json:"globalvpc"`

	DnsZone int `default:"-1" allow_zero:"true" json:"dns_zone"`
}

func (self *SDomainQuota) GetKeys() quotas.IQuotaKeys {
	return self.SBaseDomainQuotaKeys
}

func (self *SDomainQuota) SetKeys(keys quotas.IQuotaKeys) {
	self.SBaseDomainQuotaKeys = keys.(quotas.SBaseDomainQuotaKeys)
}

func (self *SDomainQuota) FetchSystemQuota() {
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
	self.Globalvpc = defaultValue(options.Options.DefaultGlobalvpcQuota)
	self.Cloudaccount = defaultValue(options.Options.DefaultCloudaccountQuota)
	self.DnsZone = defaultValue(options.Options.DefaultDnsZoneQuota)
}

func (self *SDomainQuota) FetchUsage(ctx context.Context) error {
	keys := self.SBaseDomainQuotaKeys

	scope := keys.Scope()
	ownerId := keys.OwnerId()

	self.Globalvpc = GlobalVpcManager.totalCount(scope, ownerId)
	self.Cloudaccount = CloudaccountManager.totalCount(scope, ownerId)
	self.DnsZone = DnsZoneManager.totalCount(scope, ownerId)

	return nil
}

func (self *SDomainQuota) ResetNegative() {
	if self.Globalvpc < 0 {
		self.Globalvpc = 0
	}
	if self.Cloudaccount < 0 {
		self.Cloudaccount = 0
	}
	if self.DnsZone < 0 {
		self.DnsZone = 0
	}
}

func (self *SDomainQuota) IsEmpty() bool {
	if self.Globalvpc > 0 {
		return false
	}
	if self.Cloudaccount > 0 {
		return false
	}
	if self.DnsZone > 0 {
		return false
	}
	return true
}

func (self *SDomainQuota) Add(quota quotas.IQuota) {
	squota := quota.(*SDomainQuota)
	self.Globalvpc = self.Globalvpc + quotas.NonNegative(squota.Globalvpc)
	self.Cloudaccount = self.Cloudaccount + quotas.NonNegative(squota.Cloudaccount)
	self.DnsZone = self.DnsZone + quotas.NonNegative(squota.DnsZone)
}

func (self *SDomainQuota) Sub(quota quotas.IQuota) {
	squota := quota.(*SDomainQuota)
	self.Globalvpc = nonNegative(self.Globalvpc - squota.Globalvpc)
	self.Cloudaccount = nonNegative(self.Cloudaccount - squota.Cloudaccount)
	self.DnsZone = nonNegative(self.DnsZone - squota.DnsZone)
}

func (self *SDomainQuota) Allocable(request quotas.IQuota) int {
	squota := request.(*SDomainQuota)
	cnt := -1
	if self.Globalvpc >= 0 && squota.Globalvpc > 0 && (cnt < 0 || cnt > self.Globalvpc/squota.Globalvpc) {
		cnt = self.Globalvpc / squota.Globalvpc
	}
	if self.Cloudaccount >= 0 && squota.Cloudaccount > 0 && (cnt < 0 || cnt > self.Cloudaccount/squota.Cloudaccount) {
		cnt = self.Cloudaccount / squota.Cloudaccount
	}
	if self.DnsZone >= 0 && squota.DnsZone > 0 && (cnt < 0 || cnt > self.DnsZone/squota.DnsZone) {
		cnt = self.DnsZone / squota.DnsZone
	}
	return cnt
}

func (self *SDomainQuota) Update(quota quotas.IQuota) {
	squota := quota.(*SDomainQuota)
	if squota.Globalvpc > 0 {
		self.Globalvpc = squota.Globalvpc
	}
	if squota.Cloudaccount > 0 {
		self.Cloudaccount = squota.Cloudaccount
	}
	if squota.DnsZone > 0 {
		self.DnsZone = squota.DnsZone
	}
}

func (used *SDomainQuota) Exceed(request quotas.IQuota, quota quotas.IQuota) error {
	err := quotas.NewOutOfQuotaError()
	sreq := request.(*SDomainQuota)
	squota := quota.(*SDomainQuota)
	if quotas.Exceed(used.Globalvpc, sreq.Globalvpc, squota.Globalvpc) {
		err.Add("globalvpc", squota.Globalvpc, used.Globalvpc, sreq.Globalvpc)
	}
	if quotas.Exceed(used.Cloudaccount, sreq.Cloudaccount, squota.Cloudaccount) {
		err.Add("cloudaccount", squota.Cloudaccount, used.Cloudaccount, sreq.Cloudaccount)
	}
	if quotas.Exceed(used.DnsZone, sreq.DnsZone, squota.DnsZone) {
		err.Add("dns_zone", squota.DnsZone, used.DnsZone, sreq.DnsZone)
	}
	if err.IsError() {
		return err
	} else {
		return nil
	}
}

func (self *SDomainQuota) ToJSON(prefix string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewInt(int64(self.Globalvpc)), keyName(prefix, "globalvpc"))
	ret.Add(jsonutils.NewInt(int64(self.Cloudaccount)), keyName(prefix, "cloudaccount"))
	ret.Add(jsonutils.NewInt(int64(self.DnsZone)), keyName(prefix, "dns_zone"))
	return ret
}
