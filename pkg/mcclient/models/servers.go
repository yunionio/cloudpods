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

package models

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
)

type Server struct {
	VirtualResource

	VcpuCount int
	VmemSize  int

	BootOrder string

	DisableDelete    tristate.TriState
	ShutdownBehavior string

	KeypairId string

	HostId       string
	BackupHostId string

	Vga     string
	Vdi     string
	Machine string
	Bios    string
	OsType  string

	FlavorId string

	SecgrpId      string
	AdminSecgrpId string

	Hypervisor string

	InstanceType string

	// Derived attributes
	Networks string
	Eip      string
	Nics     ServerNics
}

type ServerNics []ServerNic
type ServerNic struct {
	IpAddr    string `json:"ip_addr"`
	Mac       string `json:"mac"`
	NetworkId string `json:"network_id"`
	VpcId     string `json:"VpcId"`
}

type ServerNetworks []ServerNetwork
type ServerNetwork struct {
	Index   int
	Ip      net.IP
	IpMask  int
	MacAddr net.HardwareAddr
	VlanId  int

	Name   string
	Driver string
	Bw     int
}

func (sns ServerNetworks) GetPrivateIPs() []net.IP {
	ips := []net.IP{}
	for _, sn := range sns {
		ipStr := sn.Ip.String()
		ipAddr, err := netutils.NewIPV4Addr(ipStr)
		if err != nil {
			continue
		}
		if netutils.IsPrivate(ipAddr) {
			ips = append(ips, sn.Ip)
		}
	}
	return ips
}

func ParseServerNetworkDetailedString(s string) (ServerNetworks, error) {
	sns := ServerNetworks{}
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "eth") {
			return nil, errors.New("should be prefixed with 'eth'")
		}
		parts := strings.Split(line, "/")
		if len(parts) < 7 {
			return nil, errors.New("less than 7 parts separated by '/'")
		}
		sn := ServerNetwork{
			Name:   parts[4],
			Driver: parts[5],
		}
		{ // index, ipaddr
			idip := parts[0][3:]
			i := strings.IndexByte(idip, ':')
			if i <= 0 {
				return nil, errors.New("no sep ':'")
			}
			idStr, ipStr := idip[:i], idip[i+1:]
			{
				id, err := strconv.ParseUint(idStr, 10, 8)
				if err != nil {
					return nil, errors.WithMessagef(err, "bad index %s", idStr)
				}
				sn.Index = int(id)
			}
			{
				ip := net.ParseIP(ipStr)
				if ip == nil {
					return nil, fmt.Errorf("bad ip %s", ipStr)
				}
				ip = ip.To4()
				if ip == nil {
					return nil, fmt.Errorf("not ipv4 addr: %s", ipStr)
				}
				sn.Ip = ip
			}
		}
		{ // ip mask
			masklenStr := parts[1]
			masklen, err := strconv.ParseUint(masklenStr, 10, 8)
			if err != nil {
				return nil, errors.WithMessagef(err, "bad network mask len: %s", masklenStr)
			}
			sn.IpMask = int(masklen)
		}
		{ // macaddr
			macaddrStr := parts[2]
			macaddr, err := net.ParseMAC(macaddrStr)
			if err != nil {
				return nil, errors.WithMessagef(err, "bad macaddr: %s", macaddrStr)
			}
			sn.MacAddr = macaddr
		}
		{ // vlan
			vlanStr := parts[3]
			vlan, err := strconv.ParseUint(vlanStr, 10, 16)
			if err != nil {
				return nil, errors.WithMessagef(err, "bad vlan id: %s", vlanStr)
			}
			sn.VlanId = int(vlan)
		}
		{ // bw
			bwStr := parts[6]
			bw, err := strconv.ParseUint(bwStr, 10, 32)
			if err != nil {
				return nil, errors.WithMessagef(err, "bad bw: %s", bwStr)
			}
			sn.Bw = int(bw)
		}
		sns = append(sns, sn)
	}
	return sns, nil
}
