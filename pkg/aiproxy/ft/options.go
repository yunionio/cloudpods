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

type ChatOptions struct {
	Provider       string `help:"provider_key from catalog"`
	Model          string `help:"model_key from catalog"`
	ApiKey         string `help:"upstream API key (or use env AIPROXY_TEST_API_KEY / provider-specific env)"`
	Prompt         string `help:"user message content"`
	KeyName        string `help:"ai_key resource name override"`
	VkName         string `help:"ai_virtual_key resource name override"`
	RoutingName    string `help:"ai_routing resource name override"`
	AiproxyURL     string `help:"aiproxy public base URL (default: AIPROXY_URL or endpoint-list)"`
	SkipStream     bool   `help:"skip streaming test"`
	NonInteractive bool   `help:"fail instead of prompting (also AIPROXY_TEST_NONINTERACTIVE=1)"`
	KeepResources  bool   `help:"keep created test resources after run (AIPROXY_TEST_KEEP_RESOURCES=1)"`
}

type AnthropicOptions struct {
	Provider        string `help:"provider_key (default anthropic; use openai for DeepSeek)"`
	Model           string `help:"model_key"`
	ApiKey          string `help:"upstream API key"`
	Prompt          string `help:"user message content"`
	KeyName         string `help:"ai_key resource name override"`
	VkName          string `help:"ai_virtual_key resource name override"`
	RoutingName     string `help:"ai_routing resource name override"`
	AiproxyURL      string `help:"aiproxy public base URL"`
	UpstreamBaseURL string `help:"optional reminder: ensure ai_provider config.base_url is set"`
	SkipStream      bool   `help:"skip streaming test"`
	NonInteractive  bool   `help:"fail instead of prompting"`
	KeepResources   bool   `help:"keep created test resources after run (AIPROXY_TEST_KEEP_RESOURCES=1)"`
}

type ProviderCreateOptions struct {
	Name           string `help:"ai_provider resource name"`
	ProviderKey    string `help:"provider_key (unique catalog identifier)"`
	BaseURL        string `help:"config.base_url for OpenAI-compatible upstream"`
	Config         string `help:"full provider config JSON (overrides --base-url)"`
	Enabled        bool   `help:"create with --enabled"`
	DeleteExisting bool   `help:"delete existing resource with same name before create"`
	NonInteractive bool   `help:"fail instead of prompting (AIPROXY_PROVIDER_TEST_NONINTERACTIVE=1)"`
	KeepResources  bool   `help:"keep created ai_provider after test (AIPROXY_TEST_KEEP_RESOURCES=1)"`
}
