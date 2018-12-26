package handler

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd/models/base"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func NewEtcdModelHandler(manger base.IEtcdModelManager) *SEtcdModelHandler {
	return &SEtcdModelHandler{
		manager: manger,
	}
}

type SEtcdModelHandler struct {
	manager base.IEtcdModelManager
}

func (disp *SEtcdModelHandler) Filter(f appsrv.FilterHandler) appsrv.FilterHandler {
	return auth.Authenticate(f)
}

func (disp *SEtcdModelHandler) Keyword() string {
	return disp.manager.Keyword()
}

func (disp *SEtcdModelHandler) KeywordPlural() string {
	return disp.manager.KeywordPlural()
}

func (disp *SEtcdModelHandler) ContextKeywordPlural() []string {
	return nil
}

func (disp *SEtcdModelHandler) CustomizeHandlerInfo(handler *appsrv.SHandlerInfo) {
	disp.manager.CustomizeHandlerInfo(handler)
}

func (disp *SEtcdModelHandler) FetchCreateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return disp.manager.FetchCreateHeaderData(ctx, header)
}

func (disp *SEtcdModelHandler) FetchUpdateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return disp.manager.FetchUpdateHeaderData(ctx, header)
}

func (disp *SEtcdModelHandler) List(ctx context.Context, query jsonutils.JSONObject, ctxId string) (*modules.ListResult, error) {
	objs, err := disp.manager.AllJson(ctx)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return &modules.ListResult{
		Data:   objs,
		Total:  len(objs),
		Limit:  0,
		Offset: 0,
	}, nil
}

func (disp *SEtcdModelHandler) Get(ctx context.Context, idstr string, query jsonutils.JSONObject, isHead bool) (jsonutils.JSONObject, error) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	model := disp.manager.Allocate()

	err := disp.manager.Get(ctx, idstr, model)

	// obj, err := disp.manager.GetJson(ctx, idstr)
	if err != nil {
		if err != etcd.ErrNoSuchKey {
			return nil, httperrors.NewGeneralError(err)
		} else {
			return nil, httperrors.NewResourceNotFoundError("%s %s not found", disp.manager.Keyword(), idstr)
		}
	}
	appParams := appsrv.AppContextGetParams(ctx)
	if appParams == nil && isHead {
		log.Errorf("fail to get http response writer???")
		return nil, httperrors.NewInternalServerError("fail to get http response writer from context")
	}
	hdrs := model.GetExtraDetailsHeaders(ctx, userCred, query)
	for k, v := range hdrs {
		appParams.Response.Header().Add(k, v)
	}

	if isHead {
		appParams.Response.Header().Add("Content-Length", "0")
		appParams.Response.Write([]byte{})
		return nil, nil
	}

	return jsonutils.Marshal(model), nil
}

func (disp *SEtcdModelHandler) GetSpecific(ctx context.Context, idstr string, spec string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)

	model := disp.manager.Allocate()

	err := disp.manager.Get(ctx, idstr, model)
	if err != nil {
		if err != etcd.ErrNoSuchKey {
			return nil, httperrors.NewGeneralError(err)
		} else {
			return nil, httperrors.NewResourceNotFoundError("%s %s not found", disp.manager.Keyword(), idstr)
		}
	}

	params := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(userCred),
		reflect.ValueOf(query),
	}

	specCamel := utils.Kebab2Camel(spec, "-")
	modelValue := reflect.ValueOf(model)

	funcName := fmt.Sprintf("GetDetails%s", specCamel)
	funcValue := modelValue.MethodByName(funcName)
	if !funcValue.IsValid() || funcValue.IsNil() {
		return nil, httperrors.NewSpecNotFoundError(fmt.Sprintf("%s %s %s not found", disp.Keyword(), idstr, spec))
	}

	outs := funcValue.Call(params)
	if len(outs) != 2 {
		return nil, httperrors.NewInternalServerError("Invald %s return value", funcName)
	}

	resVal := outs[0].Interface()
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	} else {
		if gotypes.IsNil(resVal) {
			return nil, nil
		} else {
			return resVal.(jsonutils.JSONObject), nil
		}
	}
}

func (disp *SEtcdModelHandler) Create(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, ctxId string) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotImplementedError("not implemented")
}

func (disp *SEtcdModelHandler) BatchCreate(ctx context.Context, query jsonutils.JSONObject, data jsonutils.JSONObject, count int, ctxId string) ([]modules.SubmitResult, error) {
	return nil, httperrors.NewNotImplementedError("not implemented")
}

func (disp *SEtcdModelHandler) PerformClassAction(ctx context.Context, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotImplementedError("not implemented")
}

func (disp *SEtcdModelHandler) PerformAction(ctx context.Context, idstr string, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotImplementedError("not implemented")
}

func (disp *SEtcdModelHandler) Update(ctx context.Context, idstr string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotImplementedError("not implemented")
}

func (disp *SEtcdModelHandler) Delete(ctx context.Context, idstr string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotImplementedError("not implemented")
}
