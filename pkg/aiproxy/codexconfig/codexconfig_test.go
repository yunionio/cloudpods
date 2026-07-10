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
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOpenAIBaseURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://gw.example.com", "https://gw.example.com/ai/openai/v1"},
		{"https://gw.example.com/", "https://gw.example.com/ai/openai/v1"},
		{"https://gw.example.com/ai/openai/v1", "https://gw.example.com/ai/openai/v1"},
		{"https://gw.example.com/openai/v1", "https://gw.example.com/ai/openai/v1"},
	}
	for _, tc := range cases {
		if got := BuildOpenAIBaseURL(tc.in); got != tc.want {
			t.Fatalf("BuildOpenAIBaseURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGenerateConfigToml(t *testing.T) {
	codexHome := t.TempDir()
	catalog := BuildCatalogFromIDs([]ModelListEntry{{ID: "my-route/gpt-4", OwnedBy: "openai"}})
	var out bytes.Buffer
	if err := GenerateConfigToml(&out, "my-route/gpt-4", "https://gw/ai/openai/v1", "aiproxy", "", codexHome, catalog); err != nil {
		t.Fatalf("GenerateConfigToml() error = %v", err)
	}
	generated := out.String()
	for _, want := range []string{
		`model = "my-route/gpt-4"`,
		`base_url = "https://gw/ai/openai/v1"`,
		`wire_api = "responses"`,
		"[model_providers.aiproxy]",
		`env_key = "OPENAI_API_KEY"`,
		filepath.Join(codexHome, codexEnvFileName),
		filepath.Join(codexHome, catalogFileName),
		"[mcp_servers.deepwiki]",
	} {
		if !strings.Contains(generated, want) {
			t.Fatalf("missing %q in generated config:\n%s", want, generated)
		}
	}
	if strings.Contains(generated, "requires_openai_auth") {
		t.Fatalf("should not require openai auth: %s", generated)
	}
}

func TestGenerateConfigTomlStdoutMode(t *testing.T) {
	catalog := BuildCatalogFromIDs([]ModelListEntry{{ID: "qwen-turbo", OwnedBy: "aliyun"}})
	var out bytes.Buffer
	if err := GenerateConfigToml(&out, "qwen-turbo", "https://gw/ai/openai/v1", "aiproxy", "", "", catalog); err != nil {
		t.Fatalf("GenerateConfigToml() error = %v", err)
	}
	generated := out.String()
	if strings.Contains(generated, "model_catalog_json") {
		t.Fatalf("stdout mode should not include model_catalog_json: %s", generated)
	}
}

func TestGenerateEnvFile(t *testing.T) {
	out := GenerateEnvFile(`vk"secret`)
	want := `export OPENAI_API_KEY="vk\"secret"`
	if out != want+"\n" {
		t.Fatalf("GenerateEnvFile = %q, want %q", out, want+"\n")
	}
}

func TestIntersectModelEntries(t *testing.T) {
	entries := []ModelListEntry{
		{ID: "route-a/m1", OwnedBy: "aliyun"},
		{ID: "route-b/m2", OwnedBy: "openai"},
		{ID: "route-a/m3", OwnedBy: "aliyun"},
	}
	allowed := map[string]struct{}{
		"route-a/m1": {},
		"route-a/m3": {},
	}
	got := intersectModelEntries(entries, allowed)
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].ID != "route-a/m1" || got[1].ID != "route-a/m3" {
		t.Fatalf("got = %#v", got)
	}
	if len(intersectModelEntries(entries, map[string]struct{}{})) != 0 {
		t.Fatal("expected empty intersection")
	}
}

func TestWriteCodexFiles(t *testing.T) {
	dir := t.TempDir()
	catalog := BuildCatalogFromIDs([]ModelListEntry{{ID: "gpt-4o", OwnedBy: "openai"}})
	if err := writeCodexFiles(dir, "gpt-4o", "https://gw/ai/openai/v1", "aiproxy", defaultProviderDisplayName, GenerateEnvFile("vk-test"), catalog); err != nil {
		t.Fatalf("writeCodexFiles() error = %v", err)
	}
	for _, name := range []string{"config.toml", catalogFileName, codexEnvFileName} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestDisplayNameFromSlug(t *testing.T) {
	cases := map[string]string{
		"gpt-5.5-codex":   "GPT 5.5 Codex",
		"deepseek-v4-pro": "Deepseek V4 Pro",
	}
	for slug, want := range cases {
		if got := DisplayNameFromSlug(slug); got != want {
			t.Fatalf("DisplayNameFromSlug(%q) = %q, want %q", slug, got, want)
		}
	}
}

func TestBuildCatalogFromIDs(t *testing.T) {
	models := BuildCatalogFromIDs([]ModelListEntry{
		{ID: "route/qwen-turbo", OwnedBy: "aliyun"},
		{ID: "gpt-4o", OwnedBy: "openai"},
	})
	if len(models) != 2 {
		t.Fatalf("len(models) = %d, want 2", len(models))
	}
	if models[0].Slug != "gpt-4o" {
		t.Fatalf("first slug = %q, want gpt-4o", models[0].Slug)
	}
	if models[0].ShellType != "unified_exec" {
		t.Fatalf("shell_type = %q", models[0].ShellType)
	}
	if models[0].TruncationPolicy.Limit != defaultCatalogTruncationLimit {
		t.Fatalf("truncation limit = %d", models[0].TruncationPolicy.Limit)
	}
	if models[0].BaseInstructions == "" {
		t.Fatal("base_instructions should not be empty")
	}
}

func TestEnsureModelInCatalog(t *testing.T) {
	catalog := EnsureModelInCatalog(nil, "custom-model")
	if len(catalog) != 1 || catalog[0].Slug != "custom-model" {
		t.Fatalf("catalog = %+v", catalog)
	}
	catalog = EnsureModelInCatalog(catalog, "custom-model")
	if len(catalog) != 1 {
		t.Fatalf("duplicate append: %+v", catalog)
	}
}

func TestWriteModelsCatalogJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), catalogFileName)
	models := BuildCatalogFromIDs([]ModelListEntry{{ID: "gpt-4o", OwnedBy: "openai"}})
	if err := WriteModelsCatalog(path, models); err != nil {
		t.Fatalf("WriteModelsCatalog() error = %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	for _, field := range []string{
		`"support_verbosity"`,
		`"supports_search_tool"`,
		`"default_verbosity"`,
	} {
		if !strings.Contains(string(raw), field) {
			t.Fatalf("catalog JSON missing %s:\n%s", field, string(raw))
		}
	}
	var parsed struct {
		Models []ModelInfo `json:"models"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(parsed.Models) != 1 || parsed.Models[0].Slug != "gpt-4o" {
		t.Fatalf("parsed models = %+v", parsed.Models)
	}
}
