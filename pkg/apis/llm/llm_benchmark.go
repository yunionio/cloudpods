package llm

import (
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	LLMBenchmarkStatePending    = "pending"
	LLMBenchmarkStateQueued     = "queued"
	LLMBenchmarkStateValidating = "validating"
	LLMBenchmarkStateRunning    = "running"
	LLMBenchmarkStateCompleted  = "completed"
	LLMBenchmarkStateStopped    = "stopped"
	LLMBenchmarkStateError      = "error"
)

const (
	LLMBenchmarkDatasetSyntheticText = "synthetic_text"
	LLMBenchmarkDatasetPackage       = "benchmark_package"
	LLMBenchmarkProfileConstant      = "constant"
)

const (
	LLMBenchmarkEvaluationStateEvaluating = "evaluating"
	LLMBenchmarkEvaluationStateCompleted  = "completed"
	LLMBenchmarkEvaluationStateSkipped    = "skipped"
	LLMBenchmarkEvaluationStateError      = "error"
)

const (
	LLMBenchmarkArtifactStorageLocal = "local"
	LLMBenchmarkArtifactStorageMinio = "minio"
)

const (
	LLMBenchmarkDefaultRequestFormat = "/v1/chat/completions"
	LLMBenchmarkDefaultImage         = "registry.cn-beijing.aliyuncs.com/cloudpods/guidellm:v0.7.0-amd64"
)

type LLMBenchmarkCreateInput struct {
	apis.VirtualResourceCreateInput

	LLMId            string `json:"llm_id"`
	BenchmarkImage   string `json:"benchmark_image,omitempty"`
	BenchmarkPackage string `json:"benchmark_package,omitempty"`

	RequestFormat string `json:"request_format,omitempty"`
	Model         string `json:"model,omitempty"`

	Profile     string `json:"profile,omitempty"`
	RequestRate int    `json:"request_rate,omitempty"`

	TotalRequests      int `json:"total_requests,omitempty"`
	MaxDurationSeconds int `json:"max_duration_seconds,omitempty"`
	MaxErrors          int `json:"max_errors,omitempty"`

	DatasetName         string `json:"dataset_name,omitempty"`
	DatasetInputTokens  int    `json:"dataset_input_tokens,omitempty"`
	DatasetOutputTokens int    `json:"dataset_output_tokens,omitempty"`
	DatasetPath         string `json:"dataset_path,omitempty"`

	LLMDeploymentId    string `json:"llm_deployment_id,omitempty"`
	LLMSkuId           string `json:"llm_sku_id,omitempty"`
	LLMImageId         string `json:"llm_image_id,omitempty"`
	BenchmarkPackageId string `json:"benchmark_package_id,omitempty"`
	Backend            string `json:"backend,omitempty"`
	TargetUrl          string `json:"target_url,omitempty"`
	WorkDir            string `json:"work_dir,omitempty"`
	TargetSnapshot     string `json:"target_snapshot,omitempty"`
	GuideLLMSpec       string `json:"guide_llm_spec,omitempty"`
}

type LLMBenchmarkCopyInput struct {
	Name            string `json:"name"`
	LLMDeploymentId string `json:"llm_deployment_id"`

	Description         *string `json:"description,omitempty"`
	Model               *string `json:"model,omitempty"`
	RequestRate         *int    `json:"request_rate,omitempty"`
	TotalRequests       *int    `json:"total_requests,omitempty"`
	MaxDurationSeconds  *int    `json:"max_duration_seconds,omitempty"`
	MaxErrors           *int    `json:"max_errors,omitempty"`
	DatasetInputTokens  *int    `json:"dataset_input_tokens,omitempty"`
	DatasetOutputTokens *int    `json:"dataset_output_tokens,omitempty"`
}

type LLMBenchmarkUpdateInput struct {
	apis.VirtualResourceBaseUpdateInput

	Model               *string `json:"model,omitempty"`
	RequestRate         *int    `json:"request_rate,omitempty"`
	TotalRequests       *int    `json:"total_requests,omitempty"`
	MaxDurationSeconds  *int    `json:"max_duration_seconds,omitempty"`
	MaxErrors           *int    `json:"max_errors,omitempty"`
	DatasetInputTokens  *int    `json:"dataset_input_tokens,omitempty"`
	DatasetOutputTokens *int    `json:"dataset_output_tokens,omitempty"`
}

type LLMBenchmarkRetestInput struct{}

type LLMBenchmarkListInput struct {
	apis.VirtualResourceListInput

	LLMId           string `json:"llm_id"`
	LLMDeploymentId string `json:"llm_deployment_id"`
	State           string `json:"state"`
}

type LLMBenchmarkDatasetPreflight struct {
	State              string  `json:"state"`
	ExpectedSamples    int     `json:"expected_samples"`
	ActualSamples      int     `json:"actual_samples"`
	Successful         int     `json:"successful"`
	Errored            int     `json:"errored"`
	ErrorRate          float64 `json:"error_rate"`
	LatencyMeanSeconds float64 `json:"latency_mean_sec"`
	Message            string  `json:"message"`
}

func (p *LLMBenchmarkDatasetPreflight) String() string {
	return jsonutils.Marshal(p).String()
}

func (p *LLMBenchmarkDatasetPreflight) IsZero() bool {
	return p == nil || *p == (LLMBenchmarkDatasetPreflight{})
}

type LLMBenchmarkDatasetEvaluation struct {
	State        string  `json:"state"`
	AnswerColumn string  `json:"answer_column,omitempty"`
	RequestTotal int     `json:"request_total"`
	Evaluated    int     `json:"evaluated"`
	Correct      int     `json:"correct"`
	Incorrect    int     `json:"incorrect"`
	Unscored     int     `json:"unscored"`
	Accuracy     float64 `json:"accuracy"`
	Message      string  `json:"message,omitempty"`
}

func (e *LLMBenchmarkDatasetEvaluation) String() string {
	return jsonutils.Marshal(e).String()
}

func (e *LLMBenchmarkDatasetEvaluation) IsZero() bool {
	return e == nil || *e == (LLMBenchmarkDatasetEvaluation{})
}

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(new(LLMBenchmarkDatasetPreflight)), func() gotypes.ISerializable {
		return new(LLMBenchmarkDatasetPreflight)
	})
	gotypes.RegisterSerializable(reflect.TypeOf(new(LLMBenchmarkDatasetEvaluation)), func() gotypes.ISerializable {
		return new(LLMBenchmarkDatasetEvaluation)
	})
}

type LLMBenchmarkDetails struct {
	apis.VirtualResourceDetails

	LLMDeployment string `json:"llm_deployment"`
}
