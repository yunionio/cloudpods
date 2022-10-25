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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rand"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSecurityGroupCacheManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SVpcResourceBaseManager
	SSecurityGroupResourceBaseManager
}

type SSecurityGroupCache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase
	SManagedResourceBase
	SSecurityGroupResourceBase

	// 被其他安全组引用的次数
	ReferenceCount int `nullable:"false" list:"user" json:"reference_count"`

	// 虚拟私有网络外部Id
	VpcId             string `width:"128" charset:"ascii" list:"user" create:"required"`
	ExternalProjectId string `width:"128" charset:"ascii" list:"user" create:"optional"`
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

func (self *SSecurityGroupCache) GetOwnerId() mcclient.IIdentityProvider {
	sec, err := self.GetSecgroup()
	if err != nil {
		return &db.SOwnerId{}
	}
	return &db.SOwnerId{DomainId: sec.DomainId, ProjectId: sec.ProjectId}
}

func (manager *SSecurityGroupCacheManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeProject
}

// 安全组缓存列表
func (manager *SSecurityGroupCacheManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SecurityGroupCacheListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SSecurityGroupCacheManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SecurityGroupCacheListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SSecurityGroupCacheManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSecurityGroupResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SSecurityGroupCache) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, err
	}
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (manager *SSecurityGroupCacheManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if userCred != nil {
		sq := SecurityGroupManager.Query("id")
		switch scope {
		case rbacutils.ScopeProject:
			if len(userCred.GetProjectId()) > 0 {
				sq = sq.Equals("tenant_id", userCred.GetProjectId())
				return q.In("secgroup_id", sq)
			}
		case rbacutils.ScopeDomain:
			if len(userCred.GetProjectDomainId()) > 0 {
				sq = sq.Equals("domain_id", userCred.GetProjectDomainId())
				return q.In("secgroup_id", sq)
			}
		}
	}
	return q
}

func (self *SSecurityGroupCache) GetVpc() (*SVpc, error) {
	vpc, err := VpcManager.FetchById(self.VpcId)
	if err != nil {
		return nil, err
	}
	return vpc.(*SVpc), nil
}

func (manager *SSecurityGroupCacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SecurityGroupCacheDetails {
	rows := make([]api.SecurityGroupCacheDetails, len(objs))

	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	cacheIds := make([]string, len(objs))
	secIds := make([]string, len(objs))
	vpcIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.SecurityGroupCacheDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ManagedResourceInfo:             manRows[i],
			CloudregionResourceInfo:         regRows[i],
		}
		cache := objs[i].(*SSecurityGroupCache)
		cacheIds[i] = cache.Id
		vpcIds[i] = cache.VpcId
		secIds[i] = cache.SecgroupId
	}
	vpcMaps, _ := db.FetchIdNameMap2(VpcManager, vpcIds)
	for i := range rows {
		rows[i].Vpc = vpcMaps[vpcIds[i]]
	}

	secgroups := make(map[string]SSecurityGroup)
	err := db.FetchStandaloneObjectsByIds(SecurityGroupManager, secIds, &secgroups)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds fail: %v", err)
		return rows
	}

	virObjs := make([]interface{}, len(objs))
	for i := range rows {
		if secgroup, ok := secgroups[secIds[i]]; ok {
			virObjs[i] = &secgroup
			rows[i].ProjectId = secgroup.ProjectId
		}
	}

	projRows := SecurityGroupManager.SProjectizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, virObjs, fields, isList)
	for i := range rows {
		rows[i].ProjectizedResourceInfo = projRows[i]
	}

	return rows
}

func (manager *SSecurityGroupCacheManager) GetSecgroupCache(ctx context.Context, userCred mcclient.TokenCredential, secgroupId, vpcId string, regionId string, providerId string, projectId string) (*SSecurityGroupCache, error) {
	secgroupCache := SSecurityGroupCache{}
	query := manager.Query()
	conds := []sqlchemy.ICondition{
		sqlchemy.Equals(query.Field("secgroup_id"), secgroupId),
		sqlchemy.Equals(query.Field("vpc_id"), vpcId),
		sqlchemy.Equals(query.Field("manager_id"), providerId),
	}
	if len(projectId) > 0 {
		conds = append(conds, sqlchemy.Equals(query.Field("external_project_id"), projectId))
	}
	_region, err := CloudregionManager.FetchById(regionId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudregionManager.FetchById(%s)", regionId)
	}
	region := _region.(*SCloudregion)
	if !region.GetDriver().IsSecurityGroupBelongGlobalVpc() {
		conds = append(conds, sqlchemy.Equals(query.Field("cloudregion_id"), regionId))
	}
	query = query.Filter(sqlchemy.AND(conds...))

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

func (manager *SSecurityGroupCacheManager) NewCache(ctx context.Context, userCred mcclient.TokenCredential, secgroupId, vpcId, regionId string, providerId string, projectId string) (*SSecurityGroupCache, error) {
	lockman.LockClass(ctx, manager, userCred.GetProjectId())
	defer lockman.ReleaseClass(ctx, manager, userCred.GetProjectId())

	secgroup, err := SecurityGroupManager.FetchById(secgroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "SecurityGroupManager.FetchById(%s)", secgroupId)
	}

	return manager.newCache(ctx, secgroupId, secgroup.GetName(), vpcId, regionId, providerId, projectId)
}

