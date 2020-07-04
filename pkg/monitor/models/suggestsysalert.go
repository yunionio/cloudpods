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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	SuggestSysAlertManager *SSuggestSysAlertManager
)

func init() {
	SuggestSysAlertManager = &SSuggestSysAlertManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			&SSuggestSysAlert{},
			"suggestsysalert_tbl",
			"suggestsysalert",
			"suggestsysalerts",
		),
	}
	SuggestSysAlertManager.SetVirtualObject(SuggestSysAlertManager)
}

// +onecloud:swagger-gen-model-singular=suggestsysalert
// +onecloud:swagger-gen-model-plural=suggestsysalerts
type SSuggestSysAlertManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SSuggestSysAlert struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	//监控规则对应的json对象
	RuleName      string               `list:"user" update:"user"`
	MonitorConfig jsonutils.JSONObject `list:"user" create:"required" update:"user"`
	//监控规则type：Rule Type
	Type    string               `width:"256" charset:"ascii" list:"user" update:"user"`
	ResMeta jsonutils.JSONObject `list:"user" update:"user"`
	Problem jsonutils.JSONObject `list:"user" update:"user"`
	//Suggest string               `width:"256"  list:"user" update:"user"`
	Action string `width:"256" charset:"ascii" list:"user" update:"user"`
	ResId  string `width:"256" charset:"ascii" list:"user" update:"user"`

	//
	CloudEnv     string `list:"user" update:"user"`
	Provider     string `list:"user" update:"user"`
	Project      string `list:"user" update:"user"`
	Cloudaccount string `list:"user" update:"user"`

	//费用
	Amount float64 `list:"user" update:"user"`
	//币种
	Currency string `list:"user" update:"user"`
}

func NewSuggestSysAlertManager(dt interface{}, keyword, keywordPlural string) *SSuggestSysAlertManager {
	man := &SSuggestSysAlertManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			dt,
			"sugalart_tbl",
			keyword,
			keywordPlural,
		),
	}
	man.SetVirtualObject(man)
	return man
}

func (manager *SSuggestSysAlertManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.SuggestSysAlertListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	if len(query.Type) > 0 {
		q = q.Equals("type", query.Type)
	}
	if len(query.ResId) > 0 {
		q = q.Equals("res_id", query.ResId)
	}
	//if len(query.Project) > 0 {
	//
	//	q = q.Equals("project", query.Project)
	//}
	if len(query.Providers) > 0 {
		q = q.In("provider", query.Providers)
	}
	if len(query.Brands) > 0 {
		q = q.In("provider", query.Brands)
	}
	if len(query.Cloudaccount) > 0 {
		q.In("cloudaccount", query.Cloudaccount)
	}
	if len(query.CloudEnv) > 0 {
		q = q.Equals("cloud_env", query.CloudEnv)
	}
	return q, nil
}

func (manager *SSuggestSysAlertManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*db.CustomizeListFilters, error) {
	filters := db.NewCustomizeListFilters()
	listInput := new(monitor.SuggestSysAlertListInput)
	if err := query.Unmarshal(listInput); err != nil {
		return filters, errors.Wrap(err, "unmarshal list input")
	}
	scope := rbacutils.ScopeProject
	if listInput.Scope != "" {
		scope = rbacutils.TRbacScope(listInput.Scope)
	}
	scopeFilter := func(obj jsonutils.JSONObject) (bool, error) {
		ignoreConfigs, err := SuggestSysRuleConfigManager.GetConfigsByScope(scope, userCred, true)
		if err != nil {
			return false, err
		}
		alert := new(SSuggestSysAlert)
		if err := obj.Unmarshal(alert); err != nil {
			return false, errors.Wrap(err, "unmarshal suggest alert")
		}
		for _, conf := range ignoreConfigs {
			if conf.ShouldIgnoreAlert(alert) {
				return false, nil
			}
		}
		return true, nil
	}
	filters.Append(scopeFilter)
	return filters, nil
}

func (manager *SSuggestSysAlertManager) GetAlert(id string) (*SSuggestSysAlert, error) {
	obj, err := manager.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SSuggestSysAlert), nil
}

