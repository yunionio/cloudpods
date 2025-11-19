package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMs)
	cmd.BatchCreate(new(options.LLMCreateOptions))
	cmd.List(new(options.LLMListOptions))
	cmd.Show(new(options.LLMShowOptions))
	cmd.Delete(new(options.LLMDeleteOptions))
	// cmd.Perform("change-model", new(options.LLMChangeModelOptions))
	cmd.BatchPerform("stop", new(options.LLMStopOptions))
	cmd.BatchPerform("start", new(options.LLMStartOptions))
	cmd.Get("probed-packages", new(options.LLMIdOptions))
	cmd.Perform("save-instant-app", new(options.LLMSaveInstantAppOptions))
}