func (manager *SSecurityGroupCacheManager) newCache(ctx context.Context, secgroupId, secgroupName, vpcId, regionId string, providerId string, projectId string) (*SSecurityGroupCache, error) {
	secgroupCache := &SSecurityGroupCache{}
	secgroupCache.SecgroupId = secgroupId
	secgroupCache.VpcId = vpcId
	secgroupCache.ManagerId = providerId
	secgroupCache.Status = api.SECGROUP_CACHE_STATUS_CACHING
	secgroupCache.CloudregionId = regionId
	secgroupCache.Name = secgroupName
	secgroupCache.ExternalProjectId = projectId
	secgroupCache.SetModelManager(manager, secgroupCache)
	err := manager.TableSpec().Insert(ctx, secgroupCache)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}
	return secgroupCache, nil
}

func (manager *SSecurityGroupCacheManager) Register(ctx context.Context, userCred mcclient.TokenCredential, secgroupId, vpcId, regionId string, providerId string, projectId string) (*SSecurityGroupCache, error) {
	secgroupCache, err := manager.GetSecgroupCache(ctx, userCred, secgroupId, vpcId, regionId, providerId, projectId)
	if err != nil {
		return nil, err
	}

	if secgroupCache != nil {
		return secgroupCache, nil
	}

	return manager.NewCache(ctx, userCred, secgroupId, vpcId, regionId, providerId, projectId)
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
		return nil, errors.Wrapf(err, "SecurityGroupManager.FetchById(%s)", self.SecgroupId)
	}
	return model.(*SSecurityGroup), nil
}

