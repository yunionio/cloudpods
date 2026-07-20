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
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	computeoptions "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func mcpArgsToArgv(parser *structarg.ArgumentParser, args map[string]interface{}) []string {
	argv := make([]string, 0)

	normalize := func(k string) string {
		return strings.ReplaceAll(k, "_", "-")
	}
	lookup := func(token string) (interface{}, bool) {
		if v, ok := args[token]; ok {
			return v, true
		}
		alt := strings.ReplaceAll(token, "-", "_")
		if v, ok := args[alt]; ok {
			return v, true
		}
		for k, v := range args {
			if normalize(k) == token {
				return v, true
			}
		}
		return nil, false
	}

	for _, arg := range parser.GetPosArgs() {
		v, ok := lookup(arg.Token())
		if !ok {
			continue
		}
		argv = append(argv, valueToArgvParts(v)...)
	}

	for _, arg := range parser.GetOptArgs() {
		token := arg.Token()
		if token == "help" {
			continue
		}
		v, ok := lookup(token)
		if !ok {
			continue
		}
		if !arg.NeedData() {
			if isTruthy(v) {
				argv = append(argv, "--"+token)
			} else if neg := arg.NegativeToken(); neg != "" && isFalsy(v) {
				argv = append(argv, "--"+neg)
			}
			continue
		}
		if arg.IsMulti() {
			for _, part := range valueToArgvParts(v) {
				argv = append(argv, "--"+token, part)
			}
			continue
		}
		parts := valueToArgvParts(v)
		if len(parts) == 0 {
			continue
		}
		argv = append(argv, "--"+token, parts[0])
	}
	return argv
}

func valueToArgvParts(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case string:
		if x == "" {
			return nil
		}
		return []string{x}
	case bool:
		return []string{strconv.FormatBool(x)}
	case float64:
		if x == float64(int64(x)) {
			return []string{strconv.FormatInt(int64(x), 10)}
		}
		return []string{strconv.FormatFloat(x, 'f', -1, 64)}
	case int:
		return []string{strconv.Itoa(x)}
	case int64:
		return []string{strconv.FormatInt(x, 10)}
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			out = append(out, valueToArgvParts(item)...)
		}
		return out
	case []string:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	default:
		s := fmt.Sprint(x)
		if s == "" || s == "<nil>" {
			return nil
		}
		return []string{s}
	}
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

func isFalsy(v interface{}) bool {
	switch x := v.(type) {
	case bool:
		return !x
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		return s == "false" || s == "0" || s == "no" || s == "off"
	default:
		return false
	}
}

func invokeCommand(cmd shell.CMD, session *mcclient.ClientSession, args map[string]interface{}) (string, error) {
	parser, _, err := newArgumentParser(cmd)
	if err != nil {
		return "", err
	}

	// list 查询默认 scope=max，避免权限范围过窄查不到资源
	if strings.HasSuffix(cmd.Command, "-list") {
		if _, ok := args["scope"]; !ok {
			if _, ok := args["Scope"]; !ok {
				args["scope"] = "max"
			}
		}
	}

	// 创建虚拟机前查区域：默认 usable=true，只返回网络可用的区域
	if cmd.Command == "cloud-region-list" {
		if _, ok := argLookup(args, "usable"); !ok {
			args["usable"] = true
		}
	}

	if cmd.Command == "server-sku-list" {
		normalizeServerSkuListArgs(args)
	}

	argv := mcpArgsToArgv(parser, args)
	if err := parser.ParseArgs(argv, false); err != nil {
		return "", fmt.Errorf("parse args for %s: %w (argv=%v)", cmd.Command, err, argv)
	}
	filled := parser.Options()

	cbVal := reflect.ValueOf(cmd.Callback)
	if cbVal.Kind() != reflect.Func {
		return "", fmt.Errorf("callback of %s is not a function", cmd.Command)
	}

	// 使用 goroutine 本地 writer，避免劫持全局 os.Stdout / stdoutMu 串行化所有 tools/call。
	var buf bytes.Buffer
	restore := shell.PushOutput(&buf, shell.OUTPUT_FORMAT_JSON)
	defer restore()

	var callErr error
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				callErr = fmt.Errorf("panic invoking %s: %v", cmd.Command, rec)
			}
		}()
		outs := cbVal.Call([]reflect.Value{
			reflect.ValueOf(session),
			reflect.ValueOf(filled),
		})
		if len(outs) == 1 && !outs[0].IsNil() {
			callErr = outs[0].Interface().(error)
		}
	}()

	out := buf.String()
	if callErr != nil {
		return out, callErr
	}
	// 压缩 JSON，降低模型上下文占用，减少“只看完区域就不往下走”
	if compact := compactJSON(out); compact != "" {
		out = compact
	}
	return out, nil
}

func compactJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// normalizeServerSkuListArgs 让 AI 传的 2c2g / cpu-core-count 等别名落到 climc 的 spec/cpu/mem。
func normalizeServerSkuListArgs(args map[string]interface{}) {
	aliasCopy := func(from, to string) {
		if _, ok := argLookup(args, to); ok {
			return
		}
		if v, ok := argLookup(args, from); ok {
			args[to] = v
		}
	}
	aliasCopy("cpu-core-count", "cpu")
	aliasCopy("cpu_core_count", "cpu")
	aliasCopy("memory-size-mb", "mem")
	aliasCopy("memory_size_mb", "mem")

	if _, hasSpec := argLookup(args, "spec"); !hasSpec {
		// search/name 里如果是口语规格，提升为 spec
		for _, key := range []string{"search", "name"} {
			v, ok := argLookup(args, key)
			if !ok {
				continue
			}
			s := strings.TrimSpace(firstString(v))
			if s == "" {
				continue
			}
			if _, _, err := computeoptions.ParseSkuSpec(s); err == nil {
				args["spec"] = s
				delete(args, key)
				delete(args, strings.ReplaceAll(key, "-", "_"))
				break
			}
		}
	}
}
