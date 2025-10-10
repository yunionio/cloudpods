package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	base_options "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMImage)
	cmd.List(new(options.LLMImageListOptions))
	cmd.Show(new(options.LLMImageShowOptions))
	cmd.Create(new(options.LLMImageCreateOptions))
	cmd.Update(new(options.LLMImageUpdateOptions))
	cmd.Delete(new(options.LLMImageDeleteOptions))
	cmd.Perform("public", &base_options.BasePublicOptions{})
	cmd.Perform("private", &base_options.BaseIdOptions{})
}
