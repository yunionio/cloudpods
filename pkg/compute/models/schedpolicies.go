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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/conditionparser"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSchedpolicyManager struct {
	db.SStandaloneResourceBaseManager
	SSchedtagResourceBaseManager
}

var SchedpolicyManager *SSchedpolicyManager

func init() {
	SchedpolicyManager = &SSchedpolicyManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SSchedpolicy{},
			"schedpolicies_tbl",
			"schedpolicy",
			"schedpolicies",
		),
	}
	SchedpolicyManager.SetVirtualObject(SchedpolicyManager)
}

// sched policy is called before calling scheduler, add additional preferences for schedtags
type SSchedpolicy struct {
	db.SStandaloneResourceBase
	SSchedtagResourceBase

	Condition string `width:"1024" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	Strategy  string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`

	Enabled tristate.TriState `default:"true" create:"optional" list:"user" update:"user"`
}

func validateSchedpolicyInputData(ctx context.Context, data *jsonutils.JSONDict, create bool) error {
	err := validateDynamicSchedtagInputData(ctx, data, create)
	if err != nil {
		return err
	}

	strategyStr := jsonutils.GetAnyString(data, []string{"strategy"})
	if len(strategyStr) == 0 && create {
		return httperrors.NewMissingParameterError("strategy")
	}

	if len(strategyStr) > 0 && !utils.IsInStringArray(strategyStr, STRATEGY_LIST) {
		return httperrors.NewInputParameterError("invalid strategy %s", strategyStr)
	}

	return nil
}

func (manager *SSchedpolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	err := validateSchedpolicyInputData(ctx, data, true)
	if err != nil {
		return nil, err
	}

	input := apis.StandaloneResourceCreateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, httperrors.NewInternalServerError("unmarshal StandaloneResourceCreateInput fail %s", err)
	}
	input, err = manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input)
	if err != nil {
		return nil, err
	}
	data.Update(jsonutils.Marshal(input))
	return data, nil
}

func (self *SSchedpolicy) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	err := validateSchedpolicyInputData(ctx, data, false)
	if err != nil {
		return nil, err
	}

	input := apis.StandaloneResourceBaseUpdateInput{}
	err = data.Unmarshal(&input)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	input, err = self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBase.ValidateUpdateData")
	}
	data.Update(jsonutils.Marshal(input))

	return data, nil
}

func (self *SSchedpolicy) getSchedtag() *SSchedtag {
	obj, err := SchedtagManager.FetchById(self.SchedtagId)
	if err != nil {
		log.Errorf("fail to fetch sched tag by id %s", err)
		return nil
	}
	return obj.(*SSchedtag)
}

func (manager *SSchedpolicyManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SchedpolicyDetails {
	rows := make([]api.SchedpolicyDetails, len(objs))

	stdRows := manager.SStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	tagRows := manager.SSchedtagResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.SchedpolicyDetails{
			StandaloneResourceDetails: stdRows[i],
			SchedtagResourceInfo:      tagRows[i],
		}
	}

	return rows
}

func (manager *SSchedpolicyManager) getAllEnabledPoliciesByResource(resType string) []SSchedpolicy {
	policies := make([]SSchedpolicy, 0)
	q := SchedpolicyManager.Query().IsTrue("enabled")
	schedtags := SchedtagManager.Query().SubQuery()
	q = q.Join(schedtags, sqlchemy.AND(
		sqlchemy.Equals(q.Field("schedtag_id"), schedtags.Field("id")),
		sqlchemy.Equals(schedtags.Field("resource_type"), resType)))
	err := db.FetchModelObjects(manager, q, &policies)
	if err != nil {
		log.Errorf("getAllEnabledPolicies fail %s", err)
		return nil
	}
	return policies
}

func (manager *SSchedpolicyManager) getHostEnabledPolicies() []SSchedpolicy {
	return manager.getAllEnabledPoliciesByResource(HostManager.KeywordPlural())
}

func (manager *SSchedpolicyManager) getStorageEnabledPolicies() []SSchedpolicy {
	return manager.getAllEnabledPoliciesByResource(StorageManager.KeywordPlural())
}

func (manager *SSchedpolicyManager) getNetworkEnabledPolicies() []SSchedpolicy {
	return manager.getAllEnabledPoliciesByResource(NetworkManager.KeywordPlural())
}

func (self *SSchedpolicy) PerformEvaluate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	objectId := jsonutils.GetAnyString(data, []string{"object", "object_id"})
	resType := jsonutils.GetAnyString(data, []string{"resource_type"})
	resMan := DynamicschedtagManager.VirtualResourcesManager[resType]
	if resMan == nil {
		return nil, httperrors.NewNotAcceptableError("ResourceType %q not support", resType)
	}
	obj, err := FetchDynamicResourceObject(ctx, resMan, userCred, objectId)
	if err != nil {
		return nil, err
	}

	desc := obj.GetDynamicConditionInput()

	params := jsonutils.NewDict()
	params.Add(desc, obj.Keyword())

	log.V(10).Debugf("Schedpolicy evaluate input: %s", params.PrettyString())

	meet, err := conditionparser.EvalBool(self.Condition, params)
	if err != nil {
		return nil, err
	}
	result := jsonutils.NewDict()
	result.Add(desc, obj.Keyword())
	if meet {
		result.Add(jsonutils.JSONTrue, "result")
	} else {
		result.Add(jsonutils.JSONFalse, "result")
	}
	return result, nil
}

