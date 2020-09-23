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
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SLoadbalancerListenerRuleManager struct {
	SLoadbalancerLogSkipper
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SLoadbalancerListenerResourceBaseManager
}

var LoadbalancerListenerRuleManager *SLoadbalancerListenerRuleManager

func init() {
	LoadbalancerListenerRuleManager = &SLoadbalancerListenerRuleManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerListenerRule{},
			"loadbalancerlistenerrules_tbl",
			"loadbalancerlistenerrule",
			"loadbalancerlistenerrules",
		),
	}
	LoadbalancerListenerRuleManager.SetVirtualObject(LoadbalancerListenerRuleManager)
}

type SLoadbalancerListenerRule struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SLoadbalancerListenerResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	// 默认转发策略，目前只有aws用到其它云都是false
	IsDefault bool `default:"false" nullable:"true" list:"user" create:"optional"`

	// ListenerId     string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	BackendGroupId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`

	Domain    string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	Path      string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	Condition string `charset:"ascii" nullable:"true" list:"user" create:"optional"`

	SLoadbalancerHealthCheck // 目前只有腾讯云HTTP、HTTPS类型的健康检查是和规则绑定的。
	SLoadbalancerHTTPRateLimiter
	SLoadbalancerHTTPRedirect
}

func ValidateListenerRuleConditions(condition string) error {
	// total limit 5
	// host-header  limit 1
	// path-pattern limit 1
	// source-ip limit 1
	// http-request-method limit 1
	// http-header  no limit
	// query-string no limit
	limitations := &map[string]int{
		"rules":               5,
		"http-header":         5,
		"query-string":        5,
		"path-pattern":        1,
		"http-request-method": 1,
		"host-header":         1,
		"source-ip":           1,
	}

	obj, err := jsonutils.ParseString(condition)
	if err != nil {
		return httperrors.NewInputParameterError("invalid conditions format,required json")
	}

	conditionArray, ok := obj.(*jsonutils.JSONArray)
	if !ok {
		return httperrors.NewInputParameterError("invalid conditions fromat,required json array")
	}

	if conditionArray.Length() > 5 {
		return httperrors.NewInputParameterError("condition values limit (5 per rule). %d given.", conditionArray.Length())
	}

	cs, _ := conditionArray.GetArray()
	for i := range cs {
		err := validateListenerRuleCondition(cs[i], limitations)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateListenerRuleCondition(condition jsonutils.JSONObject, limitations *map[string]int) error {
	conditionDict, ok := condition.(*jsonutils.JSONDict)
	if !ok {
		return fmt.Errorf("invalid condition fromat,required dict. %#v", condition)
	}

	dict, _ := conditionDict.GetMap()
	field, ok := dict["field"]
	if !ok {
		return fmt.Errorf("parseCondition invalid condition, missing field: %#v", condition)
	}

	f, _ := field.GetString()
	switch f {
	case "http-header":
		return parseHttpHeaderCondition(conditionDict, limitations)
	case "path-pattern":
		return parsePathPatternCondition(conditionDict, limitations)
	case "http-request-method":
		return parseRequestModthdCondition(conditionDict, limitations)
	case "host-header":
		return parseHostHeaderCondition(conditionDict, limitations)
	case "query-string":
		return parseQueryStringCondition(conditionDict, limitations)
	case "source-ip":
		return parseSourceIpCondition(conditionDict, limitations)
	default:
		return fmt.Errorf("parseCondition invalid condition key %#v", field)
	}
}

func parseHttpHeaderCondition(conditon *jsonutils.JSONDict, limitations *map[string]int) error {
	(*limitations)["http-header"] = (*limitations)["http-header"] - 1
	if (*limitations)["http-header"] < 0 {
		return fmt.Errorf("http-header exceeded limiation.")
	}

	values, err := conditon.GetMap("httpHeaderConfig")
	if err != nil {
		return err
	}

	name, ok := values["HttpHeaderName"]
	if !ok {
		return fmt.Errorf("parseHttpHeaderCondition missing filed HttpHeaderName")
	}

	_, ok = name.(*jsonutils.JSONString)
	if !ok {
		return fmt.Errorf("parseHttpHeaderCondition missing invalid data %#v", name)
	}

	vs, ok := values["values"]
	if !ok {
		return fmt.Errorf("parseHttpHeaderCondition missing filed values")
	}

	err = parseConditionStringArrayValues(vs, limitations)
	if err != nil {
		return err
	}

	return nil
}

func parsePathPatternCondition(condition *jsonutils.JSONDict, limitations *map[string]int) error {
	(*limitations)["path-pattern"] = (*limitations)["path-pattern"] - 1
	if (*limitations)["path-pattern"] < 0 {
		return fmt.Errorf("path-pattern exceeded limiation.")
	}

	values, err := condition.GetMap("pathPatternConfig")
	if err != nil {
		return err
	}

	vs, ok := values["values"]
	if !ok {
		return fmt.Errorf("parsePathPatternCondition missing filed values")
	}

	err = parseConditionStringArrayValues(vs, limitations)
	if err != nil {
		return err
	}

	return nil

}

func parseRequestModthdCondition(condition *jsonutils.JSONDict, limitations *map[string]int) error {
	(*limitations)["http-request-method"] = (*limitations)["http-request-method"] - 1
	if (*limitations)["http-request-method"] < 0 {
		return fmt.Errorf("http-request-method exceeded limiation.")
	}

	values, err := condition.GetMap("httpRequestMethodConfig")
	if err != nil {
		return err
	}

	vs, ok := values["values"]
	if !ok {
		return fmt.Errorf("parseRequestModthdCondition missing filed values")
	}

	err = parseConditionStringArrayValues(vs, limitations)
	if err != nil {
		return err
	}

	return nil
}

func parseHostHeaderCondition(condition *jsonutils.JSONDict, limitations *map[string]int) error {
	(*limitations)["host-header"] = (*limitations)["host-header"] - 1
	if (*limitations)["host-header"] < 0 {
		return fmt.Errorf("host-header exceeded limiation.")
	}

	values, err := condition.GetMap("hostHeaderConfig")
	if err != nil {
		return err
	}

	vs, ok := values["values"]
	if !ok {
		return fmt.Errorf("parseHostHeaderCondition missing filed values")
	}

	err = parseConditionStringArrayValues(vs, limitations)
	if err != nil {
		return err
	}

	return nil
}

func parseQueryStringCondition(condition *jsonutils.JSONDict, limitations *map[string]int) error {
	(*limitations)["query-string"] = (*limitations)["query-string"] - 1
	if (*limitations)["query-string"] < 0 {
		return fmt.Errorf("query-string exceeded limiation.")
	}

	values, err := condition.GetMap("queryStringConfig")
	if err != nil {
		return err
	}

	vs, ok := values["values"]
	if !ok {
		return fmt.Errorf("parseQueryStringCondition missing filed values")
	}

	err = parseConditionDictArrayValues(vs, limitations)
	if err != nil {
		return err
	}

	return nil
}

func parseSourceIpCondition(condition *jsonutils.JSONDict, limitations *map[string]int) error {
	(*limitations)["source-ip"] = (*limitations)["source-ip"] - 1
	if (*limitations)["source-ip"] < 0 {
		return fmt.Errorf("source-ip exceeded limiation.")
	}

	values, err := condition.GetMap("sourceIpConfig")
	if err != nil {
		return err
	}

	vs, ok := values["values"]
	if !ok {
		return fmt.Errorf("parseSourceIpCondition missing filed values")
	}

	err = parseConditionStringArrayValues(vs, limitations)
	if err != nil {
		return err
	}

	return nil
}

func parseConditionStringArrayValues(values jsonutils.JSONObject, limitations *map[string]int) error {
	objs, ok := values.(*jsonutils.JSONArray)
	if !ok {
		return fmt.Errorf("parseConditionStringArrayValues invalid values format, required array: %#v", values)
	}

	vs, _ := objs.GetArray()
	for i := range vs {
		(*limitations)["rules"] = (*limitations)["rules"] - 1
		if (*limitations)["rules"] < 0 {
			return fmt.Errorf("rules exceeded limiation.")
		}

		v, ok := vs[i].(*jsonutils.JSONString)
		if !ok {
			return fmt.Errorf("parseConditionStringArrayValues invalid value, required string: %#v", v)
		}
	}

	return nil
}

func parseConditionDictArrayValues(values jsonutils.JSONObject, limitations *map[string]int) error {
	objs, ok := values.(*jsonutils.JSONArray)
	if !ok {
		return fmt.Errorf("parseConditionDictArrayValues invalid values format, required array: %#v", values)
	}

	vs, _ := objs.GetArray()
	for i := range vs {
		(*limitations)["rules"] = (*limitations)["rules"] - 1
		if (*limitations)["rules"] < 0 {
			return fmt.Errorf("rules exceeded limiation.")
		}

		v, ok := vs[i].(*jsonutils.JSONDict)
		if !ok {
			return fmt.Errorf("parseConditionDictArrayValues invalid value, required dict: %#v", v)
		}

		_, err := v.GetString("key")
		if err != nil {
			return err
		}

		_, err = v.GetString("value")
		if err != nil {
			return err
		}
	}

	return nil
}

func LoadbalancerListenerRuleCheckUniqueness(ctx context.Context, lbls *SLoadbalancerListener, domain, path string) error {
	q := LoadbalancerListenerRuleManager.Query().
		IsFalse("pending_deleted").
		Equals("listener_id", lbls.Id).
		Equals("domain", domain).
		Equals("path", path)
	var lblsr SLoadbalancerListenerRule
	q.First(&lblsr)
	if len(lblsr.Id) > 0 {
		return httperrors.NewConflictError("rule %s/%s already occupied by rule %s(%s)", domain, path, lblsr.Name, lblsr.Id)
	}
	return nil
}

func (man *SLoadbalancerListenerRuleManager) pendingDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	subs := []SLoadbalancerListenerRule{}
	db.FetchModelObjects(man, q, &subs)
	for _, sub := range subs {
		sub.DoPendingDelete(ctx, userCred)
	}
}

// 负载均衡监听器规则列表
func (man *SLoadbalancerListenerRuleManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerListenerRuleListInput,
) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = man.SLoadbalancerListenerResourceBaseManager.ListItemFilter(ctx, q, userCred, query.LoadbalancerListenerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerListenerResourceBaseManager.ListItemFilter")
	}

	// userProjId := userCred.GetProjectId()
	data := jsonutils.Marshal(query).(*jsonutils.JSONDict)
	q, err = validators.ApplyModelFilters(q, data, []*validators.ModelFilterOptions{
		// {Key: "listener", ModelKeyword: "loadbalancerlistener", OwnerId: userCred},
		{Key: "backend_group", ModelKeyword: "loadbalancerbackendgroup", OwnerId: userCred},
	})
	if err != nil {
		return nil, err
	}

	if query.IsDefault != nil {
		if *query.IsDefault {
			q = q.IsTrue("is_default")
		} else {
			q = q.IsFalse("is_default")
		}
	}
	if len(query.Domain) > 0 {
		q = q.In("domain", query.Domain)
	}
	if len(query.Path) > 0 {
		q = q.In("path", query.Path)
	}

	return q, nil
}

func (man *SLoadbalancerListenerRuleManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.LoadbalancerListenerRuleListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SLoadbalancerListenerResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.LoadbalancerListenerFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SLoadbalancerListenerResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (man *SLoadbalancerListenerRuleManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = man.SLoadbalancerListenerResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (man *SLoadbalancerListenerRuleManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	listenerV := validators.NewModelIdOrNameValidator("listener", "loadbalancerlistener", nil)
	if err := listenerV.Validate(data.(*jsonutils.JSONDict)); err == nil {
		return listenerV.Model.GetOwnerId(), nil
	}
	return man.SVirtualResourceBaseManager.FetchOwnerId(ctx, data)
}

func (man *SLoadbalancerListenerRuleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input := apis.VirtualResourceCreateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))

	listenerV := validators.NewModelIdOrNameValidator("listener", "loadbalancerlistener", ownerId)
	if err := listenerV.Validate(data); err != nil {
		return nil, err
	}

	listener := listenerV.Model.(*SLoadbalancerListener)
	region := listener.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find region for loadbalancer listener %s", listener.Name)
	}

	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerId)
	if region.GetDriver().IsSupportLoadbalancerListenerRuleRedirect() {
		// backend group can be empty if you support redirect in rule
		backendGroupV.Optional(true)
	}
	if err := backendGroupV.Validate(data); err != nil {
		return nil, err
	}

	return region.GetDriver().ValidateCreateLoadbalancerListenerRuleData(ctx, userCred, ownerId, data, backendGroupV.Model)
}

