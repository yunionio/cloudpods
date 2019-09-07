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

package handler

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type RPCHandlers struct {
	*SHandlers
}

func NewRPCHandlers(prefix string) *RPCHandlers {
	return &RPCHandlers{NewHandlers(prefix)}
}

func (h *RPCHandlers) AddGet(mf appsrv.MiddlewareFunc) *RPCHandlers {
	h.AddByMethod(GET, mf, NewHP(rpcHandler, APIVer, "rpc"))
	return h
}

func (h *RPCHandlers) AddPost(mf appsrv.MiddlewareFunc) *RPCHandlers {
	h.AddByMethod(POST, mf, NewHP(rpcHandler, APIVer, "rpc"))
	return h
}

func rpcHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	curpath := appctx.AppContextCurrentPath(ctx)
	var resType string
	var resId string
	var callName string
	resType = curpath[0]
	if len(curpath) == 2 {
		callName = curpath[1]
	} else {
		resId = curpath[1]
		callName = curpath[2]
	}
	var e error
	var verb string
	var params jsonutils.JSONObject = nil
	switch req.Method {
	case "GET":
		verb = "Get"
		params, e = jsonutils.ParseQueryString(req.URL.RawQuery)
		if e != nil {
			log.Errorf("Error parse query string: %s", e)
		}
	case "POST":
		verb = "Do"
		params, e = appsrv.FetchJSON(req)
		if e != nil {
			log.Errorf("Error get JSON body: %s", e)
		}
	default:
		httperrors.InvalidInputError(w, fmt.Sprintf("Unsupported RPC method %s", req.Method))
		return
	}
	token := AppContextToken(ctx)
	pathParams := appctx.AppContextParams(ctx)
	s := auth.GetSession(ctx, token, FetchRegion(req), pathParams["<apiver>"])
	funcname := verb + utils.Kebab2Camel(callName, "-")
	mod, e := modulebase.GetModule(s, resType)
	if e != nil || mod == nil {
		if e != nil {
			log.Debugf("module %s not found %s", resType, e)
		}
		httperrors.NotFoundError(w, fmt.Sprintf("resource %s not exists", resType))
		return
	}
	modvalue := reflect.ValueOf(mod)
	funcvalue := modvalue.MethodByName(funcname)
	if !funcvalue.IsValid() || funcvalue.IsNil() {
		httperrors.NotFoundError(w, fmt.Sprintf("RPC method %s not found", funcname))
		return
	}
	callParams := make([]reflect.Value, 0)
	callParams = append(callParams, reflect.ValueOf(s))
	if len(resId) > 0 {
		callParams = append(callParams, reflect.ValueOf(resId))
	}
	if params == nil {
		params = jsonutils.NewDict()
	}
	callParams = append(callParams, reflect.ValueOf(params))
	log.Debugf("%s", callParams)
	retValue := funcvalue.Call(callParams)
	retobj := retValue[0]
	reterr := retValue[1]
	if reterr.IsNil() {
		v, ok := retobj.Interface().(jsonutils.JSONObject)
		if ok {
			appsrv.SendJSON(w, v)
		} else {
			httperrors.BadGatewayError(w, "recv invalid data")
		}
	} else {
		v, ok := reterr.Interface().(*httputils.JSONClientError)
		if ok {
			httperrors.JsonClientError(w, v)
		} else {
			httperrors.BadGatewayError(w, fmt.Sprintf("%s", reterr.Interface()))
		}
	}
}
