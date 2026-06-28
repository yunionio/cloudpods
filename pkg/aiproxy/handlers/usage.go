package handlers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/aiproxy/chatlog"
	"yunion.io/x/onecloud/pkg/aiproxy/models"
	api "yunion.io/x/onecloud/pkg/apis/aiproxy"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_policy "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/excelutils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type usageFilter struct {
	Range    string
	Start    time.Time
	End      time.Time
	Timezone string

	APIKeyID   string
	RequestID  string
	RequestIDs map[string]struct{}
	Model      string
	Provider   string
	Source     string
	AuthIndex  string
	Result     string
	Limit      int
	Offset     int
}

type aiProxyUsageManager struct{}
type usageListResult struct {
	*printutils.ListResult
	Truncated bool
}

const maxUsageReadLimit = 10000

var aiProxyUsage = aiProxyUsageManager{}

var aiProxyUsageResources = []api.UsageResource{
	{ID: "overview", Path: "/api/v2/ai_proxy_usage/overview"},
	{ID: "analysis", Path: "/api/v2/ai_proxy_usage/analysis"},
	{ID: "events", Path: "/api/v2/ai_proxy_usage/events"},
	{ID: "api-keys-options", Path: "/api/v2/ai_proxy_usage/api-keys-options"},
}

func (m aiProxyUsageManager) List() []api.UsageResource {
	return append([]api.UsageResource(nil), aiProxyUsageResources...)
}

func (m aiProxyUsageManager) Get(ctx context.Context, id string, query jsonutils.JSONObject) (interface{}, error) {
	switch id {
	case "overview", "analysis", "events", "api-keys-options":
	default:
		return nil, httperrors.NewResourceNotFoundError2("ai_proxy_usage", id)
	}
	filter, err := parseUsageFilterQuery(query)
	if err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}
	readLimit := maxUsageReadLimit
	pushdownFilter := false
	if id == "events" {
		readLimit = eventReadLimit(filter)
		pushdownFilter = true
	}
	records, truncated, err := m.read(ctx, filter, readLimit, pushdownFilter)
	if err != nil {
		return nil, err
	}
	switch id {
	case "overview":
		overview := buildUsageOverview(records, filter)
		overview.Truncated = truncated
		return overview, nil
	case "analysis":
		analysis := buildUsageAnalysis(records, filter, resolveUsageNames(records))
		analysis.Truncated = truncated
		return analysis, nil
	case "events":
		names := resolveUsageNames(records)
		return usageListResult{
			ListResult: buildUsageEvents(records, filter, names),
			Truncated:  truncated,
		}, nil
	case "api-keys-options":
		options := buildUsageAPIKeyOptions(records, resolveUsageNames(records))
		options.Truncated = truncated
		return options, nil
	}
	return nil, httperrors.NewResourceNotFoundError2("ai_proxy_usage", id)
}

func eventReadLimit(filter usageFilter) int {
	if filter.Limit <= 0 {
		return maxUsageReadLimit
	}
	limit := filter.Offset + filter.Limit
	if limit <= 0 || limit > maxUsageReadLimit {
		return maxUsageReadLimit
	}
	return limit
}

func (m aiProxyUsageManager) read(ctx context.Context, filter usageFilter, limit int, pushdownFilter bool) ([]chatlog.Record, bool, error) {
	var readFilter func(chatlog.Record) bool
	if pushdownFilter {
		readFilter = func(rec chatlog.Record) bool {
			return recordMatchesFilter(rec, filter)
		}
	}
	ret, err := chatlog.Read(ctx, chatlog.ReadOptions{
		Start:     filter.Start,
		End:       filter.End,
		Limit:     limit,
		RequestID: filter.RequestID,
		Filter:    readFilter,
	})
	if err != nil {
		return nil, false, err
	}
	records := ret.Logs
	if !pushdownFilter {
		records = filterRecords(records, filter)
	}
	return records, ret.Truncated, nil
}

