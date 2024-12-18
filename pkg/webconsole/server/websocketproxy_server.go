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
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/koding/websocketproxy"

	"yunion.io/x/onecloud/pkg/webconsole/session"
)

type WebsocketProxyServer struct {
	Session *session.SSession
	proxy   *websocketproxy.WebsocketProxy
	cookie  string
}

func NewWebsocketProxyServer(s *session.SSession) (*WebsocketProxyServer, error) {
	info := s.ISessionData.(*session.RemoteConsoleInfo)
	if info.Url == "" {
		return nil, fmt.Errorf("Empty proxy url")
	}
	u, err := url.Parse(info.Url)
	if err != nil {
		return nil, fmt.Errorf("Parse url %s: %v", info.Url, err)
	}
	proxySrv := websocketproxy.NewProxy(u)
	proxySrv.Dialer = &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	proxySrv.Backend = func(_ *http.Request) *url.URL {
		return u
	}
	proxySrv.Upgrader = &upgrader
	return &WebsocketProxyServer{
		Session: s,
		proxy:   proxySrv,
		cookie:  info.Cookie,
	}, nil
}

func (s *WebsocketProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(s.cookie) > 0 {
		r.Header.Set("Cookie", s.cookie)
	}
	s.proxy.ServeHTTP(w, r)
}
