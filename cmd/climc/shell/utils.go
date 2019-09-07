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
	"fmt"
	"os"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/excelutils"
	"yunion.io/x/onecloud/pkg/util/printutils"
)

const (
	OUTPUT_FORMAT_TABLE         = "table"         // pretty table
	OUTPUT_FORMAT_FLATTEN_TABLE = "flatten-table" // pretty table with flattened keys
	OUTPUT_FORMAT_JSON          = "json"          // json string
	OUTPUT_FORMAT_KV            = "kv"            // "key: value" as separate line
	OUTPUT_FORMAT_FLATTEN_KV    = "flatten-kv"    // kv with flattened keys
)

var outputFormat = OUTPUT_FORMAT_TABLE

func OutputFormat(s string) {
	outputFormat = s
}

func printList(list *modulebase.ListResult, columns []string) {
	printutils.PrintJSONList(list, columns)
}

func printObject(obj jsonutils.JSONObject) {
	switch outputFormat {
	case OUTPUT_FORMAT_TABLE:
		printutils.PrintJSONObject(obj)
	case OUTPUT_FORMAT_KV:
		printObjectFmtKv(obj)
	case OUTPUT_FORMAT_JSON:
		fmt.Print(obj.String())
		fmt.Print("\n")
	case OUTPUT_FORMAT_FLATTEN_TABLE:
		printObjectRecursive(obj)
	case OUTPUT_FORMAT_FLATTEN_KV:
		printObjectRecursiveEx(obj, printObjectFmtKv)
	default:
		fmt.Fprintf(os.Stderr, "unknown output format: %q\n", outputFormat)
	}
}

func printObjectFmtKv(obj jsonutils.JSONObject) {
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
		fmt.Printf("%*s: %s\n", maxWidth, k, s)
	}
}

func printObjectRecursive(obj jsonutils.JSONObject) {
	printutils.PrintJSONObjectRecursive(obj)
}

func printObjectRecursiveEx(obj jsonutils.JSONObject, cb printutils.PrintJSONObjectRecursiveExFunc) {
	printutils.PrintJSONObjectRecursiveEx(obj, cb)
}

func printBatchResults(results []modulebase.SubmitResult, columns []string) {
	printutils.PrintJSONBatchResults(results, columns)
}

func exportList(list *modulebase.ListResult, file string, exportKeys string, exportTexts string, columns []string) {
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
