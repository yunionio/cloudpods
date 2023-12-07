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
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/webconsole/guac"
	"yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/session"
)

type RDPServer struct {
	Session *session.SSession

	Host         string
	Port         int
	Username     string
	Password     string
	ConnectionId string

	Width  int
	Height int
	Dpi    int
}

func NewRDPServer(s *session.SSession) (*RDPServer, error) {
	info := s.ISessionData.(*session.RemoteRDPConsoleInfo)
	server := &RDPServer{
		Session:      s,
		Host:         info.Host,
		Port:         info.Port,
		Username:     info.Username,
		Password:     info.Password,
		ConnectionId: info.ConnectionId,

		Width:  info.Width,
		Height: info.Height,
		Dpi:    info.Dpi,
	}
	return server, nil
}

func (s *RDPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var up = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	ws, err := up.Upgrade(w, r, http.Header{"Sec-Websocket-Protocol": []string{
		r.Header.Get("Sec-Websocket-Protocol"),
	}})
	if err != nil {
		log.Errorf("upgrade error: %v", err)
		return
	}

	defer ws.Close()

	tunnel, err := guac.NewGuacamoleTunnel(
		s.Host,
		s.Port,
		s.Username,
		s.Password,
		s.ConnectionId,
		s.Width,
		s.Height,
		s.Dpi,
		s.Session.GetClientSession().GetUserId(),
	)
	if err != nil {
		log.Errorf("NewGuacamoleTunnel error: %v", err)
		return
	}

	err = tunnel.Start()
	if err != nil {
		log.Errorf("Start error: %v", err)
		return
	}

	done := make(chan bool, 4)
	timer := time.NewTimer(time.Microsecond * 100)
	setDone := func() {
		done <- true
	}

	go func() {
		defer setDone()

		for {
			ins, err := tunnel.ReadOne()
			if err != nil {
				return
			}
			if options.Options.RdpSessionTimeoutMinutes > 0 && timer != nil {
				timer.Reset(time.Duration(options.Options.RdpSessionTimeoutMinutes) * time.Minute)
			}
			err = ws.WriteMessage(websocket.TextMessage, []byte(ins.String()))
			if err != nil {
				log.Errorf("Failed writing to guacd %s: %v", ins.String(), err)
				return
			}
		}
	}()

	go func() {
		defer setDone()
		defer tunnel.Stop()

		for {
			_, p, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				log.Errorf("read message error %v", err)
				return
			}
			if options.Options.RdpSessionTimeoutMinutes > 0 && timer != nil {
				timer.Reset(time.Duration(options.Options.RdpSessionTimeoutMinutes) * time.Minute)
			}
			_, err = tunnel.Write(p)
			if err != nil {
				log.Errorf("Failed writing to guacd: %v", err)
				return
			}
		}
	}()

	stop := make(chan bool)
	go func() {
		if options.Options.RdpSessionTimeoutMinutes > 0 {
			timer.Reset(time.Duration(options.Options.RdpSessionTimeoutMinutes) * time.Minute)
		}
		defer timer.Stop()
		defer setDone()

		for {
			select {
			case <-stop:
				return
			case <-timer.C:
				if options.Options.RdpSessionTimeoutMinutes > 0 {
					return
				}
				timer.Reset(time.Microsecond * 100)
			}
		}
	}()

	go func() {
		defer setDone()

		err = tunnel.Wait()
		if err != nil && errors.Cause(err) != guac.TunnerClose {
			log.Errorf("wait error: %v", err)
		}
	}()

	<-done
	stop <- true
	log.Infof("rdp %s@%s:%d complete", s.Username, s.Host, s.Port)
}
