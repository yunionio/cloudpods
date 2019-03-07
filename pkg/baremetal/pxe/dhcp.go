package pxe

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/models"
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
	ClientMac             net.HardwareAddr // client nic mac
	ClientAddr            net.IP           // IP address from DHCP client
	RelayAddr             net.IP           // IP address of DHCP relay agent
	Options               dhcp.Options     // dhcp packet options
	VendorClassId         string
	ClientArch            uint16
	NetworkInterfaceIdent NetworkInterfaceIdent
	ClientGuid            string
	packet                dhcp.Packet

	// baremetal manager
	baremetalManager IBaremetalManager
	// baremetal instance
	baremetalInstance IBaremetalInstance
	// cloud network config
	netConfig *types.SNetworkConfig
}

func (h *DHCPHandler) ServeDHCP(pkt dhcp.Packet, _ *net.UDPAddr, _ *net.Interface) (dhcp.Packet, error) {
	//log.V(4).Debugf("[DHCP] request: %s", pkt.DebugString())
	err := h.parsePacket(pkt)
	if err != nil {
		log.Errorf("[DHCP] parse packet error: %v", err)
	}
	log.V(4).Debugf("[DHCP] parse packet end: %#v", h)

	if h.RelayAddr.String() == "0.0.0.0" {
		return nil, fmt.Errorf("Request not from a DHCP relay, ignore mac: %s", h.ClientMac)
	}
	conf, err := h.fetchConfig()
	if err != nil {
		return nil, err
	}
	if conf == nil {
		return nil, fmt.Errorf("Empty packet config")
	}
	return dhcp.MakeReplyPacket(pkt, conf)
}

func (h *DHCPHandler) parsePacket(pkt dhcp.Packet) error {
	h.packet = pkt
	h.ClientAddr = pkt.CIAddr()
	h.ClientMac = pkt.CHAddr()
	h.RelayAddr = pkt.RelayAddr()
	h.Options = pkt.ParseOptions()

	var (
		vendorClsId string
		cliArch     uint16
		err         error
		netIfIdent  NetworkInterfaceIdent
		cliGuid     string
	)

	for optCode, data := range h.Options {
		switch optCode {
		case dhcp.OptionVendorClassIdentifier:
			vendorClsId, err = h.Options.String(optCode)
		case dhcp.OptionClientArchitecture:
			cliArch, err = h.Options.Uint16(optCode)
		case dhcp.OptionClientNetworkInterfaceIdentifier:
			netIfIdentBs, err := h.Options.Bytes(optCode)
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
			cliGuid, err = h.Options.String(optCode)
		}
		if err != nil {
			log.Errorf("[DHCP] parse vendor option %d error: %v", optCode, err)
		}
	}
	h.VendorClassId = vendorClsId
	h.ClientArch = cliArch
	h.NetworkInterfaceIdent = netIfIdent
	h.ClientGuid = cliGuid
	return err
}

func (h *DHCPHandler) fetchConfig() (*dhcp.ResponseConfig, error) {
	// 1. find_network_conf
	netConf, err := h.findNetworkConf(false)
	if err != nil {
		return nil, err
	}
	h.netConfig = netConf

	// TODO: set cache for netConf
	//
	if h.isPXERequest() {
		// handle PXE DHCP request
		log.Infof("DHCP relay from %s(%s) for %s, find matched networks: %#v", h.RelayAddr, h.ClientAddr, h.ClientMac, netConf)
		bmDesc, err := h.createOrUpdateBaremetal()
		if err != nil {
			return nil, err
		}
		err = h.doInitBaremetalAdminNetif(bmDesc)
		if err != nil {
			return nil, err
		}
		if h.baremetalInstance.NeedPXEBoot() {
			return h.baremetalInstance.GetPXEDHCPConfig(h.ClientArch)
		}
		// ignore
		log.Warningf("No need to pxeboot, ignore the request ...(mac:%s guid:%s)", h.ClientMac, h.ClientGuid)
		return nil, nil
	} else {
		// handle normal DHCP request
		bmInstance := h.baremetalManager.GetBaremetalByMac(h.ClientMac)
		if bmInstance == nil {
			// options.EnableGeneralGuestDhcp
			// cloud be an instance not served by a host-server
			// from guestdhcp import GuestDHCPHelperTask
			// task = GuestDHCPHelperTask(self)
			// task.start()
			return nil, nil
		}
		h.baremetalInstance = bmInstance
		ipmiNic := h.baremetalInstance.GetIPMINic(h.ClientMac)
		if ipmiNic != nil && ipmiNic.Mac == h.ClientMac.String() {
			err = h.baremetalInstance.InitAdminNetif(h.ClientMac, h.netConfig, types.NIC_TYPE_IPMI, models.NETWORK_TYPE_IPMI)
			if err != nil {
				return nil, err
			}
		} else {
			err = h.baremetalInstance.RegisterNetif(h.ClientMac, h.netConfig)
			if err != nil {
				log.Errorf("RegisterNetif error: %v", err)
				return nil, err
			}
		}
		return h.baremetalInstance.GetDHCPConfig(h.ClientMac)
	}
}

