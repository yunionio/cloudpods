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
	"fmt"

	"yunion.io/x/jsonutils"
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
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SAccessGroupCacheManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
	SAccessGroupResourceBaseManager
}

var AccessGroupCacheManager *SAccessGroupCacheManager

func init() {
	AccessGroupCacheManager = &SAccessGroupCacheManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SAccessGroupCache{},
			"access_group_caches_tbl",
			"access_group_cache",
			"access_group_caches",
		),
	}
	AccessGroupCacheManager.SetVirtualObject(AccessGroupCacheManager)
}

type SAccessGroupCache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase
	SManagedResourceBase
	SAccessGroupResourceBase

	// 已关联的挂载点数量
	MountTargetCount int `nullable:"false" list:"user" json:"mount_target_count"`

	FileSystemType string `width:"16" charset:"ascii" nullable:"false" index:"true" list:"user"`
	NetworkType    string `width:"8" charset:"ascii" nullable:"false" index:"true" list:"user" default:"vpc"`
}

func (manager *SAccessGroupCacheManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (manager *SAccessGroupCacheManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowList(userCred, manager)
}

func (self *SAccessGroupCache) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	group, err := self.GetAccessGroup()
	if err != nil {
		return false
	}
	return group.AllowUpdateItem(ctx, userCred)
}

func (manager *SAccessGroupCacheManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AccessGroupCacheListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SStatusStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SManagedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SAccessGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.AccessGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SAccessGroupResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SAccessGroupCacheManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AccessGroupCacheListInput,
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
	q, err = manager.SAccessGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.AccessGroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SAccessGroupResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SAccessGroupCacheManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SAccessGroupResourceBaseManager.QueryDistinctExtraField(q, field)
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

func (self *SAccessGroupCache) GetOwnerId() mcclient.IIdentityProvider {
	ag, err := self.GetAccessGroup()
	if err != nil {
		return &db.SOwnerId{}
	}
	return &db.SOwnerId{DomainId: ag.DomainId}
}

func (manager *SAccessGroupCacheManager) FilterByOwner(q *sqlchemy.SQuery, userCred mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if userCred != nil {
		sq := AccessGroupManager.Query("id")
		if scope == rbacutils.ScopeDomain && len(userCred.GetProjectDomainId()) > 0 {
			sq = sq.Equals("domain_id", userCred.GetProjectDomainId())
			return q.In("access_group_id", sq)
		}
	}
	return q
}

func (manager *SAccessGroupCacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AccessGroupCacheDetails {
	rows := make([]api.AccessGroupCacheDetails, len(objs))

	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	manRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	groupRows := manager.SAccessGroupResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.AccessGroupCacheDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			ManagedResourceInfo:             manRows[i],
			CloudregionResourceInfo:         regRows[i],
			AccessGroupResourceInfo:         groupRows[i],
		}
	}

	return rows
}

func (manager *SAccessGroupCacheManager) ListItemExportKeys(ctx context.Context,
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

	if keys.ContainsAny(manager.SAccessGroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SAccessGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SAccessGroupResourceBaseManager.ListItemExportKey")
		}
	}

	return q, nil
}

