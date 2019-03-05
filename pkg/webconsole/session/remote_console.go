package session

import (
	"fmt"
	"net/url"
	"os/exec"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const (
	VNC       = "vnc"
	ALIYUN    = "aliyun"
	QCLOUD    = "qcloud"
	OPENSTACK = "openstack"
	SPICE     = "spice"
	WMKS      = "wmks"
)

type RemoteConsoleInfo struct {
	Host        string `json:"host"`
	Port        int64  `json:"port"`
	Protocol    string `json:"protocol"`
	Id          string `json:"id"`
	OsName      string `json:"osName"`
	VncPassword string `json:"vncPassword"`

	// used by aliyun server
	InstanceId string `json:"instance_id"`
	Url        string `json:"url"`
	Password   string `json:"password"`
}

func NewRemoteConsoleInfoByCloud(s *mcclient.ClientSession, serverId string) (*RemoteConsoleInfo, error) {
	metadata, err := modules.Servers.GetSpecific(s, serverId, "metadata", nil)
	if err != nil {
		return nil, err
	}
	osName, _ := metadata.GetString("os_name")
	vncPasswd, _ := metadata.GetString("__vnc_password")
	ret, err := modules.Servers.GetSpecific(s, serverId, "vnc", nil)
	if err != nil {
		return nil, err
	}
	vncInfo := RemoteConsoleInfo{}
	err = ret.Unmarshal(&vncInfo)
	if err != nil {
		return nil, err
	}
	vncInfo.OsName = osName
	vncInfo.VncPassword = vncPasswd
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

// GetData implements ISessionData interface
func (info *RemoteConsoleInfo) Connect() error {
	return nil
}

// GetData implements ISessionData interface
func (info *RemoteConsoleInfo) GetData(s string) (bool, string, string) {
	return false, "", ""
}

// ShowInfo implements ISessionData interface
func (info *RemoteConsoleInfo) ShowInfo() string {
	return ""
}

func (info *RemoteConsoleInfo) GetConnectParams() (string, error) {
	switch info.Protocol {
	case ALIYUN:
		return info.getAliyunURL()
	case QCLOUD:
		return info.getQcloudURL()
	case OPENSTACK:
		return info.getOpenStackURL()
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

func (info *RemoteConsoleInfo) getOpenStackURL() (string, error) {
	return info.Url, nil
}

func (info *RemoteConsoleInfo) getQcloudURL() (string, error) {
	return "https://img.qcloud.com/qcloud/app/active_vnc/index.html?InstanceVncUrl=" + info.Url, nil
}

func (info *RemoteConsoleInfo) getAliyunURL() (string, error) {
	isWindows := "False"
	if info.OsName == "Windows" {
		isWindows = "True"
	}
	base := "https://g.alicdn.com/aliyun/ecs-console-vnc/0.0.7/index.html"
	params := url.Values{
		"vncUrl":     {info.Url},
		"instanceId": {info.InstanceId},
		"isWindows":  {isWindows},
		"password":   {info.Password},
		"protocol":   {info.Protocol},
	}
	queryURL := params.Encode()
	return fmt.Sprintf("%s?%s", base, queryURL), nil
}
