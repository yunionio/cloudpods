package llm

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMBenchmarkShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMBenchmarkShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMBenchmarkListOptions struct {
	options.BaseListOptions

	LLMId           string `json:"llm_id" token:"llm" help:"filter by LLM id"`
	LLMDeploymentId string `json:"llm_deployment_id" token:"llm-deployment" help:"filter by LLM deployment id or name"`
	State           string `json:"state" choices:"pending|queued|running|completed|stopped|error" help:"filter by state"`
}

func (o *LLMBenchmarkListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMBenchmarkCreateOptions struct {
	apis.VirtualResourceCreateInput

	LLMId            string `json:"llm_id" token:"llm" help:"target LLM id or name"`
	LLMDeploymentId  string `json:"llm_deployment_id" token:"llm-deployment" help:"target LLM deployment id or name"`
	BenchmarkImage   string `json:"benchmark_image" help:"benchmark llm_image id or name"`
	BenchmarkPackage string `json:"benchmark_package" help:"benchmark package id or name"`

	RequestFormat string `json:"request_format" default:"/v1/chat/completions"`
	Model         string `json:"model"`
	Profile       string `json:"profile" choices:"constant"`
	RequestRate   int    `json:"request_rate"`

	TotalRequests      int `json:"total_requests"`
	MaxDurationSeconds int `json:"max_duration_seconds"`
	MaxErrors          int `json:"max_errors"`

	DatasetName         string `json:"dataset_name" choices:"synthetic_text|benchmark_package"`
	DatasetInputTokens  int    `json:"dataset_input_tokens"`
	DatasetOutputTokens int    `json:"dataset_output_tokens"`
	DatasetPath         string `json:"dataset_path"`
}

func (o *LLMBenchmarkCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type LLMBenchmarkUpdateOptions struct {
	options.BaseIdOptions

	Name                string  `json:"name,omitempty"`
	Description         *string `json:"description,omitempty"`
	Model               *string `json:"model,omitempty"`
	RequestRate         *int    `json:"request_rate,omitempty"`
	TotalRequests       *int    `json:"total_requests,omitempty"`
	MaxDurationSeconds  *int    `json:"max_duration_seconds,omitempty"`
	MaxErrors           *int    `json:"max_errors,omitempty"`
	DatasetInputTokens  *int    `json:"dataset_input_tokens,omitempty"`
	DatasetOutputTokens *int    `json:"dataset_output_tokens,omitempty"`
}

func (o *LLMBenchmarkUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMBenchmarkUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMBenchmarkCopyOptions struct {
	options.BaseIdOptions

	NAME                string  `json:"name"`
	LLMDeploymentId     string  `json:"llm_deployment_id"`
	Description         *string `json:"description,omitempty"`
	Model               *string `json:"model,omitempty"`
	RequestRate         *int    `json:"request_rate,omitempty"`
	TotalRequests       *int    `json:"total_requests,omitempty"`
	MaxDurationSeconds  *int    `json:"max_duration_seconds,omitempty"`
	MaxErrors           *int    `json:"max_errors,omitempty"`
	DatasetInputTokens  *int    `json:"dataset_input_tokens,omitempty"`
	DatasetOutputTokens *int    `json:"dataset_output_tokens,omitempty"`
}

func (o *LLMBenchmarkCopyOptions) GetId() string {
	return o.ID
}

func (o *LLMBenchmarkCopyOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMBenchmarkRetestOptions struct {
	options.BaseIdOptions
}

func (o *LLMBenchmarkRetestOptions) GetId() string {
	return o.ID
}

func (o *LLMBenchmarkRetestOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}

type LLMBenchmarkDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMBenchmarkDeleteOptions) GetId() string {
	return o.ID
}

type LLMBenchmarkStopOptions struct {
	options.BaseIdOptions
}

func (o *LLMBenchmarkStopOptions) GetId() string {
	return o.ID
}

type LLMBenchmarkArtifactOptions struct {
	ID     string `help:"ID or name of benchmark"`
	Type   string `help:"artifact type" choices:"log|json|csv"`
	Output string `short-token:"o" help:"destination file, if omitted, output to stdout"`
}
