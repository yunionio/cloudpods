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
	"reflect"
	"strings"

	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/structarg"
)

// TagMCP 与 climc Options 字段上的 mcp tag 对应。
// 用法：`mcp:"true"` 表示暴露给 AI/MCP；`mcp:"required"` 表示 MCP 调用时建议必填。
const TagMCP = "mcp"

// TagMCPDesc 写在 Options 结构体任意字段上（常用 `_ struct{}`），作为 MCP tool 补充说明。
// 例：`_ struct{} \`mcp-desc:"创建虚拟机最终动作..."\“
const TagMCPDesc = "mcp-desc"

type mcpFieldMeta struct {
	Required bool
}

// collectMcpDesc 读取 Options 上的 mcp-desc。
// 只看本结构体的非嵌入字段，不递归匿名嵌入，避免复用 Options（如 statistics 嵌入 List）被误注册。
func collectMcpDesc(optionsProto interface{}) string {
	if optionsProto == nil {
		return ""
	}
	return collectMcpDescFromType(reflect.TypeOf(optionsProto))
}

func collectMcpDescFromType(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			continue
		}
		if d := strings.TrimSpace(f.Tag.Get(TagMCPDesc)); d != "" {
			return d
		}
	}
	return ""
}

// collectMcpFields 从 Options 结构体读取 mcp tag，返回 CLI token -> 元信息。
// token 计算方式与 structarg 一致（token 标签或 json/字段名，再 CamelSplit 为 kebab-case）。
func collectMcpFields(optionsProto interface{}) map[string]mcpFieldMeta {
	out := make(map[string]mcpFieldMeta)
	if optionsProto == nil {
		return out
	}
	v := reflect.ValueOf(optionsProto)
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v = reflect.New(v.Type().Elem())
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return out
	}

	sets := reflectutils.FetchAllStructFieldValueSetForWrite(v)
	for i := range sets {
		info := sets[i].Info
		if info == nil {
			continue
		}
		tagMap := info.Tags
		mcpVal, ok := tagMap[TagMCP]
		if !ok || mcpVal == "" || mcpVal == "false" {
			continue
		}
		// 即便 json:"-"（Ignore）也保留：climc 仍可能用字段名作为 CLI token（如 MemSpec -> mem-spec）
		token, tokOK := tagMap["token"]
		if !tokOK {
			if jsonName := tagMap["json"]; jsonName != "" && jsonName != "-" {
				token = info.MarshalName()
			} else if alias := tagMap["alias"]; alias != "" {
				token = alias
			} else {
				// Ignore 字段的 info.Name 为空，必须用 FieldName
				token = info.FieldName
			}
		}
		if token == "" {
			continue
		}
		cliToken := utils.CamelSplit(token, "-")
		meta := mcpFieldMeta{Required: mcpVal == "required"}
		out[cliToken] = meta
		// 同时登记字段名 token，兼容 structarg 对 json:"-" 使用字段名的行为
		if fieldTok := utils.CamelSplit(info.FieldName, "-"); fieldTok != "" && fieldTok != cliToken {
			out[fieldTok] = meta
		}
		// json 名与字段名不一致时（如 Region / prefer_region）两边都登记
		if jsonName := tagMap["json"]; jsonName != "" && jsonName != "-" {
			if jsonTok := utils.CamelSplit(info.MarshalName(), "-"); jsonTok != "" && jsonTok != cliToken {
				out[jsonTok] = meta
			}
		}
	}
	return out
}

// mcpKeepArg：仅保留 positional/required，以及带 mcp tag 的可选参数。
// 无 mcp tag 的可选参数一律不暴露，避免 schema 过大。
func mcpKeepArg(fields map[string]mcpFieldMeta, arg structarg.Argument) bool {
	name := arg.Token()
	if arg.IsRequired() || arg.IsPositional() {
		return true
	}
	if name == "" || name == "help" {
		return false
	}
	_, ok := fields[name]
	return ok
}
