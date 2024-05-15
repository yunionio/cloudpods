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
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SWafRuleGroupManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var WafRuleGroupManager *SWafRuleGroupManager

func init() {
	WafRuleGroupManager = &SWafRuleGroupManager{
		SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(
			SWafRuleGroup{},
			"waf_rule_groups_tbl",
			"waf_rule_group",
			"waf_rule_groups",
		),
	}
	WafRuleGroupManager.SetVirtualObject(WafRuleGroupManager)
}

type SWafRuleGroup struct {
	db.SStatusInfrasResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase
	SCloudregionResourceBase

	WafType cloudprovider.TWafType `width:"40" charset:"ascii" list:"domain" nullable:"false"`
}

func (manager *SWafRuleGroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WafRuleGroupDetails {
	rows := make([]api.WafRuleGroupDetails, len(objs))
	siRows := manager.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.WafRuleGroupDetails{
			StatusInfrasResourceBaseDetails: siRows[i],
			ManagedResourceInfo:             managerRows[i],
			CloudregionResourceInfo:         regionRows[i],
		}
	}
	return rows
}

// 列出WAF RuleGroups
func (manager *SWafRuleGroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafRuleGroupListInput,
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

func (manager *SWafRuleGroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (manager *SWafRuleGroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafRuleGroupListInput,
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

func (manager *SWafRuleGroupManager) ListItemExportKeys(ctx context.Context,
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

func (self *SWafRuleGroup) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SWafRuleGroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SWafRuleGroup) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	rules, err := self.GetWafRules()
	if err != nil {
		return errors.Wrapf(err, "GetWafRules")
	}
	for i := range rules {
		err = rules[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "Delete rule %s %s", rules[i].Id, rules[i].Name)
		}
	}
	return self.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SWafRuleGroup) GetIRegion(ctx context.Context) (cloudprovider.ICloudRegion, error) {
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

func (self *SWafRuleGroup) GetICloudWafRuleGroup(ctx context.Context) (cloudprovider.ICloudWafRuleGroup, error) {
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

func (self *SWafRuleGroup) syncWithCloudRuleGroup(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudWafRuleGroup) error {
	_, err := db.Update(self, func() error {
		self.Status = apis.STATUS_AVAILABLE
		self.Name = ext.GetName()
		self.WafType = ext.GetWafType()
		self.Description = ext.GetDesc()
		return nil
	})
	return err
}

func (self *SCloudregion) GetRuleGroups(managerId string) ([]SWafRuleGroup, error) {
	q := WafRuleGroupManager.Query().Equals("cloudregion_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	ret := []SWafRuleGroup{}
	err := db.FetchModelObjects(WafRuleGroupManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (self *SCloudregion) SyncWafRuleGroups(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudWafRuleGroup) compare.SyncResult {
	lockman.LockRawObject(ctx, WafRuleGroupManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))
	defer lockman.ReleaseRawObject(ctx, WafRuleGroupManager.Keyword(), fmt.Sprintf("%s-%s", self.Id, provider.Id))

	result := compare.SyncResult{}

	dbRuleGroups, err := self.GetRuleGroups(provider.Id)
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SWafRuleGroup, 0)
	commondb := make([]SWafRuleGroup, 0)
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

func (self *SCloudregion) newFromCloudWafRuleGroup(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudWafRuleGroup) error {
	ret := &SWafRuleGroup{}
	ret.SetModelManager(WafRuleGroupManager, ret)
	ret.Name = ext.GetName()
	ret.CloudregionId = self.Id
	ret.ManagerId = provider.Id
	ret.ExternalId = ext.GetGlobalId()
	ret.Status = apis.STATUS_AVAILABLE
	ret.WafType = ext.GetWafType()
	ret.Description = ext.GetDesc()
	return WafRuleGroupManager.TableSpec().Insert(ctx, ret)
}
