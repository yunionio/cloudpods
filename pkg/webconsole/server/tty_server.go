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
	"github.com/creack/pty"
	socketio "github.com/googollee/go-socket.io"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/webconsole/session"
)

const (
	ON_CONNECTION    = "connection"
	ON_DISCONNECTION = "disconnection"
	ON_ERROR         = "error"

	DISCONNECT_EVENT = "disconnect"
	OUTPUT_EVENT     = "output"
	INPUT_EVENT      = "input"
	RESIZE_EVENT     = "resize"

	COMMAND_QUERY = "command"
	ARGS_QUERY    = "args"
)

type TTYServer struct {
	*socketio.Server
}

func NewTTYServer(s *session.SSession) (*TTYServer, error) {
	socketioServer, err := socketio.NewServer(nil)
	if err != nil {
		return nil, err
	}
	server := &TTYServer{
		Server: socketioServer,
	}
	server.initEventHandler(s)
	return server, nil
}

func (server *TTYServer) initEventHandler(s *session.SSession) {
	server.On(ON_CONNECTION, func(so socketio.Socket) error {
		log.Infof("[%q] On connection", so.Id())
		p, err := session.NewPty(s)
		if err != nil {
			log.Errorf("Create Pty error: %v", err)
			return err
		}
		initSocketHandler(so, p)
		return nil
	})
}

func initSocketHandler(so socketio.Socket, p *session.Pty) {
	// handle command output
	go func() {
		for !p.Exit {
			if p.IsInShellMode() {
				data, err := p.Read()
				if err != nil {
					log.Errorf("[%s] read data error: %v", so.Id(), err)
					cleanUp(so, p)
				} else {
					// log.Errorf("--p.Pty.output data: %q", data)
					so.Emit(OUTPUT_EVENT, string(data))
					go p.Session.GetRecorder().Write("", string(data))
				}
				continue
			}
		}
	}()

	// handle user input write
	so.On(INPUT_EVENT, func(data string) {
		if !p.IsInShellMode() {
			for _, d := range []byte(data) {
				p.Session.Scan(d, func(msg string) {
					if len(msg) > 0 {
						so.Emit(OUTPUT_EVENT, msg)
					}
				})
			}
			cmd := p.Session.GetCommand()
			if cmd != nil {
				pty, err := pty.Start(cmd)
				if err != nil {
					log.Errorf("failed to start cmd: %v, error: %v", cmd, err)
					so.Emit(OUTPUT_EVENT, err.Error()+"\r\n")
					return
				}
				p.Pty, p.Cmd = pty, cmd
				if p.OriginSize != nil {
					p.Resize(p.OriginSize)
				}
			}
		} else {
			p.Pty.Write([]byte(data))
			go p.Session.GetRecorder().Write(data, "")
		}
	})

	// handle resize
	so.On(RESIZE_EVENT, func(colRow []uint16) {
		if len(colRow) != 2 {
			log.Errorf("Invalid window size: %v", colRow)
			cleanUp(so, p)
			return
		}
		//size, err := pty.GetsizeFull(p.Pty)
		//if err != nil {
		//log.Errorf("Get pty window size error: %v", err)
		//return
		//}
		newSize := pty.Winsize{
			Cols: colRow[0],
			Rows: colRow[1],
		}
		p.Resize(&newSize)
	})

	// handle disconnection
	so.On(ON_DISCONNECTION, func(msg string) {
		log.Infof("[%s] closed: %s", so.Id(), msg)
		cleanUp(so, p)
	})

	// handle error
	so.On(ON_ERROR, func(err error) {
		log.Errorf("[%s] on error: %v", so.Id(), err)
		cleanUp(so, p)
	})
}

func cleanUp(so socketio.Socket, p *session.Pty) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("recover error: %v", err)
		}
	}()
	so.Disconnect()
	p.Stop()
	p.Exit = true
}
