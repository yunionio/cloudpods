package monitor

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewResourceCmd(monitor.UnifiedMonitorManager).WithKeyword("simple-query")
	cmd.Show(&options.SimpleInputQuery{})
}
