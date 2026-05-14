package llm

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
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

func (m *LLMModelSetsManager) GetSpecific(session *mcclient.ClientSession, id string, spec string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if spec != "specs" {
		return m.ResourceManager.GetSpecific(session, id, spec, params)
	}
	path := fmt.Sprintf("/%s/%s/%s", m.ContextPath(nil), url.PathEscape(id), url.PathEscape(spec))
	if params != nil {
		if qs := params.QueryString(); qs != "" {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return modulebase.Get(m.ResourceManager, session, path, "")
}

type LLMModelSpecsManager struct {
	modulebase.ResourceManager
}
