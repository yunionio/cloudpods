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

// Option is a DHCP option.
type Option byte

// Some of the more commonly seen DHCP options. Refer to
// http://www.iana.org/assignments/bootp-dhcp-parameters/bootp-dhcp-parameters.xhtml
// for the full authoritative list.
const (
	OptSubnetMask         Option = 1  // IPMask
	OptTimeOffset         Option = 2  // int32
	OptRouters            Option = 3  // IPs
	OptDNSServers         Option = 6  // IPs
	OptHostname           Option = 12 // string
	OptBootFileSize       Option = 13 // uint16
	OptDomainName         Option = 15 // string
	OptInterfaceMTU       Option = 26 // uint16
	OptBroadcastAddr      Option = 28 // IP
	OptNTPServers         Option = 42 // IP
	OptVendorSpecific     Option = 43 // []byte
	OptRequestedIP        Option = 50 // IP
	OptLeaseTime          Option = 51 // uint32
	OptOverload           Option = 52 // byte
	OptServerIdentifier   Option = 54 // IP
	OptRequestedOptions   Option = 55 // []byte
	OptMessage            Option = 56 // string
	OptMaximumMessageSize Option = 57 // uint16
	OptRenewalTime        Option = 58 // uint32
	OptRebindingTime      Option = 59 // uint32
	OptVendorIdentifier   Option = 60 // string
	OptClientIdentifier   Option = 61 // string
	OptFQDN               Option = 81 // string

	// You shouldn't need to use the following directly. Instead,
	// refer to the fields in the Packet struct, and Marshal/Unmarshal
	// will handle encoding for you.

	OptTFTPServer      Option = 66 // string
	OptBootFile        Option = 67 // string
	OptDHCPMessageType Option = 53 // byte
)

// Options stores DHCP options.
type Options map[Option][]byte

var (
	errOptionNotPresent = errors.New("option not present in Options")
	errOptionWrongSize  = errors.New("option value is the wrong size")
)

// Unmarshal parses DHCP options into o.
func (o Options) Unmarshal(bs []byte) error {
	for len(bs) > 0 {
		opt := Option(bs[0])
		switch opt {
		case 0:
			// Padding byte
			bs = bs[1:]
		case 255:
			// End of options
			return nil
		default:
			// In theory, DHCP permits multiple instances of the same
			// option in a packet, as a way to have option values >255
			// bytes. Unfortunately, this is very loosely specified as
			// "up to individual options", and AFAICT, isn't at all
			// used in the wild.
			//
			// So, for now, seeing the same option twice in a packet
			// is going to be an error, until I get a bug report about
			// something that actually does it.
			if _, ok := o[opt]; ok {
				// Okay fine option 56 can be duped.
				if opt != 56 {
					return fmt.Errorf("packet has duplicate option %d (please file a bug with a pcap!)", opt)
				}
			}
			if len(bs) < 2 {
				return fmt.Errorf("option %d has no length byte", opt)
			}
			l := int(bs[1])
			if len(bs[2:]) < l {
				return fmt.Errorf("option %d claims to have %d bytes of payload, but only has %d bytes", opt, l, len(bs[2:]))
			}
			o[opt] = bs[2 : 2+l]
			bs = bs[2+l:]
		}
	}

	return errors.New("options are not terminated by a 255 byte")
}

// Marshal returns the wire encoding of o.
func (o Options) Marshal() ([]byte, error) {
	var ret bytes.Buffer
	opts, err := o.marshalLimited(&ret, 0, false)
	if err != nil {
		return nil, err
	}
	if len(opts) > 0 {
		return nil, errors.New("some options not written, but no limit was given (please file a bug)")
	}

	return ret.Bytes(), nil
}

// Copy returns a shallow copy of o.
func (o Options) Copy() Options {
	ret := make(Options, len(o))
	for k, v := range o {
		ret[k] = v
	}
	return ret
}

