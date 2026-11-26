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
	"strings"
	"testing"

	"yunion.io/x/jsonutils"
)

func parseDict(t *testing.T, s string) *jsonutils.JSONDict {
	t.Helper()
	obj, err := jsonutils.ParseString(s)
	if err != nil {
		t.Fatalf("parse %s: %v", s, err)
	}
	dict, ok := obj.(*jsonutils.JSONDict)
	if !ok {
		t.Fatalf("not a dict: %s", s)
	}
	return dict
}

func TestValidateEmbeddingsRequestAcceptsShapes(t *testing.T) {
	cases := []string{
		`{"model":"m","input":"hello"}`,
		`{"model":"m","input":["a","b"]}`,
		`{"model":"m","input":[1,2,3]}`,
		`{"model":"m","input":[[1,2],[3]]}`,
		`{"model":"m","input":"hi","encoding_format":"base64"}`,
		`{"model":"m","input":"hi","dimensions":256}`,
	}
	for _, c := range cases {
		if err := ValidateEmbeddingsRequest(parseDict(t, c)); err != nil {
			t.Errorf("%s: unexpected error %v", c, err)
		}
	}
}

func TestValidateEmbeddingsRequestRejects(t *testing.T) {
	cases := []string{
		`{"input":"hi"}`,
		`{"model":"m"}`,
		`{"model":"m","input":""}`,
		`{"model":"m","input":[]}`,
		`{"model":"m","input":{} }`,
		`{"model":"m","input":"hi","encoding_format":"hex"}`,
		`{"model":"m","input":"hi","dimensions":0}`,
	}
	for _, c := range cases {
		if err := ValidateEmbeddingsRequest(parseDict(t, c)); err == nil {
			t.Errorf("%s: expected error", c)
		}
	}
}

func TestValidateRerankRequest(t *testing.T) {
	ok := []string{
		`{"model":"m","query":"q","documents":["a","b"]}`,
		`{"model":"m","query":"q","documents":[{"text":"a"}]}`,
	}
	for _, c := range ok {
		if err := ValidateRerankRequest(parseDict(t, c)); err != nil {
			t.Errorf("%s: unexpected error %v", c, err)
		}
	}
	bad := []string{
		`{"query":"q","documents":["a"]}`,
		`{"model":"m","documents":["a"]}`,
		`{"model":"m","query":"q"}`,
		`{"model":"m","query":"q","documents":[]}`,
		`{"model":"m","query":"q","documents":123}`,
	}
	for _, c := range bad {
		if err := ValidateRerankRequest(parseDict(t, c)); err == nil {
			t.Errorf("%s: expected error", c)
		}
	}
}

func TestRerankURL(t *testing.T) {
	cases := map[string]string{
		"http://host:8000":          "http://host:8000/v1/rerank",
		"http://host:8000/":         "http://host:8000/v1/rerank",
		"http://host:8000/v1":       "http://host:8000/v1/rerank",
		"http://host:8000/v1/":      "http://host:8000/v1/rerank",
		"http://host:8000/v2":       "http://host:8000/v2/rerank",
		"http://host:8000/x/rerank": "http://host:8000/x/rerank",
	}
	for in, want := range cases {
		if got := RerankURL(in); got != want {
			t.Errorf("RerankURL(%q)=%q want %q", in, got, want)
		}
	}
}

func TestNormalizeEmbeddingsBodyPreservesVectors(t *testing.T) {
	in := `{"data":[{"embedding":"aGVsbG8="},{"embedding":[0.1,0.2]}],"model":"m"}`
	out, err := NormalizeEmbeddingsBody([]byte(in))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if !strings.Contains(string(out), `"object":"list"`) {
		t.Errorf("missing object=list: %s", out)
	}
	var root map[string]interface{}
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data := root["data"].([]interface{})
	first := data[0].(map[string]interface{})
	if first["embedding"] != "aGVsbG8=" {
		t.Errorf("base64 vector was rewritten: %v", first["embedding"])
	}
	if first["object"] != "embedding" {
		t.Errorf("missing data object: %v", first)
	}
	if first["index"].(float64) != 0 || data[1].(map[string]interface{})["index"].(float64) != 1 {
		t.Errorf("index not filled: %v", data)
	}
}

func TestNormalizeEmbeddingsBodyLeavesValidBodyIntact(t *testing.T) {
	in := `{"object":"list","model":"m","data":[{"object":"embedding","index":0,"embedding":[0.1]}],"usage":{"prompt_tokens":3}}`
	out, err := NormalizeEmbeddingsBody([]byte(in))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if string(out) != in {
		t.Errorf("valid body changed:\n in=%s\nout=%s", in, out)
	}
}

func TestNormalizeRerankBodyFillsIndex(t *testing.T) {
	in := `{"model":"m","results":[{"relevance_score":0.9},{"relevance_score":0.1}]}`
	out, err := NormalizeRerankBody([]byte(in))
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	var root struct {
		Results []struct {
			Index          *int    `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if root.Results[0].Index == nil || *root.Results[0].Index != 0 || *root.Results[1].Index != 1 {
		t.Errorf("index not filled: %s", out)
	}
	if root.Results[0].RelevanceScore != 0.9 {
		t.Errorf("score changed: %s", out)
	}
}
