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

package tasks

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	baremetalapi "yunion.io/x/onecloud/pkg/apis/compute/baremetal"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/profiles"
	"yunion.io/x/onecloud/pkg/baremetal/utils/detect_storages"
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type sBaremetalPrepareTask struct {
	baremetal IBaremetal
	startTime time.Time
	userCred  mcclient.TokenCredential
}

func newBaremetalPrepareTask(baremetal IBaremetal, userCred mcclient.TokenCredential) *sBaremetalPrepareTask {
	return &sBaremetalPrepareTask{
		baremetal: baremetal,
		userCred:  userCred,
		startTime: time.Now().UTC(),
	}
}

type baremetalPrepareInfo struct {
	architecture        string
	sysInfo             *types.SSystemInfo
	cpuInfo             *types.SCPUInfo
	dmiCpuInfo          *types.SDMICPUInfo
	memInfo             *types.SDMIMemInfo
	nicsInfo            []*types.SNicDevInfo
	diskInfo            []*baremetal.BaremetalStorage
	storageDriver       string
	ipmiInfo            *types.SIPMIInfo
	isolatedDevicesInfo []*isolated_device.PCIDevice
}

func (task *sBaremetalPrepareTask) GetStartTime() time.Time {
	return task.startTime
}

func (task *sBaremetalPrepareTask) prepareBaremetalInfo(cli *ssh.Client) (*baremetalPrepareInfo, error) {
	arch, err := getArchitecture(cli)
	if err != nil {
		return nil, errors.Wrap(err, "getArchitecture")
	}

	if arch == apis.OS_ARCH_X86_64 {
		if _, err := cli.Run("/lib/mos/sysinit.sh"); err != nil {
			return nil, errors.Wrap(err, "run /lib/mos/sysinit.sh")
		}
	} else {
		log.Infof("Skip running /lib/mos/sysinit.sh when arch is %q", arch)
	}

	sysInfo, err := getDMISysinfo(cli)
	if err != nil {
		return nil, err
	}
	cpuInfo, err := getCPUInfo(cli)
	if err != nil {
		return nil, err
	}
	dmiCPUInfo, err := getDMICPUInfo(cli)
	if err != nil {
		return nil, err
	}
	memInfo, err := getDMIMemInfo(cli)
	if err != nil {
		return nil, err
	}
	nicsInfo, err := getNicsInfo(cli)
	if err != nil {
		return nil, err
	}
	dhcpServerIp, err := task.baremetal.GetDHCPServerIP()
	if err != nil {
		log.Errorf("failed get dhcp server %s", err)
	}
	isolatedDevicesInfo, err := getIsolatedDevicesInfo(cli, dhcpServerIp)
	if err != nil {
		return nil, err
	}

	raidDiskInfo, nonRaidDiskInfo, pcieDiskInfo, err := detect_storages.DetectStorageInfo(cli, true)
	if err != nil {
		return nil, err
	}
	diskInfo := make([]*baremetal.BaremetalStorage, 0)
	diskInfo = append(diskInfo, raidDiskInfo...)
	diskInfo = append(diskInfo, nonRaidDiskInfo...)
	diskInfo = append(diskInfo, pcieDiskInfo...)
	var storageDriver string
	if len(raidDiskInfo) > 0 {
		raidDrivers := []string{}
		for _, drv := range raidDiskInfo {
			if !utils.IsInStringArray(drv.Driver, raidDrivers) {
				raidDrivers = append(raidDrivers, drv.Driver)
			}
		}
		storageDriver = strings.Join(raidDrivers, ",")
	} else {
		storageDriver = baremetal.DISK_DRIVER_LINUX
	}

	ipmiEnable, err := isIPMIEnable(cli)
	if err != nil {
		return nil, err
	}

	ipmiInfo := task.baremetal.GetRawIPMIConfig()
	if ipmiInfo == nil {
		ipmiInfo = &types.SIPMIInfo{}
	}
	if !ipmiInfo.Verified && !ipmiInfo.Present && ipmiEnable {
		ipmiInfo.Present = true
		ipmiInfo.Verified = false
	}

	prepareInfo := &baremetalPrepareInfo{
		architecture:        arch,
		sysInfo:             sysInfo,
		cpuInfo:             cpuInfo,
		dmiCpuInfo:          dmiCPUInfo,
		memInfo:             memInfo,
		nicsInfo:            nicsInfo,
		diskInfo:            diskInfo,
		storageDriver:       storageDriver,
		ipmiInfo:            ipmiInfo,
		isolatedDevicesInfo: isolatedDevicesInfo,
	}

	return prepareInfo, nil
}

