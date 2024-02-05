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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SWafRuleManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
}

var WafRuleManager *SWafRuleManager

func init() {
	WafRuleManager = &SWafRuleManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SWafRule{},
			"waf_rules_tbl",
			"waf_rule",
			"waf_rules",
		),
	}
	WafRuleManager.SetVirtualObject(WafRuleManager)
}

type SWafRule struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase

	// 规则优先级
	Priority int `nullable:"false" list:"domain" create:"required"`
	// 规则默认行为
	Action *cloudprovider.DefaultAction `charset:"utf8" nullable:"true" list:"user" update:"domain" create:"required"`
	// 条件
	StatementConditon cloudprovider.TWafStatementCondition `width:"20" charset:"ascii" nullable:"false" list:"domain" create:"optional"`
	// 规则组的id
	WafRuleGroupId string `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"optional"`
	// 所属waf实例id
	WafInstanceId string `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"optional"`
}

func (manager *SWafRuleManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	values := struct {
		WafRuleGroupId string
		WafInstanceId  string
	}{}
	data.Unmarshal(&values)
	return jsonutils.Marshal(values)
}

func (manager *SWafRuleManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	data := struct {
		WafRuleGroupId string
		WafInstanceId  string
	}{}
	if len(data.WafRuleGroupId) > 0 {
		q = q.Equals("waf_rule_group_id", data.WafRuleGroupId)
	}
	if len(data.WafInstanceId) > 0 {
		q = q.Equals("waf_instance_id", data.WafInstanceId)
	}
	return q
}

func (manager *SWafRuleManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	values := struct {
		WafRuleGroupId string
		WafInstanceId  string
	}{}
	data.Unmarshal(&values)
	if len(values.WafInstanceId) > 0 {
		ins, err := db.FetchById(WafInstanceManager, values.WafInstanceId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(WafInstanceManager, %s)", values.WafInstanceId)
		}
		waf := ins.(*SWafInstance)
		return waf.GetOwnerId(), nil
	}
	if len(values.WafRuleGroupId) > 0 {
		rg, err := db.FetchById(WafRuleGroupManager, values.WafRuleGroupId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(WafRuleGroupManager, %s)", values.WafRuleGroupId)
		}
		return rg.GetOwnerId(), nil
	}
	return db.FetchDomainInfo(ctx, data)
}

func (manager *SWafRuleManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	sq1 := WafInstanceManager.Query("id")
	sq1 = db.SharableManagerFilterByOwner(ctx, WafInstanceManager, sq1, userCred, ownerId, scope)
	sq2 := WafRuleGroupManager.Query("id")
	sq2 = db.SharableManagerFilterByOwner(ctx, WafRuleGroupManager, sq2, userCred, ownerId, scope)
	return q.Filter(sqlchemy.OR(
		sqlchemy.In(q.Field("waf_instance_id"), sq1.SubQuery()),
		sqlchemy.In(q.Field("waf_rule_group_id"), sq2.SubQuery()),
	))
}

func (manager *SWafRuleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.WafRuleCreateInput) (api.WafRuleCreateInput, error) {
	if len(input.WafInstanceId) > 0 {
		ins, err := validators.ValidateModel(ctx, userCred, WafInstanceManager, &input.WafInstanceId)
		if err != nil {
			return input, err
		}
		waf := ins.(*SWafInstance)
		if waf.Status != api.WAF_STATUS_AVAILABLE {
			return input, httperrors.NewInvalidStatusError("waf %s status is not available", waf.Name)
		}
		region, err := waf.GetRegion()
		if err != nil {
			return input, httperrors.NewGeneralError(errors.Wrapf(err, "GetRegion"))
		}
		input, err = region.GetDriver().ValidateCreateWafRuleData(ctx, userCred, waf, input)
		if err != nil {
			return input, err
		}
	} else if len(input.WafRuleGroupId) > 0 {
		return input, httperrors.NewInputParameterError("not implement")
	} else {
		return input, httperrors.NewMissingParameterError("waf_instance_id")
	}

	var err error
	input.StatusStandaloneResourceCreateInput, err = manager.SStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusStandaloneResourceCreateInput)
	if err != nil {
		return input, err
	}

	return input, nil
}

func (self *SWafRule) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	input := &api.WafRuleCreateInput{}
	data.Unmarshal(input)

	for _, s := range input.Statements {
		statement := &SWafRuleStatement{}
		statement.SetModelManager(WafRuleStatementManager, statement)
		statement.SWafStatement = s
		statement.WafRuleId = self.Id
		WafRuleStatementManager.TableSpec().Insert(ctx, statement)
	}

	self.StartCreateTask(ctx, userCred)
}

func (self *SWafRule) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafRuleCreateTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.WAF_RULE_STATUS_CREATING, "")
	return task.ScheduleRun(nil)
}

