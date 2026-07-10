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

package codexconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	apmodels "yunion.io/x/onecloud/pkg/aiproxy/models"
)

const (
	defaultApplyPatchToolType     = "freeform"
	defaultCatalogTruncationLimit = int64(10000)
	catalogFileName               = "models_catalog.json"
)

// ModelListEntry is one OpenAI-compatible model from GET /ai/openai/v1/models.
type ModelListEntry struct {
	ID      string
	OwnedBy string
}

// ModelInfo represents a model entry in Codex models_catalog.json.
type ModelInfo struct {
	Slug                        string                    `json:"slug"`
	DisplayName                 string                    `json:"display_name"`
	Description                 string                    `json:"description,omitempty"`
	DefaultReasoningLevel       string                    `json:"default_reasoning_level,omitempty"`
	SupportedReasoningLevels    []ReasoningLevelPresetDTO `json:"supported_reasoning_levels"`
	ShellType                   string                    `json:"shell_type"`
	Visibility                  string                    `json:"visibility"`
	SupportedInAPI              bool                      `json:"supported_in_api"`
	Priority                    int                       `json:"priority"`
	AdditionalSpeedTiers        []string                  `json:"additional_speed_tiers"`
	AvailabilityNux             *ModelAvailabilityNux     `json:"availability_nux"`
	Upgrade                     *ModelInfoUpgrade         `json:"upgrade"`
	BaseInstructions            string                    `json:"base_instructions"`
	SupportsReasoningSummaries  bool                      `json:"supports_reasoning_summaries"`
	DefaultReasoningSummary     string                    `json:"default_reasoning_summary"`
	SupportVerbosity            bool                      `json:"support_verbosity"`
	DefaultVerbosity            *string                   `json:"default_verbosity"`
	ApplyPatchToolType          *string                   `json:"apply_patch_tool_type"`
	WebSearchToolType           string                    `json:"web_search_tool_type"`
	TruncationPolicy            TruncationPolicyConfig    `json:"truncation_policy"`
	SupportsParallelToolCalls   bool                      `json:"supports_parallel_tool_calls"`
	SupportsImageDetailOriginal bool                      `json:"supports_image_detail_original"`
	ContextWindow               *int                      `json:"context_window,omitempty"`
	MaxContextWindow            *int                      `json:"max_context_window,omitempty"`
	AutoCompactTokenLimit       *int                      `json:"auto_compact_token_limit,omitempty"`
	EffectiveContextWindowPct   int                       `json:"effective_context_window_percent"`
	ExperimentalSupportedTools  []string                  `json:"experimental_supported_tools"`
	InputModalities             []string                  `json:"input_modalities"`
	SupportsSearchTool          bool                      `json:"supports_search_tool"`
}

// ModelAvailabilityNux is a placeholder for Codex model availability nux.
type ModelAvailabilityNux struct{}

// ModelInfoUpgrade is a placeholder for Codex model upgrade info.
type ModelInfoUpgrade struct{}

// ReasoningLevelPresetDTO is the JSON shape Codex expects for reasoning presets.
type ReasoningLevelPresetDTO struct {
	Effort      string `json:"effort"`
	Description string `json:"description"`
}

// TruncationPolicyConfig matches Codex's truncation_policy field.
type TruncationPolicyConfig struct {
	Mode  string `json:"mode"`
	Limit int64  `json:"limit"`
}

// DisplayNameFromSlug converts a slug like "gpt-5.5-codex" to "GPT 5.5 Codex".
func DisplayNameFromSlug(slug string) string {
	slug = strings.ReplaceAll(slug, "-", " ")
	words := strings.Fields(slug)
	for i, w := range words {
		lower := strings.ToLower(w)
		if isASCIIGPTPrefix(lower) {
			words[i] = "GPT" + w[3:]
			continue
		}
		words[i] = asciiTitle(w)
	}
	return strings.Join(words, " ")
}

