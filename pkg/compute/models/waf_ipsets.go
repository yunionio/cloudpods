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

type SWafIPSetManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var WafIPSetManager *SWafIPSetManager

func init() {
	WafIPSetManager = &SWafIPSetManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SWafIPSet{},
			"waf_ipsets_tbl",
			"waf_ipset",
			"waf_ipsets",
		),
	}
	WafIPSetManager.SetVirtualObject(WafIPSetManager)
}

type SWafIPSet struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SCloudregionResourceBase

	Type      cloudprovider.TWafType      `width:"20" charset:"utf8" nullable:"false" list:"user"`
	Addresses *cloudprovider.WafAddresses `list:"domain" update:"domain" create:"required"`
}

func (manager *SWafIPSetManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WafIPSetDetails {
	rows := make([]api.WafIPSetDetails, len(objs))
	siRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.WafIPSetDetails{
			StatusInfrasResourceBaseDetails: siRows[i],
			ManagedResourceInfo:             managerRows[i],
			CloudregionResourceInfo:         regionRows[i],
		}
	}
	return rows
}

// 列出WAF IPSets
func (manager *SWafIPSetManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafIPSetListInput,
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

func (manager *SWafIPSetManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (manager *SWafIPSetManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafIPSetListInput,
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

func (manager *SWafIPSetManager) ListItemExportKeys(ctx context.Context,
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

func (self *SWafIPSet) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SWafIPSet) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SWafIPSet) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SWafIPSet) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafIPSetDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SCloudregion) GetIPSets(managerId string) ([]SWafIPSet, error) {
	q := WafIPSetManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SWafIPSet{}
	err := db.FetchModelObjects(WafIPSetManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) SyncWafIPSets(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	exts []cloudprovider.ICloudWafIPSet,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, WafIPSetManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, WafIPSetManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))

	result := compare.SyncResult{}

	dbIPSets, err := self.GetIPSets(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SWafIPSet, 0)
	commondb := make([]SWafIPSet, 0)
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
		err = self.newFromCloudWafIPSet(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (self *SWafIPSet) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SWafIPSet) syncWithCloudIPSet(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudWafIPSet) error {
	_, err := db.Update(self, func() error {
		self.Status = apis.STATUS_AVAILABLE
		if options.Options.EnableSyncName {
			self.Name = ext.GetName()
		}
		address := ext.GetAddresses()
		self.Addresses = &address
		if desc := ext.GetDesc(); len(desc) > 0 {
			self.Description = desc
		}
		return nil
	})
	return err
}

func (self *SCloudregion) newFromCloudWafIPSet(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafIPSet) error {
	ret := &SWafIPSet{}
	ret.SetModelManager(WafIPSetManager, ret)
	ret.Name = ext.GetName()
	ret.CloudregionId = self.Id
	ret.ManagerId = provider.Id
	ret.ExternalId = ext.GetGlobalId()
	ret.Status = apis.STATUS_AVAILABLE
	ret.Type = ext.GetType()
	ret.Description = ext.GetDesc()
	address := ext.GetAddresses()
	ret.Addresses = &address
	return WafIPSetManager.TableSpec().Insert(ctx, ret)
}

func (self *SWafIPSet) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
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

func (self *SWafIPSet) GetICloudWafIPSet(ctx context.Context) (cloudprovider.ICloudWafIPSet, error) {
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
