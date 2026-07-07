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

func RunAnthropicTest(session *mcclient.ClientSession, opts *AnthropicOptions) error {
	tracker := NewResourceTracker(envKeepResources(opts.KeepResources))
	defer tracker.Cleanup(session)

	providerKey := strings.TrimSpace(opts.Provider)
	if providerKey == "" {
		providerKey = "anthropic"
	}

	modelKey := strings.TrimSpace(opts.Model)
	if modelKey == "" {
		modelKey = resolveModelFromEnv()
	}
	if modelKey == "" {
		modelKey = DefaultModelForProvider(providerKey)
	}

	apiSecret := strings.TrimSpace(opts.ApiKey)
	if apiSecret == "" {
		var err error
		apiSecret, err = promptApiKey(providerKey, "", envNonInteractive(opts.NonInteractive))
		if err != nil {
			return err
		}
	}

	prompt := strings.TrimSpace(opts.Prompt)
	if prompt == "" {
		prompt = resolvePromptFromEnv()
	}
	if prompt == "" {
		prompt = "Say hi in one short sentence."
	}

	skipStream := envSkipStream(opts.SkipStream)
	names := DefaultAnthropicAdminNames(providerKey)
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
	fmt.Println("=== aiproxy Anthropic Messages 测试 ===")
	fmt.Printf("provider: %s  model: %s  catalog_id: %s\n", providerKey, modelKey, catalogModelID)
	fmt.Println()

	Step("1. Resolve aiproxy URL")
	aiproxyURL, err := ResolveAiproxyURL(session, opts.AiproxyURL)
	if err != nil {
		return err
	}
	fmt.Printf("AIPROXY_URL=%s\n", aiproxyURL)
	upstreamBase := strings.TrimSpace(opts.UpstreamBaseURL)
	if upstreamBase == "" {
		upstreamBase = envFirst("AIPROXY_TEST_BASE_URL", "AIPROXY_FT_BASE_URL")
	}
	if upstreamBase != "" {
		if err := EnsureAiProviderBaseURL(session, tracker, providerKey, upstreamBase); err != nil {
			return err
		}
	}

	Step(fmt.Sprintf("2. Catalog %s / %s", providerKey, modelKey))
	if err := VerifyCatalog(session, providerKey, modelKey, true); err != nil {
		return err
	}

	Step("3. ai_key / ai_virtual_key / ai_routing")
	vk, _, err := SetupAdminResources(session, tracker, providerKey, modelKey, apiSecret, names)
	if err != nil {
		return err
	}
	fmt.Printf("virtual_key=%s...\n", previewText(vk, 12))

	Step("4. POST /ai/anthropic/v1/messages")
	client := httpClientFromSession(session)
	payload := map[string]interface{}{
		"model":      modelKey,
		"max_tokens": 128,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	code, body, err := postJSON(client, anthropicMessagesURL(aiproxyURL), vk, payload)
	if err != nil {
		return err
	}
	fmt.Printf("HTTP %d\n", code)
	if err := printJSONBody(body); err != nil {
		return err
	}
	if code != 200 {
		return errors.Errorf("anthropic messages request failed with HTTP %d", code)
	}
	content, err := extractAnthropicTextContent(body)
	if err != nil {
		return err
	}
	fmt.Printf("text: %s\n", previewText(content, 120))

	if !skipStream {
		Step("5. POST /ai/anthropic/v1/messages (stream=true)")
		streamPayload := map[string]interface{}{
			"model":      modelKey,
			"stream":     true,
			"max_tokens": 128,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
		}
		streamCode, streamBody, err := postJSONStream(client, anthropicMessagesURL(aiproxyURL), vk, streamPayload)
		if err != nil {
			return err
		}
		defer streamBody.Close()
		fmt.Printf("HTTP %d (anthropic stream)\n", streamCode)
		aggregated, err := aggregateSSEStream(streamBody, parseAnthropicStreamDelta)
		if err != nil {
			return err
		}
		fmt.Printf("stream text: %s\n", previewText(aggregated, 120))
	}

	fmt.Println()
	fmt.Printf("OK: anthropic messages test passed for %s/%s.\n", providerKey, modelKey)
	return nil
}
