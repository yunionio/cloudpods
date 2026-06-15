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
	stderrors "errors"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// ChatUpstream holds resolved upstream and the model id to send.
type ChatUpstream struct {
	BaseURL       string
	APIKey        string
	UpstreamModel string
	ProviderKey   string
	AiProviderId  string
	AiKeyId       string

	// VirtualKeyId and usage/rate snapshots come from the matched ai_virtual_key row.
	VirtualKeyId        string
	MaxTokensPerRequest int
	RequestsPerMinute   int
}

func modelPatternMatches(pattern, requestedModel string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return true
	}
	rm := strings.TrimSpace(requestedModel)
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(rm, strings.TrimSuffix(pattern, "*"))
	}
	return strings.EqualFold(pattern, rm)
}

func virtualKeyAllowsProvider(vk *SAiVirtualKey, prov *SAiProvider) bool {
	if vk == nil || prov == nil {
		return false
	}
	if vk.Limits == nil || len(vk.Limits.AllowedAiProviderIds) == 0 {
		return true
	}
	for _, idOrName := range vk.Limits.AllowedAiProviderIds {
		idOrName = strings.TrimSpace(idOrName)
		if idOrName == "" {
			continue
		}
		if idOrName == prov.Id || strings.EqualFold(idOrName, prov.Name) {
			return true
		}
	}
	return false
}

func loadEnabledVirtualKey(virtualKey string) (*SAiVirtualKey, error) {
	virtualKey = strings.TrimSpace(virtualKey)
	if virtualKey == "" {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "missing virtual key (Authorization: Bearer <vk> or X-Ai-Virtual-Key)")
	}
	vk := SAiVirtualKey{}
	qvk := AiVirtualKeyManager.Query().Equals("virtual_key", virtualKey).Equals("enabled", true)
	err := qvk.First(&vk)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, errors.Wrap(httperrors.ErrInvalidStatus, "virtual key not found or disabled")
		}
		return nil, errors.Wrap(err, "query ai_virtual_key")
	}
	if strings.TrimSpace(vk.ProjectId) == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "virtual key has no project")
	}
	return &vk, nil
}

// listProjectRoutingsForVirtualKey returns enabled ai_routing rows owned by or shared with the virtual key's project.
func listProjectRoutingsForVirtualKey(ctx context.Context, userCred mcclient.TokenCredential, vk *SAiVirtualKey) ([]SAiRouting, error) {
	if vk == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "nil virtual key")
	}
	routings := make([]SAiRouting, 0, 16)
	qr := AiRoutingManager.Query().Equals("enabled", true)
	qr = AiRoutingManager.FilterByOwner(ctx, qr, AiRoutingManager, userCred, vk.GetOwnerId(), rbacscope.ScopeProject)
	qr = qr.Asc("priority")
	if err := qr.All(&routings); err != nil {
		return nil, errors.Wrap(err, "list ai_routings for virtual key project")
	}
	return routings, nil
}

// pickRoutingForRequest chooses the first matching ai_routing (lowest priority value wins)
// on the current aiproxy instance. When a matched rule is bound to another node, returns forbidden.
func pickRoutingForRequest(routings []SAiRouting, reqModel, currentNodeId string) (*SAiRouting, error) {
	var boundElsewhere *SAiRouting
	for i := range routings {
		r := &routings[i]
		if !modelPatternMatches(r.ModelPattern, reqModel) {
			continue
		}
		if !proxyNodeScopeMatches(r.AiProxyNodeId, currentNodeId) {
			if boundElsewhere == nil && strings.TrimSpace(r.AiProxyNodeId) != "" {
				boundElsewhere = r
			}
			continue
		}
		return r, nil
	}
	if boundElsewhere != nil {
		return nil, errors.Wrapf(httperrors.ErrForbidden,
			"ai_routing %q is bound to ai_proxy_node %q; use that instance endpoint",
			boundElsewhere.Name, boundElsewhere.AiProxyNodeId)
	}
	return nil, nil
}

type resolvedCatalogModel struct {
	provider *SAiProvider
	model    *SAiModel
}

