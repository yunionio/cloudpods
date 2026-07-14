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

import (
	"strings"

	"yunion.io/x/onecloud/pkg/aiproxy/models"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
)

// RuntimeConfig is the resolved visual extension settings for one request.
type RuntimeConfig struct {
	Enabled   bool
	MaxRounds int
	MaxTokens int
}

// RuntimeConfigFromModel extracts visual runtime settings from ai_model config.
func RuntimeConfigFromModel(cfg *api.SAiModelConfig) (RuntimeConfig, *api.SAiModelVisualConfig) {
	out := RuntimeConfig{MaxRounds: 4, MaxTokens: 2048}
	if cfg == nil || cfg.Extensions == nil || cfg.Extensions.Visual == nil {
		return out, nil
	}
	vis := cfg.Extensions.Visual
	out.Enabled = vis.Enabled
	if vis.MaxRounds > 0 {
		out.MaxRounds = vis.MaxRounds
	}
	if vis.MaxTokens > 0 {
		out.MaxTokens = vis.MaxTokens
	}
	return out, vis
}

// Enabled reports whether visual extension is active on the resolved upstream
// (config.enabled plus visual_provider_id / visual_model_key columns).
func Enabled(up *models.ChatUpstream) bool {
	if up == nil || !up.ModelConfig.VisualEnabled() {
		return false
	}
	return strings.TrimSpace(up.VisualProviderId) != "" && strings.TrimSpace(up.VisualModelKey) != ""
}
