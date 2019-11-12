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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

type sBaremetalRegisterTask struct {
	sBaremetalPrepareTask

	BmManager IBmManager
	SshCli    *ssh.Client

	Hostname string
	RemoteIp string

	IpmiUsername string
	IpmiPassword string
	IpmiIpAddr   string
	IpmiMac      net.HardwareAddr

	IpmiLanChannel int

	AdminWire string
	IpmiWire  string

	accessNic *types.SNicDevInfo
}

func NewBaremetalRegisterTask(bmManager IBmManager, sshCli *ssh.Client,
	hostname, remoteIp, ipmiUsername, ipmiPassword, ipmiIpAddr string,
	ipmiMac net.HardwareAddr, ipmiLanChannel int, adminWire, ipmiWire string) *sBaremetalRegisterTask {
	return &sBaremetalRegisterTask{
		BmManager:      bmManager,
		SshCli:         sshCli,
		Hostname:       hostname,
		RemoteIp:       remoteIp,
		IpmiUsername:   ipmiUsername,
		IpmiPassword:   ipmiPassword,
		IpmiIpAddr:     ipmiIpAddr,
		IpmiMac:        ipmiMac,
		IpmiLanChannel: ipmiLanChannel,
		AdminWire:      adminWire,
		IpmiWire:       ipmiWire,
	}
}

func (s *sBaremetalRegisterTask) CreateBaremetal() error {
	zoneId := s.BmManager.GetZoneId()
	ret, err := s.SshCli.Run("/lib/mos/lsnic")
	if err != nil {
		return fmt.Errorf("Register baremeatl failed on lsnic: %s", err)
	}

	nicinfo := sysutils.ParseNicInfo(ret)
	for _, nic := range nicinfo {
		ret, err := s.SshCli.RawRun("/sbin/ip a show " + nic.Dev)
		if err != nil {
			return fmt.Errorf("Register baremeatl failed on ip command: %s", err)
		}
		if strings.Index(ret[0], s.RemoteIp) >= 0 {
			s.accessNic = nic
			break
		} else {
			continue
		}
	}
	if s.accessNic == nil {
		s.accessNic = nicinfo[0]
	}

	params := jsonutils.NewDict()
	params.Set("name", jsonutils.NewString("BM"+strings.Replace(s.accessNic.Mac.String(), ":", "", -1)))
	params.Set("access_mac", jsonutils.NewString(s.accessNic.Mac.String()))
	params.Set("host_type", jsonutils.NewString("baremetal"))
	params.Set("is_baremetal", jsonutils.JSONTrue)
	params.Set("is_import", jsonutils.JSONTrue)
	res, err := modules.Hosts.CreateInContext(s.BmManager.GetClientSession(), params, &modules.Zones, zoneId)
	if err != nil {
		return fmt.Errorf("Create baremetal failed: %s", err)
	}
	pxeBm, err := s.BmManager.AddBaremetal(res)
	if err != nil {
		return fmt.Errorf("BmManager add baremetal failed: %s", err)
	}
	err = pxeBm.InitAdminNetif(
		s.accessNic.Mac, s.AdminWire, api.NIC_TYPE_ADMIN, api.NETWORK_TYPE_PXE, true, s.RemoteIp)
	if err != nil {
		return fmt.Errorf("BmManager add admin netif failed: %s", err)
	}
	err = pxeBm.InitAdminNetif(
		s.IpmiMac, s.IpmiWire, api.NIC_TYPE_IPMI, api.NETWORK_TYPE_IPMI, true, s.IpmiIpAddr)
	if err != nil {
		return fmt.Errorf("BmManager add ipmi netif failed: %s", err)
	}
	for _, nic := range nicinfo {
		if nic.Dev != s.accessNic.Dev {
			pxeBm.RegisterNetif(nic.Mac, s.AdminWire)
		}
	}
	s.baremetal = pxeBm.(IBaremetal)
	return nil
}

func (s *sBaremetalRegisterTask) update() {

}

func (s *sBaremetalRegisterTask) doRedfishProbe(ctx context.Context) (redfishSupport bool, cdromBoot bool) {
	redfishCli := redfish.NewRedfishDriver(ctx, "https://"+s.IpmiIpAddr, s.IpmiUsername, s.IpmiPassword, false)
	if redfishCli != nil {
		_, cdInfo, _ := redfishCli.GetVirtualCdromInfo(ctx)
		redfishSupport = true
		cdromBoot = cdInfo.SupportAction
	}
	return
}

