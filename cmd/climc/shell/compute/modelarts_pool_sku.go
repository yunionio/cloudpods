package compute

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.ModelartsPoolSku).WithKeyword("modelarts-pool-sku")
	cmd.List(&compute.ModelartsPoolSkuListOptions{})
}
