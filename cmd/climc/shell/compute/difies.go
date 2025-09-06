package compute

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Difies)
	cmd.Create(new(options.DifyCreateOptions))
	cmd.List(new(options.DifyListOptions))
	cmd.Show(new(options.DifyShowOptions))
}
