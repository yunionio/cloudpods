package models

import api "yunion.io/x/onecloud/pkg/apis/llm"

func benchmarkRunFinalState(state string, stopRequested bool, runErr error) (string, string) {
	if stopRequested || state == api.LLMBenchmarkStateStopped {
		return api.LLMBenchmarkStateStopped, ""
	}
	if runErr != nil {
		return api.LLMBenchmarkStateError, runErr.Error()
	}
	return api.LLMBenchmarkStateCompleted, ""
}