func (task *sBaremetalPrepareTask) configIPMISetting(ctx context.Context, cli *ssh.Client, i *baremetalPrepareInfo) error {
	if !i.ipmiInfo.Present {
		return nil
	}

	// if verified, skip ipmi config
	if i.ipmiInfo.Verified {
		return nil
	}

	var (
		sysInfo  = i.sysInfo
		ipmiInfo = i.ipmiInfo
	)
	sshIPMI := ipmitool.NewSSHIPMI(cli)
	setIPMILanPortShared(sshIPMI, sysInfo)
	profile, err := profiles.GetProfile(ctx, sysInfo)
	if err != nil {
		return errors.Wrap(err, "GetProfile")
	}
	ipmiUser, ipmiPasswd, ipmiIpAddr := task.getIPMIUserPasswd(i.ipmiInfo, profile)
	ipmiInfo.Username = ipmiUser
	ipmiInfo.Password = ipmiPasswd

	var ipmiLanChannel uint8 = 0
	for _, lanChannel := range profile.LanChannels {
		log.Infof("Try lan channel %d ...", lanChannel)
		conf, err := ipmitool.GetLanConfig(sshIPMI, lanChannel)
		if err != nil {
			log.Errorf("Get lan channel %d config error: %v", lanChannel, err)
			continue
		}
		if conf.Mac == nil {
			log.Errorf("Lan channel %d MAC address is empty", lanChannel)
			continue
		}

		ipmiNic := &types.SNicDevInfo{
			Mac:   conf.Mac,
			Speed: 100,
			Mtu:   1500,
		}
		if err := task.sendNicInfo(ctx, ipmiNic, -1, api.NIC_TYPE_IPMI, true, "", false); err != nil {
			// ignore the error
			log.Errorf("Send IPMI nic %#v info: %v", ipmiNic, err)
		}
		rootId := profile.RootId
		err = ipmitool.CreateOrSetAdminUser(sshIPMI, lanChannel, rootId, ipmiUser, ipmiPasswd)
		if err != nil {
			// ignore the error
			log.Errorf("Lan channel %d set user password error: %v", lanChannel, err)
		}
		err = ipmitool.EnableLanAccess(sshIPMI, lanChannel)
		if err != nil {
			// ignore the error
			log.Errorf("Lan channel %d enable lan access error: %v", lanChannel, err)
		}

		tryAddrs := make([]string, 0)
		if ipmiIpAddr != "" {
			tryAddrs = append(tryAddrs, ipmiIpAddr)
		}
		if conf.IPAddr != "" && conf.IPAddr != ipmiIpAddr {
			tryAddrs = append(tryAddrs, conf.IPAddr)
		}
		if len(tryAddrs) > 0 && !o.Options.ForceDhcpProbeIpmi {
			for _, tryAddr := range tryAddrs {
				tryResult := task.tryLocalIpmiAddr(ctx, sshIPMI, ipmiNic, lanChannel,
					ipmiUser, ipmiPasswd, tryAddr)
				if tryResult {
					ipmiInfo.IpAddr = tryAddr
					ipmiLanChannel = lanChannel
					break
				}
			}
			if ipmiLanChannel > 0 {
				// found and set config on lanChannel
				break
			}
		}

		if len(tryAddrs) > 0 {
			task.baremetal.SetExistingIPMIIPAddr(tryAddrs[0])
		}

		err = ipmitool.SetLanDHCP(sshIPMI, lanChannel)
		if err != nil {
			// ignore error
			log.Errorf("Set lan channel %d dhcp error: %v", lanChannel, err)
		}
		time.Sleep(2 * time.Second)
		nic := task.baremetal.GetIPMINic(conf.Mac)
		maxTries := 180 // wait 3 minutes
		for tried := 0; nic != nil && nic.IpAddr == "" && tried < maxTries; tried++ {
			nic = task.baremetal.GetIPMINic(conf.Mac)
		}
		if len(nic.IpAddr) == 0 {
			err = ipmitool.DoBMCReset(sshIPMI) // do BMC reset to force DHCP request
			if err != nil {
				log.Errorf("Do BMC reset error: %v", err)
			}
			time.Sleep(1 * time.Second)
		}
		for tried := 0; nic != nil && nic.IpAddr == "" && tried < maxTries; tried++ {
			nic = task.baremetal.GetIPMINic(conf.Mac)
			time.Sleep(1 * time.Second)
		}
		if nic != nil && len(nic.IpAddr) == 0 {
			log.Errorf("DHCP wait IPMI address fail, retry ...")
			continue
		}
		log.Infof("DHCP get IPMI address succ, wait 2 seconds ...")
		var tried int = 0
		for tried < maxTries {
			time.Sleep(2 * time.Second)
			lanConf, err := ipmitool.GetLanConfig(sshIPMI, lanChannel)
			if err != nil {
				log.Errorf("Get lan config at channel %d error: %v", lanChannel, err)
				tried += 2
				continue
			}
			if lanConf.IPAddr == nic.IpAddr {
				break
			}
			log.Infof("waiting IPMI DHCP address old:%s expect:%s", lanConf.IPAddr, nic.IpAddr)
			tried += 2
		}
		if tried >= maxTries {
			continue
		}
		err = ipmitool.SetLanStatic(
			sshIPMI,
			lanChannel,
			nic.IpAddr,
			nic.GetNetMask(),
			nic.Gateway,
		)
		if err != nil {
			log.Errorf("Set lanChannel %d static net %#v error: %v", lanChannel, nic, err)
			continue
		}
		ipmiInfo.IpAddr = nic.IpAddr
		ipmiLanChannel = lanChannel
	}
	if ipmiLanChannel == 0 {
		return fmt.Errorf("Fail to get IPMI address from DHCP")
	}
	ipmiInfo.LanChannel = ipmiLanChannel
	ipmiInfo.Verified = true
	return nil
}

