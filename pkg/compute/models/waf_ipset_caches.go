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

type SWafIPSetCacheManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var WafIPSetCacheManager *SWafIPSetCacheManager

func init() {
	WafIPSetCacheManager = &SWafIPSetCacheManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SWafIPSetCache{},
			"waf_ipset_caches_tbl",
			"waf_ipset_cache",
			"waf_ipset_caches",
		),
	}
	WafIPSetCacheManager.SetVirtualObject(WafIPSetCacheManager)
}

type SWafIPSetCache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	Type       cloudprovider.TWafType `width:"20" charset:"utf8" nullable:"false" list:"user"`
	WafIPSetId string                 `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SWafIPSetCacheManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (manager *SWafIPSetCacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WafIPSetCacheDetails {
	rows := make([]api.WafIPSetCacheDetails, len(objs))
	ssRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.WafIPSetCacheDetails{
			StatusStandaloneResourceDetails: ssRows[i],
			ManagedResourceInfo:             managerRows[i],
			CloudregionResourceInfo:         regionRows[i],
		}
	}
	return rows
}

// 列出WAF IPSet缓存
func (manager *SWafIPSetCacheManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafIPSetCacheListInput,
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

func (manager *SWafIPSetCacheManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (manager *SWafIPSetCacheManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafIPSetCacheListInput,
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

func (manager *SWafIPSetCacheManager) ListItemExportKeys(ctx context.Context,
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

func (self *SWafIPSetCache) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SWafIPSetCache) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SWafIPSetCache) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SWafIPSetCache) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SWafIPSetCache) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafIPSetCacheDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(userCred, api.WAF_IPSET_STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SWafIPSetCache) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
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

func (self *SWafIPSetCache) GetICloudWafIPSet(ctx context.Context) (cloudprovider.ICloudWafIPSet, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	iRegion, err := self.GetIRegion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetIRegion")
	}
	caches, err := iRegion.GetICloudWafIPSets()
	if err != nil {
		return nil, errors.Wrapf(err, "GetICloudWafIPSets")
	}
	for i := range caches {
		if caches[i].GetGlobalId() == self.ExternalId {
			return caches[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.ExternalId)
}

func (self *SWafIPSetCache) syncWithCloudIPSet(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudWafIPSet) error {
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

func (self *SCloudregion) GetIPSets(managerId string) ([]SWafIPSetCache, error) {
	q := WafIPSetCacheManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	caches := []SWafIPSetCache{}
	err := db.FetchModelObjects(WafIPSetCacheManager, q, &caches)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return caches, nil
}

func (self *SCloudregion) findOrCreateWafIPSet(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafIPSet) (*SWafIPSet, error) {
	q := WafIPSetManager.Query().Equals("domain_id", provider.DomainId).Equals("addresses", ext.GetAddresses().String())
	ipSets := []SWafIPSet{}
	err := db.FetchModelObjects(WafIPSetManager, q, &ipSets)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	if len(ipSets) > 0 {
		return &ipSets[0], nil
	}
	ipSet := &SWafIPSet{}
	ipSet.SetModelManager(WafIPSetManager, ipSet)
	ipSet.Name = ext.GetName()
	ipSet.Status = api.WAF_IPSET_STATUS_AVAILABLE
	ipSet.DomainId = provider.DomainId
	addrs := ext.GetAddresses()
	ipSet.Addresses = &addrs
	return ipSet, WafIPSetManager.TableSpec().Insert(ctx, ipSet)
}

func (self *SCloudregion) newFromCloudWafIPSet(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafIPSet, ipSetId string) error {
	cache := &SWafIPSetCache{}
	cache.SetModelManager(WafIPSetCacheManager, cache)
	cache.Name = ext.GetName()
	cache.WafIPSetId = ipSetId
	cache.CloudregionId = self.Id
	cache.ManagerId = provider.Id
	cache.ExternalId = ext.GetGlobalId()
	cache.Status = api.WAF_IPSET_STATUS_AVAILABLE
	cache.Type = ext.GetType()
	cache.Description = ext.GetDesc()
	return WafIPSetCacheManager.TableSpec().Insert(ctx, cache)
}

func (self *SCloudregion) SyncWafIPSets(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	exts []cloudprovider.ICloudWafIPSet,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, WafIPSetCacheManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, WafIPSetCacheManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))

	result := compare.SyncResult{}

	dbIPSets, err := self.GetIPSets(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SWafIPSetCache, 0)
	commondb := make([]SWafIPSetCache, 0)
	commonext := make([]cloudprovider.ICloudWafIPSet, 0)
	added := make([]cloudprovider.ICloudWafIPSet, 0)
	err = compare.CompareSets(dbIPSets, exts, &removed, &commondb, &commonext, &added)
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
			err := commondb[i].syncWithCloudIPSet(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}

	for i := 0; i < len(added); i++ {
		ipSet, err := self.findOrCreateWafIPSet(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		err = self.newFromCloudWafIPSet(ctx, userCred, provider, added[i], ipSet.Id)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}