func (lbr *SLoadbalancerListenerRule) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	lbr.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	lbr.SetStatus(userCred, api.LB_CREATING, "")
	if err := lbr.StartLoadBalancerListenerRuleCreateTask(ctx, userCred, ""); err != nil {
		log.Errorf("Failed to create loadbalancer listener rule error: %v", err)
	}
}

func (lbr *SLoadbalancerListenerRule) StartLoadBalancerListenerRuleCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerRuleCreateTask", lbr, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbr *SLoadbalancerListenerRule) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, lbr, "purge")
}

func (lbr *SLoadbalancerListenerRule) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	parasm := jsonutils.NewDict()
	parasm.Add(jsonutils.JSONTrue, "purge")
	return nil, lbr.StartLoadBalancerListenerRuleDeleteTask(ctx, userCred, parasm, "")
}

func (lbr *SLoadbalancerListenerRule) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lbr.SetStatus(userCred, api.LB_STATUS_DELETING, "")
	return lbr.StartLoadBalancerListenerRuleDeleteTask(ctx, userCred, jsonutils.NewDict(), "")
}

func (lbr *SLoadbalancerListenerRule) StartLoadBalancerListenerRuleDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, params *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LoadbalancerListenerRuleDeleteTask", lbr, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (lbr *SLoadbalancerListenerRule) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return lbr.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lbr, "status")
}

