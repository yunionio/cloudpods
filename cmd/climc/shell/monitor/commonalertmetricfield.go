package monitor

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewResourceCmd(modules.MetricFieldManager)
	cmd.List(new(options.MonitorMetricFieldListOptions))
	cmd.Update(new(options.MetricFieldUpdateOptions))
	cmd.Show(new(options.MetricFieldShowOptions))
	cmd.Delete(new(options.MetricFieldDeleteOptions))
}
