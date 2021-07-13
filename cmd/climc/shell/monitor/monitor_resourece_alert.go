package monitor

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewJointCmd(modules.MonitorResourceAlertManager)
	cmd.List(new(options.MonitorResourceAlertListOptions))
}
