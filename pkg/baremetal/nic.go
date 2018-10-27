package baremetal

import (
	"fmt"
	"net"

	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/baremetal/pxe"
	"yunion.io/x/onecloud/pkg/baremetal/types"
)

func GetNicDHCPConfig(
	n *types.Nic,
	serverIP string,
	hostName string,
	isPxe bool,
	arch uint16,
) (*pxe.ResponseConfig, error) {
	ipAddr, err := netutils.NewIPV4Addr(n.IpAddr)
	if err != nil {
		return nil, fmt.Errorf("Parse IP address error: %q", n.IpAddr)
	}

	subnetMask := net.ParseIP(netutils.Masklen2Mask(n.MaskLen).String())

	conf := &pxe.ResponseConfig{
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
