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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=waf_regexset
// +onecloud:swagger-gen-model-plural=waf_regexsets
type SWafRegexSetManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var WafRegexSetManager *SWafRegexSetManager

func init() {
	WafRegexSetManager = &SWafRegexSetManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SWafRegexSet{},
			"waf_regexsets_tbl",
			"waf_regexset",
			"waf_regexsets",
		),
	}
	WafRegexSetManager.SetVirtualObject(WafRegexSetManager)
}

type SWafRegexSet struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase
	SCloudregionResourceBase

	RegexPatterns *cloudprovider.WafRegexPatterns `list:"domain" update:"domain" create:"required"`
	Type          cloudprovider.TWafType          `width:"20" charset:"utf8" nullable:"false" list:"user"`
}

func (manager *SWafRegexSetManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WafRegexSetDetails {
	rows := make([]api.WafRegexSetDetails, len(objs))
	siRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.WafRegexSetDetails{
			StatusInfrasResourceBaseDetails: siRows[i],
			ManagedResourceInfo:             managerRows[i],
			CloudregionResourceInfo:         regionRows[i],
		}
	}
	return rows
}

// 列出WAF RegexSets
func (manager *SWafRegexSetManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafRegexSetListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemFilter")
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

func (manager *SWafRegexSetManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
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

func (manager *SWafRegexSetManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafRegexSetListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.OrderByExtraFields")
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

func (manager *SWafRegexSetManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.ListItemExportKeys")
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

func (self *SWafRegexSet) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SWafRegexSet) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SWafRegexSet) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SWafRegexSet) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SWafRegexSet) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafRegexSetDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SWafRegexSet) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
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

func (self *SWafRegexSet) GetICloudWafRegexSet(ctx context.Context) (cloudprovider.ICloudWafRegexSet, error) {
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

func (self *SWafRegexSet) syncWithCloudRegexSet(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudWafRegexSet) error {
	_, err := db.Update(self, func() error {
		self.Status = apis.STATUS_AVAILABLE
		if options.Options.EnableSyncName {
			self.Name = ext.GetName()
		}
		if desc := ext.GetDesc(); len(desc) > 0 {
			self.Description = desc
		}
		patterns := ext.GetRegexPatterns()
		self.RegexPatterns = &patterns
		return nil
	})
	return err
}

func (self *SCloudregion) GetRegexSets(managerId string) ([]SWafRegexSet, error) {
	q := WafRegexSetManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SWafRegexSet{}
	err := db.FetchModelObjects(WafRegexSetManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) newFromCloudWafRegexSet(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafRegexSet) error {
	ret := &SWafRegexSet{}
	ret.SetModelManager(WafRegexSetManager, ret)
	ret.Name = ext.GetName()
	ret.CloudregionId = self.Id
	ret.ManagerId = provider.Id
	ret.ExternalId = ext.GetGlobalId()
	ret.Status = apis.STATUS_AVAILABLE
	ret.Type = ext.GetType()
	patterns := ext.GetRegexPatterns()
	ret.RegexPatterns = &patterns
	ret.Description = ext.GetDesc()
	return WafRegexSetManager.TableSpec().Insert(ctx, ret)
}

func (self *SCloudregion) SyncWafRegexSets(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	exts []cloudprovider.ICloudWafRegexSet,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, WafRegexSetManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, WafRegexSetManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))

	result := compare.SyncResult{}

	dbRegexSets, err := self.GetRegexSets(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SWafRegexSet, 0)
	commondb := make([]SWafRegexSet, 0)
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
		err = self.newFromCloudWafRegexSet(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}
