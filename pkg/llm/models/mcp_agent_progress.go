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

package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// progressLabelCache 缓存 list 结果中的 id→可读名称，供后续进度文案使用。
type progressLabelCache struct {
	regions map[string]string
}

func newProgressLabelCache() *progressLabelCache {
	return &progressLabelCache{regions: make(map[string]string)}
}

func (c *progressLabelCache) rememberFromTool(toolName, resultText string) {
	if c == nil || isToolResultError(resultText) {
		return
	}
	name := strings.TrimPrefix(toolName, "climc_")
	if strings.Contains(name, "cloud_region_list") {
		c.rememberRegions(resultText)
	}
}

func (c *progressLabelCache) rememberRegions(resultText string) {
	if c.regions == nil {
		c.regions = make(map[string]string)
	}
	items, _ := extractListItems(resultText)
	for _, item := range items {
		id := jsonString(item, "id")
		name := jsonString(item, "name")
		if id == "" || name == "" {
			continue
		}
		ext := jsonString(item, "external_id")
		label := name
		if ext != "" && !strings.EqualFold(ext, name) {
			label = fmt.Sprintf("%s（%s）", name, ext)
		}
		c.regions[id] = label
		c.regions[name] = label
		if ext != "" {
			c.regions[ext] = label
		}
	}
}

func (c *progressLabelCache) regionLabel(idOrName string) string {
	if idOrName == "" {
		return ""
	}
	if c != nil && c.regions != nil {
		if v := c.regions[idOrName]; v != "" {
			return v
		}
	}
	return idOrName
}

// summarizeToolProgress 将工具调用结果整理成面向用户的进度文案（逐项展示资源，而非工具名）。
func summarizeToolProgress(toolName string, args map[string]interface{}, resultText string, labels *progressLabelCache) string {
	name := strings.TrimPrefix(toolName, "climc_")
	if isToolResultError(resultText) {
		return fmt.Sprintf("✗ %s失败：%s\n", progressLabel(name), truncateRunes(stripMCPHint(resultText), 180))
	}

	switch {
	case name == "docs_search" || strings.HasSuffix(name, "docs_search"):
		return formatResourceListProgress("文档", resultText, []string{"name", "title", "path"})

	case name == "docs_get" || strings.HasSuffix(name, "docs_get"):
		path := firstArg(args, "path", "PATH")
		if path != "" {
			return fmt.Sprintf("✓ 已阅读文档：%s\n", path)
		}
		return "✓ 已阅读文档\n"

	case strings.Contains(name, "cloud_region_capability"):
		id := firstArg(args, "id", "ID", "name")
		region := labels.regionLabel(id)
		types := extractStorageTypeHints(resultText)
		if region != "" && types != "" {
			return fmt.Sprintf("✓ 区域 %s 可用磁盘类型：%s\n", region, types)
		}
		if types != "" {
			return fmt.Sprintf("✓ 可用磁盘类型：%s\n", types)
		}
		return fmt.Sprintf("✓ 已查询区域能力%s\n", paren(region))

	case strings.Contains(name, "cloud_region_list"):
		return formatResourceListProgress("区域", resultText, []string{"name", "external_id", "id"})

	case strings.Contains(name, "cached_image_list"), strings.Contains(name, "image_list"):
		return formatResourceListProgress("镜像", resultText, []string{"name", "os_type", "os_distribution", "id"})

	case strings.Contains(name, "server_sku_list"):
		return formatResourceListProgress("套餐", resultText, []string{"name", "instance_type_category", "cpu_core_count", "memory_size_mb", "id"})

	case strings.Contains(name, "network_list"):
		return formatResourceListProgress("网络", resultText, []string{"name", "guest_ip_prefix", "vpc", "id"})

	case strings.Contains(name, "vpc_list"):
		return formatResourceListProgress("VPC", resultText, []string{"name", "cidr_block", "id"})

	case strings.Contains(name, "storage_list"):
		return formatResourceListProgress("存储", resultText, []string{"name", "storage_type", "capacity", "id"})

	case strings.Contains(name, "server_list"):
		return formatResourceListProgress("虚拟机", resultText, []string{"name", "status", "id"})

	case strings.Contains(name, "server_create"):
		return formatServerCreateProgress(args, resultText, labels)

	case strings.Contains(name, "server_show"):
		return formatSingleResourceProgress("虚拟机详情", resultText, []string{"name", "status", "id"})

	case strings.HasPrefix(name, "server_"):
		id := firstArg(args, "id", "ID", "name")
		action := strings.TrimPrefix(name, "server_")
		return fmt.Sprintf("✓ 虚拟机%s%s\n", actionLabel(action), paren(id))

	default:
		return formatGenericProgress(name, args, resultText)
	}
}

