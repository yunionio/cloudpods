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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
)

type AdminNames struct {
	KeyName     string
	VkName      string
	RoutingName string
}

func DefaultAdminNames(providerKey string, suffix string) AdminNames {
	base := fmt.Sprintf("aiproxy-test-%s", providerKey)
	if suffix != "" {
		base = fmt.Sprintf("aiproxy-test-%s-%s", providerKey, suffix)
	}
	return AdminNames{
		KeyName:     base,
		VkName:      base + "-vk",
		RoutingName: base + "-routing",
	}
}

func DefaultAnthropicAdminNames(providerKey string) AdminNames {
	if providerKey == "anthropic" {
		return AdminNames{
			KeyName:     "aiproxy-test-anthropic",
			VkName:      "aiproxy-test-anthropic-vk",
			RoutingName: "aiproxy-test-anthropic-routing",
		}
	}
	return AdminNames{
		KeyName:     fmt.Sprintf("aiproxy-test-%s-anthropic", providerKey),
		VkName:      "aiproxy-test-anthropic-vk",
		RoutingName: "aiproxy-test-anthropic-routing",
	}
}

func ensureAiKey(session *mcclient.ClientSession, tracker *ResourceTracker, providerKey, keyName, apiSecret string) error {
	provider, err := apmodules.AiProviders.Get(session, providerKey, nil)
	if err != nil {
		return errors.Wrapf(err, "ai_provider %s not found", providerKey)
	}
	providerID, _ := provider.GetString("id")
	if providerID == "" {
		return errors.Errorf("ai_provider %s has empty id", providerKey)
	}

	if _, err := apmodules.AiKeys.Get(session, keyName, nil); err == nil {
		fmt.Printf("ai_key %s exists, syncing secret and ai_provider_id\n", keyName)
		params := jsonutils.NewDict()
		params.Set("ai_provider_id", jsonutils.NewString(providerID))
		params.Set("secret", jsonutils.NewString(apiSecret))
		params.Set("weight", jsonutils.NewInt(10))
		if _, err := apmodules.AiKeys.Update(session, keyName, params); err != nil {
			return errors.Wrap(err, "ai-key-update")
		}
	} else {
		params := jsonutils.NewDict()
		params.Set("name", jsonutils.NewString(keyName))
		params.Set("ai_provider_id", jsonutils.NewString(providerID))
		params.Set("secret", jsonutils.NewString(apiSecret))
		params.Set("weight", jsonutils.NewInt(10))
		params.Set("enabled", jsonutils.JSONTrue)
		if _, err := apmodules.AiKeys.Create(session, params); err != nil {
			return errors.Wrap(err, "ai-key-create")
		}
		if tracker != nil {
			tracker.createdAiKey = keyName
		}
	}
	return ensureAiKeyEnabled(session, keyName)
}

func ensureAiKeyEnabled(session *mcclient.ClientSession, keyName string) error {
	obj, err := apmodules.AiKeys.Get(session, keyName, nil)
	if err != nil {
		return err
	}
	enabled, _ := obj.Bool("enabled")
	if enabled {
		return nil
	}
	fmt.Printf("ai_key %s is disabled, enabling\n", keyName)
	_, err = apmodules.AiKeys.PerformAction(session, keyName, "enable", nil)
	return err
}

func VerifyAiKeyForProvider(session *mcclient.ClientSession, providerKey string) error {
	provider, err := apmodules.AiProviders.Get(session, providerKey, nil)
	if err != nil {
		return errors.Wrapf(err, "ai_provider %s not found", providerKey)
	}
	providerID, _ := provider.GetString("id")
	query := jsonutils.NewDict()
	query.Set("ai_provider_id", jsonutils.NewString(providerID))
	result, err := apmodules.AiKeys.List(session, query)
	if err != nil {
		return err
	}
	count := 0
	for _, item := range result.Data {
		ok, _ := item.Bool("enabled")
		if ok {
			count++
		}
	}
	if count == 0 {
		return errors.Errorf("no enabled ai_key bound to ai_provider_id=%s", providerID)
	}
	fmt.Printf("enabled ai_key rows for %s: %d\n", providerID, count)
	return nil
}

