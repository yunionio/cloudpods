// Copyright 2026 Yunion
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

package dashboardcodegen

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	llm_client "yunion.io/x/onecloud/pkg/llm/drivers/llm_client"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	// CodegenMaxTokens caps completion size so thinking-heavy models cannot run until aiproxy's
	// non-stream upstream timeout (120s) with empty/useless content.
	CodegenMaxTokens = 4096
)

// Generate calls the default MCP agent's LLM (no tools) and returns validated JS without saving.
func Generate(ctx context.Context, userCred mcclient.TokenCredential, input api.DashboardPanelGenerateInput) (*api.DashboardPanelGenerateOutput, error) {
	if err := ValidatePromptTitle(input.Prompt, input.Title); err != nil {
		return nil, err
	}
	scope, err := NormalizeScope(input.Scope)
	if err != nil {
		return nil, err
	}

	userId := ""
	if userCred != nil {
		userId = userCred.GetUserId()
	}
	if err := AcquireGenerate(userId); err != nil {
		return nil, err
	}
	defer ReleaseGenerate(userId)

	agent, err := models.GetMCPAgentManager().GetDefaultAgent(ctx, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "GetDefaultAgent")
	}
	if agent == nil {
		return nil, errors.Error("no default MCP agent set (set one agent with default_agent=true)")
	}
	if strings.ToLower(strings.TrimSpace(agent.LLMDriver)) != string(api.LLM_CLIENT_OPENAI) {
		return nil, errors.Errorf("dashboard panel generate requires openai-compatible agent, got %q", agent.LLMDriver)
	}

	msgs := make([]llm_client.OpenAIChatMessage, 0, 2+len(input.History))
	msgs = append(msgs, llm_client.OpenAIChatMessage{
		Role:    "system",
		Content: api.DASHBOARD_PANEL_CODEGEN_SYSTEM_PROMPT,
	})

	histLimit := api.DashboardPanelMaxHistoryItems
	start := 0
	if len(input.History) > histLimit {
		start = len(input.History) - histLimit
	}
	for _, h := range input.History[start:] {
		role := strings.ToLower(strings.TrimSpace(h.Role))
		content := strings.TrimSpace(h.Content)
		if content == "" {
			continue
		}
		switch role {
		case "assistant":
			msgs = append(msgs, llm_client.OpenAIChatMessage{Role: "assistant", Content: content})
		case "system":
			continue
		default:
			msgs = append(msgs, llm_client.OpenAIChatMessage{Role: "user", Content: content})
		}
	}

	scopeHint := "运行时由前端视图注入 ctx.scope（随 system/domain/project 视图切换），代码必须用 ctx.scope，禁止写死某一作用域"
	if scope != "" {
		scopeHint = fmt.Sprintf("请求侧提示作用域为 %s；仍须用 ctx.scope 取实际 scope，并按 usage_key 前缀规则适配", scope)
	}
	userPrompt := fmt.Sprintf("作用域：%s\n用户请求（JSON，含需求描述与内容区参考宽高 px）：\n%s", scopeHint, input.Prompt)
	msgs = append(msgs, llm_client.OpenAIChatMessage{Role: "user", Content: userPrompt})

	// Disable thinking + max_tokens: default agent model (e.g. Qwen thinking) otherwise
	// burns the whole aiproxy non-stream 120s budget and returns 502 context deadline exceeded.
	resp, err := llm_client.ChatWithOptions(ctx, agent, msgs, nil, &llm_client.ChatCompletionOptions{
		MaxTokens:       CodegenMaxTokens,
		DisableThinking: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "llm chat")
	}
	raw := ""
	if resp != nil {
		raw = resp.GetContent()
	}
	source := ExtractSource(raw)
	result, err := ValidateSource(source, nil)
	if err != nil {
		log.Infof("dashboard panel generate validation failed user=%s: %v", userId, err)
		return nil, errors.Wrap(err, "validate generated source")
	}

	out := &api.DashboardPanelGenerateOutput{
		Source:        result.Source,
		Entry:         result.Entry,
		Scope:         scope,
		ContentSha256: result.ContentSha256,
		ApiAllowlist:  result.ApiAllowlist,
		Warnings:      result.Warnings,
	}
	return out, nil
}
