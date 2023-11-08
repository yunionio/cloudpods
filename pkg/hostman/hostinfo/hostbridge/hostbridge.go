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
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/iproute2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type IBridgeDriver interface {
	ConfirmToConfig() (bool, error)
	GetMac() string
	GetVlanId() int
	FetchConfig()
	Setup(IBridgeDriver) error
	SetupAddresses(net.IPMask) error
	SetupSlaveAddresses([][]string) error
	SetupRoutes(routes []iproute2.RouteSpec) error
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
}

type SBaseBridgeDriver struct {
	bridge *netutils2.SNetInterface
	ip     string
	inter  *netutils2.SNetInterface

	drv IBridgeDriver
}

func NewBaseBridgeDriver(bridge, inter, ip string) (*SBaseBridgeDriver, error) {
	var bd = new(SBaseBridgeDriver)
	bd.bridge = netutils2.NewNetInterface(bridge)
	if len(inter) > 0 {
		bd.inter = netutils2.NewNetInterface(inter)
		if !bd.inter.Exist() {
			return nil, fmt.Errorf("%s not exists", inter)
		}
		bd.ip = ip
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
	} else if len(ip) > 0 {
		return nil, fmt.Errorf("A bridge without interface must have no IP")
	}
	return bd, nil
}

func (d *SBaseBridgeDriver) FetchConfig() {
	d.bridge.FetchConfig()
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

func (d *SBaseBridgeDriver) ConfirmToConfig() (bool, error) {
	exist, err := d.drv.Exists()
	if err != nil {
		return false, err
	}
	if exist {
		d.bridge.FetchConfig()
		if len(d.ip) > 0 {
			if len(d.bridge.Addr) == 0 {
				log.Infof("bridge %s has no ip assignment initially", d.bridge)
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
			} else {
				log.Infof("bridge %s already has ip %s", d.bridge, d.bridge.Addr)
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
		infs, err := d.drv.Interfaces()
		if err != nil {
			return false, err
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
	br := d.bridge.String()
	{
		var (
			addr    string
			masklen int
		)
		if len(d.ip) == 0 {
			addr, masklen = netutils2.GetSecretInterfaceAddress()
		} else {
			addr = d.ip
			masklen, _ = mask.Size()
		}
		addrStr := fmt.Sprintf("%s/%d", addr, masklen)
		if err := iproute2.NewAddress(br, addrStr).Exact().Err(); err != nil {
			return errors.Wrapf(err, "set bridge %s address", br)
		}
	}
	{
		brLink := iproute2.NewLink(br).Up()
		if options.HostOptions.TunnelPaddingBytes > 0 {
			mtu := 1500 + int(options.HostOptions.TunnelPaddingBytes)
			brLink.MTU(mtu)
		}
		if err := brLink.Err(); err != nil {
			return errors.Wrapf(err, "setting bridge %s up", br)
		}
	}
	if d.inter != nil {
		ifname := d.inter.String()
		if err := iproute2.NewAddress(ifname).Exact().Err(); err != nil {
			return errors.Wrapf(err, "remove addresses on slave ifname: %s", ifname)
		}
		if err := iproute2.NewLink(ifname).Up().Err(); err != nil {
			return errors.Wrapf(err, "setting bridge %s ifname %s up", br, ifname)
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) SetupSlaveAddresses(slaveAddrs [][]string) error {
	br := d.bridge.String()
	addrs := make([]string, len(slaveAddrs))
	for i, slaveAddr := range slaveAddrs {
		addrs[i] = fmt.Sprintf("%s/%s", slaveAddr[0], slaveAddr[1])
	}
	if err := iproute2.NewAddress(br, addrs...).Add().Err(); err != nil {
		return errors.Wrap(err, "move secondary addresses to bridge interface")
	}
	return nil
}

func (d *SBaseBridgeDriver) SetupRoutes(routespecs []iproute2.RouteSpec) error {
	bridgeIP := d.inter.Addr
	bridgeMask := d.inter.Mask
	br := d.bridge.String()
	for i := len(routespecs) - 1; i >= 0; i-- {
		errs := []error{}
		routespec := routespecs[i]
		if routespec.Dst.Contains(net.ParseIP(bridgeIP)) && bridgeMask.String() == routespec.Dst.Mask.String() {
			log.Infof("skip setup route: %s", routespec.String())
			continue
		}
		cmd := []string{
			"route", "add", routespec.Dst.String(),
		}
		if routespec.Gw != nil {
			cmd = append(cmd, "via", routespec.Gw.String())
		}
		cmd = append(cmd, "dev", br)

		output, err := procutils.NewRemoteCommandAsFarAsPossible("ip", cmd...).Output()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "run cmd: ip %s, output: %s", strings.Join(cmd, " "), output))
			cmd = append(cmd, "onlink")
			if output, err := procutils.NewRemoteCommandAsFarAsPossible("ip", cmd...).Output(); err != nil {
				errs = append(errs, errors.Wrapf(err, "run cmd: ip %s, output: %s", strings.Join(cmd, " "), output))
				return errors.Wrapf(errors.NewAggregate(errs), "setup route %s", routespec.String())
			}
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) Setup(o IBridgeDriver) error {
	var routes []iproute2.RouteSpec
	var slaveAddrs [][]string
	if d.inter != nil && len(d.inter.Addr) > 0 {
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
	if len(d.bridge.Addr) == 0 {
		if len(d.ip) > 0 {
			if err := o.SetupAddresses(d.inter.Mask); err != nil {
				return errors.Wrap(err, "SetupAddresses")
			}
			time.Sleep(1 * time.Second)
			if len(slaveAddrs) > 0 {
				tried := 0
				const MAX_TRIES = 4
				errs := make([]error, 0)
				for tried < MAX_TRIES {
					if err := o.SetupSlaveAddresses(slaveAddrs); err != nil {
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
			if len(routes) > 0 {
				tried := 0
				const MAX_TRIES = 4
				errs := make([]error, 0)
				for {
					if err := o.SetupRoutes(routes); err != nil {
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
		} else {
			if err := o.SetupAddresses(nil); err != nil {
				return errors.Wrap(err, "SetupAddresses nil")
			}
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

func NewDriver(bridgeDriver, bridge, inter, ip string) (IBridgeDriver, error) {
	if bridgeDriver == DRV_OPEN_VSWITCH {
		return NewOVSBridgeDriver(bridge, inter, ip)
	} else if bridgeDriver == DRV_LINUX_BRIDGE {
		return NewLinuxBridgeDeriver(bridge, inter, ip)
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
