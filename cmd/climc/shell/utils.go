package shell

import (
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient/modules"
	"github.com/yunionio/pkg/util/printjson"
)

func printList(list *modules.ListResult, columns []string) {
	printjson.PrintList(list, columns)
}

func printObject(obj jsonutils.JSONObject) {
	printjson.PrintObject(obj)
}

func printBatchResults(results []modules.SubmitResult, columns []string) {
	printjson.PrintBatchResults(results, columns)
}