func (self *SCloudregion) GetAccessGroupCaches() ([]SAccessGroupCache, error) {
	caches := []SAccessGroupCache{}
	q := AccessGroupCacheManager.Query().Equals("cloudregion_id", self.Id)
	err := db.FetchModelObjects(AccessGroupCacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SAccessGroupCache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SAccessGroupCache) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SCloudregion) SyncAccessGroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, iAccessGroups []cloudprovider.ICloudAccessGroup) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Id, "access_groups")
	defer lockman.ReleaseRawObject(ctx, self.Id, "access_groups")

	result := compare.SyncResult{}

	dbCaches, err := self.GetAccessGroupCaches()
	if err != nil {
		result.Error(errors.Wrapf(err, "self.GetAccessGroupCaches"))
		return result
	}

	removed := make([]SAccessGroupCache, 0)
	commondb := make([]SAccessGroupCache, 0)
	commonext := make([]cloudprovider.ICloudAccessGroup, 0)
	added := make([]cloudprovider.ICloudAccessGroup, 0)
	err = compare.CompareSets(dbCaches, iAccessGroups, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(errors.Wrapf(err, "compare.CompareSets"))
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].RealDelete(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithAccessGroup(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		err := provider.newFromCloudAccessGroup(ctx, userCred, self, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SCloudprovider) GetDomainAccessGroups() ([]SAccessGroup, error) {
	groups := []SAccessGroup{}
	q := AccessGroupManager.Query().Equals("domain_id", self.DomainId).NotEquals("id", api.DEFAULT_ACCESS_GROUP)
	err := db.FetchModelObjects(AccessGroupManager, q, &groups)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return groups, nil
}

func (self *SAccessGroupCache) init(iAccessGroup cloudprovider.ICloudAccessGroup) {
	self.Name = iAccessGroup.GetName()
	self.NetworkType = iAccessGroup.GetNetworkType()
	self.FileSystemType = iAccessGroup.GetFileSystemType()
	self.MountTargetCount = iAccessGroup.GetMountTargetCount()
	self.ExternalId = iAccessGroup.GetGlobalId()
	self.Description = iAccessGroup.GetDesc()
	self.Status = api.ACCESS_GROUP_STATUS_AVAILABLE
}

func (self *SCloudprovider) newFromCloudAccessGroup(ctx context.Context, userCred mcclient.TokenCredential, region *SCloudregion, iAccessGroup cloudprovider.ICloudAccessGroup) error {
	if iAccessGroup.IsDefault() {
		cache := &SAccessGroupCache{}
		cache.SetModelManager(AccessGroupCacheManager, cache)
		cache.CloudregionId = region.Id
		cache.ManagerId = self.Id
		cache.AccessGroupId = api.DEFAULT_ACCESS_GROUP
		cache.init(iAccessGroup)
		return AccessGroupCacheManager.TableSpec().Insert(ctx, cache)
	}
	groups, err := self.GetDomainAccessGroups()
	if err != nil {
		return errors.Wrapf(err, "GetDomainAccessGroups")
	}
	src, err := cloudprovider.GetAccessGroupRuleInfo(iAccessGroup)
	if err != nil {
		return errors.Wrapf(err, "GetAccessGroupRuleInfo")
	}
	for i := range groups {
		dest, err := groups[i].GetAccessGroupRuleInfo()
		if err != nil {
			continue
		}
		_, added, removed := cloudprovider.CompareAccessGroupRules(src, dest, false)
		if len(added)+len(removed) == 0 {
			cache := &SAccessGroupCache{}
			cache.SetModelManager(AccessGroupCacheManager, cache)
			cache.Name = iAccessGroup.GetName()
			cache.AccessGroupId = groups[i].Id
			cache.ManagerId = self.Id
			cache.CloudregionId = region.Id
			cache.init(iAccessGroup)
			return AccessGroupCacheManager.TableSpec().Insert(ctx, cache)
		}
	}
	group := &SAccessGroup{}
	group.SetModelManager(AccessGroupManager, group)
	group.Status = api.ACCESS_GROUP_STATUS_AVAILABLE
	group.DomainId = self.DomainId

	err = func() error {
		lockman.LockRawObject(ctx, AccessGroupManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, AccessGroupManager.Keyword(), "name")

		group.Name, err = db.GenerateName(ctx, AccessGroupManager, self.GetOwnerId(), iAccessGroup.GetName())
		if err != nil {
			return errors.Wrapf(err, "db.GenerateName")
		}
		return AccessGroupManager.TableSpec().Insert(ctx, group)
	}()
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}
	return group.SyncRules(ctx, userCred, src)
}

func (self *SAccessGroup) GetAccessGroupCaches() ([]SAccessGroupCache, error) {
	q := AccessGroupCacheManager.Query().Equals("access_group_id", self.Id)
	caches := []SAccessGroupCache{}
	err := db.FetchModelObjects(AccessGroupCacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SAccessGroup) removeRules(ctx context.Context, ruleIds []string) error {
	rules := []SAccessGroupRule{}
	q := AccessGroupRuleManager.Query().Equals("access_group_id", self.Id).In("id", ruleIds)
	err := db.FetchModelObjects(AccessGroupRuleManager, q, &rules)
	if err != nil {
		return errors.Wrapf(err, "db.FetchModelObjects")
	}
	for i := range rules {
		rules[i].Delete(ctx, nil)
	}
	return nil
}

func (self *SAccessGroup) newAccessGroupRule(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.AccessGroupRule) error {
	rule := &SAccessGroupRule{}
	rule.SetModelManager(AccessGroupRuleManager, rule)
	rule.AccessGroupId = self.Id
	rule.Source = ext.Source
	rule.RWAccessType = string(ext.RWAccessType)
	rule.UserAccessType = string(ext.UserAccessType)
	rule.Priority = ext.Priority
	return AccessGroupRuleManager.TableSpec().Insert(ctx, rule)
}

func (self *SAccessGroup) SyncRules(ctx context.Context, userCred mcclient.TokenCredential, src cloudprovider.AccessGroupRuleInfo) error {
	dest, err := self.GetAccessGroupRuleInfo()
	if err != nil {
		return errors.Wrapf(err, "GetAccessGroupRuleInfo")
	}
	_, added, removed := cloudprovider.CompareAccessGroupRules(src, dest, false)
	removeIds := []string{}
	for i := range removed {
		if len(removed[i].ExternalId) > 0 {
			removeIds = append(removeIds, removed[i].ExternalId)
		}
	}
	self.removeRules(ctx, removeIds)
	for i := range added {
		self.newAccessGroupRule(ctx, userCred, added[i])
	}
	return nil
}

func (self *SAccessGroupCache) syncAccessGroupBaseInfo(ctx context.Context, userCred mcclient.TokenCredential, iAccessGroup cloudprovider.ICloudAccessGroup) error {
	_, err := db.Update(self, func() error {
		self.init(iAccessGroup)
		return nil
	})
	return errors.Wrapf(err, "db.Update")
}

func (self *SAccessGroupCache) syncWithAccessGroup(ctx context.Context, userCred mcclient.TokenCredential, iAccessGroup cloudprovider.ICloudAccessGroup) error {
	err := self.syncAccessGroupBaseInfo(ctx, userCred, iAccessGroup)
	if err != nil {
		return errors.Wrapf(err, "syncAccessGroupBaseInfo")
	}
	if self.AccessGroupId == api.DEFAULT_ACCESS_GROUP {
		return nil
	}
	group, err := self.GetAccessGroup()
	if err != nil {
		return errors.Wrapf(err, "GetAccessGroup")
	}
	caches, err := group.GetAccessGroupCaches()
	if err != nil {
		return errors.Wrapf(err, "GetAccessGroupCaches")
	}
	if len(caches) > 1 {
		return nil
	}
	src, err := cloudprovider.GetAccessGroupRuleInfo(iAccessGroup)
	if err != nil {
		return errors.Wrapf(err, "cloudprovider.GetAccessGroupRuleInfo")
	}
	return group.SyncRules(ctx, userCred, src)
}

func (self *SAccessGroupCache) GetRegion() (*SCloudregion, error) {
	region, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudregionManager.FetchById(%s)", self.CloudregionId)
	}
	return region.(*SCloudregion), nil
}

func (self *SAccessGroupCache) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetDriver")
	}
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetRegion")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SAccessGroupCache) GetICloudAccessGroup() (cloudprovider.ICloudAccessGroup, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetIRegion")
	}
	iAccessGroup, err := iRegion.GetICloudAccessGroupById(self.ExternalId)
	if err != nil {
		return nil, errors.Wrapf(err, "iRegion.GetICloudAccessGroupById(%s)", self.ExternalId)
	}
	return iAccessGroup, nil
}

