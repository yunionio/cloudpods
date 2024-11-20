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
	"net/url"
	"os/exec"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/webconsole/options"
	"yunion.io/x/onecloud/pkg/webconsole/recorder"
)

const (
	VNC        = api.VNC
	RDP        = api.RDP
	ALIYUN     = api.ALIYUN
	QCLOUD     = api.QCLOUD
	OPENSTACK  = api.OPENSTACK
	SPICE      = api.SPICE
	WMKS       = api.WMKS
	WS         = api.WS
	VMRC       = api.VMRC
	ZSTACK     = api.ZSTACK
	CTYUN      = api.CTYUN
	HUAWEI     = api.HUAWEI
	HCS        = api.HCS
	APSARA     = api.APSARA
	JDCLOUD    = api.JDCLOUD
	CLOUDPODS  = api.CLOUDPODS
	PROXMOX    = api.PROXMOX
	VOLCENGINE = api.VOLC_ENGINE
	BAIDU      = api.BAIDU
	SANGFOR    = api.SANGFOR
)

type RemoteConsoleInfo struct {
	cloudprovider.ServerVncOutput

	s *mcclient.ClientSession
}

func NewRemoteConsoleInfoByCloud(s *mcclient.ClientSession, serverId string, query jsonutils.JSONObject) (*RemoteConsoleInfo, error) {
	ret, err := modules.Servers.GetSpecific(s, serverId, "vnc", query)
	if err != nil {
		return nil, err
	}
	vncInfo := RemoteConsoleInfo{}
	err = ret.Unmarshal(&vncInfo)
	if err != nil {
		return nil, err
	}
	vncInfo.s = s

	if len(vncInfo.OsName) == 0 || len(vncInfo.VncPassword) == 0 {
		metadata, err := modules.Servers.GetSpecific(s, serverId, "metadata", nil)
		if err != nil {
			return nil, err
		}
		osName, _ := metadata.GetString("os_name")
		vncPasswd, _ := metadata.GetString("__vnc_password")
		vncInfo.OsName = osName
		vncInfo.VncPassword = vncPasswd
	}

	return &vncInfo, nil
}

// GetProtocol implements ISessionData interface
func (info *RemoteConsoleInfo) GetProtocol() string {
	return info.Protocol
}

// GetCommand implements ISessionData interface
func (info *RemoteConsoleInfo) GetCommand() *exec.Cmd {
	return nil
}

// Cleanup implements ISessionData interface
func (info *RemoteConsoleInfo) Cleanup() error {
	return nil
}

// Connect implements ISessionData interface
func (info *RemoteConsoleInfo) Connect() error {
	return nil
}

// Scan implements ISessionData interface
func (info *RemoteConsoleInfo) Scan(byte, func(string)) {
	return
}

// IsNeedLogin implements ISessionData interface
func (info *RemoteConsoleInfo) IsNeedLogin() (bool, error) {
	return false, nil
}

func (info *RemoteConsoleInfo) GetClientSession() *mcclient.ClientSession {
	return info.s
}

func (info *RemoteConsoleInfo) GetConnectParams() (string, error) {
	switch info.Protocol {
	case ALIYUN:
		return info.getAliyunURL()
	case APSARA:
		return info.getApsaraURL()
	case QCLOUD:
		return info.getQcloudURL()
	case CLOUDPODS:
		return info.getCloudpodsURL()
	case OPENSTACK, VMRC, ZSTACK, CTYUN, HUAWEI, HCS, JDCLOUD, PROXMOX, SANGFOR, BAIDU:
		return info.Url, nil
	case VOLCENGINE:
		return info.getVolcEngineURL()
	default:
		return "", fmt.Errorf("Can't convert protocol %s to connect params", info.Protocol)
	}
}

func (info *RemoteConsoleInfo) GetPassword() string {
	if len(info.Password) != 0 {
		return info.Password
	}
	return info.VncPassword
}

func (info *RemoteConsoleInfo) GetId() string {
	return info.Id
}

func (info *RemoteConsoleInfo) getConnParamsURL(baseURL string, params url.Values) string {
	if params == nil {
		params = url.Values{}
	}
	params.Set("protocol", info.Protocol)
	queryURL := params.Encode()
	return fmt.Sprintf("%s?%s", baseURL, queryURL)
}

func (info *RemoteConsoleInfo) getQcloudURL() (string, error) {
	base := "https://img.qcloud.com/qcloud/app/active_vnc/index.html?InstanceVncUrl=" + info.Url
	return info.getConnParamsURL(base, nil), nil
}

func (info *RemoteConsoleInfo) getAliyunURL() (string, error) {
	isWindows := "false"
	if info.OsName == "Windows" {
		isWindows = "true"
	}
	params := url.Values{
		"vncUrl":     {info.Url},
		"instanceId": {info.InstanceId},
		"isWindows":  {isWindows},
		"password":   {info.Password},
	}
	return info.getConnParamsURL(options.Options.AliyunConsoleAddr, params), nil
}

func (info *RemoteConsoleInfo) getCloudpodsURL() (string, error) {
	base := fmt.Sprintf("%s/web-console/no-vnc", info.ApiServer)
	params := url.Values{
		"data":         {info.ConnectParams},
		"instanceId":   {info.InstanceId},
		"instanceName": {info.InstanceName},
	}
	return info.getConnParamsURL(base, params), nil
}

func (info *RemoteConsoleInfo) getVolcEngineURL() (string, error) {
	base := "https://console.volcengine.com/ecs/connect/vnc/"
	params := url.Values{
		"host":   {info.Url},
		"Region": {info.Region},
		"name":   {info.InstanceName},
	}
	return info.getConnParamsURL(base, params), nil
}

func (info *RemoteConsoleInfo) getApsaraURL() (string, error) {
	isWindows := "False"
	if info.OsName == "Windows" {
		isWindows = "True"
	}
	params := url.Values{
		"vncUrl":     {info.Url},
		"instanceId": {info.InstanceId},
		"isWindows":  {isWindows},
		"password":   {info.Password},
	}
	return info.getConnParamsURL(options.Options.ApsaraConsoleAddr, params), nil
}

func (info *RemoteConsoleInfo) GetRecordObject() *recorder.Object {
	return nil
}

func (info *RemoteConsoleInfo) GetDisplayInfo(ctx context.Context) (*SDisplayInfo, error) {
	userInfo, err := fetchUserInfo(ctx, info.GetClientSession())
	if err != nil {
		return nil, errors.Wrap(err, "fetchUserInfo")
	}
	guestDetails, err := FetchServerInfo(ctx, info.GetClientSession(), info.Id)
	if err != nil {
		return nil, errors.Wrap(err, "FetchServerInfo")
	}

	dispInfo := SDisplayInfo{}
	dispInfo.WaterMark = fetchWaterMark(userInfo)
	dispInfo.fetchGuestInfo(guestDetails)
	return &dispInfo, nil
}
