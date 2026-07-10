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

package models

import (
	"context"
	"testing"
)

func TestCatalogPartMatches(t *testing.T) {
	mdl := &SAiModel{ModelKey: "claude-sonnet-4-6"}
	entry := &SAiRoutingModel{ModelPattern: "fast"}
	if !catalogPartMatches(entry, "fast", mdl) {
		t.Fatal("expected entry pattern match")
	}
	if catalogPartMatches(entry, "slow", mdl) {
		t.Fatal("unexpected pattern match")
	}
	entry = &SAiRoutingModel{}
	if !catalogPartMatches(entry, "claude-sonnet-4-6", mdl) {
		t.Fatal("expected catalog model_key match")
	}
}

func TestPickAiRoutingModelFromEntriesDefaultPriority(t *testing.T) {
	routing := &SAiRouting{ModelKey: "claude"}
	entries := []SAiRoutingModel{
		{Priority: 50, AiModelId: "m1"},
		{Priority: 10, AiModelId: "m2"},
	}
	modelsById := map[string]*SAiModel{
		"m1": {ModelKey: "opus"},
		"m2": {ModelKey: "sonnet"},
	}
	picked, err := pickAiRoutingModelFromEntries(context.Background(), routing, "", "claude", false, nil, entries, modelsById)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if picked == nil || picked.AiModelId != "m2" {
		t.Fatalf("expected highest-priority entry m2, got %#v", picked)
	}
}

func TestPickAiRoutingModelFromEntriesHierarchicalCatalogPart(t *testing.T) {
	routing := &SAiRouting{ModelKey: "claude"}
	entries := []SAiRoutingModel{
		{Priority: 10, AiModelId: "m1"},
		{Priority: 20, AiModelId: "m2"},
	}
	modelsById := map[string]*SAiModel{
		"m1": {ModelKey: "claude-opus-4-8"},
		"m2": {ModelKey: "claude-sonnet-4-6"},
	}
	picked, err := pickAiRoutingModelFromEntries(context.Background(), routing, "claude-sonnet-4-6", "claude-sonnet-4-6", true, nil, entries, modelsById)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if picked == nil || picked.AiModelId != "m2" {
		t.Fatalf("expected sonnet entry, got %#v", picked)
	}
}

func TestPickAiRoutingModelFromEntriesFlatPattern(t *testing.T) {
	routing := &SAiRouting{ModelPattern: "qwen-*"}
	entries := []SAiRoutingModel{
		{Priority: 10, ModelPattern: "qwen-turbo"},
		{Priority: 20, ModelPattern: "qwen-plus"},
	}
	picked, err := pickAiRoutingModelFromEntries(context.Background(), routing, "qwen-turbo", "qwen-turbo", false, nil, entries, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if picked == nil || picked.ModelPattern != "qwen-turbo" {
		t.Fatalf("expected qwen-turbo entry, got %#v", picked)
	}
}

func TestPickAiRoutingModelFromEntriesHierarchicalNotFound(t *testing.T) {
	routing := &SAiRouting{ModelKey: "claude"}
	entries := []SAiRoutingModel{
		{Priority: 10, AiModelId: "m1"},
	}
	modelsById := map[string]*SAiModel{
		"m1": {ModelKey: "claude-opus-4-8"},
	}
	_, err := pickAiRoutingModelFromEntries(context.Background(), routing, "missing", "missing", true, nil, entries, modelsById)
	if err == nil {
		t.Fatal("expected not found error")
	}
}
