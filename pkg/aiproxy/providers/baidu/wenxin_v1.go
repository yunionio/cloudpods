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

package baidu

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

func buildWenxinV1ChatRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	token, err := ResolveAccessToken(ctx.APIKey)
	if err != nil {
		return nil, err
	}
	msgs, _, err := openai.ParseMessages(body)
	if err != nil {
		return nil, err
	}
	wenxinMsgs := make([]map[string]interface{}, 0, len(msgs))
	for _, m := range msgs {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role == "tool" {
			role = "user"
		}
		text := openai.MessageTextContent(m.Content)
		if text == "" {
			continue
		}
		wenxinMsgs = append(wenxinMsgs, map[string]interface{}{
			"role":    role,
			"content": text,
		})
	}
	if len(wenxinMsgs) == 0 {
		return nil, fmt.Errorf("no convertible messages for wenxin")
	}
	reqBody := map[string]interface{}{
		"messages": wenxinMsgs,
		"stream":   stream,
	}
	if v, ok := openai.IntParam(body, "max_tokens", "max_completion_tokens"); ok {
		reqBody["max_output_tokens"] = v
	}
	if v, ok := openai.FloatParam(body, "temperature"); ok {
		reqBody["temperature"] = v
	}
	if v, ok := openai.FloatParam(body, "top_p"); ok {
		reqBody["top_p"] = v
	}
	raw, err := openai.MarshalJSON(reqBody)
	if err != nil {
		return nil, err
	}
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    wenxinChatURL(ctx.BaseURL, ctx.UpstreamModel, token),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: raw,
	}, nil
}

func normalizeWenxinV1ChatResponse(body []byte) ([]byte, error) {
	var probe struct {
		Choices json.RawMessage `json:"choices"`
		Result  *string         `json:"result"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return body, nil
	}
	if len(probe.Choices) > 0 {
		return body, nil
	}
	if probe.Result == nil {
		return body, nil
	}
	var resp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		Result  string `json:"result"`
		Usage   struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, nil
	}
	if resp.ErrorCode != 0 || resp.ErrorMsg != "" {
		return nil, fmt.Errorf("wenxin error %d: %s", resp.ErrorCode, resp.ErrorMsg)
	}
	if resp.Result == "" && resp.ID == "" {
		return body, nil
	}
	out, err := openai.MarshalJSON(openai.NewChatCompletion(
		"",
		resp.ID,
		resp.Result,
		"stop",
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
	))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func convertWenxinV1StreamEvent(payload []byte, state *providerapi.StreamState) ([]providerapi.StreamChunk, error) {
	if state == nil || len(payload) == 0 {
		return nil, nil
	}
	var chunk struct {
		ID      string `json:"id"`
		Result  string `json:"result"`
		IsEnd   bool   `json:"is_end"`
		IsTrunc bool   `json:"is_truncated"`
	}
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return nil, nil
	}
	if chunk.ID != "" {
		state.ResponseID = chunk.ID
	}
	finish := ""
	if chunk.IsEnd {
		finish = "stop"
	}
	if chunk.Result == "" && finish == "" {
		return nil, nil
	}
	data, err := openai.MarshalJSON(openai.NewStreamChunk(state.Model, state.ResponseID, 0, chunk.Result, finish))
	if err != nil {
		return nil, err
	}
	return []providerapi.StreamChunk{{Data: data}}, nil
}

func buildWenxinV1EmbeddingsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	token, err := ResolveAccessToken(ctx.APIKey)
	if err != nil {
		return nil, err
	}
	texts, rawInput, isText, err := openai.ParseEmbeddingInput(body)
	if err != nil {
		return nil, err
	}
	var reqBody map[string]interface{}
	if isText {
		reqBody = map[string]interface{}{"input": texts}
	} else {
		if err := json.Unmarshal(rawInput, &reqBody); err != nil {
			return nil, fmt.Errorf("invalid embeddings input")
		}
	}
	raw, err := openai.MarshalJSON(reqBody)
	if err != nil {
		return nil, err
	}
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    wenxinEmbeddingsURL(ctx.BaseURL, ctx.UpstreamModel, token),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: raw,
	}, nil
}

func normalizeWenxinV1EmbeddingsResponse(body []byte) ([]byte, error) {
	var probe struct {
		Object  string          `json:"object"`
		Choices json.RawMessage `json:"choices"`
	}
	if err := json.Unmarshal(body, &probe); err == nil && (probe.Object == "list" || len(probe.Choices) > 0) {
		return body, nil
	}
	var resp struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
		ErrorCode int    `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, nil
	}
	if resp.ErrorCode != 0 || resp.ErrorMsg != "" {
		return nil, fmt.Errorf("wenxin embeddings error %d: %s", resp.ErrorCode, resp.ErrorMsg)
	}
	vectors := make([][]float64, len(resp.Data))
	for i := range resp.Data {
		vectors[i] = resp.Data[i].Embedding
	}
	promptTokens := resp.Usage.PromptTokens
	if promptTokens == 0 {
		promptTokens = resp.Usage.TotalTokens
	}
	return openai.NewEmbeddingsResponse("", vectors, promptTokens)
}
