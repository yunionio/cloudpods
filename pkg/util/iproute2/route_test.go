package iproute2

import (
	"testing"

	"github.com/vishvananda/netlink"
)

func TestRoute(t *testing.T) {
	t.Run("ensure non-nil route.dst", func(t *testing.T) {
		dstStr := "114.114.114.114"
		routes, err := RouteGetByDst(dstStr)
		if err != nil {
			t.Skipf("route get: %v", err)
		}
		if len(routes) == 0 {
			t.Skipf("no route to %s", dstStr)
		}
		for _, r := range routes {
			if r.LinkIndex <= 0 {
				continue
			}
			nllink, err := netlink.LinkByIndex(r.LinkIndex)
			if err != nil {
				t.Errorf("link by index: %v", err)
				continue
			}
			routes2, err := NewRoute(nllink.Attrs().Name).List4()
			for _, r2 := range routes2 {
				if r2.Dst.String() == "0.0.0.0/0" {
					t.Logf("yeah, dst of default route is not nil: %s", r2)
				}
			}
		}

	})
}
