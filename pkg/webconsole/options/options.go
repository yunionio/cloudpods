package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
)

var (
	Options WebConsoleOptions
)

type WebConsoleOptions struct {
	cloudcommon.CommonOptions

	ApiServer       string `help:"API server url to handle websocket connection, usually with public access" default:"http://webconsole.yunion.io"`
	KubectlPath     string `help:"kubectl binary path used to connect k8s cluster" default:"/usr/bin/kubectl"`
	IpmitoolPath    string `help:"ipmitool binary path used to connect baremetal sol" default:"/usr/bin/ipmitool"`
	SshToolPath     string `help:"sshtool binary path used to connect server sol" default:"/usr/bin/ssh"`
	SshpassToolPath string `help:"sshpass tool binary path used to connect server sol" default:"/usr/bin/sshpass"`
	EnableAutoLogin bool   `help:"allow webconsole to log in directly with the cloudroot public key" default:"false"`
}
