package pxe

import (
	"errors"
	"fmt"
	"net"
	"strings"

	dhcp "go.universe.tf/netboot/dhcp4"

	//"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	//o "yunion.io/x/onecloud/pkg/baremetal/options"
	//"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const (
	PXECLIENT = "PXEClient"

	OptClientArchitecture               dhcp.Option = 93
	OptClientNetworkInterfaceIdentifier dhcp.Option = 94
	OptClientMachineIdentifier          dhcp.Option = 97
)

func (s *Server) serveDHCP(conn *dhcp.Conn) error {
	for {
		pkt, intf, err := conn.RecvDHCP()
		if err != nil {
			return fmt.Errorf("Receiving DHCP packet: %s", err)
		}
		if intf == nil {
			return fmt.Errorf("Received DHCP packet with no interface information (this is a violation of dhcp4.Conn's contract)")
		}
		go func() {
			h := &DHCPHandler{}
			resp, err := h.ServeDHCP(pkt)
			if err != nil {
				log.Warningf("[DHCP] handler serve error: %v", err)
				return
			}
			if resp == nil {
				log.Warningf("[DHCP] hander response null packet")
				return
			}
			log.Debugf("[DHCP] send response packet: %s to interface: %#v", resp.DebugString(), intf)
			if err = conn.SendDHCP(resp, intf); err != nil {
				log.Errorf("[DHCP] failed to response packet for %s: %v", pkt.HardwareAddr, err)
				return
			}
		}()
	}
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
	packet                *dhcp.Packet
}

func (h *DHCPHandler) ServeDHCP(pkt *dhcp.Packet) (*dhcp.Packet, error) {
	log.Debugf("[DHCP] request: %s", pkt.DebugString())
	err := h.parsePacket(pkt)
	if err != nil {
		log.Errorf("[DHCP] parse packet error: %v", err)
	}
	log.Infof("======Parse packet end: %#v", h)

	//if h.RelayAddr.String() == "0.0.0.0" {
	//return nil, fmt.Errorf("Request not from a DHCP relay, ignore mac: %s", h.ClientMac)
	//}
	conf, err := h.fetchConfig()
	if err != nil {
		return nil, err
	}
	return h.handleRequest(pkt, conf)
}

func (h *DHCPHandler) handleRequest(pkt *dhcp.Packet, conf *ResponseConfig) (*dhcp.Packet, error) {
	msgType := dhcp.MsgOffer
	if pkt.Type == dhcp.MsgRequest {
		reqAddr, _ := pkt.Options.IP(dhcp.OptRequestedIP)
		if reqAddr != nil && !conf.ClientIP.Equal(reqAddr) {
			msgType = dhcp.MsgNack
		} else {
			msgType = dhcp.MsgAck
		}
	}
	return makeDHCPReplyPacket(pkt, conf, msgType), nil
}

