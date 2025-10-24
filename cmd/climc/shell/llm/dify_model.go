package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	base_options "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.DifyModel)
	cmd.List(new(options.DifyModelListOptions))
	cmd.Show(new(options.DifyModelShowOptions))
	cmd.Update(new(options.DifyModelUpdateOptions))
	cmd.Create(new(options.DifyModelCreateOptions))
	cmd.Delete(new(options.DifyModelDeleteOptions))
	cmd.Perform("public", &base_options.BasePublicOptions{})
	cmd.Perform("private", &base_options.BaseIdOptions{})
	// cmd.Perform("clone", new(options.DesktopModelCloneOptions))
}