func (task *sBaremetalPrepareTask) DoPrepare(ctx context.Context, cli *ssh.Client) error {
	infos, err := task.prepareBaremetalInfo(cli)
	if err != nil {
		logclient.AddActionLogWithStartable(task, task.baremetal, logclient.ACT_PREPARE, err, task.userCred, false)
		return err
	}

	// set ipmi nic address and user password
	if err = task.configIPMISetting(ctx, cli, infos); err != nil {
		logclient.AddActionLogWithStartable(task, task.baremetal, logclient.ACT_PREPARE, err, task.userCred, false)
		return errors.Wrap(err, "Config IPMI setting")
	}

	if err = task.updateBmInfo(ctx, cli, infos); err != nil {
		logclient.AddActionLogWithStartable(task, task.baremetal, logclient.ACT_PREPARE, err, task.userCred, false)
		return err
	}

	// set NTP
	if err = task.baremetal.DoNTPConfig(); err != nil {
		// ignore error
		log.Errorf("SetNTP fail: %s", err)
	}

	if err = AdjustUEFIBootOrder(ctx, cli, task.baremetal); err != nil {
		logclient.AddActionLogWithStartable(task, task.baremetal, logclient.ACT_PREPARE, err, task.userCred, false)
		return errors.Wrap(err, "Adjust UEFI boot order")
	}

	logclient.AddActionLogWithStartable(task, task.baremetal, logclient.ACT_PREPARE, infos.sysInfo, task.userCred, true)

	log.Infof("Prepare complete")
	return nil
}

func (task *sBaremetalPrepareTask) findAdminNic(cli *ssh.Client, nicsInfo []*types.SNicDevInfo) (int, *types.SNicDevInfo, error) {
	accessIp := cli.GetConfig().Host
	for idx := range nicsInfo {
		nic := nicsInfo[idx]
		output, err := cli.Run("/sbin/ifconfig " + nic.Dev)
		if err != nil {
			log.Errorf("ifconfig %s fail: %s", nic.Dev, err)
			continue
		}
		isAdmin := false
		for _, l := range output {
			if strings.Contains(l, accessIp) {
				isAdmin = true
				break
			}
		}
		if isAdmin {
			return idx, nic, nil
		}
	}
	return -1, nil, errors.Error("admin nic not found???")
}

