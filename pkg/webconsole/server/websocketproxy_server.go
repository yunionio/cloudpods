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
	}, nil
}

func (s *WebsocketProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}
