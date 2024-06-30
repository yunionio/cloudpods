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

package pxe

import (
	"context"
	"fmt"
	"net"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/jsonutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/dhcp"
)

const (
	portDHCP = 67
	portTFTP = 69
)

// Architecture describes a kind of CPU architecture
type Architecture int

// Architecture types that pxe knows how to boot
// These architectures are self-reported by the booting machine. The
// machine may support additional execution mode. For example, legacy
// PC BIOS reports itself as an ArchIA32, but may also support ArchX64
// execution
const (
	// ArchIA32 is a 32-bit x86 machine. It may also support X64
	// execution, but pxe has no way of kowning.
	ArchIA32 Architecture = iota
	// ArchX64 is a 64-bit x86 machine (aka amd64 aka x64)
	ArchX64
	ArchUnknown
)

func (a Architecture) String() string {
	switch a {
	case ArchIA32:
		return "IA32"
	case ArchX64:
		return "X64"
	default:
		return "Unknown architecture"
	}
}

// A Machine describes a machine that is attempting to boot
type Machine struct {
	MAC  net.HardwareAddr
	Arch Architecture
}

// Firmware describes a kind of firmware attempting to boot.
// This should only be used for selecting the right bootloader within
// pxe, kernel selection should key off the more generic
// Architecture
type Firmware int

// The bootloaders that pxe knows how to handle
const (
	FirmwareX86PC   Firmware = iota // "Classic" x86 BIOS with PXE/UNDI support
	FirmwareEFI32                   // 32-bit x86 processor running EFI
	FirmwareEFI64                   // 64-bit x86 processor running EFI
	FirmwareEFIBC                   // 64-bit x86 processor running EFI
	FirmwareX86Ipxe                 // "Classic" x86 BIOS running iPXE (no UNDI support)
	FirmwareUnknown
)

type IBaremetalManager interface {
	GetZoneId() string
	GetBaremetalByMac(mac net.HardwareAddr) IBaremetalInstance
	AddBaremetal(ctx context.Context, desc jsonutils.JSONObject) (IBaremetalInstance, error)
	GetClientSession() *mcclient.ClientSession
}

type IBaremetalInstance interface {
	NeedPXEBoot() bool
	GetIPMINic(cliMac net.HardwareAddr) *types.SNic
	GetPXEDHCPConfig(arch uint16) (*dhcp.ResponseConfig, error)
	GetDHCPConfig(cliMac net.HardwareAddr) (*dhcp.ResponseConfig, error)
	InitAdminNetif(ctx context.Context, cliMac net.HardwareAddr, wireId string, nicType compute.TNicType, netType computeapi.TNetworkType, isDoImport bool, ipAddr string) error
	RegisterNetif(ctx context.Context, cliMac net.HardwareAddr, wireId string) error
	GetTFTPResponse() string
}

type Server struct {
	// Address to listen on, or empty for all interfaces
	Address          string
	DHCPPort         int
	ListenIface      string
	TFTPPort         int
	TFTPRootDir      string
	errs             chan error
	BaremetalManager IBaremetalManager
}

func (s *Server) Serve() error {
	if s.Address == "" {
		s.Address = "0.0.0.0"
	}
	if s.DHCPPort == 0 {
		s.DHCPPort = portDHCP
	}
	if s.TFTPPort == 0 {
		s.TFTPPort = portTFTP
	}

	tftpConn, err := net.ListenPacket("udp", fmt.Sprintf("%s:%d", s.Address, s.TFTPPort))
	if err != nil {
		return err
	}
	tftpHandler, err := NewTFTPHandler(s.TFTPRootDir, s.BaremetalManager)
	if err != nil {
		return err
	}

	dhcpSrv, err := dhcp.NewDHCPServer3(s.Address, s.DHCPPort)
	if err != nil {
		return err
	}

	s.errs = make(chan error)

	dhcpHandler := &DHCPHandler{baremetalManager: s.BaremetalManager}

	go func() { s.errs <- s.serveDHCP(dhcpSrv, dhcpHandler) }()
	go func() { s.errs <- s.serveTFTP(tftpConn, tftpHandler) }()

	err = <-s.errs
	return err
}
