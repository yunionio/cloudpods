package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestCanRestartDeploymentStatus(t *testing.T) {
	cases := []struct {
		status string
		force  bool
		want   bool
	}{
		{api.STATUS_READY, false, true},
		{api.LLM_DEPLOYMENT_STATUS_PARTIAL, false, true},
		{api.LLM_STATUS_RUNNING, false, true},
		{api.LLM_DEPLOYMENT_STATUS_DEPLOYING, false, false},
		{api.LLM_STATUS_DELETING, false, false},
		{api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL, false, false},
		{api.LLM_DEPLOYMENT_STATUS_CREATING_SKU, false, false},
		{api.LLM_STATUS_CREATE_FAIL, false, false},
		{api.LLM_STATUS_DELETE_FAILED, false, false},
		{"unknown", false, false},
		{api.LLM_STATUS_START_FAIL, true, true},
		{api.LLM_STATUS_CREATE_FAIL, true, true},
		{api.LLM_DEPLOYMENT_STATUS_DEPLOYING, true, true},
		{api.LLM_STATUS_RESTART_FAILED, true, true},
		{api.LLM_STATUS_UNKNOWN, true, true},
		{api.STATUS_READY, true, true},
		{api.LLM_STATUS_DELETING, true, false},
		{api.LLM_STATUS_START_DELETE, true, false},
		{api.LLM_STATUS_DELETED, true, false},
		{api.LLM_STATUS_DELETE_FAILED, true, false},
		{"creating", true, false},
		{api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL, true, false},
		{api.LLM_DEPLOYMENT_STATUS_CREATING_SKU, true, false},
		{api.LLM_DEPLOYMENT_STATUS_SYNCING, true, false},
		{api.LLM_DEPLOYMENT_STATUS_RESTARTING, false, false},
		{api.LLM_DEPLOYMENT_STATUS_RESTARTING, true, false},
	}
	for _, tc := range cases {
		got := canRestartDeploymentStatus(tc.status, tc.force)
		if got != tc.want {
			t.Errorf("canRestartDeploymentStatus(%q, force=%v) = %v, want %v", tc.status, tc.force, got, tc.want)
		}
	}
}

func TestCanUpdateReplicaHealthStatus(t *testing.T) {
	cases := []struct {
		current string
		desired string
		want    bool
	}{
		{api.STATUS_READY, api.LLM_DEPLOYMENT_STATUS_PARTIAL, true},
		{api.LLM_DEPLOYMENT_STATUS_PARTIAL, api.STATUS_READY, true},
		{api.LLM_DEPLOYMENT_STATUS_DEPLOYING, api.STATUS_READY, true},
		{api.LLM_DEPLOYMENT_STATUS_RESTARTING, api.STATUS_READY, false},
		{api.LLM_DEPLOYMENT_STATUS_RESTARTING, api.LLM_DEPLOYMENT_STATUS_PARTIAL, false},
		{api.LLM_DEPLOYMENT_STATUS_RESTARTING, api.LLM_DEPLOYMENT_STATUS_DEPLOYING, false},
		{api.LLM_DEPLOYMENT_STATUS_RESTARTING, api.LLM_STATUS_START_FAIL, false},
		{api.LLM_STATUS_CREATE_FAIL, api.STATUS_READY, true},
		{api.LLM_STATUS_CREATE_FAIL, api.LLM_DEPLOYMENT_STATUS_DEPLOYING, false},
		{api.LLM_STATUS_START_FAIL, api.LLM_DEPLOYMENT_STATUS_PARTIAL, true},
		{api.LLM_STATUS_START_FAIL, api.LLM_DEPLOYMENT_STATUS_DEPLOYING, false},
		{api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL, api.STATUS_READY, false},
		{api.LLM_STATUS_DELETING, api.STATUS_READY, false},
	}
	for _, tc := range cases {
		got := canUpdateReplicaHealthStatus(tc.current, tc.desired)
		if got != tc.want {
			t.Errorf("canUpdateReplicaHealthStatus(%q, %q) = %v, want %v", tc.current, tc.desired, got, tc.want)
		}
	}
}
