package monitor

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewResourceCmd(modules.MetricManager)
	cmd.List(new(options.MonitorMetricListOptions))
	cmd.Update(new(options.MetricUpdateOptions))
	cmd.Show(new(options.MetricShowOptions))
}
