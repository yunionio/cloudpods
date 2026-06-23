package llm_container

import commonapi "yunion.io/x/onecloud/pkg/apis"

const (
	llmStartupProbeTimeoutSeconds   int32 = 5
	llmStartupProbePeriodSeconds    int32 = 10
	llmStartupProbeSuccessThreshold int32 = 1
	llmStartupProbeFailureThreshold int32 = 360
)

func newLLMHTTPStartupProbe(port int, probePath string) *commonapi.ContainerProbe {
	return &commonapi.ContainerProbe{
		ContainerProbeHandler: commonapi.ContainerProbeHandler{
			HTTPGet: &commonapi.ContainerProbeHTTPGetAction{
				Path:   probePath,
				Port:   port,
				Scheme: commonapi.URISchemeHTTP,
			},
		},
		TimeoutSeconds:   llmStartupProbeTimeoutSeconds,
		PeriodSeconds:    llmStartupProbePeriodSeconds,
		SuccessThreshold: llmStartupProbeSuccessThreshold,
		FailureThreshold: llmStartupProbeFailureThreshold,
	}
}
