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

package session

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/recorder"
)

type SSshSession struct {
	us *mcclient.ClientSession
	id string

	// vm name
	name string
	Host string

	Port       int64
	PrivateKey string
	Username   string
	Password   string
}

func NewSshSession(ctx context.Context, us *mcclient.ClientSession, name, ip string, port int64, username, password string) *SSshSession {
	ret := &SSshSession{
		us:       us,
		id:       stringutils.UUID4(),
		Port:     port,
		Host:     ip,
		name:     name,
		Username: username,
		Password: password,
	}
	if len(ret.name) == 0 {
		ret.name = ret.Username
	}
	if port <= 0 {
		ret.Port = 22
	}
	return ret
}

func (s *SSshSession) GetId() string {
	return s.id
}

func (s *SSshSession) Cleanup() error {
	return cloudprovider.ErrNotImplemented
}

func (s *SSshSession) GetClientSession() *mcclient.ClientSession {
	return s.us
}

func (s *SSshSession) GetProtocol() string {
	return api.WS
}

func (s *SSshSession) GetRecordObject() *recorder.Object {
	return recorder.NewObject(s.id, s.name, "server", s.Username, jsonutils.Marshal(map[string]interface{}{"ip": s.Host, "port": s.Port}))
}

func (s *SSshSession) GetCommand() *exec.Cmd {
	return nil
}

func (s *SSshSession) IsNeedLogin() (bool, error) {
	if len(s.Username) > 0 && len(s.Password) > 0 {
		config := &ssh.ClientConfig{
			Timeout:         time.Second,
			User:            s.Username,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Auth: []ssh.AuthMethod{
				ssh.Password(s.Password),
			},
		}
		addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			return true, err
		}
		defer client.Close()
		return false, nil
	}
	if !o.Options.EnableAutoLogin {
		return true, nil
	}
	privateKey, err := func() (string, error) {
		ctx := context.Background()
		admin := auth.GetAdminSession(ctx, o.Options.Region)
		key, err := compute.Sshkeypairs.GetById(admin, s.us.GetProjectId(), jsonutils.Marshal(map[string]bool{"admin": true}))
		if err != nil {
			return "", errors.Wrapf(err, "Sshkeypairs.GetById(%s)", s.us.GetProjectId())
		}
		privKey, err := key.GetString("private_key")
		if err != nil {
			return "", errors.Wrapf(err, "get private_key")
		}
		signer, err := ssh.ParsePrivateKey([]byte(privKey))
		if err != nil {
			return "", errors.Wrapf(err, "ParsePrivateKey")
		}
		config := &ssh.ClientConfig{
			Timeout:         time.Second,
			User:            "cloudroot",
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
		}
		addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			return "", errors.Wrapf(err, "dial %s", addr)
		}
		defer client.Close()
		return privKey, nil
	}()
	if err != nil {
		return true, err
	}
	s.Username = "cloudroot"
	s.PrivateKey = privateKey
	return false, nil
}

func (s *SSshSession) Scan(d byte, send func(msg string)) {
}
