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

package gemini

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

type provider struct{}

// New returns the Google Gemini provider adapter.
func New() providerapi.Provider {
	return &provider{}
}

func (p *provider) Key() string {
	return "gemini"
}

func (p *provider) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	msgs, system, err := openai.ParseMessages(body)
	if err != nil {
		return nil, err
	}
	contents, err := openai.MessagesToGemini(msgs)
	if err != nil {
		return nil, err
	}
	genConfig := map[string]interface{}{}
	if v, ok := openai.IntParam(body, "max_tokens", "max_completion_tokens"); ok {
		genConfig["maxOutputTokens"] = v
	}
	if v, ok := openai.FloatParam(body, "temperature"); ok {
		genConfig["temperature"] = v
	}
	if v, ok := openai.FloatParam(body, "top_p"); ok {
		genConfig["topP"] = v
	}
	reqBody := map[string]interface{}{
		"contents": contents,
	}
	if len(genConfig) > 0 {
		reqBody["generationConfig"] = genConfig
	}
	if system != "" {
		reqBody["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": system},
			},
		}
	}
	if tools, _, err := openai.ExtractTools(body); err != nil {
		return nil, err
	} else if gemTools := openai.ToolsToGemini(tools); len(gemTools) > 0 {
		reqBody["tools"] = gemTools
	}
	raw, err := openai.MarshalJSON(reqBody)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(strings.TrimSpace(ctx.BaseURL), "/")
	if base == "" {
		base = "https://generativelanguage.googleapis.com/v1beta"
	}
	action := "generateContent"
	if stream {
		action = "streamGenerateContent"
	}
	modelPath := fmt.Sprintf("/models/%s:%s", ctx.UpstreamModel, action)
	url := openai.JoinURL(base, modelPath)
	if stream {
		url += "?alt=sse"
	}
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    url,
		Headers: map[string]string{
			"x-goog-api-key": strings.TrimSpace(ctx.APIKey),
			"Content-Type":   "application/json",
		},
		Body: raw,
	}, nil
}

func (p *provider) NormalizeResponse(body []byte) ([]byte, error) {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []openai.GeminiPart `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, nil
	}
	finish := "stop"
	msg := openai.AssistantMessage{}
	if len(resp.Candidates) > 0 {
		msg = openai.GeminiPartsToAssistant(resp.Candidates[0].Content.Parts)
		if resp.Candidates[0].FinishReason != "" {
			finish = resp.Candidates[0].FinishReason
		}
	}
	out, err := openai.MarshalJSON(openai.NewChatCompletionWithTools(
		"",
		"gemini-"+uuid.New().String(),
		msg,
		finish,
		resp.UsageMetadata.PromptTokenCount,
		resp.UsageMetadata.CandidatesTokenCount,
	))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (p *provider) OpenAIStreamPassthrough() bool {
	return false
}

func (p *provider) ConvertStreamEvent(eventType string, payload []byte, state *providerapi.StreamState) ([]providerapi.StreamChunk, error) {
	if state == nil || len(payload) == 0 {
		return nil, nil
	}
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []openai.GeminiPart `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return nil, nil
	}
	if len(resp.Candidates) == 0 {
		return nil, nil
	}
	c := resp.Candidates[0]
	if state.ResponseID == "" {
		state.ResponseID = "gemini-" + uuid.New().String()
	}
	var chunks []providerapi.StreamChunk
	msg := openai.GeminiPartsToAssistant(c.Content.Parts)
	if msg.Content != "" {
		chunk, err := openai.MarshalJSON(openai.NewStreamChunk(state.Model, state.ResponseID, 0, msg.Content, ""))
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, providerapi.StreamChunk{Data: chunk})
	}
	for _, tc := range msg.ToolCalls {
		chunk, err := openai.MarshalJSON(openai.NewStreamChunkToolDelta(
			state.Model, state.ResponseID, state.ToolIndex, tc, "",
		))
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, providerapi.StreamChunk{Data: chunk})
		state.ToolIndex++
	}
	if c.FinishReason != "" {
		chunk, err := openai.MarshalJSON(openai.NewStreamChunk(state.Model, state.ResponseID, 0, "", c.FinishReason))
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, providerapi.StreamChunk{Data: chunk})
	}
	if len(chunks) == 0 {
		return nil, nil
	}
	return chunks, nil
}

