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

package server

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/webconsole/session"
)

type ConnectionServer struct {
}

func NewConnectionServer() *ConnectionServer {
	return &ConnectionServer{}
}

func (s *ConnectionServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	ctx = appctx.WithRequestLang(ctx, req)
	query, err := jsonutils.ParseQueryString(req.URL.RawQuery)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	log.Debugf("[connection] Get query: %v", query)
	accessToken, _ := query.GetString("access_token")
	if accessToken == "" {
		httperrors.BadRequestError(ctx, w, "Empty access_token")
		return
	}
	sessionObj, ok := session.Manager.Get(accessToken)
	if !ok {
		httperrors.NotFoundError(ctx, w, "session not found")
		return
	}
	var srv http.Handler
	protocol := sessionObj.GetProtocol()
	switch protocol {
	case session.VNC, session.SPICE:
		info := sessionObj.ISessionData.(*session.RemoteConsoleInfo)
		if utils.IsInStringArray(info.Hypervisor, []string{
			api.HYPERVISOR_OPENSTACK,
			api.HYPERVISOR_CTYUN,
			api.HYPERVISOR_SANGFOR,
		}) {
			srv, err = NewWebsocketProxyServer(sessionObj)
		} else {
			srv, err = NewWebsockifyServer(sessionObj)
		}
	case session.WMKS:
		srv, err = NewWebsocketProxyServer(sessionObj)
	case session.WS:
		srv, err = NewSshServer(sessionObj)
	case session.RDP:
		srv, err = NewRDPServer(sessionObj)
	default:
		srv, err = NewTTYServer(sessionObj)
	}
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	srv.ServeHTTP(w, req)
}
