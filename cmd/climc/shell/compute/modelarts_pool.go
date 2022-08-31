package compute

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.ModelartsPools).WithKeyword("modelarts-pool")
	cmd.List(&compute.ModelartsPoolListOptions{})
	cmd.Delete(&options.BaseIdOptions{})
	cmd.Create(&compute.ModelartsPoolCreateOption{})
}
