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
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/aiproxy/providerapi"
)

// SpeechCompat forwards OpenAI audio/speech requests unchanged (transparent passthrough).
type SpeechCompat struct{}

// NewSpeechCompat returns a new OpenAI-compatible speech adapter.
func NewSpeechCompat() *SpeechCompat {
	return &SpeechCompat{}
}

func (p *SpeechCompat) BuildSpeechRequest(ctx *providerapi.ChatContext, body *jsonutils.JSONDict) (*providerapi.HTTPRequest, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil chat context")
	}
	dup := CloneBodyWithModel(body, ctx.UpstreamModel)
	return &providerapi.HTTPRequest{
		Method:  http.MethodPost,
		URL:     SpeechURL(ctx.BaseURL),
		Headers: BearerAuthHeaders(ctx.APIKey),
		Body:    []byte(dup.String()),
	}, nil
}

// ValidateSpeechRequest checks a minimal OpenAI /v1/audio/speech body.
// Extra fields (task_type, language, instructions, ref_audio, …) are allowed
// and forwarded unchanged to the upstream.
func ValidateSpeechRequest(body *jsonutils.JSONDict) error {
	if body == nil {
		return fmt.Errorf("request body must be a JSON object")
	}
	if s, _ := body.GetString("model"); strings.TrimSpace(s) == "" {
		return fmt.Errorf("missing model")
	}
	if s, _ := body.GetString("input"); strings.TrimSpace(s) == "" {
		return fmt.Errorf("missing input")
	}
	return nil
}

// SpeechContentTypeFromFormat maps response_format to a Content-Type when the
// upstream omits one. Empty format defaults to audio/mpeg (mp3).
func SpeechContentTypeFromFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "mp3":
		return "audio/mpeg"
	case "opus":
		return "audio/opus"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	case "wav":
		return "audio/wav"
	case "pcm":
		return "audio/pcm"
	default:
		return "application/octet-stream"
	}
}
