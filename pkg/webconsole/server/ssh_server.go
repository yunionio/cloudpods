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
	"io"
	"net/http"
	"time"

	"github.com/anacrolix/sync"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/session"
)

type WebsocketServer struct {
	Session    *session.SSession
	Host       string
	Port       int
	Username   string
	Password   string
	PrivateKey string

	session   *ssh.Session
	StdinPipe io.WriteCloser
	ws        *websocket.Conn
	conn      *ssh.Client
	sftp      *sftp.Client
	timer     *time.Timer
}

func NewSshServer(s *session.SSession) (*WebsocketServer, error) {
	info := s.ISessionData.(*session.SSshSession)
	server := &WebsocketServer{
		Session:    s,
		Host:       info.Host,
		Port:       info.Port,
		Username:   info.Username,
		Password:   info.Password,
		PrivateKey: info.PrivateKey,
	}
	return server, nil
}

type WebSocketBufferWriter struct {
	s    *session.SSession
	ws   *websocket.Conn
	lock sync.Mutex
}

func (w *WebSocketBufferWriter) Write(p []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	go w.s.GetRecorder().Write("", string(p))
	err := w.ws.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *WebsocketServer) initWs(w http.ResponseWriter, r *http.Request) error {
	username := s.Username
	privateKey := s.PrivateKey
	password := s.Password
	config := &ssh.ClientConfig{
		Timeout:         time.Second,
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	}
	if len(privateKey) > 0 {
		if signer, err := ssh.ParsePrivateKey([]byte(privateKey)); err == nil {
			config.Auth = append(config.Auth, ssh.PublicKeys(signer))
		}
	}

	var err error
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	s.conn, err = ssh.Dial("tcp", addr, config)
	if err != nil {
		return errors.Wrapf(err, "dial %s", addr)
	}

	s.sftp, err = sftp.NewClient(s.conn)
	if err != nil {
		return errors.Wrapf(err, "new sftp client")
	}
	addSftpClient(s.Session.Id, s.sftp)

	s.session, err = s.conn.NewSession()
	if err != nil {
		return errors.Wrapf(err, "NewSession")
	}

	s.StdinPipe, err = s.session.StdinPipe()
	if err != nil {
		return errors.Wrapf(err, "StdinPip")
	}

	var up = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	s.ws, err = up.Upgrade(w, r, nil)
	if err != nil {
		return errors.Wrapf(err, "upgrade")
	}

	wsWriter := WebSocketBufferWriter{
		s:  s.Session,
		ws: s.ws,
	}

	s.session.Stdout = &wsWriter
	s.session.Stderr = &wsWriter

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	err = s.session.RequestPty("xterm-256color", 120, 32, modes)
	if err != nil {
		return errors.Wrapf(err, "request pty xterm")
	}

	err = s.session.Shell()
	if err != nil {
		return errors.Wrapf(err, "Shell")
	}
	return nil
}

func (s *WebsocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := s.initWs(w, r)
	if err != nil {
		log.Errorf("initWs error: %v", err)
		return
	}

	done := make(chan bool, 3)
	setDone := func() { done <- true }

	go func() {
		defer setDone()

		for {
			_, p, err := s.ws.ReadMessage()
			if err != nil {
				return
			}
			if options.Options.SshSessionTimeoutMinutes > 0 && s.timer != nil {
				s.timer.Reset(time.Duration(options.Options.SshSessionTimeoutMinutes) * time.Minute)
			}
			input := struct {
				Type string `json:"type" choices:"resize|input|heartbeat"`
				Data struct {
					Cols   int
					Rows   int
					Data   string `json:"data"`
					Base64 bool
				}
			}{}
			obj, err := jsonutils.Parse(p)
			if err != nil {
				log.Errorf("parse %s error: %v", string(p), err)
				continue
			}
			err = obj.Unmarshal(&input)
			if err != nil {
				log.Errorf("unmarshal %s error: %v", string(p), err)
				continue
			}

			switch input.Type {
			case "close":
				return
			case "resize":
				err = s.session.WindowChange(input.Data.Rows, input.Data.Cols)
				if err != nil {
					log.Errorf("resize %dx%d error: %v", input.Data.Cols, input.Data.Rows, err)
				}
			case "input":
				if input.Data.Base64 {
					data, _ := base64.StdEncoding.DecodeString(input.Data.Data)
					input.Data.Data = string(data)
				}
				go s.Session.GetRecorder().Write(input.Data.Data, "")
				_, err = s.StdinPipe.Write([]byte(input.Data.Data))
				if err != nil {
					log.Errorf("write %s error: %v", input.Data.Data, err)
					return
				}
			case "heartbeat":
				continue
			default:
				log.Errorf("unknow msg type %s for %s:%d", input.Type, s.Host, s.Port)
			}
		}
	}()

	defer func() {
		s.ws.Close()
		s.StdinPipe.Close()
		s.session.Close()
		delSftpClient(s.Session.Id)
		s.sftp.Close()
		s.conn.Close()
	}()

	stop := make(chan bool)
	go func() {
		s.timer = time.NewTimer(time.Microsecond * 100)
		if options.Options.SshSessionTimeoutMinutes > 0 {
			s.timer.Reset(time.Duration(options.Options.SshSessionTimeoutMinutes) * time.Minute)
		}
		defer s.timer.Stop()
		defer setDone()

		for {
			select {
			case <-stop:
				return
			case <-s.timer.C:
				if options.Options.SshSessionTimeoutMinutes > 0 {
					return
				}
				s.timer.Reset(time.Microsecond * 100)
			}
		}
	}()

	go func() {
		defer setDone()

		err = s.session.Wait()
		if err != nil {
			s.StdinPipe.Write([]byte(err.Error()))
		}
	}()

	<-done
	stop <- true
	log.Infof("ssh %s@%s:%d complete", s.Username, s.Host, s.Port)
}
