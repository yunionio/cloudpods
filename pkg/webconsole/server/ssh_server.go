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
	"net"
	"net/http"
	"time"

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

	session    *ssh.Session
	StdinPipe  io.WriteCloser
	StdoutPipe io.Reader
	StderrPipe io.Reader

	ws         *websocket.Conn
	conn       *ssh.Client
	sshNetConn net.Conn
	sftp       *sftp.Client
	timer      *time.Timer
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

func writeToWebsocket(reader io.Reader, s *WebsocketServer) error {
	var data = make([]byte, 1024)
	for {
		n, err := reader.Read(data)
		if err != nil {
			return errors.Wrap(err, "read data from reader")
		}
		out := data[:n]
		go s.Session.GetRecorder().Write("", string(out))
		if err := s.ws.WriteMessage(websocket.BinaryMessage, out); err != nil {
			return errors.Wrapf(err, "write data to websocket, out: %s", string(out))
		}
	}
	return nil
}

func (s *WebsocketServer) initWs(w http.ResponseWriter, r *http.Request) error {
	username := s.Username
	privateKey := s.PrivateKey
	password := s.Password
	config := &ssh.ClientConfig{
		Timeout:         5 * time.Second,
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
	s.conn, s.sshNetConn, err = NewSshClient("tcp", addr, config)
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
		return errors.Wrapf(err, "StdinPipe")
	}
	s.StdoutPipe, err = s.session.StdoutPipe()
	if err != nil {
		return errors.Wrapf(err, "StdoutPipe")
	}
	s.StderrPipe, err = s.session.StderrPipe()
	if err != nil {
		return errors.Wrapf(err, "StderrPipe")
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

// ref: https://github.com/golang/go/issues/19338#issuecomment-539057790
func NewSshClient(network, addr string, conf *ssh.ClientConfig) (*ssh.Client, net.Conn, error) {
	conn, err := net.DialTimeout(network, addr, conf.Timeout)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "dial %s %s", network, addr)
	}
	if conf.Timeout > 0 {
		conn.SetDeadline(time.Now().Add(conf.Timeout))
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, conf)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "new client conn %s", addr)
	}
	if conf.Timeout > 0 {
		conn.SetDeadline(time.Time{})
	}
	return ssh.NewClient(c, chans, reqs), conn, nil
}

func (s *WebsocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logPrefix := fmt.Sprintf("ssh %s@%s:%d, session_id: %s", s.Username, s.Host, s.Port, s.Session.Id)

	err := s.initWs(w, r)
	if err != nil {
		log.Errorf("%s, initWs error: %v", logPrefix, err)
		return
	}

	done := make(chan bool, 3)
	keepAliveDone := make(chan struct{})

	go func() {
		if err := sshKeepAlive(s.conn, s.sshNetConn, keepAliveDone); err != nil {
			log.Errorf("%s, keepalive error: %v", logPrefix, err)
		}
	}()

	setDone := func() {
		done <- true
	}

	for _, reader := range []io.Reader{s.StdoutPipe, s.StderrPipe} {
		tmpReader := reader
		go func() {
			if err := writeToWebsocket(tmpReader, s); err != nil {
				log.Warningf("%s, writeToWebsocket error: %v", logPrefix, err)
			}
		}()
	}

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
				log.Errorf("%s, parse %s error: %v", logPrefix, string(p), err)
				continue
			}
			err = obj.Unmarshal(&input)
			if err != nil {
				log.Errorf("%s, unmarshal %s error: %v", logPrefix, string(p), err)
				continue
			}

			switch input.Type {
			case "close":
				return
			case "resize":
				err = s.session.WindowChange(input.Data.Rows, input.Data.Cols)
				if err != nil {
					log.Errorf("%s, resize %dx%d error: %v", logPrefix, input.Data.Cols, input.Data.Rows, err)
				}
			case "input":
				if input.Data.Base64 {
					data, _ := base64.StdEncoding.DecodeString(input.Data.Data)
					input.Data.Data = string(data)
				}
				go s.Session.GetRecorder().Write(input.Data.Data, "")
				_, err = s.StdinPipe.Write([]byte(input.Data.Data))
				if err != nil {
					log.Errorf("%s, write %s error: %v", logPrefix, input.Data.Data, err)
					return
				}
			case "heartbeat":
				continue
			default:
				log.Errorf("%s, unknow msg type %s", logPrefix, input.Type)
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
		if !options.Options.KeepWebsocketSession {
			s.Session.Close()
		}
		keepAliveDone <- struct{}{}
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
			log.Warningf("%s wait error: %v", logPrefix, err)
			s.StdinPipe.Write([]byte(err.Error()))
		}
	}()

	<-done
	stop <- true
}

func sshKeepAlive(cli *ssh.Client, conn net.Conn, done <-chan struct{}) error {
	// ref:
	// - https://github.com/golang/go/issues/21478
	// - https://github.com/scylladb/go-sshtools/blob/master/keepalive.go#L36
	const keepAliveInterval = 15 * time.Second
	t := time.NewTicker(keepAliveInterval)
	defer t.Stop()
	for {
		deadline := time.Now().Add(keepAliveInterval).Add(15 * time.Second)
		if err := conn.SetDeadline(deadline); err != nil {
			return errors.Wrap(err, "failed to set deadline")
		}
		select {
		case <-t.C:
			_, _, err := cli.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				return errors.Wrap(err, "failed to send keep alive")
			}
		case <-done:
			return nil
		}
	}
}
