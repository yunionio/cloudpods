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

package server

import (
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// MCPServerBuildContext 创建 mark3labs MCPServer 时传给扩展注册回调的上下文。
type MCPServerBuildContext struct {
	// BuildInstructions 重新生成当前 Instructions（含 RegisterExtraInstructions）
	BuildInstructions func() string
	ServerName        string
	Version           string
}

var (
	mcpServerOptionFuncs []func(MCPServerBuildContext) []mcpserver.ServerOption
	mcpHooksFuncs        []func(MCPServerBuildContext, *mcpserver.Hooks)
)

// RegisterMCPServerOptions 注册额外的 NewMCPServer ServerOption（应在 init 中调用）。
// 注意：多次 WithHooks 会互相覆盖；Hooks 请用 RegisterMCPHooks。
func RegisterMCPServerOptions(fn func(MCPServerBuildContext) []mcpserver.ServerOption) {
	if fn == nil {
		return
	}
	mcpServerOptionFuncs = append(mcpServerOptionFuncs, fn)
}

// RegisterMCPHooks 向共享 Hooks 追加回调（可安全多次注册，不会互相覆盖）。
func RegisterMCPHooks(fn func(MCPServerBuildContext, *mcpserver.Hooks)) {
	if fn == nil {
		return
	}
	mcpHooksFuncs = append(mcpHooksFuncs, fn)
}

func buildRegisteredMCPServerOptions(ctx MCPServerBuildContext) []mcpserver.ServerOption {
	out := make([]mcpserver.ServerOption, 0)
	if len(mcpHooksFuncs) > 0 {
		hooks := &mcpserver.Hooks{}
		for _, fn := range mcpHooksFuncs {
			fn(ctx, hooks)
		}
		out = append(out, mcpserver.WithHooks(hooks))
	}
	for _, fn := range mcpServerOptionFuncs {
		if opts := fn(ctx); len(opts) > 0 {
			out = append(out, opts...)
		}
	}
	return out
}