type usageNames struct {
	VirtualKeys map[string]string
	AiKeys      map[string]string
	Projects    map[string]string
	Domains     map[string]string
}

func resolveUsageNames(records []chatlog.Record) usageNames {
	var virtualKeyIds, aiKeyIds, projectIds, domainIds []string
	for _, rec := range records {
		virtualKeyIds = appendUsageNameId(virtualKeyIds, rec.VirtualKey)
		aiKeyIds = appendUsageNameId(aiKeyIds, rec.AiKey)
		projectIds = appendUsageNameId(projectIds, rec.ProjectID)
		domainIds = appendUsageNameId(domainIds, rec.DomainID)
	}
	return usageNames{
		VirtualKeys: fetchUsageNameMap(models.AiVirtualKeyManager, virtualKeyIds, "ai_virtual_key"),
		AiKeys:      fetchUsageNameMap(models.AiKeyManager, aiKeyIds, "ai_key"),
		Projects:    fetchUsageNameMap(db.TenantCacheManager, projectIds, "project"),
		Domains:     fetchUsageNameMap(db.TenantCacheManager, domainIds, "domain"),
	}
}

func appendUsageNameId(ids []string, id string) []string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ids
	}
	return append(ids, id)
}

func fetchUsageNameMap(manager db.IStandaloneModelManager, ids []string, resource string) map[string]string {
	if len(ids) == 0 {
		return map[string]string{}
	}
	ret, err := db.FetchIdNameMap2(manager, ids)
	if err != nil {
		log.Errorf("FetchIdNameMap2 %s: %v", resource, err)
		return map[string]string{}
	}
	return ret
}

func usageName(names map[string]string, id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if name := strings.TrimSpace(names[id]); name != "" {
		return name
	}
	return id
}

func aiProxyUsageListHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if !checkUsageAccess(ctx, w) {
		return
	}
	body := jsonutils.NewDict()
	items := jsonutils.NewArray()
	for _, item := range aiProxyUsage.List() {
		items.Add(jsonutils.Marshal(item))
	}
	body.Set("ai_proxy_usage", items)
	body.Set("total", jsonutils.NewInt(int64(len(aiProxyUsageResources))))
	appsrv.SendJSON(w, body)
}

func aiProxyUsageGetHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if !checkUsageAccess(ctx, w) {
		return
	}
	params := appctx.AppContextParams(ctx)
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	id := params["<id>"]
	if id == "events" && usageEventsExportRequested(query) {
		if err := aiProxyUsage.ExportEvents(ctx, w, query); err != nil {
			httperrors.JsonClientError(ctx, w, httperrors.NewGeneralError(err))
		}
		return
	}
	result, err := aiProxyUsage.Get(ctx, id, query)
	if err != nil {
		httperrors.JsonClientError(ctx, w, httperrors.NewGeneralError(err))
		return
	}
	if list, ok := result.(usageListResult); ok {
		appsrv.SendJSON(w, usageListResultJSON(list))
		return
	}
	body := jsonutils.NewDict()
	body.Set("ai_proxy_usage", jsonutils.Marshal(result))
	appsrv.SendJSON(w, body)
}

func (m aiProxyUsageManager) ExportEvents(ctx context.Context, w http.ResponseWriter, query jsonutils.JSONObject) error {
	if _, _, _, err := usageEventsExportParams(query); err != nil {
		return err
	}
	filter, err := parseUsageFilterQuery(query)
	if err != nil {
		return httperrors.NewInputParameterError("%v", err)
	}
	filter.Limit = usageEventsExportLimit(query)
	records, _, err := m.read(ctx, filter, eventReadLimit(filter), true)
	if err != nil {
		return err
	}
	events := buildUsageEvents(records, filter, resolveUsageNames(records))
	return writeUsageEventsExport(w, events.Data, query)
}

