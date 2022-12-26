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

package hcso

import (
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/manager"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/responses"
)

// 常用的方法
type listFunc func(querys map[string]string) (*responses.ListResult, error)
type getFunc func(id string, querys map[string]string) (jsonutils.JSONObject, error)
type createFunc func(params jsonutils.JSONObject) (jsonutils.JSONObject, error)
type updateFunc func(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
type updateFunc2 func(ctx manager.IManagerContext, id string, spec string, params jsonutils.JSONObject, responseKey string) (jsonutils.JSONObject, error)
type deleteFunc func(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
type deleteFunc2 func(ctx manager.IManagerContext, id string, spec string, queries map[string]string, params jsonutils.JSONObject, responseKey string) (jsonutils.JSONObject, error)
type listInCtxFunc func(ctx manager.IManagerContext, querys map[string]string) (*responses.ListResult, error)
type listInCtxWithSpecFunc func(ctx manager.IManagerContext, spec string, querys map[string]string, responseKey string) (*responses.ListResult, error)

func unmarshalResult(resp jsonutils.JSONObject, respErr error, result interface{}, method string) error {
	if respErr != nil {
		switch e := respErr.(type) {
		case *httputils.JSONClientError:
			if (e.Code == 404 || utils.IsInStringArray(e.Class, NOT_FOUND_CODES)) && method != "POST" {
				return cloudprovider.ErrNotFound
			}
			return e
		default:
			return e
		}
	}

	if result == nil {
		return nil
	}

	err := resp.Unmarshal(result)
	if err != nil {
		log.Errorf("unmarshal json error %s", err)
	}

	return err
}

var pageLimit = 100

// offset 表示的是页码
func doListAllWithPagerOffset(doList listFunc, queries map[string]string, result interface{}) error {
	startIndex := 0
	resultValue := reflect.Indirect(reflect.ValueOf(result))
	queries["limit"] = fmt.Sprintf("%d", pageLimit)
	queries["offset"] = fmt.Sprintf("%d", startIndex)
	for {
		total, part, err := doListPart(doList, queries, result)
		if err != nil {
			return err
		}
		if (total > 0 && resultValue.Len() >= total) || (total == 0 && pageLimit > part) {
			break
		}

		startIndex++
		queries["offset"] = fmt.Sprintf("%d", startIndex)
	}
	return nil
}

func doListAllWithNextLink(doList listFunc, querys map[string]string, result interface{}) error {
	values := []jsonutils.JSONObject{}
	for {
		ret, err := doList(querys)
		if err != nil {
			return errors.Wrap(err, "doList")
		}
		values = append(values, ret.Data...)
		if len(ret.NextLink) == 0 || ret.NextLink == "null" {
			break
		}
	}
	if result != nil {
		return jsonutils.Update(result, values)
	}
	return nil
}

func doListAllWithOffset(doList listFunc, queries map[string]string, result interface{}) error {
	startIndex := 0
	resultValue := reflect.Indirect(reflect.ValueOf(result))
	queries["limit"] = fmt.Sprintf("%d", pageLimit)
	queries["offset"] = fmt.Sprintf("%d", startIndex)
	for {
		total, part, err := doListPart(doList, queries, result)
		if err != nil {
			return err
		}
		if (total > 0 && resultValue.Len() >= total) || (total == 0 && pageLimit > part) {
			break
		}
		queries["offset"] = fmt.Sprintf("%d", startIndex+resultValue.Len())
	}
	return nil
}

func doListAllWithPage(doList listFunc, queries map[string]string, result interface{}) error {
	startIndex := 1
	resultValue := reflect.Indirect(reflect.ValueOf(result))
	queries["limit"] = fmt.Sprintf("%d", pageLimit)
	queries["page"] = fmt.Sprintf("%d", startIndex)
	for {
		total, part, err := doListPart(doList, queries, result)
		if err != nil {
			return err
		}
		if (total > 0 && resultValue.Len() >= total) || (total == 0 && pageLimit > part) {
			break
		}
		queries["page"] = fmt.Sprintf("%d", startIndex+resultValue.Len())
	}
	return nil
}

func doListAllWithMarker(doList listFunc, queries map[string]string, result interface{}) error {
	resultValue := reflect.Indirect(reflect.ValueOf(result))
	queries["limit"] = fmt.Sprintf("%d", pageLimit)
	for {
		total, part, err := doListPart(doList, queries, result)
		if err != nil {
			return err
		}
		if (total > 0 && resultValue.Len() >= total) || (total == 0 && pageLimit > part) {
			break
		}
		lastValue := resultValue.Index(resultValue.Len() - 1)
		markerValue := lastValue.FieldByNameFunc(func(key string) bool {
			if strings.ToLower(key) == "id" {
				return true
			}
			return false
		})
		queries["marker"] = markerValue.String()
	}
	return nil
}

func doListAll(doList listFunc, queries map[string]string, result interface{}) error {
	total, _, err := doListPart(doList, queries, result)
	if err != nil {
		return err
	}
	resultValue := reflect.Indirect(reflect.ValueOf(result))
	if total > 0 && resultValue.Len() < total {
		log.Warningf("INCOMPLETE QUERY, total %d queried %d", total, resultValue.Len())
	}
	return nil
}

func doListPart(doList listFunc, queries map[string]string, result interface{}) (int, int, error) {
	ret, err := doList(queries)
	if err != nil {
		return 0, 0, err
	}
	resultValue := reflect.Indirect(reflect.ValueOf(result))
	elemType := resultValue.Type().Elem()
	for i := range ret.Data {
		elemPtr := reflect.New(elemType)
		err = ret.Data[i].Unmarshal(elemPtr.Interface())
		if err != nil {
			return 0, 0, err
		}
		resultValue.Set(reflect.Append(resultValue, elemPtr.Elem()))
	}
	return ret.Total, len(ret.Data), nil
}

func DoGet(doGet getFunc, id string, queries map[string]string, result interface{}) error {
	if len(id) == 0 {
		resultType := reflect.Indirect(reflect.ValueOf(result)).Type()
		return errors.Wrap(cloudprovider.ErrNotFound, fmt.Sprintf(" Get %s id should not be empty", resultType.Name()))
	}

	ret, err := doGet(id, queries)
	return unmarshalResult(ret, err, result, "GET")
}

func DoListInContext(listFunc listInCtxFunc, ctx manager.IManagerContext, querys map[string]string, result interface{}) error {
	ret, err := listFunc(ctx, querys)
	if err != nil {
		return err
	}

	obj := responses.ListResult2JSON(ret)
	err = obj.Unmarshal(result, "data")
	if err != nil {
		log.Errorf("unmarshal json error %s", err)
		return err
	}

	return nil
}

func DoCreate(createFunc createFunc, params jsonutils.JSONObject, result interface{}) error {
	ret, err := createFunc(params)
	return unmarshalResult(ret, err, result, "POST")
}

func DoUpdate(updateFunc updateFunc, id string, params jsonutils.JSONObject, result interface{}) error {
	ret, err := updateFunc(id, params)
	return unmarshalResult(ret, err, result, "PUT")
}

func DoUpdateWithSpec(updateFunc updateFunc2, id string, spec string, params jsonutils.JSONObject) error {
	_, err := updateFunc(nil, id, spec, params, "")
	return err
}

func DoUpdateWithSpec2(updateFunc updateFunc2, id string, spec string, params jsonutils.JSONObject, result interface{}) error {
	ret, err := updateFunc(nil, id, spec, params, "")
	return unmarshalResult(ret, err, result, "PUT")
}

func DoDelete(deleteFunc deleteFunc, id string, params jsonutils.JSONObject, result interface{}) error {
	if len(id) == 0 {
		return fmt.Errorf(" id should not be empty")
	}

	ret, err := deleteFunc(id, params)
	return unmarshalResult(ret, err, result, "DELETE")
}

func DoDeleteWithSpec(deleteFunc deleteFunc2, ctx manager.IManagerContext, id string, spec string, queries map[string]string, params jsonutils.JSONObject) error {
	if len(id) == 0 {
		return fmt.Errorf(" id should not be empty")
	}

	_, err := deleteFunc(ctx, id, spec, queries, params, "")
	return err
}

func ErrMessage(err error) string {
	switch v := err.(type) {
	case *httputils.JSONClientError:
		return fmt.Sprintf("%d(%s):%s", v.Code, v.Class, v.Details)
	default:
		return err.Error()
	}
}
