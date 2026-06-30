package models

import (
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestResolveLLMStatusFromPod(t *testing.T) {
	cases := []struct {
		name                   string
		currentStatus          string
		serverStatus           string
		primaryContainerStatus string
		wantStatus             string
		wantUpdate             bool
	}{
		{
			name:                   "server ready container exited after start_fail sync",
			currentStatus:          api.LLM_STATUS_START_FAIL,
			serverStatus:           computeapi.VM_READY,
			primaryContainerStatus: computeapi.CONTAINER_STATUS_EXITED,
			wantStatus:             api.LLM_STATUS_READY,
			wantUpdate:             true,
		},
		{
			name:                   "server ready stale probe_failed after start_fail sync",
			currentStatus:          api.LLM_STATUS_START_FAIL,
			serverStatus:           computeapi.VM_READY,
			primaryContainerStatus: computeapi.CONTAINER_STATUS_PROBE_FAILED,
			wantStatus:             api.LLM_STATUS_READY,
			wantUpdate:             true,
		},
		{
			name:                   "already ready no update",
			currentStatus:          api.LLM_STATUS_READY,
			serverStatus:           computeapi.VM_READY,
			primaryContainerStatus: computeapi.CONTAINER_STATUS_EXITED,
			wantUpdate:             false,
		},
		{
			name:                   "running with probe failed is start_fail",
			currentStatus:          api.LLM_STATUS_RUNNING,
			serverStatus:           computeapi.VM_RUNNING,
			primaryContainerStatus: computeapi.CONTAINER_STATUS_PROBE_FAILED,
			wantStatus:             api.LLM_STATUS_START_FAIL,
			wantUpdate:             true,
		},
		{
			name:                   "running container running",
			currentStatus:          api.LLM_STATUS_PROBING,
			serverStatus:           computeapi.VM_RUNNING,
			primaryContainerStatus: computeapi.CONTAINER_STATUS_RUNNING,
			wantStatus:             api.LLM_STATUS_RUNNING,
			wantUpdate:             true,
		},
		{
			name:                   "running container exited is start_fail",
			currentStatus:          api.LLM_STATUS_RUNNING,
			serverStatus:           computeapi.VM_RUNNING,
			primaryContainerStatus: computeapi.CONTAINER_STATUS_EXITED,
			wantStatus:             api.LLM_STATUS_START_FAIL,
			wantUpdate:             true,
		},
		{
			name:                   "start_fail recovers when server crash_loop but container running",
			currentStatus:          api.LLM_STATUS_START_FAIL,
			serverStatus:           computeapi.POD_STATUS_CRASH_LOOP_BACK_OFF,
			primaryContainerStatus: computeapi.CONTAINER_STATUS_RUNNING,
			wantStatus:             api.LLM_STATUS_RUNNING,
			wantUpdate:             true,
		},
		{
			name:                   "start_fail recovers when server and container running",
			currentStatus:          api.LLM_STATUS_START_FAIL,
			serverStatus:           computeapi.VM_RUNNING,
			primaryContainerStatus: computeapi.CONTAINER_STATUS_RUNNING,
			wantStatus:             api.LLM_STATUS_RUNNING,
			wantUpdate:             true,
		},
		{
			name:                   "running degrades when container crash_loop",
			currentStatus:          api.LLM_STATUS_RUNNING,
			serverStatus:           computeapi.VM_RUNNING,
			primaryContainerStatus: computeapi.CONTAINER_STATUS_CRASH_LOOP_BACK_OFF,
			wantStatus:             api.LLM_STATUS_START_FAIL,
			wantUpdate:             true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveLLMStatusFromPod(tc.currentStatus, tc.serverStatus, tc.primaryContainerStatus)
			if got.Update != tc.wantUpdate {
				t.Fatalf("Update = %v, want %v", got.Update, tc.wantUpdate)
			}
			if tc.wantUpdate && got.Status != tc.wantStatus {
				t.Fatalf("Status = %q, want %q", got.Status, tc.wantStatus)
			}
		})
	}
}
