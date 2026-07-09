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

// ResponsesStreamEvent is one Responses API SSE event.
type ResponsesStreamEvent struct {
	Event string
	Data  []byte
}

// ResponsesStreamConverter converts OpenAI chat.completion.chunk SSE to Responses SSE events.
type ResponsesStreamConverter struct {
	requestModel  string
	toolMap       CodexToolMap
	created       bool
	completed     bool
	seq           int64
	responseID    string
	model         string
	outputIndex   int
	textItemID    string
	reasonItemID  string
	textStarted   bool
	reasonStarted bool
	textBuf       strings.Builder
	reasonBuf     strings.Builder
	inputTokens   int
	outputTokens  int
	activeTools   map[int]*responsesStreamToolState
}

type responsesStreamToolState struct {
	outputIndex int
	itemID      string
	callID      string
	name        string
	namespace   string
	argsBuf     strings.Builder
	added       bool
}

// NewResponsesStreamConverter creates stream state for one Responses response.
func NewResponsesStreamConverter(requestModel string, toolMap CodexToolMap) *ResponsesStreamConverter {
	return &ResponsesStreamConverter{
		requestModel: requestModel,
		toolMap:      toolMap,
		activeTools:  make(map[int]*responsesStreamToolState),
	}
}

// Feed processes one chat SSE data payload.
func (s *ResponsesStreamConverter) Feed(payload []byte, endOfStream bool) ([]ResponsesStreamEvent, error) {
	if len(payload) == 0 {
		if endOfStream && !s.completed {
			return s.emitCompleted()
		}
		return nil, nil
	}
	if string(payload) == "[DONE]" {
		if !s.completed {
			return s.emitCompleted()
		}
		return nil, nil
	}
	var chunk struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Delta struct {
				Content          *string    `json:"content"`
				ReasoningContent *string    `json:"reasoning_content"`
				ToolCalls        []ToolCall `json:"tool_calls"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return nil, nil
	}
	var out []ResponsesStreamEvent
	if !s.created {
		s.created = true
		if chunk.ID != "" {
			s.responseID = chunk.ID
		} else {
			s.responseID = "resp_" + uuid.New().String()
		}
		if chunk.Model != "" {
			s.model = chunk.Model
		} else {
			s.model = s.requestModel
		}
		created, err := s.lifecycleEvent("response.created", "in_progress")
		if err != nil {
			return nil, err
		}
		out = append(out, created)
	}
	if chunk.Usage != nil {
		s.inputTokens = chunk.Usage.PromptTokens
		s.outputTokens = chunk.Usage.CompletionTokens
	}
	if len(chunk.Choices) == 0 {
		return out, nil
	}
	delta := chunk.Choices[0].Delta
	if delta.ReasoningContent != nil && *delta.ReasoningContent != "" {
		events, err := s.appendReasoning(*delta.ReasoningContent)
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	if delta.Content != nil && *delta.Content != "" {
		events, err := s.appendText(*delta.Content)
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	for _, tc := range delta.ToolCalls {
		events, err := s.appendToolDelta(tc)
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	if chunk.Choices[0].FinishReason == "length" && !s.completed {
		events, err := s.emitCompleted()
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	return out, nil
}

func (s *ResponsesStreamConverter) nextSeq() int64 {
	s.seq++
	return s.seq
}

func (s *ResponsesStreamConverter) lifecycleEvent(eventType, status string) (ResponsesStreamEvent, error) {
	data := map[string]interface{}{
		"type":            eventType,
		"sequence_number": s.nextSeq(),
		"response": map[string]interface{}{
			"id":     s.responseID,
			"object": "response",
			"status": status,
			"model":  s.model,
			"output": []interface{}{},
		},
	}
	b, err := json.Marshal(data)
	if err != nil {
		return ResponsesStreamEvent{}, err
	}
	return ResponsesStreamEvent{Event: eventType, Data: b}, nil
}

func (s *ResponsesStreamConverter) appendText(text string) ([]ResponsesStreamEvent, error) {
	var out []ResponsesStreamEvent
	if !s.textStarted {
		s.textStarted = true
		s.textItemID = fmt.Sprintf("msg_%d", s.outputIndex)
		added, err := s.outputItemAdded("message", s.textItemID, map[string]interface{}{
			"type":    "message",
			"id":      s.textItemID,
			"status":  "in_progress",
			"role":    "assistant",
			"content": []interface{}{},
		})
		if err != nil {
			return nil, err
		}
		out = append(out, added)
		part, err := s.contentPartAdded(s.textItemID, 0)
		if err != nil {
			return nil, err
		}
		out = append(out, part)
		s.outputIndex++
	}
	s.textBuf.WriteString(text)
	delta, err := s.textDelta(text)
	if err != nil {
		return nil, err
	}
	out = append(out, delta)
	return out, nil
}

func (s *ResponsesStreamConverter) appendReasoning(text string) ([]ResponsesStreamEvent, error) {
	var out []ResponsesStreamEvent
	if !s.reasonStarted {
		s.reasonStarted = true
		s.reasonItemID = fmt.Sprintf("rs_%d", s.outputIndex)
		added, err := s.outputItemAdded("reasoning", s.reasonItemID, map[string]interface{}{
			"type":    "reasoning",
			"id":      s.reasonItemID,
			"status":  "in_progress",
			"summary": []interface{}{},
		})
		if err != nil {
			return nil, err
		}
		out = append(out, added)
		s.outputIndex++
	}
	s.reasonBuf.WriteString(text)
	data := map[string]interface{}{
		"type":            "response.reasoning_summary_text.delta",
		"sequence_number": s.nextSeq(),
		"item_id":         s.reasonItemID,
		"output_index":    s.outputIndex - 1,
		"delta":           text,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	out = append(out, ResponsesStreamEvent{Event: "response.reasoning_summary_text.delta", Data: b})
	return out, nil
}

func (s *ResponsesStreamConverter) appendToolDelta(tc ToolCall) ([]ResponsesStreamEvent, error) {
	idx := tc.Index
	st, ok := s.activeTools[idx]
	if !ok {
		st = &responsesStreamToolState{outputIndex: s.outputIndex}
		s.activeTools[idx] = st
		s.outputIndex++
	}
	var out []ResponsesStreamEvent
	name := strings.TrimSpace(tc.Function.Name)
	if name != "" {
		st.name = name
	}
	if tc.ID != "" {
		st.callID = tc.ID
	}
	if tc.Function.Arguments != "" {
		st.argsBuf.WriteString(tc.Function.Arguments)
	}
	if !st.added && st.name != "" {
		st.added = true
		st.itemID = fmt.Sprintf("fc_%d", st.outputIndex)
		if st.callID == "" {
			st.callID = "call_" + uuid.New().String()
		}
		ns, decodedName, _ := DecodeNamespaceToolCall(s.toolMap, st.name, st.argsBuf.String())
		st.name = decodedName
		st.namespace = ns
		item := map[string]interface{}{
			"type":    "function_call",
			"id":      st.itemID,
			"status":  "in_progress",
			"call_id": st.callID,
			"name":    st.name,
		}
		if ns != "" {
			item["namespace"] = ns
		}
		added, err := s.outputItemAdded("function_call", st.itemID, item)
		if err != nil {
			return nil, err
		}
		out = append(out, added)
	}
	if tc.Function.Arguments != "" {
		data := map[string]interface{}{
			"type":            "response.function_call_arguments.delta",
			"sequence_number": s.nextSeq(),
			"item_id":         st.itemID,
			"output_index":    st.outputIndex,
			"delta":           tc.Function.Arguments,
		}
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		out = append(out, ResponsesStreamEvent{Event: "response.function_call_arguments.delta", Data: b})
	}
	return out, nil
}

func (s *ResponsesStreamConverter) outputItemAdded(itemType, itemID string, item map[string]interface{}) (ResponsesStreamEvent, error) {
	data := map[string]interface{}{
		"type":            "response.output_item.added",
		"sequence_number": s.nextSeq(),
		"output_index":    s.outputIndex,
		"item":            item,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return ResponsesStreamEvent{}, err
	}
	return ResponsesStreamEvent{Event: "response.output_item.added", Data: b}, nil
}

func (s *ResponsesStreamConverter) contentPartAdded(itemID string, contentIndex int) (ResponsesStreamEvent, error) {
	data := map[string]interface{}{
		"type":            "response.content_part.added",
		"sequence_number": s.nextSeq(),
		"item_id":         itemID,
		"output_index":    s.outputIndex - 1,
		"content_index":   contentIndex,
		"part":            map[string]interface{}{"type": "text", "text": ""},
	}
	b, err := json.Marshal(data)
	if err != nil {
		return ResponsesStreamEvent{}, err
	}
	return ResponsesStreamEvent{Event: "response.content_part.added", Data: b}, nil
}

func (s *ResponsesStreamConverter) textDelta(text string) (ResponsesStreamEvent, error) {
	data := map[string]interface{}{
		"type":            "response.output_text.delta",
		"sequence_number": s.nextSeq(),
		"item_id":         s.textItemID,
		"output_index":    s.outputIndex - 1,
		"content_index":   0,
		"delta":           text,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return ResponsesStreamEvent{}, err
	}
	return ResponsesStreamEvent{Event: "response.output_text.delta", Data: b}, nil
}

func (s *ResponsesStreamConverter) emitCompleted() ([]ResponsesStreamEvent, error) {
	if s.completed {
		return nil, nil
	}
	s.completed = true
	output := make([]interface{}, 0)
	if s.reasonBuf.Len() > 0 {
		output = append(output, map[string]interface{}{
			"type":   "reasoning",
			"status": "completed",
			"summary": []map[string]interface{}{
				{"type": "text", "text": s.reasonBuf.String()},
			},
		})
	}
	if s.textBuf.Len() > 0 {
		output = append(output, map[string]interface{}{
			"type":    "message",
			"status":  "completed",
			"role":    "assistant",
			"content": []map[string]interface{}{{"type": "text", "text": s.textBuf.String()}},
		})
	}
	for _, st := range s.activeTools {
		if st == nil || !st.added {
			continue
		}
		item := map[string]interface{}{
			"type":      "function_call",
			"status":    "completed",
			"call_id":   st.callID,
			"name":      st.name,
			"arguments": st.argsBuf.String(),
		}
		if st.namespace != "" {
			item["namespace"] = st.namespace
		}
		output = append(output, item)
	}
	status := "completed"
	data := map[string]interface{}{
		"type":            "response.completed",
		"sequence_number": s.nextSeq(),
		"response": map[string]interface{}{
			"id":          s.responseID,
			"object":      "response",
			"status":      status,
			"model":       s.model,
			"output":      output,
			"output_text": s.textBuf.String(),
			"usage": map[string]interface{}{
				"input_tokens":  s.inputTokens,
				"output_tokens": s.outputTokens,
				"total_tokens":  s.inputTokens + s.outputTokens,
			},
		},
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return []ResponsesStreamEvent{{Event: "response.completed", Data: b}}, nil
}

// ToProviderChunks maps internal events to providerapi chunks.
func ToProviderChunks(events []ResponsesStreamEvent) []providerapi.ResponsesStreamChunk {
	out := make([]providerapi.ResponsesStreamChunk, 0, len(events))
	for _, e := range events {
		out = append(out, providerapi.ResponsesStreamChunk{Event: e.Event, Data: e.Data})
	}
	return out
}
