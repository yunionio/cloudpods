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
	"errors"
	"fmt"
	"net"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/dhcp"
)

func (s *Server) serveDHCP(srv *dhcp.DHCPServer, handler dhcp.DHCPHandler) error {
	return srv.ListenAndServe(handler)
}

type NetworkInterfaceIdent struct {
	Type   uint16
	Major  uint16
	Minior uint16
}

type DHCPHandler struct {
	// baremetal manager
	baremetalManager IBaremetalManager
}

type dhcpRequest struct {
	packet                dhcp.Packet
	ClientMac             net.HardwareAddr // client nic mac
	ClientAddr            net.IP           // IP address from DHCP client
	RelayAddr             net.IP           // IP address of DHCP relay agent
	Options               dhcp.Options     // dhcp packet options
	VendorClassId         string
	ClientArch            uint16
	NetworkInterfaceIdent NetworkInterfaceIdent
	ClientGuid            string

	// baremetal manager
	baremetalManager IBaremetalManager
	// baremetal instance
	baremetalInstance IBaremetalInstance
	// cloud network config
	netConfig *types.SNetworkConfig
}

func (h *DHCPHandler) ServeDHCP(pkt dhcp.Packet, _ *net.UDPAddr, _ *net.Interface) (dhcp.Packet, error) {
	req, err := h.newRequest(pkt, h.baremetalManager)
	if err != nil {
		log.Errorf("[DHCP] new request by packet error: %v", err)
		return nil, err
	}
	log.V(4).Debugf("[DHCP] request packet: %#v", req)

	if req.RelayAddr.String() == "0.0.0.0" {
		return nil, fmt.Errorf("Request not from a DHCP relay, ignore mac: %s", req.ClientMac)
	}
	conf, err := req.fetchConfig(h.baremetalManager.GetClientSession())
	if err != nil {
		return nil, err
	}
	if conf == nil {
		return nil, fmt.Errorf("Empty packet config")
	}
	return dhcp.MakeReplyPacket(pkt, conf)
}

func (h *DHCPHandler) newRequest(pkt dhcp.Packet, man IBaremetalManager) (*dhcpRequest, error) {
	req := &dhcpRequest{
		baremetalManager: man,
		packet:           pkt,
		ClientAddr:       pkt.CIAddr(),
		ClientMac:        pkt.CHAddr(),
		RelayAddr:        pkt.RelayAddr(),
		Options:          pkt.ParseOptions(),
	}

	var (
		vendorClsId string
		cliArch     uint16
		err         error
		netIfIdent  NetworkInterfaceIdent
		cliGuid     string
	)

	for optCode, data := range req.Options {
		switch optCode {
		case dhcp.OptionVendorClassIdentifier:
			vendorClsId, err = req.Options.String(optCode)
		case dhcp.OptionClientArchitecture:
			cliArch, err = req.Options.Uint16(optCode)
		case dhcp.OptionClientNetworkInterfaceIdentifier:
			netIfIdentBs, err := req.Options.Bytes(optCode)
			if err != nil {
				break
			}
			netIfIdent = NetworkInterfaceIdent{
				Type:   uint16(netIfIdentBs[0]),
				Major:  uint16(netIfIdentBs[1]),
				Minior: uint16(netIfIdentBs[2]),
			}
			log.Debugf("[DHCP] get network iface identifier: %#v", netIfIdent)
		case dhcp.OptionClientMachineIdentifier:
			switch len(data) {
			case 0:
				// A missing GUID is invalid according to the spec, however
				// there are PXE ROMs in the wild that omit the GUID and still
				// expect to boot.
			case 17:
				if data[0] != 0 {
					err = errors.New("malformed client GUID (option 97), leading byte must be zero")
				}
			default:
				err = errors.New("malformed client GUID (option 97), wrong size")
			}
			cliGuid, err = req.Options.String(optCode)
		}
		if err != nil {
			log.Errorf("[DHCP] parse vendor option %d error: %v", optCode, err)
		}
	}
	req.VendorClassId = vendorClsId
	req.ClientArch = cliArch
	req.NetworkInterfaceIdent = netIfIdent
	req.ClientGuid = cliGuid
	return req, err
}