func (self *SAccessGroupCache) SyncRules() error {
	group, err := self.GetAccessGroup()
	if err != nil {
		return errors.Wrapf(err, "GetAccessGroup")
	}
	src, err := group.GetAccessGroupRuleInfo()
	if err != nil {
		return errors.Wrapf(err, "GetAccessGroupRuleInfo")
	}

	iAccessGroup, err := self.GetICloudAccessGroup()
	if err != nil {
		return errors.Wrapf(err, "GetICloudAccessGroup")
	}

	dest, err := cloudprovider.GetAccessGroupRuleInfo(iAccessGroup)
	if err != nil {
		return errors.Wrapf(err, "GetAccessGroupRuleInfo")
	}

	common, added, removed := cloudprovider.CompareAccessGroupRules(src, dest, false)
	return iAccessGroup.SyncRules(common, added, removed)
}

func (self *SAccessGroupCache) CreateIAccessGroup() error {
	iRegion, err := self.GetIRegion()
	if err != nil {
		return errors.Wrapf(err, "self.GetIRegion")
	}
	opts := &cloudprovider.SAccessGroup{
		Name:           self.Name,
		NetworkType:    self.NetworkType,
		FileSystemType: self.FileSystemType,
		Desc:           self.Description,
	}
	iAccessGroup, err := iRegion.CreateICloudAccessGroup(opts)
	if err != nil {
		return errors.Wrapf(err, "iRegion.CreateIAccessGroup")
	}
	_, err = db.Update(self, func() error {
		self.ExternalId = iAccessGroup.GetGlobalId()
		self.Status = api.ACCESS_GROUP_STATUS_AVAILABLE
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return self.SyncRules()
}

type SAccessGroupCacheRegisterInput struct {
	Name           string
	Desc           string
	ManagerId      string
	CloudregionId  string
	FileSystemType string
	NetworkType    string
	AccessGroupId  string
}

func (manager *SAccessGroupCacheManager) Register(ctx context.Context, opts *SAccessGroupCacheRegisterInput) (*SAccessGroupCache, error) {
	lockman.LockClass(ctx, manager, "")
	defer lockman.ReleaseClass(ctx, manager, "")

	cache := &SAccessGroupCache{}
	cache.SetModelManager(manager, cache)
	cache.Name = opts.Name
	if opts.NetworkType == api.NETWORK_TYPE_CLASSIC {
		cache.Name = fmt.Sprintf("%s-%s", opts.Name, opts.NetworkType)
	}
	cache.Description = opts.Desc
	cache.Status = api.ACCESS_GROUP_STATUS_CREATING
	cache.ManagerId = opts.ManagerId
	cache.CloudregionId = opts.CloudregionId
	cache.FileSystemType = opts.FileSystemType
	cache.NetworkType = opts.NetworkType
	cache.AccessGroupId = opts.AccessGroupId
	err := manager.TableSpec().Insert(ctx, cache)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}
	return cache, cache.CreateIAccessGroup()
}

func (self *SAccessGroupCache) ValidateDeleteCondition(ctx context.Context) error {
	if self.AccessGroupId == api.DEFAULT_ACCESS_GROUP {
		return httperrors.NewProtectedResourceError("not allow to delete default access group")
	}
	if self.MountTargetCount > 0 && self.Status != api.ACCESS_GROUP_STATUS_UNKNOWN {
		return httperrors.NewNotEmptyError("access group not empty, please delete mount target first")
	}
	return self.SStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SAccessGroupCache) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {

	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SAccessGroupCache) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	var err = func() error {
		task, err := taskman.TaskManager.NewTask(ctx, "AccessGroupCacheDeleteTask", self, userCred, nil, parentTaskId, "", nil)
		if err != nil {
			return errors.Wrapf(err, "NewTask")
		}
		return task.ScheduleRun(nil)
	}()
	if err != nil {
		self.SetStatus(userCred, api.ACCESS_GROUP_STATUS_DELETE_FAILED, err.Error())
		return nil
	}
	self.SetStatus(userCred, api.ACCESS_GROUP_STATUS_DELETING, "")
	return nil
}

func (self *SAccessGroupCache) SyncStatus(ctx context.Context, userCred mcclient.TokenCredential) error {
	iAccessGroup, err := self.GetICloudAccessGroup()
	if err != nil {
		self.SetStatus(userCred, api.ACCESS_GROUP_STATUS_CREATING, err.Error())
		return err
	}
	return self.syncAccessGroupBaseInfo(ctx, userCred, iAccessGroup)
}

func (self *SAccessGroupCache) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, self, "syncstatus")
}

// 同步权限组缓存状态
func (self *SAccessGroupCache) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.MountTargetSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, self.SyncStatus(ctx, userCred)
}