// 列出WAF规则
func (manager *SWafRuleManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.WafRuleListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusStandaloneResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	if len(query.WafInstanceId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, WafInstanceManager, &query.WafInstanceId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("waf_instance_id", query.WafInstanceId)
	}
	if len(query.WafRuleGroupId) > 0 {
		_, err := validators.ValidateModel(ctx, userCred, WafRuleGroupManager, &query.WafRuleGroupId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("waf_rule_group_id", query.WafRuleGroupId)
	}

	return q, nil
}

func (manager *SWafRuleManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.WafRuleDetails {
	rows := make([]api.WafRuleDetails, len(objs))
	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	ruleIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.WafRuleDetails{
			StatusStandaloneResourceDetails: stdRows[i],
		}
		ruleIds[i] = objs[i].(*SWafRule).Id
	}
	q := WafRuleStatementManager.Query().In("waf_rule_id", ruleIds)
	statements := []SWafRuleStatement{}
	err := q.All(&statements)
	if err != nil {
		return rows
	}
	statementMaps := map[string][]cloudprovider.SWafStatement{}
	for i := range statements {
		_, ok := statementMaps[statements[i].WafRuleId]
		if !ok {
			statementMaps[statements[i].WafRuleId] = []cloudprovider.SWafStatement{}
		}
		statementMaps[statements[i].WafRuleId] = append(statementMaps[statements[i].WafRuleId], statements[i].SWafStatement)
	}
	for i := range rows {
		rows[i].Statements, _ = statementMaps[ruleIds[i]]
	}

	return rows
}

func (self *SWafRule) GetWafInstance() (*SWafInstance, error) {
	waf, err := WafInstanceManager.FetchById(self.WafInstanceId)
	if err != nil {
		return nil, errors.Wrapf(err, "WafInstanceManager.FetchById(%s)", self.WafInstanceId)
	}
	return waf.(*SWafInstance), nil
}

func (self *SWafRule) GetWafRuleGroup() (*SWafRuleGroup, error) {
	rg, err := WafRuleGroupManager.FetchById(self.WafRuleGroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "WafRuleGroupManager.FetchById(%s)", self.WafRuleGroupId)
	}
	return rg.(*SWafRuleGroup), nil
}

func (self *SWafRule) GetOwnerId() mcclient.IIdentityProvider {
	if len(self.WafInstanceId) > 0 {
		ins, err := self.GetWafInstance()
		if err != nil {
			return nil
		}
		return ins.GetOwnerId()
	}
	if len(self.WafRuleGroupId) > 0 {
		rg, err := self.GetWafRuleGroup()
		if err != nil {
			return nil
		}
		return rg.GetOwnerId()
	}
	return nil
}

func (manager *SWafRuleManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SWafInstance) GetWafRules() ([]SWafRule, error) {
	q := WafRuleManager.Query().Equals("waf_instance_id", self.Id)
	rules := []SWafRule{}
	err := db.FetchModelObjects(WafRuleManager, q, &rules)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return rules, nil
}

func (self *SWafRule) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred)
}

func (self *SWafRule) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafRuleDeleteTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.WAF_RULE_STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

func (self *SWafRule) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SWafRule) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	statements, err := self.GetRuleStatements()
	if err != nil {
		return errors.Wrapf(err, "GetRuleStatements")
	}
	for i := range statements {
		err = statements[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "Delete statement %s(%s)", statements[i].Type, statements[i].MatchField)
		}
	}
	return self.SStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SWafRule) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.RealDelete(ctx, userCred)
}

func (self *SWafRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.WafRuleUpdateInput) (api.WafRuleUpdateInput, error) {
	var err error
	if len(input.Name) > 0 && input.Name != self.Name {
		return input, httperrors.NewInputParameterError("Not allow update rule name")
	}
	input.StatusStandaloneResourceBaseUpdateInput, err = self.SStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}

func (self *SWafRule) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)

	input := api.WafRuleUpdateInput{}
	data.Unmarshal(&input)

	statements, err := self.GetRuleStatements()
	if err != nil {
		return
	}
	for i := len(input.Statements); i < len(statements); i++ {
		statements[i].Delete(ctx, userCred)
	}
	for i := len(statements); i < len(input.Statements); i++ {
		statement := &SWafRuleStatement{}
		statement.SetModelManager(WafRuleStatementManager, statement)
		statement.SWafStatement = input.Statements[i]
		statement.WafRuleId = self.Id
		WafRuleStatementManager.TableSpec().Insert(ctx, statement)
	}
	for i := 0; i < len(input.Statements) && i < len(statements); i++ {
		db.Update(&statements[i], func() error {
			statements[i].SWafStatement = input.Statements[i]
			return nil
		})
	}
	self.StartUpdateTask(ctx, userCred, "")
}

