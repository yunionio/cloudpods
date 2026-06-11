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
	"sort"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
)

// ModelsListEntry is one OpenAI-compatible model object in GET /openai/v1/models.
type ModelsListEntry struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ListModelsForVirtualKey returns OpenAI-compatible model ids reachable by the virtual key
// on the current aiproxy node (project ai_routing -> ai_routing_model -> ai_model).
func ListModelsForVirtualKey(ctx context.Context, userCred mcclient.TokenCredential, virtualKey string) ([]ModelsListEntry, error) {
	vk, err := loadEnabledVirtualKey(virtualKey)
	if err != nil {
		return nil, err
	}
	rpm := 0
	if vk.Limits != nil {
		rpm = vk.Limits.RequestsPerMinute
	}
	if err := TakeVirtualKeyRequestsPerMinute(vk.Id, rpm); err != nil {
		return nil, err
	}
	routings, err := listProjectRoutingsForVirtualKey(ctx, userCred, vk)
	if err != nil {
		return nil, err
	}
	currentNode := CurrentProxyNodeId()
	routingById := make(map[string]*SAiRouting, len(routings))
	routingIds := make([]string, 0, len(routings))
	for i := range routings {
		routingById[routings[i].Id] = &routings[i]
		if proxyNodeScopeMatches(routings[i].AiProxyNodeId, currentNode) {
			routingIds = append(routingIds, routings[i].Id)
		}
	}
	if len(routingIds) == 0 {
		return nil, nil
	}

	entries := make([]SAiRoutingModel, 0, 16)
	q := AiRoutingModelManager.Query().In("ai_routing_id", routingIds).Equals("enabled", true)
	if err := q.All(&entries); err != nil {
		return nil, errors.Wrap(err, "list ai_routing_models")
	}

	providerIds := make([]string, 0, len(entries))
	modelIds := make([]string, 0, len(entries))
	for i := range entries {
		providerIds = append(providerIds, entries[i].AiProviderId)
		modelIds = append(modelIds, entries[i].AiModelId)
	}

	providers, err := fetchEnabledAiProvidersByIds(providerIds)
	if err != nil {
		return nil, err
	}
	modelsById, err := fetchEnabledAiModelsByIds(modelIds)
	if err != nil {
		return nil, err
	}

	firstProviderByRouting := make(map[string]string, len(routingIds))
	for i := range entries {
		routingId := entries[i].AiRoutingId
		if _, ok := firstProviderByRouting[routingId]; ok {
			continue
		}
		if prov := providers[entries[i].AiProviderId]; prov != nil {
			firstProviderByRouting[routingId] = strings.TrimSpace(prov.ProviderKey)
		}
	}

	seen := make(map[string]ModelsListEntry, len(entries)+len(routingIds))
	created := time.Now().Unix()

	// Pass 1: ai_routing.model_key is itself a client-facing model id.
	for i := range routings {
		routing := &routings[i]
		if !proxyNodeScopeMatches(routing.AiProxyNodeId, currentNode) {
			continue
		}
		id := strings.TrimSpace(routing.ModelKey)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = ModelsListEntry{
			ID:      id,
			Object:  "model",
			Created: created,
			OwnedBy: firstProviderByRouting[routing.Id],
		}
	}

	// Pass 2: routes without model_key use ai_routing_model / pattern fallbacks.
	for i := range entries {
		e := &entries[i]
		routing := routingById[e.AiRoutingId]
		if routing == nil || strings.TrimSpace(routing.ModelKey) != "" {
			continue
		}
		prov := providers[e.AiProviderId]
		mdl := modelsById[e.AiModelId]
		if prov == nil || mdl == nil {
			continue
		}
		if !virtualKeyAllowsProvider(vk, prov) {
			continue
		}
		id := clientFacingModelID(routing, e, mdl)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = ModelsListEntry{
			ID:      id,
			Object:  "model",
			Created: created,
			OwnedBy: strings.TrimSpace(prov.ProviderKey),
		}
	}
	if len(seen) == 0 {
		return nil, nil
	}
	out := make([]ModelsListEntry, 0, len(seen))
	for _, item := range seen {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func clientFacingModelID(routing *SAiRouting, entry *SAiRoutingModel, mdl *SAiModel) string {
	if entry != nil {
		if mp := strings.TrimSpace(entry.ModelPattern); mp != "" && !strings.Contains(mp, "*") {
			return mp
		}
	}
	if routing != nil {
		if mp := strings.TrimSpace(routing.ModelPattern); mp != "" && !strings.Contains(mp, "*") {
			return mp
		}
	}
	if mdl != nil {
		return strings.TrimSpace(mdl.ModelKey)
	}
	return ""
}

func fetchEnabledAiProvidersByIds(ids []string) (map[string]*SAiProvider, error) {
	ids = uniqueNonEmptyStrings(ids)
	if len(ids) == 0 {
		return map[string]*SAiProvider{}, nil
	}
	rows := make([]SAiProvider, 0, len(ids))
	q := AiProviderManager.Query().In("id", ids).Equals("enabled", true)
	if err := q.All(&rows); err != nil {
		return nil, errors.Wrap(err, "list ai_providers")
	}
	out := make(map[string]*SAiProvider, len(rows))
	for i := range rows {
		out[rows[i].Id] = &rows[i]
	}
	return out, nil
}

func fetchEnabledAiModelsByIds(ids []string) (map[string]*SAiModel, error) {
	ids = uniqueNonEmptyStrings(ids)
	if len(ids) == 0 {
		return map[string]*SAiModel{}, nil
	}
	rows := make([]SAiModel, 0, len(ids))
	q := AiModelManager.Query().In("id", ids).Equals("enabled", true)
	if err := q.All(&rows); err != nil {
		return nil, errors.Wrap(err, "list ai_models")
	}
	out := make(map[string]*SAiModel, len(rows))
	for i := range rows {
		out[rows[i].Id] = &rows[i]
	}
	return out, nil
}

func uniqueNonEmptyStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