func (man *SSuggestSysAlertManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.SuggestSysAlertListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SSuggestSysAlertManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.SuggestSysAlertCreateInput) (monitor.SuggestSysAlertCreateInput, error) {
	//rule 查询到资源信息后没有将资源id，进行转换
	if len(data.ResID) == 0 {
		return data, httperrors.NewInputParameterError("not found res_id %q", data.ResID)
	}
	if len(data.Type) == 0 {
		return data, httperrors.NewInputParameterError("not found type %q", data.Type)
	}
	return data, nil
}

func (man *SSuggestSysAlertManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.SuggestSysAlertDetails {
	rows := make([]monitor.SuggestSysAlertDetails, len(objs))
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.SuggestSysAlertDetails{
			VirtualResourceDetails: virtRows[i],
		}
		rows[i] = objs[i].(*SSuggestSysAlert).getMoreDetails(rows[i])
	}
	return rows
}

func (man *SSuggestSysRuleManager) GetDriver(drvType monitor.SuggestDriverType) ISuggestSysRuleDriver {
	return GetSuggestSysRuleDrivers()[drvType]
}

func (self *SSuggestSysAlert) GetDriver() ISuggestSysRuleDriver {
	return SuggestSysRuleManager.GetDriver(self.GetType())
}

func (self *SSuggestSysAlert) GetType() monitor.SuggestDriverType {
	return monitor.SuggestDriverType(self.Type)
}

func (self *SSuggestSysAlert) GetShowName() string {
	rule, _ := SuggestSysRuleManager.GetRules(self.GetType())
	var showName string
	if len(rule) != 0 {
		showName = fmt.Sprintf("%s-%s", self.Name, rule[0].Name)
	} else {
		showName = fmt.Sprintf("%s-%s", self.Name, self.Type)
	}
	return showName
}

func (self *SSuggestSysAlert) getMoreDetails(out monitor.SuggestSysAlertDetails) monitor.SuggestSysAlertDetails {
	err := self.ResMeta.Unmarshal(&out)
	if err != nil {
		log.Errorln("SSuggestSysAlert getMoreDetails's error:", err)
	}
	drv := self.GetDriver()
	out.Account = self.Cloudaccount
	out.ResType = string(drv.GetResourceType())
	out.RuleName = strings.ToLower(string(drv.GetType()))
	out.ShowName = self.GetShowName()
	out.Suggest = string(drv.GetSuggest())
	return out
}

func (manager *SSuggestSysAlertManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	switch field {
	case "account":
		q.AppendField(sqlchemy.DISTINCT(field, q.Field("cloudaccount"))).Distinct()
		q.NotEquals("cloudaccount", "")
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (alert *SSuggestSysAlert) ValidateUpdateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data monitor.SuggestSysAlertUpdateInput) (monitor.SuggestSysAlertUpdateInput, error) {
	//rule 查询到资源信息后没有将资源id，进行转换
	if len(data.ResID) == 0 {
		return data, httperrors.NewInputParameterError("not found res_id ")
	}
	if len(data.Type) == 0 {
		return data, httperrors.NewInputParameterError("not found type ")
	}
	var err error
	data.VirtualResourceBaseUpdateInput, err = alert.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query,
		data.VirtualResourceBaseUpdateInput)
	if err != nil {
		return data, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	return data, nil
}

func (self *SSuggestSysAlert) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (monitor.SuggestSysAlertDetails, error) {
	return monitor.SuggestSysAlertDetails{}, nil
}

func (self *SSuggestSysAlert) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {

}

func (self *SSuggestSysAlert) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred)
}

func (self *SSuggestSysAlert) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SSuggestSysAlert delete do nothing")
	return nil
}

func (self *SSuggestSysAlert) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SSuggestSysAlert) StartDeleteTask(
	ctx context.Context, userCred mcclient.TokenCredential) error {
	params := jsonutils.NewDict()

	return self.GetDriver().StartResolveTask(ctx, userCred, self, params)
}

func (self *SSuggestSysAlertManager) GetResources(tp ...monitor.SuggestDriverType) ([]SSuggestSysAlert, error) {
	resources := make([]SSuggestSysAlert, 0)
	query := self.Query()
	if len(tp) > 0 {
		query.In("type", tp)
	}
	err := db.FetchModelObjects(self, query, &resources)
	if err != nil {
		return resources, err
	}
	return resources, nil
}