func (lbr *SLoadbalancerListenerRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", lbr.GetOwnerId())
	if lbr.BackendGroupId != "" {
		backendGroupV.Default(lbr.BackendGroupId)
	} else {
		backendGroupV.Optional(true)
	}
	if err := backendGroupV.Validate(data); err != nil {
		return nil, err
	}

	input := apis.VirtualResourceBaseUpdateInput{}
	err := data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = lbr.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	data.Update(jsonutils.Marshal(input))

	region := lbr.GetRegion()
	if region == nil {
		return nil, httperrors.NewResourceNotFoundError("failed to find region for loadbalancer listener rule %s", lbr.Name)
	}

	ctx = context.WithValue(ctx, "lbr", lbr)
	return region.GetDriver().ValidateUpdateLoadbalancerListenerRuleData(ctx, userCred, data, backendGroupV.Model)
}

func (lbr *SLoadbalancerListenerRule) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.LoadbalancerListenerRuleDetails, error) {
	return api.LoadbalancerListenerRuleDetails{}, nil
}

func (lbr *SLoadbalancerListenerRule) getMoreDetails(out api.LoadbalancerListenerRuleDetails) (api.LoadbalancerListenerRuleDetails, error) {
	if lbr.BackendGroupId == "" {
		log.Errorf("loadbalancer listener rule %s(%s): empty backend group field", lbr.Name, lbr.Id)
		return out, nil
	}
	lbbg, err := LoadbalancerBackendGroupManager.FetchById(lbr.BackendGroupId)
	if err != nil {
		log.Errorf("loadbalancer listener rule %s(%s): fetch backend group (%s) error: %s",
			lbr.Name, lbr.Id, lbr.BackendGroupId, err)
		return out, err
	}
	out.BackendGroup = lbbg.GetName()

	return out, nil
}

func (man *SLoadbalancerListenerRuleManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LoadbalancerListenerRuleDetails {
	rows := make([]api.LoadbalancerListenerRuleDetails, len(objs))

	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	listenerRows := man.SLoadbalancerListenerResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.LoadbalancerListenerRuleDetails{
			VirtualResourceDetails:           virtRows[i],
			LoadbalancerListenerResourceInfo: listenerRows[i],
		}
		rows[i], _ = objs[i].(*SLoadbalancerListenerRule).getMoreDetails(rows[i])
	}

	return rows
}