func (req *dhcpRequest) fetchConfig(session *mcclient.ClientSession) (*dhcp.ResponseConfig, error) {
	// 1. find_network_conf
	netConf, err := req.findNetworkConf(session, false)
	if err != nil {
		return nil, err
	}
	req.netConfig = netConf

	// TODO: set cache for netConf
	if req.isPXERequest() {
		// handle PXE DHCP request
		log.Infof("DHCP relay from %s(%s) for %s, find matched networks: %#v", req.RelayAddr, req.ClientAddr, req.ClientMac, netConf)
		bmDesc, err := req.createOrUpdateBaremetal(session)
		if err != nil {
			return nil, err
		}
		err = req.doInitBaremetalAdminNetif(bmDesc)
		if err != nil {
			return nil, err
		}
		// always response PXE request
		// let bootloader decide boot local or remote
		// if req.baremetalInstance.NeedPXEBoot() {
		return req.baremetalInstance.GetPXEDHCPConfig(req.ClientArch)
		// }
		// ignore
		// log.Warningf("No need to pxeboot, ignore the request ...(mac:%s guid:%s)", req.ClientMac, req.ClientGuid)
		// return nil, nil
	} else {
		// handle normal DHCP request
		bmInstance := req.baremetalManager.GetBaremetalByMac(req.ClientMac)
		if bmInstance == nil {
			// options.EnableGeneralGuestDhcp
			// cloud be an instance not served by a host-server
			// from guestdhcp import GuestDHCPHelperTask
			// task = GuestDHCPHelperTask(self)
			// task.start()
			return nil, nil
		}
		req.baremetalInstance = bmInstance
		ipmiNic := req.baremetalInstance.GetIPMINic(req.ClientMac)
		if ipmiNic != nil && ipmiNic.Mac == req.ClientMac.String() {
			err = req.baremetalInstance.InitAdminNetif(
				req.ClientMac, req.netConfig.WireId, types.NIC_TYPE_IPMI, api.NETWORK_TYPE_IPMI, false, "")
			if err != nil {
				return nil, err
			}
		} else {
			err = req.baremetalInstance.RegisterNetif(req.ClientMac, req.netConfig.WireId)
			if err != nil {
				log.Errorf("RegisterNetif error: %v", err)
				return nil, err
			}
		}
		return req.baremetalInstance.GetDHCPConfig(req.ClientMac)
	}
}

func (req *dhcpRequest) findNetworkConf(session *mcclient.ClientSession, filterUseIp bool) (*types.SNetworkConfig, error) {
	params := jsonutils.NewDict()
	if filterUseIp {
		params.Add(jsonutils.NewString(req.RelayAddr.String()), "ip")
	} else {
		params.Add(jsonutils.NewString(
			fmt.Sprintf("guest_gateway.equals(%s)", req.RelayAddr)),
			"filter.0")
		params.Add(jsonutils.NewString(
			fmt.Sprintf("guest_dhcp.equals(%s)", req.RelayAddr)),
			"filter.1")
		params.Add(jsonutils.JSONTrue, "filter_any")
	}
	params.Add(jsonutils.JSONTrue, "is_on_premise")
	ret, err := modules.Networks.List(session, params)
	if err != nil {
		return nil, err
	}
	if len(ret.Data) == 0 {
		if !filterUseIp {
			// use ip filter try again
			return req.findNetworkConf(session, true)
		}
		return nil, fmt.Errorf("DHCP relay from %s(%s) for %s, find no match network", req.RelayAddr, req.ClientAddr, req.ClientMac)
	}

	network := types.SNetworkConfig{}
	err = ret.Data[0].Unmarshal(&network)
	return &network, err
}

func (req *dhcpRequest) findBaremetalsOfAnyMac(session *mcclient.ClientSession, isBaremetal bool) (*modulebase.ListResult, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(req.ClientMac.String()), "any_mac")
	if isBaremetal {
		params.Add(jsonutils.JSONTrue, "is_baremetal")
	} else {
		params.Add(jsonutils.NewString(api.HOST_TYPE_BAREMETAL), "host_type")
	}
	return modules.Hosts.List(session, params)
}

// createOrUpdateBaremetal create or update baremetal by client MAC
func (req *dhcpRequest) createOrUpdateBaremetal(session *mcclient.ClientSession) (jsonutils.JSONObject, error) {
	ret, err := req.findBaremetalsOfAnyMac(session, true)
	if err != nil {
		return nil, err
	}
	if len(ret.Data) == 0 {
		ret, err = req.findBaremetalsOfAnyMac(session, false)
		if err != nil {
			return nil, err
		}
	}
	switch len(ret.Data) {
	case 0:
		// found new baremetal, create it if auto register
		if o.Options.AutoRegisterBaremetal {
			return req.createBaremetal(session)
		}
	case 1:
		// already exists, do update
		bmId, err := ret.Data[0].GetString("id")
		if err != nil {
			return nil, err
		}
		return req.updateBaremetal(session, bmId)
	}
	return nil, fmt.Errorf("Found %d records match %s", len(ret.Data), req.ClientMac)
}

