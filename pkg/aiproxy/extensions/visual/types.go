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

package visual

import "context"

// ImageInput carries image data as URL, base64 data, or data URL.
type ImageInput struct {
	URL      string `json:"url,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// ConversationTurn is one visual clarification history entry.
type ConversationTurn struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// AnalysisRequest bundles the analysis prompt with images for the vision model.
type AnalysisRequest struct {
	Tool   string
	Prompt string
	Images []ImageInput
}

// VisionClient analyzes images with a short prompt.
type VisionClient interface {
	Analyze(context.Context, AnalysisRequest) (string, error)
}