/*func (lbr *SLoadbalancerListenerRule) GetLoadbalancerListener() *SLoadbalancerListener {
	listener, err := LoadbalancerListenerManager.FetchById(lbr.ListenerId)
	if err != nil {
		log.Errorf("failed to find listener for loadbalancer listener rule %s", lbr.Name)
		return nil
	}
	return listener.(*SLoadbalancerListener)
}*/

func (lbr *SLoadbalancerListenerRule) GetRegion() *SCloudregion {
	if listener := lbr.GetLoadbalancerListener(); listener != nil {
		return listener.GetRegion()
	}
	return nil
}

func (lbr *SLoadbalancerListenerRule) GetLoadbalancerBackendGroup() *SLoadbalancerBackendGroup {
	group, err := LoadbalancerBackendGroupManager.FetchById(lbr.BackendGroupId)
	if err != nil {
		return nil
	}
	return group.(*SLoadbalancerBackendGroup)
}

func (lbr *SLoadbalancerListenerRule) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

// Delete, Update

func (man *SLoadbalancerListenerRuleManager) getLoadbalancerListenerRulesByListener(listener *SLoadbalancerListener) ([]SLoadbalancerListenerRule, error) {
	rules := []SLoadbalancerListenerRule{}
	q := man.Query().Equals("listener_id", listener.Id).IsFalse("pending_deleted")
	if err := db.FetchModelObjects(man, q, &rules); err != nil {
		log.Errorf("failed to get lb listener rules for listener %s error: %v", listener.Name, err)
		return nil, err
	}
	return rules, nil
}

func (man *SLoadbalancerListenerRuleManager) SyncLoadbalancerListenerRules(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, listener *SLoadbalancerListener, rules []cloudprovider.ICloudLoadbalancerListenerRule, syncRange *SSyncRange) compare.SyncResult {
	syncOwnerId := provider.GetOwnerId()

	lockman.LockRawObject(ctx, "listener-rules", listener.Id)
	defer lockman.ReleaseRawObject(ctx, "listener-rules", listener.Id)

	syncResult := compare.SyncResult{}

	dbRules, err := man.getLoadbalancerListenerRulesByListener(listener)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SLoadbalancerListenerRule{}
	commondb := []SLoadbalancerListenerRule{}
	commonext := []cloudprovider.ICloudLoadbalancerListenerRule{}
	added := []cloudprovider.ICloudLoadbalancerListenerRule{}

	err = compare.CompareSets(dbRules, rules, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].syncRemoveCloudLoadbalancerListenerRule(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerListenerRule(ctx, userCred, commonext[i], syncOwnerId, provider)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		local, err := man.newFromCloudLoadbalancerListenerRule(ctx, userCred, listener, added[i], syncOwnerId, provider)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, local, added[i])
			syncResult.Add()
		}
	}

	return syncResult
}

func (lbr *SLoadbalancerListenerRule) constructFieldsFromCloudListenerRule(userCred mcclient.TokenCredential, extRule cloudprovider.ICloudLoadbalancerListenerRule) {
	// lbr.Name = extRule.GetName()
	lbr.IsDefault = extRule.IsDefault()
	lbr.Domain = extRule.GetDomain()
	lbr.Path = extRule.GetPath()
	lbr.Status = extRule.GetStatus()
	lbr.Condition = extRule.GetCondition()
	if groupId := extRule.GetBackendGroupId(); len(groupId) > 0 {
		if lbr.GetProviderName() == api.CLOUD_PROVIDER_HUAWEI {
			group, err := db.FetchByExternalId(HuaweiCachedLbbgManager, groupId)
			if err != nil {
				if err == sql.ErrNoRows {
					lbr.BackendGroupId = ""
				}
				log.Errorf("Fetch huawei loadbalancer backendgroup by external id %s failed: %s", groupId, err)
			}

			lbr.BackendGroupId = group.(*SHuaweiCachedLbbg).BackendGroupId
		} else if lbr.GetProviderName() == api.CLOUD_PROVIDER_AWS {
			if len(groupId) > 0 {
				group, err := db.FetchByExternalId(AwsCachedLbbgManager, groupId)
				if err != nil {
					log.Errorf("Fetch aws loadbalancer backendgroup by external id %s failed: %s", groupId, err)
				}

				lbr.BackendGroupId = group.(*SAwsCachedLbbg).BackendGroupId
			}
		} else if lbr.GetProviderName() == api.CLOUD_PROVIDER_QCLOUD {
			group, err := db.FetchByExternalId(QcloudCachedLbbgManager, groupId)
			if err != nil {
				if err == sql.ErrNoRows {
					lbr.BackendGroupId = ""
				}
				log.Errorf("Fetch qcloud loadbalancer backendgroup by external id %s failed: %s", groupId, err)
			}

			lbr.BackendGroupId = group.(*SQcloudCachedLbbg).BackendGroupId
		} else if backendgroup, err := db.FetchByExternalId(LoadbalancerBackendGroupManager, groupId); err == nil {
			lbr.BackendGroupId = backendgroup.GetId()
		}
	}
}

