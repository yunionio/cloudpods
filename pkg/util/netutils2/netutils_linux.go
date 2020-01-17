package netutils2

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/iproute2"
)

func (n *SNetInterface) GetAddresses() [][]string {
	ipnets, err := iproute2.NewAddress(n.name).List4()
	if err != nil {
		log.Errorf("list address %s: %v", n.name, err)
		return nil
	}
	r := make([][]string, len(ipnets))
	for i, ipnet := range ipnets {
		ip := ipnet.IP
		masklen, _ := ipnet.Mask.Size()
		r[i] = []string{
			ip.String(),
			fmt.Sprintf("%d", masklen),
		}
	}
	return r
}

func (n *SNetInterface) GetRoutes(gwOnly bool) [][]string {
	rs, err := iproute2.NewRoute(n.name).List4()
	if err != nil {
		return nil
	}

	res := [][]string{}
	for i := range rs {
		r := &rs[i]
		ok := true
		if masklen, _ := r.Dst.Mask.Size(); gwOnly && masklen != 0 {
			ok = false
		}
		if ok {
			gwStr := ""
			if len(r.Gw) > 0 {
				gwStr = r.Gw.String()
			}
			res = append(res, []string{
				r.Dst.IP.String(),
				gwStr,
				net.IP(r.Dst.Mask).String(),
			})
		}
	}
	return res
}

func DefaultSrcIpDev() (srcIp net.IP, ifname string, err error) {
	routes, err := iproute2.RouteGetByDst("114.114.114.114")
	if err != nil {
		err = errors.Wrap(err, "get route")
		return
	}
	if len(routes) == 0 {
		err = fmt.Errorf("no route")
		return
	}
	var errs []error
	for i := range routes {
		route := &routes[i]
		if len(route.Src) == 0 {
			continue
		}
		ip4 := route.Src.To4()
		if len(ip4) != 4 || ip4.Equal(net.IPv4zero) {
			errs = append(errs, fmt.Errorf("bad src ipv4 address: %s", ip4))
			continue
		}
		link, err2 := netlink.LinkByIndex(route.LinkIndex)
		if err2 != nil {
			errs = append(errs, errors.Wrap(err2, "link by index"))
			continue
		}
		srcIp = ip4
		ifname = link.Attrs().Name
		return
	}
	err = errors.NewAggregate(errs)
	return
}
