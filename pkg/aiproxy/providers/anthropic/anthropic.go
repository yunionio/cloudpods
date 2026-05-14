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

package anthropic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	"yunion.io/x/onecloud/pkg/aiproxy/providers/openai"
)

const apiVersion = "2023-06-01"

type provider struct{}

// New returns the Anthropic Messages API provider adapter.
func New() providerapi.Provider {
	return &provider{}
}

func (p *provider) Key() string {
	return "anthropic"
}

func (p *provider) BuildUpstreamRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict, stream bool) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	msgs, system, err := openai.ParseMessages(body)
	if err != nil {
		return nil, err
	}
	anthropicMsgs, err := openai.MessagesToAnthropic(msgs)
	if err != nil {
		return nil, err
	}
	maxTokens := 4096
	if v, ok := openai.IntParam(body, "max_tokens", "max_completion_tokens"); ok {
		maxTokens = v
	}
	reqBody := map[string]interface{}{
		"model":      ctx.UpstreamModel,
		"max_tokens": maxTokens,
		"messages":   anthropicMsgs,
	}
	if system != "" {
		reqBody["system"] = system
	}
	if stream {
		reqBody["stream"] = true
	}
	if v, ok := openai.FloatParam(body, "temperature"); ok {
		reqBody["temperature"] = v
	}
	if v, ok := openai.FloatParam(body, "top_p"); ok {
		reqBody["top_p"] = v
	}
	if tools, toolChoice, err := openai.ExtractTools(body); err != nil {
		return nil, err
	} else if len(tools) > 0 {
		reqBody["tools"] = openai.ToolsToAnthropic(tools)
		if tc := openai.ToolChoiceToAnthropic(toolChoice); tc != nil {
			reqBody["tool_choice"] = tc
		}
	}
	raw, err := openai.MarshalJSON(reqBody)
	if err != nil {
		return nil, err
	}
	base := strings.TrimSpace(ctx.BaseURL)
	if base == "" {
		base = "https://api.anthropic.com"
	}
	return &providerapi.HTTPRequest{
		Method: http.MethodPost,
		URL:    openai.JoinURL(base, "/v1/messages"),
		Headers: map[string]string{
			"x-api-key":         strings.TrimSpace(ctx.APIKey),
			"anthropic-version": apiVersion,
			"Content-Type":      "application/json",
		},
		Body: raw,
	}, nil
}

func (p *provider) NormalizeResponse(body []byte) ([]byte, error) {
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Content []struct {
			Type  string                 `json:"type"`
			Text  string                 `json:"text"`
			ID    string                 `json:"id"`
			Name  string                 `json:"name"`
			Input map[string]interface{} `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, nil
	}
	blocks := make([]openai.AnthropicBlock, len(resp.Content))
	for i, c := range resp.Content {
		blocks[i] = openai.AnthropicBlock{
			Type:  c.Type,
			Text:  c.Text,
			ID:    c.ID,
			Name:  c.Name,
			Input: c.Input,
		}
	}
	msg := openai.AnthropicBlocksToAssistant(blocks)
	out, err := openai.MarshalJSON(openai.NewChatCompletionWithTools(
		resp.Model,
		resp.ID,
		msg,
		resp.StopReason,
		resp.Usage.InputTokens,
		resp.Usage.OutputTokens,
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
	var wrap struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &wrap); err != nil {
		return nil, nil
	}
	switch wrap.Type {
	case "message_start":
		var start struct {
			Message struct {
				ID    string `json:"id"`
				Model string `json:"model"`
			} `json:"message"`
		}
		if err := json.Unmarshal(payload, &start); err == nil {
			if start.Message.ID != "" {
				state.ResponseID = start.Message.ID
			}
			if start.Message.Model != "" {
				state.Model = start.Message.Model
			}
		}
		return nil, nil
	case "content_block_start":
		var start struct {
			ContentBlock struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block"`
		}
		if err := json.Unmarshal(payload, &start); err != nil {
			return nil, nil
		}
		if start.ContentBlock.Type != "tool_use" {
			return nil, nil
		}
		state.InToolBlock = true
		state.ToolID = start.ContentBlock.ID
		state.ToolName = start.ContentBlock.Name
		state.ToolArgsPending = ""
		chunk, err := openai.MarshalJSON(openai.NewStreamChunkToolDelta(
			state.Model, state.ResponseID, state.ToolIndex,
			openai.ToolCall{
				ID:   state.ToolID,
				Type: "function",
				Function: openai.ToolFunction{
					Name: state.ToolName,
				},
			}, "",
		))
		if err != nil {
			return nil, err
		}
		return []providerapi.StreamChunk{{Data: chunk}}, nil
	case "content_block_delta":
		var delta struct {
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta"`
		}
		if err := json.Unmarshal(payload, &delta); err != nil {
			return nil, nil
		}
		if delta.Delta.Type == "input_json_delta" && delta.Delta.PartialJSON != "" {
			state.ToolArgsPending += delta.Delta.PartialJSON
			chunk, err := openai.MarshalJSON(openai.NewStreamChunkToolDelta(
				state.Model, state.ResponseID, state.ToolIndex,
				openai.ToolCall{
					Function: openai.ToolFunction{
						Arguments: delta.Delta.PartialJSON,
					},
				}, "",
			))
			if err != nil {
				return nil, err
			}
			return []providerapi.StreamChunk{{Data: chunk}}, nil
		}
		if delta.Delta.Text == "" {
			return nil, nil
		}
		chunk, err := openai.MarshalJSON(openai.NewStreamChunk(state.Model, state.ResponseID, 0, delta.Delta.Text, ""))
		if err != nil {
			return nil, err
		}
		return []providerapi.StreamChunk{{Data: chunk}}, nil
	case "content_block_stop":
		if state.InToolBlock {
			state.InToolBlock = false
			state.ToolIndex++
			state.ToolArgsPending = ""
		}
		return nil, nil
	case "message_delta":
		var end struct {
			Delta struct {
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
		}
		if err := json.Unmarshal(payload, &end); err != nil {
			return nil, nil
		}
		chunk, err := openai.MarshalJSON(openai.NewStreamChunk(state.Model, state.ResponseID, 0, "", end.Delta.StopReason))
		if err != nil {
			return nil, err
		}
		return []providerapi.StreamChunk{{Data: chunk}}, nil
	default:
		return nil, nil
	}
}

func (p *provider) BuildEmbeddingsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	return nil, fmt.Errorf("provider %q does not support embeddings", p.Key())
}

func (p *provider) NormalizeEmbeddingsResponse(body []byte) ([]byte, error) {
	return nil, fmt.Errorf("provider %q does not support embeddings", p.Key())
}

func (p *provider) BuildImagesGenerationsRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	return nil, fmt.Errorf("provider %q does not support images/generations", p.Key())
}

func (p *provider) NormalizeImagesGenerationsResponse(body []byte) ([]byte, error) {
	return nil, fmt.Errorf("provider %q does not support images/generations", p.Key())
}
