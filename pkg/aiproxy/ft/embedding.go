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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// RunEmbeddingTest exercises POST /ai/openai/v1/embeddings end to end.
func RunEmbeddingTest(session *mcclient.ClientSession, opts *EmbeddingOptions) error {
	tracker := NewResourceTracker(envKeepResources(opts.KeepResources))
	defer tracker.Cleanup(session)

	nonInteractive := envNonInteractive(opts.NonInteractive)

	providerKey, modelKey, apiSecret, err := resolveProviderModelSecret(session, opts.Provider, opts.Model, opts.ApiKey, nonInteractive)
	if err != nil {
		return err
	}

	names := DefaultAdminNames(providerKey, "emb")
	if opts.KeyName != "" {
		names.KeyName = opts.KeyName
	}
	if opts.VkName != "" {
		names.VkName = opts.VkName
	}
	if opts.RoutingName != "" {
		names.RoutingName = opts.RoutingName
	}

	fmt.Println()
	fmt.Println("=== aiproxy OpenAI embeddings 测试 ===")
	fmt.Printf("provider: %s  model: %s\n", providerKey, modelKey)
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

	Step("3. ai_key / ai_virtual_key / ai_routing")
	vk, _, err := SetupAdminResources(session, tracker, providerKey, modelKey, apiSecret, names)
	if err != nil {
		return err
	}
	fmt.Printf("virtual_key=%s...\n", previewText(vk, 12))

	client := httpClientFromSession(session)
	url := openAIEndpointURL(aiproxyURL, "/embeddings")

	Step("4. POST /embeddings (single input)")
	code, body, err := postJSON(client, url, vk, map[string]interface{}{
		"model": modelKey,
		"input": "The food was delicious and the waiter was friendly.",
	})
	if err != nil {
		return err
	}
	fmt.Printf("HTTP %d\n", code)
	if err := printJSONBody(body); err != nil {
		return err
	}
	if code != 200 {
		return errors.Errorf("embeddings request failed with HTTP %d", code)
	}
	summary, err := parseEmbeddingResponse(body)
	if err != nil {
		return err
	}
	if summary.Vectors == 0 || summary.Dimension == 0 {
		return errors.Errorf("embeddings response has no float vector (vectors=%d dimension=%d)", summary.Vectors, summary.Dimension)
	}
	fmt.Printf("vectors=%d dimension=%d prompt_tokens=%d\n", summary.Vectors, summary.Dimension, summary.PromptTokens)

	if !opts.SkipBatch {
		Step("5. POST /embeddings (batch input + dimensions)")
		batchPayload := map[string]interface{}{
			"model": modelKey,
			"input": []string{"first", "second", "third"},
		}
		if opts.Dimensions > 0 {
			batchPayload["dimensions"] = opts.Dimensions
		}
		code, body, err = postJSON(client, url, vk, batchPayload)
		if err != nil {
			return err
		}
		fmt.Printf("HTTP %d\n", code)
		if code != 200 {
			return errors.Errorf("batch embeddings request failed with HTTP %d: %s", code, truncateBody(body))
		}
		summary, err = parseEmbeddingResponse(body)
		if err != nil {
			return err
		}
		if summary.Vectors < 2 {
			return errors.Errorf("batch embeddings returned %d vectors, want 3", summary.Vectors)
		}
		fmt.Printf("vectors=%d dimension=%d\n", summary.Vectors, summary.Dimension)
	}

	if !opts.SkipBase64 {
		Step("6. POST /embeddings (encoding_format=base64)")
		code, body, err = postJSON(client, url, vk, map[string]interface{}{
			"model":           modelKey,
			"input":           "base64 encoding check",
			"encoding_format": "base64",
		})
		if err != nil {
			return err
		}
		fmt.Printf("HTTP %d\n", code)
		if code != 200 {
			return errors.Errorf("base64 embeddings request failed with HTTP %d: %s", code, truncateBody(body))
		}
		if !strings.Contains(string(body), `"embedding"`) {
			return errors.Error("base64 embeddings response missing data[].embedding")
		}
	}

	Step("7. GET /models exposes embedding capability")
	if err := verifyModelsListing(client, aiproxyURL, vk, modelKey, api.ModelCategoryEmbedding); err != nil {
		fmt.Printf("WARN: %v\n", err)
	}

	fmt.Println()
	fmt.Printf("OK: aiproxy embeddings test passed for %s/%s.\n", providerKey, modelKey)
	return nil
}

