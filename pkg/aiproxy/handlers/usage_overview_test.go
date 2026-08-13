package handlers

import (
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/aiproxy/chatlog"
)

func TestBuildUsageProviderNameAndAPIKeyComposition(t *testing.T) {
	now := time.Now()
	filter := usageFilter{Start: now.Add(-time.Hour), End: now, Timezone: "UTC"}
	records := []chatlog.Record{
		{AiProviderId: "p1", Provider: "openai", VirtualKey: "vk1", ModelFinal: "gpt-4", PromptTokens: 10, CompletionTokens: 20, Success: true, Timestamp: now},
		{AiProviderId: "p2", Provider: "openai", VirtualKey: "vk1", ModelFinal: "gpt-4", PromptTokens: 5, CompletionTokens: 5, Success: true, Timestamp: now},
		{Provider: "openai", VirtualKey: "vk2", ModelFinal: "gpt-4", PromptTokens: 1, CompletionTokens: 1, Success: true, Timestamp: now},
	}
	names := usageNames{
		Providers:   map[string]string{"p1": "Prod OpenAI", "p2": "Test OpenAI"},
		VirtualKeys: map[string]string{"vk1": "key-one", "vk2": "key-two"},
	}

	overview := buildUsageOverview(records, filter, names)
	if len(overview.ServiceHealth) != 3 {
		t.Fatalf("service health got %d want 3", len(overview.ServiceHealth))
	}
	healthNames := map[string]int{}
	for _, row := range overview.ServiceHealth {
		healthNames[row.ProviderName] += row.RequestCount
		if row.Provider != "openai" {
			t.Fatalf("provider key should stay openai, got %q", row.Provider)
		}
	}
	if healthNames["Prod OpenAI"] != 1 || healthNames["Test OpenAI"] != 1 || healthNames["openai"] != 1 {
		t.Fatalf("service health names: %+v", healthNames)
	}

	if len(overview.APIKeyComposition) != 2 {
		t.Fatalf("api key composition got %d want 2", len(overview.APIKeyComposition))
	}
	byKey := map[string]int{}
	byToken := map[string]int{}
	byName := map[string]string{}
	for _, item := range overview.APIKeyComposition {
		byKey[item.ID] = item.RequestCount
		byToken[item.ID] = item.TokenCount
		byName[item.ID] = item.Name
	}
	if byName["vk1"] != "key-one" || byKey["vk1"] != 2 || byToken["vk1"] != 40 {
		t.Fatalf("vk1 composition name=%s req=%d tokens=%d", byName["vk1"], byKey["vk1"], byToken["vk1"])
	}
	if byName["vk2"] != "key-two" || byKey["vk2"] != 1 || byToken["vk2"] != 2 {
		t.Fatalf("vk2 composition name=%s req=%d tokens=%d", byName["vk2"], byKey["vk2"], byToken["vk2"])
	}

	analysis := buildUsageAnalysis(records, filter, names)
	if len(analysis.AIProviderComposition) != 3 {
		t.Fatalf("provider composition got %d want 3", len(analysis.AIProviderComposition))
	}
	byProv := map[string]string{}
	for _, item := range analysis.AIProviderComposition {
		byProv[item.ID] = item.Name
	}
	if byProv["p1"] != "Prod OpenAI" || byProv["p2"] != "Test OpenAI" || byProv["openai"] != "openai" {
		t.Fatalf("provider composition: %+v", byProv)
	}
}

func TestRecordProviderIDAndName(t *testing.T) {
	names := usageNames{Providers: map[string]string{"p1": "Named"}}
	withID := chatlog.Record{AiProviderId: "p1", Provider: "openai"}
	if recordProviderID(withID) != "p1" {
		t.Fatal("expected provider id")
	}
	if recordProviderName(withID, names) != "Named" {
		t.Fatal("expected resolved provider name")
	}
	legacy := chatlog.Record{Provider: "openai"}
	if recordProviderID(legacy) != "openai" {
		t.Fatal("legacy should fall back to provider key")
	}
	if recordProviderName(legacy, names) != "openai" {
		t.Fatal("legacy name should be provider key")
	}
}

func TestRecordSourceUsesPath(t *testing.T) {
	path := "/v1/chat/completions"
	rec := chatlog.Record{Path: path, Provider: "deepseek", AiKey: "ak-1"}
	if got := recordSource(rec); got != path {
		t.Fatalf("source=%q want path %q", got, path)
	}
	empty := chatlog.Record{Provider: "deepseek", AiKey: "ak-1"}
	if got := recordSource(empty); got != "" {
		t.Fatalf("empty path should not fall back to provider, got %q", got)
	}
}

func TestBuildUsageAIKeyComposition(t *testing.T) {
	now := time.Now()
	filter := usageFilter{Start: now.Add(-time.Hour), End: now, Timezone: "UTC"}
	records := []chatlog.Record{
		{AiKey: "ak-1", VirtualKey: "vk1", ModelFinal: "gpt-4", PromptTokens: 10, CompletionTokens: 20, Success: true, Timestamp: now},
		{AiKey: "ak-1", VirtualKey: "vk2", ModelFinal: "gpt-4", PromptTokens: 5, CompletionTokens: 5, Success: true, Timestamp: now},
		{AiKey: "ak-2", VirtualKey: "vk2", ModelFinal: "gpt-4", PromptTokens: 1, CompletionTokens: 1, Success: true, Timestamp: now},
	}
	names := usageNames{
		AiKeys: map[string]string{"ak-1": "deepseek-yunion-key", "ak-2": "user1-deepseek-key"},
	}
	overview := buildUsageOverview(records, filter, names)
	if len(overview.AIKeyComposition) != 2 {
		t.Fatalf("ai key composition got %d want 2", len(overview.AIKeyComposition))
	}
	byReq := map[string]int{}
	byToken := map[string]int{}
	byName := map[string]string{}
	for _, item := range overview.AIKeyComposition {
		byReq[item.ID] = item.RequestCount
		byToken[item.ID] = item.TokenCount
		byName[item.ID] = item.Name
	}
	if byName["ak-1"] != "deepseek-yunion-key" || byReq["ak-1"] != 2 || byToken["ak-1"] != 40 {
		t.Fatalf("ak-1 composition name=%s req=%d tokens=%d", byName["ak-1"], byReq["ak-1"], byToken["ak-1"])
	}
	if byName["ak-2"] != "user1-deepseek-key" || byReq["ak-2"] != 1 || byToken["ak-2"] != 2 {
		t.Fatalf("ak-2 composition name=%s req=%d tokens=%d", byName["ak-2"], byReq["ak-2"], byToken["ak-2"])
	}
}
