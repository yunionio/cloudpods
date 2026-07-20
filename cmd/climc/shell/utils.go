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

package shell

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/util/excelutils"
)

const (
	OUTPUT_FORMAT_TABLE         = "table"         // pretty table
	OUTPUT_FORMAT_FLATTEN_TABLE = "flatten-table" // pretty table with flattened keys
	OUTPUT_FORMAT_JSON          = "json"          // json string
	OUTPUT_FORMAT_YAML          = "yaml"          // yaml string
	OUTPUT_FORMAT_KV            = "kv"            // "key: value" as separate line
	OUTPUT_FORMAT_FLATTEN_KV    = "flatten-kv"    // kv with flattened keys
)

var outputFormat = OUTPUT_FORMAT_TABLE

// goroutine 本地输出：MCP 并发 tools/call 时避免劫持全局 os.Stdout。
type outputState struct {
	writer io.Writer
	format string
}

var outputStates sync.Map // uint64(goid) -> *outputState

func OutputFormat(s string) {
	outputFormat = s
}

// PushOutput 将当前 goroutine 的 shell 输出重定向到 w，并可选覆盖格式。
// 返回的 restore 必须在同一 goroutine 调用。
func PushOutput(w io.Writer, format string) (restore func()) {
	id := goroutineID()
	prev, _ := outputStates.Load(id)
	outputStates.Store(id, &outputState{writer: w, format: format})
	return func() {
		if prev != nil {
			outputStates.Store(id, prev)
		} else {
			outputStates.Delete(id)
		}
	}
}

func currentWriter() io.Writer {
	if v, ok := outputStates.Load(goroutineID()); ok {
		if s := v.(*outputState); s != nil && s.writer != nil {
			return s.writer
		}
	}
	return os.Stdout
}

func currentFormat() string {
	if v, ok := outputStates.Load(goroutineID()); ok {
		if s := v.(*outputState); s != nil && s.format != "" {
			return s.format
		}
	}
	return outputFormat
}

func goroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	i := bytes.IndexByte(b, ' ')
	if i <= 0 {
		return 0
	}
	n, _ := strconv.ParseUint(string(b[:i]), 10, 64)
	return n
}

func PrintList(list *printutils.ListResult, columns []string) {
	w := currentWriter()
	switch currentFormat() {
	case OUTPUT_FORMAT_TABLE:
		if w == os.Stdout {
			printutils.PrintJSONList(list, columns)
			return
		}
		fmt.Fprint(w, jsonutils.Marshal(list).PrettyString())
		fmt.Fprint(w, "\n")
	case OUTPUT_FORMAT_JSON:
		fmt.Fprint(w, jsonutils.Marshal(list).PrettyString())
		fmt.Fprint(w, "\n")
	case OUTPUT_FORMAT_YAML:
		fmt.Fprint(w, jsonutils.Marshal(list).YAMLString())
	default:
		fmt.Fprintf(os.Stderr, "unknown output format: %q\n", currentFormat())
	}
}

func PrintObject(obj jsonutils.JSONObject) {
	w := currentWriter()
	switch currentFormat() {
	case OUTPUT_FORMAT_TABLE:
		if w == os.Stdout {
			printutils.PrintJSONObject(obj)
			return
		}
		fmt.Fprint(w, obj.PrettyString())
		fmt.Fprint(w, "\n")
	case OUTPUT_FORMAT_KV:
		printObjectFmtKv(obj)
	case OUTPUT_FORMAT_JSON:
		fmt.Fprint(w, obj.PrettyString())
		fmt.Fprint(w, "\n")
	case OUTPUT_FORMAT_YAML:
		fmt.Fprint(w, obj.YAMLString())
	case OUTPUT_FORMAT_FLATTEN_TABLE:
		printObjectRecursive(obj)
	case OUTPUT_FORMAT_FLATTEN_KV:
		printObjectRecursiveEx(obj, printObjectFmtKv)
	default:
		fmt.Fprintf(os.Stderr, "unknown output format: %q\n", currentFormat())
	}
}

func printObjectFmtKv(obj jsonutils.JSONObject) {
	w := currentWriter()
	m, _ := obj.GetMap()
	maxWidth := 0
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
		if maxWidth < len(k) {
			maxWidth = len(k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		var s string
		objV := m[k]
		if objS, ok := objV.(*jsonutils.JSONString); ok {
			s, _ = objS.GetString()
			s = strings.TrimRight(s, "\n")
		} else {
			s = objV.String()
		}
		fmt.Fprintf(w, "%*s: %s\n", maxWidth, k, s)
	}
}

func printObjectRecursive(obj jsonutils.JSONObject) {
	PrintObject(obj)
}

func printObjectRecursiveEx(obj jsonutils.JSONObject, cb printutils.PrintJSONObjectRecursiveExFunc) {
	printutils.PrintJSONObjectRecursiveEx(obj, cb)
}

func PrintBatchResults(results []printutils.SubmitResult, columns []string) {
	w := currentWriter()
	switch currentFormat() {
	case OUTPUT_FORMAT_JSON:
		fmt.Fprint(w, jsonutils.Marshal(results).PrettyString())
		fmt.Fprint(w, "\n")
	default:
		if w == os.Stdout {
			printutils.PrintJSONBatchResults(results, columns)
			return
		}
		fmt.Fprint(w, jsonutils.Marshal(results).PrettyString())
		fmt.Fprint(w, "\n")
	}
}

func printBatchResults(results []printutils.SubmitResult, columns []string) {
	PrintBatchResults(results, columns)
}

func ExportList(list *printutils.ListResult, file string, exportKeys string, exportTexts string, columns []string) {
	var keys []string
	var texts []string
	if len(exportKeys) > 0 {
		keys = strings.Split(exportKeys, ",")
		texts = strings.Split(exportTexts, ",")
	} else {
		keys = columns
		texts = columns
	}
	excelutils.ExportFile(list.Data, keys, texts, file)
}
