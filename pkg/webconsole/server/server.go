package server

import (
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

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
		httperrors.BadRequestError(w, fmt.Sprintf("Empty access_token"))
		return
	}
	sessionObj, ok := session.Manager.Get(accessToken)
	if !ok {
		log.Warningf("Not found session by token: %q", accessToken)
		httperrors.NotFoundError(w, fmt.Sprintf("Not found session"))
		return
	}
	ttyServer, err := NewTTYServer(sessionObj)
	if err != nil {
		httperrors.GeneralServerError(w, fmt.Errorf("New TTY error: %v", err))
		return
	}
	ttyServer.ServeHTTP(w, req)
}
