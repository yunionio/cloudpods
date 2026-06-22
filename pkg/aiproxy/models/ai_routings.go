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
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// SAiRouting stores a project-scoped (and optionally shared) routing rule.
type SAiRouting struct {
	db.SSharableVirtualResourceBase
	db.SEnabledResourceBase

	Priority int `default:"100" nullable:"false" list:"user" create:"optional" update:"user"`
	// ModelKey exactly matches the client request "model" (case-insensitive).
	// When non-empty and matched, takes precedence over ModelPattern during routing.
	ModelKey string `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	// ModelPattern optionally matches the requested model id (implementation-specific glob/prefix).
	ModelPattern string `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	// AiProxyNodeId optionally binds the rule to one aiproxy instance (ai_proxy_node id).
	AiProxyNodeId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`

	// Router fields optionally call an external model router before selecting an ai_routing_model.
	RouterEnabled        bool   `default:"false" nullable:"false" list:"user" create:"optional" update:"user"`
	RouterUrl            string `width:"512" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	RouterRoutePath      string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
	RouterTimeoutSeconds int    `default:"0" nullable:"false" list:"user" create:"optional" update:"user"`
	RouterFallbackPolicy string `width:"32" charset:"ascii" nullable:"true" list:"user" create:"optional" update:"user"`
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
	if v := strings.TrimSpace(query.ModelKey); v != "" {
		q = q.Equals("model_key", v)
	}
	if v := strings.TrimSpace(query.AiProxyNodeId); v != "" {
		q = q.Equals("ai_proxy_node_id", v)
	}
	if query.RouterEnabled != nil {
		q = q.Equals("router_enabled", *query.RouterEnabled)
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
		routing := objs[i].(*SAiRouting)
		rows[i].SharableVirtualResourceDetails = sharableRows[i]
		rows[i].Priority = routing.Priority
		rows[i].ModelKey = routing.ModelKey
		rows[i].ModelPattern = routing.ModelPattern
		rows[i].AiProxyNodeId = routing.AiProxyNodeId
		rows[i].RouterEnabled = routing.RouterEnabled
		rows[i].RouterUrl = routing.RouterUrl
		rows[i].RouterRoutePath = routing.RouterRoutePath
		rows[i].RouterTimeoutSeconds = routing.RouterTimeoutSeconds
		rows[i].RouterFallbackPolicy = routing.RouterFallbackPolicy
		rows[i].Enabled = routing.GetEnabled()
		routingIds[i] = routing.Id
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

func normalizeAiRoutingRouterFallbackPolicy(policy string) (string, error) {
	policy = strings.ToLower(strings.TrimSpace(policy))
	switch policy {
	case "":
		return api.AiRoutingRouterFallbackPriority, nil
	case api.AiRoutingRouterFallbackPriority, api.AiRoutingRouterFallbackFailClosed:
		return policy, nil
	default:
		return "", errors.Wrapf(httperrors.ErrInputParameter, "unsupported router_fallback_policy %q", policy)
	}
}

func normalizeAiRoutingRouterRoutePath(routePath string) string {
	routePath = strings.TrimSpace(routePath)
	if routePath == "" {
		return api.AiRoutingRouterDefaultRoutePath
	}
	if !strings.HasPrefix(routePath, "/") {
		return "/" + routePath
	}
	return routePath
}

func normalizeAiRoutingRouterTimeoutSeconds(timeout int) int {
	if timeout <= 0 {
		return api.AiRoutingRouterDefaultTimeoutSeconds
	}
	return timeout
}

func normalizeAiRoutingModelKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil
	}
	if strings.Contains(key, "*") {
		return "", errors.Wrap(httperrors.ErrInputParameter, "model_key must not contain '*'")
	}
	if len(key) > 256 {
		return "", errors.Wrap(httperrors.ErrInputParameter, "model_key too long")
	}
	return key, nil
}

func normalizeAiRoutingRouterCreate(input *api.AiRoutingCreateInput) error {
	input.RouterUrl = strings.TrimSpace(input.RouterUrl)
	input.RouterRoutePath = normalizeAiRoutingRouterRoutePath(input.RouterRoutePath)
	input.RouterTimeoutSeconds = normalizeAiRoutingRouterTimeoutSeconds(input.RouterTimeoutSeconds)
	policy, err := normalizeAiRoutingRouterFallbackPolicy(input.RouterFallbackPolicy)
	if err != nil {
		return err
	}
	input.RouterFallbackPolicy = policy
	if input.RouterEnabled && input.RouterUrl == "" {
		return errors.Wrap(httperrors.ErrInputParameter, "router_url is required when router_enabled is true")
	}
	return nil
}

func normalizeAiRoutingRouterUpdate(routing *SAiRouting, query jsonutils.JSONObject, input *api.AiRoutingUpdateInput) error {
	effectiveEnabled := routing.RouterEnabled
	if input.RouterEnabled != nil {
		effectiveEnabled = *input.RouterEnabled
	}
	effectiveUrl := strings.TrimSpace(routing.RouterUrl)
	if query.Contains("router_url") {
		input.RouterUrl = strings.TrimSpace(input.RouterUrl)
		effectiveUrl = input.RouterUrl
	}
	if query.Contains("router_route_path") {
		input.RouterRoutePath = normalizeAiRoutingRouterRoutePath(input.RouterRoutePath)
	}
	if query.Contains("router_timeout_seconds") {
		input.RouterTimeoutSeconds = normalizeAiRoutingRouterTimeoutSeconds(input.RouterTimeoutSeconds)
	}
	if query.Contains("router_fallback_policy") {
		policy, err := normalizeAiRoutingRouterFallbackPolicy(input.RouterFallbackPolicy)
		if err != nil {
			return err
		}
		input.RouterFallbackPolicy = policy
	}
	if effectiveEnabled && effectiveUrl == "" {
		return errors.Wrap(httperrors.ErrInputParameter, "router_url is required when router_enabled is true")
	}
	return nil
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
	if query.Contains("model_key") {
		var err error
		input.ModelKey, err = normalizeAiRoutingModelKey(input.ModelKey)
		if err != nil {
			return input, err
		}
	}
	if err := normalizeAiRoutingRouterUpdate(routing, query, input); err != nil {
		return input, err
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

	input.AiProxyNodeId, err = resolveAiProxyNodeIdForCreate(ctx, userCred, input.AiProxyNodeId)
	if err != nil {
		return input, err
	}
	input.ModelKey, err = normalizeAiRoutingModelKey(input.ModelKey)
	if err != nil {
		return input, err
	}
	if err := normalizeAiRoutingRouterCreate(&input); err != nil {
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
