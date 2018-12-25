package netutils2

import (
	"testing"
)

func TestNewNetInterface(t *testing.T) {
	n := NewNetInterface("br0")
	t.Logf("ip: %s, mac: %s, mask: %s", n.addr, n.mac, n.mask)
}
