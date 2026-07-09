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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/aiproxy/ft"
	"yunion.io/x/onecloud/pkg/mcclient"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
)

const (
	defaultProviderDisplayName = "Cloudpods AI Gateway"
	codexEnvKeyName            = "OPENAI_API_KEY"
	codexEnvFileName           = "aiproxy.env"
)

// Options are climc flags for ai-codex-config.
type Options struct {
	VirtualKey   string `help:"ai_virtual_key name or id"`
	Model        string `help:"client model id (flat or route/catalog)"`
	Routing      string `help:"ai_routing name or id; uses model_key when --model is omitted"`
	AiproxyURL   string `help:"aiproxy base URL override (default: routing ai_proxy_node access_address, else AIPROXY_URL or endpoint-list)"`
	CodexHome    string `help:"write config.toml, models_catalog.json, and aiproxy.env to this directory"`
	ListModels   bool   `help:"list model ids visible to the virtual key and exit"`
	ProviderName string `help:"model_provider section name in config.toml" default:"aiproxy"`
}

// BuildOpenAIBaseURL returns the OpenAI-compatible base URL for Codex (…/ai/openai/v1).
func BuildOpenAIBaseURL(aiproxyURL string) string {
	base := strings.TrimRight(strings.TrimSpace(aiproxyURL), "/")
	if base == "" {
		return ""
	}
	if strings.HasSuffix(base, "/ai/openai/v1") {
		return base
	}
	if strings.HasSuffix(base, "/openai/v1") {
		return strings.TrimSuffix(base, "/openai/v1") + "/ai/openai/v1"
	}
	return base + "/ai/openai/v1"
}

func openAIModelsURL(aiproxyURL string) string {
	return BuildOpenAIBaseURL(aiproxyURL) + "/models"
}

// GenerateConfigToml writes Codex config.toml aligned with moon-bridge output.
// When codexHome is non-empty, it also writes models_catalog.json and sets model_catalog_json.
func GenerateConfigToml(w io.Writer, model, baseURL, providerName, displayName, codexHome string, catalog []ModelInfo) error {
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		providerName = "aiproxy"
	}
	if displayName = strings.TrimSpace(displayName); displayName == "" {
		displayName = defaultProviderDisplayName
	}
	model = strings.TrimSpace(model)
	baseURL = strings.TrimSpace(baseURL)
	codexHome = strings.TrimSpace(codexHome)

	fmt.Fprintf(w, "model = %q\n", model)
	fmt.Fprintf(w, "model_provider = %q\n", providerName)
	if ctxWin := CatalogContextWindow(catalog, model); ctxWin > 0 {
		fmt.Fprintf(w, "model_context_window = %d\n", ctxWin)
	}

	if codexHome != "" {
		catalogPath := filepath.Join(codexHome, catalogFileName)
		if err := WriteModelsCatalog(catalogPath, catalog); err != nil {
			return errors.Wrap(err, "write models catalog")
		}
		fmt.Fprintf(w, "model_catalog_json = %q\n", catalogPath)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "[model_providers.%s]\n", providerName)
	fmt.Fprintf(w, "name = %q\n", displayName)
	fmt.Fprintf(w, "base_url = %q\n", baseURL)
	fmt.Fprintln(w, `wire_api = "responses"`)
	fmt.Fprintf(w, "env_key = %q\n", codexEnvKeyName)
	fmt.Fprintf(w, "env_key_instructions = %q\n", envKeyInstructions(codexHome))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "[mcp_servers.deepwiki]")
	fmt.Fprintln(w, `url = "https://mcp.deepwiki.com/mcp"`)
	fmt.Fprintln(w, "startup_timeout_sec = 3600")
	fmt.Fprintln(w, "tool_timeout_sec = 3600")
	return nil
}

func envKeyInstructions(codexHome string) string {
	codexHome = strings.TrimSpace(codexHome)
	if codexHome != "" {
		return fmt.Sprintf("Run: source %s", filepath.Join(codexHome, codexEnvFileName))
	}
	return fmt.Sprintf("Run: source %s (or export %s)", codexEnvFileName, codexEnvKeyName)
}

// GenerateEnvFile returns shell snippet that exports OPENAI_API_KEY for Codex.
func GenerateEnvFile(virtualKey string) string {
	virtualKey = strings.TrimSpace(virtualKey)
	return fmt.Sprintf("export %s=%q\n", codexEnvKeyName, virtualKey)
}

