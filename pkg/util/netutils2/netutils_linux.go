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

package netutils2

import (
	"fmt"
	"net"
	"strings"

	"github.com/vishvananda/netlink"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/iproute2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func (n *SNetInterface) GetAddresses() [][]string {
	addrList := iproute2.NewAddress(n.name)
	addrs4 := n.getAddresses(addrList.List4)
	addrs6 := n.getAddresses(addrList.List6)
	if len(addrs6) > 0 {
		addrs4 = append(addrs4, addrs6...)
	}
	return addrs4
}

func (n *SNetInterface) getAddresses(listFunc func() ([]net.IPNet, error)) [][]string {
	ipnets, err := listFunc()
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

func (n *SNetInterface) GetRouteSpecs() []iproute2.RouteSpec {
	routeList := iproute2.NewRoute(n.name)
	routes4 := getRouteSpecs(routeList.List4)
	routes6 := getRouteSpecs(routeList.List6)
	if len(routes6) > 0 {
		routes4 = append(routes4, routes6...)
	}
	return routes4
}

func (n *SNetInterface) ClearAddrs() error {
	cmd := procutils.NewCommand("ip", "addr", "flush", "dev", n.name)
	msg, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, strings.TrimSpace(string(msg)))
	}
	return nil
}

func getRouteSpecs(listFunc func() ([]iproute2.RouteSpec, error)) []iproute2.RouteSpec {
	routespecs, err := listFunc()
	if err != nil {
		return nil
	}
	return routespecs
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
