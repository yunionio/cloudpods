package monitor

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	cmd := shell.NewResourceCmd(modules.AlertRecordShieldManager)
	cmd.Create(new(options.AlertRecordShieldCreateOptions))
	cmd.List(new(options.AlertRecordShieldListOptions))
	cmd.Show(new(options.AlertRecordShieldShowOptions))
	cmd.Delete(new(options.AlertRecordShieldDeleteOptions))
}
