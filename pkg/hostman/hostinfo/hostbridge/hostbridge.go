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

package hostbridge

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/iproute2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type IBridgeDriver interface {
	MigrateSlaveConfigs(IBridgeDriver) error
	ConfirmToConfig() (bool, string, error)
	GetMac() string
	GetVlanId() int
	FetchConfig()
	Setup(IBridgeDriver) error
	SetupAddresses() error
	SetupSlaveAddresses([]netutils2.SNicAddress) error
	SetupRoutes(routes []iproute2.RouteSpec, add bool, dev string) error
	BringupInterface() error

	Exists() (bool, error)
	Interfaces() ([]string, error)
	WarmupConfig() error
	CleanupConfig()
	SetupBridgeDev() error
	SetupInterface() error
	PersistentConfig() error
	DisableDHCPClient() (bool, error)

	GenerateIfupScripts(scriptPath string, nic *desc.SGuestNetwork, isVolatileHost bool) error
	GenerateIfdownScripts(scriptPath string, nic *desc.SGuestNetwork, isVolatileHost bool) error
	RegisterHostlocalServer(mac, ip string) error

	getUpScripts(nic *desc.SGuestNetwork, isVolatileHost bool) (string, error)
	getDownScripts(nic *desc.SGuestNetwork, isVolatileHost bool) (string, error)

	OnVolatileGuestResume(nic *desc.SGuestNetwork) error

	Bridge() string

	IsV4Only() bool
}

type SBaseBridgeDriver struct {
	bridge *netutils2.SNetInterface
	ip     string
	ip6    string
	inter  *netutils2.SNetInterface

	maskLen  int
	mask6Len int

	drv IBridgeDriver
}

func NewBaseBridgeDriver(bridge, inter, ip string, maskLen int, ip6 string, mask6Len int) (*SBaseBridgeDriver, error) {
	var bd = new(SBaseBridgeDriver)
	bd.bridge = netutils2.NewNetInterface(bridge)
	if len(inter) > 0 {
		bd.inter = netutils2.NewNetInterface(inter)
		if !bd.inter.Exist() {
			return nil, fmt.Errorf("%s not exists", inter)
		}
		bd.ip = ip
		bd.maskLen = maskLen
		bd.ip6 = ip6
		bd.mask6Len = mask6Len
		var enableGso bool
		if len(options.HostOptions.EthtoolEnableGsoInterfaces) > 0 {
			if utils.IsInStringArray(bridge, options.HostOptions.EthtoolEnableGsoInterfaces) ||
				utils.IsInStringArray(inter, options.HostOptions.EthtoolEnableGsoInterfaces) {
				enableGso = true
			} else {
				enableGso = false
			}
		} else if len(options.HostOptions.EthtoolDisableGsoInterfaces) > 0 {
			if utils.IsInStringArray(bridge, options.HostOptions.EthtoolDisableGsoInterfaces) ||
				utils.IsInStringArray(inter, options.HostOptions.EthtoolDisableGsoInterfaces) {
				enableGso = false
			} else {
				enableGso = true
			}
		} else {
			enableGso = options.HostOptions.EthtoolEnableGso
		}
		bd.inter.SetupGso(enableGso)
	} else if len(ip) > 0 || len(ip6) > 0 {
		return nil, fmt.Errorf("A bridge without interface must have no IP")
	}
	return bd, nil
}

func (d *SBaseBridgeDriver) FetchConfig() {
	d.bridge.FetchConfig2(d.ip, d.ip6)
	d.inter.FetchConfig()
}

func (d *SBaseBridgeDriver) GetMac() string {
	if len(d.inter.GetMac()) == 0 {
		d.inter.FetchConfig()
	}
	return d.inter.GetMac()
}

func (d *SBaseBridgeDriver) GetVlanId() int {
	if len(d.inter.GetMac()) == 0 {
		d.inter.FetchConfig()
	}
	return d.inter.VlanId
}