func (task *sBaremetalPrepareTask) updateBmInfo(ctx context.Context, cli *ssh.Client, i *baremetalPrepareInfo) error {
	adminNic := task.baremetal.GetAdminNic()
	if adminNic == nil || (adminNic != nil && !adminNic.LinkUp) {
		adminIdx, adminNicDev, err := task.findAdminNic(cli, i.nicsInfo)
		if err != nil {
			return errors.Wrap(err, "task.findAdminNic")
		}
		accessIp := cli.GetConfig().Host
		err = task.sendNicInfo(ctx, adminNicDev, adminIdx, api.NIC_TYPE_ADMIN, false, accessIp, true)
		if err != nil {
			return errors.Wrap(err, "send Admin Nic Info")
		}
		adminNic = task.baremetal.GetNicByMac(adminNicDev.Mac)
	}
	// collect params
	updateInfo := make(map[string]interface{})
	oname := fmt.Sprintf("BM%s", strings.Replace(adminNic.Mac, ":", "", -1))
	if task.baremetal.GetName() == oname {
		updateInfo["name"] = fmt.Sprintf("BM-%s", strings.Replace(i.ipmiInfo.IpAddr, ".", "-", -1))
	}
	updateInfo["access_ip"] = adminNic.IpAddr
	updateInfo["access_mac"] = adminNic.Mac
	updateInfo["cpu_architecture"] = i.architecture
	updateInfo["cpu_count"] = i.cpuInfo.Count
	updateInfo["node_count"] = i.dmiCpuInfo.Nodes
	updateInfo["cpu_desc"] = i.cpuInfo.Model
	updateInfo["cpu_mhz"] = i.cpuInfo.Freq
	updateInfo["cpu_cache"] = i.cpuInfo.Cache
	updateInfo["mem_size"] = i.memInfo.Total
	updateInfo["storage_driver"] = i.storageDriver
	updateInfo["storage_info"] = i.diskInfo
	updateInfo["sys_info"] = i.sysInfo
	updateInfo["sn"] = i.sysInfo.SN
	size, diskType := task.collectDiskInfo(i.diskInfo)
	updateInfo["storage_size"] = size
	updateInfo["storage_type"] = diskType
	updateInfo["boot_mode"] = getBootMode(cli)
	updateInfo["uuid"] = getSystemGuid(cli)
	updateData := jsonutils.Marshal(updateInfo)
	updateData.(*jsonutils.JSONDict).Update(i.ipmiInfo.ToPrepareParams())
	_, err := modules.Hosts.Update(task.getClientSession(), task.baremetal.GetId(), updateData)
	if err != nil {
		log.Errorf("Update baremetal info error: %v", err)
		return errors.Wrap(err, "Hosts.Update")
	}
	if err := task.sendStorageInfo(size); err != nil {
		log.Errorf("sendStorageInfo error: %v", err)
		return errors.Wrap(err, "task.sendStorageInfo")
	}
	if len(i.isolatedDevicesInfo) > 0 {
		err = task.sendIsolatedDevicesInfo(task.getClientSession(), i.isolatedDevicesInfo)
		if err != nil {
			return errors.Wrap(err, "send isolated devices info")
		}
	}
	// XXX do not change nic order anymore
	// for i := range nicsInfo {
	// 	if nicsInfo[i].Mac.String() == adminNic.GetMac().String() {
	// 		if i != 0 {
	// 			nicsInfo = append(nicsInfo[i:], nicsInfo[0:i]...)
	// 		}
	// 		break
	// 	}
	// }
	// err = task.removeAllNics()
	// if err != nil {
	// 	return err
	// }
	removedMacs := task.removeObsoleteNics(i)
	for idx := range removedMacs {
		err = task.removeNicInfo(ctx, removedMacs[idx])
		if err != nil {
			log.Errorf("Fail to remove Netif %s: %s", removedMacs[idx], err)
			return errors.Wrap(err, "task.removeNicInfo")
		}
	}
	for idx := range i.nicsInfo {
		err = task.sendNicInfo(ctx, i.nicsInfo[idx], idx, "", false, "", false)
		if err != nil {
			log.Errorf("Send nicinfo idx: %d, %#v error: %v", idx, i.nicsInfo[idx], err)
			return errors.Wrap(err, "task.sendNicInfo")
		}
	}
	if o.Options.EnablePxeBoot && task.baremetal.EnablePxeBoot() {
		for _, nicInfo := range i.nicsInfo {
			if nicInfo.Mac.String() != adminNic.GetMac().String() && nicInfo.Up != nil && *nicInfo.Up {
				err = task.doNicWireProbe(cli, nicInfo)
				if err != nil {
					// ignore the error
					log.Errorf("doNicWireProbe nic %#v error: %v", nicInfo, err)
				}
			}
		}
	}
	return nil
}

