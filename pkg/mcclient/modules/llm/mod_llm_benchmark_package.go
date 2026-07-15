package llm

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type LLMBenchmarkPackageManager struct {
	modulebase.ResourceManager
}

var (
	LLMBenchmarkPackages LLMBenchmarkPackageManager
)

func init() {
	LLMBenchmarkPackages = LLMBenchmarkPackageManager{
		ResourceManager: modules.NewLLMManager("llm_benchmark_package", "llm_benchmark_packages",
			[]string{
				"ID",
				"Name",
				"Status",
				"Source",
				"RepoId",
				"Revision",
				"FilePath",
				"Format",
				"ImageId",
				"DatasetPath",
			},
			[]string{},
		),
	}
	modules.Register(&LLMBenchmarkPackages)
}
