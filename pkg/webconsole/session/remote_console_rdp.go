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
	"os/exec"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"

	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/webconsole/recorder"
)

type RemoteRDPConsoleInfo struct {
	Host         string
	Port         int
	Username     string
	Password     string
	ConnectionId string

	guestDetails *compute_api.ServerDetails

	Width  int
	Height int
	Dpi    int
	s      *mcclient.ClientSession
}

func NewRemoteRDPConsoleInfoByCloud(ctx context.Context, s *mcclient.ClientSession, serverId string, query jsonutils.JSONObject) (*RemoteRDPConsoleInfo, error) {
	info := &RemoteRDPConsoleInfo{s: s}
	if !gotypes.IsNil(query) {
		err := query.Unmarshal(&info)
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal")
		}
	}
	if info.Port <= 0 {
		info.Port = 3389
	}
	var err error
	info.Host, info.Port, info.guestDetails, err = resolveServerIPPortById(ctx, s, serverId, info.Host, info.Port)
	if err != nil {
		return nil, errors.Wrap(err, "resolveServerIPPortById")
	}
	if len(info.Host) == 0 {
		return nil, httperrors.NewMissingParameterError("host")
	}
	if (len(info.Password) == 0 || len(info.Username) == 0) && len(info.ConnectionId) == 0 {
		ret, err := modules.Servers.PerformAction(s, serverId, "login-info", jsonutils.NewDict())
		if err != nil {
			return nil, err
		}
		if !ret.Contains("username") || !ret.Contains("password") {
			return nil, httperrors.NewMissingParameterError("username")
		}
		info.Password, _ = ret.GetString("password")
		info.Username, _ = ret.GetString("username")
	}
	return info, nil
}

func (info *RemoteRDPConsoleInfo) GetProtocol() string {
	return api.RDP
}

func (info *RemoteRDPConsoleInfo) GetCommand() *exec.Cmd {
	return nil
}

func (info *RemoteRDPConsoleInfo) Cleanup() error {
	return nil
}

func (info *RemoteRDPConsoleInfo) Connect() error {
	return nil
}

func (info *RemoteRDPConsoleInfo) Scan(byte, func(string)) {
	return
}

func (info *RemoteRDPConsoleInfo) IsNeedLogin() (bool, error) {
	return false, nil
}

func (info *RemoteRDPConsoleInfo) GetClientSession() *mcclient.ClientSession {
	return info.s
}

func (info *RemoteRDPConsoleInfo) GetConnectParams() (string, error) {
	return "", nil
}

func (info *RemoteRDPConsoleInfo) GetPassword() string {
	return info.Password
}

func (info *RemoteRDPConsoleInfo) GetId() string {
	return ""
}

func (info *RemoteRDPConsoleInfo) GetRecordObject() *recorder.Object {
	return nil
}

func (info *RemoteRDPConsoleInfo) GetDisplayInfo(ctx context.Context) (*SDisplayInfo, error) {
	userInfo, err := fetchUserInfo(ctx, info.GetClientSession())
	if err != nil {
		return nil, errors.Wrap(err, "fetchUserInfo")
	}
	dispInfo := SDisplayInfo{}
	dispInfo.WaterMark = fetchWaterMark(userInfo)
	dispInfo.fetchGuestInfo(info.guestDetails)
	return &dispInfo, nil
}
