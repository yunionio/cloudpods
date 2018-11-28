package server

import (
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/webconsole/session"
)

type ConnectionServer struct {
}

func NewConnectionServer() *ConnectionServer {
	return &ConnectionServer{}
}

func (s *ConnectionServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	query, err := jsonutils.ParseQueryString(req.URL.RawQuery)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	log.Debugf("[connection] Get query: %v", query)
	accessToken, _ := query.GetString("access_token")
	if accessToken == "" {
		httperrors.BadRequestError(w, "Empty access_token")
		return
	}
	sessionObj, ok := session.Manager.Get(accessToken)
	if !ok {
		httperrors.NotFoundError(w, "session not found")
		return
	}
	var srv http.Handler
	protocol := sessionObj.GetProtocol()
	if sets.NewString(session.VNC, session.SPICE, session.WMKS).Has(protocol) {
		srv, err = NewWebsockifyServer(sessionObj)
	} else {
		srv, err = NewTTYServer(sessionObj)
	}
	if err != nil {
		httperrors.GeneralServerError(w, "New server error: %v", err)
		return
	}
	srv.ServeHTTP(w, req)
}
