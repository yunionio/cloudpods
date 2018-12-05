package dhcp

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	"yunion.io/x/log"
)

const (
	PXECLIENT = "PXEClient"
)

type ResponseConfig struct {
	OsName        string
	ServerIP      net.IP // OptServerIdentifier 54
	ClientIP      net.IP
	Gateway       net.IP        // OptRouters 3
	Domain        string        // OptDomainName 15
	LeaseTime     time.Duration // OptLeaseTime 51
	RenewalTime   time.Duration // OptRenewalTime 58
	BroadcastAddr net.IP        // OptBroadcastAddr 28
	Hostname      string        // OptHostname 12
	SubnetMask    net.IP        // OptSubnetMask 1
	DNSServer     net.IP        // OptDNSServers
	Routes        interface{}   // TODO: 249 for windows, 121 for linux

	// TFTP config
	BootServer string
	BootFile   string
}

func (conf ResponseConfig) GetHostname() string {
	hostname := conf.Hostname
	if conf.Domain != "" {
		hostname = fmt.Sprintf("%s.%s", hostname, conf.Domain)
	}
	return hostname
}

func GetOptIP(ip net.IP) []byte {
	return []byte(ip.To4())
}

func GetOptTime(d time.Duration) []byte {
	timeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(timeBytes, uint32(d/time.Second))
	return timeBytes
}

func MakeReplyPacket(pkt *Packet, conf *ResponseConfig) (*Packet, error) {
	msgType := MsgOffer
	if pkt.Type == MsgRequest {
		reqAddr, _ := pkt.Options.IP(OptRequestedIP)
		if reqAddr != nil && !conf.ClientIP.Equal(reqAddr) {
			msgType = MsgNack
		} else {
			msgType = MsgAck
		}
	}
	return makeDHCPReplyPacket(pkt, conf, msgType), nil
}

func getPacketVendorClassId(pkt *Packet) string {
	vendorClsId, _ := pkt.Options.String(OptVendorIdentifier)
	return vendorClsId
}

func makeDHCPReplyPacket(pkt *Packet, conf *ResponseConfig, msgType MessageType) *Packet {
	if conf.OsName == "" {
		if vendorClsId := getPacketVendorClassId(pkt); vendorClsId != "" && strings.HasPrefix(vendorClsId, "MSFT ") {
			conf.OsName = "win"
		}
	}
	resp := &Packet{
		Type:          msgType,
		TransactionID: pkt.TransactionID,
		HardwareAddr:  pkt.HardwareAddr,
		RelayAddr:     pkt.RelayAddr,
		ClientAddr:    pkt.ClientAddr,
		ServerAddr:    conf.ServerIP,
		Options:       make(Options),
	}
	if msgType == MsgNack {
		return resp
	}
	resp.YourAddr = conf.ClientIP
	resp.Options[OptServerIdentifier] = GetOptIP(conf.ServerIP)
	if conf.SubnetMask != nil {
		resp.Options[OptSubnetMask] = GetOptIP(conf.SubnetMask)
	}
	if conf.Gateway != nil {
		resp.Options[OptRouters] = GetOptIP(conf.Gateway)
	}
	if conf.Domain != "" {
		resp.Options[OptDomainName] = []byte(conf.Domain)
	}
	if conf.BroadcastAddr != nil {
		resp.Options[OptBroadcastAddr] = GetOptIP(conf.BroadcastAddr)
	}
	if conf.Hostname != "" {
		resp.Options[OptHostname] = []byte(conf.GetHostname())
	}
	if conf.DNSServer != nil {
		resp.Options[OptDNSServers] = GetOptIP(conf.DNSServer)
	}
	if conf.BootServer != "" {
		resp.BootServerName = conf.BootServer
	}
	if conf.BootFile != "" {
		resp.BootFilename = conf.BootFile
		// says the server should identify itself as a PXEClient vendor
		// type, even though it's a server. Strange.
		resp.Options[OptVendorIdentifier] = []byte(PXECLIENT)
	}
	if pkt.Options[OptClientMachineIdentifier] != nil {
		resp.Options[OptClientMachineIdentifier] = pkt.Options[OptClientMachineIdentifier]
	}
	if conf.LeaseTime > 0 {
		resp.Options[OptLeaseTime] = GetOptTime(conf.LeaseTime)
	}
	if conf.RenewalTime > 0 {
		resp.Options[OptRenewalTime] = GetOptTime(conf.RenewalTime)
	}

	// TODO: routes support
	return resp
}

func IsPXERequest(pkt *Packet) bool {
	if pkt.Type != MsgDiscover {
		log.Warningf("packet is %s, not %s", pkt.Type, MsgDiscover)
		return false
	}

	if pkt.Options[OptClientArchitecture] == nil {
		log.Warningf("not a PXE boot request (missing option 93)")
		return false
	}
	return true
}