// Run resolves inputs and prints or writes Codex config files.
func Run(session *mcclient.ClientSession, opts *Options) error {
	if opts == nil {
		return errors.Error("options is nil")
	}
	vkName := strings.TrimSpace(opts.VirtualKey)
	if vkName == "" {
		return errors.Error("--virtual-key is required")
	}

	aiproxyURL, err := ft.ResolveAiproxyURLForRouting(session, opts.AiproxyURL, opts.Routing)
	if err != nil {
		return err
	}
	openaiBase := BuildOpenAIBaseURL(aiproxyURL)
	if openaiBase == "" {
		return errors.Error("cannot resolve aiproxy OpenAI base URL")
	}

	virtualKey, err := fetchVirtualKey(session, vkName)
	if err != nil {
		return err
	}

	if opts.ListModels {
		entries, err := listModelEntries(session, aiproxyURL, virtualKey)
		if err != nil {
			return err
		}
		entries, err = filterModelEntriesForRouting(session, opts.Routing, entries)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("No models available for this virtual key.")
			return nil
		}
		for _, entry := range entries {
			fmt.Println(entry.ID)
		}
		return nil
	}

	model, err := resolveModel(session, opts, virtualKey, aiproxyURL)
	if err != nil {
		return err
	}

	entries, err := listModelEntries(session, aiproxyURL, virtualKey)
	if err != nil {
		return err
	}
	entries, err = filterModelEntriesForRouting(session, opts.Routing, entries)
	if err != nil {
		return err
	}
	catalog := EnsureModelInCatalog(BuildCatalogFromIDs(entries), model)

	providerName := strings.TrimSpace(opts.ProviderName)
	if providerName == "" {
		providerName = "aiproxy"
	}
	codexHome := strings.TrimSpace(opts.CodexHome)
	envFile := GenerateEnvFile(virtualKey)

	if codexHome == "" {
		var configBuf bytes.Buffer
		if err := GenerateConfigToml(&configBuf, model, openaiBase, providerName, defaultProviderDisplayName, "", catalog); err != nil {
			return err
		}
		fmt.Println("# config.toml")
		fmt.Print(configBuf.String())
		fmt.Println()
		fmt.Printf("# %s\n", codexEnvFileName)
		fmt.Print(envFile)
		return nil
	}

	if err := writeCodexFiles(codexHome, model, openaiBase, providerName, defaultProviderDisplayName, envFile, catalog); err != nil {
		return err
	}
	envPath := filepath.Join(codexHome, codexEnvFileName)
	catalogPath := filepath.Join(codexHome, catalogFileName)
	fmt.Printf("Wrote %s/config.toml, %s, and %s (model=%q, base_url=%q)\n", codexHome, catalogPath, envPath, model, openaiBase)
	printCodexStartScript(codexHome, envPath)
	return nil
}

func printCodexStartScript(codexHome, envPath string) {
	fmt.Println()
	fmt.Println("# 4. 加载 OPENAI_API_KEY（virtual_key）并以独立 CODEX_HOME 启动 Codex，工作目录为当前项目")
	fmt.Printf("source %q && CODEX_HOME=%q codex --cd \"$PWD\"\n", envPath, codexHome)
}

func fetchVirtualKey(session *mcclient.ClientSession, nameOrID string) (string, error) {
	obj, err := apmodules.AiVirtualKeys.Get(session, nameOrID, nil)
	if err != nil {
		return "", errors.Wrapf(err, "ai-virtual-key-show %s", nameOrID)
	}
	vk, _ := obj.GetString("virtual_key")
	vk = strings.TrimSpace(vk)
	if vk == "" {
		return "", errors.Errorf("empty virtual_key on ai_virtual_key %s", nameOrID)
	}
	return vk, nil
}

func resolveRoutingModelKey(session *mcclient.ClientSession, routing string) (string, error) {
	routing = strings.TrimSpace(routing)
	if routing == "" {
		return "", nil
	}
	obj, err := apmodules.AiRoutings.Get(session, routing, nil)
	if err != nil {
		return "", errors.Wrapf(err, "ai-routing-show %s", routing)
	}
	mk, _ := obj.GetString("model_key")
	mk = strings.TrimSpace(mk)
	if mk == "" {
		return "", errors.Errorf("ai_routing %s has empty model_key", routing)
	}
	return mk, nil
}

type modelsListResponse struct {
	Data []struct {
		ID      string `json:"id"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

func listModelEntries(session *mcclient.ClientSession, aiproxyURL, virtualKey string) ([]ModelListEntry, error) {
	url := openAIModelsURL(aiproxyURL)
	client := session.GetClient().GetClient()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+virtualKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "GET /ai/openai/v1/models")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GET /ai/openai/v1/models: HTTP %d: %s", resp.StatusCode, truncateBody(body))
	}
	var out modelsListResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, errors.Wrap(err, "decode models list")
	}
	entries := make([]ModelListEntry, 0, len(out.Data))
	for _, item := range out.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		entries = append(entries, ModelListEntry{
			ID:      id,
			OwnedBy: strings.TrimSpace(item.OwnedBy),
		})
	}
	return entries, nil
}

func listModelIDs(session *mcclient.ClientSession, aiproxyURL, virtualKey string) ([]string, error) {
	entries, err := listModelEntries(session, aiproxyURL, virtualKey)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids, nil
}

func resolveModel(session *mcclient.ClientSession, opts *Options, virtualKey, aiproxyURL string) (string, error) {
	if model := strings.TrimSpace(opts.Model); model != "" {
		return model, nil
	}
	if mk, err := resolveRoutingModelKey(session, opts.Routing); err != nil {
		return "", err
	} else if mk != "" {
		return mk, nil
	}
	ids, err := listModelIDs(session, aiproxyURL, virtualKey)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", errors.Error("no models available; specify --model or --routing")
	}
	return ids[0], nil
}

func writeCodexFiles(codexHome, model, baseURL, providerName, displayName, envFile string, catalog []ModelInfo) error {
	if err := os.MkdirAll(codexHome, 0755); err != nil {
		return errors.Wrapf(err, "mkdir %s", codexHome)
	}
	configPath := filepath.Join(codexHome, "config.toml")
	var configBuf bytes.Buffer
	if err := GenerateConfigToml(&configBuf, model, baseURL, providerName, displayName, codexHome, catalog); err != nil {
		return err
	}
	if err := os.WriteFile(configPath, configBuf.Bytes(), 0644); err != nil {
		return errors.Wrapf(err, "write %s", configPath)
	}
	envPath := filepath.Join(codexHome, codexEnvFileName)
	if err := os.WriteFile(envPath, []byte(envFile), 0600); err != nil {
		return errors.Wrapf(err, "write %s", envPath)
	}
	return nil
}

func truncateBody(b []byte) string {
	const max = 512
	s := strings.TrimSpace(string(b))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
