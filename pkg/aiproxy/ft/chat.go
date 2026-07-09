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

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func RunChatTest(session *mcclient.ClientSession, opts *ChatOptions) error {
	tracker := NewResourceTracker(envKeepResources(opts.KeepResources))
	defer tracker.Cleanup(session)

	nonInteractive := envNonInteractive(opts.NonInteractive)

	providers, err := ListCatalogProviderKeys(session)
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		return errors.Error("无 ai_provider 资源；请先创建供应商（如 climc ai-provider-create）")
	}

	providerKey, err := promptSelectProvider(providers, opts.Provider, nonInteractive)
	if err != nil {
		return err
	}

	models, err := ListCatalogModelKeys(session, providerKey)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return errors.Errorf("provider %s 下无可用 model_key；创建 ai_model 或由 ai-test 自动创建", providerKey)
	}

	modelKey, err := promptSelectModel(models, providerKey, opts.Model, nonInteractive)
	if err != nil {
		return err
	}

	apiSecret, err := promptApiKey(providerKey, opts.ApiKey, nonInteractive)
	if err != nil {
		return err
	}

	prompt := strings.TrimSpace(opts.Prompt)
	if prompt == "" {
		prompt = resolvePromptFromEnv()
	}
	if prompt == "" {
		prompt = DefaultPromptForProvider(providerKey)
	}

	runStream := promptRunStream(opts.SkipStream, nonInteractive)

	names := DefaultAdminNames(providerKey, "")
	if opts.KeyName != "" {
		names.KeyName = opts.KeyName
	}
	if opts.VkName != "" {
		names.VkName = opts.VkName
	}
	if opts.RoutingName != "" {
		names.RoutingName = opts.RoutingName
	}

	catalogModelID := CatalogModelID(providerKey, modelKey)
	fmt.Println()
	fmt.Println("=== aiproxy OpenAI chat 测试 ===")
	fmt.Printf("provider: %s  model: %s  catalog_id: %s\n", providerKey, modelKey, catalogModelID)
	fmt.Printf("ai_key: %s  virtual_key: %s  routing: %s\n", names.KeyName, names.VkName, names.RoutingName)
	fmt.Println()

	Step("1. Keystone aiproxy public endpoint")
	aiproxyURL, err := ResolveAiproxyURL(session, opts.AiproxyURL)
	if err != nil {
		return err
	}
	fmt.Printf("AIPROXY_URL=%s\n", aiproxyURL)

	Step(fmt.Sprintf("2. Catalog %s / %s", providerKey, modelKey))
	if err := VerifyCatalog(session, providerKey, modelKey, false); err != nil {
		return err
	}

	Step("3. ai_key")
	vk, _, err := SetupAdminResources(session, tracker, providerKey, modelKey, apiSecret, names)
	if err != nil {
		return err
	}
	fmt.Printf("virtual_key=%s...\n", previewText(vk, 12))

	Step("4. POST /ai/openai/v1/chat/completions")
	client := httpClientFromSession(session)
	payload := map[string]interface{}{
		"model": modelKey,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 128,
	}
	code, body, err := postJSON(client, openAIChatURL(aiproxyURL), vk, payload)
	if err != nil {
		return err
	}
	fmt.Printf("HTTP %d\n", code)
	if err := printJSONBody(body); err != nil {
		return err
	}
	if code != 200 {
		return errors.Errorf("chat request failed with HTTP %d", code)
	}
	content, err := extractOpenAIChatContent(body)
	if err != nil {
		return err
	}
	fmt.Printf("content (%d chars): %s\n", len(content), previewText(content, 120))

	if runStream {
		Step("5. POST /ai/openai/v1/chat/completions (stream=true)")
		streamPayload := map[string]interface{}{
			"model":  modelKey,
			"stream": true,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"max_tokens": 64,
		}
		streamCode, streamBody, err := postJSONStream(client, openAIChatURL(aiproxyURL), vk, streamPayload)
		if err != nil {
			return err
		}
		defer streamBody.Close()
		fmt.Printf("HTTP %d (stream)\n", streamCode)
		aggregated, err := aggregateSSEStream(streamBody, parseOpenAIStreamDelta)
		if err != nil {
			return err
		}
		fmt.Printf("stream content (%d chars): %s\n", len(aggregated), previewText(aggregated, 120))
	}

	streamNote := ""
	if runStream {
		streamNote = " + stream"
	}
	fmt.Println()
	fmt.Printf("OK: aiproxy chat test passed for %s/%s (non-stream%s).\n", providerKey, modelKey, streamNote)
	return nil
}
