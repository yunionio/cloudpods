package llm

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	{
		cmd := shell.NewResourceCmd(&modules.LLMModelSets)
		cmd.List(new(options.LLMModelSetListOptions))
		cmd.Show(new(options.LLMModelSetShowOptions))
		cmd.GetWithCustomShow("specs", printModelSetSpecs, new(options.LLMModelSetSpecsOptions))
		cmd.PerformClass("refresh", new(options.LLMModelSetRefreshOptions))
	}
	{
		cmd := shell.NewResourceCmd(&modules.LLMModelSpecs)
		cmd.Show(new(options.LLMModelSpecShowOptions))
	}
}

func printModelSetSpecs(data jsonutils.JSONObject) {
	specs, _ := data.GetArray("llm_model_specs")
	total, _ := data.Int("total")
	if total == 0 {
		total = int64(len(specs))
	}
	shell.PrintList(&printutils.ListResult{
		Data:  specs,
		Total: int(total),
	}, []string{"id", "label", "quantization", "mode", "backend", "source", "huggingface_repo_id"})
}
