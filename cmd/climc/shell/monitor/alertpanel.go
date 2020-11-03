package monitor

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewResourceCmd(modules.AlertPanelManager)
	cmd.Create(new(options.AlertPanelCreateOptions))
	cmd.List(new(options.AlertPanelListOptions))
	cmd.Show(new(options.AlertPanelShowOptions))
	cmd.Delete(new(options.AlertPanelDeleteOptions))
}
