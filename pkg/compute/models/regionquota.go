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
	RegionQuota               SRegionQuota
	RegionQuotaManager        *SQuotaManager
	RegionUsageManager        *SQuotaManager
	RegionPendingUsageManager *SQuotaManager
)

func init() {
	RegionQuota = SRegionQuota{}

	RegionUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(RegionQuota,
			"region_quota_usage_tbl",
			"region_quota_usage",
			"region_quota_usages",
		),
	}
	RegionUsageManager.SetVirtualObject(RegionUsageManager)
	RegionPendingUsageManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaUsageManager(RegionQuota,
			"region_quota_pending_usage_tbl",
			"region_quota_pending_usage",
			"region_quota_pending_usages",
		),
	}
	RegionPendingUsageManager.SetVirtualObject(RegionPendingUsageManager)
	RegionQuotaManager = &SQuotaManager{
		SQuotaBaseManager: quotas.NewQuotaBaseManager(RegionQuota,
			"region_quota_tbl",
			RegionPendingUsageManager,
			RegionUsageManager,
			"region_quota",
			"region_quotas",
		),
	}
	RegionQuotaManager.SetVirtualObject(RegionQuotaManager)
}

type SRegionQuota struct {
	quotas.SQuotaBase

	quotas.SRegionalCloudResourceKeys

	Eip   int
	Port  int
	Eport int
	Bw    int
	Ebw   int

	Snapshot int

	Bucket    int
	ObjectGB  int
	ObjectCnt int

	Rds   int
	Cache int
}

func (self *SRegionQuota) GetKeys() quotas.IQuotaKeys {
	return self.SRegionalCloudResourceKeys
}

func (self *SRegionQuota) SetKeys(keys quotas.IQuotaKeys) {
	self.SRegionalCloudResourceKeys = keys.(quotas.SRegionalCloudResourceKeys)
}

func (self *SRegionQuota) FetchSystemQuota() {
	keys := self.SRegionalCloudResourceKeys
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
	self.Eip = defaultValue(options.Options.DefaultEipQuota)
	self.Port = defaultValue(options.Options.DefaultPortQuota)
	self.Eport = defaultValue(options.Options.DefaultEportQuota)
	self.Bw = defaultValue(options.Options.DefaultBwQuota)
	self.Ebw = defaultValue(options.Options.DefaultEbwQuota)
	self.Snapshot = defaultValue(options.Options.DefaultSnapshotQuota)
	self.Bucket = defaultValue(options.Options.DefaultBucketQuota)
	self.ObjectGB = defaultValue(options.Options.DefaultObjectGBQuota)
	self.ObjectCnt = defaultValue(options.Options.DefaultObjectCntQuota)
	self.Rds = defaultValue(options.Options.DefaultRdsQuota)
	self.Cache = defaultValue(options.Options.DefaultCacheQuota)
}

func (self *SRegionQuota) FetchUsage(ctx context.Context) error {
	regionKeys := self.SRegionalCloudResourceKeys

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

	net := totalGuestNicCount(scope, ownerId, rangeObjs, false, providers, brands, regionKeys.CloudEnv)

	lbnic, _ := totalLBNicCount(scope, ownerId, rangeObjs, providers, brands, regionKeys.CloudEnv)

	eipUsage := ElasticipManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, regionKeys.CloudEnv)

	self.Eip = eipUsage.Total()
	self.Port = net.InternalNicCount + net.InternalVirtualNicCount + lbnic
	self.Eport = net.ExternalNicCount + net.ExternalVirtualNicCount
	self.Bw = net.InternalBandwidth
	self.Ebw = net.ExternalBandwidth

	snapshotCount, _ := TotalSnapshotCount(scope, ownerId, rangeObjs, providers, brands, regionKeys.CloudEnv)
	self.Snapshot = snapshotCount

	bucketUsage := BucketManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, regionKeys.CloudEnv)
	self.Bucket = bucketUsage.Buckets
	self.ObjectGB = int(bucketUsage.Bytes / 1000 / 1000 / 1000)
	self.ObjectCnt = bucketUsage.Objects

	self.Rds, _ = DBInstanceManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, regionKeys.CloudEnv)
	self.Cache, _ = ElasticcacheManager.TotalCount(scope, ownerId, rangeObjs, providers, brands, regionKeys.CloudEnv)

	return nil
}