func (manager *SSuggestSysAlertManager) GetExportExtraKeys(ctx context.Context, keys stringutils2.SSortedStrings, rowMap map[string]string) *jsonutils.JSONDict {
	alert := new(SSuggestSysAlert)
	manager.Query().RowMap2Struct(rowMap, alert)
	input := monitor.SuggestSysAlertDetails{}
	input = alert.getMoreDetails(input)
	res := jsonutils.Marshal(&input)
	dic := res.(*jsonutils.JSONDict)
	dic.Add(jsonutils.NewString(input.Account), "manager")
	dic.Add(jsonutils.NewString(alert.Provider), "hypervisor")
	return dic
}

func (self *SSuggestSysAlert) AllowPerformIgnore(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsProjectAllowPerform(userCred, self, "ignore")
}

func (self *SSuggestSysAlert) GetSuggestConfig(scope rbacutils.TRbacScope, domainId string, projectId string) (*SSuggestSysRuleConfig, error) {
	if scope == "" {
		scope = rbacutils.ScopeSystem
	}
	drv := self.GetDriver()
	resType := drv.GetResourceType()
	resId := self.ResId
	drvType := drv.GetType()
	scopeId := ""
	if scope == rbacutils.ScopeDomain {
		scopeId = domainId
	} else if scope == rbacutils.ScopeProject {
		scopeId = projectId
	}
	q := SuggestSysRuleConfigManager.Query().Equals("type", drvType).Equals("resource_id", resId).Equals("resource_type", resType)
	q = SuggestSysRuleConfigManager.FilterByScope(q, scope, scopeId)
	configs := make([]SSuggestSysRuleConfig, 0)
	if err := db.FetchModelObjects(SuggestSysRuleConfigManager, q, &configs); err != nil {
		return nil, errors.Wrap(err, "fetch suggest config")
	}
	if len(configs) == 0 {
		return nil, nil
	}
	config := configs[0]
	return &config, nil
}

