package baremetal

import (
	"fmt"
	"net"
	"time"

	"yunion.io/x/pkg/util/netutils"

	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/dhcp"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

func GetNicDHCPConfig(
	n *types.Nic,
	serverIP string,
	hostName string,
	isPxe bool,
	arch uint16,
) (*dhcp.ResponseConfig, error) {
	if n == nil {
		return nil, fmt.Errorf("Nic is nil")
	}
	if n.IpAddr == "" {
		return nil, fmt.Errorf("Nic no ip address")
	}
	ipAddr, err := netutils.NewIPV4Addr(n.IpAddr)
	if err != nil {
		return nil, fmt.Errorf("Parse IP address error: %q", n.IpAddr)
	}

	subnetMask := net.ParseIP(netutils.Masklen2Mask(n.MaskLen).String())

	conf := &dhcp.ResponseConfig{
		ServerIP:      net.ParseIP(serverIP),
		ClientIP:      net.ParseIP(ipAddr.String()),
		Gateway:       net.ParseIP(n.Gateway),
		SubnetMask:    subnetMask,
		BroadcastAddr: net.ParseIP(ipAddr.BroadcastAddr(n.MaskLen).String()),
		DNSServer:     net.ParseIP(n.Dns),
		Domain:        n.Domain,
		OsName:        "Linux",
		Hostname:      hostName,
		// TODO: routes opt
		LeaseTime:   time.Duration(o.Options.DhcpLeaseTime) * time.Second,
		RenewalTime: time.Duration(o.Options.DhcpRenewalTime) * time.Second,
	}

	if isPxe {
		conf.BootServer = serverIP
		switch arch {
		case 7, 9:
			conf.BootFile = "bootx64.efi"
		case 6:
			conf.BootFile = "bootia32.efi"
		default:
			conf.BootFile = "pxelinux.0"
		}
		// TODO: pxeblk bootblock
	}
	return conf, nil
}
