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

package db

import (
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"
)

func FetchModelExtraCountProperties(model IModel, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	selfValue := reflect.ValueOf(model)
	selfType := reflect.TypeOf(model)
	for i := 0; i < selfValue.NumMethod(); i += 1 {
		methodValue := selfValue.Method(i)
		methodType := methodValue.Type()
		if methodType.NumIn() != 0 || methodType.NumOut() != 2 {
			continue
		}
		methodName := selfType.Method(i).Name
		tokens := utils.CamelSplitTokens(methodName)
		if len(tokens) < 3 {
			continue
		}
		if strings.EqualFold(tokens[0], "get") && strings.EqualFold(tokens[len(tokens)-1], "count") {
			resName := strings.ToLower(strings.Join(tokens[1:], "_"))
			outs := methodValue.Call([]reflect.Value{})
			extra.Add(jsonutils.NewInt(outs[0].Int()), resName)
		}
	}
	return extra
}
