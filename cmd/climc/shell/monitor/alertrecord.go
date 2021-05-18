package monitor

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewResourceCmd(modules.AlertRecordManager)
	cmd.List(new(options.AlertRecordListOptions))
	cmd.Show(new(options.AlertRecordShowOptions))
	cmd.Get("", new(options.AlertRecordTotalOptions))

	cmd1 := shell.NewResourceCmd(modules.AlertRecordV2Manager)
	cmd1.List(new(options.AlertRecordListOptions))
	cmd1.Show(new(options.AlertRecordShowOptions))
}
