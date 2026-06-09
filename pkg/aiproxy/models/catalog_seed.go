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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

// standardCatalogProviderKeys lists built-in provider_key values seeded at InitDB.
var standardCatalogProviderKeys = []string{
	"anthropic",
	"azure",
	"bedrock",
	"cerebras",
	"cohere",
	"gemini",
	"groq",
	"mistral",
	"ollama",
	"openai",
	"parasail",
	"perplexity",
	"sgl",
	"vertex",
	"openrouter",
	"elevenlabs",
	"huggingface",
	"nebius",
	"xai",
	"replicate",
	"vllm",
	"runway",
	"fireworks",
	"aliyun",
	"baidu",
	"xiaomi",
}

// defaultPublicBaseURL returns a well-known public API base for OpenAI-compatible upstreams.
// Empty string means no default in catalog (user must set base_url in provider config).
func defaultPublicBaseURL(providerKey string) string {
	switch strings.ToLower(strings.TrimSpace(providerKey)) {
	case "openai":
		return "https://api.openai.com"
	case "anthropic":
		return "https://api.anthropic.com"
	case "azure", "bedrock", "sgl", "vertex":
		return ""
	case "cerebras":
		return "https://api.cerebras.ai"
	case "cohere":
		return "https://api.cohere.ai"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta"
	case "groq":
		return "https://api.groq.com/openai"
	case "mistral":
		return "https://api.mistral.ai"
	case "ollama":
		return "http://127.0.0.1:11434"
	case "vllm":
		return "http://127.0.0.1:8000"
	case "parasail":
		return "https://api.parasail.io"
	case "perplexity":
		return "https://api.perplexity.ai"
	case "openrouter":
		return "https://openrouter.ai/api"
	case "elevenlabs":
		return "https://api.elevenlabs.io"
	case "huggingface":
		return "https://router.huggingface.co"
	case "nebius":
		return "https://api.tokenfactory.nebius.com"
	case "xai":
		return "https://api.x.ai"
	case "replicate":
		return "https://api.replicate.com"
	case "runway":
		return "https://api.dev.runwayml.com"
	case "fireworks":
		return "https://api.fireworks.ai/inference"
	case "aliyun":
		return "https://dashscope.aliyuncs.com/compatible-mode"
	case "baidu":
		return "https://qianfan.baidubce.com/v2"
	case "xiaomi":
		return "https://api.xiaomimimo.com"
	default:
		return ""
	}
}

func standardProviderConfig(providerKey string) *api.SAiProviderConfig {
	if u := defaultPublicBaseURL(providerKey); u != "" {
		return &api.SAiProviderConfig{BaseURL: u}
	}
	return nil
}

const placeholderCatalogModelKey = "default"

func catalogProviderId(providerKey string) string {
	return providerKey
}

func catalogProviderExists(providerId string) (bool, error) {
	cnt, err := AiProviderManager.Query().Equals("id", providerId).CountWithError()
	if err != nil {
		return false, errors.Wrap(err, "count catalog ai_provider")
	}
	return cnt > 0, nil
}

func catalogModelExists(modelId string) (bool, error) {
	cnt, err := AiModelManager.Query().Equals("id", modelId).CountWithError()
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
