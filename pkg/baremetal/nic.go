package baremetal

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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

	routes := make([][]string, 0)
	for _, route := range n.Routes {
		routes = append(routes, []string{route[0], route[1]})
	}

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
		Routes:        routes,
		LeaseTime:     time.Duration(o.Options.DhcpLeaseTime) * time.Second,
		RenewalTime:   time.Duration(o.Options.DhcpRenewalTime) * time.Second,
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
		pxePath := filepath.Join(o.Options.TftpRoot, conf.BootFile)
		if f, err := os.Open(pxePath); err != nil {
			return nil, err
		} else {
			if info, err := f.Stat(); err != nil {
				return nil, err
			} else {
				pxeSize := info.Size()
				pxeBlk := pxeSize / 512
				if pxeSize > pxeBlk*512 {
					pxeBlk += 1
				}
				conf.BootBlock = uint16(pxeBlk)
			}
		}
	}
	return conf, nil
}
