// MIT License
// ===========

// Copyright (C) 2015 Matt Layher

// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package ethernet

import (
	"encoding/binary"
	"errors"
	"net"
)

const (
	// minPayload is the minimum payload size for an Ethernet frame, assuming
	// that no 802.1Q VLAN tags are present.
	minPayload = 46
)

var (
	// Broadcast is a special hardware address which indicates a Frame should
	// be sent to every device on a given LAN segment.
	Broadcast = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
)

var (
	// ErrInvalidFCS is returned when Frame.UnmarshalFCS detects an incorrect
	// Ethernet frame check sequence in a byte slice for a Frame.
	ErrInvalidFCS = errors.New("invalid frame check sequence")
)

// An EtherType is a value used to identify an upper layer protocol
// encapsulated in a Frame.
//
// A list of IANA-assigned EtherType values may be found here:
// http://www.iana.org/assignments/ieee-802-numbers/ieee-802-numbers.xhtml.
type EtherType uint16

// Common EtherType values frequently used in a Frame.
const (
	EtherTypeIPv4 EtherType = 0x0800
	EtherTypeARP  EtherType = 0x0806
	EtherTypeIPv6 EtherType = 0x86DD

	// EtherTypeVLAN and EtherTypeServiceVLAN are used as 802.1Q Tag Protocol
	// Identifiers (TPIDs).
	EtherTypeVLAN        EtherType = 0x8100
	EtherTypeServiceVLAN EtherType = 0x88a8
)

// A Frame is an IEEE 802.3 Ethernet II frame.  A Frame contains information
// such as source and destination hardware addresses, zero or more optional
// 802.1Q VLAN tags, an EtherType, and payload data.
type Frame struct {
	// Destination specifies the destination hardware address for this Frame.
	//
	// If this address is set to Broadcast, the Frame will be sent to every
	// device on a given LAN segment.
	Destination net.HardwareAddr

	// Source specifies the source hardware address for this Frame.
	//
	// Typically, this is the hardware address of the network interface used to
	// send this Frame.
	Source net.HardwareAddr

	// ServiceVLAN specifies an optional 802.1Q service VLAN tag, for use with
	// 802.1ad double tagging, or "Q-in-Q". If ServiceVLAN is not nil, VLAN must
	// not be nil as well.
	//
	// Most users should leave this field set to nil and use VLAN instead.
	ServiceVLAN *VLAN

	// VLAN specifies an optional 802.1Q customer VLAN tag, which may or may
	// not be present in a Frame.  It is important to note that the operating
	// system may automatically strip VLAN tags before they can be parsed.
	VLAN *VLAN

	// EtherType is a value used to identify an upper layer protocol
	// encapsulated in this Frame.
	EtherType EtherType

	// Payload is a variable length data payload encapsulated by this Frame.
	Payload []byte
}

// read reads data from a Frame into b.  read is used to marshal a Frame
// into binary form, but does not allocate on its own.
func (f *Frame) read(b []byte) (int, error) {
	// S-VLAN must also have accompanying C-VLAN.
	if f.ServiceVLAN != nil && f.VLAN == nil {
		return 0, ErrInvalidVLAN
	}

	copy(b[0:6], f.Destination)
	copy(b[6:12], f.Source)

	// Marshal each non-nil VLAN tag into bytes, inserting the appropriate
	// EtherType/TPID before each, so devices know that one or more VLANs
	// are present.
	vlans := []struct {
		vlan *VLAN
		tpid EtherType
	}{
		{vlan: f.ServiceVLAN, tpid: EtherTypeServiceVLAN},
		{vlan: f.VLAN, tpid: EtherTypeVLAN},
	}

	n := 12
	for _, vt := range vlans {
		if vt.vlan == nil {
			continue
		}

		// Add VLAN EtherType and VLAN bytes.
		binary.BigEndian.PutUint16(b[n:n+2], uint16(vt.tpid))
		if _, err := vt.vlan.read(b[n+2 : n+4]); err != nil {
			return 0, err
		}
		n += 4
	}

	// Marshal actual EtherType after any VLANs, copy payload into
	// output bytes.
	binary.BigEndian.PutUint16(b[n:n+2], uint16(f.EtherType))
	copy(b[n+2:], f.Payload)

	return len(b), nil
}
