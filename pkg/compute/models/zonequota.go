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

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	commonOptions "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

var (
	ZoneQuota               SZoneQuota
	ZoneQuotaManager        *SQuotaManager
	ZoneUsageManager        *SQuotaManager
	ZonePendingUsageManager *SQuotaManager
)

func init() {
	ZoneQuota = SZoneQuota{}

	ZoneUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(ZoneQuota,
			"zone_quota_usage_tbl",
			"zone_quota_usage",
			"zone_quota_usages",
		),
	}
	ZoneUsageManager.SetVirtualObject(ZoneUsageManager)
	ZonePendingUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(ZoneQuota,
			"zone_quota_pending_usage_tbl",
			"zone_quota_pending_usage",
			"zone_quota_pending_usages",
		),
	}
	ZonePendingUsageManager.SetVirtualObject(ZonePendingUsageManager)
	ZoneQuotaManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaBaseManager(ZoneQuota,
			"zone_quota_tbl",
			ZonePendingUsageManager,
			ZoneUsageManager,
			"zone_quota",
			"zone_quotas",
		),
	}
	ZoneQuotaManager.SetVirtualObject(ZoneQuotaManager)
}

type SZoneQuota struct {
	quotas.SQuotaBase

	quotas.SZonalCloudResourceKeys

	Loadbalancer int
}

func (self *SZoneQuota) GetKeys() quotas.IQuotaKeys {
	return self.SZonalCloudResourceKeys
}

func (self *SZoneQuota) SetKeys(keys quotas.IQuotaKeys) {
	self.SZonalCloudResourceKeys = keys.(quotas.SZonalCloudResourceKeys)
}

func (self *SZoneQuota) FetchSystemQuota() {
	keys := self.SBaseQuotaKeys
	base := 0
	switch options.Options.DefaultQuotaValue {
	case commonOptions.DefaultQuotaUnlimit:
		base = -1
	case commonOptions.DefaultQuotaZero:
		base = 0
		if keys.Scope() == rbacutils.ScopeDomain { // domain level quota
			base = 10
		} else if keys.DomainId == identityapi.DEFAULT_DOMAIN_ID && keys.ProjectId == auth.AdminCredential().GetProjectId() {
			base = 1
		}
	case commonOptions.DefaultQuotaDefault:
		base = 1
		if keys.Scope() == rbacutils.ScopeDomain {
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
	self.Loadbalancer = defaultValue(options.Options.DefaultLoadbalancerQuota)
}

func (self *SZoneQuota) FetchUsage(ctx context.Context) error {
	keys := self.SZonalCloudResourceKeys

	scope := keys.Scope()
	ownerId := keys.OwnerId()

	rangeObjs := make([]db.IStandaloneModel, 0)
	if len(keys.ManagerId) > 0 {
		obj, err := CloudproviderManager.FetchById(keys.ManagerId)
		if err != nil {
			return errors.Wrap(err, "CloudproviderManager.FetchById")
		}
		rangeObjs = append(rangeObjs, obj.(db.IStandaloneModel))
	} else if len(keys.AccountId) > 0 {
		obj, err := CloudaccountManager.FetchById(keys.AccountId)
		if err != nil {
			return errors.Wrap(err, "CloudaccountManager.FetchById")
		}
		rangeObjs = append(rangeObjs, obj.(db.IStandaloneModel))
	}

	if len(keys.ZoneId) > 0 {
		obj, err := ZoneManager.FetchById(keys.ZoneId)
		if err != nil {
			return errors.Wrap(err, "ZoneManager.FetchById")
		}
		rangeObjs = append(rangeObjs, obj.(db.IStandaloneModel))
	} else if len(keys.RegionId) > 0 {
		obj, err := CloudregionManager.FetchById(keys.RegionId)
		if err != nil {
			return errors.Wrap(err, "CloudregionManager.FetchById")
		}
		rangeObjs = append(rangeObjs, obj.(db.IStandaloneModel))
	}
	var providers []string
	if len(keys.Provider) > 0 {
		providers = []string{keys.Provider}
	}
	var brands []string
	if len(keys.Brand) > 0 {
		brands = []string{keys.Brand}
	}

	self.Loadbalancer, _ = LoadbalancerManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, keys.CloudEnv)

	return nil
}

func (self *SZoneQuota) IsEmpty() bool {
	if self.Loadbalancer > 0 {
		return false
	}
	return true
}

func (self *SZoneQuota) Add(quota quotas.IQuota) {
	squota := quota.(*SZoneQuota)
	self.Loadbalancer = self.Loadbalancer + quotas.NonNegative(squota.Loadbalancer)
}

func (self *SZoneQuota) Sub(quota quotas.IQuota) {
	squota := quota.(*SZoneQuota)
	self.Loadbalancer = nonNegative(self.Loadbalancer - squota.Loadbalancer)
}

func (self *SZoneQuota) Update(quota quotas.IQuota) {
	squota := quota.(*SZoneQuota)
	if squota.Loadbalancer > 0 {
		self.Loadbalancer = squota.Loadbalancer
	}
}

func (self *SZoneQuota) Exceed(request quotas.IQuota, quota quotas.IQuota) error {
	err := quotas.NewOutOfQuotaError()
	sreq := request.(*SZoneQuota)
	squota := quota.(*SZoneQuota)
	if sreq.Loadbalancer > 0 && self.Loadbalancer+sreq.Loadbalancer > squota.Loadbalancer {
		err.Add("loadbalancer", squota.Loadbalancer, self.Loadbalancer, sreq.Loadbalancer)
	}
	if err.IsError() {
		return err
	} else {
		return nil
	}
}

func (self *SZoneQuota) ToJSON(prefix string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewInt(int64(self.Loadbalancer)), keyName(prefix, "loadbalancer"))
	return ret
}