func (d *SBaseBridgeDriver) Bridge() string {
	return d.bridge.String()
}

func (d *SBaseBridgeDriver) PersistentConfig() error {
	return nil
}

func (d *SBaseBridgeDriver) BringupInterface() error {
	var infs = []*netutils2.SNetInterface{d.bridge}
	if d.inter != nil {
		infs = append(infs, d.inter)
	}
	for _, inf := range infs {
		l := iproute2.NewLink(inf.String())
		l.Up()
		if options.HostOptions.TunnelPaddingBytes > 0 {
			mtu := int(1500 + options.HostOptions.TunnelPaddingBytes)
			l.MTU(mtu)
		}
		if err := l.Err(); err != nil {
			return err
		}
	}
	return nil
}

func trySetupSlaveAddressesRoutes(o IBridgeDriver, migrateAddrs []netutils2.SNicAddress, delRoutes []iproute2.RouteSpec, migrateRoutes []iproute2.RouteSpec) error {
	if len(migrateAddrs) > 0 {
		tried := 0
		const MAX_TRIES = 4
		errs := make([]error, 0)
		for tried < MAX_TRIES {
			if err := o.SetupSlaveAddresses(migrateAddrs); err != nil {
				errs = append(errs, err)
				log.Errorf("SetupSlaveAddresses fail: %s", err)
				tried += 1
				if tried >= MAX_TRIES {
					return errors.Wrap(errors.NewAggregate(errs), "SetupSlaveAddresses")
				} else {
					time.Sleep(time.Duration(tried) * time.Second)
				}
			} else {
				break
			}
		}
	}
	if len(delRoutes) > 0 {
		tried := 0
		const MAX_TRIES = 4
		errs := make([]error, 0)
		for {
			if err := o.SetupRoutes(delRoutes, false, o.Bridge()); err != nil {
				errs = append(errs, err)
				log.Errorf("delRoutes fail: %s", err)
				tried += 1
				if tried >= MAX_TRIES {
					return errors.Wrap(errors.NewAggregate(errs), "DeleteRoutes")
				} else {
					time.Sleep(time.Duration(tried) * time.Second)
				}
			} else {
				break
			}
		}
	}
	if len(migrateRoutes) > 0 {
		tried := 0
		const MAX_TRIES = 4
		errs := make([]error, 0)
		for {
			if err := o.SetupRoutes(migrateRoutes, true, o.Bridge()); err != nil {
				errs = append(errs, err)
				log.Errorf("SetupRoutes fail: %s", err)
				tried += 1
				if tried >= MAX_TRIES {
					return errors.Wrap(errors.NewAggregate(errs), "SetupRoutes")
				} else {
					time.Sleep(time.Duration(tried) * time.Second)
				}
			} else {
				break
			}
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) MigrateSlaveConfigs(o IBridgeDriver) error {
	if d.inter != nil {
		migrateAddrs := make([]netutils2.SNicAddress, 0)
		migrateRoutes := make([]iproute2.RouteSpec, 0)
		delRoutes := make([]iproute2.RouteSpec, 0)
		{
			currentRoutes := d.bridge.GetRouteSpecs()
			currentSlaves := d.bridge.GetSlaveAddresses()
			routes := d.inter.GetRouteSpecs()
			slaveAddrs := d.inter.GetSlaveAddresses()

			log.Infof("to migrate routes: %s slaveAddress: %s", jsonutils.Marshal(routes), jsonutils.Marshal(slaveAddrs))

			for i := range slaveAddrs {
				if strings.HasPrefix(slaveAddrs[i].Addr, "fe80:") || strings.HasPrefix(slaveAddrs[i].Addr, "169.254.") {
					// skip link local address
					continue
				}
				if slaveAddrs[i].Addr == d.bridge.Addr || slaveAddrs[i].Addr == d.bridge.Addr6 {
					continue
				}
				find := false
				for j := range currentSlaves {
					if slaveAddrs[i].Addr == currentSlaves[j].Addr && slaveAddrs[i].MaskLen == currentSlaves[j].MaskLen {
						find = true
						break
					}
				}
				if !find {
					// need to migrate address
					migrateAddrs = append(migrateAddrs, slaveAddrs[i])
				}
			}

			for i := range routes {
				find := false
				for j := range currentRoutes {
					log.Infof("new %s(%d,%s) current %s(%d,%s)", routes[i].Dst.String(), routes[i].Table, routes[i].Gw.String(), currentRoutes[j].Dst.String(), currentRoutes[j].Table, currentRoutes[j].Gw.String())
					if routes[i].Dst.String() == currentRoutes[j].Dst.String() && routes[i].Table == currentRoutes[j].Table {
						if routes[i].Gw.String() != currentRoutes[j].Gw.String() {
							// need to replace
							delRoutes = append(delRoutes, currentRoutes[j])
							migrateRoutes = append(migrateRoutes, routes[i])
						}
						find = true
						break
					}
				}
				if !find {
					for j := range slaveAddrs {
						if routes[i].Dst.String() == addr2Prefix(slaveAddrs[j].Addr, slaveAddrs[j].MaskLen) {
							find = true
							break
						}
					}
				}
				if !find {
					// need to migrate route
					migrateRoutes = append(migrateRoutes, routes[i])
				}
			}
		}
		log.Infof("to migrate routes: %s slaveAddress: %s delRoutes: %s", jsonutils.Marshal(migrateRoutes), jsonutils.Marshal(migrateAddrs), jsonutils.Marshal(delRoutes))
		{
			err := trySetupSlaveAddressesRoutes(o, migrateAddrs, delRoutes, migrateRoutes)
			if err != nil {
				return errors.Wrap(err, "trySetupSlaveAddressesRoutes")
			}
		}
		{
			tryUnmanageInterface(d.inter.String())
			err := d.inter.FlushAddrs()
			if err != nil {
				return errors.Wrap(err, "ClearAddrs")
			}
		}
		{
			routes := d.inter.GetRouteSpecs()
			if len(routes) > 0 {
				rt := iproute2.NewRoute(d.inter.String())
				for i := range routes {
					rt = rt.DelByCidr(routes[i].Dst.String())
				}
				err := rt.Err()
				if err != nil {
					return errors.Wrap(err, "Clear rotues")
				}
			}
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) ConfirmToConfig() (bool, string, error) {
	exist, err := d.drv.Exists()
	if err != nil {
		return false, "", err
	}
	if exist {
		d.bridge.FetchConfig2(d.ip, d.ip6)
		if len(d.ip) > 0 {
			if len(d.bridge.Addr) == 0 {
				log.Infof("bridge %s has no ip assignment initially", d.bridge)
				if len(d.inter.Addr) == 0 {
					return false, "", fmt.Errorf("Neither %s nor %s owner address %s", d.inter, d.bridge, d.ip)
				}
				if d.inter.Addr != d.ip {
					return false, "", fmt.Errorf("%s!=%s, %s not same as config", d.ip, d.inter.Addr, d.inter)
				}
				log.Infof("Bridge address is not configured")
				return false, fmt.Sprintf("bridge %s address is empty, should be configured", d.bridge), nil
			} else {
				log.Infof("bridge %s already has ip %s", d.bridge, d.bridge.Addr)
			}
			if d.bridge.Addr != d.ip {
				return false, "", fmt.Errorf("bridge %s IP %s is not expected IP %s, mismatch", d.bridge, d.bridge.Addr, d.ip)
			}
		} else {
			if d.inter != nil && len(d.inter.Addr) > 0 {
				return false, "", fmt.Errorf("interface %s should have no address", d.inter)
			}
			if len(d.bridge.Addr) > 0 {
				return false, "", fmt.Errorf("bridge %s (%s) should have no ipv4 address", d.bridge, d.bridge.Addr)
			}
			if !d.bridge.IsSecretInterface() {
				return false, "", fmt.Errorf("%s should have address in 169.254.0.0/16", d.bridge)
			}
		}
		if len(d.ip6) > 0 {
			if len(d.bridge.Addr6) == 0 {
				log.Infof("bridge %s has no ipv6 assignment initially", d.bridge)
				if len(d.inter.Addr6) == 0 {
					return false, "", fmt.Errorf("neither interface%s nor bridge %s owner ipv6 address %s", d.inter, d.bridge, d.ip6)
				}
				if d.inter.Addr6 != d.ip6 {
					return false, "", fmt.Errorf("interface %s IP %s is not %s as configured", d.inter, d.inter.Addr6, d.ip6)
				}
				log.Infof("Bridge ipv6 address is not configured")
				return false, fmt.Sprintf("bridge %s ipv6 address is empty, should be configured", d.bridge), nil
			} else {
				log.Infof("bridge %s already has ipv6 address %s", d.bridge, d.bridge.Addr6)
			}
			if d.bridge.Addr6 != d.ip6 {
				return false, "", fmt.Errorf("%s IP %s!=%s, mismatch", d.bridge, d.bridge.Addr6, d.ip6)
			}
		} else {
			if d.inter != nil && len(d.inter.Addr6) > 0 {
				return false, "", fmt.Errorf("interface %s should have no ipv6 address", d.inter)
			}
			if len(d.bridge.Addr6) != 0 {
				return false, "", fmt.Errorf("bridge %s (%s) shoud have no ipv6 address", d.bridge, d.bridge.Addr6)
			}
			if !d.bridge.IsSecretInterface6() {
				log.Warningf("bridge %s have no link local address in fe80::/10", d.bridge)
			}
		}
		infs, err := d.drv.Interfaces()
		if err != nil {
			return false, "", fmt.Errorf("get all interfaces of bridge %s failed: %s", d.bridge, err)
		}
		if d.inter != nil && !utils.IsInStringArray(d.inter.String(), infs) {
			log.Infof("Interface %s not in bridge...", d.inter)
			return false, fmt.Sprintf("interface %s not in bridge %s", d.inter, d.bridge), nil
		}
		if err := d.BringupInterface(); err != nil {
			log.Errorln(err)
			return false, "", fmt.Errorf("bring up bridge %s failed: %s", d.bridge, err)
		}
		return true, "", nil
	} else {
		if d.bridge.FetchInter() != nil {
			return false, "", fmt.Errorf("bridge %s exists, but not created by this driver????", d.bridge)
		}
		if len(d.ip) > 0 && (d.inter == nil || len(d.inter.Addr) == 0) {
			return false, "", fmt.Errorf("interface %s not configured", d.inter)
		}
		if len(d.ip6) > 0 && (d.inter == nil || len(d.inter.Addr6) == 0) {
			return false, "", fmt.Errorf("interface %s ipv6 not configured", d.inter)
		}
		return false, fmt.Sprintf("bridge %s does not exist", d.bridge), nil
	}
}

func tryUnmanageInterface(ifname string) {
	// ensure the device is not managed by NetworkManager
	// nmcli dev set <interface_name> managed no
	output, err := procutils.NewRemoteCommandAsFarAsPossible("nmcli", "dev", "set", ifname, "managed", "no").Output()
	if err != nil {
		log.Errorf("run cmd: nmcli dev set %s managed no, output: %s, error: %s", ifname, string(output), err)
	}
}

func (d *SBaseBridgeDriver) SetupAddresses() error {
	br := d.bridge.String()
	if d.inter != nil {
		// first shutdown the origin interface
		ifname := d.inter.String()
		tryUnmanageInterface(ifname)
		if err := d.inter.FlushAddrs(); err != nil {
			return errors.Wrapf(err, "bridge %s slave ifname: %s flush addrs fail", br, ifname)
		}
	}
	{
		// assign address to bridge interface
		var (
			addr    string
			masklen int
		)
		if len(d.ip) == 0 {
			addr, masklen = netutils2.GetSecretInterfaceAddress()
		} else {
			addr = d.ip
			masklen = d.maskLen
		}
		addrStr := []string{}
		addrStr = append(addrStr, fmt.Sprintf("%s/%d", addr, masklen))
		if len(d.ip6) > 0 {
			addrStr = append(addrStr, fmt.Sprintf("%s/%d", d.ip6, d.mask6Len))
		}
		if err := iproute2.NewAddress(br, addrStr...).Exact().Err(); err != nil {
			return errors.Wrapf(err, "set bridge %s address", br)
		}
	}
	{
		// bring up the bridge interface
		brLink := iproute2.NewLink(br).Up()
		if options.HostOptions.TunnelPaddingBytes > 0 {
			mtu := 1500 + int(options.HostOptions.TunnelPaddingBytes)
			brLink.MTU(mtu)
		}
		if err := brLink.Err(); err != nil {
			return errors.Wrapf(err, "setting bridge %s up", br)
		}
		if d.inter != nil {
			ifname := d.inter.String()
			// bring up the origin interface
			ethLink := iproute2.NewLink(ifname).Up()
			if options.HostOptions.TunnelPaddingBytes > 0 {
				mtu := 1500 + int(options.HostOptions.TunnelPaddingBytes)
				ethLink.MTU(mtu)
			}
			if err := ethLink.Err(); err != nil {
				return errors.Wrapf(err, "setting origin interface %s up", d.inter.String())
			}
			if err := iproute2.NewAddress(ifname).Exact().Err(); err != nil {
				return errors.Wrapf(err, "clear interface %s address", ifname)
			}
		}
	}

	return nil
}

func (d *SBaseBridgeDriver) SetupSlaveAddresses(slaveAddrs []netutils2.SNicAddress) error {
	br := d.bridge.String()
	addrs := make([]string, len(slaveAddrs))
	for i, slaveAddr := range slaveAddrs {
		addrs[i] = fmt.Sprintf("%s/%d", slaveAddr.Addr, slaveAddr.MaskLen)
	}
	if err := iproute2.NewAddress(br, addrs...).Add().Err(); err != nil {
		return errors.Wrap(err, "move secondary addresses to bridge interface")
	}
	return nil
}

func (d *SBaseBridgeDriver) SetupRoutes(routespecs []iproute2.RouteSpec, add bool, dev string) error {
	for i := 0; i < len(routespecs); i++ {
		errs := []error{}
		routespec := routespecs[i]
		var cmd []string
		if regutils.MatchCIDR6(routespec.Dst.String()) {
			cmd = append(cmd, "-6")
		}
		cmd = append(cmd, "route")
		if add {
			cmd = append(cmd, "add")
		} else {
			cmd = append(cmd, "del")
		}
		cmd = append(cmd, routespec.Dst.String())
		if routespec.Gw != nil {
			cmd = append(cmd, "via", routespec.Gw.String())
		}
		cmd = append(cmd, "dev", dev)

		output, err := procutils.NewRemoteCommandAsFarAsPossible("ip", cmd...).Output()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "run cmd: ip %s, output: %s", strings.Join(cmd, " "), output))
			if add {
				cmd = append(cmd, "onlink")
				if output, err := procutils.NewRemoteCommandAsFarAsPossible("ip", cmd...).Output(); err != nil {
					errs = append(errs, errors.Wrapf(err, "run cmd: ip %s, output: %s", strings.Join(cmd, " "), output))
					return errors.Wrapf(errors.NewAggregate(errs), "setup route %s", routespec.String())
				}
			} else {
				return errors.Wrapf(errors.NewAggregate(errs), "setup route %s", routespec.String())
			}
		}
	}
	return nil
}

func addr2Prefix(addrStr string, maskLen int) string {
	if regutils.MatchIP6Addr(addrStr) {
		v6Addr, _ := netutils.NewIPV6Addr(addrStr)
		netAddr := v6Addr.NetAddr(uint8(maskLen))
		return fmt.Sprintf("%s/%d", netAddr.String(), maskLen)
	} else {
		v4Addr, _ := netutils.NewIPV4Addr(addrStr)
		netAddr := v4Addr.NetAddr(int8(maskLen))
		return fmt.Sprintf("%s/%d", netAddr.String(), maskLen)
	}
}

/*func addr2Prefix2(addrStr string, mask net.IPMask) string {
	if regutils.MatchIP6Addr(addrStr) {
		v6Addr, _ := netutils.NewIPV6Addr(addrStr)
		maskLen, _ := mask.Size()
		netAddr := v6Addr.NetAddr(uint8(maskLen))
		return fmt.Sprintf("%s/%d", netAddr.String(), maskLen)
	} else {
		v4Addr, _ := netutils.NewIPV4Addr(addrStr)
		maskLen, _ := mask.Size()
		netAddr := v4Addr.NetAddr(int8(maskLen))
		return fmt.Sprintf("%s/%d", netAddr.String(), maskLen)
	}
}*/

func (d *SBaseBridgeDriver) Setup(o IBridgeDriver) error {
	var routes []iproute2.RouteSpec
	var slaveAddrs []netutils2.SNicAddress
	if d.inter != nil && (len(d.inter.Addr) > 0 || len(d.inter.Addr6) > 0) {
		routes = d.inter.GetRouteSpecs()
		slaveAddrs = d.inter.GetSlaveAddresses()
		log.Infof("to migrate routes: %s slaveAddress: %s", jsonutils.Marshal(routes), jsonutils.Marshal(slaveAddrs))
	}
	exist, err := o.Exists()
	if err != nil {
		return errors.Wrap(err, "Exists")
	}
	if !exist {
		if err := o.SetupBridgeDev(); err != nil {
			return errors.Wrap(err, "SetupBridgeDev")
		}
	}

	infs, err := o.Interfaces()
	if err != nil {
		return errors.Wrap(err, "Interfaces")
	}
	if d.inter != nil && !utils.IsInStringArray(d.inter.String(), infs) {
		if err := o.SetupInterface(); err != nil {
			return errors.Wrap(err, "SetupInterface")
		}
	}
	if len(d.bridge.Addr) == 0 && len(d.bridge.Addr6) == 0 {
		// need to do bridge setup
		if err := o.SetupAddresses(); err != nil {
			return errors.Wrap(err, "SetupAddresses")
		}
		// sleep 1 second to wait for bridge setup
		time.Sleep(1 * time.Second)
		// to setup default routes
		if len(d.ip) > 0 || len(d.ip6) > 0 {
			setupRoutes := make([]iproute2.RouteSpec, 0)
			{
				for i := range routes {
					find := false
					if !find {
						if (len(d.ip) > 0 && routes[i].Dst.String() == addr2Prefix(d.ip, d.maskLen)) || (len(d.ip6) > 0 && routes[i].Dst.String() == addr2Prefix(d.ip6, d.mask6Len)) {
							find = true
						}
					}
					if !find {
						for j := range slaveAddrs {
							if routes[i].Dst.String() == addr2Prefix(slaveAddrs[j].Addr, slaveAddrs[j].MaskLen) {
								find = true
								break
							}
						}
					}
					if !find {
						// need to migrate route
						log.Infof("need to migrate route: %s", routes[i].String())
						setupRoutes = append(setupRoutes, routes[i])
					}
				}
			}
			if len(setupRoutes) > 0 {
				if err := o.SetupRoutes(setupRoutes, true, d.bridge.String()); err != nil {
					return errors.Wrap(err, "SetupRoutes")
				}
			}
			/*if len(setupRoutes) > 0 {
				if err := trySetupSlaveAddressesRoutes(o, slaveAddrs, nil, setupRoutes); err != nil {
					return errors.Wrap(err, "trySetupSlaveAddressesRoutes")
				}
			}*/
		}

	}

	return o.BringupInterface()
}

func (d *SBaseBridgeDriver) CleanupConfig() {
	// pass
}

func (d *SBaseBridgeDriver) saveFileExecutable(scriptPath, script string) error {
	if err := fileutils2.FilePutContents(scriptPath, script, false); err != nil {
		return err
	}
	return os.Chmod(scriptPath, syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IXUSR)
}

func (d *SBaseBridgeDriver) generateIfdownScripts(driver IBridgeDriver, scriptPath string, nic *desc.SGuestNetwork, isVolatileHost bool) error {
	script, err := driver.getDownScripts(nic, isVolatileHost)
	if err != nil {
		return errors.Wrap(err, "getDownScripts")
	}
	return d.saveFileExecutable(scriptPath, script)
}

func (d *SBaseBridgeDriver) generateIfupScripts(driver IBridgeDriver, scriptPath string, nic *desc.SGuestNetwork, isVolatileHost bool) error {
	script, err := driver.getUpScripts(nic, isVolatileHost)
	if err != nil {
		log.Errorln(err)
		return err
	}
	return d.saveFileExecutable(scriptPath, script)
}

func (d *SBaseBridgeDriver) GetMetadataServerPort() int {
	return options.HostOptions.Port + 1000
}

func (d *SBaseBridgeDriver) WarmupConfig() error {
	return nil
}

func (d *SBaseBridgeDriver) DisableDHCPClient() (bool, error) {
	if d.inter != nil {
		filename := fmt.Sprintf("/var/run/dhclient-%s.pid", d.inter.String())
		if !fileutils2.Exists(filename) {
			return false, nil
		}
		s, err := fileutils2.FileGetContents(filename)
		if err != nil {
			return false, errors.Wrap(err, "get dhclient pid")
		}
		pid, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return false, errors.Wrap(err, "convert pid str to int")
		}
		if fileutils2.Exists(fmt.Sprintf("/proc/%d/cmdline", pid)) {
			cmdline, err := fileutils2.FileGetContents(fmt.Sprintf("/proc/%d/cmdline", pid))
			if err != nil {
				return false, errors.Wrap(err, "get proc cmdline")
			}
			if strings.Contains(cmdline, "dhclient") {
				// kill process
				p, _ := os.FindProcess(pid)
				return true, p.Kill()
			}
		}
	}
	return false, nil
}

