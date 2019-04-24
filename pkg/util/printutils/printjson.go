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

package printutils

import (
	"fmt"
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/prettytable"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func PrintJSONList(list *modules.ListResult, columns []string) {
	colsWithData := make([]string, 0)
	if columns == nil || len(columns) == 0 {
		colsWithDataMap := make(map[string]bool)
		for _, obj := range list.Data {
			objdict, _ := obj.(*jsonutils.JSONDict)
			objmap, _ := objdict.GetMap()
			for k := range objmap {
				colsWithDataMap[k] = true
			}
		}
		for k := range colsWithDataMap {
			colsWithData = append(colsWithData, k)
		}
		sort.Strings(colsWithData)
	} else {
		colsWithDataMap := make(map[string]bool)
		for _, obj := range list.Data {
			for _, k := range columns {
				if _, ok := colsWithDataMap[k]; !ok {
					_, e := obj.GetIgnoreCases(k)
					if e == nil {
						colsWithDataMap[k] = true
					}
				}
			}
		}
		for _, k := range columns {
			if _, ok := colsWithDataMap[k]; ok {
				colsWithData = append(colsWithData, k)
			}
		}
	}
	pt := prettytable.NewPrettyTable(colsWithData)
	rows := make([][]string, 0)
	for _, obj := range list.Data {
		row := make([]string, 0)
		for _, k := range colsWithData {
			v, e := obj.GetIgnoreCases(k)
			if e == nil {
				s, _ := v.GetString()
				row = append(row, s)
			} else {
				row = append(row, "")
			}
		}
		rows = append(rows, row)
	}
	fmt.Print(pt.GetString(rows))
	if list.Total == 0 {
		list.Total = len(list.Data)
	}
	title := fmt.Sprintf("Total: %d", list.Total)
	if list.Limit == 0 && list.Total > len(list.Data) {
		list.Limit = len(list.Data)
	}
	if list.Limit > 0 {
		pages := int(list.Total / list.Limit)
		if pages*list.Limit < list.Total {
			pages += 1
		}
		page := int(list.Offset/list.Limit) + 1
		title = fmt.Sprintf("%s Pages: %d Limit: %d Offset: %d Page: %d",
			title, pages, list.Limit, list.Offset, page)
	}
	fmt.Println("*** ", title, " ***")
}

func printJSONObject(dict *jsonutils.JSONDict, cb PrintJSONObjectFunc) {
	keys := dict.SortedKeys()
	pt := prettytable.NewPrettyTable([]string{"Field", "Value"})
	rows := make([][]string, 0)
	for _, k := range keys {
		row := make([]string, 0)
		row = append(row, k)
		vs, e := dict.GetString(k)
		if e != nil {
			row = append(row, fmt.Sprintf("Error: %s", e))
		} else {
			row = append(row, vs)
		}
		rows = append(rows, row)
	}
	cb(pt.GetString(rows))
}

func PrintJSONObject(obj jsonutils.JSONObject) {
	dict, ok := obj.(*jsonutils.JSONDict)
	if !ok {
		fmt.Println("Not a valid JSON object:", obj.String())
		return
	}
	printJSONObject(dict, func(s string) {
		fmt.Print(s)
	})
}

func flattenJSONObjectRecursive(v jsonutils.JSONObject, k string, rootDict *jsonutils.JSONDict) {
	switch vv := v.(type) {
	case *jsonutils.JSONString, *jsonutils.JSONInt, *jsonutils.JSONBool, *jsonutils.JSONFloat:
		rootDict.Set(k, vv)
	case *jsonutils.JSONArray:
		arr, _ := vv.GetArray()
		for i, arrElem := range arr {
			nextK := fmt.Sprintf("%s.%d", k, i)
			flattenJSONObjectRecursive(arrElem, nextK, rootDict)
		}
		if k != "" {
			rootDict.Remove(k)
		}
	case *jsonutils.JSONDict:
		m, _ := vv.GetMap()
		for kk, w := range m {
			nextK := kk
			if k != "" {
				nextK = k + "." + nextK
			}
			flattenJSONObjectRecursive(w, nextK, rootDict)
		}
		if k != "" {
			rootDict.Remove(k)
		}
	}
}

type PrintJSONObjectRecursiveExFunc func(jsonutils.JSONObject)
type PrintJSONObjectFunc func(string)

func printJSONObjectRecursive_(obj jsonutils.JSONObject, cb PrintJSONObjectRecursiveExFunc) {
	dict, ok := obj.(*jsonutils.JSONDict)
	if !ok {
		fmt.Println("Not a valid JSON object:", obj.String())
		return
	}
	dictCopy := jsonutils.DeepCopy(dict).(*jsonutils.JSONDict)
	flattenJSONObjectRecursive(dictCopy, "", dictCopy)
	cb(dictCopy)
}

func PrintJSONObjectRecursive(obj jsonutils.JSONObject) {
	printJSONObjectRecursive_(obj, PrintJSONObject)
}

func PrintJSONObjectRecursiveEx(obj jsonutils.JSONObject, cb PrintJSONObjectRecursiveExFunc) {
	printJSONObjectRecursive_(obj, cb)
}

func PrintJSONBatchResults(results []modules.SubmitResult, columns []string) {
	objs := make([]jsonutils.JSONObject, 0)
	errs := make([]jsonutils.JSONObject, 0)
	for _, r := range results {
		if r.Status == 200 {
			objs = append(objs, r.Data)
		} else {
			err := jsonutils.NewDict()
			err.Add(jsonutils.NewInt(int64(r.Status)), "status")
			err.Add(jsonutils.Marshal(r.Id), "id")
			err.Add(r.Data, "error")
			errs = append(errs, err)
		}
	}
	if len(objs) > 0 {
		PrintJSONList(&modules.ListResult{Data: objs}, columns)
	}
	if len(errs) > 0 {
		PrintJSONList(&modules.ListResult{Data: errs}, []string{"status", "id", "error"})
	}
}