func (self *SWafRule) StartUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "WafRuleUpdateTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrapf(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.WAF_RULE_STATUS_UPDATING, "")
	return task.ScheduleRun(nil)
}

func (self *SWafRule) SyncWithCloudRule(ctx context.Context, userCred mcclient.TokenCredential, rule cloudprovider.ICloudWafRule) error {
	_, err := db.Update(self, func() error {
		self.Action = rule.GetAction()
		self.StatementConditon = rule.GetStatementCondition()
		self.Priority = rule.GetPriority()
		self.Status = api.WAF_RULE_STATUS_AVAILABLE
		self.Name = rule.GetName()
		self.ExternalId = rule.GetGlobalId()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	return self.SyncStatements(ctx, userCred, rule)
}

func (self *SWafInstance) newFromCloudRule(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudWafRule) error {
	rule := &SWafRule{}
	rule.SetModelManager(WafRuleManager, rule)
	rule.WafInstanceId = self.Id
	rule.Name = ext.GetName()
	rule.Description = ext.GetDesc()
	rule.ExternalId = ext.GetGlobalId()
	rule.Action = ext.GetAction()
	rule.StatementConditon = ext.GetStatementCondition()
	rule.Priority = ext.GetPriority()
	rule.Status = api.WAF_RULE_STATUS_AVAILABLE
	err := WafRuleManager.TableSpec().Insert(ctx, rule)
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}
	return rule.SyncStatements(ctx, userCred, ext)
}

func (self *SWafInstance) SyncWafRules(ctx context.Context, userCred mcclient.TokenCredential, exts []cloudprovider.ICloudWafRule) compare.SyncResult {
	lockman.LockRawObject(ctx, WafInstanceManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, WafInstanceManager.Keyword(), self.Id)

	result := compare.SyncResult{}

	dbRules, err := self.GetWafRules()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SWafRule, 0)
	commondb := make([]SWafRule, 0)
	commonext := make([]cloudprovider.ICloudWafRule, 0)
	added := make([]cloudprovider.ICloudWafRule, 0)
	if err := compare.CompareSets(dbRules, exts, &removed, &commondb, &commonext, &added); err != nil {
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
		err := commondb[i].SyncWithCloudRule(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	for i := 0; i < len(added); i++ {
		err := self.newFromCloudRule(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SWafRuleGroup) GetWafRules() ([]SWafRule, error) {
	q := WafRuleManager.Query().Equals("waf_rule_group_id", self.Id)
	rules := []SWafRule{}
	err := db.FetchModelObjects(WafRuleManager, q, &rules)
	return rules, err
}

func (self *SWafRuleGroup) newFromManagedRule(ctx context.Context, userCred mcclient.TokenCredential, ext SWafRule) error {
	ext.SetModelManager(WafRuleManager, &ext)
	ext.WafRuleGroupId = self.Id
	return WafRuleManager.TableSpec().Insert(ctx, &ext)
}

func (self *SWafRuleGroup) SyncManagedWafRules(ctx context.Context, userCred mcclient.TokenCredential, exts []SWafRule) (compare.SyncResult, error) {
	lockman.LockRawObject(ctx, WafRuleGroupManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, WafRuleGroupManager.Keyword(), self.Id)

	result := compare.SyncResult{}

	dbRules, err := self.GetWafRules()
	if err != nil {
		return result, errors.Wrapf(err, "GetWafRules")
	}

	removed := make([]SWafRule, 0)
	commondb := make([]SWafRule, 0)
	commonext := make([]SWafRule, 0)
	added := make([]SWafRule, 0)
	err = compare.CompareSets(dbRules, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		return result, errors.Wrapf(err, "compare.CompareSets")
	}

	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	for i := 0; i < len(added); i++ {
		err := self.newFromManagedRule(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result, nil
}

func (self *SWafRule) GetICloudWafInstance(ctx context.Context) (cloudprovider.ICloudWafInstance, error) {
	ins, err := self.GetWafInstance()
	if err != nil {
		return nil, errors.Wrapf(err, "GetWafInstance")
	}
	iWaf, err := ins.GetICloudWafInstance(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "GetICloudWafInstance")
	}
	return iWaf, nil

}

func (self *SWafRule) GetICloudWafRule(ctx context.Context) (cloudprovider.ICloudWafRule, error) {
	if len(self.ExternalId) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty external id")
	}
	if len(self.WafInstanceId) > 0 {
		iWaf, err := self.GetICloudWafInstance(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "GetICloudWafInstance")
		}
		rules, err := iWaf.GetRules()
		if err != nil {
			return nil, errors.Wrapf(err, "GetWafRules")
		}
		for i := range rules {
			if rules[i].GetGlobalId() == self.ExternalId {
				return rules[i], nil
			}
		}
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, self.ExternalId)
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "")
}

// 同步WAF规则状态
func (self *SWafRule) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.WafSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "WafRuleSyncstatusTask", "")
}
