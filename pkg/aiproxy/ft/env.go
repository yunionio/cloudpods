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
	"os"
	"strings"
)

func envFirst(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func envTruthy(keys ...string) bool {
	for _, k := range keys {
		switch strings.TrimSpace(os.Getenv(k)) {
		case "1", "true", "TRUE", "yes", "YES":
			return true
		}
	}
	return false
}

func envSkipStream(explicit bool) bool {
	if explicit {
		return true
	}
	return envTruthy("AIPROXY_TEST_SKIP_STREAM", "AIPROXY_FT_SKIP_STREAM")
}

func envNonInteractive(explicit bool) bool {
	if explicit {
		return true
	}
	return envTruthy("AIPROXY_TEST_NONINTERACTIVE", "AIPROXY_FT_NONINTERACTIVE")
}

func resolveProviderFromEnv() string {
	return envFirst("AIPROXY_TEST_PROVIDER", "AIPROXY_FT_PROVIDER")
}

func resolveModelFromEnv() string {
	return envFirst("AIPROXY_TEST_MODEL", "AIPROXY_FT_MODEL")
}

func resolvePromptFromEnv() string {
	return envFirst("AIPROXY_TEST_PROMPT", "AIPROXY_FT_PROMPT")
}

func resolveApiKeyFromEnv(providerKey string) string {
	if v := envFirst("AIPROXY_TEST_API_KEY", "AIPROXY_FT_API_KEY"); v != "" {
		return v
	}
	switch providerKey {
	case "aliyun":
		return os.Getenv("DASHSCOPE_API_KEY")
	case "xiaomi":
		return os.Getenv("MIMO_API_KEY")
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "openai":
		if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
			return v
		}
		return os.Getenv("OPENAI_API_KEY")
	}
	return ""
}

func resolveAiproxyURLFromEnv() string {
	return strings.TrimRight(envFirst("AIPROXY_URL"), "/")
}
