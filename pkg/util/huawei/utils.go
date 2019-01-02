package huawei

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/huawei/client/manager"
	"yunion.io/x/onecloud/pkg/util/huawei/client/responses"
)

// 常用的方法
type listFunc func(querys map[string]string) (*responses.ListResult, error)
type getFunc func(id string, querys map[string]string) (jsonutils.JSONObject, error)
type listInCtxFunc func(ctx *manager.ManagerContext, spec string, querys map[string]string) (*responses.ListResult, error)

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
	if err != nil {
		return err
	}

	err = ret.Unmarshal(result)
	if err != nil {
		log.Errorf("unmarshal json error %s", err)
		return err
	}

	return nil
}

func DoListInContext(listFunc listInCtxFunc, ctx *manager.ManagerContext, querys map[string]string, result interface{}) error {
	ret, err := listFunc(ctx, "", querys)
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
