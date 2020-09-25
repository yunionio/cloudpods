package monitor

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewResourceCmd(modules.CommonAlertManager)
	cmd.List(new(options.CommonAlertListOptions))
	cmd.Show(new(options.CommonAlertShowOptions))
	cmd.Perform("enable", &options.CommonAlertShowOptions{})
	cmd.Perform("disable", &options.CommonAlertShowOptions{})
	cmd.Delete(new(options.CommonAlertDeleteOptions))
}
