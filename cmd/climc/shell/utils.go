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
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/excelutils"
	"yunion.io/x/onecloud/pkg/util/printutils"
)

func printList(list *modules.ListResult, columns []string) {
	printutils.PrintJSONList(list, columns)
}

func printObject(obj jsonutils.JSONObject) {
	printutils.PrintJSONObject(obj)
}

func printObjectRecursive(obj jsonutils.JSONObject) {
	printutils.PrintJSONObjectRecursive(obj)
}

func printObjectRecursiveEx(obj jsonutils.JSONObject, cb printutils.PrintJSONObjectRecursiveExFunc) {
	printutils.PrintJSONObjectRecursiveEx(obj, cb)
}

func printBatchResults(results []modules.SubmitResult, columns []string) {
	printutils.PrintJSONBatchResults(results, columns)
}

func exportList(list *modules.ListResult, file string, exportKeys string, exportTexts string, columns []string) {
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
