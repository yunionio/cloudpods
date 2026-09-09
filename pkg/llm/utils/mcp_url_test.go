// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or authorized to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestValidateMCPServerURL(t *testing.T) {
	prev := options.Options.MCPServerURL
	options.Options.MCPServerURL = "http://default-mcp-server:30876"
	defer func() { options.Options.MCPServerURL = prev }()

	if err := ValidateMCPServerURL("http://default-mcp-server:30876"); err != nil {
		t.Fatalf("configured url: %v", err)
	}
	if err := ValidateMCPServerURL("http://default-mcp-server:30876/"); err != nil {
		t.Fatalf("configured url with slash: %v", err)
	}
	if err := ValidateMCPServerURL("https://8.8.8.8"); err != nil {
		t.Fatalf("public ip: %v", err)
	}

	for _, raw := range []string{
		"",
		"ftp://example.com",
		"http://127.0.0.1",
		"http://10.1.2.3",
		"http://169.254.169.254/",
		"http://[::1]/",
		"http://localhost",
		"http://user:pass@8.8.8.8",
		"http://8.8.8.8/#/sse",
		"http://8.8.8.8#",
	} {
		if err := ValidateMCPServerURL(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func TestIsTrustedMCPServerURL(t *testing.T) {
	prev := options.Options.MCPServerURL
	options.Options.MCPServerURL = "http://default-mcp-server:30876"
	defer func() { options.Options.MCPServerURL = prev }()

	if !IsTrustedMCPServerURL("http://default-mcp-server:30876") {
		t.Fatal("same url should be trusted")
	}
	if !IsTrustedMCPServerURL("http://DEFAULT-mcp-server:30876/") {
		t.Fatal("same origin should be trusted")
	}
	if IsTrustedMCPServerURL("https://8.8.8.8") {
		t.Fatal("other host should not be trusted")
	}
}

func TestNewMCPClientOmitsCredForUntrusted(t *testing.T) {
	prev := options.Options.MCPServerURL
	options.Options.MCPServerURL = "http://default-mcp-server:30876"
	defer func() { options.Options.MCPServerURL = prev }()

	tok := &mcclient.SSimpleToken{Token: "secret-token"}
	c := NewMCPClient("https://8.8.8.8", time.Second, tok)
	if c.userCred != nil {
		t.Fatal("untrusted url should not keep caller cred")
	}
	c = NewMCPClient("http://default-mcp-server:30876", time.Second, tok)
	if c.userCred == nil {
		t.Fatal("configured url should keep caller cred")
	}
}

func TestJoinMCPEndpoint(t *testing.T) {
	got, err := joinMCPEndpoint("http://default-mcp-server:30876", "/sse")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://default-mcp-server:30876/sse" {
		t.Fatalf("got %q", got)
	}
	got, err = joinMCPEndpoint("http://default-mcp-server:30876", "/message?sessionId=abc")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://default-mcp-server:30876/message?sessionId=abc" {
		t.Fatalf("got %q", got)
	}
	if _, err := joinMCPEndpoint("http://default-mcp-server:30876", "http://8.8.8.8/message"); err == nil {
		t.Fatal("absolute other host should fail")
	}
	if _, err := joinMCPEndpoint("http://default-mcp-server:30876", "//8.8.8.8/message"); err == nil {
		t.Fatal("scheme-relative should fail")
	}
}
