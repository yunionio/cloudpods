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
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/netutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type IBridgeDriver interface {
	ConfirmToConfig() (bool, error)
	GetMac() string
	FetchConfig()
	Setup(IBridgeDriver) error
	SetupAddresses(net.IPMask) error
	SetupSlaveAddresses([][]string) error
	SetupRoutes(routes [][]string) error
	BringupInterface() error

	Exists() (bool, error)
	Interfaces() ([]string, error)
	WarmupConfig() error
	CleanupConfig()
	SetupBridgeDev() error
	SetupInterface() error
	PersistentMac() error

	GenerateIfupScripts(scriptPath string, nic jsonutils.JSONObject) error
	GenerateIfdownScripts(scriptPath string, nic jsonutils.JSONObject) error
	RegisterHostlocalServer(mac, ip string) error

	getUpScripts(nic jsonutils.JSONObject) (string, error)
	getDownScripts(nic jsonutils.JSONObject) (string, error)
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
		bd.inter.DisableGso()
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
	if len(d.bridge.Mac) == 0 {
		d.bridge.FetchConfig()
	}
	return d.bridge.Mac
}

func (d *SBaseBridgeDriver) PersistentMac() error {
	return nil
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
		if _, err := procutils.NewCommand(cmd[0], cmd[1:]...).Output(); err != nil {
			return err
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) ConfirmToConfig() (bool, error) {
	output, err := procutils.NewCommand("ifconfig").Output()
	if err != nil {
		return false, errors.Wrapf(err, "exec ifconfig %s", output)
	}

	exist, err := d.drv.Exists()
	if err != nil {
		return false, err
	}
	if exist {
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
	var addr string
	if len(d.ip) == 0 {
		addr, mask = netutils2.GetSecretInterfaceAddress()
	} else {
		addr = d.ip
	}
	cmd := []string{"ifconfig", d.bridge.String(), addr, "netmask", netutils2.NetBytes2Mask(mask)}
	if options.HostOptions.TunnelPaddingBytes > 0 {
		cmd = append(cmd, "mtu", fmt.Sprintf("%d", options.HostOptions.TunnelPaddingBytes+1500))
	}
	if _, err := procutils.NewCommand(cmd[0], cmd[1:]...).Output(); err != nil {
		log.Errorln(err)
		return fmt.Errorf("Failed to bring up bridge %s", d.bridge)
	}
	if d.inter != nil {
		if _, err := procutils.NewCommand("ifconfig", d.inter.String(), "0", "up").Output(); err != nil {
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
		if _, err := procutils.NewCommand(cmd[0], cmd[1:]...).Output(); err != nil {
			log.Errorf("Failed to remove slave address from interface %s: %s", d.inter, err)
		}

		cmd = []string{"ip", "address", "add",
			fmt.Sprintf("%s/%s", slaveAddr[0], slaveAddr[1]), "dev", d.bridge.String()}
		if _, err := procutils.NewCommand(cmd[0], cmd[1:]...).Output(); err != nil {
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
		if _, err := procutils.NewCommand(cmd[0], cmd[1:]...).Output(); err != nil {
			log.Errorln(err)
			return fmt.Errorf("Failed to add slave address to bridge %s", d.bridge)
		}
	}
	return nil
}

func (d *SBaseBridgeDriver) Setup(o IBridgeDriver) error {
	var routes [][]string
	var slaveAddrs [][]string
	if d.inter != nil && len(d.inter.Addr) > 0 {
		routes = d.inter.GetRoutes(true)
		slaveAddrs = d.inter.GetSlaveAddresses()
	}
	exist, err := o.Exists()
	if err != nil {
		return err
	}
	if !exist {
		if err := o.SetupBridgeDev(); err != nil {
			return err
		}
	}

	infs, err := o.Interfaces()
	if err != nil {
		return err
	}
	if d.inter != nil && !utils.IsInStringArray(d.inter.String(), infs) {
		if err := o.SetupInterface(); err != nil {
			return err
		}
	}
	if len(d.bridge.Addr) == 0 {
		if len(d.ip) > 0 {
			if err := o.SetupAddresses(d.inter.Mask); err != nil {
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

func (d *SBaseBridgeDriver) CleanupConfig() {
	// pass
}

func (d *SBaseBridgeDriver) saveFileExecutable(scriptPath, script string) error {
	if err := fileutils2.FilePutContents(scriptPath, script, false); err != nil {
		return err
	}
	return os.Chmod(scriptPath, syscall.S_IRUSR|syscall.S_IWUSR|syscall.S_IXUSR)
}

func (d *SBaseBridgeDriver) generateIfdownScripts(driver IBridgeDriver, scriptPath string, nic jsonutils.JSONObject) error {
	script, err := driver.getDownScripts(nic)
	if err != nil {
		log.Errorln(err)
		return err
	}
	return d.saveFileExecutable(scriptPath, script)
}

func (d *SBaseBridgeDriver) generateIfupScripts(driver IBridgeDriver, scriptPath string, nic jsonutils.JSONObject) error {
	script, err := driver.getUpScripts(nic)
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

func NewDriver(bridgeDriver, bridge, inter, ip string) (IBridgeDriver, error) {
	if bridgeDriver == "openvswitch" {
		return NewOVSBridgeDriver(bridge, inter, ip)
	} else if bridgeDriver == "linux_bridge" {
		return NewLinuxBridgeDeriver(bridge, inter, ip)
	}
	return nil, fmt.Errorf("Dirver %s not found", bridgeDriver)
}

func Prepare(bridgeDriver string) error {
	if bridgeDriver == "openvswitch" {
		return OVSPrepare()
	} else if bridgeDriver == "linux_bridge" {
		return LinuxBridgePrepare()
	}
	return fmt.Errorf("Dirver %s not found", bridgeDriver)
}

func CleanDeletedPorts(bridgeDriver string) {
	if bridgeDriver == "openvswitch" {
		cleanOvsBridge()
	} else if bridgeDriver == "linux_bridge" {
		cleanLinuxBridge()
	}
}
