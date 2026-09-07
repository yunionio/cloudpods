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

const LLMServiceReadyTimeoutSeconds = 1800

const llmCrashLoopFailThreshold = 3

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

func isLLMCrashLoopStatus(status string) bool {
	return status == computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF ||
		status == computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF
}

func isLLMServiceFailedContainerStatus(status string) bool {
	return status == computeapi.CONTAINER_STATUS_PROBE_FAILED ||
		status == computeapi.CONTAINER_STATUS_NET_FAILED ||
		status == computeapi.CONTAINER_STATUS_EXITED
}

// crashLoopEpisodeAfter counts crash_loop_back_off episodes, not poll samples.
// The count increments only when entering crash_loop from a non-crash status.
// Staying in crash_loop does not increment. Leaving crash_loop keeps the count.
func crashLoopEpisodeAfter(episodes int, inCrashLoop bool, status string) (int, bool, bool) {
	if !isLLMCrashLoopStatus(status) {
		return episodes, false, false
	}
	if inCrashLoop {
		return episodes, true, false
	}
	next := episodes + 1
	return next, true, next >= llmCrashLoopFailThreshold
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
	crashLoopEpisodes := 0
	inCrashLoop := false
	if isLLMCrashLoopStatus(server.Status) {
		var failed bool
		crashLoopEpisodes, inCrashLoop, failed = crashLoopEpisodeAfter(crashLoopEpisodes, inCrashLoop, server.Status)
		if failed {
			return nil, errors.Wrapf(errors.ErrInvalidStatus, "server status %s (crash_loop episodes %d)", server.Status, crashLoopEpisodes)
		}
	} else if server.Status != computeapi.VM_RUNNING {
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
			crashLoopEpisodes, inCrashLoop, _ = crashLoopEpisodeAfter(crashLoopEpisodes, inCrashLoop, ctr.Status)
			if onProbing != nil && !probingNotified {
				if err := onProbing(); err != nil {
					return nil, errors.Wrap(err, "on probing")
				}
				probingNotified = true
			}
			time.Sleep(time.Second)
			continue
		}
		if isLLMCrashLoopStatus(ctr.Status) {
			var failed bool
			crashLoopEpisodes, inCrashLoop, failed = crashLoopEpisodeAfter(crashLoopEpisodes, inCrashLoop, ctr.Status)
			if failed {
				return nil, errors.Wrapf(errors.ErrInvalidStatus, "container status %s (crash_loop episodes %d)", ctr.Status, crashLoopEpisodes)
			}
			time.Sleep(time.Second)
			continue
		}
		crashLoopEpisodes, inCrashLoop, _ = crashLoopEpisodeAfter(crashLoopEpisodes, inCrashLoop, ctr.Status)
		if isLLMServiceFailedContainerStatus(ctr.Status) {
			return nil, errors.Wrapf(errors.ErrInvalidStatus, "container status %s", ctr.Status)
		}
		time.Sleep(time.Second)
	}
	return nil, errors.Wrapf(httperrors.ErrTimeout, "wait llm service ready timeout")
}