func (self *SRegionQuota) IsEmpty() bool {
	if self.Port > 0 {
		return false
	}
	if self.Eip > 0 {
		return false
	}
	if self.Eport > 0 {
		return false
	}
	if self.Bw > 0 {
		return false
	}
	if self.Ebw > 0 {
		return false
	}
	if self.Snapshot > 0 {
		return false
	}
	if self.Bucket > 0 {
		return false
	}
	if self.ObjectGB > 0 {
		return false
	}
	if self.ObjectCnt > 0 {
		return false
	}
	if self.Rds > 0 {
		return false
	}
	if self.Cache > 0 {
		return false
	}
	return true
}

func (self *SRegionQuota) Add(quota quotas.IQuota) {
	squota := quota.(*SRegionQuota)
	self.Port = self.Port + quotas.NonNegative(squota.Port)
	self.Eip = self.Eip + quotas.NonNegative(squota.Eip)
	self.Eport = self.Eport + quotas.NonNegative(squota.Eport)
	self.Bw = self.Bw + quotas.NonNegative(squota.Bw)
	self.Ebw = self.Ebw + quotas.NonNegative(squota.Ebw)
	self.Snapshot = self.Snapshot + quotas.NonNegative(squota.Snapshot)
	self.Bucket = self.Bucket + quotas.NonNegative(squota.Bucket)
	self.ObjectGB = self.ObjectGB + quotas.NonNegative(squota.ObjectGB)
	self.ObjectCnt = self.ObjectCnt + quotas.NonNegative(squota.ObjectCnt)
	self.Rds = self.Rds + quotas.NonNegative(squota.Rds)
	self.Cache = self.Cache + quotas.NonNegative(squota.Cache)
}

func (self *SRegionQuota) Sub(quota quotas.IQuota) {
	squota := quota.(*SRegionQuota)
	self.Port = nonNegative(self.Port - squota.Port)
	self.Eip = nonNegative(self.Eip - squota.Eip)
	self.Eport = nonNegative(self.Eport - squota.Eport)
	self.Bw = nonNegative(self.Bw - squota.Bw)
	self.Ebw = nonNegative(self.Ebw - squota.Ebw)
	self.Snapshot = nonNegative(self.Snapshot - squota.Snapshot)
	self.Bucket = nonNegative(self.Bucket - squota.Bucket)
	self.ObjectGB = nonNegative(self.ObjectGB - squota.ObjectGB)
	self.ObjectCnt = nonNegative(self.ObjectCnt - squota.ObjectCnt)
	self.Rds = nonNegative(self.Rds - squota.Rds)
	self.Cache = nonNegative(self.Cache - squota.Cache)
}

func (self *SRegionQuota) Update(quota quotas.IQuota) {
	squota := quota.(*SRegionQuota)
	if squota.Port > 0 {
		self.Port = squota.Port
	}
	if squota.Eip > 0 {
		self.Eip = squota.Eip
	}
	if squota.Eport > 0 {
		self.Eport = squota.Eport
	}
	if squota.Bw > 0 {
		self.Bw = squota.Bw
	}
	if squota.Ebw > 0 {
		self.Ebw = squota.Ebw
	}
	if squota.Snapshot > 0 {
		self.Snapshot = squota.Snapshot
	}
	if squota.Bucket > 0 {
		self.Bucket = squota.Bucket
	}
	if squota.ObjectGB > 0 {
		self.ObjectGB = squota.ObjectGB
	}
	if squota.ObjectCnt > 0 {
		self.ObjectCnt = squota.ObjectCnt
	}
	if squota.Rds > 0 {
		self.Rds = squota.Rds
	}
	if squota.Cache > 0 {
		self.Cache = squota.Cache
	}
}

