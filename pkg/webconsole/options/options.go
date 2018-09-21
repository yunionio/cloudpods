package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
)

var (
	Options WebConsoleOptions
)

type WebConsoleOptions struct {
	cloudcommon.Options

	ApiServer       string `help:"API server url to handle websocket connection, usually with public access" default:"http://webconsole.yunion.io"`
	KubectlPath     string `help:"kubectl binary path used to connect k8s cluster" default:"/usr/bin/kubectl"`
	IpmitoolPath    string `help:"ipmitool binary path used to connect baremetal sol" default:"/usr/bin/ipmitool"`
	SSHtoolPath     string `help:"sshtool binary path used to connect server sol" default:"/usr/bin/ssh"`
	SSHPasstoolPath string `help:"sshpass tool binary path used to connect server sol" default:"/usr/local/bin/sshpass"`
}
