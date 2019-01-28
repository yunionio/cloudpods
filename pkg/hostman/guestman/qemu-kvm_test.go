package guestman

import (
	"testing"
)

func TestSKVMGuestInstance_getPid(t *testing.T) {
	manager := NewGuestManager(nil, "/opt/cloud/workspace/servers")
	s := NewKVMGuestInstance("05b787e9-b78e-4ebc-8128-04f55d37306f", manager)
	t.Logf("Guest is ->> %d", s.GetPid())
}
