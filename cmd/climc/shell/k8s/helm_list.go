package k8s

import (
	"fmt"

	"github.com/gosuri/uitable"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ListPrinter interface {
	Columns() []interface{}
	RowKeys(obj *jsonutils.JSONDict) []interface{}
}

func ListerTable(res *modules.ListResult, printer ListPrinter) *uitable.Table {
	min := func(x, y int) int {
		if x < y {
			return x
		}
		return y
	}
	table := uitable.New()
	table.MaxColWidth = 80
	table.AddRow(printer.Columns()...)
	var idx int
	for ; idx < min(res.Limit, res.Total); idx++ {
		table.AddRow(printer.RowKeys(res.Data[idx].(*jsonutils.JSONDict))...)
	}
	return table
}

func PrintHelmListResult(res *modules.ListResult, printer ListPrinter) {
	fmt.Println(ListerTable(res, printer))

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
