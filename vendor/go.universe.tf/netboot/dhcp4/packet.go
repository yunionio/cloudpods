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

package dhcp4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
)

var magic = []byte{99, 130, 83, 99}

// MessageType is the type of a DHCP message.
type MessageType int

// Message types as described in RFC 2131.
const (
	MsgDiscover MessageType = iota + 1
	MsgOffer
	MsgRequest
	MsgDecline
	MsgAck
	MsgNack
	MsgRelease
	MsgInform
)

func (mt MessageType) String() string {
	switch mt {
	case MsgDiscover:
		return "DHCPDISCOVER"
	case MsgOffer:
		return "DHCPOFFER"
	case MsgRequest:
		return "DHCPREQUEST"
	case MsgDecline:
		return "DHCPDECLINE"
	case MsgAck:
		return "DHCPACK"
	case MsgNack:
		return "DHCPNAK"
	case MsgRelease:
		return "DHCPRELEASE"
	case MsgInform:
		return "DHCPINFORM"
	default:
		return fmt.Sprintf("<unknown DHCP message type %d>", mt)
	}
}

// Packet represents a DHCP packet.
type Packet struct {
	Type          MessageType
	TransactionID []byte // Always 4 bytes
	Broadcast     bool
	HardwareAddr  net.HardwareAddr // Only ethernet supported at the moment

	ClientAddr net.IP // Client's current IP address (it will respond to ARP for this IP)
	YourAddr   net.IP // Client IP address offered/assigned by server
	ServerAddr net.IP // Responding server's IP address
	RelayAddr  net.IP // IP address of DHCP relay agent, if an agent forwarded the request

	BootServerName string
	BootFilename   string

	Options Options
}

// decides how to send Packet on the wire, based on its field values.
//
// This implements the transmission decision process in section 4.1 of
// RFC 2131.
func (p *Packet) txType() txType {
	switch {
	case p.RelayAddr != nil && p.RelayAddr.IsGlobalUnicast():
		return txRelayAddr
	case p.Type == MsgNack:
		return txBroadcast
	case p.ClientAddr != nil && (p.ClientAddr.IsGlobalUnicast() || p.ClientAddr.IsLoopback()):
		return txClientAddr
	case p.Broadcast:
		return txBroadcast
	default:
		return txHardwareAddr
	}
}

// DebugString prints the contents of a DHCP packet for human consumption.
func (p *Packet) DebugString() string {
	var b bytes.Buffer
	bcast := "Unicast"
	if p.Broadcast {
		bcast = "Broadcast"
	}
	fmt.Fprintf(&b, `%s
  %#v
  %s
  MAC: %s
  ClientIP: %s
  YourIP: %s
  ServerIP: %s
  RelayIP: %s

  BootServerName: %s
  BootFilename: %s

  Options:
`, p.Type, p.TransactionID, bcast, p.HardwareAddr, p.ClientAddr, p.YourAddr, p.ServerAddr, p.RelayAddr, p.BootServerName, p.BootFilename)

	var opts []int
	for n := range p.Options {
		opts = append(opts, int(n))
	}
	sort.Ints(opts)
	for _, n := range opts {
		fmt.Fprintf(&b, "    %d: %#v\n", n, p.Options[Option(n)])
	}
	return b.String()
}

// Marshal returns the wire encoding of p.
func (p *Packet) Marshal() ([]byte, error) {
	if len(p.TransactionID) != 4 {
		return nil, errors.New("transaction ID must be 4 bytes")
	}
	if len(p.HardwareAddr) != 6 {
		return nil, errors.New("non-ethernet hardware address not supported")
	}
	if len(p.BootServerName) > 64 {
		return nil, errors.New("sname must be <= 64 bytes")
	}
	optsInFile, optsInSname := false, false
	v, err := p.Options.Byte(OptOverload)
	if err == nil {
		optsInFile, optsInSname = v&1 != 0, v&2 != 0
	}
	if optsInFile && p.BootFilename != "" {
		return nil, errors.New("DHCP option 52 says to use the 'file' field for options, but BootFilename is not empty")
	}
	if optsInSname && p.BootServerName != "" {
		return nil, errors.New("DHCP option 52 says to use the 'sname' field for options, but BootServerName is not empty")
	}

	ret := new(bytes.Buffer)
	ret.Grow(244)

	switch p.Type {
	case MsgDiscover, MsgRequest, MsgDecline, MsgRelease, MsgInform:
		ret.WriteByte(1)
	case MsgOffer, MsgAck, MsgNack:
		ret.WriteByte(2)
	default:
		return nil, fmt.Errorf("unknown DHCP message type %d", p.Type)
	}
	// Hardware address type = Ethernet
	ret.WriteByte(1)
	// Hardware address length = 6
	ret.WriteByte(6)
	// Hops = 0
	ret.WriteByte(0)
	// Transaction ID
	ret.Write(p.TransactionID)
	// Seconds elapsed
	ret.Write([]byte{0, 0})
	// Broadcast flag
	if p.Broadcast {
		ret.Write([]byte{0x80, 0})
	} else {
		ret.Write([]byte{0, 0})
	}

	writeIP(ret, p.ClientAddr)
	writeIP(ret, p.YourAddr)
	writeIP(ret, p.ServerAddr)
	writeIP(ret, p.RelayAddr)

	// MAC address + 10 bytes of padding
	ret.Write([]byte(p.HardwareAddr))
	ret.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

	opts := make(Options, len(p.Options)+1)
	for k, v := range p.Options {
		opts[k] = v
	}
	opts[53] = []byte{byte(p.Type)}
	if optsInSname {
		opts, err = opts.marshalLimited(ret, 64, true)
		if err != nil {
			return nil, err
		}
	} else {
		ret.WriteString(p.BootServerName)
		for i := len(p.BootServerName); i < 64; i++ {
			ret.WriteByte(0)
		}
	}

	if optsInFile {
		opts, err = opts.marshalLimited(ret, 128, true)
		if err != nil {
			return nil, err
		}
	} else {
		ret.WriteString(p.BootFilename)
		for i := len(p.BootFilename); i < 128; i++ {
			ret.WriteByte(0)
		}
	}

	ret.Write(magic)
	leftover, err := opts.marshalLimited(ret, 0, false)
	if err != nil {
		return nil, err
	}
	if len(leftover) > 0 {
		return nil, errors.New("some options not written, but no limit was given (please file a bug)")
	}

	return ret.Bytes(), nil
}