func matchResourceSchedPolicy(
	policy SSchedpolicy,
	input *jsonutils.JSONDict,
) bool {
	meet, err := conditionparser.EvalBool(policy.Condition, input)
	if err != nil {
		log.Errorf("Eval Condition %s error: %v", policy.Condition, err)
		return false
	}
	return meet
}

func applyResourceSchedPolicy(
	policies []SSchedpolicy,
	oldTags []*api.SchedtagConfig,
	input *jsonutils.JSONDict,
	setTags func([]*api.SchedtagConfig),
) {
	schedtags := make(map[string]*api.SchedtagConfig)

	for _, tag := range oldTags {
		schedtags[tag.Id] = tag
	}

	log.Infof("original schedtag %#v", schedtags)

	for i := 0; i < len(policies); i += 1 {
		policy := policies[i]
		st := policy.getSchedtag()
		if matchResourceSchedPolicy(policy, input) {
			if conf, idOk := schedtags[st.GetId()]; idOk {
				conf.Id = st.GetId()
				conf.Strategy = policy.Strategy
				schedtags[st.GetId()] = conf
			} else if conf, nameOk := schedtags[st.GetName()]; nameOk {
				conf.Id = st.GetId()
				conf.Strategy = policy.Strategy
				schedtags[st.GetId()] = conf
				delete(schedtags, st.GetName())
			} else {
				schedtags[st.GetId()] = &api.SchedtagConfig{
					Id:           st.GetId(),
					Strategy:     policy.Strategy,
					ResourceType: st.ResourceType,
				}
			}
		}
	}
	log.Infof("updated sched tag %#v", schedtags)

	newSchedtags := make([]*api.SchedtagConfig, 0)
	for _, tag := range schedtags {
		newSchedtags = append(newSchedtags, tag)
	}
	setTags(newSchedtags)
}

func GetDynamicConditionInput(man IDynamicResourceManager, input *jsonutils.JSONDict) *jsonutils.JSONDict {
	ret := jsonutils.NewDict()
	ret.Add(input, man.Keyword())
	return ret
}

func applyServerSchedtags(policies []SSchedpolicy, input *schedapi.ScheduleInput) {
	inputCond := GetDynamicConditionInput(GuestManager, input.ToConditionInput())
	setFunc := func(tags []*api.SchedtagConfig) {
		input.Schedtags = tags
	}
	applyResourceSchedPolicy(policies, input.Schedtags, inputCond, setFunc)
}

func applyDiskSchedtags(policies []SSchedpolicy, input *api.DiskConfig) {
	inputCond := GetDynamicConditionInput(DiskManager, jsonutils.Marshal(input).(*jsonutils.JSONDict))
	setFunc := func(tags []*api.SchedtagConfig) {
		input.Schedtags = tags
	}
	applyResourceSchedPolicy(policies, input.Schedtags, inputCond, setFunc)
}

func applyNetworkSchedtags(policies []SSchedpolicy, input *api.NetworkConfig) {
	inputCond := GetDynamicConditionInput(NetworkManager, jsonutils.Marshal(input).(*jsonutils.JSONDict))
	setFunc := func(tags []*api.SchedtagConfig) {
		input.Schedtags = tags
	}
	applyResourceSchedPolicy(policies, input.Schedtags, inputCond, setFunc)
}

func ApplySchedPolicies(input *schedapi.ScheduleInput) *schedapi.ScheduleInput {
	// TODO: refactor this duplicate code
	hostPolicies := SchedpolicyManager.getHostEnabledPolicies()
	storagePolicies := SchedpolicyManager.getStorageEnabledPolicies()
	networkPolicies := SchedpolicyManager.getNetworkEnabledPolicies()

	config := input.ServerConfigs

	applyServerSchedtags(hostPolicies, input)
	for _, disk := range config.Disks {
		applyDiskSchedtags(storagePolicies, disk)
	}
	for _, net := range config.Networks {
		applyNetworkSchedtags(networkPolicies, net)
	}

	input.ServerConfig.ServerConfigs = config

	return input
}

// 动态调度策略列表
func (manager *SSchedpolicyManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.SchedpolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SSchedtagResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SchedtagFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagResourceBaseManager.ListItemFilter")
	}

	if len(input.Strategy) > 0 {
		q = q.In("strategy", input.Strategy)
	}
	if input.Enabled != nil {
		if *input.Enabled {
			q = q.IsTrue("enabled")
		} else {
			q = q.IsFalse("enabled")
		}
	}

	return q, nil
}

func (manager *SSchedpolicyManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.SchedpolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SSchedtagResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.SchedtagFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SSchedpolicyManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SSchedtagResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}