// resolveCatalogModelFromRouting picks ai_routing_models for the routing and loads catalog provider/model rows.
func resolveCatalogModelFromRouting(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	vk *SAiVirtualKey,
	routing *SAiRouting,
	reqModel string,
	body *jsonutils.JSONDict,
) (*resolvedCatalogModel, error) {
	if routing == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "nil ai_routing")
	}
	providerId, modelId, err := pickAiRoutingModel(ctx, userCred, routing, reqModel, body)
	if err != nil {
		return nil, err
	}

	pObj, err := AiProviderManager.FetchByIdOrName(ctx, userCred, providerId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch ai_provider")
	}
	prov := pObj.(*SAiProvider)
	if !prov.GetEnabled() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_provider disabled")
	}
	if !virtualKeyAllowsProvider(vk, prov) {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_provider not allowed for this virtual key")
	}

	mObj, err := AiModelManager.FetchByIdOrName(ctx, userCred, modelId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch ai_model")
	}
	mdl := mObj.(*SAiModel)
	if !mdl.GetEnabled() {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_model disabled")
	}
	if strings.TrimSpace(mdl.AiProviderId) != "" && mdl.AiProviderId != prov.Id {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_model does not belong to resolved ai_provider")
	}
	return &resolvedCatalogModel{provider: prov, model: mdl}, nil
}

// ResolveChatUpstream resolves upstream URL, API key, and catalog model_key for a chat request:
//  1. ai_virtual_key (auth + project scope)
//  2. ai_routing in that project (model_pattern / optional proxy-node scope, priority)
//  3. ai_routing_model -> ai_provider + ai_model
//  4. ai_key rows for that provider matching the catalog model_key (weight), else provider.config api_key
func ResolveChatUpstream(ctx context.Context, userCred mcclient.TokenCredential, virtualKey string, body *jsonutils.JSONDict) (*ChatUpstream, error) {
	vk, err := loadEnabledVirtualKey(virtualKey)
	if err != nil {
		return nil, err
	}

	reqModel, _ := body.GetString("model")
	if strings.TrimSpace(reqModel) == "" {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "missing model in JSON body")
	}

	routings, err := listProjectRoutingsForVirtualKey(ctx, userCred, vk)
	if err != nil {
		return nil, err
	}
	routing, err := pickRoutingForRequest(routings, reqModel, CurrentProxyNodeId())
	if err != nil {
		return nil, err
	}
	if routing == nil {
		return nil, errors.Wrap(httperrors.ErrNotFound, "no ai_routing matched for virtual key project on this aiproxy node")
	}

	resolved, err := resolveCatalogModelFromRouting(ctx, userCred, vk, routing, reqModel, body)
	if err != nil {
		return nil, err
	}
	prov := resolved.provider
	mdl := resolved.model

	if prov.Config == nil {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_provider.config is empty")
	}
	baseURL := prov.Config.ResolvedBaseURL()
	if baseURL == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_provider.config must include base_url")
	}

	upstreamModel := strings.TrimSpace(mdl.ModelKey)
	if upstreamModel == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "ai_model.model_key is empty")
	}

	// Keys are scoped to ai_provider; routing on each ai_key matches the resolved catalog model_key.
	keyRes, err := resolveUpstreamAPIKey(prov, upstreamModel)
	if err != nil {
		return nil, err
	}
	if keyRes == nil || keyRes.Secret == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "no api_key for ai_provider and catalog model")
	}

	up := &ChatUpstream{
		BaseURL:       baseURL,
		APIKey:        keyRes.Secret,
		UpstreamModel: upstreamModel,
		ProviderKey:   prov.ProviderKey,
		AiProviderId:  prov.Id,
		AiKeyId:       keyRes.AiKeyId,
		VirtualKeyId:  vk.Id,
	}
	if vk.Limits != nil {
		up.MaxTokensPerRequest = vk.Limits.MaxTokensPerRequest
		up.RequestsPerMinute = vk.Limits.RequestsPerMinute
	}
	return up, nil
}
