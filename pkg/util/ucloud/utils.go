package ucloud

import (
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

func unmarshalResult(resp jsonutils.JSONObject, respErr error, resultKey string, result interface{}) error {
	if respErr != nil {
		return respErr
	}

	if result == nil {
		return nil
	}

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

func doListPart(client *SUcloudClient, action string, params SParams, resultKey string, result interface{}) (int, int, error) {
	params.SetAction(action)
	ret, err := jsonRequest(client, params)
	if err != nil {
		return 0, 0, err
	}

	total, err := ret.Int("TotalCount")
	if err != nil {
		log.Debugf("%s TotalCount %s", action, err.Error())
	}

	var lst []jsonutils.JSONObject
	lst, err = ret.GetArray(resultKey)
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
func DoAction(client *SUcloudClient, action string, params SParams, resultKey string, result interface{}) error {
	params.SetAction(action)
	resp, err := jsonRequest(client, params)
	return unmarshalResult(resp, err, resultKey, result)
}

// 遍历所有结果
func DoListAll(client *SUcloudClient, action string, params SParams, resultKey string, result interface{}) error {
	pageLimit := 100
	offset := 0

	resultValue := reflect.Indirect(reflect.ValueOf(result))
	params.SetPagination(pageLimit, offset)
	for {
		total, part, err := doListPart(client, action, params, resultKey, result)
		if err != nil {
			return err
		}
		if (total > 0 && resultValue.Len() >= total) || (total == 0 && pageLimit > part) {
			break
		}

		params.SetPagination(pageLimit, offset+resultValue.Len())
	}
	return nil
}
