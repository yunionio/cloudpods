package server

import (
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/websocket"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/webconsole/session"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Subprotocols: []string{"binary"},
}

type WebsockifyServer struct {
	Session    *session.SSession
	TargetHost string
	TargetPort int64
}

func NewWebsockifyServer(s *session.SSession) (*WebsockifyServer, error) {
	info := s.ISessionData.(*session.RemoteConsoleInfo)
	if info.Host == "" {
		return nil, fmt.Errorf("Empty remote host")
	}
	if info.Port <= 0 {
		return nil, fmt.Errorf("Invalid remote port: %d", info.Port)
	}

	server := &WebsockifyServer{
		Session:    s,
		TargetHost: info.Host,
		TargetPort: info.Port,
	}
	return server, nil
}

func (s *WebsockifyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	targetAddr := fmt.Sprintf("%s:%d", s.TargetHost, s.TargetPort)
	subs := websocket.Subprotocols(r)
	log.Debugf("Get subprotocols: %v", subs)
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("New websocket connection error: %v", err)
		return
	}

	log.Debugf("Handle websocket connect, target: %s", targetAddr)
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Errorf("Connection to target %s error: %v", targetConn, err)
		wsConn.Close()
		return
	}

	s.doProxy(wsConn, targetConn)
}

func (s *WebsockifyServer) doProxy(wsConn *websocket.Conn, tcpConn net.Conn) {
	go s.wsToTcp(wsConn, tcpConn)
	s.tcpToWs(wsConn, tcpConn)
}

func (s *WebsockifyServer) wsToTcp(wsConn *websocket.Conn, tcpConn net.Conn) {
	defer s.onExit(wsConn, tcpConn)

	for {
		_, data, err := wsConn.ReadMessage()
		if err != nil {
			log.Errorf("Read from websocket error: %v", err)
			return
		}

		_, err = tcpConn.Write(data)
		if err != nil {
			log.Errorf("Write to tcp socket error: %v", err)
			return
		}
	}
}

func (s *WebsockifyServer) tcpToWs(wsConn *websocket.Conn, tcpConn net.Conn) {
	defer s.onExit(wsConn, tcpConn)

	buffer := make([]byte, 1024)
	for {
		n, err := tcpConn.Read(buffer)
		if err != nil {
			log.Errorf("Read from tcp socket error: %v", err)
			return
		}

		err = wsConn.WriteMessage(websocket.BinaryMessage, buffer[0:n])
		if err != nil {
			log.Errorf("Write to websocket error: %v", err)
			return
		}
	}
}

func (s *WebsockifyServer) onExit(wsConn *websocket.Conn, tcpConn net.Conn) {
	wsConn.Close()
	tcpConn.Close()
	s.Session.Close()
}
