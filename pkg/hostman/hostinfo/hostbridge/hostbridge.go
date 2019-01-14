package hostbridge

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/ovsutils"
)

type IBridgeDriver interface {
	ConfirmToConfig(bool, []string) (bool, error)
	Setup() error
	Exists() bool
	Interfaces() []string

	GetMac() string
}

type SBaseBridgeDriver struct {
	bridge *netutils2.SNetInterface
	ip     string
	inter  *netutils2.SNetInterface
}

func NewBaseBridgeDriver(bridge, inter, ip string) (*SBaseBridgeDriver, error) {
	var bd = new(SBaseBridgeDriver)
	bd.bridge = netutils2.NewNetInterface(bridge)
	if len(inter) > 0 {
		bd.inter = netutils2.NewNetInterface(inter)
		if bd.inter == nil {
			return nil, fmt.Errorf("%s not exists", inter)
		}
		bd.ip = ip
		bd.inter.DisableGso()
	} else if len(ip) > 0 {
		return nil, fmt.Errorf("A bridge without interface must have no IP")
	}
	return bd, nil
}

func (d *SBaseBridgeDriver) GetMac() string {
	if len(d.bridge.Mac) == 0 {
		d.bridge.FetchConfig()
	}
	return d.bridge.Mac
}

func (d *SBaseBridgeDriver) BringupInterface() error {
	var infs = []*netutils2.SNetInterface{d.bridge}
	if d.inter != nil {
		infs = append(infs, d.inter)
	}
	for _, inf := range infs {
		cmd := []string{"ifconfig", inf.String(), "up"}
		if options.HostOptions.TunnelPaddingBytes > 0 {
			cmd = append(cmd, "mtu", fmt.Sprintf("%d", options.HostOptions.TunnelPaddingBytes))
		}
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return err
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) ConfirmToConfig(exists bool, infs []string) (bool, error) {
	if exists {
		d.bridge.FetchConfig()
		if len(d.ip) > 0 {
			if len(d.bridge.Addr) == 0 {
				if len(d.inter.Addr) == 0 {
					return false, fmt.Errorf("Neither %s nor %s owner address %s",
						d.inter, d.bridge, d.ip)
				}
				if d.inter.Addr != d.ip {
					return false, fmt.Errorf("%s!=%s, %s not same as config",
						d.ip, d.inter.Addr, d.inter)
				}
				log.Infof("Bridge address is not configured")
				return false, nil
			}
			if d.bridge.Addr != d.ip {
				return false, fmt.Errorf("%s IP %s!=%s, mismatch", d.bridge, d.bridge.Addr, d.ip)
			}
		} else {
			if d.inter != nil && len(d.inter.Addr) > 0 {
				return false, fmt.Errorf("%s should have no address", d.inter)
			}
			if len(d.bridge.Addr) == 0 {
				return false, nil
			}
			if !d.bridge.IsSecretInterface() {
				return false, fmt.Errorf("%s should have address in 169.254.0.0/16", d.bridge)
			}
		}
		if d.inter != nil && !utils.IsInStringArray(d.inter.String(), infs) {
			log.Infof("Interface %s not in bridge...", d.inter)
			return false, nil
		}
		if err := d.BringupInterface(); err != nil {
			log.Errorln(err)
			return false, err
		}
		return true, nil
	} else {
		if d.bridge.FetchInter() != nil {
			return false, fmt.Errorf("Bridge %s exists, but not created by this driver????", d.bridge)
		}
		if len(d.ip) > 0 && (d.inter == nil || len(d.inter.Addr) == 0) {
			return false, fmt.Errorf("Interface %s not configured", d.inter)
		}
		return false, nil
	}
}

func (d *SBaseBridgeDriver) SetupAddresses(mask net.IPMask) error {
	var addr string
	if len(d.ip) > 0 {
		addr, mask = netutils2.GetSecretInterfaceAddress()
	} else {
		addr = d.ip
	}
	cmd := []string{"ifconfig", d.bridge.String(), addr, "netmask", netutils2.NetBytes2Mask(mask)}
	if options.HostOptions.TunnelPaddingBytes > 0 {
		cmd = append(cmd, "mtu", fmt.Sprintf("%d", options.HostOptions.TunnelPaddingBytes+1500))
	}
	if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
		log.Errorln(err)
		return fmt.Errorf("Failed to bring up bridge %s", d.bridge)
	}
	if d.inter != nil {
		if err := exec.Command("ifconfig", d.inter.String(), "0", "up").Run(); err != nil {
			log.Errorln(err)
			return fmt.Errorf("Failed to bring up interface %s", d.inter)
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) SetupSlaveAddresses(slaveAddrs [][]string) error {
	for _, slaveAddr := range slaveAddrs {
		cmd := []string{"ip", "address", "del",
			fmt.Sprintf("%s/%s", slaveAddr[0], slaveAddr[1]), "dev", d.inter.String()}
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			log.Errorln("Failed to remove slave address from interface %s: %s", d.inter, err)
		}

		cmd = []string{"ip", "address", "add",
			fmt.Sprintf("%s/%s", slaveAddr[0], slaveAddr[1]), "dev", d.bridge.String()}
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("Failed to remove slave address from interface %s: %s", d.bridge, err)
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) SetupRoutes(routes [][]string) error {
	for _, r := range routes {
		var cmd []string
		if r[2] == "0.0.0.0" {
			cmd = []string{"route", "add", "default", "gw", r[1], "dev", d.bridge.String()}
		} else {
			cmd = []string{"route", "add", "-net", r[0], "netmask", r[2], "gw", r[1], "dev", d.bridge.String()}
		}
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			log.Errorln(err)
			return fmt.Errorf("Failed to add slave address to bridge %s", d.bridge)
		}
	}
	return nil
}

type SOVSBridgeDriver struct {
	SBaseBridgeDriver
}

func (o *SOVSBridgeDriver) Exists() bool {
	data, err := exec.Command("ovs-vsctl", "list-br").Output()
	if err != nil {
		log.Errorln(err)
		return false
	}
	for _, d := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(d) == o.bridge.String() {
			return true
		}
	}
	return false
}

