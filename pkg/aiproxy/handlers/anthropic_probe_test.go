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

package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicBaseProbeHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, anthropicBasePrefix+"/", nil)
	rec := httptest.NewRecorder()
	anthropicBaseProbeHandler(context.Background(), rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
}

func TestAnthropicMessagesHeadHandlerNoAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, anthropicCompatAPIPrefix+"/messages", nil)
	rec := httptest.NewRecorder()
	anthropicMessagesHeadHandler(context.Background(), rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAnthropicMessagesHeadHandlerWithAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, anthropicCompatAPIPrefix+"/messages", nil)
	req.Header.Set("Authorization", "Bearer sk-test-vk")
	rec := httptest.NewRecorder()
	anthropicMessagesHeadHandler(context.Background(), rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAnthropicMessagesHeadHandlerWithXAiVirtualKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, anthropicCompatAPIPrefix+"/messages", nil)
	req.Header.Set(headerAiVirtualKey, "sk-test-vk")
	rec := httptest.NewRecorder()
	anthropicMessagesHeadHandler(context.Background(), rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusNoContent)
	}
}