func asciiTitle(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func isASCIIGPTPrefix(s string) bool {
	if len(s) < 3 {
		return false
	}
	lower := strings.ToLower(s)
	return lower[:3] == "gpt"
}

// BuildCatalogFromIDs builds Codex catalog entries from aiproxy model list results.
func BuildCatalogFromIDs(entries []ModelListEntry) []ModelInfo {
	if len(entries) == 0 {
		return nil
	}
	sorted := append([]ModelListEntry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})
	models := make([]ModelInfo, 0, len(sorted))
	for _, entry := range sorted {
		models = append(models, buildModelInfoFromEntry(entry))
	}
	return models
}

func buildModelInfoFromEntry(entry ModelListEntry) ModelInfo {
	slug := strings.TrimSpace(entry.ID)
	displayName := displayNameForSlug(slug)
	description := seedDescriptionForSlug(slug)
	return newModelInfo(slug, displayName, description, 0, nil)
}

func displayNameForSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	if idx := strings.LastIndex(slug, "/"); idx >= 0 && idx < len(slug)-1 {
		return DisplayNameFromSlug(slug[idx+1:])
	}
	return DisplayNameFromSlug(slug)
}

func seedDescriptionForSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	if desc := apmodels.CatalogSeedDescription(slug); desc != "" {
		return desc
	}
	if idx := strings.LastIndex(slug, "/"); idx >= 0 && idx < len(slug)-1 {
		return apmodels.CatalogSeedDescription(slug[idx+1:])
	}
	return ""
}

func newModelInfo(slug, displayName, description string, contextWindow int, inputModalities []string) ModelInfo {
	if len(inputModalities) == 0 {
		inputModalities = []string{"text"}
	}
	var ctxWin, maxCtxWin *int
	if contextWindow > 0 {
		v := contextWindow
		ctxWin = &v
		maxCtxWin = &v
	}
	applyPatchToolType := defaultApplyPatchToolType
	baseInstructions := defaultBaseInstructions(slug)
	return ModelInfo{
		Slug:                        slug,
		DisplayName:                 displayName,
		Description:                 description,
		SupportedReasoningLevels:    []ReasoningLevelPresetDTO{},
		ShellType:                   "unified_exec",
		Visibility:                  "list",
		SupportedInAPI:              true,
		Priority:                    0,
		AdditionalSpeedTiers:        []string{},
		BaseInstructions:            baseInstructions,
		DefaultReasoningSummary:     "none",
		WebSearchToolType:           "text",
		ApplyPatchToolType:          &applyPatchToolType,
		TruncationPolicy:            TruncationPolicyConfig{Mode: "tokens", Limit: defaultCatalogTruncationLimit},
		SupportsParallelToolCalls:   true,
		ContextWindow:               ctxWin,
		MaxContextWindow:            maxCtxWin,
		EffectiveContextWindowPct:   95,
		ExperimentalSupportedTools:  []string{},
		InputModalities:             inputModalities,
		SupportsImageDetailOriginal: false,
	}
}

// EnsureModelInCatalog appends a fallback catalog entry when model is not listed.
func EnsureModelInCatalog(catalog []ModelInfo, model string) []ModelInfo {
	model = strings.TrimSpace(model)
	if model == "" {
		return catalog
	}
	for _, item := range catalog {
		if item.Slug == model {
			return catalog
		}
	}
	return append(catalog, buildModelInfoFromEntry(ModelListEntry{ID: model}))
}

// CatalogContextWindow returns context_window for model when present in catalog.
func CatalogContextWindow(catalog []ModelInfo, model string) int {
	for _, item := range catalog {
		if item.Slug == model && item.ContextWindow != nil {
			return *item.ContextWindow
		}
	}
	return 0
}

// WriteModelsCatalog writes Codex-compatible models_catalog.json.
func WriteModelsCatalog(path string, models []ModelInfo) error {
	catalog := struct {
		Models []ModelInfo `json:"models"`
	}{Models: models}
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