func (lbr *SLoadbalancerListenerRule) updateCachedLoadbalancerBackendGroupAssociate(ctx context.Context, extRule cloudprovider.ICloudLoadbalancerListenerRule) error {
	exteralLbbgId := extRule.GetBackendGroupId()
	if len(exteralLbbgId) == 0 {
		return nil
	}

	switch lbr.GetProviderName() {
	case api.CLOUD_PROVIDER_HUAWEI:
		_group, err := db.FetchByExternalId(HuaweiCachedLbbgManager, exteralLbbgId)
		if err != nil {
			if err == sql.ErrNoRows {
				lbr.BackendGroupId = ""
			}
			return fmt.Errorf("Fetch huawei loadbalancer backendgroup by external id %s failed: %s", exteralLbbgId, err)
		}

		if _group != nil {
			group := _group.(*SHuaweiCachedLbbg)
			if group.AssociatedId != lbr.Id {
				_, err := db.UpdateWithLock(ctx, group, func() error {
					group.AssociatedId = lbr.Id
					group.AssociatedType = api.LB_ASSOCIATE_TYPE_RULE
					return nil
				})
				if err != nil {
					return errors.Wrap(err, "LoadbalancerListener.updateCachedLoadbalancerBackendGroupAssociate.huawei")
				}
			}
		}
	case api.CLOUD_PROVIDER_QCLOUD:
		_group, err := db.FetchByExternalId(QcloudCachedLbbgManager, exteralLbbgId)
		if err != nil {
			if err == sql.ErrNoRows {
				lbr.BackendGroupId = ""
			}
			return fmt.Errorf("Fetch qcloud loadbalancer backendgroup by external id %s failed: %s", exteralLbbgId, err)
		}

		if _group != nil {
			group := _group.(*SQcloudCachedLbbg)
			if group.AssociatedId != lbr.Id {
				_, err := db.UpdateWithLock(ctx, group, func() error {
					group.AssociatedId = lbr.Id
					group.AssociatedType = api.LB_ASSOCIATE_TYPE_RULE
					return nil
				})
				if err != nil {
					return errors.Wrap(err, "LoadbalancerListener.updateCachedLoadbalancerBackendGroupAssociate.qcloud")
				}
			}
		}
	case api.CLOUD_PROVIDER_OPENSTACK:
		_group, err := db.FetchByExternalId(OpenstackCachedLbbgManager, exteralLbbgId)
		if err != nil {
			if err == sql.ErrNoRows {
				lbr.BackendGroupId = ""
			}
			return fmt.Errorf("Fetch openstack loadbalancer backendgroup by external id %s failed: %s", exteralLbbgId, err)
		}

		if _group != nil {
			group := _group.(*SOpenstackCachedLbbg)
			if group.AssociatedId != lbr.Id {
				_, err := db.UpdateWithLock(ctx, group, func() error {
					group.AssociatedId = lbr.Id
					group.AssociatedType = api.LB_ASSOCIATE_TYPE_RULE
					return nil
				})
				if err != nil {
					return errors.Wrap(err, "LoadbalancerListener.updateCachedLoadbalancerBackendGroupAssociate.openstack")
				}
			}
		}
	default:
		return nil
	}

	return nil
}

