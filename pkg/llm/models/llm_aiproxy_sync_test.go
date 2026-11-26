package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestShouldUnsyncAiproxyOnLeaveRunning(t *testing.T) {
	if shouldUnsyncAiproxyOnLeaveRunning() {
		t.Fatal("leaving running must not unsync aiproxy; catalog is removed only on deployment delete or unregister")
	}
}

func TestUnsyncKeepsDeploymentRouting(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "keeps existing id on last replica", in: "routing-abc", want: "routing-abc"},
		{name: "trims space", in: "  routing-abc  ", want: "routing-abc"},
		{name: "empty stays empty", in: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, deleteRouting := unsyncKeepsDeploymentRouting(tc.in)
			if deleteRouting {
				t.Fatalf("deleteRouting = true, want false (routing must survive unsync)")
			}
			if got != tc.want {
				t.Fatalf("keepRoutingId = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAiproxyStatusWhenNoRunningReplicas(t *testing.T) {
	synced, syncedMsg := aiproxyStatusWhenNoRunningReplicas(&SLLMDeployment{Replicas: 0})
	if synced != api.AIPROXY_SYNC_STATUS_SYNCED {
		t.Fatalf("replicas=0 status = %q, want %q", synced, api.AIPROXY_SYNC_STATUS_SYNCED)
	}
	if syncedMsg != "scaled to zero, routing disabled" {
		t.Fatalf("replicas=0 message = %q, want the routing-disabled explanation", syncedMsg)
	}

	pending, pendingMsg := aiproxyStatusWhenNoRunningReplicas(&SLLMDeployment{Replicas: 2})
	if pending != api.AIPROXY_SYNC_STATUS_PENDING {
		t.Fatalf("replicas>0 status = %q, want %q", pending, api.AIPROXY_SYNC_STATUS_PENDING)
	}
	if pendingMsg != "waiting for running replicas" {
		t.Fatalf("replicas>0 message = %q", pendingMsg)
	}

	if got := inferAiproxySyncStatus(&SLLMDeployment{AutoRegisterAiproxy: true, Replicas: 0}); got != api.AIPROXY_SYNC_STATUS_SYNCED {
		t.Fatalf("infer replicas=0 = %q, want synced", got)
	}
	if got := inferAiproxySyncStatus(&SLLMDeployment{AutoRegisterAiproxy: true, Replicas: 1}); got != api.AIPROXY_SYNC_STATUS_PENDING {
		t.Fatalf("infer replicas>0 = %q, want pending", got)
	}
}

func TestShouldDisableAiRouting(t *testing.T) {
	cases := []struct {
		name string
		dep  *SLLMDeployment
		want bool
	}{
		{name: "nil deployment", dep: nil, want: false},
		{name: "auto register off", dep: &SLLMDeployment{Replicas: 0}, want: false},
		{name: "scaled to zero", dep: &SLLMDeployment{AutoRegisterAiproxy: true, Replicas: 0}, want: true},
		{name: "negative replicas", dep: &SLLMDeployment{AutoRegisterAiproxy: true, Replicas: -1}, want: true},
		{name: "replicas>0 keeps routing enabled", dep: &SLLMDeployment{AutoRegisterAiproxy: true, Replicas: 2}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldDisableAiRouting(tc.dep); got != tc.want {
				t.Fatalf("shouldDisableAiRouting = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAiproxyCatalogDeleteEntrypoints(t *testing.T) {
	if api.AIPROXY_SYNC_STATUS_DISABLED == "" {
		t.Fatal("disabled status must remain the unregister/delete end state")
	}
	if shouldUnsyncAiproxyOnLeaveRunning() {
		t.Fatal("status leave-running is not a catalog delete entrypoint")
	}
	_, deleteRouting := unsyncKeepsDeploymentRouting("keep-me")
	if deleteRouting {
		t.Fatal("UnsyncLlmInstance must not delete deployment routing")
	}
}
