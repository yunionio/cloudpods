package llm

import (
	"yunion.io/x/onecloud/cmd/climc/shell"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMBenchmarkPackages)
	cmd.List(new(options.LLMBenchmarkPackageListOptions))
	cmd.Show(new(options.LLMBenchmarkPackageShowOptions))
	cmd.Create(new(options.LLMBenchmarkPackageCreateOptions))
	cmd.Delete(new(options.LLMBenchmarkPackageDeleteOptions))
	cmd.PerformClass("import", new(options.LLMBenchmarkPackageImportOptions))
}
