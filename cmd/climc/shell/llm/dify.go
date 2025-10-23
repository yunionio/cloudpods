package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Difies)
	cmd.BatchCreate(new(options.DifyCreateOptions))
	cmd.List(new(options.DifyListOptions))
	cmd.Show(new(options.DifyShowOptions))
	cmd.Delete(new(options.DifyDeleteOptions))
	cmd.BatchPerform("stop", new(options.DifyStopOptions))
	cmd.BatchPerform("start", new(options.DifyStartOptions))
}