func usageEventsExportLimit(query jsonutils.JSONObject) int {
	limit := maxUsageReadLimit
	if query != nil {
		if v, err := query.Int("export_limit"); err == nil && v > 0 {
			limit = int(v)
		} else if v, err := query.Int("limit"); err == nil && v > 0 {
			limit = int(v)
		}
	}
	if limit > maxUsageReadLimit {
		return maxUsageReadLimit
	}
	return limit
}

func usageEventsExportRequested(query jsonutils.JSONObject) bool {
	if query == nil {
		return false
	}
	export, _ := query.GetString("export")
	return strings.TrimSpace(export) != ""
}

func writeUsageEventsExport(w http.ResponseWriter, data []jsonutils.JSONObject, query jsonutils.JSONObject) error {
	keys, texts, fileName, err := usageEventsExportParams(query)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.xlsx\"", fileName))
	return excelutils.Export(data, keys, texts, w)
}

func usageEventsExportParams(query jsonutils.JSONObject) ([]string, []string, string, error) {
	if query == nil {
		return nil, nil, "", fmt.Errorf("missing export keys")
	}
	exportKeys, _ := query.GetString("export_keys")
	if strings.TrimSpace(exportKeys) == "" {
		return nil, nil, "", fmt.Errorf("missing export keys")
	}
	keys := strings.Split(exportKeys, ",")
	exportTexts, _ := query.GetString("export_texts")
	texts := keys
	if exportTexts != "" {
		texts = strings.Split(exportTexts, ",")
	}
	if len(keys) != len(texts) {
		return nil, nil, "", fmt.Errorf("inconsistent export keys and texts")
	}
	fileName, _ := query.GetString("export_file_name")
	if fileName == "" {
		fileName = "export-ai_proxy_usage_events"
	}
	return keys, texts, fileName, nil
}

func aiProxyUsageEventsDistinctFieldHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if !checkUsageAccess(ctx, w) {
		return
	}
	_, query, _ := appsrv.FetchEnv(ctx, w, r)
	result, err := aiProxyUsage.DistinctField(ctx, query)
	if err != nil {
		httperrors.JsonClientError(ctx, w, httperrors.NewGeneralError(err))
		return
	}
	appsrv.SendJSON(w, result)
}

func usageListResultJSON(list usageListResult) jsonutils.JSONObject {
	body := modulebase.ListResult2JSON(list.ListResult).(*jsonutils.JSONDict)
	if list.Truncated {
		body.Set("truncated", jsonutils.JSONTrue)
	}
	return body
}

func (m aiProxyUsageManager) DistinctField(ctx context.Context, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	field, err := distinctFieldName(query)
	if err != nil {
		return nil, err
	}
	filter, err := parseUsageFilterQuery(query)
	if err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}
	records, _, err := m.read(ctx, filter, maxUsageReadLimit, true)
	if err != nil {
		return nil, err
	}
	return buildUsageEventDistinctField(records, resolveUsageNames(records), field)
}

func distinctFieldName(query jsonutils.JSONObject) (string, error) {
	field := ""
	if query != nil {
		field, _ = query.GetString("field")
	}
	field = strings.TrimSpace(field)
	if field == "" {
		return "", httperrors.NewMissingParameterError("field")
	}
	return field, nil
}

func checkUsageAccess(ctx context.Context, w http.ResponseWriter) bool {
	userCred := auth.FetchUserCredential(ctx, common_policy.FilterPolicyCredential)
	result := common_policy.PolicyManager.Allow(rbacscope.ScopeSystem, userCred, api.SERVICE_TYPE, "usage", common_policy.PolicyActionList)
	if result.Result == rbacutils.Deny {
		httperrors.ForbiddenError(ctx, w, "Not allow to access")
		return false
	}
	return true
}

