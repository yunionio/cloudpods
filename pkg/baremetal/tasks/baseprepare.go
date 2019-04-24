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
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/seclib"

	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/profiles"
	"yunion.io/x/onecloud/pkg/baremetal/utils/detect_storages"
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type sBaremetalPrepareTask struct {
	baremetal IBaremetal
}

func newBaremetalPrepareTask(baremetal IBaremetal) *sBaremetalPrepareTask {
	return &sBaremetalPrepareTask{
		baremetal: baremetal,
	}
}

func (task *sBaremetalPrepareTask) DoPrepare(cli *ssh.Client) error {
	_, err := cli.Run("/lib/mos/sysinit.sh")
	if err != nil {
		return err
	}

	sysInfo, err := getDMISysinfo(cli)
	if err != nil {
		return err
	}
	cpuInfo, err := getCPUInfo(cli)
	if err != nil {
		return err
	}
	dmiCPUInfo, err := getDMICPUInfo(cli)
	if err != nil {
		return err
	}
	memInfo, err := getDMIMemInfo(cli)
	if err != nil {
		return err
	}
	nicsInfo, err := getNicsInfo(cli)
	if err != nil {
		return err
	}

	raidDiskInfo, nonRaidDiskInfo, pcieDiskInfo, err := detect_storages.DetectStorageInfo(cli, true)
	if err != nil {
		return err
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
		return err
	}

	ipmiInfo := &types.SIPMIInfo{
		Present: ipmiEnable,
	}
	// set ipmi nic DHCP
	if ipmiEnable {
		sshIPMI := ipmitool.NewSSHIPMI(cli)
		// ipmitool.SetSysInfo
		ipmiSysInfo := sysInfo.ToIPMISystemInfo()
		SetIPMILanPortShared(sshIPMI, ipmiSysInfo)
		ipmiUser, ipmiPasswd, ipmiIpAddr := task.getIPMIUserPasswd(ipmiSysInfo)
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
			task.sendNicInfo(ipmiNic, -1, types.NIC_TYPE_IPMI, true, "")
			rootId := ipmitool.GetRootId(ipmiSysInfo)
			err = ipmitool.SetLanUserPasswd(sshIPMI, lanChannel, rootId, ipmiUser, ipmiPasswd)
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
				log.Errorf("Set lan channel %d dhcp error: %v", lanChannel, err)
			}
			time.Sleep(1 * time.Second)
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
			}
			for tried := 0; nic != nil && nic.IpAddr == "" && tried < maxTries; tried++ {
				nic = task.baremetal.GetIPMINic(conf.Mac)
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
				log.Infof("waiting IPMI DHCP address %s %s", lanConf.IPAddr, nic.IpAddr)
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
	}

	adminNic := task.baremetal.GetAdminNic()

	// collect params
	updateInfo := make(map[string]interface{})
	oname := fmt.Sprintf("BM%s", strings.Replace(adminNic.Mac, ":", "", -1))
	if task.baremetal.GetName() == oname {
		updateInfo["name"] = fmt.Sprintf("BM-%s", strings.Replace(ipmiInfo.IpAddr, ".", "-", -1))
	}
	updateInfo["access_ip"] = adminNic.IpAddr
	updateInfo["cpu_count"] = cpuInfo.Count
	updateInfo["node_count"] = dmiCPUInfo.Nodes
	updateInfo["cpu_desc"] = cpuInfo.Model
	updateInfo["cpu_mhz"] = cpuInfo.Freq
	updateInfo["cpu_cache"] = cpuInfo.Cache
	updateInfo["mem_size"] = memInfo.Total
	updateInfo["storage_driver"] = storageDriver
	updateInfo["storage_info"] = diskInfo
	updateInfo["sys_info"] = sysInfo
	updateInfo["sn"] = sysInfo.SN
	size, diskType := task.collectDiskInfo(diskInfo)
	updateInfo["storage_size"] = size
	updateInfo["storage_type"] = diskType
	updateData := jsonutils.Marshal(updateInfo)
	updateData.(*jsonutils.JSONDict).Update(ipmiInfo.ToPrepareParams())
	_, err = modules.Hosts.Update(task.getClientSession(), task.baremetal.GetId(), updateData)
	if err != nil {
		log.Errorf("Update baremetal info error: %v", err)
	}
	if err := task.sendStorageInfo(size); err != nil {
		log.Errorf("sendStorageInfo error: %v", err)
	}
	for i := range nicsInfo {
		if nicsInfo[i].Mac.String() == adminNic.GetMac().String() {
			if i != 0 {
				nicsInfo = append(nicsInfo[i:], nicsInfo[0:i]...)
			}
			break
		}
	}
	err = task.removeAllNics()
	if err != nil {
		return err
	}
	for i := range nicsInfo {
		err = task.sendNicInfo(nicsInfo[i], i, "", false, "")
		if err != nil {
			log.Errorf("Send nicinfo idx: %d, %#v error: %v", i, nicsInfo[i], err)
		}
	}
	for _, nicInfo := range nicsInfo {
		if nicInfo.Mac.String() != adminNic.GetMac().String() && nicInfo.Up {
			err = task.doNicWireProbe(cli, nicInfo)
			if err != nil {
				log.Errorf("doNicWireProbe nic %#v error: %v", nicInfo, err)
			}
		}
	}

	log.Infof("Prepare complete")
	return nil
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
		err := task.sendNicInfo(ipmiNic, -1, types.NIC_TYPE_IPMI, false, tryAddr)
		if err != nil {
			log.Errorf("Fail to set existing BMC IP address to %s", tryAddr)
		} else {
			return true
		}
	}
	return false
}

