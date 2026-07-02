package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/aiproxy/chatlog"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	common_policy "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type usageFilter struct {
	Range    string
	Start    time.Time
	End      time.Time
	Timezone string

	APIKeyID  string
	Model     string
	Provider  string
	AuthIndex string
	Result    string
	Page      int
	PageSize  int
}

type usageOverview struct {
	Usage         usageOverviewUsage   `json:"usage"`
	Summary       usageOverviewSummary `json:"summary"`
	Series        []usageOverviewPoint `json:"series"`
	ServiceHealth []usageServiceHealth `json:"service_health"`
	Timezone      string               `json:"timezone"`
	RangeStart    time.Time            `json:"range_start"`
	RangeEnd      time.Time            `json:"range_end"`
	Truncated     bool                 `json:"truncated,omitempty"`
}

type usageOverviewUsage struct {
	RequestCount int     `json:"request_count"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
	TokenCount   int     `json:"token_count"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCost    float64 `json:"total_cost"`
}

type usageOverviewSummary struct {
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

type usageOverviewPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	RequestCount int       `json:"request_count"`
	SuccessCount int       `json:"success_count"`
	FailureCount int       `json:"failure_count"`
	TokenCount   int       `json:"token_count"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
}

type usageServiceHealth struct {
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

type usageAnalysis struct {
	TokenUsage            []usageOverviewPoint    `json:"token_usage"`
	APIKeyComposition     []usageComposition      `json:"api_key_composition"`
	ModelComposition      []usageComposition      `json:"model_composition"`
	AuthFilesComposition  []usageComposition      `json:"auth_files_composition"`
	AIProviderComposition []usageComposition      `json:"ai_provider_composition"`
	Heatmap               []usageHeatmapPoint     `json:"heatmap"`
	CostBreakdown         usageCostBreakdown      `json:"cost_breakdown"`
	ModelEfficiency       []usageModelEfficiency  `json:"model_efficiency"`
	LatencyDiagnostics    usageLatencyDiagnostics `json:"latency_diagnostics"`
	Timezone              string                  `json:"timezone"`
	RangeStart            time.Time               `json:"range_start"`
	RangeEnd              time.Time               `json:"range_end"`
	Truncated             bool                    `json:"truncated,omitempty"`
}

type usageComposition struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	RequestCount int     `json:"request_count"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
	TokenCount   int     `json:"token_count"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCost    float64 `json:"total_cost"`
	SuccessRate  float64 `json:"success_rate"`
}

type usageHeatmapPoint struct {
	Weekday      string `json:"weekday"`
	Hour         int    `json:"hour"`
	RequestCount int    `json:"request_count"`
	TokenCount   int    `json:"token_count"`
}

type usageCostBreakdown struct {
	TotalCost float64            `json:"total_cost"`
	Items     []usageComposition `json:"items"`
}

type usageModelEfficiency struct {
	Model                  string  `json:"model"`
	RequestCount           int     `json:"request_count"`
	TokensPerRequest       float64 `json:"tokens_per_request"`
	OutputTokensPerRequest float64 `json:"output_tokens_per_request"`
	CostPerRequest         float64 `json:"cost_per_request"`
}

type usageLatencyDiagnostics struct {
	RequestCount int     `json:"request_count"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P50LatencyMs int64   `json:"p50_latency_ms"`
	P95LatencyMs int64   `json:"p95_latency_ms"`
	MaxLatencyMs int64   `json:"max_latency_ms"`
}