func parseUsageFilterQuery(query jsonutils.JSONObject) (usageFilter, error) {
	rawQuery := ""
	if query != nil {
		rawQuery = query.QueryString()
	}
	return parseUsageFilter(&http.Request{URL: &url.URL{RawQuery: rawQuery}})
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
	limit := parseNonNegativeInt(firstNonEmpty(q.Get("limit"), q.Get("page_size")), 50)
	if limit > 1000 {
		limit = 1000
	}
	offset := parseNonNegativeInt(q.Get("offset"), -1)
	if offset < 0 {
		page := parsePositiveInt(q.Get("page"), 1)
		if page > 1 && limit > 0 {
			offset = (page - 1) * limit
		} else {
			offset = 0
		}
	}
	requestIDs := parseUsageRequestIDIn(q)
	requestID := firstNonEmpty(q.Get("id"), q.Get("request_id"))
	if requestID == "" && len(requestIDs) == 1 {
		for id := range requestIDs {
			requestID = id
		}
	}
	return usageFilter{
		Range:      rng,
		Start:      start,
		End:        end,
		Timezone:   timezone,
		APIKeyID:   strings.TrimSpace(q.Get("api_key_id")),
		RequestID:  requestID,
		RequestIDs: requestIDs,
		Model:      strings.TrimSpace(q.Get("model")),
		Provider:   strings.TrimSpace(q.Get("provider")),
		Source:     strings.TrimSpace(q.Get("source")),
		AuthIndex:  strings.TrimSpace(q.Get("auth_index")),
		Result:     strings.TrimSpace(q.Get("result")),
		Limit:      limit,
		Offset:     offset,
	}, nil
}