func writeIP(w io.Writer, ip net.IP) {
	ip = ip.To4()
	if ip == nil {
		w.Write([]byte{0, 0, 0, 0})
	} else {
		w.Write([]byte(ip))
	}
}

// Unmarshal parses a DHCP message and returns a Packet.
func Unmarshal(bs []byte) (*Packet, error) {
	// 244 bytes is the minimum size of a valid DHCP message:
	//  - BOOTP header (236b)
	//  - DHCP magic (4b)
	//  - DHCP message type mandatory option (3b)
	//  - End of options marker (1b)
	if len(bs) < 244 {
		return nil, errors.New("packet too short")
	}

	if !bytes.Equal(bs[236:240], magic) {
		return nil, errors.New("packet does not have DHCP magic number")
	}

	ret := &Packet{
		Options: make(Options),
	}

	if bs[1] != 1 || bs[2] != 6 {
		return nil, fmt.Errorf("packet has unsupported hardware address type/length %d/%d", bs[1], bs[2])
	}
	ret.HardwareAddr = net.HardwareAddr(bs[28:34])
	ret.TransactionID = bs[4:8]
	if binary.BigEndian.Uint16(bs[10:12])&0x8000 != 0 {
		ret.Broadcast = true
	}

	ret.ClientAddr = net.IP(bs[12:16])
	ret.YourAddr = net.IP(bs[16:20])
	ret.ServerAddr = net.IP(bs[20:24])
	ret.RelayAddr = net.IP(bs[24:28])

	if err := ret.Options.Unmarshal(bs[240:]); err != nil {
		return nil, fmt.Errorf("packet has malformed options section: %s", err)
	}

	// The 'file' and 'sname' BOOTP fields can either have the obvious
	// meaning from BOOTP, or can store extra DHCP options if the main
	// options section specifies the "Option overload" option.
	file, sname := false, false
	v, err := ret.Options.Byte(OptOverload)
	if err == nil {
		file, sname = v&1 != 0, v&2 != 0
	}
	if sname {
		if err := ret.Options.Unmarshal(bs[44:108]); err != nil {
			return nil, fmt.Errorf("packet has malformed options in 'sname' field: %s", err)
		}
	} else {
		s, ok := nullStr(bs[44:108])
		if !ok {
			return nil, fmt.Errorf("unterminated 'sname' string")
		}
		ret.BootServerName = s
	}
	if file {
		if err := ret.Options.Unmarshal(bs[108:236]); err != nil {
			return nil, fmt.Errorf("packet has malformed options in 'file' field: %s", err)
		}
	} else {
		s, ok := nullStr(bs[108:236])
		if !ok {
			return nil, fmt.Errorf("unterminated 'file' string")
		}
		ret.BootFilename = s
	}

	// DHCP packets must all have at least the "DHCP Message Type"
	// option.
	typ, err := ret.Options.Byte(OptDHCPMessageType)
	if err != nil {
		return nil, fmt.Errorf("getting DHCP Message type: %s", err)
	}
	ret.Type = MessageType(typ)
	delete(ret.Options, OptDHCPMessageType)
	switch ret.Type {
	case MsgDiscover, MsgRequest, MsgDecline, MsgRelease, MsgInform:
		if bs[0] != 1 {
			return nil, fmt.Errorf("BOOTP message type (%d) doesn't match DHCP message type (%s)", bs[0], ret.Type)
		}
	case MsgOffer, MsgAck, MsgNack:
		if bs[0] != 2 {
			return nil, fmt.Errorf("BOOTP message type (%d) doesn't match DHCP message type (%s", bs[0], ret.Type)
		}
	default:
		return nil, fmt.Errorf("Unknown DHCP message type %d", ret.Type)
	}

	return ret, nil
}

func nullStr(bs []byte) (string, bool) {
	i := bytes.IndexByte(bs, 0)
	if i == -1 {
		return "", false
	}
	return string(bs[:i]), true
}
