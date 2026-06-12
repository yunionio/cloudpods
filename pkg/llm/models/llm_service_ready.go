package models

import (
	"context"
	"time"

	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	llmutils "yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const LLMServiceReadyTimeoutSeconds = 3600

var errLLMServiceProbing = errors.Error("llm service probing")

var llmServiceReadyServerStatuses = []string{
	computeapi.VM_RUNNING,
	computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF,
	computeapi.POD_STATUS_CONTAINER_EXITED,
	computeapi.POD_STATUS_UPLOADING_STATUS_FAILED,
}

var llmServiceReadyContainerStatuses = []string{
	computeapi.CONTAINER_STATUS_RUNNING,
	computeapi.CONTAINER_STATUS_PROBING,
	computeapi.CONTAINER_STATUS_PROBE_FAILED,
	computeapi.CONTAINER_STATUS_NET_FAILED,
	computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF,
	computeapi.CONTAINER_STATUS_EXITED,
}

func isLLMServiceReadyContainerStatus(status string) bool {
	return status == computeapi.CONTAINER_STATUS_RUNNING
}

func newLLMServiceProbingError(status string) error {
	return errors.Wrapf(errLLMServiceProbing, "container status %s", status)
}

func IsLLMServiceProbingError(err error) bool {
	return errors.Cause(err) == errLLMServiceProbing
}

func isLLMServiceFailedContainerStatus(status string) bool {
	return status == computeapi.CONTAINER_STATUS_PROBE_FAILED ||
		status == computeapi.CONTAINER_STATUS_NET_FAILED ||
		status == computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF ||
		status == computeapi.CONTAINER_STATUS_EXITED
}

func (llm *SLLM) WaitServiceReady(ctx context.Context, userCred mcclient.TokenCredential, timeoutSecs int) (*computeapi.SContainer, error) {
	return llm.WaitServiceReadyWithProbingCallback(ctx, userCred, timeoutSecs, nil)
}

func (llm *SLLM) WaitServiceReadyWithProbingCallback(ctx context.Context, userCred mcclient.TokenCredential, timeoutSecs int, onProbing func() error) (*computeapi.SContainer, error) {
	if timeoutSecs <= 0 {
		timeoutSecs = LLMServiceReadyTimeoutSeconds
	}

	server, err := llm.WaitServerStatus(ctx, userCred, llmServiceReadyServerStatuses, timeoutSecs)
	if err != nil {
		return nil, errors.Wrap(err, "WaitServerStatus")
	}
	if server.Status != computeapi.VM_RUNNING {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "server status %s", server.Status)
	}

	llmCtr, err := llm.GetLLMContainer()
	if err != nil {
		return nil, errors.Wrap(err, "GetLLMContainer")
	}

	expire := time.Now().Add(time.Second * time.Duration(timeoutSecs))
	probingNotified := false
	for time.Now().Before(expire) {
		ctr, err := llmutils.GetContainer(ctx, llmCtr.CmpId)
		if err != nil {
			return nil, errors.Wrap(err, "GetContainer")
		}
		if isLLMServiceReadyContainerStatus(ctr.Status) {
			return ctr, nil
		}
		if ctr.Status == computeapi.CONTAINER_STATUS_PROBING {
			if onProbing != nil && !probingNotified {
				if err := onProbing(); err != nil {
					return nil, errors.Wrap(err, "on probing")
				}
				probingNotified = true
			}
			time.Sleep(time.Second)
			continue
		}
		if isLLMServiceFailedContainerStatus(ctr.Status) {
			return nil, errors.Wrapf(errors.ErrInvalidStatus, "container status %s", ctr.Status)
		}
		time.Sleep(time.Second)
	}
	return nil, errors.Wrapf(httperrors.ErrTimeout, "wait llm service ready timeout")
}