func parseUsageRequestIDIn(q url.Values) map[string]struct{} {
	ids := map[string]struct{}{}
	addIDs := func(raw string) {
		for _, id := range strings.Split(raw, ",") {
			id = strings.TrimSpace(id)
			id = strings.Trim(id, `"'`)
			if id != "" {
				ids[id] = struct{}{}
			}
		}
	}
	for _, raw := range q["id.in"] {
		addIDs(raw)
	}
	for key, vals := range q {
		if key != "filter" && !strings.HasPrefix(key, "filter.") {
			continue
		}
		for _, raw := range vals {
			raw = strings.TrimSpace(raw)
			for _, prefix := range []string{"id.in(", "request_id.in("} {
				if strings.HasPrefix(raw, prefix) && strings.HasSuffix(raw, ")") {
					addIDs(strings.TrimSuffix(strings.TrimPrefix(raw, prefix), ")"))
				}
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
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
		if recordMatchesFilter(rec, filter) {
			ret = append(ret, rec)
		}
	}
	return ret
}

func recordMatchesFilter(rec chatlog.Record, filter usageFilter) bool {
	if len(filter.RequestIDs) > 0 {
		if _, ok := filter.RequestIDs[rec.RequestID]; !ok {
			return false
		}
	}
	if filter.APIKeyID != "" && rec.VirtualKey != filter.APIKeyID {
		return false
	}
	if filter.Model != "" && recordModel(rec) != filter.Model {
		return false
	}
	if filter.Provider != "" && rec.Provider != filter.Provider {
		return false
	}
	if filter.Source != "" && recordSource(rec) != filter.Source {
		return false
	}
	if filter.AuthIndex != "" && rec.AiKey != filter.AuthIndex {
		return false
	}
	if filter.Result == "success" && !rec.Success {
		return false
	}
	if filter.Result == "failed" && rec.Success {
		return false
	}
	return true
}

func buildUsageOverview(records []chatlog.Record, filter usageFilter) api.UsageOverview {
	durationMinutes := filter.End.Sub(filter.Start).Minutes()
	if durationMinutes <= 0 {
		durationMinutes = 1
	}
	overview := api.UsageOverview{
		Timezone:      filter.Timezone,
		RangeStart:    filter.Start,
		RangeEnd:      filter.End,
		Series:        []api.UsageOverviewPoint{},
		ServiceHealth: []api.UsageServiceHealth{},
	}
	bucketSize := overviewBucketSize(filter.End.Sub(filter.Start))
	series := map[time.Time]*api.UsageOverviewPoint{}
	health := map[string]*api.UsageServiceHealth{}
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
			point = &api.UsageOverviewPoint{Timestamp: bucket}
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
			row = &api.UsageServiceHealth{Provider: rec.Provider, Model: recordModel(rec)}
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
	overview.Usage = api.UsageOverviewUsage{
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

func buildUsageAnalysis(records []chatlog.Record, filter usageFilter, names usageNames) api.UsageAnalysis {
	overview := buildUsageOverview(records, filter)
	analysis := api.UsageAnalysis{
		TokenUsage:            overview.Series,
		APIKeyComposition:     []api.UsageComposition{},
		ModelComposition:      []api.UsageComposition{},
		AuthFilesComposition:  []api.UsageComposition{},
		AIProviderComposition: []api.UsageComposition{},
		Heatmap:               []api.UsageHeatmapPoint{},
		CostBreakdown: api.UsageCostBreakdown{
			Items: []api.UsageComposition{},
		},
		ModelEfficiency: []api.UsageModelEfficiency{},
		Timezone:        filter.Timezone,
		RangeStart:      filter.Start,
		RangeEnd:        filter.End,
	}
	apiKeys := map[string]*api.UsageComposition{}
	models := map[string]*api.UsageComposition{}
	authFiles := map[string]*api.UsageComposition{}
	providers := map[string]*api.UsageComposition{}
	heatmap := map[string]*api.UsageHeatmapPoint{}
	latencies := make([]int64, 0, len(records))

	for _, rec := range records {
		tokens := recordTotalTokens(rec)
		apiKeyName := usageName(names.VirtualKeys, rec.VirtualKey)
		authFileName := usageName(names.AiKeys, rec.AiKey)
		modelName := recordModel(rec)
		addComposition(apiKeys, rec.VirtualKey, apiKeyName, apiKeyName, rec, tokens)
		addComposition(models, modelName, modelName, "", rec, tokens)
		addComposition(authFiles, rec.AiKey, authFileName, authFileName, rec, tokens)
		addComposition(providers, rec.Provider, rec.Provider, "", rec, tokens)
		weekday := rec.Timestamp.Weekday().String()
		heatKey := weekday + "\x00" + strconv.Itoa(rec.Timestamp.Hour())
		point := heatmap[heatKey]
		if point == nil {
			point = &api.UsageHeatmapPoint{Weekday: weekday, Hour: rec.Timestamp.Hour()}
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

func buildUsageEvents(records []chatlog.Record, filter usageFilter, names usageNames) *printutils.ListResult {
	total := len(records)
	start := filter.Offset
	if start > total {
		start = total
	}
	if start < 0 {
		start = 0
	}
	end := total
	if filter.Limit > 0 {
		end = start + filter.Limit
	}
	if end > total {
		end = total
	}
	ret := &printutils.ListResult{
		Data:   []jsonutils.JSONObject{},
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}
	for _, rec := range records[start:end] {
		inputTokens := rec.PromptTokens
		outputTokens := rec.CompletionTokens
		totalTokens := recordTotalTokens(rec)
		result := "success"
		if !rec.Success {
			result = "failed"
		}
		apiKeyName := usageName(names.VirtualKeys, rec.VirtualKey)
		authIndexName := usageName(names.AiKeys, rec.AiKey)
		ret.Data = append(ret.Data, jsonutils.Marshal(api.UsageEvent{
			ID:             rec.RequestID,
			RequestID:      rec.RequestID,
			Timestamp:      rec.Timestamp,
			APIKeyID:       rec.VirtualKey,
			APIKey:         rec.VirtualKey,
			APIKeyName:     apiKeyName,
			APIKeyLabel:    apiKeyName,
			Model:          recordModel(rec),
			Endpoint:       rec.Path,
			Source:         recordSource(rec),
			Provider:       rec.Provider,
			AuthIndex:      rec.AiKey,
			AuthIndexName:  authIndexName,
			AuthIndexLabel: authIndexName,
			Failed:         !rec.Success,
			Result:         result,
			StatusCode:     rec.StatusCode,
			ErrorCode:      rec.ErrorCode,
			ErrorMessage:   rec.ErrorMessage,
			LatencyMs:      rec.LatencyMs,
			Tokens: api.UsageTokens{
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				TotalTokens:  totalTokens,
			},
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
			ProjectID:    rec.ProjectID,
			ProjectName:  usageName(names.Projects, rec.ProjectID),
			DomainID:     rec.DomainID,
			DomainName:   usageName(names.Domains, rec.DomainID),
		}))
	}
	return ret
}

func buildUsageEventDistinctField(records []chatlog.Record, names usageNames, field string) (*jsonutils.JSONDict, error) {
	values := map[string]struct{}{}
	for _, rec := range records {
		switch field {
		case "model":
			addDistinct(values, recordModel(rec))
		case "provider":
			addDistinct(values, rec.Provider)
		case "source":
			addDistinct(values, recordSource(rec))
		case "api_key_name", "api_key_label":
			addDistinct(values, usageName(names.VirtualKeys, rec.VirtualKey))
		default:
			return nil, httperrors.NewInputParameterError("unsupported distinct field %s", field)
		}
	}
	ret := jsonutils.NewDict()
	ret.Set(field, jsonutils.Marshal(sortedDistinct(values)))
	return ret, nil
}

func addDistinct(items map[string]struct{}, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		items[value] = struct{}{}
	}
}

func sortedDistinct(items map[string]struct{}) []string {
	ret := make([]string, 0, len(items))
	for item := range items {
		ret = append(ret, item)
	}
	sort.Strings(ret)
	return ret
}

func buildUsageAPIKeyOptions(records []chatlog.Record, names usageNames) api.UsageAPIKeyOptions {
	items := map[string]*api.UsageAPIKeyOption{}
	for _, rec := range records {
		id := strings.TrimSpace(rec.VirtualKey)
		if id == "" {
			continue
		}
		item := items[id]
		if item == nil {
			name := usageName(names.VirtualKeys, id)
			item = &api.UsageAPIKeyOption{
				ID:         id,
				Name:       name,
				Value:      id,
				Label:      name,
				DisplayKey: id,
			}
			items[id] = item
		}
		item.RequestCount++
		item.TokenCount += recordTotalTokens(rec)
	}
	options := make([]api.UsageAPIKeyOption, 0, len(items))
	for _, item := range items {
		options = append(options, *item)
	}
	sort.Slice(options, func(i, j int) bool {
		if options[i].RequestCount != options[j].RequestCount {
			return options[i].RequestCount > options[j].RequestCount
		}
		return options[i].Name < options[j].Name
	})
	return api.UsageAPIKeyOptions{
		Overview: options,
		Analysis: append(make([]api.UsageAPIKeyOption, 0, len(options)), options...),
	}
}

func addComposition(items map[string]*api.UsageComposition, id, name, label string, rec chatlog.Record, tokens int) {
	if id == "" {
		id = "unknown"
	}
	if name == "" {
		name = id
	}
	item := items[id]
	if item == nil {
		item = &api.UsageComposition{ID: id, Name: name, Label: label}
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

func sortedCompositions(items map[string]*api.UsageComposition) []api.UsageComposition {
	ret := make([]api.UsageComposition, 0, len(items))
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

func buildModelEfficiency(items []api.UsageComposition) []api.UsageModelEfficiency {
	ret := make([]api.UsageModelEfficiency, 0, len(items))
	for _, item := range items {
		row := api.UsageModelEfficiency{
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

func buildLatencyDiagnostics(latencies []int64) api.UsageLatencyDiagnostics {
	ret := api.UsageLatencyDiagnostics{RequestCount: len(latencies)}
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

func recordSource(rec chatlog.Record) string {
	return firstNonEmpty(rec.Provider, rec.AiKey)
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

func parseNonNegativeInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}
