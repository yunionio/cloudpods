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
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/profiles"
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
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

	IpmiLanChannel uint8

	AdminWire string
	IpmiWire  string

	accessNic *types.SNicDevInfo
}

func NewBaremetalRegisterTask(
	userCred mcclient.TokenCredential, bmManager IBmManager, sshCli *ssh.Client,
	hostname, remoteIp, ipmiUsername, ipmiPassword, ipmiIpAddr string,
	ipmiMac net.HardwareAddr, ipmiLanChannel uint8, adminWire, ipmiWire string) *sBaremetalRegisterTask {
	return &sBaremetalRegisterTask{
		sBaremetalPrepareTask: sBaremetalPrepareTask{userCred: userCred},
		BmManager:             bmManager,
		SshCli:                sshCli,
		Hostname:              hostname,
		RemoteIp:              remoteIp,
		IpmiUsername:          ipmiUsername,
		IpmiPassword:          ipmiPassword,
		IpmiIpAddr:            ipmiIpAddr,
		IpmiMac:               ipmiMac,
		IpmiLanChannel:        ipmiLanChannel,
		AdminWire:             adminWire,
		IpmiWire:              ipmiWire,
	}
}

func (s *sBaremetalRegisterTask) getSession() *mcclient.ClientSession {
	return auth.GetSession(context.Background(), s.userCred, o.Options.Region)
}

func (s *sBaremetalRegisterTask) getAccessDevMacAddr(ip string) (string, error) {
	nicsRet, err := s.SshCli.Run("/sbin/ip -o -4 addr show")
	if err != nil {
		return "", fmt.Errorf("Failed get access nic %s", err)
	}

	var dev string
	for i := 0; i < len(nicsRet); i++ {
		if strings.Contains(nicsRet[i], ip+"/") {
			segs := strings.Split(nicsRet[i], " ")
			if len(segs) > 1 {
				dev = segs[1]
				break
			}
		}
	}
	if len(dev) == 0 {
		return "", fmt.Errorf("Can't get access dev")
	}
	log.Infof("Access dev is %s", dev)
	macRet, err := s.SshCli.Run("/sbin/ip a show " + dev)
	if err != nil || len(macRet) < 2 {
		return "", fmt.Errorf("Failed get access nic mac address %s", err)
	}
	segs := strings.Fields(macRet[1])
	if len(segs) < 2 {
		return "", fmt.Errorf("Failed to find mac address")
	}
	return segs[1], nil
}

func (s *sBaremetalRegisterTask) CreateBaremetal(ctx context.Context) (string, error) {
	zoneId := s.BmManager.GetZoneId()
	ret, err := s.SshCli.Run("/lib/mos/lsnic")
	if err != nil {
		return "", fmt.Errorf("Register baremeatl failed on lsnic: %s", err)
	}

	accessMac, err := s.getAccessDevMacAddr(s.RemoteIp)
	if err != nil {
		return "", err
	}
	accessMacAddr, err := net.ParseMAC(accessMac)
	if err != nil {
		return "", fmt.Errorf("Failed parse access mac %s", accessMac)
	}

	nicinfo := sysutils.ParseNicInfo(ret)
	for _, nic := range nicinfo {
		if nic.Mac.String() == accessMacAddr.String() {
			s.accessNic = nic
		}
	}
	if s.accessNic == nil {
		s.accessNic = nicinfo[0]
	}

	params := jsonutils.NewDict()
	params.Set("name", jsonutils.NewString(fmt.Sprintf("bm%s", strings.Replace(s.accessNic.Mac.String(), ":", "", -1))))
	params.Set("access_mac", jsonutils.NewString(s.accessNic.Mac.String()))
	params.Set("host_type", jsonutils.NewString("baremetal"))
	params.Set("is_baremetal", jsonutils.JSONTrue)
	params.Set("is_import", jsonutils.JSONTrue)
	res, err := modules.Hosts.CreateInContext(s.getSession(), params, &modules.Zones, zoneId)
	if err != nil {
		return "", fmt.Errorf("Create baremetal failed: %s", err)
	}
	pxeBm, err := s.BmManager.AddBaremetal(ctx, res)
	if err != nil {
		return "", fmt.Errorf("BmManager add baremetal failed: %s", err)
	}

	err = pxeBm.InitAdminNetif(ctx,
		s.accessNic.Mac, s.AdminWire, api.NIC_TYPE_ADMIN, api.NETWORK_TYPE_PXE, true, s.RemoteIp)
	if err != nil {
		return "", fmt.Errorf("BmManager add admin netif failed: %s", err)
	}
	err = pxeBm.InitAdminNetif(ctx,
		s.IpmiMac, s.IpmiWire, api.NIC_TYPE_IPMI, api.NETWORK_TYPE_IPMI, true, s.IpmiIpAddr)
	if err != nil {
		return "", fmt.Errorf("BmManager add ipmi netif failed: %s", err)
	}
	for _, nic := range nicinfo {
		if nic.Dev != s.accessNic.Dev {
			pxeBm.RegisterNetif(ctx, nic.Mac, s.AdminWire)
		}
	}
	s.baremetal = pxeBm.(IBaremetal)
	bmInstanceId, _ := res.GetString("id")
	return bmInstanceId, nil
}

