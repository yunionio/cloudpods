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

package aiproxy

import "time"

type UsageOverview struct {
	Usage         UsageOverviewUsage   `json:"usage"`
	Summary       UsageOverviewSummary `json:"summary"`
	Series        []UsageOverviewPoint `json:"series"`
	ServiceHealth []UsageServiceHealth `json:"service_health"`
	Timezone      string               `json:"timezone"`
	RangeStart    time.Time            `json:"range_start"`
	RangeEnd      time.Time            `json:"range_end"`
	Truncated     bool                 `json:"truncated,omitempty"`
}

type UsageOverviewUsage struct {
	RequestCount int     `json:"request_count"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
	TokenCount   int     `json:"token_count"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCost    float64 `json:"total_cost"`
}

type UsageOverviewSummary struct {
	RequestCount    int     `json:"request_count"`
	SuccessCount    int     `json:"success_count"`
	FailureCount    int     `json:"failure_count"`
	TokenCount      int     `json:"token_count"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	CachedTokens    int     `json:"cached_tokens"`
	ReasoningTokens int     `json:"reasoning_tokens"`
	RPM             float64 `json:"rpm"`
	TPM             float64 `json:"tpm"`
	TotalCost       float64 `json:"total_cost"`
	CacheRate       float64 `json:"cache_rate"`
	AvgLatencyMs    float64 `json:"avg_latency_ms"`
}

type UsageOverviewPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	RequestCount int       `json:"request_count"`
	SuccessCount int       `json:"success_count"`
	FailureCount int       `json:"failure_count"`
	TokenCount   int       `json:"token_count"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
}

type UsageServiceHealth struct {
	Provider       string  `json:"provider"`
	Model          string  `json:"model"`
	RequestCount   int     `json:"request_count"`
	SuccessCount   int     `json:"success_count"`
	FailureCount   int     `json:"failure_count"`
	TokenCount     int     `json:"token_count"`
	SuccessRate    float64 `json:"success_rate"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	LastStatusCode int     `json:"last_status_code,omitempty"`
}

type UsageAnalysis struct {
	TokenUsage            []UsageOverviewPoint    `json:"token_usage"`
	APIKeyComposition     []UsageComposition      `json:"api_key_composition"`
	ModelComposition      []UsageComposition      `json:"model_composition"`
	AuthFilesComposition  []UsageComposition      `json:"auth_files_composition"`
	AIProviderComposition []UsageComposition      `json:"ai_provider_composition"`
	Heatmap               []UsageHeatmapPoint     `json:"heatmap"`
	CostBreakdown         UsageCostBreakdown      `json:"cost_breakdown"`
	ModelEfficiency       []UsageModelEfficiency  `json:"model_efficiency"`
	LatencyDiagnostics    UsageLatencyDiagnostics `json:"latency_diagnostics"`
	Timezone              string                  `json:"timezone"`
	RangeStart            time.Time               `json:"range_start"`
	RangeEnd              time.Time               `json:"range_end"`
	Truncated             bool                    `json:"truncated,omitempty"`
}

type UsageComposition struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Label        string  `json:"label,omitempty"`
	RequestCount int     `json:"request_count"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
	TokenCount   int     `json:"token_count"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCost    float64 `json:"total_cost"`
	SuccessRate  float64 `json:"success_rate"`
}

type UsageHeatmapPoint struct {
	Weekday      string `json:"weekday"`
	Hour         int    `json:"hour"`
	RequestCount int    `json:"request_count"`
	TokenCount   int    `json:"token_count"`
}

type UsageCostBreakdown struct {
	TotalCost float64            `json:"total_cost"`
	Items     []UsageComposition `json:"items"`
}

type UsageModelEfficiency struct {
	Model                  string  `json:"model"`
	RequestCount           int     `json:"request_count"`
	TokensPerRequest       float64 `json:"tokens_per_request"`
	OutputTokensPerRequest float64 `json:"output_tokens_per_request"`
	CostPerRequest         float64 `json:"cost_per_request"`
}

type UsageLatencyDiagnostics struct {
	RequestCount int     `json:"request_count"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P50LatencyMs int64   `json:"p50_latency_ms"`
	P95LatencyMs int64   `json:"p95_latency_ms"`
	MaxLatencyMs int64   `json:"max_latency_ms"`
}

type UsageAPIKeyOptions struct {
	Overview  []UsageAPIKeyOption `json:"overview"`
	Analysis  []UsageAPIKeyOption `json:"analysis"`
	Truncated bool                `json:"truncated,omitempty"`
}

type UsageAPIKeyOption struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Value        string `json:"value"`
	Label        string `json:"label"`
	DisplayKey   string `json:"display_key"`
	RequestCount int    `json:"request_count"`
	TokenCount   int    `json:"token_count"`
}

type UsageEvent struct {
	ID             string      `json:"id"`
	RequestID      string      `json:"request_id"`
	Timestamp      time.Time   `json:"timestamp"`
	APIKeyID       string      `json:"api_key_id"`
	APIKey         string      `json:"api_key"`
	APIKeyName     string      `json:"api_key_name,omitempty"`
	APIKeyLabel    string      `json:"api_key_label,omitempty"`
	Model          string      `json:"model"`
	Endpoint       string      `json:"endpoint"`
	Source         string      `json:"source"`
	Provider       string      `json:"provider"`
	AuthIndex      string      `json:"auth_index"`
	AuthIndexName  string      `json:"auth_index_name,omitempty"`
	AuthIndexLabel string      `json:"auth_index_label,omitempty"`
	Failed         bool        `json:"failed"`
	Result         string      `json:"result"`
	StatusCode     int         `json:"status_code"`
	ErrorCode      string      `json:"error_code,omitempty"`
	ErrorMessage   string      `json:"error_message,omitempty"`
	LatencyMs      int64       `json:"latency_ms"`
	TTFTMs         int64       `json:"ttft_ms"`
	Tokens         UsageTokens `json:"tokens"`
	InputTokens    int         `json:"input_tokens"`
	OutputTokens   int         `json:"output_tokens"`
	TotalTokens    int         `json:"total_tokens"`
	CostUSD        float64     `json:"cost_usd"`
	PricingStyle   string      `json:"pricing_style"`
	ProjectID      string      `json:"project_id,omitempty"`
	ProjectName    string      `json:"project_name,omitempty"`
	DomainID       string      `json:"domain_id,omitempty"`
	DomainName     string      `json:"domain_name,omitempty"`
}

type UsageTokens struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	ReasoningTokens     int `json:"reasoning_tokens"`
	CachedTokens        int `json:"cached_tokens"`
	CacheReadTokens     int `json:"cache_read_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
	TotalTokens         int `json:"total_tokens"`
}

type UsageResource struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}
