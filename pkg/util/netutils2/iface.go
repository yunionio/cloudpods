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
	"net"
	"time"

	"yunion.io/x/pkg/errors"
)

func getIfaceIPs(iface *net.Interface) ([]net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, errors.Wrap(err, "iface.Addrs")
	}
	ips := make([]net.IP, 0)
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP)
			} else if ipnet.IP.To16() != nil {
				ips = append(ips, ipnet.IP)
			}
		}
	}
	return ips, nil
}

func WaitIfaceIps(ifname string) (*net.Interface, []net.IP, error) {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "net.InterfaceByName %s", ifname)
	}
	var ips []net.IP
	MAX := 60
	wait := 0
	for wait < MAX {
		ips, err = getIfaceIPs(iface)
		if err != nil {
			return nil, nil, errors.Wrap(err, "getIfaceIPs")
		}
		if len(ips) == 0 {
			time.Sleep(2 * time.Second)
			wait += 2
		} else {
			break
		}
	}
	return iface, ips, nil
}
