package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	base_options "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMModel)
	cmd.List(new(options.LLMModelListOptions))
	cmd.Show(new(options.LLMModelShowOptions))
	cmd.Update(new(options.LLMModelUpdateOptions))
	cmd.Create(new(options.LLMModelCreateOptions))
	cmd.Delete(new(options.LLMModelDeleteOptions))
	cmd.Perform("public", &base_options.BasePublicOptions{})
	cmd.Perform("private", &base_options.BaseIdOptions{})
	// cmd.Perform("clone", new(options.DesktopModelCloneOptions))
}
