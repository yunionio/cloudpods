package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type LLMDeploymentManager struct {
	modulebase.ResourceManager
}

var (
	LLMDeployments LLMDeploymentManager
)

func init() {
	LLMDeployments = LLMDeploymentManager{
		ResourceManager: modules.NewLLMManager("llm_deployment", "llm_deployments",
			[]string{},
			[]string{},
		),
	}
	modules.Register(&LLMDeployments)
}