func ensureVirtualKey(session *mcclient.ClientSession, tracker *ResourceTracker, vkName string) (string, error) {
	if _, err := apmodules.AiVirtualKeys.Get(session, vkName, nil); err != nil {
		params := jsonutils.NewDict()
		params.Set("name", jsonutils.NewString(vkName))
		if _, err := apmodules.AiVirtualKeys.Create(session, params); err != nil {
			return "", errors.Wrap(err, "ai-virtual-key-create")
		}
		if tracker != nil {
			tracker.createdVirtualKey = vkName
		}
	} else {
		fmt.Printf("virtual key %s already exists\n", vkName)
	}
	obj, err := apmodules.AiVirtualKeys.Get(session, vkName, nil)
	if err != nil {
		return "", err
	}
	vk, _ := obj.GetString("virtual_key")
	if vk == "" {
		return "", errors.Error("empty virtual_key from ai-virtual-key-show")
	}
	return vk, nil
}

func ensureRouting(session *mcclient.ClientSession, tracker *ResourceTracker, routingName, providerKey, routingModelRef string) error {
	if _, err := apmodules.AiRoutings.Get(session, routingName, nil); err == nil {
		fmt.Printf("routing %s already exists\n", routingName)
		return nil
	}
	models := jsonutils.NewArray()
	models.Add(jsonutils.Marshal(map[string]interface{}{
		"ai_provider_id": providerKey,
		"ai_model_id":    routingModelRef,
		"priority":       1,
	}))
	params := jsonutils.NewDict()
	params.Set("name", jsonutils.NewString(routingName))
	params.Set("priority", jsonutils.NewInt(10))
	params.Set("models", models)
	if _, err := apmodules.AiRoutings.Create(session, params); err != nil {
		return errors.Wrap(err, "ai-routing-create")
	}
	if tracker != nil {
		tracker.createdRouting = routingName
	}
	return nil
}

func SetupAdminResources(session *mcclient.ClientSession, tracker *ResourceTracker, providerKey, modelKey, apiSecret string, names AdminNames) (virtualKey string, catalogModelID string, err error) {
	catalogModelID = CatalogModelID(providerKey, modelKey)
	routingModelRef, _, err := ensureAiModel(session, tracker, providerKey, modelKey)
	if err != nil {
		return "", "", err
	}
	if err = ensureAiKey(session, tracker, providerKey, names.KeyName, apiSecret); err != nil {
		return "", "", err
	}
	if err = VerifyAiKeyForProvider(session, providerKey); err != nil {
		return "", "", err
	}
	virtualKey, err = ensureVirtualKey(session, tracker, names.VkName)
	if err != nil {
		return "", "", err
	}
	if err = ensureRouting(session, tracker, names.RoutingName, providerKey, routingModelRef); err != nil {
		return "", "", err
	}
	return virtualKey, catalogModelID, nil
}

func EnsureAiProviderBaseURL(session *mcclient.ClientSession, tracker *ResourceTracker, providerKey, baseURL string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	obj, err := apmodules.AiProviders.Get(session, providerKey, nil)
	if err != nil {
		return errors.Wrapf(err, "ai_provider %s not found", providerKey)
	}
	current, _ := obj.GetString("config", "base_url")
	current = strings.TrimRight(strings.TrimSpace(current), "/")
	if current == baseURL {
		fmt.Printf("ai_provider %s config.base_url=%s\n", providerKey, baseURL)
		return nil
	}
	if tracker != nil && tracker.providerConfigRestore == nil {
		snap, err := snapshotProviderConfig(session, providerKey)
		if err != nil {
			return err
		}
		tracker.providerConfigRestore = snap
	}
	configDict := jsonutils.NewDict()
	if obj.Contains("config") {
		cfg, _ := obj.Get("config")
		if cfgDict, ok := cfg.(*jsonutils.JSONDict); ok {
			configDict = cfgDict
		}
	}
	configDict.Set("base_url", jsonutils.NewString(baseURL))
	params := jsonutils.NewDict()
	params.Set("config", configDict)
	fmt.Printf("updating ai_provider %s config.base_url -> %s\n", providerKey, baseURL)
	_, err = apmodules.AiProviders.Update(session, providerKey, params)
	return errors.Wrap(err, "ai-provider-update base_url")
}
