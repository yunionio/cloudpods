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

// Package providerapi defines shared types and interfaces for aiproxy provider adapters.
package providerapi // import "yunion.io/x/onecloud/pkg/aiproxy/providerapi"

import (
	"yunion.io/x/jsonutils"
)

// ChatContext holds resolved upstream connectivity for one proxied request.
type ChatContext struct {
	ProviderKey   string
	BaseURL       string
	APIKey        string
	UpstreamModel string
}

// HTTPRequest is the wire-format call sent to an upstream provider.
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
}

// StreamChunk is one normalized OpenAI chat.completion.chunk SSE payload (JSON only, no "data:" prefix).
type StreamChunk struct {
	Data []byte
	Done bool
}

// StreamState carries per-stream conversion state for providers that emit non-OpenAI SSE.
type StreamState struct {
	Model           string
	ResponseID      string
	ChunkIndex      int
	TextStarted     bool
	ToolIndex       int
	ToolID          string
	ToolName        string
	ToolArgsPending string
	InToolBlock     bool
}

// Provider converts OpenAI chat/completions to a provider-native HTTP call and
// normalizes responses back to OpenAI format.
type Provider interface {
	Key() string
	BuildUpstreamRequest(ctx *ChatContext, body *jsonutils.JSONDict, stream bool) (*HTTPRequest, error)
	NormalizeResponse(body []byte) ([]byte, error)
	OpenAIStreamPassthrough() bool
	ConvertStreamEvent(eventType string, payload []byte, state *StreamState) ([]StreamChunk, error)
}

// EmbeddingsProvider converts OpenAI /v1/embeddings requests to provider-native APIs.
type EmbeddingsProvider interface {
	BuildEmbeddingsRequest(ctx *ChatContext, body *jsonutils.JSONDict) (*HTTPRequest, error)
	NormalizeEmbeddingsResponse(body []byte) ([]byte, error)
}

// ImagesProvider converts OpenAI /v1/images/generations requests to provider-native APIs.
type ImagesProvider interface {
	BuildImagesGenerationsRequest(ctx *ChatContext, body *jsonutils.JSONDict) (*HTTPRequest, error)
	NormalizeImagesGenerationsResponse(body []byte) ([]byte, error)
}

// CompletionsProvider converts OpenAI /v1/completions requests to provider-native APIs.
type CompletionsProvider interface {
	BuildCompletionsRequest(ctx *ChatContext, body *jsonutils.JSONDict, stream bool) (*HTTPRequest, error)
	NormalizeCompletionsResponse(body []byte) ([]byte, error)
	OpenAICompletionsStreamPassthrough() bool
}
