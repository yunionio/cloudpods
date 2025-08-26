package compute

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMs)
	cmd.Create(new(options.LLMCreateOptions))
	cmd.List(new(options.LLMListOptions))
	cmd.Show(new(options.LLMShowOptions))
	cmd.BatchPerform("stop", new(options.LLMStopOptions))
	cmd.BatchPerform("start", new(options.LLMStartOptions))
}
