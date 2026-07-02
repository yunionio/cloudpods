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

package ft

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	apmodules "yunion.io/x/onecloud/pkg/mcclient/modules/aiproxy"
)

func providerCreateNonInteractive(explicit bool) bool {
	if explicit {
		return true
	}
	return envTruthy("AIPROXY_PROVIDER_TEST_NONINTERACTIVE", "AIPROXY_PROVIDER_FT_NONINTERACTIVE")
}

func buildProviderConfigJSON(configJSON, baseURL string) (jsonutils.JSONObject, error) {
	if strings.TrimSpace(configJSON) != "" {
		obj, err := jsonutils.ParseString(configJSON)
		if err != nil {
			return nil, errors.Wrap(err, "parse config JSON")
		}
		return obj, nil
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, errors.Error("set --base-url or --config (or AIPROXY_PROVIDER_TEST_BASE_URL)")
	}
	return jsonutils.Marshal(map[string]string{"base_url": baseURL}), nil
}

func collectProviderCreateInputs(opts *ProviderCreateOptions) error {
	nonInteractive := providerCreateNonInteractive(opts.NonInteractive)
	suffix := time.Now().Format("20060102150405")

	if nonInteractive {
		if opts.Name == "" {
			opts.Name = fmt.Sprintf("aiproxy-provider-test-%s", suffix)
		}
		if opts.ProviderKey == "" {
			opts.ProviderKey = fmt.Sprintf("custom-test-%s", suffix)
		}
		if opts.Config == "" && opts.BaseURL == "" {
			opts.BaseURL = envFirst("AIPROXY_PROVIDER_TEST_BASE_URL", "AIPROXY_PROVIDER_FT_BASE_URL")
		}
		if opts.Config == "" {
			opts.Config = envFirst("AIPROXY_PROVIDER_TEST_CONFIG", "AIPROXY_PROVIDER_FT_CONFIG")
		}
		if !opts.Enabled {
			opts.Enabled = envTruthy("AIPROXY_PROVIDER_TEST_ENABLED") || !envTruthy("AIPROXY_PROVIDER_TEST_DISABLED")
		}
		if !opts.DeleteExisting {
			opts.DeleteExisting = envTruthy("AIPROXY_PROVIDER_TEST_DELETE_EXISTING", "AIPROXY_PROVIDER_FT_DELETE_EXISTING")
		}
		return nil
	}

	fmt.Println("=== ai_provider 创建测试 ===")
	fmt.Println("将创建自定义 ai_provider（provider_key 不可与 catalog 重复）。")
	fmt.Println()

	var err error
	if opts.Name == "" {
		opts.Name, err = promptLine("资源名称 (climc 第一个参数 NAME)", fmt.Sprintf("aiproxy-provider-test-%s", suffix), false)
		if err != nil {
			return err
		}
	}
	if opts.ProviderKey == "" {
		opts.ProviderKey, err = promptLine("provider_key (唯一标识)", opts.Name, false)
		if err != nil {
			return err
		}
	}
	if opts.Config == "" && opts.BaseURL == "" {
		opts.BaseURL, err = promptLine("config.base_url (OpenAI 兼容上游)", "https://api.openai.com", false)
		if err != nil {
			return err
		}
	}
	if !opts.Enabled && !envTruthy("AIPROXY_PROVIDER_TEST_DISABLED") {
		opts.Enabled = promptYesNo("创建后启用 (--enabled)?", true, false)
	}
	return nil
}

func deleteExistingProvider(session *mcclient.ClientSession, name string, deleteIfExists bool) error {
	if _, err := apmodules.AiProviders.Get(session, name, nil); err != nil {
		return nil
	}
	if !deleteIfExists {
		return errors.Errorf("ai_provider %s already exists; use --delete-existing or AIPROXY_PROVIDER_TEST_DELETE_EXISTING=1", name)
	}
	fmt.Printf("deleting existing ai_provider %s\n", name)
	_, err := apmodules.AiProviders.Delete(session, name, nil)
	return err
}

func RunProviderCreateTest(session *mcclient.ClientSession, opts *ProviderCreateOptions) error {
	tracker := NewResourceTracker(envKeepResources(opts.KeepResources))
	defer tracker.Cleanup(session)

	if err := collectProviderCreateInputs(opts); err != nil {
		return err
	}

	configObj, err := buildProviderConfigJSON(opts.Config, opts.BaseURL)
	if err != nil {
		return err
	}

	if err := deleteExistingProvider(session, opts.Name, opts.DeleteExisting); err != nil {
		return err
	}

	Step("create ai_provider")
	params := jsonutils.NewDict()
	params.Set("name", jsonutils.NewString(opts.Name))
	params.Set("provider_key", jsonutils.NewString(opts.ProviderKey))
	params.Set("config", configObj)
	if opts.Enabled {
		params.Set("enabled", jsonutils.JSONTrue)
	}
	if _, err := apmodules.AiProviders.Create(session, params); err != nil {
		return errors.Wrap(err, "ai-provider-create")
	}
	tracker.createdProvider = opts.Name

	Step("verify ai-provider-show")
	row, err := apmodules.AiProviders.Get(session, opts.Name, nil)
	if err != nil {
		return err
	}
	pk, _ := row.GetString("provider_key")
	if pk != opts.ProviderKey {
		return errors.Errorf("provider_key mismatch: got %s want %s", pk, opts.ProviderKey)
	}
	if opts.BaseURL != "" {
		base, _ := row.GetString("config", "base_url")
		if base != opts.BaseURL {
			return errors.Errorf("base_url mismatch: got %s want %s", base, opts.BaseURL)
		}
	}
	enabled, _ := row.Bool("enabled")
	if opts.Enabled && !enabled {
		return errors.Error("expected enabled=true")
	}
	summary := jsonutils.NewDict()
	for _, k := range []string{"id", "name", "provider_key", "enabled"} {
		if row.Contains(k) {
			val, _ := row.Get(k)
			summary.Set(k, val)
		}
	}
	if row.Contains("config") {
		val, _ := row.Get("config")
		summary.Set("config", val)
	}
	fmt.Println(summary.PrettyString())

	Step("verify ai-provider-list filter")
	query := jsonutils.NewDict()
	query.Set("provider_key", jsonutils.NewString(opts.ProviderKey))
	result, err := apmodules.AiProviders.List(session, query)
	if err != nil {
		return err
	}
	count := 0
	for _, item := range result.Data {
		name, _ := item.GetString("name")
		if name == opts.Name {
			count++
		}
	}
	if count < 1 {
		return errors.Error("ai-provider-list --provider-key did not return created row")
	}

	fmt.Println()
	fmt.Println("OK: ai_provider create test passed.")
	fmt.Printf("  name:         %s\n", opts.Name)
	fmt.Printf("  provider_key: %s\n", opts.ProviderKey)
	return nil
}
