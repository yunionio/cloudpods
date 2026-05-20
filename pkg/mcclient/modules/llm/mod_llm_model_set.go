package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	LLMModelSets  LLMModelSetsManager
	LLMModelSpecs LLMModelSpecsManager
)

func init() {
	LLMModelSets = LLMModelSetsManager{
		modules.NewLLMManager("llm_model_set", "llm_model_sets",
			[]string{"id", "name", "categories", "parameter_size_b"},
			[]string{}),
	}
	modules.Register(&LLMModelSets)

	LLMModelSpecs = LLMModelSpecsManager{
		modules.NewLLMManager("llm_model_spec", "llm_model_specs",
			[]string{"id", "label", "quantization"},
			[]string{}),
	}
	modules.Register(&LLMModelSpecs)
}

type LLMModelSetsManager struct {
	modulebase.ResourceManager
}

type LLMModelSpecsManager struct {
	modulebase.ResourceManager
}