func (self *SSecurityGroupCache) SyncBaseInfo(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudSecurityGroup) error {
	_, err := db.Update(self, func() error {
		self.Status = api.SECGROUP_CACHE_STATUS_READY
		self.Name = ext.GetName()
		self.Description = ext.GetDescription()
		self.ExternalProjectId = ext.GetProjectId()
		references, err := ext.GetReferences()
		if err == nil {
			self.ReferenceCount = len(references)
		}
		if createdAt := ext.GetCreatedAt(); !createdAt.IsZero() {
			self.CreatedAt = createdAt
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return nil
}

func (self *SSecurityGroupCache) syncWithCloudSecurityGroup(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudSecurityGroup) ([]SSecurityGroupRule, error) {
	err := self.SyncBaseInfo(ctx, userCred, ext)
	if err != nil {
		return nil, errors.Wrapf(err, "db.Update")
	}
	secgroup, err := self.GetSecgroup()
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecurity")
	}
	cacheCount, err := secgroup.GetSecgroupCacheCount()
	if err != nil {
		return nil, errors.Wrapf(err, "GetSecgroupCacheCount")
	}
	if cacheCount > 1 {
		return nil, nil
	}
	dest := cloudprovider.NewSecRuleInfo(GetRegionDriver(provider.Provider))
	dest.Rules, err = ext.GetRules()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRules")
	}
	rules, _ := secgroup.SyncSecurityGroupRules(ctx, userCred, dest)
	return rules, nil
}

func (manager *SSecurityGroupCacheManager) SyncSecurityGroupCaches(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, secgroups []cloudprovider.ICloudSecurityGroup, vpc *SVpc) ([]SSecurityGroup, []cloudprovider.ICloudSecurityGroup, compare.SyncResult) {
	lockman.LockRawObject(ctx, "secgroups", vpc.Id)
	defer lockman.ReleaseRawObject(ctx, "secgroups", vpc.Id)

	localSecgroups := []SSecurityGroup{}
	remoteSecgroups := []cloudprovider.ICloudSecurityGroup{}
	syncResult := compare.SyncResult{}

	region, err := vpc.GetRegion()
	if err != nil {
		syncResult.Error(err)
		return localSecgroups, remoteSecgroups, syncResult
	}

	vpcId := ""
	if region.GetDriver().IsSecurityGroupBelongGlobalVpc() {
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
			notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
				Obj:    &removed[i],
				Action: notifyclient.ActionSyncDelete,
			})
		}
	}

	rules := []SSecurityGroupRule{}

	for i := 0; i < len(commondb); i++ {
		_rules, err := commondb[i].syncWithCloudSecurityGroup(ctx, userCred, provider, commonext[i])
		if err != nil {
			syncResult.UpdateError(errors.Wrapf(err, "syncWithCloudSecurityGroup"))
			continue
		}
		rules = append(rules, _rules...)
		syncResult.Update()
	}

	for i := 0; i < len(added); i++ {
		secgroup, _rules, err := SecurityGroupManager.newFromCloudSecgroup(ctx, userCred, provider, added[i])
		if err != nil {
			syncResult.AddError(errors.Wrapf(err, "newFromCloudSecgroup"))
			continue
		}
		rules = append(rules, _rules...)
		if secgroup.ProjectId != provider.ProjectId {
			_, err = secgroup.PerformPublic(ctx, userCred, nil,
				apis.PerformPublicProjectInput{
					PerformPublicDomainInput: apis.PerformPublicDomainInput{
						Scope:           "domain",
						SharedDomainIds: []string{provider.DomainId},
					},
				})
			if err != nil {
				log.Warningf("failed to set secgroup %s(%s) project sharable", secgroup.Name, secgroup.Id)
			}
		}
		cache, err := manager.NewCache(ctx, userCred, secgroup.Id, vpcId, vpc.CloudregionId, provider.Id, added[i].GetProjectId())
		if err != nil {
			syncResult.AddError(errors.Wrapf(err, "NewCache for secgroup %s provider %s", secgroup.Name, provider.Name))
			continue
		}
		_, err = db.Update(cache, func() error {
			cache.Status = api.SECGROUP_CACHE_STATUS_READY
			cache.Name = added[i].GetName()
			cache.Description = added[i].GetDescription()
			cache.ExternalId = added[i].GetGlobalId()
			references, _ := added[i].GetReferences()
			cache.ReferenceCount = len(references)

			if createdAt := added[i].GetCreatedAt(); !createdAt.IsZero() {
				cache.CreatedAt = createdAt
			}

			return nil
		})
		if err != nil {
			syncResult.AddError(errors.Wrapf(err, "db.Update"))
			continue
		}
		localSecgroups = append(localSecgroups, *secgroup)
		remoteSecgroups = append(remoteSecgroups, added[i])
		syncResult.Add()
	}
	for i := range rules {
		if len(rules[i].PeerSecgroupId) > 0 {
			cache, _ := db.FetchByExternalId(SecurityGroupCacheManager, rules[i].PeerSecgroupId)
			if cache != nil {
				db.Update(&rules[i], func() error {
					rules[i].PeerSecgroupId = cache.(*SSecurityGroupCache).SecgroupId
					return nil
				})
			}
		}
	}
	return localSecgroups, remoteSecgroups, syncResult
}

// 同步安全组缓存状态
func (self *SSecurityGroupCache) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DiskSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, self.StartSyncstatusTask(ctx, userCred, "")
}

// 获取引用信息
func (self *SSecurityGroupCache) GetDetailsReferences(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) ([]cloudprovider.SecurityGroupReference, error) {
	iSecgroup, err := self.GetISecurityGroup(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetISecurityGroup")
	}
	return iSecgroup.GetReferences()
}

func (self *SSecurityGroupCache) StartSyncstatusTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return StartResourceSyncStatusTask(ctx, userCred, self, "SecurityGroupCacheSyncstatusTask", "")
}

func (self *SSecurityGroupCache) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if self.ReferenceCount > 0 && self.Status == api.SECGROUP_CACHE_STATUS_READY {
		return httperrors.NewNotEmptyError("security group has been reference in %d security group", self.ReferenceCount)
	}

	return self.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SSecurityGroupCache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("do nothing for delete secgroup cache")
	return nil
}

