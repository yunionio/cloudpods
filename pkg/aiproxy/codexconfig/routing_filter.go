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

package codexconfig

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	apmodels "yunion.io/x/onecloud/pkg/aiproxy/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
)

func filterModelEntriesForRouting(session *mcclient.ClientSession, routingNameOrID string, entries []ModelListEntry) ([]ModelListEntry, error) {
	routingNameOrID = strings.TrimSpace(routingNameOrID)
	if routingNameOrID == "" {
		return entries, nil
	}
	allowed, err := allowedClientModelIDsForRouting(session, routingNameOrID)
	if err != nil {
		return nil, err
	}
	return intersectModelEntries(entries, allowed), nil
}

func allowedClientModelIDsForRouting(session *mcclient.ClientSession, routingNameOrID string) (map[string]struct{}, error) {
	routingObj, err := apmodules.AiRoutings.Get(session, routingNameOrID, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "ai-routing-show %s", routingNameOrID)
	}
	routingID, _ := routingObj.GetString("id")
	routing := &apmodels.SAiRouting{
		ModelKey:     strings.TrimSpace(mustString(routingObj, "model_key")),
		ModelPattern: strings.TrimSpace(mustString(routingObj, "model_pattern")),
	}

	bindings, providerIDs, err := fetchRoutingBindings(session, routingID)
	if err != nil {
		return nil, err
	}
	modelsById, err := fetchCatalogModelsForRouting(session, routingID)
	if err != nil {
		return nil, err
	}
	providers, err := fetchProvidersByIDs(session, providerIDs)
	if err != nil {
		return nil, err
	}

	ids := apmodels.ClientFacingModelIDsForRouting(routing, bindings, modelsById, providers)
	allowed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	return allowed, nil
}

func fetchRoutingBindings(session *mcclient.ClientSession, routingID string) ([]apmodels.SAiRoutingModel, []string, error) {
	params := jsonutils.NewDict()
	params.Set("ai_routing_id", jsonutils.NewString(routingID))
	params.Set("enabled", jsonutils.JSONTrue)
	params.Set("limit", jsonutils.NewInt(500))
	result, err := apmodules.AiRoutingModels.List(session, params)
	if err != nil {
		return nil, nil, errors.Wrap(err, "ai-routing-model-list")
	}
	bindings := make([]apmodels.SAiRoutingModel, 0, len(result.Data))
	providerIDs := make([]string, 0, len(result.Data))
	for _, item := range result.Data {
		pid := strings.TrimSpace(mustString(item, "ai_provider_id"))
		mid := strings.TrimSpace(mustString(item, "ai_model_id"))
		if pid == "" || mid == "" {
			continue
		}
		bindings = append(bindings, apmodels.SAiRoutingModel{
			AiProviderId: pid,
			AiModelId:    mid,
			ModelPattern: strings.TrimSpace(mustString(item, "model_pattern")),
		})
		providerIDs = append(providerIDs, pid)
	}
	return bindings, uniqueStrings(providerIDs), nil
}

func fetchCatalogModelsForRouting(session *mcclient.ClientSession, routingID string) (map[string]*apmodels.SAiModel, error) {
	params := jsonutils.NewDict()
	params.Set("ai_routing_id", jsonutils.NewString(routingID))
	params.Set("enabled", jsonutils.JSONTrue)
	params.Set("limit", jsonutils.NewInt(500))
	result, err := apmodules.AiModels.List(session, params)
	if err != nil {
		return nil, errors.Wrap(err, "ai-model-list")
	}
	out := make(map[string]*apmodels.SAiModel, len(result.Data))
	for _, item := range result.Data {
		id := strings.TrimSpace(mustString(item, "id"))
		if id == "" {
			continue
		}
		out[id] = &apmodels.SAiModel{
			AiProviderId: strings.TrimSpace(mustString(item, "ai_provider_id")),
			ModelKey:     strings.TrimSpace(mustString(item, "model_key")),
		}
	}
	return out, nil
}

func fetchProvidersByIDs(session *mcclient.ClientSession, ids []string) (map[string]*apmodels.SAiProvider, error) {
	out := make(map[string]*apmodels.SAiProvider, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := out[id]; ok {
			continue
		}
		obj, err := apmodules.AiProviders.Get(session, id, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "ai-provider-show %s", id)
		}
		out[id] = &apmodels.SAiProvider{
			ProviderKey: strings.TrimSpace(mustString(obj, "provider_key")),
		}
	}
	return out, nil
}

func intersectModelEntries(entries []ModelListEntry, allowed map[string]struct{}) []ModelListEntry {
	if len(allowed) == 0 {
		return nil
	}
	out := make([]ModelListEntry, 0, len(entries))
	for _, entry := range entries {
		if _, ok := allowed[entry.ID]; ok {
			out = append(out, entry)
		}
	}
	return out
}

func mustString(obj jsonutils.JSONObject, key string) string {
	v, _ := obj.GetString(key)
	return v
}

func uniqueStrings(in []string) []string {
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