func (man *SLoadbalancerListenerRuleManager) newFromCloudLoadbalancerListenerRule(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	listener *SLoadbalancerListener,
	extRule cloudprovider.ICloudLoadbalancerListenerRule,
	syncOwnerId mcclient.IIdentityProvider,
	provider *SCloudprovider,
) (*SLoadbalancerListenerRule, error) {
	lbr := &SLoadbalancerListenerRule{}
	lbr.SetModelManager(man, lbr)

	lbr.ExternalId = extRule.GetGlobalId()
	lbr.ListenerId = listener.Id
	//lbr.ManagerId = listener.ManagerId
	//lbr.CloudregionId = listener.CloudregionId

	lbr.constructFieldsFromCloudListenerRule(userCred, extRule)
	var err = func() error {
		lockman.LockRawObject(ctx, man.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, man.Keyword(), "name")

		newName, err := db.GenerateName(ctx, man, syncOwnerId, extRule.GetName())
		if err != nil {
			return err
		}
		lbr.Name = newName

		return man.TableSpec().Insert(ctx, lbr)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	err = lbr.updateCachedLoadbalancerBackendGroupAssociate(ctx, extRule)
	if err != nil {
		return nil, errors.Wrap(err, "LoadbalancerListenerRuleManager.newFromCloudLoadbalancerListenerRule")
	}

	SyncCloudProject(userCred, lbr, syncOwnerId, extRule, provider.Id)

	db.OpsLog.LogEvent(lbr, db.ACT_CREATE, lbr.GetShortDesc(ctx), userCred)

	return lbr, nil
}

func (lbr *SLoadbalancerListenerRule) syncRemoveCloudLoadbalancerListenerRule(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, lbr)
	defer lockman.ReleaseObject(ctx, lbr)

	err := lbr.ValidateDeleteCondition(ctx)
	if err != nil { // cannot delete
		err = lbr.SetStatus(userCred, api.LB_STATUS_UNKNOWN, "sync to delete")
	} else {
		err = lbr.DoPendingDelete(ctx, userCred)
	}
	return err
}

func (lbr *SLoadbalancerListenerRule) SyncWithCloudLoadbalancerListenerRule(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	extRule cloudprovider.ICloudLoadbalancerListenerRule,
	syncOwnerId mcclient.IIdentityProvider,
	provider *SCloudprovider,
) error {
	// listener := lbr.GetLoadbalancerListener()
	diff, err := db.UpdateWithLock(ctx, lbr, func() error {
		lbr.constructFieldsFromCloudListenerRule(userCred, extRule)
		// lbr.ManagerId = provider.Id
		// lbr.CloudregionId = listener.CloudregionId
		return nil
	})
	if err != nil {
		return err
	}

	err = lbr.updateCachedLoadbalancerBackendGroupAssociate(ctx, extRule)
	if err != nil {
		return errors.Wrap(err, "LoadbalancerListenerRule.SyncWithCloudLoadbalancerListenerRule")
	}

	db.OpsLog.LogSyncUpdate(lbr, diff, userCred)

	SyncCloudProject(userCred, lbr, syncOwnerId, extRule, provider.Id)

	return nil
}

/*func (manager *SLoadbalancerListenerRuleManager) InitializeData() error {
	rules := []SLoadbalancerListenerRule{}
	q := manager.Query()
	q = q.Filter(sqlchemy.IsNullOrEmpty(q.Field("cloudregion_id")))
	if err := db.FetchModelObjects(manager, q, &rules); err != nil {
		return err
	}
	for i := 0; i < len(rules); i++ {
		rule := &rules[i]
		if listener := rule.GetLoadbalancerListener(); listener != nil && len(listener.CloudregionId) > 0 {
			_, err := db.Update(rule, func() error {
				rule.CloudregionId = listener.CloudregionId
				rule.ManagerId = listener.ManagerId
				return nil
			})
			if err != nil {
				log.Errorf("failed to update loadbalancer listener rule %s cloudregion_id", rule.Name)
			}
		}
	}
	return nil
}*/

func (manager *SLoadbalancerListenerRuleManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	virts := manager.Query().IsFalse("pending_deleted")
	return db.CalculateResourceCount(virts, "tenant_id")
}

func (manager *SLoadbalancerListenerRuleManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SLoadbalancerListenerResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SLoadbalancerListenerResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SLoadbalancerListenerResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}
