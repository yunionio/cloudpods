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
	"os"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/prettytable"
	"yunion.io/x/pkg/util/sets"
)

type StatusInfo struct {
	Status              string `json:"status"`
	TotalCount          int64  `json:"total_count"`
	PendingDeletedCount int64  `json:"pending_deleted_count"`
}

type StatusInfoList []StatusInfo

func (a StatusInfoList) Len() int      { return len(a) }
func (a StatusInfoList) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a StatusInfoList) Less(i, j int) bool {
	if a[i].TotalCount != a[j].TotalCount {
		return a[i].TotalCount > a[j].TotalCount
	}
	if a[i].PendingDeletedCount != a[j].PendingDeletedCount {
		return a[i].PendingDeletedCount > a[j].PendingDeletedCount
	}
	return a[i].Status < a[j].Status
}

type TotalCountWithStatusInfo struct {
	StatusInfo StatusInfoList `json:"status_info"`
}

func PrintJSONList(list *ListResult, columns []string) {
	colsWithData := make([]string, 0)
	if len(columns) == 0 {
		colsWithDataMap := make(map[string]bool)
		for _, obj := range list.Data {
			objdict, _ := obj.(*jsonutils.JSONDict)
			objmap, _ := objdict.GetMap()
			for k := range objmap {
				colsWithDataMap[k] = true
			}
		}
		prefixCols := []string{}
		for k := range colsWithDataMap {
			if sets.NewString("id", "name").Has(strings.ToLower(k)) {
				prefixCols = append(prefixCols, k)
			} else {
				colsWithData = append(colsWithData, k)
			}
		}
		sort.Strings(prefixCols)
		sort.Strings(colsWithData)
		colsWithData = append(prefixCols, colsWithData...)
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

	const (
		OS_MAX_COLUMN_TEXT_LENGTH = "OS_MAX_COLUMN_TEXT_LENGTH"
		OS_TRY_TERM_WIDTH         = "OS_TRY_TERM_WIDTH"

		defaultMaxColLength = 512
	)
	maxColLength := int64(-1)
	colTruncated := false
	tryTermWidth := false
	screenWidth, _ := termWidth()
	if screenWidth > 0 {
		// the width of screen is available, which is an interactive shell
		// truncate the text to make it prettier
		osMaxTextLength := os.Getenv(OS_MAX_COLUMN_TEXT_LENGTH)
		if len(osMaxTextLength) > 0 {
			maxColLength, _ = strconv.ParseInt(osMaxTextLength, 10, 64)
			if maxColLength >= 0 && maxColLength < 3 {
				maxColLength = defaultMaxColLength
			}
		} else {
			maxColLength = defaultMaxColLength
		}
		osTryTermWidth := os.Getenv(OS_TRY_TERM_WIDTH)
		if strings.ToLower(osTryTermWidth) == "true" {
			tryTermWidth = true
		}
	}
	pt := prettytable.NewPrettyTableWithTryTermWidth(colsWithData, tryTermWidth)
	rows := make([][]string, 0)
	for _, obj := range list.Data {
		row := make([]string, 0)
		for _, k := range colsWithData {
			v, e := obj.GetIgnoreCases(k)
			if e == nil {
				s, _ := v.GetString()
				if maxColLength > 0 && int64(len(s)) > maxColLength {
					s = s[0:maxColLength-3] + "..."
					colTruncated = true
				}
				row = append(row, s)
			} else {
				row = append(row, "")
			}
		}
		rows = append(rows, row)
	}
	fmt.Print(pt.GetString(rows))
	total := int64(list.Total)
	if list.Total == 0 {
		total = int64(len(list.Data))
	}
	title := fmt.Sprintf("Total: %d", total)
	if len(list.MarkerField) > 0 {
		title += fmt.Sprintf(" Field: %s Order: %s", list.MarkerField, list.MarkerOrder)
		if len(list.NextMarker) > 0 {
			title += fmt.Sprintf(" NextMarker: %s", list.NextMarker)
		}
	} else {
		if list.Limit == 0 && total > int64(len(list.Data)) {
			list.Limit = len(list.Data)
		}
		if list.Limit > 0 {
			pages := int(total / int64(list.Limit))
			if int64(pages*list.Limit) < total {
				pages += 1
			}
			page := int(list.Offset/list.Limit) + 1
			title = fmt.Sprintf("%s Pages: %d Limit: %d Offset: %d Page: %d",
				title, pages, list.Limit, list.Offset, page)
		}
	}
	fmt.Println("*** ", title, " ***")
	if list.Totals != nil {
		totalWithStatusInfo := TotalCountWithStatusInfo{}
		err := list.Totals.Unmarshal(&totalWithStatusInfo)
		if err != nil {
			fmt.Println("error to unmarshal totals to TotalCountWithStatusInfo", err)
		} else if len(totalWithStatusInfo.StatusInfo) > 0 {
			sort.Sort(totalWithStatusInfo.StatusInfo)
			pt := prettytable.NewPrettyTableWithTryTermWidth([]string{"#", "Status", "Count", "PendingDeletedCount"}, tryTermWidth)
			rows := make([][]string, 0)
			for i, statusInfo := range totalWithStatusInfo.StatusInfo {
				rows = append(rows, []string{fmt.Sprintf("#%d", i+1), statusInfo.Status, strconv.FormatInt(statusInfo.TotalCount, 10), strconv.FormatInt(statusInfo.PendingDeletedCount, 10)})
			}
			fmt.Print(pt.GetString(rows))
		}
	}
	if colTruncated {
		fmt.Println("!!!Some text truncated, set env", OS_MAX_COLUMN_TEXT_LENGTH, "=-1 to show full text!!!")
	}
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
	switch jObj := obj.(type) {
	case *jsonutils.JSONDict:
		printJSONObject(jObj, func(s string) {
			fmt.Print(s)
		})
	case *jsonutils.JSONArray:
		printJSONArray(jObj)
	default:
		fmt.Println("Not a valid JSON object:", obj.String())
		return
	}
}

func printJSONArray(array *jsonutils.JSONArray) {
	objs, err := array.GetArray()
	if err != nil {
		fmt.Printf("GetArray objects error: %v", err)
	}

	if len(objs) == 0 {
		return
	}

	columns := sets.NewString()

	for _, obj := range objs {
		dict, ok := obj.(*jsonutils.JSONDict)
		if !ok {
			fmt.Printf("Object %q is not dict", obj)
			return
		}
		jMap, err := dict.GetMap()
		if err != nil {
			fmt.Printf("GetMap error: %v, object: %q", err, obj)
			return
		}
		for k := range jMap {
			columns.Insert(k)
		}
	}

	list := &ListResult{
		Data:   objs,
		Total:  array.Length(),
		Limit:  0,
		Offset: 0,
	}
	PrintJSONList(list, columns.List())
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

func PrintJSONBatchResults(results []SubmitResult, columns []string) {
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
		PrintJSONList(&ListResult{Data: objs}, columns)
	}
	if len(errs) > 0 {
		PrintJSONList(&ListResult{Data: errs}, []string{"status", "id", "error"})
	}
}