// marshalLimited serializes o into w. If nBytes > 0, as many options
// as possible are packed into that many bytes, inserting padding as
// needed, and the remaining unwritten options are returned.
func (o Options) marshalLimited(w io.Writer, nBytes int, skip52 bool) (Options, error) {
	ks := make([]int, 0, len(o))
	for n := range o {
		if n <= 0 || n >= 255 {
			return nil, fmt.Errorf("invalid DHCP option number %d", n)
		}
		ks = append(ks, int(n))
	}
	sort.Ints(ks)

	ret := make(Options)
	for _, n := range ks {
		opt := o[Option(n)]
		if len(opt) > 255 {
			return nil, fmt.Errorf("DHCP option %d has value >255 bytes", n)
		}

		// If space is limited, verify that we can fit the option plus
		// the final end-of-options marker.
		if nBytes > 0 && ((skip52 && n == 52) || len(opt)+3 > nBytes) {
			ret[Option(n)] = opt
			continue
		}

		w.Write([]byte{byte(n), byte(len(opt))})
		w.Write(opt)
		nBytes -= len(opt) + 2
	}

	w.Write([]byte{255})
	nBytes--
	if nBytes > 0 {
		w.Write(make([]byte, nBytes))
	}

	return ret, nil
}

// Bytes returns the value of option n as a byte slice.
func (o Options) Bytes(n Option) ([]byte, error) {
	bs := o[n]
	if bs == nil {
		return nil, errOptionNotPresent
	}
	return bs, nil
}

// String returns the value of option n as a string.
func (o Options) String(n Option) (string, error) {
	bs, err := o.Bytes(n)
	if err != nil {
		return "", err
	}
	return string(bs), err
}

// Byte returns the value of option n as a byte.
func (o Options) Byte(n Option) (byte, error) {
	bs, err := o.Bytes(n)
	if err != nil {
		return 0, err
	}
	if len(bs) != 1 {
		return 0, errOptionWrongSize
	}
	return bs[0], nil
}

// Uint16 returns the value of option n as a uint16.
func (o Options) Uint16(n Option) (uint16, error) {
	bs, err := o.Bytes(n)
	if err != nil {
		return 0, err
	}
	if len(bs) != 2 {
		return 0, errOptionWrongSize
	}
	return binary.BigEndian.Uint16(bs), nil
}

// Uint32 returns the value of option n as a uint32.
func (o Options) Uint32(n Option) (uint32, error) {
	bs, err := o.Bytes(n)
	if err != nil {
		return 0, err
	}
	if len(bs) != 4 {
		return 0, errOptionWrongSize
	}
	return binary.BigEndian.Uint32(bs), nil
}

// Int32 returns the value of option n as an int32.
func (o Options) Int32(n Option) (int32, error) {
	bs, err := o.Bytes(n)
	if err != nil {
		return 0, err
	}
	if len(bs) != 4 {
		return 0, errOptionWrongSize
	}
	return int32(binary.BigEndian.Uint32(bs)), nil
}

// IPs returns the value of option n as a list of IPv4 addresses.
func (o Options) IPs(n Option) ([]net.IP, error) {
	bs, err := o.Bytes(n)
	if err != nil {
		return nil, err
	}
	if len(bs) < 4 || len(bs)%4 != 0 {
		return nil, errOptionWrongSize
	}
	ret := make([]net.IP, 0, len(bs)/4)
	for i := 0; i < len(bs); i += 4 {
		ret = append(ret, net.IP(bs[i:i+4]))
	}
	return ret, nil
}

// IP returns the value of option n as an IPv4 address.
func (o Options) IP(n Option) (net.IP, error) {
	ips, err := o.IPs(n)
	if err != nil {
		return nil, err
	}
	if len(ips) != 1 {
		return nil, errOptionWrongSize
	}
	return ips[0], nil
}

// IPMask returns the value of option n as a net.IPMask.
func (o Options) IPMask(n Option) (net.IPMask, error) {
	bs := o[n]
	if bs == nil {
		return nil, fmt.Errorf("option %d not found", n)
	}
	if len(bs) != 4 {
		return nil, fmt.Errorf("option %d is the wrong size for an IPMask", n)
	}
	return net.IPMask(bs), nil
}