func NewDriver(bridgeDriver, bridge, inter, ip string, maskLen int, ip6 string, mask6Len int) (IBridgeDriver, error) {
	if bridgeDriver == DRV_OPEN_VSWITCH {
		return NewOVSBridgeDriver(bridge, inter, ip, maskLen, ip6, mask6Len)
	} else if bridgeDriver == DRV_LINUX_BRIDGE {
		return NewLinuxBridgeDeriver(bridge, inter, ip, maskLen, ip6, mask6Len)
	}
	return nil, fmt.Errorf("Dirver %s not found", bridgeDriver)
}

func Prepare(bridgeDriver string) error {
	if bridgeDriver == DRV_OPEN_VSWITCH {
		return OVSPrepare()
	} else if bridgeDriver == DRV_LINUX_BRIDGE {
		return LinuxBridgePrepare()
	}
	return fmt.Errorf("Dirver %s not found", bridgeDriver)
}

func CleanDeletedPorts(bridgeDriver string) {
	if bridgeDriver == DRV_OPEN_VSWITCH {
		cleanOvsBridge()
	} else if bridgeDriver == DRV_LINUX_BRIDGE {
		cleanLinuxBridge()
	}
}

func (d *SBaseBridgeDriver) IsV4Only() bool {
	return d.ip6 == "" && !d.bridge.IsSecretInterface6()
}
