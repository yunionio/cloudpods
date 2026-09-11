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
