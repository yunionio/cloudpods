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

package climcgen

import (
	"strings"

	"yunion.io/x/log"
)

// ExtraToolsFunc 通过 RegisterExtraTools 注册，用于追加 MCP tools。
type ExtraToolsFunc func() []Tool

// ExtraInstructionsFunc 通过 RegisterExtraInstructions 注册，追加到 MCP ServerInstructions。
type ExtraInstructionsFunc func() string

var (
	extraToolsFuncs        []ExtraToolsFunc
	extraInstructionsFuncs []ExtraInstructionsFunc
)

// RegisterExtraTools 注册额外工具构建回调。应在 init() 中调用；StartService 前完成注册。
func RegisterExtraTools(fn ExtraToolsFunc) {
	if fn == nil {
		return
	}
	extraToolsFuncs = append(extraToolsFuncs, fn)
}

// RegisterExtraInstructions 注册额外全局说明回调。
func RegisterExtraInstructions(fn ExtraInstructionsFunc) {
	if fn == nil {
		return
	}
	extraInstructionsFuncs = append(extraInstructionsFuncs, fn)
}

// BuildExtraTools 执行已注册的 ExtraTools 回调，合并返回。
func BuildExtraTools() []Tool {
	out := make([]Tool, 0)
	for _, fn := range extraToolsFuncs {
		tools := fn()
		if len(tools) == 0 {
			continue
		}
		out = append(out, tools...)
		for _, t := range tools {
			log.Infof("climcgen: registered extra MCP tool %s", t.GetName())
		}
	}
	return out
}

// BuildExtraInstructions 合并已注册的额外说明。
func BuildExtraInstructions() string {
	parts := make([]string, 0, len(extraInstructionsFuncs))
	for _, fn := range extraInstructionsFuncs {
		if s := strings.TrimSpace(fn()); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n\n")
}
