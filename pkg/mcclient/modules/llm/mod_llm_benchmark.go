package llm

import (
	"fmt"
	"io"
	"net/url"
	"strconv"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type LLMBenchmarkManager struct {
	modulebase.ResourceManager
}

var (
	LLMBenchmarks LLMBenchmarkManager
)

func (m *LLMBenchmarkManager) Artifact(s *mcclient.ClientSession, id string, kind string) (io.ReadCloser, int64, error) {
	path := fmt.Sprintf("/%s/%s/artifacts/%s", m.URLPath(), url.PathEscape(id), url.PathEscape(kind))
	resp, err := modulebase.RawRequest(m.ResourceManager, s, "GET", path, nil, nil)
	if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		sizeBytes, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			log.Errorf("download benchmark artifact unknown size")
			sizeBytes = -1
		}
		return resp.Body, sizeBytes, nil
	}
	_, _, err = s.ParseJSONResponse("", resp, err)
	return nil, -1, err
}

func init() {
	LLMBenchmarks = LLMBenchmarkManager{
		ResourceManager: modules.NewLLMManager("llm_benchmark", "llm_benchmarks",
			[]string{
				"ID",
				"Name",
				"LLMDeploymentId",
				"LLMDeployment",
				"LLMId",
				"State",
				"RequestRate",
				"TotalRequests",
				"RequestTotal",
				"RequestSuccessful",
				"RequestErrored",
				"ErrorRate",
			},
			[]string{},
		),
	}
	modules.Register(&LLMBenchmarks)
}
