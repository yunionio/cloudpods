package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMs)
	cmd.Create(new(options.LLMCreateOptions))
	cmd.List(new(options.LLMListOptions))
	cmd.Show(new(options.LLMShowOptions))
	cmd.Perform("change-model", new(options.LLMChangeModelOptions))
	// cmd.BatchPerform("stop", new(options.LLMStopOptions))
	// cmd.BatchPerform("start", new(options.LLMStartOptions))
}
