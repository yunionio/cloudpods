package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	commonoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMInstantModel)
	cmd.List(new(options.LLMInstantModelListOptions))
	cmd.Show(new(options.LLMInstantModelShowOptions))
	cmd.Update(new(options.LLMInstantModelUpdateOptions))
	cmd.Create(new(options.LLMInstantModelCreateOptions))
	cmd.Delete(new(options.LLMInstantModelDeleteOptions))
	cmd.Perform("syncstatus", new(commonoptions.BaseIdOptions))
	cmd.Perform("change-owner", new(commonoptions.ChangeOwnerOptions))
	cmd.Perform("enable", new(commonoptions.BaseIdOptions))
	cmd.Perform("disable", new(commonoptions.BaseIdOptions))
	cmd.Perform("public", new(commonoptions.BasePublicOptions))
	cmd.Perform("private", new(commonoptions.BaseIdOptions))
	cmd.PerformClass("import", new(options.LLMInstantModelImportOptions))
	cmd.GetProperty(new(options.LLMInstantModelCommunityRegistryOptions))
}
