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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/profiles"
	"yunion.io/x/onecloud/pkg/baremetal/utils/detect_storages"
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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
	sysInfo       *types.SDMISystemInfo
	cpuInfo       *types.SCPUInfo
	dmiCpuInfo    *types.SDMICPUInfo
	memInfo       *types.SDMIMemInfo
	nicsInfo      []*types.SNicDevInfo
	diskInfo      []*baremetal.BaremetalStorage
	storageDriver string
	ipmiInfo      *types.SIPMIInfo
}

func (task *sBaremetalPrepareTask) GetStartTime() time.Time {
	return task.startTime
}

func (task *sBaremetalPrepareTask) prepareBaremetalInfo(cli *ssh.Client) (*baremetalPrepareInfo, error) {
	_, err := cli.Run("/lib/mos/sysinit.sh")
	if err != nil {
		return nil, err
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
			raidDrivers = append(raidDrivers, drv.Driver)
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
	if !ipmiInfo.Present && ipmiEnable {
		ipmiInfo.Present = true
		ipmiInfo.Verified = false
	}

	return &baremetalPrepareInfo{
		sysInfo,
		cpuInfo,
		dmiCPUInfo,
		memInfo,
		nicsInfo,
		diskInfo,
		storageDriver,
		ipmiInfo,
	}, nil
}

func (task *sBaremetalPrepareTask) configIPMISetting(cli *ssh.Client, i *baremetalPrepareInfo) error {
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
	// ipmitool.SetSysInfo
	ipmiSysInfo := sysInfo.ToIPMISystemInfo()
	setIPMILanPortShared(sshIPMI, ipmiSysInfo)
	ipmiUser, ipmiPasswd, ipmiIpAddr := task.getIPMIUserPasswd(i.ipmiInfo, ipmiSysInfo)
	ipmiInfo.Username = ipmiUser
	ipmiInfo.Password = ipmiPasswd

	var ipmiLanChannel int = -1
	for _, lanChannel := range ipmitool.GetLanChannels(ipmiSysInfo) {
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
			Up:    false,
			Speed: 100,
			Mtu:   1500,
		}
		if err := task.sendNicInfo(ipmiNic, -1, api.NIC_TYPE_IPMI, true, "", false); err != nil {
			// ignore the error
			log.Errorf("Send IPMI nic %#v info: %v", ipmiNic, err)
		}
		rootId := ipmitool.GetRootId(ipmiSysInfo)
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
				tryResult := task.tryLocalIpmiAddr(sshIPMI, ipmiNic, lanChannel,
					ipmiUser, ipmiPasswd, tryAddr)
				if tryResult {
					ipmiInfo.IpAddr = tryAddr
					ipmiLanChannel = lanChannel
					break
				}
			}
			if ipmiLanChannel >= 0 {
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
	if ipmiLanChannel == -1 {
		return fmt.Errorf("Fail to get IPMI address from DHCP")
	}
	ipmiInfo.LanChannel = ipmiLanChannel
	ipmiInfo.Verified = true
	return nil
}

func (task *sBaremetalPrepareTask) DoPrepare(cli *ssh.Client) error {
	infos, err := task.prepareBaremetalInfo(cli)
	if err != nil {
		logclient.AddActionLogWithStartable(task, task.baremetal, logclient.ACT_PREPARE, err, task.userCred, false)
		return err
	}

	// set ipmi nic address and user password
	if err = task.configIPMISetting(cli, infos); err != nil {
		logclient.AddActionLogWithStartable(task, task.baremetal, logclient.ACT_PREPARE, err, task.userCred, false)
		return err
	}

	if err = task.updateBmInfo(cli, infos); err != nil {
		logclient.AddActionLogWithStartable(task, task.baremetal, logclient.ACT_PREPARE, err, task.userCred, false)
		return err
	}

	// set NTP
	if err = task.baremetal.DoNTPConfig(); err != nil {
		// ignore error
		log.Errorf("SetNTP fail: %s", err)
	}

	logclient.AddActionLogWithStartable(task, task.baremetal, logclient.ACT_PREPARE, infos.sysInfo, task.userCred, true)

	log.Infof("Prepare complete")
	return nil
}

func (task *sBaremetalPrepareTask) findAdminNic(cli *ssh.Client, nicsInfo []*types.SNicDevInfo) (int, *types.SNicDevInfo, error) {
	for idx := range nicsInfo {
		nic := nicsInfo[idx]
		output, err := cli.Run("/sbin/ifconfig " + nic.Dev)
		if err != nil {
			log.Errorf("ifconfig %s fail: %s", nic.Dev, err)
			continue
		}
		isAdmin := false
		for _, l := range output {
			if strings.Contains(l, task.baremetal.GetAccessIp()) {
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

func (task *sBaremetalPrepareTask) updateBmInfo(cli *ssh.Client, i *baremetalPrepareInfo) error {
	adminNic := task.baremetal.GetAdminNic()
	if adminNic == nil {
		adminIdx, adminNicDev, err := task.findAdminNic(cli, i.nicsInfo)
		if err != nil {
			return errors.Wrap(err, "task.findAdminNic")
		}
		err = task.sendNicInfo(adminNicDev, adminIdx, api.NIC_TYPE_ADMIN, false, task.baremetal.GetAccessIp(), true)
		if err != nil {
			return errors.Wrap(err, "send Admin Nic Info")
		}
		adminNic = task.baremetal.GetAdminNic()
	}
	// collect params
	updateInfo := make(map[string]interface{})
	oname := fmt.Sprintf("BM%s", strings.Replace(adminNic.Mac, ":", "", -1))
	if task.baremetal.GetName() == oname {
		updateInfo["name"] = fmt.Sprintf("BM-%s", strings.Replace(i.ipmiInfo.IpAddr, ".", "-", -1))
	}
	updateInfo["access_ip"] = adminNic.IpAddr
	updateInfo["access_mac"] = adminNic.Mac
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
		err = task.removeNicInfo(removedMacs[idx])
		if err != nil {
			log.Errorf("Fail to remove Netif %s: %s", removedMacs[idx], err)
			return errors.Wrap(err, "task.removeNicInfo")
		}
	}
	for idx := range i.nicsInfo {
		err = task.sendNicInfo(i.nicsInfo[idx], idx, "", false, "", false)
		if err != nil {
			log.Errorf("Send nicinfo idx: %d, %#v error: %v", idx, i.nicsInfo[idx], err)
			return errors.Wrap(err, "task.sendNicInfo")
		}
	}
	if o.Options.EnablePxeBoot && task.baremetal.EnablePxeBoot() {
		for _, nicInfo := range i.nicsInfo {
			if nicInfo.Mac.String() != adminNic.GetMac().String() && nicInfo.Up {
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
		if utils.IsInStringArray(existNics[idx].Type, api.NIC_TYPES) {
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

func (task *sBaremetalPrepareTask) tryLocalIpmiAddr(sshIPMI *ipmitool.SSHIPMI, ipmiNic *types.SNicDevInfo, lanChannel int, ipmiUser, ipmiPasswd, tryAddr string) bool {
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
		err := task.sendNicInfo(ipmiNic, -1, api.NIC_TYPE_IPMI, false, tryAddr, true)
		if err != nil {
			log.Errorf("Fail to set existing BMC IP address to %s", tryAddr)
		} else {
			return true
		}
	}
	return false
}

func (task *sBaremetalPrepareTask) getIPMIUserPasswd(oldIPMIConf *types.SIPMIInfo, sysInfo *types.SIPMISystemInfo) (string, string, string) {
	var (
		ipmiUser   string
		ipmiPasswd string
		ipmiIpAddr string
	)
	ipmiUser = profiles.GetRootName(sysInfo)
	isStrongPass := profiles.IsStrongPass(sysInfo)
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
	params.Add(jsonutils.JSONTrue, "is_on_premise")
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

func getDMISysinfo(cli *ssh.Client) (*types.SDMISystemInfo, error) {
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

func getNicsInfo(cli *ssh.Client) ([]*types.SNicDevInfo, error) {
	ret, err := cli.Run("/lib/mos/lsnic")
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve NIC info: %v", err)
	}
	return sysutils.ParseNicInfo(ret), nil
}

func isIPMIEnable(cli *ssh.Client) (bool, error) {
	ret, err := cli.Run("/usr/sbin/dmidecode -t 38")
	if err != nil {
		return false, fmt.Errorf("Failed to retrieve IPMI info: %v", err)
	}
	return sysutils.ParseDMIIPMIInfo(ret), nil
}

func (task *sBaremetalPrepareTask) removeNicInfo(mac string) error {
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
	return task.baremetal.SaveDesc(resp)
}

func (task *sBaremetalPrepareTask) sendNicInfo(nic *types.SNicDevInfo, idx int, nicType string, reset bool, ipAddr string, reserve bool) error {
	return task.baremetal.SendNicInfo(nic, idx, nicType, reset, ipAddr, reserve)
}

func (task *sBaremetalPrepareTask) sendStorageInfo(size int64) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(size), "capacity")
	params.Add(jsonutils.NewString(task.baremetal.GetZoneId()), "zone_id")
	params.Add(jsonutils.NewString(task.baremetal.GetStorageCacheId()), "storagecache_id")
	_, err := modules.Hosts.PerformAction(task.getClientSession(), task.baremetal.GetId(), "update-storage", params)
	return err
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

func setIPMILanPortShared(cli ipmitool.IPMIExecutor, sysInfo *types.SIPMISystemInfo) {
	if !o.Options.IpmiLanPortShared {
		return
	}
	oemName := strings.ToLower(sysInfo.Manufacture)
	var err error
	if strings.Contains(oemName, "huawei") {
		err = ipmitool.SetHuaweiIPMILanPortShared(cli)
	} else if strings.Contains(oemName, "dell") {
		err = ipmitool.SetDellIPMILanPortShared(cli)
	}
	if err != nil {
		log.Errorf("Set %s ipmi lan port shared failed: %v", oemName, err)
	}
}
