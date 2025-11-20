package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"

	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	commonoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMInstantApp)
	cmd.List(new(options.LLMInstantAppListOptions))
	cmd.Show(new(options.LLMInstantAppShowOptions))
	// cmd.Update(new(options.InstantAppUpdateOptions))
	cmd.Create(new(options.LLMInstantAppCreateOptions))
	// cmd.Delete(new(options.InstantAppDeleteOptions))
	cmd.Perform("syncstatus", new(commonoptions.BaseIdOptions))
	cmd.Perform("change-owner", new(commonoptions.ChangeOwnerOptions))
	cmd.Perform("enable", new(commonoptions.BaseIdOptions))
	cmd.Perform("disable", new(commonoptions.BaseIdOptions))
	cmd.Perform("public", new(commonoptions.BasePublicOptions))
	cmd.Perform("private", new(commonoptions.BaseIdOptions))
	// cmd.PerformClass("import", new(options.InstantAppImportOptions))
}
