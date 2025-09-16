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

package dhcp

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"
)

const (
	PXECLIENT = "PXEClient"

	OptClasslessRouteLin OptionCode = OptionClasslessRouteFormat //Classless Static Route Option
	OptClasslessRouteWin OptionCode = 249
)

// http://www.networksorcery.com/enp/rfc/rfc2132.txt
type ResponseConfig struct {
	OsName        string
	ServerIP      net.IP // OptServerIdentifier 54
	ClientIP      net.IP
	Gateway       net.IP        // OptRouters 3
	Domain        string        // OptDomainName 15
	LeaseTime     time.Duration // OptLeaseTime 51
	RenewalTime   time.Duration // OptRenewalTime 58
	BroadcastAddr net.IP        // OptBroadcastAddr 28
	Hostname      string        // OptHostname 12
	SubnetMask    net.IP        // OptSubnetMask 1
	DNSServers    []net.IP      // OptDNSServers
	Routes        [][]string    // TODO: 249 for windows, 121 for linux
	NTPServers    []net.IP      // OptNTPServers 42
	MTU           uint16        // OptMTU 26

	// Relay Info https://datatracker.ietf.org/doc/html/rfc3046
	RelayInfo []byte

	// TFTP config
	BootServer string
	BootFile   string
	BootBlock  uint16
}

func (conf ResponseConfig) GetHostname() string {
	return conf.Hostname
}

func GetOptUint16(val uint16) []byte {
	opts := []byte{0, 0}
	binary.BigEndian.PutUint16(opts, val)
	return opts
}

func GetOptIP(ip net.IP) []byte {
	return []byte(ip.To4())
}

func GetOptIPs(ips []net.IP) []byte {
	buf := make([]byte, 0)
	for _, ip := range ips {
		buf = append(buf, []byte(ip.To4())...)
	}
	return buf
}

func GetOptTime(d time.Duration) []byte {
	timeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(timeBytes, uint32(d/time.Second))
	return timeBytes
}

func getClasslessRoutePack(route []string) []byte {
	var snet, gw = route[0], route[1]
	tmp := strings.Split(snet, "/")
	netaddr := net.ParseIP(tmp[0])
	if netaddr != nil {
		netaddr = netaddr.To4()
	}
	masklen, _ := strconv.Atoi(tmp[1])
	netlen := masklen / 8
	if masklen%8 > 0 {
		netlen += 1
	}
	if netlen < 4 {
		netaddr = netaddr[0:netlen]
	}
	gwaddr := net.ParseIP(gw)
	if gwaddr != nil {
		gwaddr = gwaddr.To4()
	}

	res := []byte{byte(masklen)}
	res = append(res, []byte(netaddr)...)
	return append(res, []byte(gwaddr)...)
}

func MakeReplyPacket(pkt Packet, conf *ResponseConfig) (Packet, error) {
	msgType := Offer
	if pkt.Type() == Request {
		reqAddr, _ := pkt.ParseOptions().IP(OptionRequestedIPAddress)
		if reqAddr != nil && !conf.ClientIP.Equal(reqAddr) {
			msgType = NAK
		} else {
			msgType = ACK
		}
	}
	return makeDHCPReplyPacket(pkt, conf, msgType), nil
}

func getPacketVendorClassId(pkt Packet) string {
	bs := pkt.ParseOptions()[OptionVendorClassIdentifier]
	vendorClsId := string(bs)
	return vendorClsId
}

func makeDHCPReplyPacket(req Packet, conf *ResponseConfig, msgType MessageType) Packet {
	if conf.OsName == "" {
		if vendorClsId := getPacketVendorClassId(req); vendorClsId != "" && strings.HasPrefix(vendorClsId, "MSFT ") {
			conf.OsName = "win"
		}
	}

	opts := make([]Option, 0)

	if conf.SubnetMask != nil {
		opts = append(opts, Option{OptionSubnetMask, GetOptIP(conf.SubnetMask)})
	}
	if conf.Gateway != nil {
		opts = append(opts, Option{OptionRouter, GetOptIP(conf.Gateway)})
	}
	if conf.Domain != "" {
		opts = append(opts, Option{OptionDomainName, []byte(conf.Domain)})
	}
	if conf.BroadcastAddr != nil {
		opts = append(opts, Option{OptionBroadcastAddress, GetOptIP(conf.BroadcastAddr)})
	}
	if conf.Hostname != "" {
		opts = append(opts, Option{OptionHostName, []byte(conf.GetHostname())})
	}
	if len(conf.DNSServers) > 0 {
		opts = append(opts, Option{OptionDomainNameServer, GetOptIPs(conf.DNSServers)})
	}
	if len(conf.NTPServers) > 0 {
		opts = append(opts, Option{OptionNetworkTimeProtocolServers, GetOptIPs(conf.NTPServers)})
	}
	if conf.MTU > 0 {
		opts = append(opts, Option{OptionInterfaceMTU, GetOptUint16(conf.MTU)})
	}
	if conf.RelayInfo != nil {
		opts = append(opts, Option{OptionRelayAgentInformation, conf.RelayInfo})
	}
	resp := ReplyPacket(req, msgType, conf.ServerIP, conf.ClientIP, conf.LeaseTime, opts)
	if conf.BootServer != "" {
		//resp.Options[OptOverload] = []byte{3}
		resp.SetSIAddr(net.ParseIP(conf.BootServer))
		resp.AddOption(OptionTFTPServerName, []byte(fmt.Sprintf("%s\x00", conf.BootServer)))
	}
	if conf.BootFile != "" {
		resp.AddOption(OptionBootFileName, []byte(fmt.Sprintf("%s\x00", conf.BootFile)))
		sz := make([]byte, 2)
		binary.BigEndian.PutUint16(sz, conf.BootBlock)
		resp.AddOption(OptionBootFileSize, sz)
	}
	//if bs, _ := req.ParseOptions().Bytes(OptionClientMachineIdentifier); bs != nil {
	//resp.AddOption(OptionClientMachineIdentifier, bs)
	//}
	if conf.RenewalTime > 0 {
		resp.AddOption(OptionRenewalTimeValue, GetOptTime(conf.RenewalTime))
	}
	if conf.Routes != nil {
		var optCode = OptClasslessRouteLin
		if strings.HasPrefix(strings.ToLower(conf.OsName), "win") {
			optCode = OptClasslessRouteWin
		}
		for _, route := range conf.Routes {
			routeBytes := getClasslessRoutePack(route)
			resp.AddOption(optCode, routeBytes)
		}
	}
	return resp
}

func IsPXERequest(pkt Packet) bool {
	//if pkt.Type != MsgDiscover {
	//log.Warningf("packet is %s, not %s", pkt.Type, MsgDiscover)
	//return false
	//}

	if pkt.GetOptionValue(OptionClientArchitecture) == nil {
		log.Debugf("%s not a PXE boot request (missing option 93)", pkt.CHAddr().String())
		return false
	}
	return true
}
