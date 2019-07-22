package aliyun

import (
	"fmt"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

type jsonRequestFunc func(action string, params map[string]string) (jsonutils.JSONObject, error)

func unmarshalResult(resp jsonutils.JSONObject, respErr error, resultKey []string, result interface{}) error {
	if respErr != nil {
		return respErr
	}

	if result == nil {
		return nil
	}

	if resultKey != nil && len(resultKey) > 0 {
		respErr = resp.Unmarshal(result, resultKey...)
	} else {
		respErr = resp.Unmarshal(result)
	}

	if respErr != nil {
		log.Errorf("unmarshal json error %s", respErr)
	}

	return nil
}

func doListPart(client jsonRequestFunc, action string, limit int, offset int, params map[string]string, resultKey []string, result interface{}) (int, int, error) {
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	ret, err := client(action, params)
	if err != nil {
		return 0, 0, err
	}

	total, _ := ret.Int("TotalCount")

	var lst []jsonutils.JSONObject
	lst, err = ret.GetArray(resultKey...)
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
func DoAction(client jsonRequestFunc, action string, params map[string]string, resultKey []string, result interface{}) error {
	resp, err := client(action, params)
	return unmarshalResult(resp, err, resultKey, result)
}

// 遍历所有结果
func DoListAll(client jsonRequestFunc, action string, params map[string]string, resultKey []string, result interface{}) error {
	pageLimit := 50
	offset := 0

	resultValue := reflect.Indirect(reflect.ValueOf(result))
	for {
		total, part, err := doListPart(client, action, pageLimit, offset, params, resultKey, result)
		if err != nil {
			return err
		}

		// total 大于零的情况下通过total字段判断列表是否遍历完成。total不存在或者为0的情况下，通过返回列表的长度判断是否遍历完成
		if (total > 0 && resultValue.Len() >= total) || (total == 0 && pageLimit > part) {
			break
		}

		offset = resultValue.Len()
	}

	return nil
}
