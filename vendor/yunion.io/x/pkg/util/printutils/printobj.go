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

	"yunion.io/x/jsonutils"
)

func PrintInterfaceList(data interface{}, total, offset, limit int, columns []string) {
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
	list := &ListResult{
		Data:   jsonList,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
	PrintJSONList(list, columns)
}

func PrintInterfaceObject(obj interface{}) {
	PrintJSONObject(jsonutils.Marshal(obj))
}
