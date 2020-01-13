package iproute2

import (
	"fmt"
	"testing"

	"github.com/vishvananda/netlink"
)

func genDummyName(t *testing.T) string {
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("dummy%d", i)
		if _, err := netlink.LinkByName(name); err != nil {
			return name
		}
	}
	t.Fatalf("can't even find a dummy name")
	return ""
}

func addDummy(t *testing.T, name string) *netlink.Dummy {
	attrs := netlink.NewLinkAttrs()
	attrs.Name = name
	dum := &netlink.Dummy{
		LinkAttrs: attrs,
	}
	if err := netlink.LinkAdd(dum); err != nil {
		t.Skipf("add %s: %v", name, err)
	}
	return dum
}

func delDummy(t *testing.T, dum *netlink.Dummy) {
	if err := netlink.LinkDel(dum); err != nil {
		t.Errorf("del %s: %v", dum.Name, err)
	}
}

func TestLink(t *testing.T) {
	ifname := genDummyName(t)
	dum := addDummy(t, ifname)
	defer delDummy(t, dum)

	l := NewLink(ifname)
	l.Up().MTU(100).Address("00:11:22:33:44:55")
	if err := l.Err(); err != nil {
		t.Fatalf("got error: %v", err)
	}

	l.Down().MTU(120).Address("00:11:22:33:44:88")
	if err := l.Err(); err != nil {
		t.Fatalf("got error: %v", err)
	}
}