func (req *dhcpRequest) createBaremetal(session *mcclient.ClientSession) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	mac := req.ClientMac.String()
	zoneId := req.baremetalManager.GetZoneId()
	name := fmt.Sprintf("BM%s", strings.Replace(mac, ":", "", -1))
	params.Add(jsonutils.NewString(name), "name")
	params.Add(jsonutils.NewString(mac), "access_mac")
	params.Add(jsonutils.NewString("baremetal"), "host_type")
	params.Add(jsonutils.JSONTrue, "is_baremetal")
	params.Add(jsonutils.NewString(zoneId), "zone_id")
	desc, err := modules.Hosts.Create(session, params)
	if err != nil {
		return nil, err
	}
	return desc, nil
}

func (req *dhcpRequest) updateBaremetal(session *mcclient.ClientSession, id string) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(req.ClientMac.String()), "access_mac")
	params.Add(jsonutils.NewString(req.baremetalManager.GetZoneId()), "zone_id")
	// params.Add(jsonutils.NewString("baremetal"), "host_type")
	params.Add(jsonutils.JSONTrue, "is_baremetal")
	desc, err := modules.Hosts.Update(session, id, params)
	if err != nil {
		return nil, err
	}
	return desc, nil
}

func (req *dhcpRequest) doInitBaremetalAdminNetif(desc jsonutils.JSONObject) error {
	var err error
	req.baremetalInstance, err = req.baremetalManager.AddBaremetal(desc)
	if err != nil {
		return err
	}
	err = req.baremetalInstance.InitAdminNetif(
		req.ClientMac, req.netConfig.WireId, types.NIC_TYPE_ADMIN, api.NETWORK_TYPE_PXE, false, "")
	return err
}

func (req *dhcpRequest) isPXERequest() bool {
	pkt := req.packet
	return dhcp.IsPXERequest(pkt)
}

func (s *Server) validateDHCP(pkt dhcp.Packet) (Machine, Firmware, error) {
	var mach Machine
	var fwtype Firmware
	fwt, err := pkt.ParseOptions().Uint16(dhcp.OptionClientArchitecture)
	if err != nil {
		return mach, fwtype, fmt.Errorf("malformed DHCP option 93 (required for PXE): %s", err)
	}

	// Basic architecture and firmware identification, based purely on
	// the PXE architecture option.
	switch fwt {
	// TODO: complete case 1, 2, 3, 4, 5, 8
	case 0:
		// Intel x86PC
		mach.Arch = ArchIA32
		fwtype = FirmwareX86PC
	case 1:
		// NEC/PC98
		mach.Arch = ArchUnknown
		fwtype = FirmwareUnknown
	case 2:
		// EFI Itanium
		mach.Arch = ArchUnknown
		fwtype = FirmwareUnknown
	case 3:
		// DEC Alpha
		mach.Arch = ArchUnknown
		fwtype = FirmwareUnknown
	case 4:
		// Arc x86
		mach.Arch = ArchUnknown
		fwtype = FirmwareUnknown
	case 5:
		// Intel Lean Client
		mach.Arch = ArchUnknown
		fwtype = FirmwareUnknown
	case 6:
		// EFI IA32
		mach.Arch = ArchIA32
		fwtype = FirmwareEFI32
	case 7:
		// EFI BC
		mach.Arch = ArchX64
		fwtype = FirmwareEFI64
	case 8:
		// EFI Xscale
		mach.Arch = ArchUnknown
		fwtype = FirmwareUnknown
	case 9:
		// EFI x86-64
		mach.Arch = ArchX64
		fwtype = FirmwareEFIBC
	default:
		return mach, 0, fmt.Errorf("unsupported client firmware type '%d'", fwtype)
	}

	guid, _ := pkt.ParseOptions().Bytes(dhcp.OptionClientMachineIdentifier)
	switch len(guid) {
	case 0:
		// A missing GUID is invalid according to the spec, however
		// there are PXE ROMs in the wild that omit the GUID and still
		// expect to boot. The only thing we do with the GUID is
		// mirror it back to the client if it's there, so we might as
		// well accept these buggy ROMs.
	case 17:
		if guid[0] != 0 {
			return mach, 0, errors.New("malformed client GUID (option 97), leading byte must be zero")
		}
	default:
		return mach, 0, errors.New("malformed client GUID (option 97), wrong size")
	}

	mach.MAC = pkt.CHAddr()
	return mach, fwtype, nil
}
