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

	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/webconsole/helper"
	o "yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/recorder"
)

type SSshSession struct {
	us *mcclient.ClientSession
	id string

	// vm name
	name string
	Host string

	Port       int
	PrivateKey string
	Username   string
	// 保持原有 Username ，不实用 cloudroot 的同时使用 PrivateKey
	KeepUsername bool
	Password     string

	guestDetails *compute_api.ServerDetails
	hostDetails  *compute_api.HostDetails
}

func NewSshSession(ctx context.Context, us *mcclient.ClientSession, conn SSshConnectionInfo) *SSshSession {
	ret := &SSshSession{
		us:           us,
		id:           stringutils.UUID4(),
		Port:         conn.Port,
		Host:         conn.IP,
		name:         conn.Name,
		Username:     conn.Username,
		KeepUsername: conn.KeepUsername,
		Password:     conn.Password,

		guestDetails: conn.GuestDetails,
		hostDetails:  conn.HostDetails,
	}
	if conn.Port <= 0 {
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
	if len(s.name) == 0 {
		s.name = s.Username
	}
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
	if !s.KeepUsername {
		s.Username = "cloudroot"
	} else {
		if s.Username == "" {
			return true, errors.Error("username is empty")
		}
	}
	privateKey, err := helper.GetValidPrivateKey(s.Host, s.Port, s.Username, s.us.GetProjectId())
	if err != nil {
		return true, errors.Wrap(err, "try to use cloud admin private_key for ssh login")
	}
	s.PrivateKey = privateKey
	return false, nil
}

func (s *SSshSession) Scan(d byte, send func(msg string)) {
}

func (s *SSshSession) GetDisplayInfo(ctx context.Context) (*SDisplayInfo, error) {
	userInfo, err := fetchUserInfo(ctx, s.GetClientSession())
	if err != nil {
		return nil, errors.Wrap(err, "fetchUserInfo")
	}
	dispInfo := SDisplayInfo{}
	dispInfo.WaterMark = fetchWaterMark(userInfo)
	if s.guestDetails != nil {
		dispInfo.fetchGuestInfo(s.guestDetails)
	} else if s.hostDetails != nil {
		dispInfo.fetchHostInfo(s.hostDetails)
	} else {
		dispInfo.Ips = s.Host
		if len(s.name) > 0 {
			dispInfo.InstanceName = s.name
		}
	}

	return &dispInfo, nil
}
