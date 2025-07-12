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
	"net"
)

type DHCPHandler interface {
	ServeDHCP(pkt Packet, cliMac net.HardwareAddr, addr *net.UDPAddr) (Packet, []string, error)
}

type SendPacket struct {
	Packet  []byte
	DestMac net.HardwareAddr
}

type DHCP6Handler interface {
	DHCPHandler

	OnRecvICMP6(pkt Packet) error
}