func isToolResultError(resultText string) bool {
	s := strings.TrimSpace(resultText)
	return strings.Contains(s, "调用失败") ||
		strings.Contains(s, "返回错误") ||
		strings.HasPrefix(s, "工具 ") && strings.Contains(s, "失败")
}

func progressLabel(toolName string) string {
	switch {
	case strings.Contains(toolName, "cloud_region_list"):
		return "查询区域"
	case strings.Contains(toolName, "cloud_region_capability"):
		return "查询区域能力"
	case strings.Contains(toolName, "cached_image"):
		return "查询镜像"
	case strings.Contains(toolName, "image_list"):
		return "查询镜像"
	case strings.Contains(toolName, "server_sku"):
		return "查询套餐"
	case strings.Contains(toolName, "network_list"):
		return "查询网络"
	case strings.Contains(toolName, "vpc_list"):
		return "查询 VPC"
	case strings.Contains(toolName, "server_create"):
		return "创建虚拟机"
	default:
		return toolName
	}
}

func actionLabel(action string) string {
	switch action {
	case "start":
		return "已启动"
	case "stop":
		return "已停止"
	case "restart":
		return "已重启"
	case "delete":
		return "已删除"
	case "set_password", "set-password":
		return "已重置密码"
	default:
		return "操作完成"
	}
}

