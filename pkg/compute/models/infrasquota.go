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
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	commonOptions "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	InfrasQuota               SInfrasQuota
	InfrasQuotaManager        *SQuotaManager
	InfrasUsageManager        *SQuotaManager
	InfrasPendingUsageManager *SQuotaManager
)

func init() {
	InfrasQuota = SInfrasQuota{}

	InfrasUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(InfrasQuota,
			rbacscope.ScopeDomain,
			"infras_quota_usage_tbl",
			"infras_quota_usage",
			"infras_quota_usages",
		),
	}
	InfrasPendingUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(InfrasQuota,
			rbacscope.ScopeDomain,
			"infras_quota_pending_usage_tbl",
			"infras_quota_pending_usage",
			"infras_quota_pending_usages",
		),
	}
	InfrasQuotaManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaBaseManager(InfrasQuota,
			rbacscope.ScopeDomain,
			"infras_quota_tbl",
			InfrasPendingUsageManager,
			InfrasUsageManager,
			"infras_quota",
			"infras_quotas",
		),
	}
	quotas.Register(InfrasQuotaManager)
}

type SInfrasQuota struct {
	quotas.SQuotaBase

	quotas.SDomainRegionalCloudResourceKeys

	Host int `default:"-1" allow_zero:"true" json:"host"`
	Vpc  int `default:"-1" allow_zero:"true" json:"vpc"`
}

func (self *SInfrasQuota) GetKeys() quotas.IQuotaKeys {
	return self.SDomainRegionalCloudResourceKeys
}

func (self *SInfrasQuota) SetKeys(keys quotas.IQuotaKeys) {
	self.SDomainRegionalCloudResourceKeys = keys.(quotas.SDomainRegionalCloudResourceKeys)
}

func (self *SInfrasQuota) FetchSystemQuota() {
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
	self.Host = defaultValue(options.Options.DefaultHostQuota)
	self.Vpc = defaultValue(options.Options.DefaultVpcQuota)
}

func (self *SInfrasQuota) FetchUsage(ctx context.Context) error {
	regionKeys := self.SDomainRegionalCloudResourceKeys

	scope := regionKeys.Scope()
	ownerId := regionKeys.OwnerId()

	var rangeObjs []db.IStandaloneModel
	if len(regionKeys.RegionId) > 0 {
		obj, err := CloudregionManager.FetchById(regionKeys.RegionId)
		if err != nil {
			return errors.Wrap(err, "CloudregionManager.FetchById")
		}
		rangeObjs = append(rangeObjs, obj.(db.IStandaloneModel))
	}
	if len(regionKeys.ManagerId) > 0 {
		obj, err := CloudproviderManager.FetchById(regionKeys.ManagerId)
		if err != nil {
			return errors.Wrap(err, "CloudproviderManager.FetchById")
		}
		rangeObjs = append(rangeObjs, obj.(db.IStandaloneModel))
	} else if len(regionKeys.AccountId) > 0 {
		obj, err := CloudaccountManager.FetchById(regionKeys.AccountId)
		if err != nil {
			return errors.Wrap(err, "CloudaccountManager.FetchById")
		}
		rangeObjs = append(rangeObjs, obj.(db.IStandaloneModel))
	}

	var providers []string
	if len(regionKeys.Provider) > 0 {
		providers = []string{regionKeys.Provider}
	}
	var brands []string
	if len(regionKeys.Brand) > 0 {
		brands = []string{regionKeys.Brand}
	}

	hostStat := HostManager.TotalCount(ctx, ownerId, scope, rangeObjs, "", "", nil, nil, providers, brands, regionKeys.CloudEnv, tristate.None, tristate.None, rbacutils.SPolicyResult{})
	self.Host = int(hostStat.Count)
	self.Vpc = VpcManager.totalCount(ctx, ownerId, scope, rangeObjs, providers, brands, regionKeys.CloudEnv)

	return nil
}

func (self *SInfrasQuota) ResetNegative() {
	if self.Host < 0 {
		self.Host = 0
	}
	if self.Vpc < 0 {
		self.Vpc = 0
	}
}

func (self *SInfrasQuota) IsEmpty() bool {
	if self.Host > 0 {
		return false
	}
	if self.Vpc > 0 {
		return false
	}
	return true
}

func (self *SInfrasQuota) Add(quota quotas.IQuota) {
	squota := quota.(*SInfrasQuota)
	self.Host = self.Host + quotas.NonNegative(squota.Host)
	self.Vpc = self.Vpc + quotas.NonNegative(squota.Vpc)
}

func (self *SInfrasQuota) Sub(quota quotas.IQuota) {
	squota := quota.(*SInfrasQuota)
	self.Host = nonNegative(self.Host - squota.Host)
	self.Vpc = nonNegative(self.Vpc - squota.Vpc)
}

func (self *SInfrasQuota) Allocable(request quotas.IQuota) int {
	squota := request.(*SInfrasQuota)
	cnt := -1
	if self.Host >= 0 && squota.Host > 0 && (cnt < 0 || cnt > self.Host/squota.Host) {
		cnt = self.Host / squota.Host
	}
	if self.Vpc >= 0 && squota.Vpc > 0 && (cnt < 0 || cnt > self.Vpc/squota.Vpc) {
		cnt = self.Vpc / squota.Vpc
	}
	return cnt
}

func (self *SInfrasQuota) Update(quota quotas.IQuota) {
	squota := quota.(*SInfrasQuota)
	if squota.Host > 0 {
		self.Host = squota.Host
	}
	if squota.Vpc > 0 {
		self.Vpc = squota.Vpc
	}
}

func (used *SInfrasQuota) Exceed(request quotas.IQuota, quota quotas.IQuota) error {
	err := quotas.NewOutOfQuotaError()
	sreq := request.(*SInfrasQuota)
	squota := quota.(*SInfrasQuota)
	if quotas.Exceed(used.Host, sreq.Host, squota.Host) {
		err.Add(used, "host", squota.Host, used.Host, sreq.Host)
	}
	if quotas.Exceed(used.Vpc, sreq.Vpc, squota.Vpc) {
		err.Add(used, "vpc", squota.Vpc, used.Vpc, sreq.Vpc)
	}
	if err.IsError() {
		return err
	} else {
		return nil
	}
}

func (self *SInfrasQuota) ToJSON(prefix string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewInt(int64(self.Host)), keyName(prefix, "host"))
	ret.Add(jsonutils.NewInt(int64(self.Vpc)), keyName(prefix, "vpc"))
	return ret
}
