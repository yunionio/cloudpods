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
	DNSServer     net.IP        // OptDNSServers
	Routes        [][]string    // TODO: 249 for windows, 121 for linux

	// TFTP config
	BootServer string
	BootFile   string
	BootBlock  uint16
}

func (conf ResponseConfig) GetHostname() string {
	hostname := conf.Hostname
	if conf.Domain != "" {
		hostname = fmt.Sprintf("%s.%s", hostname, conf.Domain)
	}
	return hostname
}

func GetOptIP(ip net.IP) []byte {
	return []byte(ip.To4())
}

func GetOptTime(d time.Duration) []byte {
	timeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(timeBytes, uint32(d/time.Second))
	return timeBytes
}

func GetClasslessRoutePack(route []string) []byte {
	var snet, gw = route[0], route[1]
	tmp := strings.Split(snet, "/")
	netaddr := net.ParseIP(tmp[0])
	masklen, _ := strconv.Atoi(tmp[1])
	netlen := masklen / 8
	if masklen%8 > 0 {
		netlen += 1
	}
	if netlen < 4 {
		netaddr = netaddr[0:netlen]
	}
	gwaddr := net.ParseIP(gw)

	res := []byte{byte(masklen)}
	res = append(res, []byte(netaddr.To4())...)
	return append(res, []byte(gwaddr.To4())...)
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
	if conf.DNSServer != nil {
		opts = append(opts, Option{OptionDomainNameServer, GetOptIP(conf.DNSServer)})
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
			routeBytes := GetClasslessRoutePack(route)
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
		log.Debugf("not a PXE boot request (missing option 93)")
		return false
	}
	return true
}
