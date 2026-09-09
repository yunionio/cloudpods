package models

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func TestContainerWatchStatusChanged(t *testing.T) {
	probing := jsonutils.NewDict()
	probing.Set("status", jsonutils.NewString("probing"))
	probing.Set("guest_id", jsonutils.NewString("srv-1"))

	running := jsonutils.NewDict()
	running.Set("status", jsonutils.NewString("running"))
	running.Set("guest_id", jsonutils.NewString("srv-1"))

	if !containerWatchStatusChanged(nil, probing) {
		t.Fatal("add with status should trigger")
	}
	if containerWatchStatusChanged(probing, probing) {
		t.Fatal("same status should not trigger")
	}
	if !containerWatchStatusChanged(probing, running) {
		t.Fatal("probing -> running should trigger")
	}
	if containerWatchStatusChanged(running, nil) {
		t.Fatal("nil new object should not trigger")
	}
	empty := jsonutils.NewDict()
	if containerWatchStatusChanged(probing, empty) {
		t.Fatal("missing status should not trigger")
	}
}
