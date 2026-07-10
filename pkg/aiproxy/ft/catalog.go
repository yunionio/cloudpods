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

package ft

import (
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/mcclient"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
)

func CatalogModelID(providerKey, modelKey string) string {
	return fmt.Sprintf("%s-%s", providerKey, modelKey)
}

func DefaultModelForProvider(providerKey string) string {
	switch providerKey {
	case api.ProviderKeyAliyun:
		return "qwen-turbo"
	case api.ProviderKeyXiaomi:
		return "mimo-v2-flash"
	case api.ProviderKeyMoonshot:
		return "kimi-k2.6"
	case api.ProviderKeyDeepseek:
		return "deepseek-v4-flash"
	case api.ProviderKeyOpenAI:
		return "gpt-4o-mini"
	case api.ProviderKeyAnthropic:
		return "claude-sonnet-4-5"
	default:
		return ""
	}
}

func DefaultPromptForProvider(providerKey string) string {
	switch providerKey {
	case api.ProviderKeyAliyun:
		return "用一句话介绍通义千问"
	case api.ProviderKeyXiaomi:
		return "用一句话介绍小米 MiMo"
	case api.ProviderKeyMoonshot:
		return "用一句话介绍 Kimi"
	default:
		return "用一句话介绍这个模型"
	}
}

func ListCatalogProviderKeys(session *mcclient.ClientSession) ([]string, error) {
	query := jsonutils.NewDict()
	query.Set("limit", jsonutils.NewInt(500))
	result, err := apmodules.AiProviders.List(session, query)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(result.Data))
	seen := map[string]struct{}{}
	for _, item := range result.Data {
		pk, _ := item.GetString("provider_key")
		pk = strings.TrimSpace(pk)
		if pk == "" {
			continue
		}
		if _, ok := seen[pk]; ok {
			continue
		}
		seen[pk] = struct{}{}
		keys = append(keys, pk)
	}
	sort.Strings(keys)
	return keys, nil
}

func ListCatalogModelKeys(session *mcclient.ClientSession, providerKey string) ([]string, error) {
	query := jsonutils.NewDict()
	query.Set("limit", jsonutils.NewInt(500))
	query.Set("ai_provider_id", jsonutils.NewString(providerKey))
	result, err := apmodules.AiModels.List(session, query)
	if err != nil {
		return nil, err
	}
	models := make([]string, 0, len(result.Data))
	seen := map[string]struct{}{}
	for _, item := range result.Data {
		mk, _ := item.GetString("model_key")
		mk = strings.TrimSpace(mk)
		if mk == "" || mk == "default" {
			continue
		}
		if _, ok := seen[mk]; ok {
			continue
		}
		seen[mk] = struct{}{}
		models = append(models, mk)
	}
	sort.Strings(models)
	return models, nil
}

func VerifyCatalog(session *mcclient.ClientSession, providerKey, modelKey string, warnMissingModel bool) error {
	if _, err := apmodules.AiProviders.Get(session, providerKey, nil); err != nil {
		return errors.Wrapf(err, "ai_provider %s missing; create it first (e.g. climc ai-provider-create)", providerKey)
	}
	if _, err := findAiModelByKey(session, providerKey, modelKey); err == nil {
		return nil
	}
	catalogID := CatalogModelID(providerKey, modelKey)
	if _, err := apmodules.AiModels.Get(session, catalogID, nil); err == nil {
		return nil
	}
	if warnMissingModel {
		fmt.Printf("WARN: ai_model %s not in catalog; will create for test if needed\n", catalogID)
		return nil
	}
	return errors.Errorf("ai_model %s not found; create ai_model or let ai-test-* create it for the test", catalogID)
}

func findAiModelByKey(session *mcclient.ClientSession, providerKey, modelKey string) (jsonutils.JSONObject, error) {
	query := jsonutils.NewDict()
	query.Set("limit", jsonutils.NewInt(10))
	query.Set("ai_provider_id", jsonutils.NewString(providerKey))
	query.Set("model_key", jsonutils.NewString(modelKey))
	result, err := apmodules.AiModels.List(session, query)
	if err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, errors.Errorf("model_key %s not found under provider %s", modelKey, providerKey)
	}
	return result.Data[0], nil
}

// EnsureAiModel guarantees an ai_model row exists for providerKey/modelKey.
// Returns the model reference for ai_routing and the created resource name (if any).
func ensureAiModel(session *mcclient.ClientSession, tracker *ResourceTracker, providerKey, modelKey string) (routingModelRef string, createdName string, err error) {
	catalogID := CatalogModelID(providerKey, modelKey)
	if _, err := apmodules.AiModels.Get(session, catalogID, nil); err == nil {
		return modelKey, "", nil
	}
	if _, err := findAiModelByKey(session, providerKey, modelKey); err == nil {
		return modelKey, "", nil
	}
	fmt.Printf("ai_model %s not in catalog, creating for test\n", catalogID)
	params := jsonutils.NewDict()
	params.Set("name", jsonutils.NewString(catalogID))
	params.Set("ai_provider_id", jsonutils.NewString(providerKey))
	params.Set("model_key", jsonutils.NewString(modelKey))
	params.Set("enabled", jsonutils.JSONTrue)
	if _, err := apmodules.AiModels.Create(session, params); err != nil {
		return "", "", errors.Wrapf(err, "ai-model-create %s/%s", providerKey, modelKey)
	}
	if tracker != nil {
		tracker.createdAiModel = catalogID
	}
	return modelKey, catalogID, nil
}