func (h *DHCPHandler) findNetworkConf(filterUseIp bool) (*types.SNetworkConfig, error) {
	params := jsonutils.NewDict()
	if filterUseIp {
		params.Add(jsonutils.NewString(h.RelayAddr.String()), "ip")
	} else {
		params.Add(jsonutils.NewString(
			fmt.Sprintf("guest_gateway.equals(%s)", h.RelayAddr)),
			"filter.0")
		params.Add(jsonutils.NewString(
			fmt.Sprintf("guest_dhcp.equals(%s)", h.RelayAddr)),
			"filter.1")
		params.Add(jsonutils.JSONTrue, "filter_any")
	}
	params.Add(jsonutils.JSONTrue, "is_on_premise")
	session := h.baremetalManager.GetClientSession()
	ret, err := modules.Networks.List(session, params)
	if err != nil {
		return nil, err
	}
	if len(ret.Data) == 0 {
		if !filterUseIp {
			// use ip filter try again
			return h.findNetworkConf(true)
		}
		return nil, fmt.Errorf("DHCP relay from %s(%s) for %s, find no match network", h.RelayAddr, h.ClientAddr, h.ClientMac)
	}

	network := types.SNetworkConfig{}
	err = ret.Data[0].Unmarshal(&network)
	return &network, err
}

// createOrUpdateBaremetal create or update baremetal by client MAC
func (h *DHCPHandler) createOrUpdateBaremetal() (jsonutils.JSONObject, error) {
	session := h.baremetalManager.GetClientSession()
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(models.HOST_TYPE_BAREMETAL), "host_type")
	params.Add(jsonutils.NewString(h.ClientMac.String()), "any_mac")
	params.Add(jsonutils.JSONTrue, "is_baremetal")
	ret, err := modules.Hosts.List(session, params)
	if err != nil {
		return nil, err
	}
	switch len(ret.Data) {
	case 0:
		// found new baremetal, create it if auto register
		if o.Options.AutoRegisterBaremetal {
			return h.createBaremetal()
		}
	case 1:
		// already exists, do update
		bmId, err := ret.Data[0].GetString("id")
		if err != nil {
			return nil, err
		}
		return h.updateBaremetal(bmId)
	}
	return nil, fmt.Errorf("Found %d records match %s", len(ret.Data), h.ClientMac)
}

func (h *DHCPHandler) createBaremetal() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	mac := h.ClientMac.String()
	zoneId := h.baremetalManager.GetZoneId()
	name := fmt.Sprintf("BM%s", strings.Replace(mac, ":", "", -1))
	params.Add(jsonutils.NewString(name), "name")
	params.Add(jsonutils.NewString(mac), "access_mac")
	params.Add(jsonutils.NewString("baremetal"), "host_type")
	params.Add(jsonutils.JSONTrue, "is_baremetal")
	params.Add(jsonutils.NewString(zoneId), "zone_id")
	session := h.baremetalManager.GetClientSession()
	desc, err := modules.Hosts.Create(session, params)
	if err != nil {
		return nil, err
	}
	return desc, nil
}

func (h *DHCPHandler) updateBaremetal(id string) (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(h.ClientMac.String()), "access_mac")
	params.Add(jsonutils.NewString(h.baremetalManager.GetZoneId()), "zone_id")
	// params.Add(jsonutils.NewString("baremetal"), "host_type")
	params.Add(jsonutils.JSONTrue, "is_baremetal")
	session := h.baremetalManager.GetClientSession()
	desc, err := modules.Hosts.Update(session, id, params)
	if err != nil {
		return nil, err
	}
	return desc, nil
}

func (h *DHCPHandler) doInitBaremetalAdminNetif(desc jsonutils.JSONObject) error {
	var err error
	h.baremetalInstance, err = h.baremetalManager.AddBaremetal(desc)
	if err != nil {
		return err
	}
	err = h.baremetalInstance.InitAdminNetif(h.ClientMac, h.netConfig, types.NIC_TYPE_ADMIN, models.NETWORK_TYPE_PXE)
	return err
}

func (h *DHCPHandler) isPXERequest() bool {
	pkt := h.packet
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
