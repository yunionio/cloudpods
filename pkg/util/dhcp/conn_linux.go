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
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/mdlayher/packet"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type rawSocketConn struct {
	conn *packet.Conn

	iface *net.Interface
	ip    net.IP

	serverPort uint16
}

func newRawSocketConn(iface string, filter []bpf.RawInstruction, serverPort uint16) (conn, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, errors.Wrap(err, "interface by name")
	}

	ip, err := interfaceToIPv4Addr(ifi)
	if err != nil {
		return nil, errors.Wrap(err, "interfaceToIPv4Addr")
	}

	// unix.ETH_P_ALL
	conn, err := packet.Listen(ifi, packet.Raw, unix.ETH_P_IP, &packet.Config{
		Filter: filter,
	})
	if err != nil {
		return nil, errors.Wrap(err, "packet.Listen")
	}
	log.Debugf("newRawSocketConn on %s %s", ifi.Name, ip)

	return &rawSocketConn{
		conn:       conn,
		iface:      ifi,
		ip:         ip,
		serverPort: serverPort,
	}, nil
}

func (s *rawSocketConn) Close() error {
	return s.conn.Close()
}

func (s *rawSocketConn) Recv(b []byte) ([]byte, *net.UDPAddr, net.HardwareAddr, error) {
	// read packet
	n, addr, err := s.conn.ReadFrom(b)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Read from errror")
	}
	log.Debugf("rawSocketConn Recv %d bytes", n)

	b = b[:n]

	srcMac, err := net.ParseMAC(addr.String())
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Parse mac error")
	}

	p := gopacket.NewPacket(b, layers.LayerTypeEthernet, gopacket.Default)
	if p.ErrorLayer() != nil {
		return nil, nil, nil, errors.Wrap(p.ErrorLayer().Error(), "Failed to decode packet")
	}

	var srcIp net.IP
	{
		ipLayer := p.Layer(layers.LayerTypeIPv4)
		if ipLayer != nil {
			// ipv4
			ip4 := ipLayer.(*layers.IPv4)
			srcIp = ip4.SrcIP
		} else {
			return nil, nil, nil, errors.Wrap(p.ErrorLayer().Error(), "Expect IP packet")
		}
	}

	var srcPort uint16
	udpLayer := p.Layer(layers.LayerTypeUDP)
	if udpLayer != nil {
		udpInfo := udpLayer.(*layers.UDP)
		srcPort = uint16(udpInfo.SrcPort)
	} else {
		return nil, nil, nil, errors.Wrap(p.ErrorLayer().Error(), "Expect UDP packet")
	}

	dhcpLayer := p.Layer(layers.LayerTypeDHCPv4)
	if dhcpLayer != nil {
		// dhcpv4
		dhcp4 := dhcpLayer.(*layers.DHCPv4)
		sbf := gopacket.NewSerializeBuffer()
		if err := dhcp4.SerializeTo(sbf, gopacket.SerializeOptions{}); err != nil {
			return nil, nil, nil, errors.Wrap(err, "Serialize dhcp packet error")
		}
		return sbf.Bytes(), &net.UDPAddr{IP: srcIp, Port: int(srcPort)}, srcMac, nil
	} else {
		return nil, nil, nil, errors.Wrap(p.ErrorLayer().Error(), "Expect DHCP packet")
	}
}

func (s *rawSocketConn) Send(b []byte, addr *net.UDPAddr, destMac net.HardwareAddr) error {
	var dhcp = new(layers.DHCPv4)
	if err := dhcp.DecodeFromBytes(b, gopacket.NilDecodeFeedback); err != nil {
		return errors.Wrap(err, "Decode dhcp bytes error")
	}

	var eth = &layers.Ethernet{
		EthernetType: layers.EthernetTypeIPv4,
		SrcMAC:       s.iface.HardwareAddr,
		DstMAC:       destMac,
	}

	var ip = &layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    s.ip,
		DstIP:    addr.IP,
		Protocol: layers.IPProtocolUDP,
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

func (s *rawSocketConn) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}

func (s *rawSocketConn) SetWriteDeadline(t time.Time) error {
	return s.conn.SetWriteDeadline(t)
}
