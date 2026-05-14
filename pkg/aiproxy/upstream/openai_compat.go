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

package upstream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Request is one upstream chat call (OpenAI-compatible by default, or provider-native when URL/headers are set).
type Request struct {
	BaseURL string
	URL     string
	APIKey  string
	Headers map[string]string
	Body    []byte
}

// Response is a non-streaming upstream response body.
type Response struct {
	StatusCode int
	Body       []byte
}

// StreamChunk is one SSE event payload (bytes after "data: ").
type StreamChunk struct {
	Data []byte
	Done bool
}

// RawSSEEvent is one parsed server-sent event line group from an upstream.
type RawSSEEvent struct {
	Event string
	Data  []byte
}

// Error carries upstream HTTP status and optional JSON error body.
type Error struct {
	StatusCode int
	Message    string
	Body       []byte
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if len(e.Body) > 0 {
		return string(e.Body)
	}
	return fmt.Sprintf("upstream HTTP %d", e.StatusCode)
}

// ChatCompletionsURL builds the chat completions endpoint from a provider base URL.
// BaseURL is the origin + optional path prefix (e.g. https://dashscope.aliyuncs.com/compatible-mode).
func ChatCompletionsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

func sharedHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		httpClient = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        256,
				MaxIdleConnsPerHost: 64,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	})
	return httpClient
}

func requestURL(req *Request) string {
	if req == nil {
		return ""
	}
	if u := strings.TrimSpace(req.URL); u != "" {
		return u
	}
	return ChatCompletionsURL(req.BaseURL)
}

func newUpstreamRequest(ctx context.Context, req *Request) (*http.Request, error) {
	if req == nil {
		return nil, fmt.Errorf("nil upstream request")
	}
	url := requestURL(req)
	apiKey := strings.TrimSpace(req.APIKey)
	if url == "" {
		return nil, fmt.Errorf("empty upstream URL")
	}
	if apiKey == "" && len(req.Headers) == 0 {
		return nil, fmt.Errorf("empty API key")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(req.Body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	if apiKey != "" && httpReq.Header.Get("Authorization") == "" && httpReq.Header.Get("x-api-key") == "" && httpReq.Header.Get("api-key") == "" && httpReq.Header.Get("x-goog-api-key") == "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	return httpReq, nil
}

func readResponseBody(resp *http.Response, maxBytes int64) ([]byte, error) {
	defer resp.Body.Close()
	if maxBytes <= 0 {
		maxBytes = 32 << 20
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxBytes))
}

func errorFromResponse(resp *http.Response, body []byte) *Error {
	status := resp.StatusCode
	msg := strings.TrimSpace(resp.Status)
	if len(body) > 0 {
		var wrap struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &wrap); err == nil && wrap.Error != nil && wrap.Error.Message != "" {
			msg = wrap.Error.Message
		}
	}
	return &Error{StatusCode: status, Message: msg, Body: body}
}

// ChatCompletion performs a non-streaming chat completions request.
func ChatCompletion(ctx context.Context, req *Request) (*Response, *Error) {
	httpReq, err := newUpstreamRequest(ctx, req)
	if err != nil {
		return nil, &Error{StatusCode: http.StatusBadGateway, Message: err.Error()}
	}
	resp, err := sharedHTTPClient().Do(httpReq)
	if err != nil {
		return nil, &Error{StatusCode: http.StatusBadGateway, Message: err.Error()}
	}
	body, err := readResponseBody(resp, 32<<20)
	if err != nil {
		return nil, &Error{StatusCode: http.StatusBadGateway, Message: err.Error()}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errorFromResponse(resp, body)
	}
	return &Response{StatusCode: resp.StatusCode, Body: body}, nil
}

// ChatCompletionStream opens a streaming chat completions request and returns SSE data chunks.
func ChatCompletionStream(ctx context.Context, req *Request) (<-chan StreamChunk, *Error) {
	_, resp, uerr := openChatCompletionStream(ctx, req)
	if uerr != nil {
		return nil, uerr
	}
	out := make(chan StreamChunk, 16)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		for evt := range readSSE(resp.Body) {
			if evt.Done {
				out <- StreamChunk{Done: true}
				return
			}
			out <- StreamChunk{Data: evt.Data}
		}
		if ctx.Err() != nil {
			return
		}
	}()
	return out, nil
}

// ChatCompletionStreamRaw opens a streaming request and returns raw SSE events (event + data lines).
func ChatCompletionStreamRaw(ctx context.Context, req *Request) (<-chan RawSSEEvent, *Error) {
	_, resp, uerr := openChatCompletionStream(ctx, req)
	if uerr != nil {
		return nil, uerr
	}
	out := make(chan RawSSEEvent, 16)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		for evt := range readSSE(resp.Body) {
			if evt.Done {
				return
			}
			out <- RawSSEEvent{Event: evt.Event, Data: evt.Data}
		}
	}()
	return out, nil
}

type sseFrame struct {
	Event string
	Data  []byte
	Done  bool
}

func openChatCompletionStream(ctx context.Context, req *Request) (*http.Request, *http.Response, *Error) {
	httpReq, err := newUpstreamRequest(ctx, req)
	if err != nil {
		return nil, nil, &Error{StatusCode: http.StatusBadGateway, Message: err.Error()}
	}
	httpReq.Header.Set("Accept", "text/event-stream")
	resp, err := sharedHTTPClient().Do(httpReq)
	if err != nil {
		return nil, nil, &Error{StatusCode: http.StatusBadGateway, Message: err.Error()}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := readResponseBody(resp, 1<<20)
		return nil, nil, errorFromResponse(resp, body)
	}
	return httpReq, resp, nil
}

func readSSE(r io.Reader) <-chan sseFrame {
	out := make(chan sseFrame, 16)
	go func() {
		defer close(out)
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		var pendingEvent string
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "event:") {
				pendingEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "[DONE]" {
				out <- sseFrame{Done: true}
				return
			}
			out <- sseFrame{Event: pendingEvent, Data: []byte(payload)}
			pendingEvent = ""
		}
	}()
	return out
}
