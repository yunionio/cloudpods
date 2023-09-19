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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	identity_api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/onecloud/pkg/util/yunionmeta"
)

type SWafRuleGroupManager struct {
	db.SStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
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

	// 支持的WAF类型,仅is_system=true时有效
	WafType  cloudprovider.TWafType `width:"40" charset:"ascii" list:"domain" nullable:"false"`
	Provider string                 `width:"20" charset:"ascii" list:"domain" nullable:"false"`
	CloudEnv string                 `width:"20" charset:"ascii" list:"domain" nullable:"false"`
	IsSystem bool                   `nullable:"false" default:"false" list:"domain" update:"domain" create:"optional"`
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
	for i := range rows {
		rows[i] = api.WafRuleGroupDetails{
			StatusInfrasResourceBaseDetails: siRows[i],
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

	if query.IsSystem != nil {
		q = q.Equals("is_system", *query.IsSystem)
	}

	if len(query.Provider) > 0 {
		q = q.Equals("provider", query.Provider)
	}

	if len(query.CloudEnv) > 0 {
		q = q.Equals("cloud_env", query.CloudEnv)
	}

	return q, nil
}

func (manager *SWafRuleGroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
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
	return q, nil
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

func (manager *SWafRuleGroupManager) GetWafGroups(cloudEnv string) ([]SWafRuleGroup, error) {
	q := manager.Query().Equals("cloud_env", cloudEnv).IsTrue("is_system")
	groups := []SWafRuleGroup{}
	err := db.FetchModelObjects(WafRuleGroupManager, q, &groups)
	return groups, err
}

func (self *SWafRuleGroup) syncWithCloudSku(ctx context.Context, userCred mcclient.TokenCredential, ext sWafGroup) error {
	_, err := db.Update(self, func() error {
		self.Name = ext.Name
		self.Description = ext.Description
		self.IsPublic = true
		self.Status = api.WAF_RULE_GROUP_STATUS_AVAILABLE
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	result, err := self.SyncManagedWafRules(ctx, userCred, ext.Rules)
	if err != nil {
		return errors.Wrapf(err, "SyncManagedWafRules")
	}
	log.Debugf("Sync waf group %s rule result: %s", self.Name, result.Result())
	return nil
}

func (manager *SWafRuleGroupManager) newFromCloudWafGroup(ctx context.Context, userCred mcclient.TokenCredential, ext sWafGroup) error {
	group := &ext.SWafRuleGroup
	group.SetModelManager(manager, group)
	group.Status = api.WAF_RULE_GROUP_STATUS_AVAILABLE
	group.IsPublic = true
	err := WafRuleGroupManager.TableSpec().Insert(ctx, group)
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}
	result, err := group.SyncManagedWafRules(ctx, userCred, ext.Rules)
	if err != nil {
		return errors.Wrapf(err, "SyncManagedWafRules")
	}
	log.Debugf("Sync waf group %s rule result: %s", group.Name, result.Result())
	return nil
}

type sWafGroup struct {
	SWafRuleGroup
	Rules []SWafRule
}

func (self sWafGroup) GetGlobalId() string {
	return self.ExternalId
}

func (self SWafRule) GetGlobalId() string {
	return self.ExternalId
}

func (manager *SWafRuleGroupManager) SyncWafGroups(ctx context.Context, userCred mcclient.TokenCredential, cloudEnv string, isStart bool) compare.SyncResult {
	lockman.LockRawObject(ctx, cloudEnv, manager.Keyword())
	defer lockman.ReleaseRawObject(ctx, cloudEnv, manager.Keyword())

	result := compare.SyncResult{}

	meta, err := yunionmeta.FetchYunionmeta(ctx)
	if err != nil {
		result.Error(errors.Wrapf(err, "FetchYunionmeta"))
		return result
	}

	exts := []sWafGroup{}
	err = meta.List(WafRuleManager.Keyword(), cloudEnv, &exts)
	if err != nil {
		result.Error(errors.Wrapf(err, "List(%s)", cloudEnv))
		return result
	}
	dbGroup, err := manager.GetWafGroups(cloudEnv)
	if err != nil {
		result.Error(errors.Wrapf(err, "GetWafGroups"))
		return result
	}

	if isStart && len(dbGroup) > 0 {
		log.Infof("%s waf group already synced, skip...", cloudEnv)
		return result
	}

	removed := make([]SWafRuleGroup, 0)
	commondb := make([]SWafRuleGroup, 0)
	commonext := make([]sWafGroup, 0)
	added := make([]sWafGroup, 0)

	err = compare.CompareSets(dbGroup, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
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
		err = commondb[i].syncWithCloudSku(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}
	for i := 0; i < len(added); i += 1 {
		err = manager.newFromCloudWafGroup(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func fetchCloudEnvs() ([]string, error) {
	accounts := []SCloudaccount{}
	q := CloudaccountManager.Query("provider", "access_url").In("provider", CloudproviderManager.GetPublicProviderProvidersQuery()).Distinct()
	err := q.All(&accounts)
	if err != nil {
		return nil, errors.Wrapf(err, "q.All")
	}
	ret := []string{}
	for i := range accounts {
		ret = append(ret, api.GetCloudEnv(accounts[i].Provider, accounts[i].AccessUrl))
	}
	return ret, nil
}

func SyncWafGroups(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	err := func() error {
		cloudEnvs, err := fetchCloudEnvs()
		if err != nil {
			return errors.Wrapf(err, "fetchCloudEnvs")
		}

		meta, err := yunionmeta.FetchYunionmeta(ctx)
		if err != nil {
			return errors.Wrapf(err, "FetchYunionmeta")
		}

		index, err := meta.Index(WafRuleManager.Keyword())
		if err != nil {
			return errors.Wrapf(err, "getWafIndex")
		}

		for _, cloudEnv := range cloudEnvs {
			skuMeta := &SWafRuleGroup{}
			skuMeta.SetModelManager(WafRuleGroupManager, skuMeta)
			skuMeta.DomainId = identity_api.DEFAULT_DOMAIN_ID
			skuMeta.Id = cloudEnv

			oldMd5 := db.Metadata.GetStringValue(ctx, skuMeta, db.SKU_METADAT_KEY, userCred)
			newMd5, ok := index[cloudEnv]
			if !ok || newMd5 == yunionmeta.EMPTY_MD5 || len(oldMd5) > 0 && newMd5 == oldMd5 {
				continue
			}

			db.Metadata.SetValue(ctx, skuMeta, db.SKU_METADAT_KEY, newMd5, userCred)

			result := WafRuleGroupManager.SyncWafGroups(ctx, userCred, cloudEnv, isStart)
			log.Debugf("sync %s waf group result: %s", cloudEnv, result.Result())
		}
		return nil
	}()
	if err != nil {
		log.Errorf("SyncWafGroups: error: %v", err)
	}
}
