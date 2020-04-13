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

package responses

import (
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
)

type ListResult struct {
	Data     []jsonutils.JSONObject
	Total    int
	Limit    int
	Offset   int
	NextLink string
}

func ListResult2JSONWithKey(result *ListResult, key string) jsonutils.JSONObject {
	obj := jsonutils.NewDict()
	if result.Total > 0 {
		obj.Add(jsonutils.NewInt(int64(result.Total)), "total")
	}
	if result.Limit > 0 {
		obj.Add(jsonutils.NewInt(int64(result.Limit)), "limit")
	}
	if result.Offset > 0 {
		obj.Add(jsonutils.NewInt(int64(result.Offset)), "offset")
	}
	arr := jsonutils.NewArray(result.Data...)
	obj.Add(arr, key)
	return obj
}

func ListResult2JSON(result *ListResult) jsonutils.JSONObject {
	return ListResult2JSONWithKey(result, "data")
}

func JSON2ListResult(result jsonutils.JSONObject) *ListResult {
	total, _ := result.Int("total")
	limit, _ := result.Int("limit")
	offset, _ := result.Int("offset")
	data, _ := result.GetArray("data")
	return &ListResult{Data: data, Total: int(total), Limit: int(limit), Offset: int(offset)}
}

// 将key中的冒号替换成
func TransColonToDot(obj jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	re, _ := regexp.Compile("[a-zA-Z0-9](:+)[^\"]+\"\\s*:\\s*")

	if obj == nil {
		return obj, nil
	}

	newStr := re.ReplaceAllStringFunc(obj.String(), func(s string) string {
		count := strings.Count(s, ":")
		if count > 1 {
			return strings.Replace(s, ":", ".", count-1)
		}
		return s
	})

	return jsonutils.ParseString(newStr)
}
