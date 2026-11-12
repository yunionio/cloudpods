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

	RequestFormat   string  `json:"request_format" default:"/v1/chat/completions"`
	Model           string  `json:"model" help:"served model name; required for an external target"`
	TargetUrl       string  `json:"target_url" help:"external OpenAI-compatible service root, mutually exclusive with llm"`
	ApiKey          string  `json:"api_key" help:"optional API key for an external target; not returned by list or get"`
	Tokenizer       string  `json:"tokenizer" help:"optional tokenizer name for an external target; defaults to the model"`
	NetworkId       string  `json:"network_id" help:"network used by the external benchmark runner"`
	Profile         string  `json:"profile" choices:"constant|peak|slo"`
	RequestRate     float64 `json:"request_rate"`
	WarmupRequests  *int    `json:"warmup_requests"`
	SloAvgTtftMs    float64 `json:"slo_avg_ttft_ms"`
	SloAvgTpotMs    float64 `json:"slo_avg_tpot_ms"`
	SloAvgLatencyMs float64 `json:"slo_avg_latency_ms"`

	TotalRequests      int  `json:"total_requests"`
	MaxDurationSeconds int  `json:"max_duration_seconds"`
	MaxErrors          int  `json:"max_errors"`
	KeepRunner         bool `json:"keep_runner" help:"keep the runner container after the benchmark completes or fails"`

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

	Name                string   `json:"name,omitempty"`
	Description         *string  `json:"description,omitempty"`
	Model               *string  `json:"model,omitempty"`
	TargetUrl           *string  `json:"target_url,omitempty"`
	ApiKey              *string  `json:"api_key,omitempty"`
	Tokenizer           *string  `json:"tokenizer,omitempty"`
	NetworkId           *string  `json:"network_id,omitempty"`
	RequestRate         *float64 `json:"request_rate,omitempty"`
	WarmupRequests      *int     `json:"warmup_requests,omitempty"`
	TotalRequests       *int     `json:"total_requests,omitempty"`
	MaxDurationSeconds  *int     `json:"max_duration_seconds,omitempty"`
	MaxErrors           *int     `json:"max_errors,omitempty"`
	DatasetInputTokens  *int     `json:"dataset_input_tokens,omitempty"`
	DatasetOutputTokens *int     `json:"dataset_output_tokens,omitempty"`
}

func (o *LLMBenchmarkUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMBenchmarkUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMBenchmarkCopyOptions struct {
	options.BaseIdOptions

	NAME                string   `json:"name"`
	LLMDeploymentId     string   `json:"llm_deployment_id"`
	Description         *string  `json:"description,omitempty"`
	Model               *string  `json:"model,omitempty"`
	TargetUrl           *string  `json:"target_url,omitempty"`
	ApiKey              *string  `json:"api_key,omitempty"`
	Tokenizer           *string  `json:"tokenizer,omitempty"`
	NetworkId           *string  `json:"network_id,omitempty"`
	RequestRate         *float64 `json:"request_rate,omitempty"`
	WarmupRequests      *int     `json:"warmup_requests,omitempty"`
	TotalRequests       *int     `json:"total_requests,omitempty"`
	MaxDurationSeconds  *int     `json:"max_duration_seconds,omitempty"`
	MaxErrors           *int     `json:"max_errors,omitempty"`
	DatasetInputTokens  *int     `json:"dataset_input_tokens,omitempty"`
	DatasetOutputTokens *int     `json:"dataset_output_tokens,omitempty"`
	KeepRunner          *bool    `json:"keep_runner,omitempty" help:"keep the runner container after the benchmark completes or fails"`
}

func (o *LLMBenchmarkCopyOptions) GetId() string {
	return o.ID
}

func (o *LLMBenchmarkCopyOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMBenchmarkRetestOptions struct {
	options.BaseIdOptions

	ReuseRunner bool `json:"reuse_runner" help:"rerun inside the existing runner container"`
}

func (o *LLMBenchmarkRetestOptions) GetId() string {
	return o.ID
}

func (o *LLMBenchmarkRetestOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
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
