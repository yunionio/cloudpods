package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/baremetal"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	PREFIX = "baremetals"

	PARAMS_ID_KEY = "<id>"
)

func getBaremetalPrefix(action string) string {
	return fmt.Sprintf("%s/%s/%s", PREFIX, PARAMS_ID_KEY, action)
}

type handlerFunc func(ctx *Context)

func authMiddleware(h handlerFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		newCtx := NewContext(ctx, w, r)
		h(newCtx)
	}
}

type objectHandlerFunc func(ctx *Context, bm *baremetal.SBaremetalInstance)

func objectMiddleware(h objectHandlerFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		newCtx := NewContext(ctx, w, r)
		bmId := newCtx.Params()[PARAMS_ID_KEY]
		baremetal := newCtx.GetBaremetalManager().GetBaremetalById(bmId)
		if baremetal == nil {
			newCtx.ResponseError(httperrors.NewNotFoundError("Not found baremetal by id: %s", bmId))
			return
		}
		h(newCtx, baremetal)
	}
}

type Context struct {
	context.Context
	userCred mcclient.TokenCredential
	params   map[string]string
	query    jsonutils.JSONObject
	data     jsonutils.JSONObject
	request  *http.Request
	writer   http.ResponseWriter
}

func NewContext(ctx context.Context, w http.ResponseWriter, r *http.Request) *Context {
	params, query, body := appsrv.FetchEnv(ctx, w, r)
	return &Context{
		Context:  ctx,
		userCred: auth.FetchUserCredential(ctx, nil),
		params:   params,
		query:    query,
		data:     body,
		request:  r,
		writer:   w,
	}
}

func (ctx *Context) Params() map[string]string {
	return ctx.params
}

func (ctx *Context) Data() jsonutils.JSONObject {
	return ctx.data
}

func (ctx *Context) Query() jsonutils.JSONObject {
	return ctx.query
}

func (ctx *Context) UserCred() mcclient.TokenCredential {
	return ctx.userCred
}

func (ctx *Context) TaskId() string {
	return ctx.Request().Header.Get(mcclient.TASK_ID)
}

func (ctx *Context) ResponseStruct(obj interface{}) {
	appsrv.SendStruct(ctx.writer, obj)
}

func (ctx *Context) ResponseJson(obj jsonutils.JSONObject) {
	appsrv.SendJSON(ctx.writer, obj)
}

func (ctx *Context) ResponseError(err error) {
	httperrors.GeneralServerError(ctx.writer, err)
}

func (ctx *Context) Request() *http.Request {
	return ctx.request
}

func (ctx *Context) RequestRemoteIP() string {
	remoteAddr := ctx.Request().RemoteAddr
	return strings.Split(remoteAddr, ":")[0]
}

func (ctx *Context) ResponseOk() {
	obj := jsonutils.NewDict()
	obj.Add(jsonutils.NewString("ok"), "result")
	appsrv.SendJSON(ctx.writer, obj)
}

func (ctx *Context) GetBaremetalManager() *baremetal.SBaremetalManager {
	return baremetal.GetBaremetalManager()
}

func (ctx *Context) DelayProcess(process ProcessFunc) {
	DelayProcess(process, ctx.GetBaremetalManager().GetClientSession(), ctx.TaskId())
}
