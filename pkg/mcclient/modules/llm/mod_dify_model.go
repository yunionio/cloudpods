package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type DifyModelManager struct {
	modulebase.ResourceManager
}

var (
	DifyModel DifyModelManager
)

func init() {
	DifyModel = DifyModelManager{
		ResourceManager: modules.NewLLMManager("dify_model", "dify_models",
			[]string{},
			[]string{},
		),
	}
	modules.Register(&DifyModel)
}
