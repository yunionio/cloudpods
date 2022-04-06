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
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SWafRuleGroupCacheManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var WafRuleGroupCacheManager *SWafRuleGroupCacheManager

func init() {
	WafRuleGroupCacheManager = &SWafRuleGroupCacheManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SWafRuleGroupCache{},
			"waf_rule_group_caches_tbl",
			"waf_rule_group_cache",
			"waf_rule_group_caches",
		),
	}
	WafRuleGroupCacheManager.SetVirtualObject(WafRuleGroupCacheManager)
}

type SWafRuleGroupCache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	Type           cloudprovider.TWafType `width:"20" charset:"utf8" nullable:"false" list:"user"`
	WafRuleGroupId string                 `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SWafRuleGroupCacheManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (manager *SWafRuleGroupCacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WafRuleGroupCacheDetails {
	rows := make([]api.WafRuleGroupCacheDetails, len(objs))
	ssRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.WafRuleGroupCacheDetails{
			StatusStandaloneResourceDetails: ssRows[i],
			ManagedResourceInfo:             managerRows[i],
			CloudregionResourceInfo:         regionRows[i],
		}
	}
	return rows
}

// 列出WAF RuleGroup缓存
func (manager *SWafRuleGroupCacheManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafRuleGroupCacheListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBase.ListItemFilter")
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
	return q, nil
}

func (manager *SWafRuleGroupCacheManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
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

func (manager *SWafRuleGroupCacheManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafRuleGroupCacheListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
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
	return q, nil
}

func (manager *SWafRuleGroupCacheManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (self *SWafRuleGroupCache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SWafRuleGroupCache) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SWafRuleGroupCache) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SWafRuleGroupCache) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SWafRuleGroupCache) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafRuleGroupCacheDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(userCred, api.WAF_RULE_GROUP_STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SWafRuleGroupCache) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
	region, err := self.GetRegion()
	if err != nil {
		return nil, errors.Wrapf(err, "GetRegion")
	}
	provider, err := self.GetDriver(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDriver")
	}
	return provider.GetIRegionById(region.ExternalId)
}

func (self *SWafRuleGroupCache) GetICloudWafRuleGroup(ctx context.Context) (cloudprovider.ICloudWafRuleGroup, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	caches, err := iRegion.GetICloudWafRuleGroups()
	if err != nil {
		return nil, errors.Wrapf(err, "GetICloudWafRuleGroups")
	}
	for i := range caches {
		if caches[i].GetGlobalId() == self.ExternalId {
			return caches[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.ExternalId)
}

func (self *SWafRuleGroupCache) syncWithCloudRuleGroup(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudWafRuleGroup) error {
	_, err := db.Update(self, func() error {
		self.Status = api.WAF_RULE_GROUP_STATUS_AVAILABLE
		self.Name = ext.GetName()
		self.Type = ext.GetWafType()
		self.Description = ext.GetDesc()
		return nil
	})
	return err
}

func (self *SCloudregion) GetRuleGroups(managerId string) ([]SWafRuleGroupCache, error) {
	q := WafRuleGroupCacheManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	caches := []SWafRuleGroupCache{}
	err := db.FetchModelObjects(WafRuleGroupCacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SCloudregion) createWafRuleGroup(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafRuleGroup) (*SWafRuleGroup, error) {
	rg := &SWafRuleGroup{}
	rg.SetModelManager(WafRuleGroupManager, rg)
	rg.Name = ext.GetName()
	rg.Status = api.WAF_RULE_GROUP_STATUS_AVAILABLE
	rg.Description = ext.GetDesc()
	rg.DomainId = provider.DomainId
	return rg, WafRuleGroupManager.TableSpec().Insert(ctx, rg)
}

func (self *SCloudregion) createRuleGroup(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafRuleGroup) (*SWafRuleGroup, error) {
	rg := &SWafRuleGroup{}
	rg.SetModelManager(WafRuleGroupManager, rg)
	rg.Name = ext.GetName()
	rg.Status = api.WAF_RULE_GROUP_STATUS_AVAILABLE
	rg.Description = ext.GetDesc()
	rg.DomainId = provider.DomainId
	return rg, WafRuleGroupManager.TableSpec().Insert(ctx, rg)
}

func (self *SCloudregion) newFromCloudWafRuleGroup(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafRuleGroup) error {
	rg, err := self.createRuleGroup(ctx, userCred, provider, ext)
	if err != nil {
		return errors.Wrapf(err, "createRuleGroup")
	}
	cache := &SWafRuleGroupCache{}
	cache.SetModelManager(WafRuleGroupCacheManager, cache)
	cache.Name = ext.GetName()
	cache.WafRuleGroupId = rg.Id
	cache.CloudregionId = self.Id
	cache.ManagerId = provider.Id
	cache.ExternalId = ext.GetGlobalId()
	cache.Status = api.WAF_RULE_GROUP_STATUS_AVAILABLE
	cache.Type = ext.GetWafType()
	cache.Description = ext.GetDesc()
	return WafRuleGroupCacheManager.TableSpec().Insert(ctx, cache)
}

func (self *SCloudregion) SyncWafRuleGroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudWafRuleGroup) compare.SyncResult {
	lockman.LockRawObject(ctx, WafRuleGroupCacheManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, WafRuleGroupCacheManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))

	result := compare.SyncResult{}

	dbRuleGroups, err := self.GetRuleGroups(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SWafRuleGroupCache, 0)
	commondb := make([]SWafRuleGroupCache, 0)
	commonext := make([]cloudprovider.ICloudWafRuleGroup, 0)
	added := make([]cloudprovider.ICloudWafRuleGroup, 0)
	err = compare.CompareSets(dbRuleGroups, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(commondb); i++ {
		err := commondb[i].syncWithCloudRuleGroup(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		err = self.newFromCloudWafRuleGroup(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}
