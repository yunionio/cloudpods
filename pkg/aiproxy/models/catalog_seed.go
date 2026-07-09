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
	if exists {
		return nil
	}
	dataDict := jsonutils.NewDict()
	dataDict.Set("ai_provider_id", jsonutils.NewString(prov.Id))
	dataDict.Set("model_key", jsonutils.NewString(modelKey))
	dataDict.Set("enabled", jsonutils.JSONTrue)
	dataDict.Set("generate_name", jsonutils.NewString(defaultAiModelName(prov.Name, modelKey)))
	if desc := strings.TrimSpace(description); desc != "" {
		dataDict.Set("description", jsonutils.NewString(desc))
	}
	if _, err := db.DoCreate(AiModelManager, ctx, userCred, nil, dataDict, ownerId); err != nil {
		return errors.Wrapf(err, "create ai_model %q for provider %s", modelKey, prov.Id)
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
	if prov == nil {
		return nil
	}
	for _, modelKey := range modelKeys {
		if err := createUserProviderModel(ctx, userCred, ownerId, prov, modelKey, ""); err != nil {
			return err
		}
	}
	return nil
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
			"Catalog seed placeholder; replace with concrete model_key values or use a provider with a built-in catalog.")
	}
	for i := range entries {
		if err := createUserProviderModel(ctx, userCred, ownerId, prov, entries[i].ModelKey, entries[i].Description); err != nil {
			return err
		}
	}
	return nil
}
