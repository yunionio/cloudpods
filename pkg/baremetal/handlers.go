package baremetal

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func InitHandlers(app *appsrv.Application) {
	initBaremetalsHandler(app)
}

func initBaremetalsHandler(app *appsrv.Application) {
	prefix := "baremetals"
	app.AddHandler("GET", fmt.Sprintf("%s/<id>/notify", prefix), authMiddleware(handleBaremetalNotify))
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

func (ctx *Context) ResponseOk() {
	obj := jsonutils.NewDict()
	obj.Add(jsonutils.NewString("ok"), "result")
	appsrv.SendJSON(ctx.writer, obj)
}

type handlerFunc func(ctx *Context)

func authMiddleware(h handlerFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		newCtx := NewContext(ctx, w, r)
		h(newCtx)
	}
}

func handleBaremetalNotify(ctx *Context) {
	bmId := ctx.Params()["<id>"]
	key, err := ctx.Query().GetString("key")
	if err != nil {
		ctx.ResponseError(httperrors.NewInputParameterError("Not found key in query"))
		return
	}
	remoteAddr := ctx.Request().RemoteAddr
	log.Debugf("===Get key %q from remote address: %s, bmId: %s", key, remoteAddr, bmId)
	ctx.ResponseOk()
}
