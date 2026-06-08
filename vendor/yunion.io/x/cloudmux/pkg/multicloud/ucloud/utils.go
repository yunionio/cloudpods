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

package ucloud

import (
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

var responseMetaKeys = map[string]bool{
	"RetCode":    true,
	"Message":    true,
	"Action":     true,
	"TotalCount": true,
}

func detectResultKey(resp jsonutils.JSONObject, result interface{}) string {
	if result == nil {
		return ""
	}
	rt := reflect.TypeOf(result)
	if rt.Kind() != reflect.Ptr {
		return ""
	}
	et := rt.Elem()
	switch et.Kind() {
	case reflect.Slice:
		if et.Elem().Kind() == reflect.String {
			return detectStringResultKey(resp)
		}
		return detectArrayResultKey(resp)
	case reflect.String:
		return detectStringResultKey(resp)
	default:
		return ""
	}
}

func detectStringResultKey(resp jsonutils.JSONObject) string {
	dict, ok := resp.(*jsonutils.JSONDict)
	if !ok {
		return ""
	}
	for _, key := range dict.SortedKeys() {
		if responseMetaKeys[key] {
			continue
		}
		if strings.HasSuffix(key, "Id") || strings.HasSuffix(key, "Ids") {
			return key
		}
	}
	return ""
}

func detectArrayResultKey(resp jsonutils.JSONObject) string {
	dict, ok := resp.(*jsonutils.JSONDict)
	if !ok {
		return ""
	}
	candidates := []string{}
	for _, key := range dict.SortedKeys() {
		if responseMetaKeys[key] {
			continue
		}
		val, _ := dict.Get(key)
		if val == nil {
			continue
		}
		if _, ok := val.(*jsonutils.JSONArray); ok {
			candidates = append(candidates, key)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	for _, key := range candidates {
		if strings.HasSuffix(key, "Set") {
			return key
		}
	}
	return candidates[0]
}

func unmarshalResult(resp jsonutils.JSONObject, respErr error, result interface{}) error {
	if respErr != nil {
		return respErr
	}

	if result == nil {
		return nil
	}

	resultKey := detectResultKey(resp, result)
	if len(resultKey) > 0 {
		respErr = resp.Unmarshal(result, resultKey)
	} else {
		respErr = resp.Unmarshal(result)
	}

	if respErr != nil {
		log.Errorf("unmarshal json error %s", respErr)
	}

	return nil
}

func doListPart(client *SUcloudClient, action string, params SParams, result interface{}) (int, int, error) {
	params.SetAction(action)
	ret, err := jsonRequest(client, params)
	if err != nil {
		return 0, 0, err
	}

	total, _ := ret.Int("TotalCount")

	resultKey := detectResultKey(ret, result)
	lst, err := ret.GetArray(resultKey)
	if err != nil {
		return 0, 0, nil
	}

	resultValue := reflect.Indirect(reflect.ValueOf(result))
	elemType := resultValue.Type().Elem()
	for i := range lst {
		elemPtr := reflect.New(elemType)
		err = lst[i].Unmarshal(elemPtr.Interface())
		if err != nil {
			return 0, 0, err
		}
		resultValue.Set(reflect.Append(resultValue, elemPtr.Elem()))
	}
	return int(total), len(lst), nil
}

// 执行操作
func DoAction(client *SUcloudClient, action string, params SParams, result interface{}) error {
	params.SetAction(action)
	resp, err := jsonRequest(client, params)
	return unmarshalResult(resp, err, result)
}

// 遍历所有结果
func DoListAll(client *SUcloudClient, action string, params SParams, result interface{}) error {
	pageLimit := 100
	offset := 0

	resultValue := reflect.Indirect(reflect.ValueOf(result))
	params.SetPagination(pageLimit, offset)
	for {
		total, part, err := doListPart(client, action, params, result)
		if err != nil {
			return err
		}
		// total 大于零的情况下通过total字段判断列表是否遍历完成。total不存在或者为0的情况下，通过返回列表的长度判断是否遍历完成
		if (total > 0 && resultValue.Len() >= total) || (total == 0 && pageLimit > part) {
			break
		}

		params.SetPagination(pageLimit, offset+resultValue.Len())
	}
	return nil
}
