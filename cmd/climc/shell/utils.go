package shell

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/printutils"
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
