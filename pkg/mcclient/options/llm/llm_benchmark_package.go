package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMBenchmarkPackageShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMBenchmarkPackageShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMBenchmarkPackageListOptions struct {
	options.BaseListOptions

	Source string `json:"source" choices:"huggingface"`
	RepoId string `json:"repo_id"`
	Format string `json:"format" choices:"guidellm_jsonl"`
}

func (o *LLMBenchmarkPackageListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMBenchmarkPackageCreateOptions struct {
	apis.SharableVirtualResourceCreateInput

	Source   string `json:"source" choices:"huggingface"`
	RepoId   string `json:"repo_id"`
	Revision string `json:"revision"`
	FilePath string `json:"file_path"`
	Format   string `json:"format" choices:"guidellm_jsonl"`
	ImageId  string `json:"image_id"`
}

func (o *LLMBenchmarkPackageCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMBenchmarkPackageImportOptions struct {
	NAME      string `json:"name" help:"benchmark package name"`
	REPO_ID   string `json:"repo_id" help:"HuggingFace dataset repo id, e.g. org/dataset"`
	FILE_PATH string `json:"file_path" help:"JSONL file path in the dataset repo"`
	Revision  string `json:"revision" help:"HuggingFace revision, default main"`
}

func (o *LLMBenchmarkPackageImportOptions) Params() (jsonutils.JSONObject, error) {
	input := api.LLMBenchmarkPackageImportInput{
		LLMBenchmarkPackageCreateInput: api.LLMBenchmarkPackageCreateInput{
			Source:   api.LLMBenchmarkPackageSourceHuggingFace,
			RepoId:   o.REPO_ID,
			Revision: o.Revision,
			FilePath: o.FILE_PATH,
			Format:   api.LLMBenchmarkPackageFormatGuideLLMJSONL,
		},
	}
	input.Name = o.NAME
	return jsonutils.Marshal(input), nil
}

type LLMBenchmarkPackageDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMBenchmarkPackageDeleteOptions) GetId() string {
	return o.ID
}
