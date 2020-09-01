package monitor

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewResourceCmd(modules.AlertDashBoardManager)
	cmd.Create(new(options.AlertDashBoardCreateOptions))
	cmd.List(new(options.AlertDashBoardListOptions))
	cmd.Show(new(options.AlertDashBoardShowOptions))
	cmd.Delete(new(options.AlertDashBoardDeleteOptions))
}
