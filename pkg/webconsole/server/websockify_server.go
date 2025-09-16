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
	"encoding/base64"
	"fmt"
	"net"
	"net/http"

	"github.com/gorilla/websocket"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/webconsole/session"
)

const (
	BINARY_PROTOL = "binary"
	BASE64_PROTOL = "base64"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Subprotocols: []string{BINARY_PROTOL, BASE64_PROTOL},
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

func (s *WebsockifyServer) isBase64Subprotocol(wsConn *websocket.Conn) bool {
	return wsConn.Subprotocol() == BASE64_PROTOL
}

func (s *WebsockifyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	targetAddr := fmt.Sprintf("%s:%d", s.TargetHost, s.TargetPort)
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("New websocket connection error: %v", err)
		return
	}
	log.Debugf("Get coordinate subprotocol: %s", wsConn.Subprotocol())

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
	s.Session.RegisterDuplicateHook(func() {
		wsConn.Close()
		tcpConn.Close()
	})
	go s.wsToTcp(wsConn, tcpConn)
	s.tcpToWs(wsConn, tcpConn)
}

func (s *WebsockifyServer) ReadFromWs(wsConn *websocket.Conn) ([]byte, error) {
	_, data, err := wsConn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if s.isBase64Subprotocol(wsConn) {
		data, err = base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

func (s *WebsockifyServer) wsToTcp(wsConn *websocket.Conn, tcpConn net.Conn) {
	defer s.onExit(wsConn, tcpConn)

	for {
		data, err := s.ReadFromWs(wsConn)
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

func (s *WebsockifyServer) WriteToWs(wsConn *websocket.Conn, data []byte) error {
	msg := string(data)
	msgType := websocket.BinaryMessage
	if s.isBase64Subprotocol(wsConn) {
		msg = base64.StdEncoding.EncodeToString(data)
		msgType = websocket.TextMessage
	}
	return wsConn.WriteMessage(msgType, []byte(msg))
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

		err = s.WriteToWs(wsConn, buffer[0:n])
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