func (self *SRegionQuota) Exceed(request quotas.IQuota, quota quotas.IQuota) error {
	err := quotas.NewOutOfQuotaError()
	sreq := request.(*SRegionQuota)
	squota := quota.(*SRegionQuota)
	if sreq.Port > 0 && self.Port+sreq.Port > squota.Port {
		err.Add("port", squota.Port, self.Port, sreq.Port)
	}
	if sreq.Eip > 0 && self.Eip+sreq.Eip > squota.Eip {
		err.Add("eip", squota.Eip, self.Eip, sreq.Eip)
	}
	if sreq.Eport > 0 && self.Eport+sreq.Eport > squota.Eport {
		err.Add("eport", squota.Eport, self.Eport, sreq.Eport)
	}
	if sreq.Bw > 0 && self.Bw+sreq.Bw > squota.Bw {
		err.Add("bw", squota.Bw, self.Bw, sreq.Bw)
	}
	if sreq.Ebw > 0 && self.Ebw+sreq.Ebw > squota.Ebw {
		err.Add("ebw", squota.Ebw, self.Ebw, sreq.Ebw)
	}
	if sreq.Snapshot > 0 && self.Snapshot+sreq.Snapshot > squota.Snapshot {
		err.Add("snapshot", squota.Snapshot, self.Snapshot, sreq.Snapshot)
	}
	if sreq.Bucket > 0 && self.Bucket+sreq.Bucket > squota.Bucket {
		err.Add("bucket", squota.Bucket, self.Bucket, sreq.Bucket)
	}
	if sreq.ObjectGB > 0 && self.ObjectGB+sreq.ObjectGB > squota.ObjectGB {
		err.Add("object_gb", squota.ObjectGB, self.ObjectGB, sreq.ObjectGB)
	}
	if sreq.ObjectCnt > 0 && self.ObjectCnt+sreq.ObjectCnt > squota.ObjectCnt {
		err.Add("object_cnt", squota.ObjectCnt, self.ObjectCnt, sreq.ObjectCnt)
	}
	if sreq.Rds > 0 && self.Rds+sreq.Rds > squota.Rds {
		err.Add("rds", squota.Rds, self.Rds, sreq.Rds)
	}
	if sreq.Cache > 0 && self.Cache+sreq.Cache > squota.Cache {
		err.Add("cache", squota.Cache, self.Cache, sreq.Cache)
	}
	if err.IsError() {
		return err
	} else {
		return nil
	}
}

func (self *SRegionQuota) ToJSON(prefix string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewInt(int64(self.Port)), keyName(prefix, "port"))
	ret.Add(jsonutils.NewInt(int64(self.Eip)), keyName(prefix, "eip"))
	ret.Add(jsonutils.NewInt(int64(self.Eport)), keyName(prefix, "eport"))
	ret.Add(jsonutils.NewInt(int64(self.Bw)), keyName(prefix, "bw"))
	ret.Add(jsonutils.NewInt(int64(self.Ebw)), keyName(prefix, "ebw"))
	ret.Add(jsonutils.NewInt(int64(self.Snapshot)), keyName(prefix, "snapshot"))
	ret.Add(jsonutils.NewInt(int64(self.Bucket)), keyName(prefix, "bucket"))
	ret.Add(jsonutils.NewInt(int64(self.ObjectGB)), keyName(prefix, "object_gb"))
	ret.Add(jsonutils.NewInt(int64(self.ObjectCnt)), keyName(prefix, "object_cnt"))
	ret.Add(jsonutils.NewInt(int64(self.Rds)), keyName(prefix, "rds"))
	ret.Add(jsonutils.NewInt(int64(self.Cache)), keyName(prefix, "cache"))
	return ret
}
