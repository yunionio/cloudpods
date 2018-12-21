package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/baremetal"
	baremetaltypes "yunion.io/x/onecloud/pkg/baremetal/types"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	BM_PREFIX     = "baremetals"
	SERVER_PREFIX = "servers"

	PARAMS_BMID_KEY  = "<bm_id>"
	PARAMS_SRVID_KEY = "<srv_id>"
)

func bmIdPrefix() string {
	// baremetals/<bm_id>
	return fmt.Sprintf("%s/%s", BM_PREFIX, PARAMS_BMID_KEY)
}

func bmActionPrefix(action string) string {
	// baremetals/<bm_id>/action
	return fmt.Sprintf("%s/%s", bmIdPrefix(), action)
}

func srvIdPrefix() string {
	// baremetals/<bm_id>/servers/<srv_id>
	return fmt.Sprintf("%s/%s/%s", bmIdPrefix(), SERVER_PREFIX, PARAMS_SRVID_KEY)
}

func srvActionPrefix(action string) string {
	// baremetals/<bm_id>/servers/<srv_id>/action
	return fmt.Sprintf("%s/%s", srvIdPrefix(), action)
}

type handlerFunc func(ctx *Context)

func authMiddleware(h handlerFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		newCtx := NewContext(ctx, w, r)
		h(newCtx)
	}
}

type bmObjHandlerFunc func(ctx *Context, bm *baremetal.SBaremetalInstance)

func bmObjMiddleware(h bmObjHandlerFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		newCtx := NewContext(ctx, w, r)
		bmId := newCtx.Params()[PARAMS_BMID_KEY]
		baremetal := newCtx.GetBaremetalManager().GetBaremetalById(bmId)
		if baremetal == nil {
			newCtx.ResponseError(httperrors.NewNotFoundError("Not found baremetal by id: %s", bmId))
			return
		}
		h(newCtx, baremetal)
	}
}

type srvObjHandlerFunc func(ctx *Context, bm *baremetal.SBaremetalInstance, srv baremetaltypes.IBaremetalServer)

func srvClassMiddleware(h bmObjHandlerFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		newCtx := NewContext(ctx, w, r)
		bmId := newCtx.Params()[PARAMS_BMID_KEY]
		//srvId := newCtx.Params()[PARAMS_SRVID_KEY]
		baremetal := newCtx.GetBaremetalManager().GetBaremetalById(bmId)
		if baremetal == nil {
			newCtx.ResponseError(httperrors.NewNotFoundError("Not found baremetal by id: %s", bmId))
			return
		}
		if baremetal.GetServerId() != "" {
			newCtx.ResponseError(httperrors.NewNotAcceptableError("Baremetal %s occupied", bmId))
			return
		}
		h(newCtx, baremetal)
	}
}

func srvObjMiddleware(h srvObjHandlerFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		newCtx := NewContext(ctx, w, r)
		bmId := newCtx.Params()[PARAMS_BMID_KEY]
		srvId := newCtx.Params()[PARAMS_SRVID_KEY]
		baremetal := newCtx.GetBaremetalManager().GetBaremetalById(bmId)
		if baremetal == nil {
			newCtx.ResponseError(httperrors.NewNotFoundError("Not found baremetal by id: %s", bmId))
			return
		}
		if baremetal.GetServerId() != srvId {
			newCtx.ResponseError(httperrors.NewNotFoundError("Not found server by id: %s", srvId))
			return
		}
		srv := baremetal.GetServer()
		h(newCtx, baremetal, srv)
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

func (ctx *Context) DelayProcess(process ProcessFunc, data jsonutils.JSONObject) {
	DelayProcess(process, ctx.GetBaremetalManager().GetClientSession(), ctx.TaskId(), data)
}
