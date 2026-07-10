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
	"testing"

	"yunion.io/x/pkg/appctx"
)

func TestTruncateLogBytes(t *testing.T) {
	if got := truncateLogBytes(nil, 10); got != "" {
		t.Fatalf("empty input: got %q", got)
	}
	raw := []byte("hello")
	if got := truncateLogBytes(raw, 10); got != "hello" {
		t.Fatalf("short input: got %q", got)
	}
	long := []byte("0123456789abcdef")
	got := truncateLogBytes(long, 10)
	if got != "0123456789...(truncated, total=16)" {
		t.Fatalf("truncated: got %q", got)
	}
}

func TestMaskSecret(t *testing.T) {
	if got := maskSecret(""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
	if got := maskSecret("short"); got != "***" {
		t.Fatalf("short secret: got %q", got)
	}
	if got := maskSecret("abcdefghijklmnop"); got != "abcd...mnop" {
		t.Fatalf("long secret: got %q", got)
	}
}

func TestNewProxyDebugSessionUsesContextRequestID(t *testing.T) {
	ctx := (&appctx.AppContextData{RequestId: "req-123"}).GetContext()
	dbg := NewProxyDebugSession(ctx, "openai-chat")
	if dbg.ReqID != "req-123" {
		t.Fatalf("ReqID = %q, want req-123", dbg.ReqID)
	}
	if dbg.Protocol != "openai-chat" {
		t.Fatalf("Protocol = %q, want openai-chat", dbg.Protocol)
	}
	if got := dbg.prefix(); got != "aiproxy openai-chat [req-123]" {
		t.Fatalf("prefix = %q", got)
	}
}

func TestNewProxyDebugSessionFallbackID(t *testing.T) {
	dbg := NewProxyDebugSession(context.Background(), "anthropic-messages")
	if dbg.ReqID == "" {
		t.Fatal("expected fallback req id")
	}
}

func TestProxyDebugSessionNilSafe(t *testing.T) {
	var dbg *ProxyDebugSession
	dbg.ClientRequest(nil, nil, nil, false)
	dbg.ClientRequestNoBody(nil)
	dbg.UpstreamRequest(nil)
	dbg.UpstreamResponse(nil)
	dbg.ClientResponse(nil)
	dbg.UpstreamStreamChunk(1, nil)
	dbg.ClientStreamData(1, nil)
	dbg.ClientStreamSSE(1, "", nil)
	dbg.ClientStreamEvents(1, nil)
	dbg.RoutingResolved(nil, nil)
	dbg.Error("noop")
}
