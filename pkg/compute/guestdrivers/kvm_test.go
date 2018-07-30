package guestdrivers

import "testing"

func TestSKVMGuestDriver_GetGuestVncInfo(t *testing.T) {
	results := `Server:
address: 0.0.0.0:5901
auth: none
Client: none`
	t.Logf("port=%d", findVNCPort(results))
}
