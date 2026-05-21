package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	{
		cmd := shell.NewResourceCmd(&modules.LLMModelSets)
		cmd.List(new(options.LLMModelSetListOptions))
		cmd.Show(new(options.LLMModelSetShowOptions))
		cmd.Get("specs", new(options.LLMModelSetSpecsOptions))
		cmd.PerformClass("refresh", new(options.LLMModelSetRefreshOptions))
	}
	{
		cmd := shell.NewResourceCmd(&modules.LLMModelSpecs)
		cmd.Show(new(options.LLMModelSpecShowOptions))
	}
}
