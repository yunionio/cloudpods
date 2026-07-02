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
	"strings"
)

// AnthropicStreamEvent is one Anthropic Messages SSE event for upstream clients.
type AnthropicStreamEvent struct {
	Event string
	Data  []byte
}

// AnthropicStreamConverter converts OpenAI chat.completion.chunk SSE payloads to Anthropic SSE events.
type AnthropicStreamConverter struct {
	requestModel   string
	messageStarted bool
	hasOpenBlock   bool
	openBlockIndex int
	closingEmitted bool
	messageID      string
	model          string
	stopReason     string
	inputTokens    int
	outputTokens   int
	blockIndex     int
	activeTools    map[int]*streamToolState
}

type streamToolState struct {
	blockIdx int
	id       string
	name     string
}

// NewAnthropicStreamConverter creates stream conversion state for one Anthropic Messages response.
func NewAnthropicStreamConverter(requestModel string) *AnthropicStreamConverter {
	return &AnthropicStreamConverter{
		requestModel:   requestModel,
		activeTools:    make(map[int]*streamToolState),
		openBlockIndex: -1,
	}
}

// Feed processes one OpenAI SSE data payload (without the "data:" prefix).
func (s *AnthropicStreamConverter) Feed(payload []byte, endOfStream bool) ([]AnthropicStreamEvent, error) {
	if len(payload) == 0 {
		if endOfStream && !s.closingEmitted {
			return s.emitClosing()
		}
		return nil, nil
	}
	if string(payload) == "[DONE]" {
		if !s.closingEmitted {
			return s.emitClosing()
		}
		return nil, nil
	}
	var chunk struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Delta struct {
				Content   *string    `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return nil, nil
	}
	if chunk.ID != "" && s.messageID == "" {
		s.messageID = chunk.ID
	}
	if chunk.Model != "" && s.model == "" {
		s.model = chunk.Model
	}
	if chunk.Usage != nil {
		s.inputTokens = chunk.Usage.PromptTokens
		s.outputTokens = chunk.Usage.CompletionTokens
	}
	var out []AnthropicStreamEvent
	if chunk.Usage != nil && len(chunk.Choices) == 0 {
		closing, err := s.emitClosing()
		if err != nil {
			return nil, err
		}
		return closing, nil
	}
	if len(chunk.Choices) == 0 {
		if endOfStream && !s.closingEmitted {
			return s.emitClosing()
		}
		return nil, nil
	}
	choice := chunk.Choices[0]
	if !s.messageStarted {
		events, err := s.emitMessageStart()
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	if choice.Delta.Content != nil && *choice.Delta.Content != "" {
		if !s.hasOpenBlock {
			events, err := s.emitTextBlockStart()
			if err != nil {
				return nil, err
			}
			out = append(out, events...)
		}
		events, err := s.emitTextDelta(*choice.Delta.Content)
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	for _, tc := range choice.Delta.ToolCalls {
		events, err := s.handleToolDelta(tc)
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	if choice.FinishReason != "" {
		s.stopReason = openAIFinishReasonToAnthropic(choice.FinishReason)
	}
	if endOfStream && !s.closingEmitted {
		closing, err := s.emitClosing()
		if err != nil {
			return nil, err
		}
		out = append(out, closing...)
	}
	return out, nil
}

func (s *AnthropicStreamConverter) emitMessageStart() ([]AnthropicStreamEvent, error) {
	s.messageStarted = true
	model := s.model
	if model == "" {
		model = s.requestModel
	}
	data, err := json.Marshal(map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            s.messageID,
			"type":          "message",
			"role":          "assistant",
			"model":         model,
			"content":       []interface{}{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]interface{}{
				"input_tokens":  0,
				"output_tokens": 0,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return []AnthropicStreamEvent{{Event: "message_start", Data: data}}, nil
}

func (s *AnthropicStreamConverter) emitTextBlockStart() ([]AnthropicStreamEvent, error) {
	s.hasOpenBlock = true
	s.openBlockIndex = s.blockIndex
	data, err := json.Marshal(map[string]interface{}{
		"type":          "content_block_start",
		"index":         s.blockIndex,
		"content_block": map[string]interface{}{"type": "text", "text": ""},
	})
	if err != nil {
		return nil, err
	}
	return []AnthropicStreamEvent{{Event: "content_block_start", Data: data}}, nil
}

func (s *AnthropicStreamConverter) emitTextDelta(text string) ([]AnthropicStreamEvent, error) {
	data, err := json.Marshal(map[string]interface{}{
		"type":  "content_block_delta",
		"index": s.blockIndex,
		"delta": map[string]interface{}{"type": "text_delta", "text": text},
	})
	if err != nil {
		return nil, err
	}
	return []AnthropicStreamEvent{{Event: "content_block_delta", Data: data}}, nil
}

func (s *AnthropicStreamConverter) handleToolDelta(tc ToolCall) ([]AnthropicStreamEvent, error) {
	var out []AnthropicStreamEvent
	idx := tc.Index
	st, ok := s.activeTools[idx]
	if !ok {
		if s.hasOpenBlock {
			events, err := s.closeOpenBlock()
			if err != nil {
				return nil, err
			}
			out = append(out, events...)
		}
		st = &streamToolState{blockIdx: s.blockIndex}
		s.activeTools[idx] = st
		id := strings.TrimSpace(tc.ID)
		name := strings.TrimSpace(tc.Function.Name)
		if id != "" {
			st.id = id
		}
		if name != "" {
			st.name = name
		}
		if st.id == "" {
			st.id = "toolu_" + st.name
		}
		data, err := json.Marshal(map[string]interface{}{
			"type":  "content_block_start",
			"index": st.blockIdx,
			"content_block": map[string]interface{}{
				"type":  "tool_use",
				"id":    st.id,
				"name":  st.name,
				"input": map[string]interface{}{},
			},
		})
		if err != nil {
			return nil, err
		}
		s.hasOpenBlock = true
		s.openBlockIndex = st.blockIdx
		s.blockIndex++
		out = append(out, AnthropicStreamEvent{Event: "content_block_start", Data: data})
	}
	if tc.Function.Name != "" {
		st.name = tc.Function.Name
	}
	if tc.ID != "" {
		st.id = tc.ID
	}
	if tc.Function.Arguments == "" {
		return out, nil
	}
	data, err := json.Marshal(map[string]interface{}{
		"type":  "content_block_delta",
		"index": st.blockIdx,
		"delta": map[string]interface{}{
			"type":         "input_json_delta",
			"partial_json": tc.Function.Arguments,
		},
	})
	if err != nil {
		return nil, err
	}
	out = append(out, AnthropicStreamEvent{Event: "content_block_delta", Data: data})
	return out, nil
}

func (s *AnthropicStreamConverter) closeOpenBlock() ([]AnthropicStreamEvent, error) {
	if !s.hasOpenBlock {
		return nil, nil
	}
	idx := s.openBlockIndex
	if idx < 0 {
		idx = s.blockIndex
	}
	data, err := json.Marshal(map[string]interface{}{
		"type":  "content_block_stop",
		"index": idx,
	})
	if err != nil {
		return nil, err
	}
	s.hasOpenBlock = false
	s.openBlockIndex = -1
	events := []AnthropicStreamEvent{{Event: "content_block_stop", Data: data}}
	if s.blockIndex <= idx {
		s.blockIndex = idx + 1
	}
	return events, nil
}

func (s *AnthropicStreamConverter) emitClosing() ([]AnthropicStreamEvent, error) {
	if s.closingEmitted {
		return nil, nil
	}
	s.closingEmitted = true
	var out []AnthropicStreamEvent
	if s.hasOpenBlock {
		events, err := s.closeOpenBlock()
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	stopReason := s.stopReason
	if stopReason == "" {
		stopReason = "end_turn"
	}
	deltaData, err := json.Marshal(map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]interface{}{
			"output_tokens": s.outputTokens,
		},
	})
	if err != nil {
		return nil, err
	}
	out = append(out, AnthropicStreamEvent{Event: "message_delta", Data: deltaData})

	stopData, err := json.Marshal(map[string]interface{}{"type": "message_stop"})
	if err != nil {
		return nil, err
	}
	out = append(out, AnthropicStreamEvent{Event: "message_stop", Data: stopData})
	return out, nil
}
