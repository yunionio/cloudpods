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

package providers

import (
	"encoding/json"
	"strings"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestOpenAIImagesCompatBuild(t *testing.T) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("dall-e-3"), "model")
	body.Add(jsonutils.NewString("a red cat"), "prompt")

	p := GetImages("openai")
	req, err := p.BuildImagesGenerationsRequest(&ChatContext{
		BaseURL:       "https://api.openai.com",
		APIKey:        "sk-test",
		UpstreamModel: "dall-e-3",
	}, body)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://api.openai.com/v1/images/generations" {
		t.Fatalf("unexpected url: %s", req.URL)
	}
}

func TestGeminiImagesBuild(t *testing.T) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("imagen-3.0-generate-002"), "model")
	body.Add(jsonutils.NewString("sunset over mountains"), "prompt")
	body.Add(jsonutils.NewString("1792x1024"), "size")
	body.Add(jsonutils.NewInt(2), "n")

	p := GetImages("gemini")
	req, err := p.BuildImagesGenerationsRequest(&ChatContext{
		BaseURL:       "https://generativelanguage.googleapis.com/v1beta",
		APIKey:        "key",
		UpstreamModel: "imagen-3.0-generate-002",
	}, body)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://generativelanguage.googleapis.com/v1beta/models/imagen-3.0-generate-002:predict" {
		t.Fatalf("unexpected url: %s", req.URL)
	}
	var wire struct {
		Parameters struct {
			SampleCount int    `json:"sampleCount"`
			AspectRatio string `json:"aspectRatio"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(req.Body, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Parameters.SampleCount != 2 || wire.Parameters.AspectRatio != "16:9" {
		t.Fatalf("unexpected parameters: %+v", wire.Parameters)
	}
}

func TestGeminiImagesNormalize(t *testing.T) {
	p := GetImages("gemini")
	out, err := p.NormalizeImagesGenerationsResponse([]byte(`{"predictions":[{"bytesBase64Encoded":"abc123"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	var resp struct {
		Data []struct {
			B64 string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].B64 != "abc123" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAzureImagesBuild(t *testing.T) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("dall-e-3"), "model")
	body.Add(jsonutils.NewString("test"), "prompt")

	p := GetImages("azure")
	req, err := p.BuildImagesGenerationsRequest(&ChatContext{
		BaseURL:       "https://example.openai.azure.com",
		APIKey:        "key",
		UpstreamModel: "dall-e-3",
	}, body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(req.URL, "images/generations") {
		t.Fatalf("unexpected url: %s", req.URL)
	}
}

func TestAnthropicImagesUnsupported(t *testing.T) {
	p := GetImages("anthropic")
	_, err := p.BuildImagesGenerationsRequest(&ChatContext{ProviderKey: "anthropic"}, jsonutils.NewDict())
	if err == nil {
		t.Fatal("expected error for anthropic images")
	}
}
