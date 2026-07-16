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
	"sort"
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
	requestModel      string
	toolMap           CodexToolMap
	created           bool
	completed         bool
	seq               int64
	responseID        string
	model             string
	outputIndex       int
	textItemID        string
	reasonItemID      string
	textOutputIndex   int
	reasonOutputIndex int
	textStarted       bool
	reasonStarted     bool
	textDone          bool
	reasonDone        bool
	textBuf           strings.Builder
	reasonBuf         strings.Builder
	inputTokens       int
	outputTokens      int
	activeTools       map[int]*responsesStreamToolState
}

type responsesStreamToolState struct {
	outputIndex int
	itemID      string
	callID      string
	name        string
	namespace   string
	argsBuf     strings.Builder
	added       bool
	finalized   bool
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
		s.textOutputIndex = s.outputIndex
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
		s.reasonOutputIndex = s.outputIndex
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
		added, err := s.outputItemAddedAt(st.outputIndex, "function_call", st.itemID, item)
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
	return s.outputItemAddedAt(s.outputIndex, itemType, itemID, item)
}

func (s *ResponsesStreamConverter) outputItemAddedAt(outputIndex int, itemType, itemID string, item map[string]interface{}) (ResponsesStreamEvent, error) {
	data := map[string]interface{}{
		"type":            "response.output_item.added",
		"sequence_number": s.nextSeq(),
		"output_index":    outputIndex,
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

func (s *ResponsesStreamConverter) emitFinalizeEvents() ([]ResponsesStreamEvent, error) {
	var out []ResponsesStreamEvent
	if s.reasonStarted && !s.reasonDone {
		events, err := s.emitReasoningDone()
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	if s.textStarted && !s.textDone {
		events, err := s.emitTextDone()
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	toolIndexes := make([]int, 0, len(s.activeTools))
	for idx := range s.activeTools {
		toolIndexes = append(toolIndexes, idx)
	}
	sort.Ints(toolIndexes)
	for _, idx := range toolIndexes {
		st := s.activeTools[idx]
		if st == nil || !st.added || st.finalized {
			continue
		}
		events, err := s.emitToolDone(st)
		if err != nil {
			return nil, err
		}
		out = append(out, events...)
	}
	return out, nil
}

func (s *ResponsesStreamConverter) emitReasoningDone() ([]ResponsesStreamEvent, error) {
	if !s.reasonStarted || s.reasonDone {
		return nil, nil
	}
	s.reasonDone = true
	var out []ResponsesStreamEvent
	partDone := map[string]interface{}{
		"type":            "response.reasoning_summary_part.done",
		"sequence_number": s.nextSeq(),
		"item_id":         s.reasonItemID,
		"output_index":    s.reasonOutputIndex,
		"summary_index":   0,
	}
	b, err := json.Marshal(partDone)
	if err != nil {
		return nil, err
	}
	out = append(out, ResponsesStreamEvent{Event: "response.reasoning_summary_part.done", Data: b})

	item := map[string]interface{}{
		"type":   "reasoning",
		"id":     s.reasonItemID,
		"status": "completed",
		"summary": []map[string]interface{}{
			{"type": "text", "text": s.reasonBuf.String()},
		},
	}
	itemDone, err := s.outputItemDoneAt(s.reasonOutputIndex, item)
	if err != nil {
		return nil, err
	}
	out = append(out, itemDone)
	return out, nil
}

func (s *ResponsesStreamConverter) emitTextDone() ([]ResponsesStreamEvent, error) {
	if !s.textStarted || s.textDone {
		return nil, nil
	}
	s.textDone = true
	text := s.textBuf.String()
	var out []ResponsesStreamEvent
	textDone := map[string]interface{}{
		"type":            "response.output_text.done",
		"sequence_number": s.nextSeq(),
		"item_id":         s.textItemID,
		"output_index":    s.textOutputIndex,
		"content_index":   0,
		"text":            text,
	}
	b, err := json.Marshal(textDone)
	if err != nil {
		return nil, err
	}
	out = append(out, ResponsesStreamEvent{Event: "response.output_text.done", Data: b})

	partDone := map[string]interface{}{
		"type":            "response.content_part.done",
		"sequence_number": s.nextSeq(),
		"item_id":         s.textItemID,
		"output_index":    s.textOutputIndex,
		"content_index":   0,
		"part":            map[string]interface{}{"type": "output_text", "text": text},
	}
	b, err = json.Marshal(partDone)
	if err != nil {
		return nil, err
	}
	out = append(out, ResponsesStreamEvent{Event: "response.content_part.done", Data: b})

	item := map[string]interface{}{
		"type":   "message",
		"id":     s.textItemID,
		"status": "completed",
		"role":   "assistant",
		"content": []map[string]interface{}{
			{"type": "output_text", "text": text},
		},
	}
	itemDone, err := s.outputItemDoneAt(s.textOutputIndex, item)
	if err != nil {
		return nil, err
	}
	out = append(out, itemDone)
	return out, nil
}

func (s *ResponsesStreamConverter) emitToolDone(st *responsesStreamToolState) ([]ResponsesStreamEvent, error) {
	if st == nil || !st.added || st.finalized {
		return nil, nil
	}
	st.finalized = true
	args := st.argsBuf.String()
	var out []ResponsesStreamEvent
	argsDone := map[string]interface{}{
		"type":            "response.function_call_arguments.done",
		"sequence_number": s.nextSeq(),
		"item_id":         st.itemID,
		"output_index":    st.outputIndex,
		"arguments":       args,
	}
	b, err := json.Marshal(argsDone)
	if err != nil {
		return nil, err
	}
	out = append(out, ResponsesStreamEvent{Event: "response.function_call_arguments.done", Data: b})

	item := map[string]interface{}{
		"type":      "function_call",
		"id":        st.itemID,
		"status":    "completed",
		"call_id":   st.callID,
		"name":      st.name,
		"arguments": args,
	}
	if st.namespace != "" {
		item["namespace"] = st.namespace
	}
	itemDone, err := s.outputItemDoneAt(st.outputIndex, item)
	if err != nil {
		return nil, err
	}
	out = append(out, itemDone)
	return out, nil
}

func (s *ResponsesStreamConverter) outputItemDoneAt(outputIndex int, item map[string]interface{}) (ResponsesStreamEvent, error) {
	data := map[string]interface{}{
		"type":            "response.output_item.done",
		"sequence_number": s.nextSeq(),
		"output_index":    outputIndex,
		"item":            item,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return ResponsesStreamEvent{}, err
	}
	return ResponsesStreamEvent{Event: "response.output_item.done", Data: b}, nil
}

func (s *ResponsesStreamConverter) buildCompletedOutput() []interface{} {
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
	toolIndexes := make([]int, 0, len(s.activeTools))
	for idx := range s.activeTools {
		toolIndexes = append(toolIndexes, idx)
	}
	sort.Ints(toolIndexes)
	for _, idx := range toolIndexes {
		st := s.activeTools[idx]
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
	return output
}

func (s *ResponsesStreamConverter) emitCompleted() ([]ResponsesStreamEvent, error) {
	if s.completed {
		return nil, nil
	}
	finalizeEvents, err := s.emitFinalizeEvents()
	if err != nil {
		return nil, err
	}
	s.completed = true
	output := s.buildCompletedOutput()
	data := map[string]interface{}{
		"type":            "response.completed",
		"sequence_number": s.nextSeq(),
		"response": map[string]interface{}{
			"id":          s.responseID,
			"object":      "response",
			"status":      "completed",
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
	out := append(finalizeEvents, ResponsesStreamEvent{Event: "response.completed", Data: b})
	return out, nil
}

// ToProviderChunks maps internal events to providerapi chunks.
func ToProviderChunks(events []ResponsesStreamEvent) []providerapi.ResponsesStreamChunk {
	out := make([]providerapi.ResponsesStreamChunk, 0, len(events))
	for _, e := range events {
		out = append(out, providerapi.ResponsesStreamChunk{Event: e.Event, Data: e.Data})
	}
	return out
}

type syntheticChatChunkDelta struct {
	Content          *string    `json:"content,omitempty"`
	ReasoningContent *string    `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

type syntheticChatChunkChoice struct {
	Delta        syntheticChatChunkDelta `json:"delta,omitempty"`
	FinishReason string                  `json:"finish_reason,omitempty"`
}

type syntheticChatChunkPayload struct {
	ID      string                     `json:"id,omitempty"`
	Model   string                     `json:"model,omitempty"`
	Choices []syntheticChatChunkChoice `json:"choices,omitempty"`
	Usage   *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// NonStreamChatCompletionToStreamPayloads converts a chat.completion JSON body into
// synthetic chat.completion.chunk payloads for ResponsesStreamConverter.
func NonStreamChatCompletionToStreamPayloads(body []byte) ([][]byte, error) {
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content          json.RawMessage `json:"content"`
				ReasoningContent string          `json:"reasoning_content"`
				ToolCalls        []ToolCall      `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("invalid chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty chat completion choices")
	}
	choice := resp.Choices[0]
	payloads := make([][]byte, 0, 4)

	appendChunk := func(chunk syntheticChatChunkPayload) error {
		if chunk.ID == "" {
			chunk.ID = resp.ID
		}
		if chunk.Model == "" {
			chunk.Model = resp.Model
		}
		b, err := json.Marshal(chunk)
		if err != nil {
			return err
		}
		payloads = append(payloads, b)
		return nil
	}

	if err := appendChunk(syntheticChatChunkPayload{
		ID:      resp.ID,
		Model:   resp.Model,
		Choices: []syntheticChatChunkChoice{{}},
	}); err != nil {
		return nil, err
	}

	if rc := strings.TrimSpace(choice.Message.ReasoningContent); rc != "" {
		reason := rc
		if err := appendChunk(syntheticChatChunkPayload{
			Choices: []syntheticChatChunkChoice{{
				Delta: syntheticChatChunkDelta{ReasoningContent: &reason},
			}},
		}); err != nil {
			return nil, err
		}
	}

	if text := strings.TrimSpace(MessageTextContent(choice.Message.Content)); text != "" {
		content := text
		if err := appendChunk(syntheticChatChunkPayload{
			Choices: []syntheticChatChunkChoice{{
				Delta: syntheticChatChunkDelta{Content: &content},
			}},
		}); err != nil {
			return nil, err
		}
	}

	for i, tc := range choice.Message.ToolCalls {
		call := tc
		if call.Index == 0 && i > 0 {
			call.Index = i
		}
		if call.Type == "" {
			call.Type = "function"
		}
		if err := appendChunk(syntheticChatChunkPayload{
			Choices: []syntheticChatChunkChoice{{
				Delta: syntheticChatChunkDelta{ToolCalls: []ToolCall{call}},
			}},
		}); err != nil {
			return nil, err
		}
	}

	if resp.Usage != nil {
		if err := appendChunk(syntheticChatChunkPayload{Usage: resp.Usage}); err != nil {
			return nil, err
		}
	}

	if choice.FinishReason == "length" {
		if err := appendChunk(syntheticChatChunkPayload{
			Choices: []syntheticChatChunkChoice{{
				FinishReason: "length",
			}},
		}); err != nil {
			return nil, err
		}
	}

	return payloads, nil
}