func (self *SSuggestSysAlert) PerformIgnore(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONDict, data monitor.SuggestAlertIngoreInput) (jsonutils.JSONObject, error) {
	if data.Scope == "" {
		data.Scope = string(rbacutils.ScopeSystem)
	}
	config, err := self.GetSuggestConfig(rbacutils.TRbacScope(data.Scope), data.ProjectDomain, data.Project)
	if err != nil {
		return nil, err
	}
	drv := self.GetDriver()
	drvType := drv.GetType()
	resType := drv.GetResourceType()
	if config == nil {
		createData := new(monitor.SuggestSysRuleConfigCreateInput)
		createData.Name = self.GetShowName()
		createData.ScopedResourceCreateInput = data.ScopedResourceCreateInput
		createData.Type = &drvType
		createData.ResourceType = &resType
		createData.ResourceId = &self.ResId
		createData.IgnoreAlert = true
		data := createData.JSON(createData)
		conf, err := db.DoCreate(SuggestSysRuleConfigManager, ctx, userCred, nil, data, userCred)
		if err != nil {
			return nil, err
		}
		func() {
			lockman.LockObject(ctx, conf)
			defer lockman.ReleaseObject(ctx, conf)

			conf.PostCreate(ctx, userCred, userCred, nil, data)
		}()
	} else {
		if _, err := db.Update(config, func() error {
			config.IgnoreAlert = true
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

type SuggestAlertCost struct {
	CostType string
	Amount   float64
}

func (self *SSuggestSysAlertManager) AllowGetPropertyCost(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) bool {
	return true
}

func (self *SSuggestSysAlertManager) GetPropertyCost(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	costData := jsonutils.NewDict()
	suggestAlertCosts, err := self.getSuggestAlertCosts(ctx, userCred, query)
	if err != nil {
		return jsonutils.NewDict(), err
	}
	costData.Add(jsonutils.Marshal(&suggestAlertCosts), "suggest_cost")
	details, _ := query.Bool("details")
	if details {
		return costData, nil
	}
	meterCost, err := self.getMeterForcastCosts(ctx, userCred, query)
	if err != nil {
		log.Errorln(err)
		return costData, nil
	}
	costData.Add(jsonutils.Marshal(&meterCost), "meter_forcast_cost")
	return costData, nil
}

func (self *SSuggestSysAlertManager) getSuggestAlertCosts(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) ([]SuggestAlertCost, error) {
	suggestAlertCosts := make([]SuggestAlertCost, 0)
	details, _ := query.Bool("details")
	if details {
		return self.getDetailCosts(ctx, userCred, query)
	}
	cost, err := self.getCostWithSuggestAlertType("", ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	typeCost := SuggestAlertCost{
		Amount:   cost,
		CostType: "all",
	}
	suggestAlertCosts = append(suggestAlertCosts, typeCost)
	return suggestAlertCosts, nil
}

func (self *SSuggestSysAlertManager) getDetailCosts(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) ([]SuggestAlertCost, error) {
	suggestAlertCosts := make([]SuggestAlertCost, 0)
	types, err := self.getSuggestAlertTypes()
	if err != nil {
		return nil, err
	}
	allTypeCost := SuggestAlertCost{
		CostType: "all",
		Amount:   0,
	}

	for _, typ := range types {
		cost, err := self.getCostWithSuggestAlertType(typ, ctx, userCred, query)
		if err != nil {
			return nil, err
		}
		typeCost := SuggestAlertCost{
			Amount:   cost,
			CostType: typ,
		}
		allTypeCost.Amount += cost
		suggestAlertCosts = append(suggestAlertCosts, typeCost)
	}
	suggestAlertCosts = append(suggestAlertCosts, allTypeCost)
	return suggestAlertCosts, nil
}

func (self *SSuggestSysAlertManager) getMeterForcastCosts(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (SuggestAlertCost, error) {
	meterCost := SuggestAlertCost{
		CostType: "",
		Amount:   0,
	}
	domainId := ""
	projectId := ""
	reqScope, _ := query.GetString("scope")
	switch reqScope {
	case "system":
	case "domain":
		domainId = userCred.GetProjectDomain()
	default:
		projectId = userCred.GetProjectId()
	}
	session := auth.GetAdminSession(ctx, "", "")
	param := jsonutils.NewDict()
	if len(domainId) > 0 {
		param.Add(jsonutils.NewString(domainId), "domain_id")
	}
	if len(projectId) > 0 {
		param.Add(jsonutils.NewString(projectId), "project_id")
	}
	meterRtn, err := modules.AmountEstimations.GetById(session, "month", param)
	if err != nil {
		return meterCost, err
	}
	amount, _ := meterRtn.Float("amount")
	meterCost.Amount = amount
	return meterCost, nil
}

func (self *SSuggestSysAlertManager) getCostWithSuggestAlertType(typ string, ctx context.Context,
	userCred mcclient.TokenCredential,
	param jsonutils.JSONObject) (float64, error) {
	typeCost := float64(0)
	queryScope := rbacutils.ScopeProject
	suggestAlerts := make([]SSuggestSysAlert, 0)
	query := self.Query("amount")
	if len(typ) > 0 {
		query.Equals("type", typ)
	}
	reqScope, _ := param.GetString("scope")
	if len(reqScope) > 0 {
		queryScope = rbacutils.String2Scope(reqScope)
	}
	query = self.FilterByOwner(query, userCred, queryScope)
	err := db.FetchModelObjects(self, query, &suggestAlerts)
	if err != nil {
		log.Errorln(errors.Wrap(err, "getCostWithSuggestAlertType"))
		return 0, err
	}
	for _, suggestAlert := range suggestAlerts {
		typeCost += suggestAlert.Amount
	}
	return typeCost, nil
}

func (self *SSuggestSysAlertManager) getSuggestAlertTypes() ([]string, error) {
	suggestAlerts := make([]SSuggestSysAlert, 0)
	query := self.Query("type").Distinct()

	err := db.FetchModelObjects(self, query, &suggestAlerts)
	if err != nil {
		log.Errorln(errors.Wrap(err, "getSuggestAlertTypes"))
		return nil, err
	}
	types := make([]string, 0)
	for _, suggestAlert := range suggestAlerts {
		types = append(types, suggestAlert.Type)
	}
	return types, nil
}