type usageEvents struct {
	Events     []usageEvent `json:"events"`
	TotalCount int          `json:"total_count"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
	Truncated  bool         `json:"truncated,omitempty"`
}

type usageAPIKeyOptions struct {
	Overview  []usageAPIKeyOption `json:"overview"`
	Analysis  []usageAPIKeyOption `json:"analysis"`
	Truncated bool                `json:"truncated,omitempty"`
}

type usageAPIKeyOption struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Value        string `json:"value"`
	Label        string `json:"label"`
	DisplayKey   string `json:"display_key"`
	RequestCount int    `json:"request_count"`
	TokenCount   int    `json:"token_count"`
}

type usageEvent struct {
	ID           string      `json:"id"`
	RequestID    string      `json:"request_id"`
	Timestamp    time.Time   `json:"timestamp"`
	APIKeyID     string      `json:"api_key_id"`
	APIKey       string      `json:"api_key"`
	Model        string      `json:"model"`
	Endpoint     string      `json:"endpoint"`
	Source       string      `json:"source"`
	Provider     string      `json:"provider"`
	AuthIndex    string      `json:"auth_index"`
	Failed       bool        `json:"failed"`
	Result       string      `json:"result"`
	StatusCode   int         `json:"status_code"`
	ErrorCode    string      `json:"error_code,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
	LatencyMs    int64       `json:"latency_ms"`
	TTFTMs       int64       `json:"ttft_ms"`
	Tokens       usageTokens `json:"tokens"`
	InputTokens  int         `json:"input_tokens"`
	OutputTokens int         `json:"output_tokens"`
	TotalTokens  int         `json:"total_tokens"`
	CostUSD      float64     `json:"cost_usd"`
	PricingStyle string      `json:"pricing_style"`
	ProjectID    string      `json:"project_id,omitempty"`
	DomainID     string      `json:"domain_id,omitempty"`
}

