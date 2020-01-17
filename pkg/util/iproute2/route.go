package iproute2

import (
	"net"

	"github.com/vishvananda/netlink"

	"yunion.io/x/pkg/errors"
)

const (
	errBadIP = errors.Error("bad ip")
)

type Route struct {
	*Link
}

func NewRoute(ifname string) *Route {
	l := NewLink(ifname)
	route := &Route{
		Link: l,
	}
	return route
}

func (route *Route) link() (link netlink.Link, ok bool) {
	link = route.Link.link
	if link != nil {
		ok = true
	}
	return
}

func (route *Route) List4() ([]netlink.Route, error) {
	link, ok := route.link()
	if !ok {
		return nil, route.Err()
	}

	rs, err := netlink.RouteList(link, netlink.FAMILY_V4)
	if err != nil {
		route.addErr(err, "route list")
		return nil, route.Err()
	}
	for i := range rs {
		if rs[i].Dst == nil {
			// make the return value easier to work with
			rs[i].Dst = &net.IPNet{
				IP:   net.IPv4zero,
				Mask: net.IPMask(net.IPv4zero),
			}
		}
	}
	return rs, nil
}

func (route *Route) AddByIPNet(ipnet *net.IPNet, gw net.IP) *Route {
	link, ok := route.link()
	if !ok {
		return route
	}

	r := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       ipnet,
	}
	if len(gw) > 0 {
		r.Gw = gw
	}
	if err := netlink.RouteReplace(r); err != nil {
		route.addErr(err, "RouteReplace %s", r.String())
	}
	return route
}

func (route *Route) AddByCidr(cidr string, gwStr string) *Route {
	var (
		dst *net.IPNet
		gw  net.IP
		err error
	)
	if _, dst, err = net.ParseCIDR(cidr); err != nil {
		route.addErr(err, "parse cidr")
		return route
	}

	if gwStr != "" {
		gw = net.ParseIP(gwStr)
		if len(gw) == 0 {
			route.addErr(errBadIP, "gwStr: %s", gwStr)
			return route
		}
	}

	return route.AddByIPNet(dst, gw)
}

func (route *Route) Add(netStr, maskStr, gwStr string) *Route {
	var (
		ip   net.IP
		mask net.IPMask
		gw   net.IP
	)

	if ip = net.ParseIP(netStr); len(ip) == 0 {
		route.addErr(errBadIP, "netStr %s", netStr)
		return route
	}
	if maskIp := net.ParseIP(maskStr); len(maskIp) == 0 {
		route.addErr(errBadIP, "maskStr %s", maskStr)
		return route
	} else {
		if ip := maskIp.To4(); len(ip) > 0 {
			maskIp = ip
		}
		mask = net.IPMask(maskIp)
		ones, bits := mask.Size()
		if ones == 0 && bits == 0 {
			route.addErr(errBadIP, "bad mask %s", maskStr)
			return route
		}
	}
	if gwStr != "" {
		if gw = net.ParseIP(gwStr); len(gw) == 0 {
			route.addErr(errBadIP, "gwStr %s", gwStr)
			return route
		}
	}

	ipnet := &net.IPNet{
		IP:   ip,
		Mask: mask,
	}
	return route.AddByIPNet(ipnet, gw)
}

func RouteGetByDst(dstStr string) ([]netlink.Route, error) {
	dstIp := net.ParseIP(dstStr)
	routes, err := netlink.RouteGet(dstIp)
	return routes, err
}
