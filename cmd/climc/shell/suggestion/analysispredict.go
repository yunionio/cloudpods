package suggestion

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	options "yunion.io/x/onecloud/pkg/mcclient/options/suggestion"
)

func init() {
	cmd := shell.NewResourceCmd(modules.AnalysisPredictManager)
	cmd.Get("", new(options.AnalysisPredictConfigOptions))
}
