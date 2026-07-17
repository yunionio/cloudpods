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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/cmd/climc/shell"
)

// maxSchemaProperties 限制生成的参数数量，避免 BaseListOptions 等过大 schema 淹没模型
const maxSchemaProperties = 48

func newOptionsInstance(cmd shell.CMD) (interface{}, error) {
	if cmd.Options == nil {
		return nil, fmt.Errorf("command %s has nil Options", cmd.Command)
	}
	t := reflect.TypeOf(cmd.Options)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("command %s Options is not a struct: %s", cmd.Command, t.Kind())
	}
	return reflect.New(t).Interface(), nil
}

func newArgumentParser(cmd shell.CMD) (*structarg.ArgumentParser, interface{}, error) {
	optPtr, err := newOptionsInstance(cmd)
	if err != nil {
		return nil, nil, err
	}
	parser, err := structarg.NewArgumentParserWithHelp(optPtr, cmd.Command, cmd.Desc, "")
	if err != nil {
		return nil, nil, fmt.Errorf("build argument parser for %s: %w", cmd.Command, err)
	}
	return parser, optPtr, nil
}

func buildInputSchema(cmd shell.CMD) (json.RawMessage, error) {
	parser, optPtr, err := newArgumentParser(cmd)
	if err != nil {
		return nil, err
	}

	mcpFields := collectMcpFields(optPtr)

	properties := map[string]interface{}{}
	required := make([]string, 0)
	requiredSet := map[string]bool{}

	addRequired := func(name string) {
		if name == "" || requiredSet[name] {
			return
		}
		requiredSet[name] = true
		required = append(required, name)
	}

	addArg := func(arg structarg.Argument) {
		name := arg.Token()
		if name == "" || name == "help" {
			return
		}
		if _, exists := properties[name]; exists {
			return
		}
		prop := map[string]interface{}{
			"description": strings.TrimSpace(arg.HelpString("")),
		}
		switch {
		case !arg.NeedData():
			prop["type"] = "boolean"
		case arg.IsMulti():
			prop["type"] = "array"
			items := map[string]interface{}{"type": "string"}
			if choices := argChoices(arg); len(choices) > 0 {
				items["enum"] = choices
			}
			prop["items"] = items
		default:
			prop["type"] = "string"
		}
		if choices := argChoices(arg); len(choices) > 0 && !arg.IsMulti() {
			prop["enum"] = choices
		}
		if arg.IsPositional() {
			meta := arg.MetaVar()
			if meta != "" {
				desc, _ := prop["description"].(string)
				if desc != "" {
					prop["description"] = fmt.Sprintf("%s (positional: %s)", desc, meta)
				} else {
					prop["description"] = fmt.Sprintf("positional argument %s", meta)
				}
			}
		}
		properties[name] = prop
		if arg.IsRequired() || arg.IsPositional() {
			addRequired(name)
		}
		if meta, ok := mcpFields[name]; ok && meta.Required {
			addRequired(name)
		}
	}

	for _, arg := range parser.GetPosArgs() {
		addArg(arg)
	}
	for _, arg := range parser.GetOptArgs() {
		if arg.IsRequired() {
			addArg(arg)
		}
	}

	isList := strings.HasSuffix(cmd.Command, "-list")
	isCreate := cmd.Command == "server-create"
	for _, arg := range parser.GetOptArgs() {
		if arg.IsRequired() || arg.Token() == "help" {
			continue
		}
		if len(properties) >= maxSchemaProperties {
			break
		}
		if mcpKeepArg(mcpFields, arg) {
			addArg(arg)
		}
	}

	// 强化 create 关键字段说明，降低调用门槛
	if isCreate {
		if prop, ok := properties["disk"].(map[string]interface{}); ok {
			prop["description"] = "系统盘描述，数组。公有云示例：[\"size=40g,image=<镜像ID>,backend=cloud_essd\"]；backend 必须来自 climc_cloud_region_capability 的 storage_types2；ISO 请用 cdrom 而不是 disk.image"
			properties["disk"] = prop
		}
		if prop, ok := properties["net"].(map[string]interface{}); ok {
			prop["description"] = "网络描述，可省略。省略或 [] 时自动 random（等价 API nets:[{exit:false}]）由调度器选网；指定时示例：[\"<网络ID>\"]"
			properties["net"] = prop
		}
		if prop, ok := properties["cdrom"].(map[string]interface{}); ok {
			prop["description"] = "ISO/光驱镜像 ID；系统盘不要挂 ISO 镜像"
			properties["cdrom"] = prop
		}
	}

	// list 命令确保 scope 出现在 schema 中，并强化 provider 说明
	if isList {
		if prop, ok := properties["scope"].(map[string]interface{}); ok {
			prop["description"] = "resource scope；省略时 MCP 默认注入 max"
			properties["scope"] = prop
		} else {
			properties["scope"] = map[string]interface{}{
				"type":        "string",
				"description": "resource scope；省略时 MCP 默认注入 max",
				"enum":        []string{"system", "domain", "project", "user", "max"},
			}
		}
		if prop, ok := properties["provider"].(map[string]interface{}); ok {
			prop["description"] = "云厂商过滤；用户指定阿里云/AWS 等公有云时必须传，例如 [\"Aliyun\"]，不要省略"
			properties["provider"] = prop
		}
	}

	if cmd.Command == "cloud-region-list" {
		if prop, ok := properties["usable"].(map[string]interface{}); ok {
			prop["description"] = "创建虚拟机时必须为 true，只返回网络可用区域；省略时 MCP 默认注入 true"
			properties["usable"] = prop
		} else {
			properties["usable"] = map[string]interface{}{
				"type":        "boolean",
				"description": "创建虚拟机时必须为 true，只返回网络可用区域；省略时 MCP 默认注入 true",
			}
		}
	}

	if cmd.Command == "cached-image-list" {
		if prop, ok := properties["provider"].(map[string]interface{}); ok {
			prop["description"] = "公有云选镜像必传，例如 [\"Aliyun\"]；不传会混入其他云（如 AWS）镜像"
			properties["provider"] = prop
		}
		if prop, ok := properties["region"].(map[string]interface{}); ok {
			prop["description"] = "必须传 climc_cloud_region_list 返回的 id（UUID）。禁止传 cn-shanghai / Aliyun/cn-shanghai 这类外部 region code"
			properties["region"] = prop
		}
		if prop, ok := properties["image-type"].(map[string]interface{}); ok {
			prop["description"] = "镜像类型；创建系统盘优先 system，避免误用 ISO/驱动盘"
			properties["image-type"] = prop
		}
	}

	if cmd.Command == "server-sku-list" {
		if prop, ok := properties["spec"].(map[string]interface{}); ok {
			prop["description"] = "口语规格，优先使用。例：2c2g、2核2G、4C8G；自动转为 cpu + mem(MB)。用户说「2核2G」就传 spec=\"2c2g\""
			properties["spec"] = prop
		} else {
			properties["spec"] = map[string]interface{}{
				"type":        "string",
				"description": "口语规格，优先使用。例：2c2g、2核2G、4C8G；自动转为 cpu + mem(MB)",
			}
		}
		if prop, ok := properties["cpu"].(map[string]interface{}); ok {
			prop["description"] = "CPU 核数；与 mem 联用。有口语规格时优先用 spec"
			properties["cpu"] = prop
		}
		if prop, ok := properties["mem"].(map[string]interface{}); ok {
			prop["description"] = "内存 MB；2G=2048。有口语规格时优先用 spec"
			properties["mem"] = prop
		}
		if prop, ok := properties["postpaid-status"].(map[string]interface{}); ok {
			prop["description"] = "按量付费状态；创建时优先 available，避免选到 soldout"
			properties["postpaid-status"] = prop
		}
	}

	// 认证：请使用连接 Header（X-Auth-Token / AK+SK / X-API-Key），不要在工具参数里传密钥。

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func argChoices(arg structarg.Argument) []string {
	type chooser interface {
		Choices() []string
	}
	if c, ok := arg.(chooser); ok {
		return c.Choices()
	}
	return nil
}

func toolNameFromCommand(command string) string {
	return "climc_" + strings.ReplaceAll(command, "-", "_")
}

func buildDescription(cmd shell.CMD) string {
	desc := strings.TrimSpace(cmd.Desc)
	if desc == "" {
		desc = fmt.Sprintf("Execute climc command %s", cmd.Command)
	}
	desc = fmt.Sprintf("[climc %s] %s", cmd.Command, desc)
	if mcp := collectMcpDesc(cmd.Options); mcp != "" {
		desc += "。" + mcp
	}
	return desc
}