func (p *provider) BuildEmbeddingsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	texts, _, isText, err := openai.ParseEmbeddingInput(body)
	if err != nil {
		return nil, err
	}
	if !isText {
		return openai.DefaultEmbeddingCompat().BuildEmbeddingsRequest(ctx, body)
	}
	base := strings.TrimRight(strings.TrimSpace(ctx.BaseURL), "/")
	if base == "" {
		base = "https://generativelanguage.googleapis.com/v1beta"
	}
	modelRef := modelRef(ctx.UpstreamModel)
	var raw []byte
	var action string
	if len(texts) == 1 {
		action = "embedContent"
		reqBody := map[string]interface{}{
			"content": map[string]interface{}{
				"parts": []map[string]interface{}{{"text": texts[0]}},
			},
		}
		raw, err = openai.MarshalJSON(reqBody)
	} else {
		action = "batchEmbedContents"
		requests := make([]map[string]interface{}, len(texts))
		for i, text := range texts {
			requests[i] = map[string]interface{}{
				"model": modelRef,
				"content": map[string]interface{}{
					"parts": []map[string]interface{}{{"text": text}},
				},
			}
		}
		raw, err = openai.MarshalJSON(map[string]interface{}{"requests": requests})
	}
	if err != nil {
		return nil, err
	}
	url := openai.JoinURL(base, fmt.Sprintf("/models/%s:%s", ctx.UpstreamModel, action))
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    url,
		Headers: map[string]string{
			"x-goog-api-key": strings.TrimSpace(ctx.APIKey),
			"Content-Type":   "application/json",
		},
		Body: raw,
	}, nil
}

func (p *provider) NormalizeEmbeddingsResponse(body []byte) ([]byte, error) {
	var single struct {
		Embedding struct {
			Values []float64 `json:"values"`
		} `json:"embedding"`
	}
	if err := json.Unmarshal(body, &single); err == nil && len(single.Embedding.Values) > 0 {
		return openai.NewEmbeddingsResponse("", [][]float64{single.Embedding.Values}, 0)
	}
	var batch struct {
		Embeddings []struct {
			Values []float64 `json:"values"`
		} `json:"embeddings"`
	}
	if err := json.Unmarshal(body, &batch); err != nil {
		return body, nil
	}
	vectors := make([][]float64, len(batch.Embeddings))
	for i := range batch.Embeddings {
		vectors[i] = batch.Embeddings[i].Values
	}
	return openai.NewEmbeddingsResponse("", vectors, 0)
}

func modelRef(model string) string {
	model = strings.TrimSpace(model)
	if strings.HasPrefix(model, "models/") {
		return model
	}
	return "models/" + model
}

func (p *provider) BuildImagesGenerationsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	prompt, err := openai.ParseImagePrompt(body)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(strings.TrimSpace(ctx.BaseURL), "/")
	if base == "" {
		base = "https://generativelanguage.googleapis.com/v1beta"
	}
	reqBody := map[string]interface{}{
		"instances": []map[string]interface{}{
			{"prompt": prompt},
		},
		"parameters": map[string]interface{}{
			"sampleCount": openai.ImageCount(body),
			"aspectRatio": openai.SizeToAspectRatio(openai.ImageSize(body)),
		},
	}
	raw, err := openai.MarshalJSON(reqBody)
	if err != nil {
		return nil, err
	}
	url := openai.JoinURL(base, fmt.Sprintf("/models/%s:predict", ctx.UpstreamModel))
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    url,
		Headers: map[string]string{
			"x-goog-api-key": strings.TrimSpace(ctx.APIKey),
			"Content-Type":   "application/json",
		},
		Body: raw,
	}, nil
}

func (p *provider) NormalizeImagesGenerationsResponse(body []byte) ([]byte, error) {
	var predict struct {
		Predictions []struct {
			BytesBase64Encoded string `json:"bytesBase64Encoded"`
		} `json:"predictions"`
	}
	if err := json.Unmarshal(body, &predict); err == nil && len(predict.Predictions) > 0 {
		items := make([]openai.ImageItem, 0, len(predict.Predictions))
		for _, pred := range predict.Predictions {
			if pred.BytesBase64Encoded == "" {
				continue
			}
			items = append(items, openai.ImageItem{B64: pred.BytesBase64Encoded})
		}
		if len(items) > 0 {
			return openai.NewImagesGenerationsResponse(items)
		}
	}
	var generated struct {
		GeneratedImages []struct {
			Image struct {
				ImageBytes string `json:"imageBytes"`
			} `json:"image"`
		} `json:"generatedImages"`
	}
	if err := json.Unmarshal(body, &generated); err == nil && len(generated.GeneratedImages) > 0 {
		items := make([]openai.ImageItem, 0, len(generated.GeneratedImages))
		for _, img := range generated.GeneratedImages {
			if img.Image.ImageBytes == "" {
				continue
			}
			items = append(items, openai.ImageItem{B64: img.Image.ImageBytes})
		}
		if len(items) > 0 {
			return openai.NewImagesGenerationsResponse(items)
		}
	}
	return body, nil
}