// RunRerankTest exercises POST /ai/openai/v1/rerank end to end.
func RunRerankTest(session *mcclient.ClientSession, opts *RerankOptions) error {
	tracker := NewResourceTracker(envKeepResources(opts.KeepResources))
	defer tracker.Cleanup(session)

	nonInteractive := envNonInteractive(opts.NonInteractive)

	providerKey, modelKey, apiSecret, err := resolveProviderModelSecret(session, opts.Provider, opts.Model, opts.ApiKey, nonInteractive)
	if err != nil {
		return err
	}

	names := DefaultAdminNames(providerKey, "rerank")
	if opts.KeyName != "" {
		names.KeyName = opts.KeyName
	}
	if opts.VkName != "" {
		names.VkName = opts.VkName
	}
	if opts.RoutingName != "" {
		names.RoutingName = opts.RoutingName
	}

	fmt.Println()
	fmt.Println("=== aiproxy OpenAI rerank 测试 ===")
	fmt.Printf("provider: %s  model: %s\n", providerKey, modelKey)
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

	Step("3. ai_key / ai_virtual_key / ai_routing")
	vk, _, err := SetupAdminResources(session, tracker, providerKey, modelKey, apiSecret, names)
	if err != nil {
		return err
	}
	fmt.Printf("virtual_key=%s...\n", previewText(vk, 12))

	query := strings.TrimSpace(opts.Query)
	if query == "" {
		query = "What is the capital of China?"
	}
	payload := map[string]interface{}{
		"model":     modelKey,
		"query":     query,
		"documents": []string{"Beijing is the capital of China.", "The weather is nice today.", "Paris is in France."},
	}
	if opts.TopN > 0 {
		payload["top_n"] = opts.TopN
	}

	Step("4. POST /rerank")
	client := httpClientFromSession(session)
	code, body, err := postJSON(client, openAIEndpointURL(aiproxyURL, "/rerank"), vk, payload)
	if err != nil {
		return err
	}
	fmt.Printf("HTTP %d\n", code)
	if err := printJSONBody(body); err != nil {
		return err
	}
	if code != 200 {
		return errors.Errorf("rerank request failed with HTTP %d", code)
	}
	scores, err := parseRerankScores(body)
	if err != nil {
		return err
	}
	if len(scores) == 0 {
		return errors.Error("rerank response has no results")
	}
	for i := 1; i < len(scores); i++ {
		if scores[i] > scores[i-1]+1e-9 {
			return errors.Errorf("rerank results are not ordered by relevance score: %v", scores)
		}
	}
	fmt.Printf("results=%d top_score=%.4f\n", len(scores), scores[0])

	Step("5. GET /models exposes rerank capability")
	if err := verifyModelsListing(client, aiproxyURL, vk, modelKey, api.ModelCategoryReranker); err != nil {
		fmt.Printf("WARN: %v\n", err)
	}

	fmt.Println()
	fmt.Printf("OK: aiproxy rerank test passed for %s/%s.\n", providerKey, modelKey)
	return nil
}

// resolveProviderModelSecret prompts/validates provider, model and upstream key.
func resolveProviderModelSecret(
	session *mcclient.ClientSession,
	providerOpt, modelOpt, apiKeyOpt string,
	nonInteractive bool,
) (providerKey, modelKey, apiSecret string, err error) {
	providers, err := ListCatalogProviderKeys(session)
	if err != nil {
		return "", "", "", err
	}
	if len(providers) == 0 {
		return "", "", "", errors.Error("无 ai_provider 资源；请先创建供应商（如 climc ai-provider-create）")
	}
	providerKey, err = promptSelectProvider(providers, providerOpt, nonInteractive)
	if err != nil {
		return "", "", "", err
	}
	models, err := ListCatalogModelKeys(session, providerKey)
	if err != nil {
		return "", "", "", err
	}
	if len(models) == 0 {
		return "", "", "", errors.Errorf("provider %s 下无可用 model_key；创建 ai_model 或由 ai-test 自动创建", providerKey)
	}
	modelKey, err = promptSelectModel(models, providerKey, modelOpt, nonInteractive)
	if err != nil {
		return "", "", "", err
	}
	apiSecret, err = promptApiKey(providerKey, apiKeyOpt, nonInteractive)
	if err != nil {
		return "", "", "", err
	}
	return providerKey, modelKey, apiSecret, nil
}

func openAIEndpointURL(baseURL, endpoint string) string {
	return strings.TrimRight(baseURL, "/") + "/ai/openai/v1" + endpoint
}

// embeddingSummary holds the parts of an embeddings response the E2E test cares about.
type embeddingSummary struct {
	Vectors      int
	Dimension    int
	PromptTokens int
}

func parseEmbeddingResponse(body []byte) (embeddingSummary, error) {
	var resp struct {
		Data []struct {
			Embedding json.RawMessage `json:"embedding"`
		} `json:"data"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}
	var out embeddingSummary
	if err := json.Unmarshal(body, &resp); err != nil {
		return out, errors.Wrap(err, "parse embeddings response")
	}
	out.PromptTokens = resp.Usage.PromptTokens
	if out.PromptTokens == 0 {
		out.PromptTokens = resp.Usage.TotalTokens
	}
	out.Vectors = len(resp.Data)
	if out.Vectors == 0 {
		return out, nil
	}
	var vec []float64
	if err := json.Unmarshal(resp.Data[0].Embedding, &vec); err == nil {
		out.Dimension = len(vec)
	}
	return out, nil
}

func parseRerankScores(body []byte) ([]float64, error) {
	var resp struct {
		Results []struct {
			RelevanceScore float64 `json:"relevance_score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, errors.Wrap(err, "parse rerank response")
	}
	out := make([]float64, 0, len(resp.Results))
	for _, r := range resp.Results {
		out = append(out, r.RelevanceScore)
	}
	return out, nil
}

// verifyModelsListing checks GET /ai/openai/v1/models exposes the model with the
// expected category. Missing capability metadata is reported as a warning only.
func verifyModelsListing(
	client *http.Client,
	aiproxyURL, vk, modelKey, wantCategory string,
) error {
	req, err := http.NewRequest(http.MethodGet, openAIEndpointURL(aiproxyURL, "/models"), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+vk)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("GET /models returned HTTP %d", resp.StatusCode)
	}
	var listing struct {
		Data []struct {
			ID       string `json:"id"`
			Category string `json:"category"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listing); err != nil {
		return errors.Wrap(err, "parse models response")
	}
	for _, item := range listing.Data {
		if item.ID != modelKey {
			continue
		}
		if item.Category != wantCategory {
			return errors.Errorf("model %s listed with category %q, want %q", modelKey, item.Category, wantCategory)
		}
		return nil
	}
	return errors.Errorf("model %s is not listed by GET /models", modelKey)
}