func (o *SOVSBridgeDriver) Interfaces() []string {
	data, err := exec.Command("ovs-vsctl", "list-ifaces", o.bridge.String()).Output()
	if err != nil {
		log.Errorln(err)
		return nil
	}

	var infs = make([]string, 0)
	for _, d := range strings.Split(string(data), "\n") {
		if len(strings.TrimSpace(d)) > 0 {
			infs = append(infs, strings.TrimSpace(d))
		}
	}
	return infs
}

func (o *SOVSBridgeDriver) Setup() error {
	var routes [][]string
	var slaveAddrs [][]string
	if o.inter != nil && len(o.inter.Addr) > 0 {
		routes = o.inter.GetRoutes(true)
		slaveAddrs = o.inter.GetSlaveAddresses()
	}
	if !o.Exists() {
		if err := o.SetupBridgeDev(); err != nil {
			return err
		}
	}

	if len(o.bridge.Addr) == 0 {
		if len(o.ip) > 0 {
			if err := o.SetupAddresses(o.inter.Mask); err != nil {
				return err
			}
			if len(slaveAddrs) > 0 {
				if err := o.SetupSlaveAddresses(slaveAddrs); err != nil {
					return err
				}
			}
			if len(routes) > 0 {
				if err := o.SetupRoutes(routes); err != nil {
					return err
				}
			}
		} else {
			if err := o.SetupAddresses(nil); err != nil {
				return err
			}
		}
	}

	return o.BringupInterface()
}

func (o *SOVSBridgeDriver) SetupBridgeDev() error {
	if !o.Exists() {
		return exec.Command("ovs-vsctl", "--", "--may-exist", "add-br", o.bridge.String()).Run()
	}
	return nil
}

func CleanOvsBridge() {
	ovsutils.CleanAllHiddenPorts()
}

func NewOVSBridgeDriver(bridge, inter, ip string) (*SOVSBridgeDriver, error) {
	base, err := NewBaseBridgeDriver(bridge, inter, ip)
	if err != nil {
		return nil, err
	}
	return &SOVSBridgeDriver{*base}, nil
}

func NewDriver(bridgeDriver, bridge, inter, ip string) (IBridgeDriver, error) {
	if bridgeDriver == "openvswitch" {
		return NewOVSBridgeDriver(bridge, inter, ip)
	} else {
		return nil, fmt.Errorf("Not Implentment")
	}
}

func CleanDeletedPorts() {
	if options.HostOptions.BridgeDriver == "openvswitch" {
		CleanOvsBridge()
	}
}