type usageTokens struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	ReasoningTokens     int `json:"reasoning_tokens"`
	CachedTokens        int `json:"cached_tokens"`
	CacheReadTokens     int `json:"cache_read_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
	TotalTokens         int `json:"total_tokens"`
}

func usageOverviewHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	filter, records, truncated, ok := readUsageRequest(ctx, w, r)
	if !ok {
		return
	}
	overview := buildUsageOverview(records, filter)
	overview.Truncated = truncated
	sendJSON(w, overview)
}

func usageAnalysisHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	filter, records, truncated, ok := readUsageRequest(ctx, w, r)
	if !ok {
		return
	}
	analysis := buildUsageAnalysis(records, filter)
	analysis.Truncated = truncated
	sendJSON(w, analysis)
}

func usageEventsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	filter, records, truncated, ok := readUsageRequest(ctx, w, r)
	if !ok {
		return
	}
	events := buildUsageEvents(records, filter)
	events.Truncated = truncated
	sendJSON(w, events)
}

func usageAPIKeysOptionsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	_, records, truncated, ok := readUsageRequest(ctx, w, r)
	if !ok {
		return
	}
	options := buildUsageAPIKeyOptions(records)
	options.Truncated = truncated
	sendJSON(w, options)
}

func readUsageRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) (usageFilter, []chatlog.Record, bool, bool) {
	if r.Method != http.MethodGet {
		httperrors.InvalidInputError(ctx, w, "only GET is supported")
		return usageFilter{}, nil, false, false
	}
	userCred := auth.FetchUserCredential(ctx, common_policy.FilterPolicyCredential)
	result := common_policy.PolicyManager.Allow(rbacscope.ScopeSystem, userCred, api.SERVICE_TYPE, "usage", common_policy.PolicyActionList)
	if result.Result == rbacutils.Deny {
		httperrors.ForbiddenError(ctx, w, "Not allow to access")
		return usageFilter{}, nil, false, false
	}
	filter, err := parseUsageFilter(r)
	if err != nil {
		httperrors.InvalidInputError(ctx, w, "%v", err)
		return usageFilter{}, nil, false, false
	}
	ret, err := chatlog.Read(ctx, chatlog.ReadOptions{
		Start: filter.Start,
		End:   filter.End,
		Limit: 10000,
	})
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return usageFilter{}, nil, false, false
	}
	return filter, filterRecords(ret.Logs, filter), ret.Truncated, true
}

func sendJSON(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(obj)
}

func parseUsageFilter(r *http.Request) (usageFilter, error) {
	q := r.URL.Query()
	loc := time.Local
	timezone := strings.TrimSpace(q.Get("timezone"))
	if timezone == "" {
		timezone = strings.TrimSpace(q.Get("tz"))
	}
	if timezone != "" {
		loaded, err := time.LoadLocation(timezone)
		if err != nil {
			return usageFilter{}, err
		}
		loc = loaded
	} else {
		timezone = loc.String()
	}
	now := time.Now().In(loc)
	rng := strings.TrimSpace(q.Get("range"))
	if rng == "" {
		rng = "24h"
	}
	start, end, err := usageRange(rng, q.Get("start"), q.Get("end"), now)
	if err != nil {
		return usageFilter{}, err
	}
	page := parsePositiveInt(q.Get("page"), 1)
	pageSize := parsePositiveInt(firstNonEmpty(q.Get("page_size"), q.Get("limit")), 50)
	if pageSize > 1000 {
		pageSize = 1000
	}
	return usageFilter{
		Range:     rng,
		Start:     start,
		End:       end,
		Timezone:  timezone,
		APIKeyID:  strings.TrimSpace(q.Get("api_key_id")),
		Model:     strings.TrimSpace(q.Get("model")),
		Provider:  strings.TrimSpace(q.Get("provider")),
		AuthIndex: firstNonEmpty(q.Get("auth_index"), q.Get("source")),
		Result:    strings.TrimSpace(q.Get("result")),
		Page:      page,
		PageSize:  pageSize,
	}, nil
}

func usageRange(rng, rawStart, rawEnd string, now time.Time) (time.Time, time.Time, error) {
	switch rng {
	case "4h", "8h", "12h", "24h":
		d, _ := time.ParseDuration(rng)
		return now.Add(-d), now, nil
	case "7d":
		return now.AddDate(0, 0, -7), now, nil
	case "30d":
		return now.AddDate(0, 0, -30), now, nil
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return start, now, nil
	case "yesterday":
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return end.AddDate(0, 0, -1), end, nil
	case "custom":
		start, err := time.Parse(time.RFC3339, strings.TrimSpace(rawStart))
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end, err := time.Parse(time.RFC3339, strings.TrimSpace(rawEnd))
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		if !end.After(start) {
			return time.Time{}, time.Time{}, errors.New("end must be after start")
		}
		return start, end, nil
	default:
		return time.Time{}, time.Time{}, errors.New("unsupported range")
	}
}

func filterRecords(records []chatlog.Record, filter usageFilter) []chatlog.Record {
	ret := make([]chatlog.Record, 0, len(records))
	for _, rec := range records {
		if filter.APIKeyID != "" && rec.VirtualKey != filter.APIKeyID {
			continue
		}
		if filter.Model != "" && recordModel(rec) != filter.Model {
			continue
		}
		if filter.Provider != "" && rec.Provider != filter.Provider {
			continue
		}
		if filter.AuthIndex != "" && rec.AiKey != filter.AuthIndex {
			continue
		}
		if filter.Result == "success" && !rec.Success {
			continue
		}
		if filter.Result == "failed" && rec.Success {
			continue
		}
		ret = append(ret, rec)
	}
	return ret
}

func buildUsageOverview(records []chatlog.Record, filter usageFilter) usageOverview {
	durationMinutes := filter.End.Sub(filter.Start).Minutes()
	if durationMinutes <= 0 {
		durationMinutes = 1
	}
	overview := usageOverview{
		Timezone:      filter.Timezone,
		RangeStart:    filter.Start,
		RangeEnd:      filter.End,
		Series:        []usageOverviewPoint{},
		ServiceHealth: []usageServiceHealth{},
	}
	bucketSize := overviewBucketSize(filter.End.Sub(filter.Start))
	series := map[time.Time]*usageOverviewPoint{}
	health := map[string]*usageServiceHealth{}
	latencySum := int64(0)
	latencyCount := 0
	healthLatencySum := map[string]int64{}
	healthLatencyCount := map[string]int{}

	for _, rec := range records {
		tokens := recordTotalTokens(rec)
		overview.Summary.RequestCount++
		overview.Summary.InputTokens += rec.PromptTokens
		overview.Summary.OutputTokens += rec.CompletionTokens
		overview.Summary.TokenCount += tokens
		if rec.Success {
			overview.Summary.SuccessCount++
		} else {
			overview.Summary.FailureCount++
		}
		if rec.LatencyMs > 0 {
			latencySum += rec.LatencyMs
			latencyCount++
		}

		bucket := rec.Timestamp.Truncate(bucketSize)
		point := series[bucket]
		if point == nil {
			point = &usageOverviewPoint{Timestamp: bucket}
			series[bucket] = point
		}
		point.RequestCount++
		point.InputTokens += rec.PromptTokens
		point.OutputTokens += rec.CompletionTokens
		point.TokenCount += tokens
		if rec.Success {
			point.SuccessCount++
		} else {
			point.FailureCount++
		}

		key := rec.Provider + "\x00" + recordModel(rec)
		row := health[key]
		if row == nil {
			row = &usageServiceHealth{Provider: rec.Provider, Model: recordModel(rec)}
			health[key] = row
		}
		row.RequestCount++
		row.TokenCount += tokens
		row.LastStatusCode = rec.StatusCode
		if rec.Success {
			row.SuccessCount++
		} else {
			row.FailureCount++
		}
		if rec.LatencyMs > 0 {
			healthLatencySum[key] += rec.LatencyMs
			healthLatencyCount[key]++
		}
	}

	overview.Summary.RPM = float64(overview.Summary.RequestCount) / durationMinutes
	overview.Summary.TPM = float64(overview.Summary.TokenCount) / durationMinutes
	if latencyCount > 0 {
		overview.Summary.AvgLatencyMs = float64(latencySum) / float64(latencyCount)
	}
	overview.Usage = usageOverviewUsage{
		RequestCount: overview.Summary.RequestCount,
		SuccessCount: overview.Summary.SuccessCount,
		FailureCount: overview.Summary.FailureCount,
		TokenCount:   overview.Summary.TokenCount,
		InputTokens:  overview.Summary.InputTokens,
		OutputTokens: overview.Summary.OutputTokens,
		TotalCost:    overview.Summary.TotalCost,
	}

	for _, point := range series {
		overview.Series = append(overview.Series, *point)
	}
	sort.Slice(overview.Series, func(i, j int) bool {
		return overview.Series[i].Timestamp.Before(overview.Series[j].Timestamp)
	})
	for key, row := range health {
		if row.RequestCount > 0 {
			row.SuccessRate = float64(row.SuccessCount) / float64(row.RequestCount)
		}
		if healthLatencyCount[key] > 0 {
			row.AvgLatencyMs = float64(healthLatencySum[key]) / float64(healthLatencyCount[key])
		}
		overview.ServiceHealth = append(overview.ServiceHealth, *row)
	}
	sort.Slice(overview.ServiceHealth, func(i, j int) bool {
		if overview.ServiceHealth[i].RequestCount != overview.ServiceHealth[j].RequestCount {
			return overview.ServiceHealth[i].RequestCount > overview.ServiceHealth[j].RequestCount
		}
		if overview.ServiceHealth[i].Provider != overview.ServiceHealth[j].Provider {
			return overview.ServiceHealth[i].Provider < overview.ServiceHealth[j].Provider
		}
		return overview.ServiceHealth[i].Model < overview.ServiceHealth[j].Model
	})
	return overview
}

func buildUsageAnalysis(records []chatlog.Record, filter usageFilter) usageAnalysis {
	overview := buildUsageOverview(records, filter)
	analysis := usageAnalysis{
		TokenUsage:            overview.Series,
		APIKeyComposition:     []usageComposition{},
		ModelComposition:      []usageComposition{},
		AuthFilesComposition:  []usageComposition{},
		AIProviderComposition: []usageComposition{},
		Heatmap:               []usageHeatmapPoint{},
		CostBreakdown: usageCostBreakdown{
			Items: []usageComposition{},
		},
		ModelEfficiency: []usageModelEfficiency{},
		Timezone:        filter.Timezone,
		RangeStart:      filter.Start,
		RangeEnd:        filter.End,
	}
	apiKeys := map[string]*usageComposition{}
	models := map[string]*usageComposition{}
	authFiles := map[string]*usageComposition{}
	providers := map[string]*usageComposition{}
	heatmap := map[string]*usageHeatmapPoint{}
	latencies := make([]int64, 0, len(records))

	for _, rec := range records {
		tokens := recordTotalTokens(rec)
		addComposition(apiKeys, rec.VirtualKey, rec.VirtualKey, rec, tokens)
		addComposition(models, recordModel(rec), recordModel(rec), rec, tokens)
		addComposition(authFiles, rec.AiKey, rec.AiKey, rec, tokens)
		addComposition(providers, rec.Provider, rec.Provider, rec, tokens)
		weekday := rec.Timestamp.Weekday().String()
		heatKey := weekday + "\x00" + strconv.Itoa(rec.Timestamp.Hour())
		point := heatmap[heatKey]
		if point == nil {
			point = &usageHeatmapPoint{Weekday: weekday, Hour: rec.Timestamp.Hour()}
			heatmap[heatKey] = point
		}
		point.RequestCount++
		point.TokenCount += tokens
		if rec.LatencyMs > 0 {
			latencies = append(latencies, rec.LatencyMs)
		}
	}

	analysis.APIKeyComposition = sortedCompositions(apiKeys)
	analysis.ModelComposition = sortedCompositions(models)
	analysis.AuthFilesComposition = sortedCompositions(authFiles)
	analysis.AIProviderComposition = sortedCompositions(providers)
	for _, point := range heatmap {
		analysis.Heatmap = append(analysis.Heatmap, *point)
	}
	sort.Slice(analysis.Heatmap, func(i, j int) bool {
		if analysis.Heatmap[i].Weekday != analysis.Heatmap[j].Weekday {
			return analysis.Heatmap[i].Weekday < analysis.Heatmap[j].Weekday
		}
		return analysis.Heatmap[i].Hour < analysis.Heatmap[j].Hour
	})
	analysis.CostBreakdown.Items = analysis.ModelComposition
	analysis.ModelEfficiency = buildModelEfficiency(analysis.ModelComposition)
	analysis.LatencyDiagnostics = buildLatencyDiagnostics(latencies)
	return analysis
}

func buildUsageEvents(records []chatlog.Record, filter usageFilter) usageEvents {
	total := len(records)
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	ret := usageEvents{
		Events:     []usageEvent{},
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
	for _, rec := range records[start:end] {
		inputTokens := rec.PromptTokens
		outputTokens := rec.CompletionTokens
		totalTokens := recordTotalTokens(rec)
		result := "success"
		if !rec.Success {
			result = "failed"
		}
		ret.Events = append(ret.Events, usageEvent{
			ID:           rec.RequestID,
			RequestID:    rec.RequestID,
			Timestamp:    rec.Timestamp,
			APIKeyID:     rec.VirtualKey,
			APIKey:       rec.VirtualKey,
			Model:        recordModel(rec),
			Endpoint:     rec.Path,
			Source:       firstNonEmpty(rec.Provider, rec.AiKey),
			Provider:     rec.Provider,
			AuthIndex:    rec.AiKey,
			Failed:       !rec.Success,
			Result:       result,
			StatusCode:   rec.StatusCode,
			ErrorCode:    rec.ErrorCode,
			ErrorMessage: rec.ErrorMessage,
			LatencyMs:    rec.LatencyMs,
			Tokens: usageTokens{
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				TotalTokens:  totalTokens,
			},
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
			ProjectID:    rec.ProjectID,
			DomainID:     rec.DomainID,
		})
	}
	return ret
}

func buildUsageAPIKeyOptions(records []chatlog.Record) usageAPIKeyOptions {
	items := map[string]*usageAPIKeyOption{}
	for _, rec := range records {
		id := strings.TrimSpace(rec.VirtualKey)
		if id == "" {
			continue
		}
		item := items[id]
		if item == nil {
			item = &usageAPIKeyOption{
				ID:         id,
				Name:       id,
				Value:      id,
				Label:      id,
				DisplayKey: id,
			}
			items[id] = item
		}
		item.RequestCount++
		item.TokenCount += recordTotalTokens(rec)
	}
	options := make([]usageAPIKeyOption, 0, len(items))
	for _, item := range items {
		options = append(options, *item)
	}
	sort.Slice(options, func(i, j int) bool {
		if options[i].RequestCount != options[j].RequestCount {
			return options[i].RequestCount > options[j].RequestCount
		}
		return options[i].Name < options[j].Name
	})
	return usageAPIKeyOptions{
		Overview: options,
		Analysis: append(make([]usageAPIKeyOption, 0, len(options)), options...),
	}
}

func addComposition(items map[string]*usageComposition, id, name string, rec chatlog.Record, tokens int) {
	if id == "" {
		id = "unknown"
	}
	if name == "" {
		name = id
	}
	item := items[id]
	if item == nil {
		item = &usageComposition{ID: id, Name: name}
		items[id] = item
	}
	item.RequestCount++
	item.InputTokens += rec.PromptTokens
	item.OutputTokens += rec.CompletionTokens
	item.TokenCount += tokens
	if rec.Success {
		item.SuccessCount++
	} else {
		item.FailureCount++
	}
}

func sortedCompositions(items map[string]*usageComposition) []usageComposition {
	ret := make([]usageComposition, 0, len(items))
	for _, item := range items {
		if item.RequestCount > 0 {
			item.SuccessRate = float64(item.SuccessCount) / float64(item.RequestCount)
		}
		ret = append(ret, *item)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].RequestCount != ret[j].RequestCount {
			return ret[i].RequestCount > ret[j].RequestCount
		}
		return ret[i].Name < ret[j].Name
	})
	return ret
}

func buildModelEfficiency(items []usageComposition) []usageModelEfficiency {
	ret := make([]usageModelEfficiency, 0, len(items))
	for _, item := range items {
		row := usageModelEfficiency{
			Model:        item.Name,
			RequestCount: item.RequestCount,
		}
		if item.RequestCount > 0 {
			row.TokensPerRequest = float64(item.TokenCount) / float64(item.RequestCount)
			row.OutputTokensPerRequest = float64(item.OutputTokens) / float64(item.RequestCount)
			row.CostPerRequest = item.TotalCost / float64(item.RequestCount)
		}
		ret = append(ret, row)
	}
	return ret
}

func buildLatencyDiagnostics(latencies []int64) usageLatencyDiagnostics {
	ret := usageLatencyDiagnostics{RequestCount: len(latencies)}
	if len(latencies) == 0 {
		return ret
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	sum := int64(0)
	for _, latency := range latencies {
		sum += latency
	}
	ret.AvgLatencyMs = float64(sum) / float64(len(latencies))
	ret.P50LatencyMs = percentileLatency(latencies, 0.50)
	ret.P95LatencyMs = percentileLatency(latencies, 0.95)
	ret.MaxLatencyMs = latencies[len(latencies)-1]
	return ret
}

func percentileLatency(sorted []int64, pct float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(len(sorted))*pct)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func overviewBucketSize(d time.Duration) time.Duration {
	if d <= 24*time.Hour {
		return time.Hour
	}
	return 24 * time.Hour
}

func recordModel(rec chatlog.Record) string {
	if rec.ModelFinal != "" {
		return rec.ModelFinal
	}
	if rec.ModelRequested != "" {
		return rec.ModelRequested
	}
	return "unknown"
}

func recordTotalTokens(rec chatlog.Record) int {
	if rec.TotalTokens > 0 {
		return rec.TotalTokens
	}
	return rec.PromptTokens + rec.CompletionTokens
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
