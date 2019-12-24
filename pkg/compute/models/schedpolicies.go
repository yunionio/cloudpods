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
)

type SSchedpolicyManager struct {
	db.SStandaloneResourceBaseManager
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

	Condition  string `width:"256" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	SchedtagId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`
	Strategy   string `width:"32" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`

	Enabled tristate.TriState `nullable:"false" default:"true" create:"optional" list:"user" update:"user"`
}

func validateSchedpolicyInputData(data *jsonutils.JSONDict, create bool) error {
	err := validateDynamicSchedtagInputData(data, create)
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

func (self *SSchedpolicyManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SSchedpolicyManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SSchedpolicy) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SSchedpolicy) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SSchedpolicy) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SSchedpolicyManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	err := validateSchedpolicyInputData(data, true)
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
	err := validateSchedpolicyInputData(data, false)
	if err != nil {
		return nil, err
	}

	return self.SStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SSchedpolicy) getSchedtag() *SSchedtag {
	obj, err := SchedtagManager.FetchById(self.SchedtagId)
	if err != nil {
		log.Errorf("fail to fetch sched tag by id %s", err)
		return nil
	}
	return obj.(*SSchedtag)
}

func (self *SSchedpolicy) getMoreColumns(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	schedtag := self.getSchedtag()
	if schedtag != nil {
		extra.Add(jsonutils.NewString(schedtag.GetName()), "schedtag")
		extra.Add(jsonutils.NewString(schedtag.ResourceType), "resource_type")
	}
	return extra
}

func (self *SSchedpolicy) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreColumns(extra)
}

func (self *SSchedpolicy) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreColumns(extra), nil
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

func (self *SSchedpolicy) AllowPerformEvaluate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "evaluate")
}

func (self *SSchedpolicy) PerformEvaluate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	objectId := jsonutils.GetAnyString(data, []string{"object", "object_id"})
	resType := jsonutils.GetAnyString(data, []string{"resource_type"})
	resMan := DynamicschedtagManager.VirtualResourcesManager[resType]
	if resMan == nil {
		return nil, httperrors.NewNotAcceptableError("ResourceType %q not support", resType)
	}
	obj, err := FetchDynamicResourceObject(resMan, userCred, objectId)
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
	schedtags := make(map[string]string)

	for _, tag := range oldTags {
		schedtags[tag.Id] = tag.Strategy
	}

	log.Infof("original schedtag %#v", schedtags)

	for i := 0; i < len(policies); i += 1 {
		policy := policies[i]
		st := policy.getSchedtag()
		if matchResourceSchedPolicy(policy, input) {
			schedtags[st.Name] = policy.Strategy
		}
	}
	log.Infof("updated sched tag %s", schedtags)

	newSchedtags := make([]*api.SchedtagConfig, 0)
	for name, strategy := range schedtags {
		newSchedtags = append(newSchedtags, &api.SchedtagConfig{
			Id:       name,
			Strategy: strategy,
		})
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
