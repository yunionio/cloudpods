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

package openai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
)

// ResponsesAnthropicStreamConverter converts Anthropic Messages SSE to Responses SSE.
type ResponsesAnthropicStreamConverter struct {
	requestModel string
	toolMap      CodexToolMap
	inner        *ResponsesStreamConverter
	textBuf      strings.Builder
	reasonBuf    strings.Builder
	toolArgs     map[int]*strings.Builder
	toolNames    map[int]string
	toolIDs      map[int]string
	messageID    string
	model        string
	inputTokens  int
	outputTokens int
	created      bool
	completed    bool
}

// NewResponsesAnthropicStreamConverter creates anthropic-to-responses stream state.
func NewResponsesAnthropicStreamConverter(requestModel string, toolMap CodexToolMap) *ResponsesAnthropicStreamConverter {
	return &ResponsesAnthropicStreamConverter{
		requestModel: requestModel,
		toolMap:      toolMap,
		inner:        NewResponsesStreamConverter(requestModel, toolMap),
		toolArgs:     make(map[int]*strings.Builder),
		toolNames:    make(map[int]string),
		toolIDs:      make(map[int]string),
	}
}

// Feed processes one Anthropic SSE event (event type + data JSON).
func (s *ResponsesAnthropicStreamConverter) Feed(eventType string, payload []byte, endOfStream bool) ([]ResponsesStreamEvent, error) {
	if endOfStream && !s.completed {
		return s.inner.emitCompleted()
	}
	if len(payload) == 0 {
		return nil, nil
	}
	et := strings.TrimSpace(eventType)
	switch et {
	case "message_start":
		var data struct {
			Message struct {
				ID    string `json:"id"`
				Model string `json:"model"`
			} `json:"message"`
		}
		if json.Unmarshal(payload, &data) == nil {
			s.messageID = data.Message.ID
			s.model = data.Message.Model
		}
		if !s.created {
			s.created = true
			s.inner.created = true
			if s.messageID != "" {
				s.inner.responseID = s.messageID
			} else {
				s.inner.responseID = "resp_" + uuid.New().String()
			}
			if s.model != "" {
				s.inner.model = s.model
			} else {
				s.inner.model = s.requestModel
			}
			ev := s.mustLifecycle("response.created", "in_progress")
			return []ResponsesStreamEvent{ev}, nil
		}
	case "content_block_start":
		var data struct {
			Index        int `json:"index"`
			ContentBlock struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block"`
		}
		if json.Unmarshal(payload, &data) != nil {
			return nil, nil
		}
		switch data.ContentBlock.Type {
		case "thinking":
			s.inner.reasonStarted = false
			events, err := s.inner.appendReasoning("")
			if err != nil {
				return nil, err
			}
			return events, nil
		case "text":
			events, err := s.inner.appendText("")
			if err != nil {
				return nil, err
			}
			return events, nil
		case "tool_use":
			s.toolNames[data.Index] = data.ContentBlock.Name
			s.toolIDs[data.Index] = data.ContentBlock.ID
			s.toolArgs[data.Index] = &strings.Builder{}
			tc := ToolCall{Index: data.Index, ID: data.ContentBlock.ID, Function: ToolFunction{Name: data.ContentBlock.Name}}
			return s.inner.appendToolDelta(tc)
		}
	case "content_block_delta":
		var data struct {
			Index int `json:"index"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				Thinking    string `json:"thinking"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta"`
		}
		if json.Unmarshal(payload, &data) != nil {
			return nil, nil
		}
		switch data.Delta.Type {
		case "thinking_delta":
			return s.inner.appendReasoning(data.Delta.Thinking)
		case "text_delta":
			return s.inner.appendText(data.Delta.Text)
		case "input_json_delta":
			if b, ok := s.toolArgs[data.Index]; ok {
				b.WriteString(data.Delta.PartialJSON)
			}
			name := s.toolNames[data.Index]
			tc := ToolCall{Index: data.Index, ID: s.toolIDs[data.Index], Function: ToolFunction{Name: name, Arguments: data.Delta.PartialJSON}}
			return s.inner.appendToolDelta(tc)
		}
	case "message_delta":
		var data struct {
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(payload, &data) == nil {
			s.inner.inputTokens = data.Usage.InputTokens
			s.inner.outputTokens = data.Usage.OutputTokens
		}
	case "message_stop":
		if !s.completed {
			s.completed = true
			return s.inner.emitCompleted()
		}
	}
	return nil, nil
}

func (s *ResponsesAnthropicStreamConverter) mustLifecycle(eventType, status string) ResponsesStreamEvent {
	ev, err := s.inner.lifecycleEvent(eventType, status)
	if err != nil {
		return ResponsesStreamEvent{Event: eventType, Data: []byte(fmt.Sprintf(`{"type":%q}`, eventType))}
	}
	return ev
}

// AnthropicToProviderChunks maps events to providerapi chunks.
func AnthropicToProviderChunks(events []ResponsesStreamEvent) []providerapi.ResponsesStreamChunk {
	return ToProviderChunks(events)
}
