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
	"testing"

	"yunion.io/x/jsonutils"
)

func TestOpenAIEmbeddingsCompatBuild(t *testing.T) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("text-embedding-3-small"), "model")
	body.Add(jsonutils.NewString("hello"), "input")

	p := GetEmbeddings("openai")
	req, err := p.BuildEmbeddingsRequest(&ChatContext{
		BaseURL:       "https://api.openai.com",
		APIKey:        "sk-test",
		UpstreamModel: "text-embedding-3-small",
	}, body)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://api.openai.com/v1/embeddings" {
		t.Fatalf("unexpected url: %s", req.URL)
	}
}

func TestGeminiEmbeddingsBuildSingle(t *testing.T) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("text-embedding-004"), "model")
	body.Add(jsonutils.NewString("hello world"), "input")

	p := GetEmbeddings("gemini")
	req, err := p.BuildEmbeddingsRequest(&ChatContext{
		BaseURL:       "https://generativelanguage.googleapis.com/v1beta",
		APIKey:        "key",
		UpstreamModel: "text-embedding-004",
	}, body)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:embedContent" {
		t.Fatalf("unexpected url: %s", req.URL)
	}
}

func TestGeminiEmbeddingsNormalize(t *testing.T) {
	p := GetEmbeddings("gemini")
	out, err := p.NormalizeEmbeddingsResponse([]byte(`{"embedding":{"values":[0.1,0.2]}}`))
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if resp["object"] != "list" {
		t.Fatalf("unexpected object: %#v", resp["object"])
	}
}

func TestCohereEmbeddingsBuild(t *testing.T) {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewString("embed-english-v3.0"), "model")
	body.Add(jsonutils.NewArray(jsonutils.NewString("a"), jsonutils.NewString("b")), "input")

	p := GetEmbeddings("cohere")
	req, err := p.BuildEmbeddingsRequest(&ChatContext{
		BaseURL:       "https://api.cohere.ai",
		APIKey:        "key",
		UpstreamModel: "embed-english-v3.0",
	}, body)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL != "https://api.cohere.ai/v2/embed" {
		t.Fatalf("unexpected url: %s", req.URL)
	}
}

func TestCohereEmbeddingsNormalize(t *testing.T) {
	p := GetEmbeddings("cohere")
	out, err := p.NormalizeEmbeddingsResponse([]byte(`{"embeddings":{"float":[[0.1],[0.2]]}}`))
	if err != nil {
		t.Fatal(err)
	}
	var resp struct {
		Data []struct {
			Index int `json:"index"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 embeddings, got %d", len(resp.Data))
	}
}

func TestAnthropicEmbeddingsUnsupported(t *testing.T) {
	p := GetEmbeddings("anthropic")
	_, err := p.BuildEmbeddingsRequest(&ChatContext{ProviderKey: "anthropic"}, jsonutils.NewDict())
	if err == nil {
		t.Fatal("expected error for anthropic embeddings")
	}
}
