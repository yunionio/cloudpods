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
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"

	"yunion.io/x/log"
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
			return nil, err
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
	return s.run(false, cmds, nil)
}

func (s *Client) Run(cmds ...string) ([]string, error) {
	return s.run(true, cmds, nil)
}

func (s *Client) RunWithInput(input io.Reader, cmds ...string) ([]string, error) {
	return s.run(true, cmds, input)
}

func (s *Client) run(parseOutput bool, cmds []string, input io.Reader) ([]string, error) {
	ret := []string{}
	for _, cmd := range cmds {
		session, err := s.client.NewSession()
		if err != nil {
			return nil, err
		}
		defer session.Close()
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
			outputErr = errors.New(errMsg)
			err = errors.Errorf("%q error: %v, cmd error: %v", cmd, err, outputErr)
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
