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
	"net"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

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

func bmRegisterPrefix() string {
	// baremetals/register-baremetal
	return fmt.Sprintf("%s/register-baremetal", BM_PREFIX)
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
	return bmObjMiddlewareWithFetch(h, true)
}

func bmObjMiddlewareWithFetch(h bmObjHandlerFunc, fetch bool) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		newCtx := NewContext(ctx, w, r)
		bmId := newCtx.Params()[PARAMS_BMID_KEY]
		baremetal := newCtx.GetBaremetalManager().GetBaremetalById(bmId)
		if baremetal == nil {
			if fetch {
				err := newCtx.GetBaremetalManager().InitBaremetal(bmId, false)
				if err != nil {
					newCtx.ResponseError(err)
					return
				}
				baremetal = newCtx.GetBaremetalManager().GetBaremetalById(bmId)
			} else {
				newCtx.ResponseError(httperrors.NewNotFoundError("Not found baremetal by id: %s", bmId))
				return
			}
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

type bmRegisterFunc func(*Context, *baremetal.BmRegisterInput)

func bmRegisterMiddleware(h bmRegisterFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		newCtx := NewContext(ctx, w, r)
		sshPasswd, _ := newCtx.Data().GetString("ssh_password")
		if len(sshPasswd) == 0 {
			sshPasswd = "yunion@123"
		}
		hostname, err := newCtx.Data().GetString("hostname")
		if err != nil {
			newCtx.ResponseError(httperrors.NewMissingParameterError("hostname"))
			return
		}
		remoteIp, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			newCtx.ResponseError(httperrors.NewInternalServerError("Parse ip error %s", err))
			return
		}
		sshPort, err := newCtx.Data().Int("ssh_port")
		if err != nil {
			newCtx.ResponseError(httperrors.NewMissingParameterError("ssh_port"))
			return
		}
		username, err := newCtx.Data().GetString("username")
		if err != nil {
			newCtx.ResponseError(httperrors.NewMissingParameterError("username"))
			return
		}
		password, err := newCtx.Data().GetString("password")
		if err != nil {
			newCtx.ResponseError(httperrors.NewMissingParameterError("password"))
			return
		}

		ctx, cancel := context.WithTimeout(ctx, time.Second*298)
		defer cancel()

		data := &baremetal.BmRegisterInput{
			Ctx:       ctx,
			R:         r,
			W:         w,
			C:         make(chan struct{}),
			SshPort:   int(sshPort),
			SshPasswd: sshPasswd,
			Hostname:  hostname,
			RemoteIp:  remoteIp,
			Username:  username,
			Password:  password,
		}

		h(newCtx, data)

		select {
		case <-ctx.Done():
			log.Errorln("Register timeout")
			newCtx.ResponseError(httperrors.NewTimeoutError("RegisterTimeOut"))
		case <-data.C:
			return
		}
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
