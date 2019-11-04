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
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSecurityGroupCacheManager struct {
	db.SStatusStandaloneResourceBaseManager
}

type SSecurityGroupCache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase
	SManagedResourceBase

	SecgroupId string `width:"128" charset:"ascii" list:"user" create:"required"`
	VpcId      string `width:"128" charset:"ascii" list:"user" create:"required"`
}

var SecurityGroupCacheManager *SSecurityGroupCacheManager

func init() {
	SecurityGroupCacheManager = &SSecurityGroupCacheManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SSecurityGroupCache{},
			"secgroupcache_tbl",
			"secgroupcache",
			"secgroupcaches",
		),
	}
	SecurityGroupCacheManager.SetVirtualObject(SecurityGroupCacheManager)
}

func (manager *SSecurityGroupCacheManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *SSecurityGroupCacheManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSecurityGroupCache) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (manager *SSecurityGroupCacheManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	if defsecgroup, _ := query.GetString("secgroup"); len(defsecgroup) > 0 {
		secgroup, err := SecurityGroupManager.FetchByIdOrName(userCred, defsecgroup)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(SecurityGroupManager.Keyword(), defsecgroup)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		q = q.Equals("secgroup_id", secgroup.GetId())
	}
	return q, nil
}

func (self *SSecurityGroupCache) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, err
	}
	if region := CloudregionManager.FetchRegionById(self.CloudregionId); region != nil {
		return provider.GetIRegionById(region.ExternalId)
	}
	return nil, fmt.Errorf("failed to find iregion for secgroupcache %s vpc: %s externalId: %s", self.Id, self.VpcId, self.ExternalId)
}

func (self *SSecurityGroupCache) GetVpc() (*SVpc, error) {
	vpc, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		return nil, err
	}
	return vpc.(*SVpc), nil
}

func (self *SSecurityGroupCache) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	regionInfo := self.SCloudregionResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if regionInfo != nil {
		extra.Update(regionInfo)
	}
	accountInfo := self.SManagedResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if accountInfo != nil {
		extra.Update(accountInfo)
	}
	vpc, _ := self.GetVpc()
	if vpc != nil {
		extra.Add(jsonutils.NewString(vpc.Name), "vpc")
	}
	return extra
}

func (manager *SSecurityGroupCacheManager) GetSecgroupCache(ctx context.Context, userCred mcclient.TokenCredential, secgroupId, vpcId string, regionId string, providerId string) (*SSecurityGroupCache, error) {
	secgroupCache := SSecurityGroupCache{}
	query := manager.Query()
	cond := sqlchemy.AND(sqlchemy.Equals(query.Field("secgroup_id"), secgroupId), sqlchemy.Equals(query.Field("vpc_id"), vpcId), sqlchemy.Equals(query.Field("cloudregion_id"), regionId), sqlchemy.Equals(query.Field("manager_id"), providerId))
	query = query.Filter(cond)

	count, err := query.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	query.First(&secgroupCache)
	secgroupCache.SetModelManager(manager, &secgroupCache)
	return &secgroupCache, nil
}

func (manager *SSecurityGroupCacheManager) NewCache(ctx context.Context, userCred mcclient.TokenCredential, secgroupId, vpcId, regionId string, providerId string) (*SSecurityGroupCache, error) {
	lockman.LockClass(ctx, manager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, manager, userCred.GetProjectId())

	secgroup, err := SecurityGroupManager.FetchById(secgroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "SecurityGroupManager.FetchById(%s)", secgroupId)
	}

	secgroupCache := &SSecurityGroupCache{
		SecgroupId: secgroupId,
		VpcId:      vpcId,
	}
	secgroupCache.ManagerId = providerId
	secgroupCache.Status = api.SECGROUP_CACHE_STATUS_CACHING
	secgroupCache.CloudregionId = regionId
	secgroupCache.Name = secgroup.GetName()
	secgroupCache.SetModelManager(manager, secgroupCache)
	if err := manager.TableSpec().Insert(secgroupCache); err != nil {
		log.Errorf("insert secgroupcache error: %v", err)
		return nil, err
	}
	return secgroupCache, nil
}

func (manager *SSecurityGroupCacheManager) Register(ctx context.Context, userCred mcclient.TokenCredential, secgroupId, vpcId, regionId string, providerId string) (*SSecurityGroupCache, error) {
	secgroupCache, err := manager.GetSecgroupCache(ctx, userCred, secgroupId, vpcId, regionId, providerId)
	if err != nil {
		return nil, err
	}

	if secgroupCache != nil {
		return secgroupCache, nil
	}

	return manager.NewCache(ctx, userCred, secgroupId, vpcId, regionId, providerId)
}

