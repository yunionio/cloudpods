package llm

import (
	"io"
	"os"

	"github.com/cheggaaa/pb/v3"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
	options "yunion.io/x/onecloud/pkg/mcclient/options/llm"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.LLMBenchmarks)
	cmd.List(new(options.LLMBenchmarkListOptions))
	cmd.Show(new(options.LLMBenchmarkShowOptions))
	cmd.Create(new(options.LLMBenchmarkCreateOptions))
	cmd.Update(new(options.LLMBenchmarkUpdateOptions))
	cmd.Delete(new(options.LLMBenchmarkDeleteOptions))
	cmd.Perform("copy", new(options.LLMBenchmarkCopyOptions))
	cmd.Perform("retest", new(options.LLMBenchmarkRetestOptions))
	cmd.Perform("stop", new(options.LLMBenchmarkStopOptions))

	shell.R(new(options.LLMBenchmarkArtifactOptions), "llm-benchmark-artifact", "Download benchmark artifact", func(s *mcclient.ClientSession, args *options.LLMBenchmarkArtifactOptions) error {
		src, size, err := modules.LLMBenchmarks.Artifact(s, args.ID, args.Type)
		if err != nil {
			return errors.Wrap(err, "download artifact")
		}
		defer src.Close()
		return writeBenchmarkArtifact(src, size, args.Output)
	})
}

func writeBenchmarkArtifact(src io.Reader, size int64, output string) error {
	if output == "" {
		_, err := io.Copy(os.Stdout, src)
		return err
	}
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()
	if size < 0 {
		_, err = io.Copy(f, src)
		return err
	}
	bar := pb.Full.Start64(size)
	_, err = io.Copy(f, bar.NewProxyReader(src))
	return err
}
