package shell

import (
	"github.com/yunionio/jsonutils"

	"github.com/yunionio/onecloud/pkg/mcclient/modules"
	"github.com/yunionio/onecloud/pkg/util/printutils"
)

func printList(list *modules.ListResult, columns []string) {
	printutils.PrintJSONList(list, columns)
}

func printObject(obj jsonutils.JSONObject) {
	printutils.PrintJSONObject(obj)
}

func printBatchResults(results []modules.SubmitResult, columns []string) {
	printutils.PrintJSONBatchResults(results, columns)
}
