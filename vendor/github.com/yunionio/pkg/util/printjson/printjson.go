package printjson

import (
	"fmt"
	"sort"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient/modules"
	"github.com/yunionio/pkg/prettytable"
)

func PrintList(list *modules.ListResult, columns []string) {
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
	fmt.Println(pt.GetString(rows))
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

func PrintObject(obj jsonutils.JSONObject) {
	dict, ok := obj.(*jsonutils.JSONDict)
	if !ok {
		fmt.Println("Not a valid JSON object:", obj.String())
		return
	}
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
	fmt.Println(pt.GetString(rows))
}

func PrintBatchResults(results []modules.SubmitResult, columns []string) {
	objs := make([]jsonutils.JSONObject, 0)
	errs := make([]jsonutils.JSONObject, 0)
	for _, r := range results {
		if r.Status == 200 {
			objs = append(objs, r.Data)
		} else {
			err := jsonutils.NewDict()
			err.Add(jsonutils.NewInt(int64(r.Status)), "status")
			err.Add(jsonutils.NewString(r.Id), "id")
			err.Add(r.Data, "error")
			errs = append(errs, err)
		}
	}
	if len(objs) > 0 {
		PrintList(&modules.ListResult{Data: objs}, columns)
	}
	if len(errs) > 0 {
		PrintList(&modules.ListResult{Data: errs}, []string{"status", "id", "error"})
	}
}
