package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMDeployments)
	cmd.List(new(options.LLMDeploymentListOptions))
	cmd.Show(new(options.LLMDeploymentShowOptions))
	cmd.Create(new(options.LLMDeploymentCreateOptions))
	cmd.Update(new(options.LLMDeploymentUpdateOptions))
	cmd.Delete(new(options.LLMDeploymentDeleteOptions))
	cmd.Perform("register-aiproxy", new(options.LLMDeploymentRegisterAiproxyOptions))
	cmd.Perform("unregister-aiproxy", new(options.LLMDeploymentUnregisterAiproxyOptions))
	cmd.Perform("restart", new(options.LLMDeploymentRestartOptions))
	cmd.Perform("syncstatus", new(options.LLMDeploymentSyncstatusOptions))
}
