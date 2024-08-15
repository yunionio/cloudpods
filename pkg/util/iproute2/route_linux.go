// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package iproute2

import (
	"net"

	"github.com/vishvananda/netlink"

	"yunion.io/x/pkg/errors"
)

const (
	errBadIP = errors.Error("bad ip")
)

type RouteSpec = netlink.Route

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

func (route *Route) List4() ([]RouteSpec, error) {
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

func (route *Route) List6() ([]RouteSpec, error) {
	link, ok := route.link()
	if !ok {
		return nil, route.Err()
	}

	rs, err := netlink.RouteList(link, netlink.FAMILY_V6)
	if err != nil {
		route.addErr(err, "route list")
		return nil, route.Err()
	}
	for i := range rs {
		if rs[i].Dst == nil {
			// make the return value easier to work with
			rs[i].Dst = &net.IPNet{
				IP:   net.IPv6zero,
				Mask: net.IPMask(net.IPv6zero),
			}
		}
	}
	return rs, nil
}

func (route *Route) AddByIPNet(ipnet *net.IPNet, gw net.IP) *Route {
	r := RouteSpec{
		Dst: ipnet,
	}
	if len(gw) > 0 {
		r.Gw = gw
	}
	return route.AddByRouteSpec(r)
}

func (route *Route) AddByCidr(cidr string, gwStr string) *Route {
	dst, gw, err := route.parseCidr(cidr, gwStr)
	if err != nil {
		route.addErr2(err)
		return route
	}

	return route.AddByIPNet(dst, gw)
}

func (route *Route) AddByRouteSpec(r RouteSpec) *Route {
	link, ok := route.link()
	if !ok {
		return route
	}

	r.LinkIndex = link.Attrs().Index
	if err := netlink.RouteReplace(&r); err != nil {
		route.addErr(err, "RouteReplace %s", r.String())
	}
	return route
}

func (route *Route) parseCidr(cidr, gwStr string) (dst *net.IPNet, gw net.IP, err error) {
	if _, dst, err = net.ParseCIDR(cidr); err != nil {
		err = errors.Wrap(err, "parse cidr")
		return
	}

	if gwStr != "" {
		gw = net.ParseIP(gwStr)
		if len(gw) == 0 {
			err = errors.Wrapf(errBadIP, "gwStr: %s", gwStr)
			return
		}
	}
	return
}

func (route *Route) parse(netStr, maskStr, gwStr string) (ip net.IP, mask net.IPMask, gw net.IP, err error) {
	if ip = net.ParseIP(netStr); len(ip) == 0 {
		err = errors.Wrapf(errBadIP, "netStr %s", netStr)
		return
	}
	if maskIp := net.ParseIP(maskStr); len(maskIp) == 0 {
		err = errors.Wrapf(errBadIP, "maskStr %s", maskStr)
		return
	} else {
		if ip := maskIp.To4(); len(ip) > 0 {
			maskIp = ip
		}
		mask = net.IPMask(maskIp)
		ones, bits := mask.Size()
		if ones == 0 && bits == 0 {
			err = errors.Wrapf(errBadIP, "bad mask %s", maskStr)
			return
		}
	}
	if gwStr != "" {
		if gw = net.ParseIP(gwStr); len(gw) == 0 {
			err = errors.Wrapf(errBadIP, "gwStr %s", gwStr)
			return
		}
	}
	return
}

func (route *Route) Add(netStr, maskStr, gwStr string) *Route {
	var (
		ip   net.IP
		mask net.IPMask
		gw   net.IP
	)

	ip, mask, gw, err := route.parse(netStr, maskStr, gwStr)
	if err != nil {
		route.addErr2(err)
	}

	ipnet := &net.IPNet{
		IP:   ip,
		Mask: mask,
	}
	return route.AddByIPNet(ipnet, gw)
}

func (route *Route) Del(netStr, maskStr string) *Route {
	ip, mask, _, err := route.parse(netStr, maskStr, "")
	if err != nil {
		route.addErr2(err)
		return route
	}
	ipnet := &net.IPNet{
		IP:   ip,
		Mask: mask,
	}
	return route.DelByIPNet(ipnet)
}

func (route *Route) DelByCidr(cidr string) *Route {
	dst, _, err := route.parseCidr(cidr, "")
	if err != nil {
		route.addErr2(err)
		return route
	}

	return route.DelByIPNet(dst)
}

func (route *Route) DelByIPNet(ipnet *net.IPNet) *Route {
	link, ok := route.link()
	if !ok {
		return route
	}

	r := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       ipnet,
	}
	if err := netlink.RouteDel(r); err != nil {
		route.addErr(err, "RouteDel %s", r)
		return route
	}
	return route
}

func RouteGetByDst(dstStr string) ([]RouteSpec, error) {
	dstIp := net.ParseIP(dstStr)
	routes, err := netlink.RouteGet(dstIp)
	return routes, err
}