func (task *sBaremetalPrepareTask) getIPMIUserPasswd(sysInfo *types.SIPMISystemInfo) (string, string, string) {
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
	oldIPMIConf := task.baremetal.GetRawIPMIConfig()
	if oldIPMIConf != nil {
		if oldIPMIConf.Username != "" {
			ipmiUser = oldIPMIConf.Username
		}
		if oldIPMIConf.Password != "" {
			ipmiPasswd = oldIPMIConf.Password
		}
		if oldIPMIConf.IpAddr != "" {
			ipmiIpAddr = oldIPMIConf.IpAddr
		}
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

func (task *sBaremetalPrepareTask) removeAllNics() error {
	resp, err := modules.Hosts.PerformAction(
		task.getClientSession(),
		task.baremetal.GetId(),
		"remove-all-netifs",
		nil,
	)
	if err != nil {
		return nil
	}
	return task.baremetal.SaveDesc(resp)
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

func (task *sBaremetalPrepareTask) sendNicInfo(nic *types.SNicDevInfo, idx int, nicType string, reset bool, ipAddr string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(nic.Mac.String()), "mac")
	params.Add(jsonutils.NewInt(int64(nic.Speed)), "rate")
	if idx >= 0 {
		params.Add(jsonutils.NewInt(int64(idx)), "index")
	}
	if nicType != "" {
		params.Add(jsonutils.NewString(nicType), "nic_type")
	}
	params.Add(jsonutils.NewInt(int64(nic.Mtu)), "mtu")
	params.Add(jsonutils.NewBool(nic.Up), "link_up")
	if reset {
		params.Add(jsonutils.JSONTrue, "reset")
	}
	if ipAddr != "" {
		params.Add(jsonutils.NewString(ipAddr), "ip_addr")
		params.Add(jsonutils.JSONTrue, "require_designated_ip")
	}
	resp, err := modules.Hosts.PerformAction(
		task.getClientSession(),
		task.baremetal.GetId(),
		"add-netif",
		params,
	)
	if err != nil {
		return err
	}
	task.baremetal.SaveDesc(resp)
	return nil
}

func (task *sBaremetalPrepareTask) sendStorageInfo(size int64) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(size), "capacity")
	params.Add(jsonutils.NewString(task.baremetal.GetZoneId()), "zone_id")
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

func SetIPMILanPortShared(cli ipmitool.IPMIExecutor, sysInfo *types.SIPMISystemInfo) {
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