func (self *SSecurityGroupCache) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SSecurityGroupCache) purge(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	if secgroup, _ := self.GetSecgroup(); secgroup != nil {
		caches, err := secgroup.GetSecurityGroupCaches()
		if err != nil {
			return errors.Wrapf(err, "secgroup.GetSecurityGroupCaches")
		}
		if len(caches) == 1 {
			err := secgroup.ValidateDeleteCondition(ctx, nil)
			if err == nil {
				secgroup.RealDelete(ctx, userCred)
			}
		}
	}

	return self.RealDelete(ctx, userCred)
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

func (manager *SSecurityGroupCacheManager) InitializeData() error {
	providerIds := CloudproviderManager.Query("id").In("provider", []string{api.CLOUD_PROVIDER_HUAWEI, api.CLOUD_PROVIDER_HCSO, api.CLOUD_PROVIDER_HCS, api.CLOUD_PROVIDER_CTYUN, api.CLOUD_PROVIDER_QCLOUD}).SubQuery()

	deprecatedSecgroups := []SSecurityGroupCache{}
	q := manager.Query().In("manager_id", providerIds).NotEquals("vpc_id", api.NORMAL_VPC_ID)
	err := db.FetchModelObjects(manager, q, &deprecatedSecgroups)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "SSecurityGroupCacheManager.InitializeData.Query")
	}

	for i := range deprecatedSecgroups {
		cache := &deprecatedSecgroups[i]
		_, err := db.Update(cache, func() error {
			return cache.MarkDelete()
		})
		if err != nil {
			return errors.Wrap(err, "SSecurityGroupCacheManager.InitializeData.Query")
		}
	}

	log.Debugf("SSecurityGroupCacheManager cleaned %d deprecated security group cache.", len(deprecatedSecgroups))
	return nil
}

func (manager *SSecurityGroupCacheManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}

	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SVpcResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
		}
	}

	if keys.ContainsAny(manager.SSecurityGroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SSecurityGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemExportKey")
		}
	}

	return q, nil
}

func (self *SSecurityGroupCache) GetISecurityGroup(ctx context.Context) (cloudprovider.ICloudSecurityGroup, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}

	manager := self.GetCloudprovider()
	if manager == nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "failed to found manager")
	}

	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	return iRegion.GetISecurityGroupById(self.ExternalId)
}

func (self *SSecurityGroupCache) GetOrCreateISecurityGroup(ctx context.Context) (cloudprovider.ICloudSecurityGroup, bool, error) {
	secgroup, err := self.GetISecurityGroup(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			secgroup, err = self.CreateISecurityGroup(ctx)
			if err != nil {
				return nil, false, errors.Wrapf(err, "CreateISecurityGroup")
			}
			return secgroup, true, nil
		}
		return nil, false, errors.Wrapf(err, "GetIISecurityGroup")
	}
	return secgroup, false, nil
}

func (self *SSecurityGroupCache) CreateISecurityGroup(ctx context.Context) (cloudprovider.ICloudSecurityGroup, error) {
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetIRegion")
	}

	regionDriver, err := self.GetRegionDriver()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegionDriver")
	}

	self.Name = regionDriver.GenerateSecurityGroupName(self.Name)

	// 避免有的云不支持重名安全组
	if !regionDriver.IsAllowSecurityGroupNameRepeat() {
		randomString := func(prefix string, length int) string {
			return fmt.Sprintf("%s-%s", prefix, rand.String(length))
		}
		opts := &cloudprovider.SecurityGroupFilterOptions{
			Name:      randomString(self.Name, 1),
			VpcId:     self.VpcId,
			ProjectId: self.ExternalProjectId,
		}
		for i := 2; i < 30; i++ {
			_, err := iRegion.GetISecurityGroupByName(opts)
			if err != nil {
				if errors.Cause(err) == cloudprovider.ErrNotFound {
					break
				}
				if errors.Cause(err) != cloudprovider.ErrDuplicateId {
					return nil, errors.Wrapf(err, "GetISecurityGroupByName")
				}
			}
			opts.Name = randomString(self.Name, i)
		}
		self.Name = opts.Name
	}

	conf := &cloudprovider.SecurityGroupCreateInput{
		Name:      self.Name,
		Desc:      self.Description,
		VpcId:     self.VpcId,
		ProjectId: self.ExternalProjectId,
	}
	iSecgroup, err := iRegion.CreateISecurityGroup(conf)
	if err != nil {
		return nil, errors.Wrapf(err, "iRegion.CreateISecurityGroup")
	}
	_, err = db.Update(self, func() error {
		self.ExternalId = iSecgroup.GetGlobalId()
		self.Name = iSecgroup.GetName()
		self.Status = api.SECGROUP_CACHE_STATUS_READY
		return nil
	})
	return iSecgroup, nil
}

func (self *SSecurityGroupCache) GetSecuritRuleSet(ctx context.Context) (cloudprovider.SecurityRuleSet, []SSecurityGroupCache, error) {
	secgroup, err := self.GetSecgroup()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "GetSecgroup")
	}
	rules, err := secgroup.getSecurityRules()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "getSecurityRules")
	}
	return self.convertRules(ctx, rules)
}