func (task *sBaremetalPrepareTask) removeObsoleteNics(i *baremetalPrepareInfo) []string {
	removes := make([]string, 0)
	existNics := task.baremetal.GetNics()
	log.Debugf("Existing nics: %s", jsonutils.Marshal(existNics))
	for idx := range existNics {
		if existNics[idx].Type != api.NIC_TYPE_NORMAL {
			continue
		}
		find := false
		for j := range i.nicsInfo {
			if existNics[idx].Mac == i.nicsInfo[j].Mac.String() {
				find = true
				break
			}
		}
		if !find {
			removes = append(removes, existNics[idx].Mac)
		}
	}
	return removes
}

func (task *sBaremetalPrepareTask) tryLocalIpmiAddr(ctx context.Context, sshIPMI *ipmitool.SSHIPMI, ipmiNic *types.SNicDevInfo, lanChannel uint8, ipmiUser, ipmiPasswd, tryAddr string) bool {
	log.Infof("IP addr found in IPMI config, try use %s as IPMI address", tryAddr)
	ipConf, err := task.getIPMIIPConfig(tryAddr)
	if err != nil {
		log.Errorf("Failed to get IPMI ipconfig for %s", tryAddr)
		return false
	}
	err = ipmitool.SetLanStatic(sshIPMI, lanChannel, ipConf.IPAddr, ipConf.Netmask, ipConf.Gateway)
	if err != nil {
		log.Errorf("Failed to set IPMI static net config %#v for %s", *ipConf, tryAddr)
		return false
	}

	var conf *types.SIPMILanConfig

	tried := 0
	maxTries := 5
	time.Sleep(2 * time.Second)
	for tried = 0; tried < maxTries; tried += 1 {
		conf, err = ipmitool.GetLanConfig(sshIPMI, lanChannel)
		if err != nil {
			log.Errorf("Failed to get lan config after set static network: %v", err)
			continue
		}
		log.Infof("Get lan config %#v", *conf)
		if conf.IPAddr == "" || conf.IPAddr != tryAddr {
			log.Errorf("Failed to set ipmi lan channel %d static ipaddr", lanChannel)
			continue
		}
		break
	}
	if tried >= maxTries {
		log.Errorf("Failed to get lan config after %d tries", tried)
		return false
	}
	rmcpIPMI := ipmitool.NewLanPlusIPMI(tryAddr, ipmiUser, ipmiPasswd)
	for tried = 0; tried < maxTries; tried += 1 {
		conf2, err := ipmitool.GetLanConfig(rmcpIPMI, lanChannel)
		if err != nil {
			log.Errorf("Failed to get lan channel %d config use RMCP mode: %v", lanChannel, err)
			continue
		}
		if len(conf2.Mac) != 0 &&
			conf2.Mac.String() == conf.Mac.String() &&
			conf2.IPAddr != "" && conf2.IPAddr == tryAddr {
			break
		} else {
			log.Errorf("fail to rmcp get IPMI ip config %v", conf2)
			time.Sleep(5 * time.Second)
		}
	}
	if tried < maxTries {
		// make sure the ipaddr is a IPMI address
		// enable the netif
		err := task.sendNicInfo(ctx, ipmiNic, -1, api.NIC_TYPE_IPMI, false, tryAddr, true)
		if err != nil {
			log.Errorf("Fail to set existing BMC IP address to %s", tryAddr)
		} else {
			return true
		}
	}
	return false
}

