package models

import (
	"context"
	"fmt"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

type LLMStatusResolution struct {
	Status string
	Reason string
	Update bool
}

type llmStatusResolution = LLMStatusResolution

func ResolveLLMStatusFromServerDetails(ctx context.Context, llm *SLLM, server *computeapi.ServerDetails) (LLMStatusResolution, error) {
	if llm == nil || server == nil {
		return LLMStatusResolution{}, nil
	}
	primaryStatus, err := getLLMPrimaryContainerStatus(ctx, llm, server.Containers)
	if err != nil {
		return LLMStatusResolution{}, err
	}
	return resolveLLMStatusFromPod(llm.Status, server.Status, primaryStatus), nil
}

func resolveLLMStatusFromPod(currentStatus string, serverStatus string, primaryContainerStatus string) llmStatusResolution {
	targetStatus := ""
	reason := fmt.Sprintf("pod status=%s primary_container_status=%s", serverStatus, primaryContainerStatus)

	switch {
	case serverStatus == computeapi.VM_READY:
		targetStatus = api.LLM_STATUS_READY
	case isPrimaryContainerRunning(primaryContainerStatus):
		targetStatus = api.LLM_STATUS_RUNNING
	case serverStatus == computeapi.VM_RUNNING && primaryContainerStatus == computeapi.CONTAINER_STATUS_PROBING:
		targetStatus = api.LLM_STATUS_PROBING
	case isLLMPodCrashLoopStatus(serverStatus, primaryContainerStatus) || isLLMStartupProbeFailedStatus(primaryContainerStatus):
		if currentStatus == commonapi.STATUS_CREATING {
			targetStatus = api.LLM_STATUS_CREATE_FAIL
		} else {
			targetStatus = api.LLM_STATUS_START_FAIL
		}
	case serverStatus == computeapi.POD_STATUS_CONTAINER_EXITED || primaryContainerStatus == computeapi.CONTAINER_STATUS_EXITED:
		targetStatus = api.LLM_STATUS_START_FAIL
	default:
		return llmStatusResolution{}
	}

	if targetStatus == currentStatus {
		return llmStatusResolution{}
	}
	if !canWatchUpdateLLMStatus(currentStatus, targetStatus) {
		return llmStatusResolution{}
	}
	return llmStatusResolution{
		Status: targetStatus,
		Reason: reason,
		Update: true,
	}
}

func isLLMPodCrashLoopStatus(serverStatus string, primaryContainerStatus string) bool {
	if isPrimaryContainerRunning(primaryContainerStatus) {
		return false
	}
	return serverStatus == computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF ||
		primaryContainerStatus == computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF
}

func isPrimaryContainerRunning(status string) bool {
	return status == computeapi.CONTAINER_STATUS_RUNNING
}

func isLLMStartupProbeFailedStatus(status string) bool {
	return status == computeapi.CONTAINER_STATUS_PROBE_FAILED ||
		status == computeapi.CONTAINER_STATUS_NET_FAILED
}

func canWatchUpdateLLMStatus(currentStatus string, targetStatus string) bool {
	if currentStatus == commonapi.STATUS_CREATING && targetStatus == api.LLM_STATUS_CREATE_FAIL {
		return true
	}
	if currentStatus == commonapi.STATUS_CREATING && targetStatus == api.LLM_STATUS_PROBING {
		return true
	}
	if currentStatus == api.LLM_STATUS_START_RESTART && targetStatus == api.LLM_STATUS_PROBING {
		return true
	}

	switch currentStatus {
	case api.LLM_STATUS_READY,
		api.LLM_STATUS_RUNNING,
		api.LLM_STATUS_PROBING,
		api.LLM_STATUS_UNKNOWN,
		api.LLM_STATUS_START_SYNCSTATUS,
		api.LLM_STATUS_SYNCSTATUS,
		api.LLM_STATUS_CREATE_FAIL,
		api.LLM_STATUS_START_FAIL:
		return true
	}

	return false
}
