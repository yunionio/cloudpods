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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SAiRouting stores a project-scoped (and optionally shared) routing rule.
type SAiRouting struct {
	db.SSharableVirtualResourceBase
	db.SEnabledResourceBase

	Priority int `default:"100" nullable:"false" list:"user" create:"optional" update:"user"`
	// ModelPattern optionally matches the requested model id (implementation-specific glob/prefix).
	ModelPattern string `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	// AiProxyNodeId optionally binds the rule to one aiproxy instance (ai_proxy_node id).
	AiProxyNodeId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	// LlmDeploymentId links this routing to an llm_deployment (set by llm sync).
	LlmDeploymentId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user" index:"true"`
}

type SAiRoutingManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

var AiRoutingManager *SAiRoutingManager

func init() {
	AiRoutingManager = &SAiRoutingManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SAiRouting{},
			"ai_routings_tbl",
			"ai_routing",
			"ai_routings",
		),
	}
	AiRoutingManager.SetVirtualObject(AiRoutingManager)
}

func (manager *SAiRoutingManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.AiRoutingListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	if v := strings.TrimSpace(query.ModelPattern); v != "" {
		q = q.Equals("model_pattern", v)
	}
	if v := strings.TrimSpace(query.AiProxyNodeId); v != "" {
		q = q.Equals("ai_proxy_node_id", v)
	}
	if v := strings.TrimSpace(query.LlmDeploymentId); v != "" {
		q = q.Equals("llm_deployment_id", v)
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SAiRoutingManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.AiRoutingDetails {
	rows := make([]api.AiRoutingDetails, len(objs))
	sharableRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	routingIds := make([]string, len(objs))
	for i := range objs {
		rows[i].SharableVirtualResourceDetails = sharableRows[i]
		routing := objs[i].(*SAiRouting)
		routingIds[i] = routing.Id
		rows[i].LlmDeploymentId = routing.LlmDeploymentId
	}
	if fields == nil || fields.Contains("routing_models") {
		for i, rid := range routingIds {
			if rid == "" {
				continue
			}
			entries, err := fetchAiRoutingModels(rid, false)
			if err != nil {
				continue
			}
			rows[i].RoutingModels = make([]api.AiRoutingModelDetails, len(entries))
			for j := range entries {
				e := entries[j]
				rows[i].RoutingModels[j] = api.AiRoutingModelDetails{
					AiRoutingId:  e.AiRoutingId,
					AiProviderId: e.AiProviderId,
					AiModelId:    e.AiModelId,
					Priority:     e.Priority,
					ModelPattern: e.ModelPattern,
					LlmId:        e.LlmId,
					Enabled:      e.Enabled.IsTrue(),
				}
			}
		}
	}
	return rows
}

func (routing *SAiRouting) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformEnableInput) (jsonutils.JSONObject, error) {
	if err := db.EnabledPerformEnable(routing, ctx, userCred, true); err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (routing *SAiRouting) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformDisableInput) (jsonutils.JSONObject, error) {
	if err := db.EnabledPerformEnable(routing, ctx, userCred, false); err != nil {
		return nil, errors.Wrap(err, "EnabledPerformEnable")
	}
	return nil, nil
}

func (routing *SAiRouting) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input *api.AiRoutingUpdateInput,
) (*api.AiRoutingUpdateInput, error) {
	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = routing.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	if input.AiProxyNodeId != "" {
		input.AiProxyNodeId, err = validateAiProxyNodeId(ctx, userCred, input.AiProxyNodeId)
		if err != nil {
			return input, err
		}
	} else if query.Contains("ai_proxy_node_id") {
		input.AiProxyNodeId = ""
	}
	if query.Contains("llm_deployment_id") {
		input.LlmDeploymentId = strings.TrimSpace(input.LlmDeploymentId)
		if err := ensureUniqueAiRoutingLlmDeploymentId(ctx, input.LlmDeploymentId, routing.Id); err != nil {
			return input, err
		}
	}
	return input, nil
}

func (manager *SAiRoutingManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.AiRoutingCreateInput,
) (api.AiRoutingCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ValidateCreateData")
	}

	validatedModels, err := validateAiRoutingModelItems(ctx, userCred, input.Models)
	if err != nil {
		return input, err
	}
	input.Models = validatedModels

	input.AiProxyNodeId, err = validateAiProxyNodeId(ctx, userCred, input.AiProxyNodeId)
	if err != nil {
		return input, err
	}

	input.LlmDeploymentId = strings.TrimSpace(input.LlmDeploymentId)
	if err := ensureUniqueAiRoutingLlmDeploymentId(ctx, input.LlmDeploymentId, ""); err != nil {
		return input, err
	}

	if input.Enabled == nil && input.Disabled == nil {
		input.SetEnabled()
	}
	return input, nil
}

func (routing *SAiRouting) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	routing.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	input := api.AiRoutingCreateInput{}
	if err := data.Unmarshal(&input); err != nil {
		log.Errorf("ai_routing PostCreate unmarshal models: %v", err)
		return
	}
	if len(input.Models) == 0 {
		return
	}
	if err := createAiRoutingModels(ctx, userCred, ownerId, routing, input.Models); err != nil {
		log.Errorf("ai_routing %s create routing_models: %v", routing.Id, err)
	}
}

// PerformSetModels replaces all ai_routing_models bound to this routing.
func (routing *SAiRouting) PerformSetModels(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.AiRoutingSetModelsInput,
) (jsonutils.JSONObject, error) {
	items, err := validateAiRoutingModelItems(ctx, userCred, input.Models)
	if err != nil {
		return nil, err
	}
	if err := deleteAiRoutingModels(ctx, routing.Id); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	if err := createAiRoutingModels(ctx, userCred, routing.GetOwnerId(), routing, items); err != nil {
		return nil, err
	}
	return nil, nil
}
