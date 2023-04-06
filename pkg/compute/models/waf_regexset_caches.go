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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SWafRegexSetCacheManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var WafRegexSetCacheManager *SWafRegexSetCacheManager

func init() {
	WafRegexSetCacheManager = &SWafRegexSetCacheManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SWafRegexSetCache{},
			"waf_regexset_caches_tbl",
			"waf_regexset_cache",
			"waf_regexset_caches",
		),
	}
	WafRegexSetCacheManager.SetVirtualObject(WafRegexSetCacheManager)
}

type SWafRegexSetCache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	Type          cloudprovider.TWafType `width:"20" charset:"utf8" nullable:"false" list:"user"`
	WafRegexSetId string                 `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SWafRegexSetCacheManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (manager *SWafRegexSetCacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WafRegexSetCacheDetails {
	rows := make([]api.WafRegexSetCacheDetails, len(objs))
	ssRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.WafRegexSetCacheDetails{
			StatusStandaloneResourceDetails: ssRows[i],
			ManagedResourceInfo:             managerRows[i],
			CloudregionResourceInfo:         regionRows[i],
		}
	}
	return rows
}

// 列出WAF RegexSet缓存
func (manager *SWafRegexSetCacheManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafRegexSetCacheListInput,
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

func (manager *SWafRegexSetCacheManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (manager *SWafRegexSetCacheManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafRegexSetCacheListInput,
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

func (manager *SWafRegexSetCacheManager) ListItemExportKeys(ctx context.Context,
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

func (self *SWafRegexSetCache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SWafRegexSetCache) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SWafRegexSetCache) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SWafRegexSetCache) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SWafRegexSetCache) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafRegexSetCacheDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(userCred, api.WAF_REGEX_SET_STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SWafRegexSetCache) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
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

func (self *SWafRegexSetCache) GetICloudWafRegexSet(ctx context.Context) (cloudprovider.ICloudWafRegexSet, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	caches, err := iRegion.GetICloudWafRegexSets()
	if err != nil {
		return nil, errors.Wrapf(err, "GetICloudWafRegexSets")
	}
	for i := range caches {
		if caches[i].GetGlobalId() == self.ExternalId {
			return caches[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.ExternalId)
}

func (self *SWafRegexSetCache) syncWithCloudRegexSet(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudWafRegexSet) error {
	_, err := db.Update(self, func() error {
		self.Status = api.WAF_IPSET_STATUS_AVAILABLE
		if options.Options.EnableSyncName {
			self.Name = ext.GetName()
		}
		self.Description = ext.GetDesc()
		return nil
	})
	return err
}

func (self *SCloudregion) GetRegexSets(managerId string) ([]SWafRegexSetCache, error) {
	q := WafRegexSetCacheManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	caches := []SWafRegexSetCache{}
	err := db.FetchModelObjects(WafRegexSetCacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SCloudregion) findOrCreateWafRegexSet(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafRegexSet) (*SWafRegexSet, error) {
	q := WafRegexSetManager.Query().Equals("domain_id", provider.DomainId).Equals("regex_patterns", ext.GetRegexPatterns().String())
	patternSets := []SWafRegexSet{}
	err := db.FetchModelObjects(WafRegexSetManager, q, &patternSets)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(patternSets) > 0 {
		return &patternSets[0], nil
	}
	ps := &SWafRegexSet{}
	ps.SetModelManager(WafRegexSetManager, ps)
	ps.Name = ext.GetName()
	ps.Status = api.WAF_IPSET_STATUS_AVAILABLE
	ps.DomainId = provider.DomainId
	patterns := ext.GetRegexPatterns()
	ps.RegexPatterns = &patterns
	return ps, WafRegexSetManager.TableSpec().Insert(ctx, ps)
}

func (self *SCloudregion) newFromCloudWafRegexSet(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafRegexSet, ipSetId string) error {
	cache := &SWafRegexSetCache{}
	cache.SetModelManager(WafRegexSetCacheManager, cache)
	cache.Name = ext.GetName()
	cache.WafRegexSetId = ipSetId
	cache.CloudregionId = self.Id
	cache.ManagerId = provider.Id
	cache.ExternalId = ext.GetGlobalId()
	cache.Status = api.WAF_IPSET_STATUS_AVAILABLE
	cache.Type = ext.GetType()
	cache.Description = ext.GetDesc()
	return WafRegexSetCacheManager.TableSpec().Insert(ctx, cache)
}

func (self *SCloudregion) SyncWafRegexSets(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	exts []cloudprovider.ICloudWafRegexSet,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, WafRegexSetCacheManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, WafRegexSetCacheManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))

	result := compare.SyncResult{}

	dbRegexSets, err := self.GetRegexSets(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SWafRegexSetCache, 0)
	commondb := make([]SWafRegexSetCache, 0)
	commonext := make([]cloudprovider.ICloudWafRegexSet, 0)
	added := make([]cloudprovider.ICloudWafRegexSet, 0)
	err = compare.CompareSets(dbRegexSets, exts, &removed, &commondb, &commonext, &added)
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

	if !xor {
		for i := 0; i < len(commondb); i++ {
			err := commondb[i].syncWithCloudRegexSet(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		ipSet, err := self.findOrCreateWafRegexSet(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		err = self.newFromCloudWafRegexSet(ctx, userCred, provider, added[i], ipSet.Id)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}
