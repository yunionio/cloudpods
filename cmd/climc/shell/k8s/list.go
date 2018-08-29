package k8s

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gosuri/uitable"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
)

func getPrinterRowValue(printer k8s.ListPrinter, obj jsonutils.JSONObject, col string) interface{} {
	getFuncName := fmt.Sprintf("Get%s", strings.Title(col))
	manValue := reflect.ValueOf(printer)
	funcValue := manValue.MethodByName(getFuncName)
	if !funcValue.IsValid() || funcValue.IsNil() {
		log.Errorf("Can't get function: %q of manager: %#v", getFuncName, printer)
		return nil
	}
	params := []reflect.Value{
		reflect.ValueOf(obj),
	}
	outs := funcValue.Call(params)
	if len(outs) != 1 {
		log.Errorf("Invalid return value of function: %q", getFuncName)
		return nil
	}
	return outs[0].Interface()
}

func getPrinterRowValues(printer k8s.ListPrinter, obj jsonutils.JSONObject, cols []string) []interface{} {
	ret := make([]interface{}, 0)
	for _, col := range cols {
		ret = append(ret, getPrinterRowValue(printer, obj, col))
	}
	return ret
}

func ListerTable(res *modules.ListResult, printer k8s.ListPrinter, s *mcclient.ClientSession) *uitable.Table {
	min := func(x, y int) int {
		if x < y {
			return x
		}
		return y
	}
	table := uitable.New()
	table.MaxColWidth = 80
	cols := printer.GetColumns(s)
	colsI := make([]interface{}, len(cols))
	for i, v := range cols {
		colsI[i] = v
	}
	table.AddRow(colsI...)
	var idx int
	for ; idx < min(res.Limit, res.Total-res.Offset); idx++ {
		table.AddRow(getPrinterRowValues(printer, res.Data[idx], cols)...)
	}
	return table
}

func PrintListResultTable(res *modules.ListResult, printer k8s.ListPrinter, s *mcclient.ClientSession) {
	fmt.Println(ListerTable(res, printer, s))

	table := uitable.New()
	total := res.Total
	offset := res.Offset
	limit := res.Limit
	page := (offset / limit) + 1
	pages := total / limit
	if pages*limit < total {
		pages += 1
	}
	table.AddRow("")
	table.AddRow("Total", "Pages", "Limit", "Offset", "Page")
	table.AddRow(total, pages, limit, offset, page)
	fmt.Println(table)
}
