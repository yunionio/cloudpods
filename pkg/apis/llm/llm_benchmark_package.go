package llm

import "yunion.io/x/onecloud/pkg/apis"

const (
	LLMBenchmarkPackageSourceHuggingFace   = "huggingface"
	LLMBenchmarkPackageFormatGuideLLMJSONL = "guidellm_jsonl"

	LLMBenchmarkPackageMountBase = "/data/benchmark-packages"
)

type LLMBenchmarkPackageCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	Source       string `json:"source,omitempty"`
	RepoId       string `json:"repo_id,omitempty"`
	Revision     string `json:"revision,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
	Format       string `json:"format,omitempty"`
	AnswerColumn string `json:"answer_column,omitempty"`

	ImageId      string `json:"image_id,omitempty"`
	Size         int64  `json:"size,omitempty"`
	ActualSizeMb int32  `json:"actual_size_mb,omitempty"`

	MountPath   string `json:"mount_path,omitempty"`
	DatasetPath string `json:"dataset_path,omitempty"`
	Manifest    string `json:"manifest,omitempty"`
}

type LLMBenchmarkPackageImportInput struct {
	LLMBenchmarkPackageCreateInput
	BenchmarkSpec *LLMBenchmarkCreateInput `json:"benchmark_spec,omitempty"`
}

type LLMBenchmarkPackageListInput struct {
	apis.SharableVirtualResourceListInput

	Source string `json:"source"`
	RepoId string `json:"repo_id"`
	Format string `json:"format"`
}