func (manager *SSecurityGroupCacheManager) getSecgroupcachesByProvider(provider *SCloudprovider, region *SCloudregion, vpcId string) ([]SSecurityGroupCache, error) {
	q := manager.Query().Equals("manager_id", provider.Id)
	if region != nil {
		q = q.Equals("cloudregion_id", region.Id)
	}
	if len(vpcId) > 0 {
		q = q.Equals("vpc_id", vpcId)
	}
	caches := []SSecurityGroupCache{}
	if err := db.FetchModelObjects(manager, q, &caches); err != nil {
		return nil, err
	}
	return caches, nil
}

func (self *SSecurityGroupCache) GetSecgroup() (*SSecurityGroup, error) {
	model, err := SecurityGroupManager.FetchById(self.SecgroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secgroup by %s", self.SecgroupId)
	}
	return model.(*SSecurityGroup), nil
}

func (manager *SSecurityGroupCacheManager) SyncSecurityGroupCaches(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, secgroups []cloudprovider.ICloudSecurityGroup, vpc *SVpc) ([]SSecurityGroup, []cloudprovider.ICloudSecurityGroup, compare.SyncResult) {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	localSecgroups := []SSecurityGroup{}
	remoteSecgroups := []cloudprovider.ICloudSecurityGroup{}
	syncResult := compare.SyncResult{}

	region, err := vpc.GetRegion()
	if err != nil {
		syncResult.Error(err)
		return localSecgroups, remoteSecgroups, syncResult
	}

	vpcId := ""
	if region.GetDriver().IsSecurityGroupBelongGlobalNetwork() { //globalnetwork没有region属性
		vpcId, err = region.GetDriver().GetSecurityGroupVpcId(ctx, userCred, region, nil, vpc, false)
		if err != nil {
			syncResult.Error(errors.Wrap(err, "GetSecurityGroupVpcId"))
			return localSecgroups, remoteSecgroups, syncResult
		}
		region = nil
	} else if region.GetDriver().IsSecurityGroupBelongVpc() {
		vpcId = vpc.ExternalId
	} else if region.GetDriver().IsSupportClassicSecurityGroup() && len(secgroups) > 0 {
		vpcId, err = region.GetDriver().GetSecurityGroupVpcId(ctx, userCred, region, nil, vpc, secgroups[0].GetVpcId() == "classic")
		if err != nil {
			syncResult.Error(errors.Wrap(err, "GetSecurityGroupVpcId"))
			return localSecgroups, remoteSecgroups, syncResult
		}
	} else {
		vpcId = region.GetDriver().GetDefaultSecurityGroupVpcId()
	}

	dbSecgroupcaches, err := manager.getSecgroupcachesByProvider(provider, region, vpcId)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := []SSecurityGroupCache{}
	commondb := []SSecurityGroupCache{}
	commonext := []cloudprovider.ICloudSecurityGroup{}
	added := []cloudprovider.ICloudSecurityGroup{}

	if err := compare.CompareSets(dbSecgroupcaches, secgroups, &removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i++ {
		_, err = db.Update(&commondb[i], func() error {
			commondb[i].Status = api.SECGROUP_CACHE_STATUS_READY
			commondb[i].Name = commonext[i].GetName()
			commondb[i].Description = commonext[i].GetDescription()
			return nil
		})
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}

	//相同的不能同步, 原因: 多个平台的安全组可能共用一个本地安全组,下面仅仅是新加的安全组
	for i := 0; i < len(added); i++ {
		secgroup, err := SecurityGroupManager.newFromCloudSecgroup(ctx, userCred, provider, added[i])
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		cache, err := manager.NewCache(ctx, userCred, secgroup.Id, vpcId, vpc.CloudregionId, provider.Id)
		if err != nil {
			syncResult.AddError(fmt.Errorf("failed to create secgroup cache for secgroup %s(%s) provider: %s: %s", secgroup.Name, secgroup.Name, provider.Name, err))
			continue
		}
		_, err = db.Update(cache, func() error {
			cache.Status = api.SECGROUP_CACHE_STATUS_READY
			cache.Name = added[i].GetName()
			cache.Description = added[i].GetDescription()
			cache.ExternalId = added[i].GetGlobalId()
			return nil
		})
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		localSecgroups = append(localSecgroups, *secgroup)
		remoteSecgroups = append(remoteSecgroups, added[i])
		syncResult.Add()
	}
	return localSecgroups, remoteSecgroups, syncResult
}

func (self *SSecurityGroupCache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("do nothing for delete secgroup cache")
	return nil
}

func (self *SSecurityGroupCache) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SSecurityGroupCache) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartSecurityGroupCacheDeleteTask(ctx, userCred, "")
}

func (self *SSecurityGroupCache) StartSecurityGroupCacheDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	self.SetStatus(userCred, api.SECGROUP_CACHE_STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SecurityGroupCacheDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}
