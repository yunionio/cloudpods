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
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"
)

func getter2json(obj interface{}) jsonutils.JSONObject {
	jsonDict := jsonutils.NewDict()

	objValue := reflect.ValueOf(obj)
	objType := reflect.TypeOf(obj)

	// log.Debugf("getter2json %d", objValue.NumMethod())

	for i := 0; i < objValue.NumMethod(); i += 1 {
		methodValue := objValue.Method(i)
		method := objType.Method(i)
		methodName := method.Name
		methodType := methodValue.Type()

		if strings.HasPrefix(methodName, "Get") && methodType.NumIn() == 0 && methodType.NumOut() >= 1 {
			fieldName := utils.CamelSplit(methodName[3:], "_")
			out := methodValue.Call([]reflect.Value{})
			if len(out) == 1 && !gotypes.IsNil(out[0].Interface()) {
				jsonDict.Add(jsonutils.Marshal(out[0].Interface()), fieldName)
			} else if len(out) == 2 {
				err, ok := out[1].Interface().(error)
				if ok {
					if err != nil && !gotypes.IsNil(out[0].Interface()) {
						jsonDict.Add(jsonutils.Marshal(out[0].Interface()), fieldName)
					}
				}
			}
		}
	}

	return jsonDict
}

func PrintGetterList(data interface{}, columns []string) {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		fmt.Println("Invalid list data")
		return
	}
	jsonList := make([]jsonutils.JSONObject, dataValue.Len())
	for i := 0; i < dataValue.Len(); i += 1 {
		jsonList[i] = getter2json(dataValue.Index(i).Interface())
	}
	list := &ListResult{
		Data:   jsonList,
		Total:  dataValue.Len(),
		Limit:  0,
		Offset: 0,
	}
	PrintJSONList(list, columns)
}

func PrintGetterObject(obj interface{}) {
	PrintJSONObject(getter2json(obj))
}