func (s *sBaremetalRegisterTask) UpdateBaremetal(ctx context.Context) (string, error) {
	accessMac, err := s.getAccessDevMacAddr(s.RemoteIp)
	if err != nil {
		return "", errors.Wrap(err, "getAccessDevMacAddr")
	}
	accessMacAddr, err := net.ParseMAC(accessMac)
	if err != nil {
		return "", errors.Wrapf(err, "Failed parse access mac %s", accessMac)
	}

	params := jsonutils.NewDict()
	params.Set("any_mac", jsonutils.NewString(accessMacAddr.String()))
	params.Set("scope", jsonutils.NewString("system"))
	res, err := modules.Hosts.List(s.BmManager.GetClientSession(), params)
	if err != nil {
		return "", errors.Wrap(err, "Fetch baremetal failed")
	}
	if len(res.Data) == 0 {
		return "", errors.Wrapf(errors.ErrNotFound, "Cann't find baremetal by access mac %s", accessMacAddr)
	}
	pxeBm, err := s.BmManager.AddBaremetal(ctx, res.Data[0])
	if err != nil {
		return "", errors.Wrap(err, "BmManager add baremetal failed")
	}

	s.baremetal = pxeBm.(IBaremetal)
	return s.baremetal.GetId(), nil
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

func (s *sBaremetalRegisterTask) DoPrepare(ctx context.Context, cli *ssh.Client, registered bool) error {
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

	s.updateIpmiInfo(ctx, cli)

	return s.updateBmInfo(ctx, cli, infos, registered)
}

func (s *sBaremetalRegisterTask) updateIpmiInfo(ctx context.Context, cli *ssh.Client) {
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
	up := true
	var nic = &types.SNicDevInfo{
		Up:    &up,
		Speed: 100,
		Mtu:   1500,
	}

	var conf *types.SIPMILanConfig
	profile, err := profiles.GetProfile(ctx, sysInfo)
	if profile != nil {
		for _, lanChannel := range profile.LanChannels {
			conf, _ = ipmitool.GetLanConfig(ipmiTool, lanChannel)
			if conf == nil || len(conf.Mac) == 0 {
				continue
			}
		}
	}

	if conf == nil || len(conf.Mac) == 0 {
		log.Errorln("Fail to get IPMI lan config !!!")
	} else {
		nic.Mac = conf.Mac
	}
	s.sendNicInfo(ctx, nic, -1, api.NIC_TYPE_IPMI, false, "", false)
}

func (s *sBaremetalRegisterTask) updateBmInfo(ctx context.Context, cli *ssh.Client, i *baremetalPrepareInfo, registered bool) error {
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
		err = s.sendNicInfo(ctx, i.nicsInfo[idx], idx, "", false, "", false)
		if err != nil {
			log.Errorf("Send nicinfo idx: %d, %#v error: %v", idx, i.nicsInfo[idx], err)
		}
	}
	if registered {
		return nil
	}
	return s.initBaremetalServer(ctx)
}

func (s *sBaremetalRegisterTask) initBaremetalServer(ctx context.Context) error {
	if err := s.baremetal.InitializeServer(s.getSession(), s.Hostname); err != nil {
		return fmt.Errorf("Baremteal Create Server Failed %s", err)
	}
	// if err := s.baremetal.SaveSSHConfig("", ""); err != nil {
	// 	log.Errorf("Save ssh config failed %s", err)
	// }
	if err := s.baremetal.ServerLoadDesc(ctx); err != nil {
		log.Errorf("Server load desc failed %s", err)
	}
	s.baremetal.SyncStatus(ctx, "running", "Register success")
	log.Infof("%s Load baremetal info success ...", s.baremetal.GetId())
	return nil
}
