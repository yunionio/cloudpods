package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	base_options "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMSku)
	cmd.List(new(options.LLMSkuListOptions))
	cmd.Show(new(options.LLMSkuShowOptions))
	cmd.Update(new(options.LLMSkuUpdateOptions))
	cmd.Create(new(options.LLMSkuCreateOptions))
	cmd.Delete(new(options.LLMSkuDeleteOptions))
	cmd.Perform("public", &base_options.BasePublicOptions{})
	cmd.Perform("private", &base_options.BaseIdOptions{})
	// cmd.Perform("clone", new(options.DesktopSkuCloneOptions))
}
