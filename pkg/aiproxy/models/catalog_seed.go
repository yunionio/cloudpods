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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// standardCatalogProviderKeys lists built-in provider_key values seeded at InitDB.
var standardCatalogProviderKeys = api.StandardCatalogProviderKeys

func standardProviderConfig(providerKey string) *api.SAiProviderConfig {
	if u := api.DefaultPublicBaseURL(providerKey); u != "" {
		return &api.SAiProviderConfig{BaseURL: u}
	}
	return nil
}

const placeholderCatalogModelKey = "default"

func catalogProviderId(providerKey string) string {
	return providerKey
}

func catalogProviderExists(providerId string) (bool, error) {
	cnt, err := AiProviderManager.RawQuery().Equals("id", providerId).CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "count catalog ai_provider")
	}
	return cnt > 0, nil
}

func catalogModelExists(modelId string) (bool, error) {
	cnt, err := AiModelManager.RawQuery().Equals("id", modelId).CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "count catalog ai_model")
	}
	return cnt > 0, nil
}

func insertCatalogProvider(ctx context.Context, providerKey, description string, cfg *api.SAiProviderConfig) error {
	providerId := catalogProviderId(providerKey)
	exists, err := catalogProviderExists(providerId)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	prov := SAiProvider{}
	prov.SetModelManager(AiProviderManager, &prov)
	prov.Id = providerId
	prov.Name = providerKey
	prov.ProviderKey = providerKey
	prov.Description = description
	prov.Config = cfg
	prov.SetEnabled(true)
	prov.Status = apis.STATUS_AVAILABLE
	prov.Progress = 100
	if err := AiProviderManager.TableSpec().Insert(ctx, &prov); err != nil {
		return errors.Wrapf(err, "insert ai_provider %s", providerKey)
	}
	return nil
}

func insertCatalogModel(ctx context.Context, providerId, providerKey, modelKey, description string) error {
	modelId := catalogModelId(providerKey, modelKey)
	exists, err := catalogModelExists(modelId)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	m := SAiModel{}
	m.SetModelManager(AiModelManager, &m)
	m.Id = modelId
	m.Name = modelId
	m.AiProviderId = providerId
	m.ModelKey = modelKey
	m.Description = description
	m.SetEnabled(true)
	m.Status = apis.STATUS_AVAILABLE
	m.Progress = 100
	if err := AiModelManager.TableSpec().Insert(ctx, &m); err != nil {
		return errors.Wrapf(err, "insert ai_model %s/%s", providerKey, modelKey)
	}
	return nil
}

func ensureSeedModelsEntries(ctx context.Context, providerId, providerKey string, entries []catalogSeedModel) error {
	if len(entries) == 0 {
		return insertCatalogModel(ctx, providerId, providerKey, placeholderCatalogModelKey,
			"Catalog seed placeholder; replace with concrete model_key values or use a provider with a built-in catalog.")
	}
	for i := range entries {
		if err := insertCatalogModel(ctx, providerId, providerKey, entries[i].ModelKey, entries[i].Description); err != nil {
			return err
		}
	}
	return nil
}

func ensureSeedProvider(ctx context.Context, providerKey string) error {
	providerKey = strings.TrimSpace(providerKey)
	providerId := catalogProviderId(providerKey)
	if err := insertCatalogProvider(ctx, providerKey,
		fmt.Sprintf("Standard provider catalog entry: %s", providerKey),
		standardProviderConfig(providerKey)); err != nil {
		return err
	}
	return ensureSeedModelsEntries(ctx, providerId, providerKey, catalogSeedModelsForProvider(providerKey))
}

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

// SeedStandardCatalog inserts built-in ai_provider / ai_model catalog rows on first init only.
// Existing rows are left unchanged so user config survives service restarts.
func SeedStandardCatalog(ctx context.Context) error {
	for _, pk := range standardCatalogProviderKeys {
		if err := ensureSeedProvider(ctx, pk); err != nil {
			return err
		}
	}
	log.Infof("aiproxy: standard catalog seed completed (%d providers)", len(standardCatalogProviderKeys))
	return nil
}