func (task *sBaremetalPrepareTask) getIPMIUserPasswd(oldIPMIConf *types.SIPMIInfo, profile *baremetalapi.BaremetalProfileSpec) (string, string, string) {
	var (
		ipmiUser   string
		ipmiPasswd string
		ipmiIpAddr string
	)
	ipmiUser = profile.RootName
	isStrongPass := profile.StrongPass
	if !isStrongPass && o.Options.DefaultIpmiPassword != "" {
		ipmiPasswd = o.Options.DefaultIpmiPassword
	} else if isStrongPass && o.Options.DefaultStrongIpmiPassword != "" {
		ipmiPasswd = o.Options.DefaultStrongIpmiPassword
	} else if isStrongPass && o.Options.DefaultIpmiPassword != "" {
		ipmiPasswd = o.Options.DefaultIpmiPassword
	} else {
		ipmiPasswd = seclib.RandomPassword(20)
	}
	if oldIPMIConf.Username != "" {
		ipmiUser = oldIPMIConf.Username
	}
	if oldIPMIConf.Password != "" {
		ipmiPasswd = oldIPMIConf.Password
	}
	if oldIPMIConf.IpAddr != "" {
		ipmiIpAddr = oldIPMIConf.IpAddr
	}
	return ipmiUser, ipmiPasswd, ipmiIpAddr
}

type ipmiIPConfig struct {
	IPAddr  string
	Netmask string
	Gateway string
}

func (task *sBaremetalPrepareTask) getIPMIIPConfig(ipAddr string) (*ipmiIPConfig, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(ipAddr), "ip")
	params.Add(jsonutils.NewString("system"), "scope")
	params.Add(jsonutils.JSONTrue, "is_classic")
	listRet, err := modules.Networks.List(task.getClientSession(), params)
	if err != nil {
		return nil, err
	}
	if len(listRet.Data) != 1 {
		return nil, fmt.Errorf("Invalid network list count: %d", len(listRet.Data))
	}
	netObj := listRet.Data[0]
	config := &ipmiIPConfig{}
	config.IPAddr = ipAddr
	maskLen, _ := netObj.Int("guest_ip_mask")
	config.Netmask = netutils.Masklen2Mask(int8(maskLen)).String()
	config.Gateway, _ = netObj.GetString("guest_gateway")
	return config, nil
}

func (task *sBaremetalPrepareTask) getClientSession() *mcclient.ClientSession {
	return task.baremetal.GetClientSession()
}

func getArchitecture(cli *ssh.Client) (string, error) {
	ret, err := cli.Run("/bin/uname -m")
	if err != nil {
		return "", err
	}
	if len(ret) == 0 {
		return "", errors.Errorf("/bin/uname not output")
	}
	arch := strings.ToLower(ret[0])
	switch arch {
	case apis.OS_ARCH_AARCH64:
		return apis.OS_ARCH_AARCH64, nil
	case apis.OS_ARCH_X86_64:
		return apis.OS_ARCH_X86_64, nil
	default:
		return arch, nil
	}
}

func getDMISysinfo(cli *ssh.Client) (*types.SSystemInfo, error) {
	ret, err := cli.Run("/usr/sbin/dmidecode -t 1")
	if err != nil {
		return nil, err
	}
	return sysutils.ParseDMISysinfo(ret)
}

func getCPUInfo(cli *ssh.Client) (*types.SCPUInfo, error) {
	ret, err := cli.Run("cat /proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	return sysutils.ParseCPUInfo(ret)
}

func getBootMode(cli *ssh.Client) string {
	lines, err := cli.Run("cat /proc/cmdline")
	if err != nil {
		return api.BOOT_MODE_PXE
	}
	const (
		key = "bootmode="
	)
	for _, line := range lines {
		pos := strings.Index(line, key)
		if pos >= 0 {
			if strings.HasPrefix(line[pos+len(key):], api.BOOT_MODE_ISO) {
				return api.BOOT_MODE_ISO
			}
		}
	}
	return api.BOOT_MODE_PXE
}

func getSystemGuid(cli *ssh.Client) string {
	lines, err := cli.Run("/usr/sbin/dmidecode -s system-uuid")
	if err != nil {
		return ""
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			return sysutils.NormalizeUuid(line)
		}
	}
	return ""
}

func getDMICPUInfo(cli *ssh.Client) (*types.SDMICPUInfo, error) {
	ret, err := cli.Run("/usr/sbin/dmidecode -t 4")
	if err != nil {
		return nil, err
	}
	return sysutils.ParseDMICPUInfo(ret), nil
}