func (s *sBaremetalRegisterTask) DoPrepare(ctx context.Context, cli *ssh.Client) error {
	infos, err := s.prepareBaremetalInfo(cli)
	if err != nil {
		return err
	}

	redfishSupport, cdromSupport := s.doRedfishProbe(ctx)

	infos.ipmiInfo.IpAddr = s.IpmiIpAddr
	infos.ipmiInfo.Username = s.IpmiUsername
	infos.ipmiInfo.Password = s.IpmiPassword
	infos.ipmiInfo.LanChannel = s.IpmiLanChannel
	infos.ipmiInfo.Verified = true
	infos.ipmiInfo.Present = true
	infos.ipmiInfo.RedfishApi = redfishSupport
	infos.ipmiInfo.CdromBoot = cdromSupport

	s.updateIpmiInfo(cli)

	return s.updateBmInfo(cli, infos)
}

func (s *sBaremetalRegisterTask) updateIpmiInfo(cli *ssh.Client) {
	ipmiTool := ipmitool.NewSSHIPMI(cli)
	sysInfo, err := ipmitool.GetSysInfo(ipmiTool)
	if err == nil && o.Options.IpmiLanPortShared {
		oemName := strings.ToLower(sysInfo.Manufacture)
		if strings.Contains(oemName, "huawei") {
			ipmitool.SetHuaweiIPMILanPortShared(ipmiTool)
		} else if strings.Contains(oemName, "dell") {
			ipmitool.SetDellIPMILanPortShared(ipmiTool)
		}
	}
	var nic = &types.SNicDevInfo{
		Up:    true,
		Speed: 100,
		Mtu:   1500,
	}

	var conf *types.SIPMILanConfig
	for _, lanChannel := range ipmitool.GetLanChannels(sysInfo) {
		conf, _ = ipmitool.GetLanConfig(ipmiTool, lanChannel)
		if conf == nil || len(conf.Mac) == 0 {
			continue
		}
	}
	if conf == nil || len(conf.Mac) == 0 {
		log.Errorln("Fail to get IPMI lan config !!!")
	} else {
		nic.Mac = conf.Mac
	}
	s.sendNicInfo(nic, -1, api.NIC_TYPE_IPMI, false, "", false)
}

func (s *sBaremetalRegisterTask) updateBmInfo(cli *ssh.Client, i *baremetalPrepareInfo) error {
	updateInfo := make(map[string]interface{})
	updateInfo["access_ip"] = s.RemoteIp
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
	size, diskType := s.collectDiskInfo(i.diskInfo)
	updateInfo["storage_size"] = size
	updateInfo["storage_type"] = diskType
	updateData := jsonutils.Marshal(updateInfo)
	updateData.(*jsonutils.JSONDict).Update(i.ipmiInfo.ToPrepareParams())
	_, err := modules.Hosts.Update(s.getClientSession(), s.baremetal.GetId(), updateData)
	if err != nil {
		log.Errorf("Update baremetal info error: %v", err)
	}
	if err := s.sendStorageInfo(size); err != nil {
		log.Errorf("sendStorageInfo error: %v", err)
	}
	for idx := range i.nicsInfo {
		err = s.sendNicInfo(i.nicsInfo[idx], idx, "", false, "", false)
		if err != nil {
			log.Errorf("Send nicinfo idx: %d, %#v error: %v", idx, i.nicsInfo[idx], err)
		}
	}
	if err := s.baremetal.InitializeServer(s.Hostname); err != nil {
		return fmt.Errorf("Baremteal Create Server Failed %s", err)
	}
	// if err := s.baremetal.SaveSSHConfig("", ""); err != nil {
	// 	log.Errorf("Save ssh config failed %s", err)
	// }
	if err := s.baremetal.ServerLoadDesc(); err != nil {
		log.Errorf("Server load desc failed %s", err)
	}
	s.baremetal.SyncStatus("running", "Register success")
	log.Infof("%s Load baremetal info success ...", s.baremetal.GetId())
	return nil
}
