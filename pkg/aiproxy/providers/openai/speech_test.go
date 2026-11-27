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
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
)

func TestSpeechURL(t *testing.T) {
	cases := []struct {
		base string
		want string
	}{
		{"https://api.example.com", "https://api.example.com/v1/audio/speech"},
		{"https://api.example.com/v1", "https://api.example.com/v1/audio/speech"},
		{"https://api.example.com/v1/audio/speech", "https://api.example.com/v1/audio/speech"},
		{"https://api.example.com/compatible-mode/v1", "https://api.example.com/compatible-mode/v1/audio/speech"},
	}
	for _, c := range cases {
		if got := SpeechURL(c.base); got != c.want {
			t.Fatalf("SpeechURL(%q)=%q want %q", c.base, got, c.want)
		}
	}
}

func TestValidateSpeechRequest(t *testing.T) {
	ok := jsonutils.Marshal(map[string]interface{}{
		"model":           "tts-1",
		"input":           "hello",
		"voice":           "alloy",
		"response_format": "mp3",
		"task_type":       "CustomVoice",
	}).(*jsonutils.JSONDict)
	if err := ValidateSpeechRequest(ok); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateSpeechRequest(jsonutils.NewDict()); err == nil {
		t.Fatal("expected missing model error")
	}
	bad := jsonutils.Marshal(map[string]interface{}{"model": "tts-1"}).(*jsonutils.JSONDict)
	if err := ValidateSpeechRequest(bad); err == nil {
		t.Fatal("expected missing input error")
	}
}

func TestSpeechContentTypeFromFormat(t *testing.T) {
	if got := SpeechContentTypeFromFormat(""); got != "audio/mpeg" {
		t.Fatalf("empty format: got %q", got)
	}
	if got := SpeechContentTypeFromFormat("wav"); got != "audio/wav" {
		t.Fatalf("wav: got %q", got)
	}
	if got := SpeechContentTypeFromFormat("pcm"); got != "audio/pcm" {
		t.Fatalf("pcm: got %q", got)
	}
}

func TestSpeechCompatBuildRequest(t *testing.T) {
	p := NewSpeechCompat()
	body := jsonutils.Marshal(map[string]interface{}{
		"model":     "client-model",
		"input":     "hi",
		"voice":     "Vivian",
		"task_type": "CustomVoice",
		"language":  "Auto",
	}).(*jsonutils.JSONDict)
	req, err := p.BuildSpeechRequest(&providerapi.ChatContext{
		BaseURL:       "https://worker.example/v1",
		APIKey:        "sk-test",
		UpstreamModel: "upstream-tts",
	}, body)
	if err != nil {
		t.Fatalf("BuildSpeechRequest: %v", err)
	}
	if req.URL != "https://worker.example/v1/audio/speech" {
		t.Fatalf("URL=%q", req.URL)
	}
	parsed, err := jsonutils.Parse(req.Body)
	if err != nil {
		t.Fatalf("parse body: %v", err)
	}
	dict := parsed.(*jsonutils.JSONDict)
	if m, _ := dict.GetString("model"); m != "upstream-tts" {
		t.Fatalf("model=%q want upstream-tts", m)
	}
	if v, _ := dict.GetString("task_type"); v != "CustomVoice" {
		t.Fatalf("task_type stripped: %q", v)
	}
	if v, _ := dict.GetString("language"); v != "Auto" {
		t.Fatalf("language stripped: %q", v)
	}
}
