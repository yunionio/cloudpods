package huawei

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/huawei/client/manager"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

// 常用的方法
type listFunc func(querys map[string]string) (*responses.ListResult, error)
type getFunc func(id string, querys map[string]string) (jsonutils.JSONObject, error)
type createFunc func(params jsonutils.JSONObject) (jsonutils.JSONObject, error)
type updateFunc func(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
type updateFunc2 func(ctx manager.IManagerContext, id string, spec string, params jsonutils.JSONObject, responseKey string) (jsonutils.JSONObject, error)
type deleteFunc func(id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error)
type listInCtxFunc func(ctx manager.IManagerContext, querys map[string]string) (*responses.ListResult, error)
type listInCtxWithSpecFunc func(ctx manager.IManagerContext, spec string, querys map[string]string, responseKey string) (*responses.ListResult, error)

func unmarshalResult(resp jsonutils.JSONObject, respErr error, result interface{}) error {
	if respErr != nil {
		switch e := respErr.(type) {
		case *httputils.JSONClientError:
			if e.Code == 404 {
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

func DoList(doList listFunc, querys map[string]string, result interface{}) error {
	ret, err := doList(querys)
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

func DoGet(doGet getFunc, id string, querys map[string]string, result interface{}) error {
	if len(id) == 0 {
		return fmt.Errorf(" id should not be empty")
	}

	ret, err := doGet(id, querys)
	return unmarshalResult(ret, err, result)
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
	return unmarshalResult(ret, err, result)
}

func DoUpdate(updateFunc updateFunc, id string, params jsonutils.JSONObject, result interface{}) error {
	ret, err := updateFunc(id, params)
	return unmarshalResult(ret, err, result)
}

func DoUpdateWithSpec(updateFunc updateFunc2, id string, spec string, params jsonutils.JSONObject) error {
	_, err := updateFunc(nil, id, spec, params, "")
	return err
}

func DoDelete(deleteFunc deleteFunc, id string, params jsonutils.JSONObject, result interface{}) error {
	if len(id) == 0 {
		return fmt.Errorf(" id should not be empty")
	}

	ret, err := deleteFunc(id, params)
	return unmarshalResult(ret, err, result)
}