func (h *DHCPHandler) parsePacket(pkt *dhcp.Packet) error {
	h.packet = pkt
	h.ClientAddr = pkt.ClientAddr
	h.ClientMac = pkt.HardwareAddr
	h.RelayAddr = pkt.RelayAddr
	h.Options = pkt.Options

	var (
		vendorClsId string
		cliArch     uint16
		err         error
		netIfIdent  NetworkInterfaceIdent
		cliGuid     string
	)

	for optCode, data := range h.Options {
		switch optCode {
		case dhcp.OptVendorIdentifier:
			vendorClsId, err = h.Options.String(optCode)
		case OptClientArchitecture:
			cliArch, err = h.Options.Uint16(optCode)
		case OptClientNetworkInterfaceIdentifier:
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
		case OptClientMachineIdentifier:
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

func (h *DHCPHandler) fetchConfig() (*ResponseConfig, error) {
	conf := &ResponseConfig{
		ServerIP:   net.IP{10, 168, 0, 1},
		ClientIP:   net.IP{10, 168, 0, 2},
		Gateway:    net.IP{10, 168, 0, 1},
		SubnetMask: net.IP{255, 255, 255, 0},
		DNSServer:  net.IP{8, 8, 8, 8},
		Hostname:   "test12345",
	}
	if h.isPXERequest() {
		conf.BootServer = "10.168.0.1"
		conf.BootFile = "pxelinux.0"
	}
	return conf, nil
}

type ResponseConfig struct {
	OsName        string
	ServerIP      net.IP // OptServerIdentifier 54
	ClientIP      net.IP
	Gateway       net.IP      // OptRouters 3
	Domain        string      // OptDomainName 15
	LeaseTime     uint32      // OptLeaseTime 51
	RenewalTime   uint32      // OptRenewalTime 58
	BroadcastAddr net.IP      // OptBroadcastAddr 28
	Hostname      string      // OptHostname 12
	SubnetMask    net.IP      // OptSubnetMask 1
	DNSServer     net.IP      // OptDNSServers
	Routes        interface{} // TODO: 249 for windows, 121 for linux

	// TFTP config
	BootServer string
	BootFile   string
}

func getPacketVendorClassId(pkt *dhcp.Packet) string {
	vendorClsId, _ := pkt.Options.String(dhcp.OptVendorIdentifier)
	return vendorClsId
}

func makeDHCPReplyPacket(pkt *dhcp.Packet, conf *ResponseConfig, msgType dhcp.MessageType) *dhcp.Packet {
	if conf.OsName == "" {
		if vendorClsId := getPacketVendorClassId(pkt); vendorClsId != "" && strings.HasPrefix(vendorClsId, "MSFT ") {
			conf.OsName = "win"
		}
	}
	resp := &dhcp.Packet{
		Type:          msgType,
		TransactionID: pkt.TransactionID,
		HardwareAddr:  pkt.HardwareAddr,
		RelayAddr:     pkt.RelayAddr,
		ServerAddr:    conf.ServerIP,
		Options:       make(dhcp.Options),
	}
	if msgType == dhcp.MsgNack {
		return resp
	}
	resp.YourAddr = conf.ClientIP
	resp.Options[dhcp.OptServerIdentifier] = conf.ServerIP
	if conf.SubnetMask != nil {
		resp.Options[dhcp.OptSubnetMask] = conf.SubnetMask
	}
	if conf.Gateway != nil {
		resp.Options[dhcp.OptRouters] = conf.Gateway
	}
	if conf.Domain != "" {
		resp.Options[dhcp.OptDomainName] = []byte(conf.Domain)
	}
	if conf.BroadcastAddr != nil {
		resp.Options[dhcp.OptBroadcastAddr] = conf.BroadcastAddr
	}
	if conf.Hostname != "" {
		resp.Options[dhcp.OptHostname] = []byte(conf.Hostname)
	}
	if conf.DNSServer != nil {
		resp.Options[dhcp.OptDNSServers] = conf.DNSServer
	}
	if conf.BootServer != "" {
		resp.BootServerName = conf.BootServer
	}
	if conf.BootFile != "" {
		resp.BootFilename = conf.BootFile
		// says the server should identify itself as a PXEClient vendor
		// type, even though it's a server. Strange.
		resp.Options[dhcp.OptVendorIdentifier] = []byte(PXECLIENT)
	}
	if pkt.Options[OptClientMachineIdentifier] != nil {
		resp.Options[OptClientMachineIdentifier] = pkt.Options[OptClientMachineIdentifier]
	}
	// TODO: routes support
	return resp
}

//func (h *DHCPHandler) findNetworkConf(filterUseIp bool) error {
//params := jsonutils.NewDict()
//if filterUseIp {
//params.Add(jsonutils.NewString(h.RelayAddr.String()), "ip")
//} else {
//params.Add(jsonutils.NewString(
//fmt.Sprintf("guest_gateway.equals(%s)", h.RelayAddr),
//"filter.0"))
//params.Add(jsonutils.NewString(
//fmt.Sprintf("guest_dhcp.equals(%s)", h.RelayAddr),
//"filter.1"))
//params.Add(jsonutils.JSONTrue, "filter_any")
//}
//ret, err := modules.Networks.List(session)
//if err != nil {
//return err
//}
//if len(ret.Data) == 0 {
//if !filterUseIp {
//// use ip filter try again
//return h.findNetworkConf(true)
//}
//return fmt.Errorf("DHCP relay from %s(%s) for %s, find no match network", h.RelayAddr, h.ClientAddr, h.ClientMac)
//}

//network := ret.Data[0]
//if h.isPXERequest() {
//log.Infof("DHCP relay from %s(%s) for %s, find matched networks: %s", h.RelayAddr, h.ClientAddr, h.ClientMac, network)
//}

//if h.isPXERequest() {
//h.findBaremetalConf()
//}
//}

//// findBaremetalConf update or create host
//func (h *DHCPHandler) findBaremetalConf() error {
//params := jsonutils.NewDict()
//params.Add(jsonutils.NewString(h.ClientMac.String()), "any_mac")
//ret, err := modules.Hosts.List(session, params)
//if err != nil {
//return err
//}
//switch len(ret.Data) {
//case 0:
//// found new baremetal, create it if auto register
//if o.Options.AutoRegisterBaremetal {
//h.createBaremetal()
//}
//case 1:
//// already exists, do update
//bmId := ret.Data[0].GetString("id")
//h.updateBaremetal(bmId)
//default:
//return fmt.Errorf("Found %d records match %s", len(ret.Data), h.ClientMac)
//}
//}

//func (h *DHCPHandler) updateBaremetal(id string) error {
//params := jsonutils.NewDict()
//params.Add(jsonutils.NewString(h.ClientMac.String()), "access_mac")
//ret, err := modules.Hosts.Update(session, id, params)
//if err != nil {
//return err
//}
//return nil
//}

func (h *DHCPHandler) isPXERequest() bool {
	pkt := h.packet
	if pkt.Type != dhcp.MsgDiscover {
		log.Warningf("packet is %s, not %s", pkt.Type, dhcp.MsgDiscover)
		return false
	}

	if pkt.Options[93] == nil {
		log.Warningf("not a PXE boot request (missing option 93)")
	}
	return true
}

func (s *Server) validateDHCP(pkt *dhcp.Packet) (Machine, Firmware, error) {
	var mach Machine
	var fwtype Firmware
	fwt, err := pkt.Options.Uint16(93)
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

	guid := pkt.Options[97]
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

	mach.MAC = pkt.HardwareAddr
	return mach, fwtype, nil
}
