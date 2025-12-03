package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	base_options "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.DifySku)
	cmd.List(new(options.DifySkuListOptions))
	cmd.Show(new(options.DifySkuShowOptions))
	cmd.Update(new(options.DifySkuUpdateOptions))
	cmd.Create(new(options.DifySkuCreateOptions))
	cmd.Delete(new(options.DifySkuDeleteOptions))
	cmd.Perform("public", &base_options.BasePublicOptions{})
	cmd.Perform("private", &base_options.BaseIdOptions{})
	// cmd.Perform("clone", new(options.DesktopSkuCloneOptions))
}
