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

// Copyright 2019 Yunion
// Copyright 2016 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package dhcp

import (
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/mdlayher/packet"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"

	"yunion.io/x/pkg/errors"
)

func newRawSocketConn6(iface string, filter []bpf.RawInstruction, serverPort uint16) (conn, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, errors.Wrap(err, "interface by name")
	}

	ip, err := interfaceToIPv6Addr(ifi)
	if err != nil {
		return nil, err
	}

	// unix.ETH_P_ALL
	conn, err := packet.Listen(ifi, packet.Raw, unix.ETH_P_IPV6, &packet.Config{
		Filter: filter,
	})
	if err != nil {
		return nil, errors.Wrap(err, "packet.Listen")
	}
	return &rawSocketConn{
		conn:       conn,
		iface:      ifi,
		ip:         ip,
		serverPort: serverPort,
	}, nil
}

func (s *rawSocketConn) Recv6(b []byte) ([]byte, *net.UDPAddr, net.HardwareAddr, int, error) {
	// read packet
	n, addr, err := s.conn.ReadFrom(b)
	if err != nil {
		return nil, nil, nil, 0, errors.Wrap(err, "Read from errror")
	}
	b = b[:n]

	srcMac, err := net.ParseMAC(addr.String())
	if err != nil {
		return nil, nil, nil, 0, errors.Wrap(err, "Parse mac error")
	}

	p := gopacket.NewPacket(b, layers.LayerTypeEthernet, gopacket.Default)
	if p.ErrorLayer() != nil {
		return nil, nil, nil, 0, errors.Wrap(p.ErrorLayer().Error(), "Failed to decode packet")
	}

	var srcIp net.IP
	{
		ipLayer := p.Layer(layers.LayerTypeIPv6)
		if ipLayer != nil {
			// ipv6
			ip6 := ipLayer.(*layers.IPv6)
			srcIp = ip6.SrcIP
		} else {
			return nil, nil, nil, 0, errors.Wrap(p.ErrorLayer().Error(), "Expect IPv6 packet")
		}
	}

	icmpLayer := p.Layer(layers.LayerTypeICMPv6)
	if icmpLayer != nil {
		raLayer := p.Layer(layers.LayerTypeICMPv6RouterSolicitation)
		if raLayer != nil {
			// icmp6, ra solitation
			raPkt := raLayer.(*layers.ICMPv6RouterSolicitation)
			sbf := gopacket.NewSerializeBuffer()
			if err := raPkt.SerializeTo(sbf, gopacket.SerializeOptions{}); err != nil {
				return nil, nil, nil, 0, errors.Wrap(err, "Serialize ICMPv6 ra solitation packet error")
			}
			return sbf.Bytes(), &net.UDPAddr{IP: srcIp, Port: icmpRAFakePort}, srcMac, 0, nil
		} else {
			return nil, nil, nil, 0, errors.Wrap(p.ErrorLayer().Error(), "expect an ICMPv6 RA solitation packet")
		}
	}

	var srcPort uint16
	udpLayer := p.Layer(layers.LayerTypeUDP)
	if udpLayer != nil {
		udpInfo := udpLayer.(*layers.UDP)
		srcPort = uint16(udpInfo.SrcPort)
	} else {
		return nil, nil, nil, 0, errors.Wrap(p.ErrorLayer().Error(), "expect UDP packet")
	}

	dhcpLayer := p.Layer(layers.LayerTypeDHCPv6)
	if dhcpLayer != nil {
		// dhcpv6
		dhcp6 := dhcpLayer.(*layers.DHCPv6)
		sbf := gopacket.NewSerializeBuffer()
		if err := dhcp6.SerializeTo(sbf, gopacket.SerializeOptions{}); err != nil {
			return nil, nil, nil, 0, errors.Wrap(err, "Serialize dhcp6 packet error")
		}
		return sbf.Bytes(), &net.UDPAddr{IP: srcIp, Port: int(srcPort)}, srcMac, 0, nil
	} else {
		return nil, nil, nil, 0, errors.Wrap(p.ErrorLayer().Error(), "Fetch dhcp layer failed")
	}
}

func (s *rawSocketConn) Send6(b []byte, addr *net.UDPAddr, destMac net.HardwareAddr) error {
	var dhcp = new(layers.DHCPv4)
	if err := dhcp.DecodeFromBytes(b, gopacket.NilDecodeFeedback); err != nil {
		return errors.Wrap(err, "Decode dhcp bytes error")
	}

	var eth = &layers.Ethernet{
		EthernetType: layers.EthernetTypeIPv4,
		SrcMAC:       s.iface.HardwareAddr,
		DstMAC:       destMac,
	}

	var ip = &layers.IPv6{
		Version:    6,
		HopLimit:   64,
		SrcIP:      s.ip,
		DstIP:      addr.IP,
		NextHeader: layers.IPProtocolUDP,
	}

	var (
		srcPort = layers.UDPPort(s.serverPort)
		dstPort = layers.UDPPort(addr.Port)
	)

	var udp = &layers.UDP{
		SrcPort: srcPort,
		DstPort: dstPort,
	}
	udp.SetNetworkLayerForChecksum(ip)

	var (
		buf  = gopacket.NewSerializeBuffer()
		opts = gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	)
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, dhcp); err != nil {
		return errors.Wrap(err, "SerializeLayers error")
	}

	// s.conn.SetWriteDeadline(time.Now().Add(DefaultWriteTimeout)) // 2 second
	if _, err := s.conn.WriteTo(buf.Bytes(), &packet.Addr{HardwareAddr: destMac}); err != nil {
		return errors.Wrap(err, "Send dhcp packet error")
	}
	return nil
}
