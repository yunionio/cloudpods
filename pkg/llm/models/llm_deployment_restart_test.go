package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func TestCanRestartDeploymentStatus(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{api.STATUS_READY, true},
		{api.LLM_DEPLOYMENT_STATUS_PARTIAL, true},
		{api.LLM_STATUS_RUNNING, true},
		{api.LLM_DEPLOYMENT_STATUS_DEPLOYING, false},
		{api.LLM_STATUS_DELETING, false},
		{api.LLM_DEPLOYMENT_STATUS_IMPORTING_MODEL, false},
		{api.LLM_DEPLOYMENT_STATUS_CREATING_SKU, false},
		{api.LLM_STATUS_CREATE_FAIL, false},
		{api.LLM_STATUS_DELETE_FAILED, false},
		{"unknown", false},
	}
	for _, tc := range cases {
		got := canRestartDeploymentStatus(tc.status)
		if got != tc.want {
			t.Errorf("canRestartDeploymentStatus(%q) = %v, want %v", tc.status, got, tc.want)
		}
	}
}
