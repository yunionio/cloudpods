package meter

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/meter"
)

func init() {
	cmd := shell.NewResourceCmd(modules.BillingDimensionManager)
	cmd.Create(&options.BillDimensionCreateOptions{})
	cmd.List(&options.BillDimensionListOptions{})
	cmd.Show(&options.BillDimensionShowOptions{})
	cmd.Delete(&options.BillDimensionDeleteOptions{})

	cmdAnalysis := shell.NewResourceCmd(modules.BillingDimensionAnalysisManager)
	cmdAnalysis.List(&options.BillDimensionAnalysisListOptions{})
	cmdAnalysis.Show(&options.BillDimensionAnalysisShowOptions{})

}
