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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const placeholderCatalogModelKey = "default"

func providerModelExists(providerId, modelKey string) (bool, error) {
	cnt, err := AiModelManager.Query().
		Equals("ai_provider_id", providerId).
		Equals("model_key", modelKey).
		CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "count ai_model for provider")
	}
	return cnt > 0, nil
}

func createUserProviderModel(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	prov *SAiProvider,
	modelKey, description string,
	contextWindow int,
	config *api.SAiModelConfig,
) error {
	if prov == nil || strings.TrimSpace(prov.Id) == "" {
		return errors.Error("ai_provider is nil or has no id")
	}
	modelKey = strings.TrimSpace(modelKey)
	if modelKey == "" {
		return errors.Error("model_key is empty")
	}
	exists, err := providerModelExists(prov.Id, modelKey)
	if err != nil {
		return err
	}
	if !exists {
		dataDict := jsonutils.NewDict()
		dataDict.Set("ai_provider_id", jsonutils.NewString(prov.Id))
		dataDict.Set("model_key", jsonutils.NewString(modelKey))
		dataDict.Set("enabled", jsonutils.JSONTrue)
		dataDict.Set("generate_name", jsonutils.NewString(defaultAiModelName(prov.Name, modelKey)))
		if desc := strings.TrimSpace(description); desc != "" {
			dataDict.Set("description", jsonutils.NewString(desc))
		}
		if contextWindow <= 0 {
			contextWindow = CatalogContextWindow(modelKey)
		}
		if contextWindow > 0 {
			dataDict.Set("context_window", jsonutils.NewInt(int64(contextWindow)))
		}
		if config != nil {
			mods := config.NormalizedInputModalities()
			if len(mods) > 0 {
				cfg := &api.SAiModelConfig{InputModalities: mods}
				dataDict.Set("config", jsonutils.Marshal(cfg))
			}
		}
		if _, err := db.DoCreate(AiModelManager, ctx, userCred, nil, dataDict, ownerId); err != nil {
			return errors.Wrapf(err, "create ai_model %q for provider %s", modelKey, prov.Id)
		}
	}
	live, err := fetchLiveAiModelByProviderAndKey(prov.Id, modelKey)
	if err != nil {
		return err
	}
	if live == nil {
		return errors.Wrapf(httperrors.ErrNotFound, "ai_model %q for provider %s", modelKey, prov.Id)
	}
	// Re-bind routings that still point at soft-deleted rows for this model_key.
	if err := remountAiRoutingModelsToModel(ctx, userCred, live); err != nil {
		return errors.Wrap(err, "remount ai_routing_models")
	}
	return nil
}

func createSelectedProviderModels(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	prov *SAiProvider,
	modelKeys []string,
) error {
	specs := make([]api.AiProviderSetModelSpec, 0, len(modelKeys))
	for _, modelKey := range modelKeys {
		specs = append(specs, api.AiProviderSetModelSpec{ModelKey: modelKey})
	}
	return createSelectedProviderModelSpecs(ctx, userCred, ownerId, prov, specs)
}

func createSelectedProviderModelSpecs(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	prov *SAiProvider,
	specs []api.AiProviderSetModelSpec,
) error {
	if prov == nil {
		return nil
	}
	for i := range specs {
		spec := specs[i]
		var cfg *api.SAiModelConfig
		if len(spec.InputModalities) > 0 {
			cfg = &api.SAiModelConfig{InputModalities: spec.InputModalities}
		}
		if err := createUserProviderModel(ctx, userCred, ownerId, prov, spec.ModelKey, "", spec.ContextWindow, cfg); err != nil {
			return err
		}
	}
	return nil
}

func normalizeProviderModelSpecs(input api.AiProviderSetModelsInput) ([]api.AiProviderSetModelSpec, error) {
	if len(input.Models) > 0 {
		seen := make(map[string]struct{}, len(input.Models))
		out := make([]api.AiProviderSetModelSpec, 0, len(input.Models))
		for i := range input.Models {
			mk, err := validateAiModelKey(input.Models[i].ModelKey)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[mk]; ok {
				continue
			}
			seen[mk] = struct{}{}
			spec := api.AiProviderSetModelSpec{
				ModelKey:      mk,
				ContextWindow: input.Models[i].ContextWindow,
			}
			if input.Models[i].ContextWindow < 0 {
				return nil, errors.Wrap(httperrors.ErrInputParameter, "context_window must be >= 0")
			}
			if mods := (&api.SAiModelConfig{InputModalities: input.Models[i].InputModalities}).NormalizedInputModalities(); len(mods) > 0 {
				spec.InputModalities = mods
			}
			out = append(out, spec)
		}
		return out, nil
	}
	keys, err := normalizeProviderModelKeys(input.ModelKeys)
	if err != nil {
		return nil, err
	}
	out := make([]api.AiProviderSetModelSpec, 0, len(keys))
	for _, mk := range keys {
		out = append(out, api.AiProviderSetModelSpec{ModelKey: mk})
	}
	return out, nil
}

// createCatalogModelsForUserProvider inserts built-in catalog model rows for a newly created public SaaS provider.
func createCatalogModelsForUserProvider(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	prov *SAiProvider,
) error {
	if prov == nil {
		return nil
	}
	pk := strings.TrimSpace(prov.ProviderKey)
	if !api.HasDefaultPublicBaseURL(pk) {
		return nil
	}
	entries := catalogSeedModelsForProvider(pk)
	if len(entries) == 0 {
		return createUserProviderModel(ctx, userCred, ownerId, prov, placeholderCatalogModelKey,
			"Catalog seed placeholder; replace with concrete model_key values or use a provider with a built-in catalog.", 0, nil)
	}
	for i := range entries {
		if err := createUserProviderModel(ctx, userCred, ownerId, prov, entries[i].ModelKey, entries[i].Description, entries[i].ContextWindow, nil); err != nil {
			return err
		}
	}
	return nil
}