func getDMIMemInfo(cli *ssh.Client) (*types.SDMIMemInfo, error) {
	ret, err := cli.Run("/usr/sbin/dmidecode -t 17")
	if err != nil {
		return nil, err
	}
	return sysutils.ParseDMIMemInfo(ret), nil
}

func GetNicsInfo(cli *ssh.Client) ([]*types.SNicDevInfo, error) {
	return getNicsInfo(cli)
}

func getNicsInfo(cli *ssh.Client) ([]*types.SNicDevInfo, error) {
	ret, err := cli.Run("/lib/mos/lsnic")
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve NIC info: %v", err)
	}
	return sysutils.ParseNicInfo(ret), nil
}

func getIsolatedDevicesInfo(cli *ssh.Client, ip net.IP) ([]*isolated_device.PCIDevice, error) {
	// fetch pci.ids from baremetal agent
	var updatedPciids bool
	if ip != nil {
		out, err := cli.Run(fmt.Sprintf("tftp -g -r pci.ids %s -l /pci.ids", ip))
		if err != nil {
			log.Errorf("tftp failed download pciids %s %s", err, out)
		} else {
			updatedPciids = true
		}
	}

	var bootVgaPath = []string{}
	lines, err := cli.Run("ls /sys/bus/pci/devices/*/boot_vga")
	if err != nil {
		log.Errorf("failed find boot vga %s", err)
	}
	for i := 0; i < len(lines); i++ {
		bootVgaPath = append(bootVgaPath, strings.TrimSpace(lines[i]))
	}

	cmd := "lspci -nnmm"
	if updatedPciids {
		cmd = "lspci -i /pci.ids -nnmm"
	}

	lines, err = cli.Run(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "run %s", cmd)
	}
	devs := []*isolated_device.PCIDevice{}
	for _, line := range lines {
		if len(line) > 0 {
			dev := isolated_device.NewPCIDevice2(line, cli)
			if len(dev.Addr) > 0 && utils.IsInArray(dev.ClassCode, isolated_device.GpuClassCodes) && !isBootVga(cli, dev, bootVgaPath) {
				devs = append(devs, dev)
			}
		}
	}
	return devs, nil
}

func isBootVga(cli *ssh.Client, dev *isolated_device.PCIDevice, bootVgaPath []string) bool {
	for i := 0; i < len(bootVgaPath); i++ {
		if strings.Contains(bootVgaPath[i], dev.Addr) {
			out, err := cli.RawRun(fmt.Sprintf("cat %s", bootVgaPath[i]))
			if err != nil {
				log.Errorf("cat boot_vga %s failed %s", bootVgaPath[i], err)
				return false
			} else if len(out) > 0 {
				if strings.HasPrefix(out[0], "1") {
					log.Infof("device %s is boot vga", bootVgaPath[i])
					return true
				} else {
					return false
				}
			}
			break
		}
	}
	return false
}

func isIPMIEnable(cli *ssh.Client) (bool, error) {
	ret, err := cli.Run("/usr/sbin/dmidecode -t 38")
	if err != nil {
		return false, fmt.Errorf("Failed to retrieve IPMI info: %v", err)
	}
	return sysutils.ParseDMIIPMIInfo(ret), nil
}

func (task *sBaremetalPrepareTask) removeNicInfo(ctx context.Context, mac string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(mac), "mac")
	resp, err := modules.Hosts.PerformAction(
		task.getClientSession(),
		task.baremetal.GetId(),
		"remove-netif",
		params,
	)
	if err != nil {
		return err
	}
	return task.baremetal.SaveDesc(ctx, resp)
}

func (task *sBaremetalPrepareTask) sendNicInfo(ctx context.Context,
	nic *types.SNicDevInfo, idx int, nicType compute.TNicType, reset bool, ipAddr string, reserve bool,
) error {
	return task.baremetal.SendNicInfo(ctx, nic, idx, nicType, reset, ipAddr, reserve)
}