func (self *SSecurityGroupCache) GetOldSecuritRuleSet(ctx context.Context) (cloudprovider.SecurityRuleSet, []SSecurityGroupCache, error) {
	secgroup, err := self.GetSecgroup()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "GetSecgroup")
	}
	rules, err := secgroup.GetOldRules()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "GetOldRules")
	}
	return self.convertRules(ctx, rules)
}

func (self *SSecurityGroupCache) convertRules(ctx context.Context, rules []SSecurityGroupRule) (cloudprovider.SecurityRuleSet, []SSecurityGroupCache, error) {
	ruleSet := cloudprovider.SecurityRuleSet{}
	caches := []SSecurityGroupCache{}
	driver := GetRegionDriver(self.GetProviderName())
	for i := range rules {
		if !driver.IsSupportPeerSecgroup() && len(rules[i].PeerSecgroupId) > 0 {
			continue
		}
		//这里没必要拆分为单个单个的端口,到公有云那边适配
		rule, err := rules[i].toRule()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "toRule")
		}
		peerId := ""
		if len(rules[i].PeerSecgroupId) > 0 {
			_peerSecgroup, err := SecurityGroupManager.FetchById(rules[i].PeerSecgroupId)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "SecurityGroupManager.FetchById(%s)", rules[i].PeerSecgroupId)
			}
			peerSecgroup := _peerSecgroup.(*SSecurityGroup)
			peerCaches, err := peerSecgroup.GetSecurityGroupCaches()
			if err != nil {
				return nil, nil, errors.Wrapf(err, "peerSecgroup.GetSecurityGroupCaches")
			}

			for _, cache := range peerCaches {
				if cache.ManagerId == self.ManagerId && cache.VpcId == self.VpcId && len(cache.ExternalId) > 0 && (!driver.IsPeerSecgroupWithSameProject() || cache.ExternalProjectId == self.ExternalProjectId) {
					peerId = cache.ExternalId
					break
				}
			}

			if len(peerId) == 0 {
				cache, err := SecurityGroupCacheManager.newCache(context.TODO(), peerSecgroup.Id, peerSecgroup.Name, self.VpcId, self.CloudregionId, self.ManagerId, self.ExternalProjectId)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "SecurityGroupCacheManager.newCache")
				}
				iSecgroup, err := cache.CreateISecurityGroup(ctx)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "cache.CreateISecurityGroup")
				}
				peerId = iSecgroup.GetGlobalId()
				caches = append(caches, *cache)
			}
		}
		ruleSet = append(ruleSet, cloudprovider.SecurityRule{SecurityRule: *rule, ExternalId: rules[i].Id, PeerSecgroupId: peerId})
	}
	return ruleSet, caches, nil
}

func (self *SSecurityGroupCache) SyncRules(ctx context.Context, skipSyncRule bool) error {
	region, err := self.GetRegion()
	if err != nil {
		return err
	}
	iSecgroup, err := self.GetISecurityGroup(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetISecurityGroup")
	}
	if skipSyncRule {
		return nil
	}

	rules, err := iSecgroup.GetRules()
	if err != nil {
		return errors.Wrapf(err, "iSecgroup.GetRules")
	}

	localRules, caches, err := self.GetSecuritRuleSet(ctx)
	if err != nil {
		return errors.Wrapf(err, "GetSecuritRuleSet")
	}

	src := cloudprovider.NewSecRuleInfo(GetRegionDriver(api.CLOUD_PROVIDER_ONECLOUD))
	src.Rules = localRules

	dest := cloudprovider.NewSecRuleInfo(GetRegionDriver(region.Provider))
	dest.Rules = rules

	common, inAdds, outAdds, inDels, outDels := cloudprovider.CompareRules(src, dest, false)

	if len(inAdds) == 0 && len(inDels) == 0 && len(outAdds) == 0 && len(outDels) == 0 {
		return nil
	}
	if len(inDels)+len(outDels) > 10 {
		return fmt.Errorf("too many rules operations with %s inDel:%d outDel:%d inAdd:%d outAdd: %d", self.Name, len(inDels), len(outDels), len(inAdds), len(outAdds))
	}
	err = iSecgroup.SyncRules(common, inAdds, outAdds, inDels, outDels)
	if err != nil {
		return errors.Wrapf(err, "iSecgroup.SyncRules")
	}
	for i := range caches {
		err = caches[i].SyncRules(ctx, skipSyncRule)
		if err != nil {
			return errors.Wrapf(err, "SyncRules for caches %s(%s)", caches[i].Name, caches[i].Id)
		}
	}
	return nil
}
