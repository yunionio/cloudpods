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

package dhcp

import (
	"net"
	"syscall"

	"golang.org/x/net/bpf"

	"yunion.io/x/pkg/errors"
)

// defined as a var so tests can override it.
var (
	dhcpv6ClientPort = 546
)

func NewRawSocketConn6(iface string, filter []bpf.RawInstruction, serverPort uint16) (*Conn, error) {
	conn, err := newRawSocketConn6(iface, filter, serverPort)
	if err != nil {
		return nil, err
	}
	return &Conn{conn, 0}, nil
}

func NewSocketConn6(addr string, port int) (*Conn, error) {
	conn, err := newSocketConn6(net.ParseIP(addr), port, false)
	if err != nil {
		return nil, err
	}
	return &Conn{conn, 0}, nil
}

func interfaceToIPv6Addr(ifi *net.Interface) (net.IP, error) {
	if ifi == nil {
		return net.IPv6zero, nil
	}
	ifat, err := ifi.Addrs()
	if err != nil {
		return nil, err
	}
	for _, ifa := range ifat {
		switch v := ifa.(type) {
		case *net.IPAddr:
			if len(v.IP) == net.IPv6len {
				return v.IP, nil
			}
		case *net.IPNet:
			if len(v.IP) == net.IPv6len {
				return v.IP, nil
			}
		}
	}
	return nil, errors.Wrapf(errors.ErrNotFound, "no such network interface %s", ifi.Name)
}

func newSocketConn6(addr net.IP, port int, disableBroadcast bool) (conn, error) {
	var broadcastOpt = 1
	if disableBroadcast {
		broadcastOpt = 0
	}
	sock, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, err
	}
	err = syscall.SetsockoptInt(sock, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return nil, err
	}
	err = syscall.SetsockoptInt(sock, syscall.SOL_SOCKET, syscall.SO_BROADCAST, broadcastOpt)
	if err != nil {
		return nil, err
	}
	byteAddr := [16]byte{}
	copy(byteAddr[:], addr.To16())
	lsa := &syscall.SockaddrInet6{
		Port: port,
		Addr: byteAddr,
	}
	if err = syscall.Bind(sock, lsa); err != nil {
		return nil, err
	}
	if err = syscall.SetNonblock(sock, false); err != nil {
		return nil, err
	}

	// Its equal syscall.CloseOnExec
	// most file descriptors are getting set to close-on-exec
	// apart from syscall open, socket etc.
	syscall.Syscall(syscall.SYS_FCNTL, uintptr(sock), syscall.F_SETFD, syscall.FD_CLOEXEC)
	return &socketConn{sock}, nil
}

func (s *socketConn) Recv6(b []byte) ([]byte, *net.UDPAddr, net.HardwareAddr, int, error) {
	n, a, err := syscall.Recvfrom(s.sock, b, 0)
	if err != nil {
		return nil, nil, nil, 0, err
	}
	if addr, ok := a.(*syscall.SockaddrInet6); !ok {
		return nil, nil, nil, 0, errors.Wrap(errors.ErrUnsupportedProtocol, "Recvfrom recevice address is not famliy Inet6")
	} else {
		ip := net.IP(addr.Addr[:])
		udpAddr := &net.UDPAddr{
			IP:   ip,
			Port: addr.Port,
		}
		// there is no interface index info
		return b[:n], udpAddr, nil, 0, nil
	}
}

func (s *socketConn) Send6(b []byte, addr *net.UDPAddr, destMac net.HardwareAddr) error {
	destIp := [16]byte{}
	copy(destIp[:], addr.IP.To16())
	destAddr := &syscall.SockaddrInet6{
		Addr: destIp,
		Port: addr.Port,
	}
	return syscall.Sendto(s.sock, b, 0, destAddr)
}