func (task *sBaremetalPrepareTask) sendStorageInfo(size int64) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(size), "capacity")
	params.Add(jsonutils.NewString(task.baremetal.GetZoneId()), "zone_id")
	params.Add(jsonutils.NewString(task.baremetal.GetStorageCacheId()), "storagecache_id")
	_, err := modules.Hosts.PerformAction(task.getClientSession(), task.baremetal.GetId(), "update-storage", params)
	return err
}

func (task *sBaremetalPrepareTask) getCloudIsolatedDevices(
	session *mcclient.ClientSession,
) ([]jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("host", jsonutils.NewString(task.baremetal.GetId()))
	params.Set("scope", jsonutils.NewString("system"))
	params.Set("show_baremetal_isolated_devices", jsonutils.JSONTrue)
	res, err := modules.IsolatedDevices.List(session, params)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

func (task *sBaremetalPrepareTask) sendIsolatedDevicesInfo(
	session *mcclient.ClientSession, devs []*isolated_device.PCIDevice,
) error {
	objs, err := task.getCloudIsolatedDevices(session)
	if err != nil {
		return errors.Wrap(err, "get cloud isolated devices")
	}

	gpuDevs := make([]isolated_device.IDevice, len(devs))
	for i := 0; i < len(devs); i++ {
		gpuDevs[i] = isolated_device.NewGPUHPCDevice(devs[i])
	}

	for _, obj := range objs {
		var notFound = true
		info := isolated_device.CloudDeviceInfo{}
		if err := obj.Unmarshal(&info); err != nil {
			return errors.Wrap(err, "unmarshal isolated device to cloud device info failed")
		}
		for i := 0; i < len(gpuDevs); i++ {
			if gpuDevs[i].GetAddr() == info.Addr && gpuDevs[i].GetVendorDeviceId() == info.VendorDeviceId {
				gpuDevs[i].SetDeviceInfo(info)
				notFound = false
				break
			}
		}
		if notFound {
			_, err := modules.IsolatedDevices.PerformAction(session, info.Id, "purge", nil)
			if err != nil {
				return errors.Wrap(err, "purge unknown isolated devices")
			}
		}
	}

	for i := 0; i < len(gpuDevs); i++ {
		if _, err := isolated_device.SyncDeviceInfo(session, task.baremetal.GetId(), gpuDevs[i], true); err != nil {
			return errors.Wrap(err, "sync device info")
		}
	}
	return nil
}

func (task *sBaremetalPrepareTask) doNicWireProbe(cli *ssh.Client, nic *types.SNicDevInfo) error {
	maxTries := 6
	for tried := 0; tried < maxTries; tried++ {
		log.Infof("doNicWireProbe %v", nic)
		_, err := cli.Run(fmt.Sprintf("/sbin/udhcpc -t 1 -T 3 -n -i %s", nic.Dev))
		if err != nil {
			log.Errorf("/sbin/udhcpc error: %v", err)
		}
		nicInfo := task.baremetal.GetNicByMac(nic.Mac)
		if nicInfo != nil && nicInfo.WireId != "" {
			log.Infof("doNicWireProbe success, get result %#v", nicInfo)
			break
		}
	}
	return nil
}

func (task *sBaremetalPrepareTask) collectDiskInfo(diskInfo []*baremetal.BaremetalStorage) (int64, string) {
	cnt := 0
	rotateCnt := 0
	var size int64 = 0
	var diskType string
	for _, d := range diskInfo {
		if d.Rotate {
			rotateCnt += 1
		}
		size += d.Size
		cnt += 1
	}
	if rotateCnt == cnt {
		diskType = "rotate"
	} else if rotateCnt == 0 {
		diskType = "ssd"
	} else {
		diskType = "hybrid"
	}
	return size, diskType
}

func setIPMILanPortShared(cli ipmitool.IPMIExecutor, sysInfo *types.SSystemInfo) {
	if !o.Options.IpmiLanPortShared {
		return
	}
	oemName := strings.ToLower(sysInfo.Manufacture)
	var err error
	switch sysInfo.OemName {
	case types.OEM_NAME_HUAWEI:
		err = ipmitool.SetHuaweiIPMILanPortShared(cli)
	case types.OEM_NAME_DELL:
		err = ipmitool.SetDellIPMILanPortShared(cli)
	}
	if err != nil {
		log.Errorf("Set %s ipmi lan port shared failed: %v", oemName, err)
	}
}