func formatServerCreateProgress(args map[string]interface{}, resultText string, labels *progressLabelCache) string {
	var b strings.Builder
	b.WriteString("✓ 选用资源创建虚拟机：")
	parts := make([]string, 0, 6)
	if v := firstArg(args, "name", "NAME"); v != "" {
		if isTruthy(args["generate-name"]) || isTruthy(args["generate_name"]) || isTruthy(args["GenerateName"]) {
			parts = append(parts, "名称模板="+v+"（自动去重）")
		} else {
			parts = append(parts, "名称="+v)
		}
	}
	if v := firstArg(args, "hypervisor"); v != "" {
		parts = append(parts, "平台="+v)
	}
	if v := firstArg(args, "prefer-region", "prefer_region", "region"); v != "" {
		parts = append(parts, "区域="+labels.regionLabel(v))
	}
	if v := firstArg(args, "instance-type", "instance_type", "sku"); v != "" {
		parts = append(parts, "规格="+v)
	}
	if v := firstArg(args, "ncpu"); v != "" {
		parts = append(parts, "CPU="+v)
	}
	if v := firstArg(args, "mem-spec", "mem_spec"); v != "" {
		parts = append(parts, "内存="+v)
	}
	if disks := argStringSlice(args, "disk"); len(disks) > 0 {
		parts = append(parts, "磁盘="+truncateRunes(disks[0], 80))
	}
	if nets := argStringSlice(args, "net"); len(nets) > 0 {
		parts = append(parts, "网络="+strings.Join(nets, ","))
	} else {
		parts = append(parts, "网络=自动调度")
	}
	b.WriteString(strings.Join(parts, "，"))
	b.WriteByte('\n')

	body := stripMCPHint(resultText)
	if obj := parseJSONObject(body); obj != nil {
		status := jsonString(obj, "final_status")
		sid := jsonString(obj, "server_id")
		sname := ""
		if sid == "" {
			if srv, ok := obj["server"].(map[string]interface{}); ok {
				sid = jsonString(srv, "id")
				sname = jsonString(srv, "name")
				if status == "" {
					status = jsonString(srv, "status")
				}
			}
		} else if srv, ok := obj["server"].(map[string]interface{}); ok {
			sname = jsonString(srv, "name")
		}
		if waitErr := jsonString(obj, "wait_error"); waitErr != "" {
			b.WriteString(fmt.Sprintf("  创建未完成：%s\n", truncateRunes(waitErr, 160)))
			if hint := jsonString(obj, "hint"); hint != "" {
				b.WriteString(fmt.Sprintf("  提示：%s\n", truncateRunes(hint, 160)))
			}
			return b.String()
		}
		if sid != "" || status != "" || sname != "" {
			b.WriteString("  结果：")
			bits := make([]string, 0, 3)
			if sname != "" {
				bits = append(bits, "名称="+sname)
			}
			if sid != "" {
				bits = append(bits, "id="+sid)
			}
			if status != "" {
				bits = append(bits, "状态="+status)
			}
			b.WriteString(strings.Join(bits, "，"))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func isTruthy(v interface{}) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		return s == "true" || s == "1" || s == "yes" || s == "on"
	case float64:
		return x != 0
	case int:
		return x != 0
	default:
		return false
	}
}

func formatResourceListProgress(kind, resultText string, fields []string) string {
	items, total := extractListItems(resultText)
	if len(items) == 0 {
		if total == 0 {
			return fmt.Sprintf("✓ 未找到可用%s\n", kind)
		}
		return fmt.Sprintf("✓ 已查询%s（共 %d 条）\n", kind, total)
	}
	if total <= 0 {
		total = len(items)
	}
	labels := make([]string, 0, 5)
	for i, item := range items {
		if i >= 5 {
			break
		}
		labels = append(labels, formatItemLabel(item, fields))
	}
	more := ""
	if total > len(labels) {
		more = fmt.Sprintf("等共 %d 个", total)
	} else {
		more = fmt.Sprintf("共 %d 个", total)
	}
	return fmt.Sprintf("✓ 已找到%s：%s（%s）\n", kind, strings.Join(labels, "、"), more)
}

func formatSingleResourceProgress(kind, resultText string, fields []string) string {
	obj := parseJSONObject(stripMCPHint(resultText))
	if obj == nil {
		return fmt.Sprintf("✓ 已获取%s\n", kind)
	}
	return fmt.Sprintf("✓ %s：%s\n", kind, formatItemLabel(obj, fields))
}

func formatGenericProgress(toolName string, args map[string]interface{}, resultText string) string {
	id := firstArg(args, "id", "ID", "name", "NAME")
	items, total := extractListItems(resultText)
	if len(items) > 0 {
		return formatResourceListProgress(progressLabel(toolName), resultText, []string{"name", "id"})
	}
	if id != "" {
		return fmt.Sprintf("✓ %s完成%s\n", progressLabel(toolName), paren(id))
	}
	_ = resultText
	if total > 0 {
		return fmt.Sprintf("✓ %s完成（%d 条）\n", progressLabel(toolName), total)
	}
	return fmt.Sprintf("✓ %s完成\n", progressLabel(toolName))
}

func formatItemLabel(item map[string]interface{}, fields []string) string {
	parts := make([]string, 0, 3)
	seen := map[string]bool{}
	for _, f := range fields {
		v := jsonString(item, f)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		parts = append(parts, v)
		if len(parts) >= 2 {
			break
		}
	}
	if len(parts) == 0 {
		return "(未命名)"
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("%s(%s)", parts[0], parts[1])
}

func extractListItems(resultText string) ([]map[string]interface{}, int) {
	body := stripMCPHint(resultText)
	obj := parseJSONObject(body)
	if obj != nil {
		total := jsonInt(obj, "total")
		if total <= 0 {
			total = jsonInt(obj, "count")
		}
		for _, key := range []string{"data", "hits"} {
			data, ok := obj[key].([]interface{})
			if !ok {
				continue
			}
			items := make([]map[string]interface{}, 0, len(data))
			for _, d := range data {
				if m, ok := d.(map[string]interface{}); ok {
					items = append(items, m)
				}
			}
			if total <= 0 {
				total = len(items)
			}
			return items, total
		}
		// 单对象结果
		if jsonString(obj, "id") != "" || jsonString(obj, "name") != "" {
			return []map[string]interface{}{obj}, 1
		}
	}
	if arr := parseJSONArray(body); len(arr) > 0 {
		items := make([]map[string]interface{}, 0, len(arr))
		for _, d := range arr {
			if m, ok := d.(map[string]interface{}); ok {
				items = append(items, m)
			}
		}
		return items, len(items)
	}
	return nil, 0
}

func extractStorageTypeHints(resultText string) string {
	obj := parseJSONObject(stripMCPHint(resultText))
	if obj == nil {
		return ""
	}
	for _, key := range []string{"storage_types2", "StorageTypes2"} {
		raw, ok := obj[key]
		if !ok {
			continue
		}
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		set := make([]string, 0, 8)
		seen := map[string]bool{}
		for _, v := range m {
			arr, ok := v.([]interface{})
			if !ok {
				continue
			}
			for _, x := range arr {
				s, ok := x.(string)
				if !ok || s == "" || seen[s] {
					continue
				}
				seen[s] = true
				if i := strings.Index(s, "/"); i > 0 {
					s = s[:i]
				}
				set = append(set, s)
				if len(set) >= 6 {
					return strings.Join(set, "、")
				}
			}
		}
		if len(set) > 0 {
			return strings.Join(set, "、")
		}
	}
	return ""
}

func stripMCPHint(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "[MCP下一步]"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

func parseJSONObject(s string) map[string]interface{} {
	s = strings.TrimSpace(s)
	if s == "" || s[0] != '{' {
		// 可能前后有非 JSON 文本，尝试截取第一个对象
		start := strings.Index(s, "{")
		end := strings.LastIndex(s, "}")
		if start < 0 || end <= start {
			return nil
		}
		s = s[start : end+1]
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return nil
	}
	return obj
}

func parseJSONArray(s string) []interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if s[0] != '[' {
		start := strings.Index(s, "[")
		end := strings.LastIndex(s, "]")
		if start < 0 || end <= start {
			return nil
		}
		s = s[start : end+1]
	}
	var arr []interface{}
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return nil
	}
	return arr
}

func firstArg(args map[string]interface{}, keys ...string) string {
	if args == nil {
		return ""
	}
	normalize := func(k string) string {
		return strings.ReplaceAll(strings.ToLower(k), "_", "-")
	}
	for _, key := range keys {
		if v, ok := args[key]; ok {
			if s := stringifyArg(v); s != "" {
				return s
			}
		}
		want := normalize(key)
		for k, v := range args {
			if normalize(k) == want {
				if s := stringifyArg(v); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func argStringSlice(args map[string]interface{}, key string) []string {
	if args == nil {
		return nil
	}
	v, ok := args[key]
	if !ok {
		alt := strings.ReplaceAll(key, "-", "_")
		v, ok = args[alt]
		if !ok {
			return nil
		}
	}
	switch x := v.(type) {
	case []string:
		return x
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s := stringifyArg(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if x != "" {
			return []string{x}
		}
	}
	return nil
}

func stringifyArg(v interface{}) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%v", x)
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	case bool:
		return fmt.Sprintf("%v", x)
	case []interface{}:
		if len(x) == 0 {
			return ""
		}
		return stringifyArg(x[0])
	case []string:
		if len(x) == 0 {
			return ""
		}
		return x[0]
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func jsonString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	return stringifyArg(v)
}

func jsonInt(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case json.Number:
		n, _ := x.Int64()
		return int(n)
	default:
		return 0
	}
}

func paren(s string) string {
	if s == "" {
		return ""
	}
	return "（" + s + "）"
}

func truncateRunes(s string, max int) string {
	rs := []rune(strings.TrimSpace(s))
	if max <= 0 || len(rs) <= max {
		return string(rs)
	}
	return string(rs[:max]) + "…"
}
