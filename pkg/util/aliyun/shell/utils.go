package shell

import (
	"fmt"
	"reflect"
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient/modules"
	"github.com/yunionio/pkg/util/printjson"
)

func printList(data interface{}, total, offset, limit int, columns []string) {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		fmt.Println("Invalid list data")
		return
	}
	jsonList := make([]jsonutils.JSONObject, dataValue.Len())
	for i := 0; i < dataValue.Len(); i += 1 {
		jsonList[i] = jsonutils.Marshal(dataValue.Index(i).Interface())
	}
	if total == 0 {
		total = dataValue.Len()
	}
	list := &modules.ListResult{
		Data:   jsonList,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
	printjson.PrintList(list, columns)
}

func printObject(obj interface{}) {
	printjson.PrintObject(jsonutils.Marshal(obj))
}
