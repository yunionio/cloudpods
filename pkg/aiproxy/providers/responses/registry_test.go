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

package responses

import (
	"errors"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

func TestGetAdapterMatrix(t *testing.T) {
	cases := []struct {
		key      string
		apiMode  string
		passthru bool
		err      bool
	}{
		{api.ProviderKeyOpenAI, api.ProviderAPIModeOpenAI, true, false},
		{api.ProviderKeyAzure, api.ProviderAPIModeOpenAI, true, false},
		{api.ProviderKeyVLLM, api.ProviderAPIModeOpenAI, false, false},
		{api.ProviderKeyAnthropic, api.ProviderAPIModeOpenAI, false, false},
		{api.ProviderKeyGemini, api.ProviderAPIModeOpenAI, false, false},
		{api.ProviderKeyDeepseek, api.ProviderAPIModeOpenAI, false, false},
		{api.ProviderKeyDeepseek, api.ProviderAPIModeAnthropic, false, false},
	}
	for _, c := range cases {
		ad, err := GetAdapter(c.key, c.apiMode)
		if err != nil {
			t.Fatalf("%s/%s: %v", c.key, c.apiMode, err)
		}
		if ad.ResponsesStreamPassthrough() != c.passthru {
			t.Fatalf("%s/%s passthrough = %v want %v", c.key, c.apiMode, ad.ResponsesStreamPassthrough(), c.passthru)
		}
		_, err = ad.BuildSubResourceRequest(&providerapi.ChatContext{
			BaseURL: "https://api.openai.com/v1",
			APIKey:  "k",
		}, "GET", "resp_1", "", nil)
		if c.passthru && err != nil {
			t.Fatalf("%s subresource: %v", c.key, err)
		}
		if !c.passthru {
			if !errors.Is(err, providerapi.ErrResponsesSubResourceNotSupported) {
				t.Fatalf("%s subresource err = %v", c.key, err)
			}
		}
	}
}

func TestDeepseekAnthropicModeUsesMessagesEndpoint(t *testing.T) {
	ad, err := GetAdapter(api.ProviderKeyDeepseek, api.ProviderAPIModeAnthropic)
	if err != nil {
		t.Fatalf("GetAdapter() error = %v", err)
	}
	body := jsonutils.NewDict()
	body.Set("model", jsonutils.NewString("deepseek-v4-flash"))
	body.Set("input", jsonutils.NewString("hi"))
	body.Set("max_output_tokens", jsonutils.NewInt(1024))
	req, err := ad.BuildUpstreamRequest(&providerapi.ChatContext{
		ProviderKey:   api.ProviderKeyDeepseek,
		BaseURL:       "https://api.deepseek.com/anthropic",
		APIKey:        "ds-key",
		UpstreamModel: "deepseek-v4-flash",
		APIMode:       api.ProviderAPIModeAnthropic,
	}, body, true)
	if err != nil {
		t.Fatalf("BuildUpstreamRequest() error = %v", err)
	}
	want := "https://api.deepseek.com/anthropic/v1/messages"
	if req.URL != want {
		t.Fatalf("URL = %q, want %q", req.URL, want)
	}
}

func TestIsNativeResponsesProvider(t *testing.T) {
	if !api.IsNativeResponsesProvider(api.ProviderKeyOpenAI) {
		t.Fatal("openai should be native")
	}
	if api.IsNativeResponsesProvider(api.ProviderKeyVLLM) {
		t.Fatal("vllm should not be native")
	}
}
