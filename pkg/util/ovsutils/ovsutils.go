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

package ovsutils

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

func GetDbPorts(brname string) []string {
	output, err := procutils.NewCommand("ovs-vsctl", "list-ifaces", brname).Output()
	if err != nil {
		log.Errorln(err)
		return nil
	}
	ifaces := make([]string, 0)
	re := regexp.MustCompile(`^[a-zA-Z0-9._@-]+$`)
	for _, line := range strings.Split(string(output), "\n") {
		ifname := strings.TrimSpace(line)
		if len(ifname) > 0 && re.MatchString(ifname) {
			ifaces = append(ifaces, ifname)
		}
	}
	return ifaces
}

func GetDpPorts(brname string) []string {
	output, err := procutils.NewCommand("ovs-dpctl", "show").Output()
	if err != nil {
		log.Errorln(err)
		return nil
	}

	ifaces := make([]string, 0)
	re := regexp.MustCompile(`port \d+: (?P<name>[a-zA-Z0-9._@-]+)`)
	for _, line := range strings.Split(string(output), "\n") {
		m := regutils2.GetParams(re, line)
		if len(m) > 0 {
			ifnmae := m["name"]
			if ifnmae != brname {
				ifaces = append(ifaces, ifnmae)
			}
		}
	}
	return ifaces
}

func GetBridges() []string {
	output, err := procutils.NewCommand("ovs-vsctl", "list-br").Output()
	if err != nil {
		log.Errorln(err)
		return nil
	}

	brs := make([]string, 0)
	for _, line := range strings.Split(string(output), "\n") {
		brname := strings.TrimSpace(line)
		if len(brname) > 0 {
			brs = append(brs, brname)
		}
	}
	return brs
}

func RemovePortFromBridge(brname, port string) {
	log.Infof("remove_port_from_bridge %s %s", brname, port)
	if err := procutils.NewCommand("ovs-vsctl", "del-port", brname, port).Run(); err != nil {
		log.Errorln(err)
	}
}

func CleanHiddenPorts(brname string) {
	// BUG patch ports are not included in dpPorts
	dbPorts := GetDbPorts(brname)
	dpPorts := GetDpPorts(brname)
	for _, p := range dbPorts {
		if !utils.IsInStringArray(p, dpPorts) {
			RemovePortFromBridge(brname, p)
		}
	}
}

func CleanAllHiddenPorts() {
	brs := GetBridges()
	for _, br := range brs {
		CleanHiddenPorts(br)
	}
}

// WHY normalize ovs-db host?
// if ovn-db url is given by domain name, ovn-nbctl will always try to resolve in IPv6 first
// if host is IPv4 only, ovn-nbctl will report error "connect: Address family not supported by protocol"
func NormalizeDbHost(db string) (string, error) {
	if strings.HasPrefix(db, "tcp:") {
		host, port, err := net.SplitHostPort(db[4:])
		if err != nil {
			return "", errors.Wrapf(err, "split host port: %s", db)
		}
		if ip := net.ParseIP(host); len(ip) == 0 {
			addrs, err := net.LookupHost(host)
			if err != nil {
				return "", errors.Wrapf(err, "dns lookup (%s) failed", host)
			}
			if len(addrs) == 0 {
				return "", fmt.Errorf("dns lookup (%s) returned empty result", host)
			}
			return "tcp:" + addrs[0] + ":" + port, nil
		}
	}
	return db, nil
}
