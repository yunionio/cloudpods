package sysutils

import (
	"testing"
)

func TestIsHypervisor(t *testing.T) {
	if IsHypervisor() {
		t.Logf("Running in a hypervisor")
	} else {
		t.Logf("Running in a baremetal")
	}
}
