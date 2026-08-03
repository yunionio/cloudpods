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
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"yunion.io/x/pkg/errors"
)

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func promptSelectProvider(keys []string, providerOverride string, nonInteractive bool) (string, error) {
	if providerOverride = strings.TrimSpace(providerOverride); providerOverride != "" {
		for _, k := range keys {
			if k == providerOverride {
				return providerOverride, nil
			}
		}
		return "", errors.Errorf("ai_provider %s not in catalog", providerOverride)
	}
	providerOverride = resolveProviderFromEnv()
	if providerOverride != "" {
		for _, k := range keys {
			if k == providerOverride {
				return providerOverride, nil
			}
		}
		return "", errors.Errorf("ai_provider %s not in catalog", providerOverride)
	}
	if nonInteractive || !isInteractive() {
		return "", errors.Error("set --provider or AIPROXY_TEST_PROVIDER (or run in interactive terminal)")
	}
	fmt.Println("可用模型提供商 (catalog):")
	for i, k := range keys {
		fmt.Printf("  [%d] %s\n", i+1, k)
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("请选择序号 [1-%d] 或直接输入 provider_key: ", len(keys))
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		choice := strings.TrimSpace(line)
		if choice == "" {
			continue
		}
		if n, err := strconv.Atoi(choice); err == nil && n >= 1 && n <= len(keys) {
			return keys[n-1], nil
		}
		for _, k := range keys {
			if k == choice {
				return choice, nil
			}
		}
		fmt.Println("无效选择，请重试。")
	}
}

func promptSelectModel(models []string, providerKey, modelOverride string, nonInteractive bool) (string, error) {
	if modelOverride = strings.TrimSpace(modelOverride); modelOverride != "" {
		for _, m := range models {
			if m == modelOverride {
				return modelOverride, nil
			}
		}
		return "", errors.Errorf("model_key %s not in provider %s catalog", modelOverride, providerKey)
	}
	modelOverride = resolveModelFromEnv()
	if modelOverride != "" {
		for _, m := range models {
			if m == modelOverride {
				return modelOverride, nil
			}
		}
		return "", errors.Errorf("model_key %s not in provider %s catalog", modelOverride, providerKey)
	}

	defaultM := DefaultModelForProvider(providerKey)
	found := false
	for _, m := range models {
		if m == defaultM {
			found = true
			break
		}
	}
	if !found {
		defaultM = models[0]
	}
	if nonInteractive || !isInteractive() {
		return defaultM, nil
	}

	fmt.Printf("提供商 %s 的模型:\n", providerKey)
	for i, m := range models {
		fmt.Printf("  [%d] %s\n", i+1, m)
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("请选择序号 [1-%d] 或输入 model_key [默认: %s]: ", len(models), defaultM)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		choice := strings.TrimSpace(line)
		if choice == "" {
			return defaultM, nil
		}
		if n, err := strconv.Atoi(choice); err == nil && n >= 1 && n <= len(models) {
			return models[n-1], nil
		}
		for _, m := range models {
			if m == choice {
				return choice, nil
			}
		}
		fmt.Println("无效选择，请重试。")
	}
}

func promptApiKey(providerKey, apiKeyOverride string, nonInteractive bool) (string, error) {
	if apiKeyOverride = strings.TrimSpace(apiKeyOverride); apiKeyOverride != "" {
		return apiKeyOverride, nil
	}
	if v := resolveApiKeyFromEnv(providerKey); v != "" {
		fmt.Println("使用环境变量中的 API Key（未回显）")
		return v, nil
	}
	if nonInteractive || !isInteractive() {
		return "", errors.Errorf("未设置 API Key：--api-key 或 AIPROXY_TEST_API_KEY 或 %s 对应的环境变量", providerKey)
	}
	fmt.Printf("请输入 %s 的 API Key（不回显）: ", providerKey)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(b))
	if key == "" {
		return "", errors.Error("API Key 不能为空")
	}
	return key, nil
}

func promptRunStream(skipStream bool, nonInteractive bool) bool {
	if skipStream || envSkipStream(false) {
		return false
	}
	if v := os.Getenv("AIPROXY_TEST_SKIP_STREAM"); v == "0" {
		return true
	}
	if v := os.Getenv("AIPROXY_FT_SKIP_STREAM"); v == "0" {
		return true
	}
	if nonInteractive || !isInteractive() {
		return true
	}
	fmt.Print("是否执行流式测试 (stream=true)? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return true
	}
	switch strings.TrimSpace(strings.ToLower(line)) {
	case "n", "no":
		return false
	default:
		return true
	}
}

func promptLine(prompt, defaultVal string, nonInteractive bool) (string, error) {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	if nonInteractive || !isInteractive() {
		if defaultVal == "" {
			return "", errors.Errorf("empty input for %s in non-interactive mode", prompt)
		}
		return defaultVal, nil
	}
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(line)
	if val == "" {
		val = defaultVal
	}
	if val == "" {
		return "", errors.Errorf("empty input for %s", prompt)
	}
	return val, nil
}

func promptYesNo(prompt string, defaultYes bool, nonInteractive bool) bool {
	if nonInteractive || !isInteractive() {
		return defaultYes
	}
	if defaultYes {
		fmt.Printf("%s [Y/n]: ", prompt)
	} else {
		fmt.Printf("%s [y/N]: ", prompt)
	}
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}
	switch strings.TrimSpace(strings.ToLower(line)) {
	case "n", "no":
		return false
	case "y", "yes":
		return true
	default:
		return defaultYes
	}
}
