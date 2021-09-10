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

package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	ErrBadConfig = errors.Error("bad config")
	ErrNetwork   = errors.Error("network error")
	ErrProtocol  = errors.Error("ssh protocol error")
)

type ClientConfig struct {
	Username   string
	Password   string
	Host       string
	Port       int
	PrivateKey string
}

func parsePrivateKey(keyBuff string) (ssh.Signer, error) {
	return ssh.ParsePrivateKey([]byte(keyBuff))
}

func (conf ClientConfig) ToSshConfig() (*ssh.ClientConfig, error) {
	cliConfig := &ssh.ClientConfig{
		User:            conf.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	auths := make([]ssh.AuthMethod, 0)
	if conf.Password != "" {
		auths = append(auths, ssh.Password(conf.Password))
	}
	if conf.PrivateKey != "" {
		signer, err := parsePrivateKey(conf.PrivateKey)
		if err != nil {
			return nil, errors.Wrapf(ErrBadConfig, "parse private key: %v", err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}
	cliConfig.Auth = auths
	return cliConfig, nil
}

func (conf ClientConfig) Connect() (*ssh.Client, error) {
	cliConfig, err := conf.ToSshConfig()
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	client, err := ssh.Dial("tcp", addr, cliConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (conf ClientConfig) ConnectContext(ctx context.Context) (*ssh.Client, error) {
	cliConfig, err := conf.ToSshConfig()
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	d := &net.Dialer{}
	netconn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, errors.Wrapf(ErrNetwork, "tcp dial: %v", err)
	}

	sshconn, chans, reqs, err := ssh.NewClientConn(netconn, addr, cliConfig)
	if err != nil {
		netconn.Close()
		return nil, errors.Wrap(ErrProtocol, err.Error())
	}

	sshc := ssh.NewClient(sshconn, chans, reqs)
	return sshc, nil
}

type Client struct {
	config ClientConfig
	client *ssh.Client
}

func (conf ClientConfig) NewClient() (*Client, error) {
	cli, err := conf.Connect()
	if err != nil {
		return nil, err
	}
	return &Client{
		config: conf,
		client: cli,
	}, nil
}

func NewClient(
	host string,
	port int,
	username string,
	password string,
	privateKey string,
) (*Client, error) {
	config := &ClientConfig{
		Host:       host,
		Port:       port,
		Username:   username,
		Password:   password,
		PrivateKey: privateKey,
	}
	return config.NewClient()
}

func (s *Client) GetConfig() ClientConfig {
	return s.config
}

func (s *Client) RawRun(cmds ...string) ([]string, error) {
	return s.run(false, cmds, nil, false)
}

func (s *Client) Run(cmds ...string) ([]string, error) {
	return s.run(true, cmds, nil, false)
}

func (s *Client) RunWithInput(input io.Reader, cmds ...string) ([]string, error) {
	return s.run(true, cmds, input, false)
}

// RunWithTTY request Pty before run command.
func (s *Client) RunWithTTY(cmds ...string) ([]string, error) {
	return s.run(false, cmds, nil, true)
}

func (s *Client) run(parseOutput bool, cmds []string, input io.Reader, withPty bool) ([]string, error) {
	ret := []string{}
	for _, cmd := range cmds {
		session, err := s.client.NewSession()
		if err != nil {
			return nil, err
		}
		defer session.Close()

		if withPty {
			modes := ssh.TerminalModes{
				ssh.ECHO:          1,     // enable echoing
				ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
				ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
			}
			if err := session.RequestPty("xterm", 24, 80, modes); err != nil {
				return nil, errors.Wrap(err, "Setup TTY")
			}
		}

		log.Debugf("Run command: %s", cmd)
		var stdOut bytes.Buffer
		var stdErr bytes.Buffer
		session.Stdout = &stdOut
		session.Stderr = &stdErr
		session.Stdin = input
		err = session.Run(cmd)
		if err != nil {
			var outputErr error
			errMsg := stdErr.String()
			if len(stdOut.String()) != 0 {
				errMsg = fmt.Sprintf("%s %s", errMsg, stdOut.String())
			}
			outputErr = errors.Error(errMsg)
			err = errors.Wrapf(outputErr, "%q error: %v, cmd error", cmd, err)
			return nil, err
		}
		if parseOutput {
			ret = append(ret, ParseOutput(stdOut.Bytes())...)
		} else {
			ret = append(ret, stdOut.String())
		}
	}

	return ret, nil
}

func ParseOutput(output []byte) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(string(output), "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func (s *Client) Close() {
	s.client.Close()
}

func updateTermSize(session *ssh.Session, quit <-chan int) {
	sigwinchCh := make(chan os.Signal, 1)
	signal.Notify(sigwinchCh, syscall.SIGWINCH)

	fd := int(os.Stdin.Fd())
	width, height, err := terminal.GetSize(fd)
	if err != nil {
		log.Errorf("get terminal size: %v", err)
	}

	for {
		select {
		case <-quit:
			return
		case sigwinCh := <-sigwinchCh:
			if sigwinCh == nil {
				<-quit
				return
			}
			termWidth, termHeight, err := terminal.GetSize(fd)
			if err != nil {
				log.Errorf("get terminal size: %v", err)
			}

			if termHeight == height && termWidth == width {
				continue
			}

			err = session.WindowChange(termHeight, termWidth)
			if err != nil {
				log.Errorf("send window-change request: %v", err)
				continue
			}

			width = termWidth
			height = termHeight
		}
	}

}

func (s *Client) RunTerminal() error {
	defer s.Close()
	session, err := s.client.NewSession()
	if err != nil {
		return errors.Wrap(err, "open new session")
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return errors.Wrap(err, "make raw terminal")
	}
	defer terminal.Restore(fd, state)

	w, h, err := terminal.GetSize(fd)
	if err != nil {
		return errors.Wrap(err, "get terminal size")
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm-256color"
	}
	if err := session.RequestPty(term, h, w, modes); err != nil {
		return errors.Wrap(err, "session xterm")
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	if err := session.Shell(); err != nil {
		return errors.Wrap(err, "session shell")
	}

	quit := make(chan int)
	go updateTermSize(session, quit)

	if err := session.Wait(); err != nil {
		if e, ok := err.(*ssh.ExitError); ok {
			switch e.ExitStatus() {
			case 130:
				quit <- 1
				return nil
			}
		}
		quit <- 1
		return errors.Wrap(err, "ssh wait")
	}
	quit <- 1
	return nil
}

func IsExitMissingError(err error) bool {
	errStr := new(ssh.ExitMissingError).Error()
	if strings.Contains(err.Error(), errStr) {
		return true
	}
	return false
}
