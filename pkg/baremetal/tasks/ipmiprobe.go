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
	"net"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/redfish"
)

type SBaremetalIpmiProbeTask struct {
	SBaremetalTaskBase
}

func NewBaremetalIpmiProbeTask(
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalIpmiProbeTask{
		SBaremetalTaskBase: newBaremetalTaskBase(baremetal, taskId, data),
	}
	task.SetVirtualObject(task)
	task.SetStage(task.DoIpmiProbe)
	return task
}

func (self *SBaremetalIpmiProbeTask) GetName() string {
	return "BaremetalIpmiProbeTask"
}

func (self *SBaremetalIpmiProbeTask) DoIpmiProbe(ctx context.Context, args interface{}) error {
	ipmiInfo := self.Baremetal.GetRawIPMIConfig()
	if ipmiInfo == nil {
		ipmiInfo = &types.SIPMIInfo{}
	}
	if ipmiInfo.IpAddr == "" {
		return errors.Error("empty IPMI ip_addr")
	}
	if ipmiInfo.Username == "" {
		return errors.Error("empty IPMI username")
	}
	if ipmiInfo.Password == "" {
		return errors.Error("empty IPMI password")
	}
	redfishCli := redfish.NewRedfishDriver(ctx, "https://"+ipmiInfo.IpAddr, ipmiInfo.Username, ipmiInfo.Password, false)
	if redfishCli != nil {
		return self.doRedfishIpmiProbe(ctx, redfishCli)
	} else {
		log.Warningf("BMC not redfish-compatible")
		ipmiTool := ipmitool.NewLanPlusIPMI(ipmiInfo.IpAddr, ipmiInfo.Username, ipmiInfo.Password)
		return self.doRawIpmiProbe(ctx, ipmiTool)
	}
}

func (self *SBaremetalIpmiProbeTask) doRedfishIpmiProbe(ctx context.Context, drv redfish.IRedfishDriver) error {
	confs, err := drv.GetLanConfigs(ctx)
	if err != nil {
		return errors.Wrap(err, "drv.GetLanConfigs")
	}
	if len(confs) == 0 {
		return errors.Wrap(httperrors.ErrNotFound, "no IPMI lan")
	}
	err = self.sendIpmiNicInfo(&confs[0])
	if err != nil {
		return errors.Wrap(err, "self.sendIpmiNicInfo")
	}
	_, sysInfo, err := drv.GetSystemInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "drv.GetSystemInfo")
	}
	updateInfo := make(map[string]interface{})
	if len(sysInfo.EthernetNICs) > 0 {
		updateInfo["access_mac"] = sysInfo.EthernetNICs[0]
	}
	updateInfo["node_count"] = sysInfo.NodeCount
	updateInfo["cpu_count"] = sysInfo.NodeCount
	updateInfo["cpu_desc"] = sysInfo.CpuDesc
	updateInfo["mem_size"] = sysInfo.MemoryGB * 1024
	updateInfo["sn"] = sysInfo.SerialNumber
	updateInfo["uuid"] = sysInfo.UUID
	dmiSysInfo := &types.SDMISystemInfo{
		Manufacture: sysInfo.Manufacturer,
		Model:       sysInfo.Model,
		SN:          sysInfo.SerialNumber,
	}
	updateInfo["sys_info"] = dmiSysInfo
	updateInfo["is_baremetal"] = true
	ipmiInfo := self.Baremetal.GetRawIPMIConfig()
	if ipmiInfo == nil {
		ipmiInfo = &types.SIPMIInfo{}
	}
	ipmiInfo.Present = true
	ipmiInfo.Verified = true
	ipmiInfo.RedfishApi = true
	_, cdInfo, _ := drv.GetVirtualCdromInfo(ctx)
	ipmiInfo.CdromBoot = cdInfo.SupportAction
	ipmiInfo.PxeBoot = o.Options.EnablePxeBoot
	updateData := jsonutils.Marshal(updateInfo)
	updateData.(*jsonutils.JSONDict).Update(ipmiInfo.ToPrepareParams())
	_, err = modules.Hosts.Update(self.Baremetal.GetClientSession(), self.Baremetal.GetId(), updateData)
	if err != nil {
		log.Errorf("Update baremetal info error: %v", err)
		return errors.Wrap(err, "modules.Hosts.Update")
	}
	for i := range sysInfo.EthernetNICs {
		mac, err := net.ParseMAC(sysInfo.EthernetNICs[i])
		if err == nil {
			err = self.sendNicInfo(i, mac)
			if err != nil {
				return errors.Wrapf(err, "sendNicInfo %d %s", i, mac)
			}
		}
	}
	self.Baremetal.SyncStatus("", "Probe Redfish finished")
	SetTaskComplete(self, nil)
	return nil
}

func (self *SBaremetalIpmiProbeTask) sendIpmiNicInfo(lanConf *types.SIPMILanConfig) error {
	speed := lanConf.SpeedMbps
	if speed <= 0 {
		speed = 100
	}
	ipmiNic := &types.SNicDevInfo{
		Mac:   lanConf.Mac,
		Up:    true,
		Speed: speed,
		Mtu:   1500,
	}
	err := self.Baremetal.SendNicInfo(ipmiNic, -1, api.NIC_TYPE_IPMI, true, lanConf.IPAddr, true)
	if err != nil {
		return errors.Wrap(err, "SendNicInfo")
	}
	return nil
}

func (self *SBaremetalIpmiProbeTask) sendNicInfo(index int, mac net.HardwareAddr) error {
	nicInfo := &types.SNicDevInfo{
		Mac: mac,
	}
	err := self.Baremetal.SendNicInfo(nicInfo, index, "", false, "", false)
	if err != nil {
		return errors.Wrap(err, "SendNicInfo")
	}
	return nil
}

func (self *SBaremetalIpmiProbeTask) doRawIpmiProbe(ctx context.Context, cli ipmitool.IPMIExecutor) error {
	sysInfo, err := ipmitool.GetSysInfo(cli)
	if err != nil {
		// ignore error for qemu
		log.Errorf("ipmitool.GetSysInfo error %s", err)
	}
	guid := ipmitool.GetSysGuid(cli)
	var conf *types.SIPMILanConfig
	var channel int
	for _, lanChannel := range ipmitool.GetLanChannels(sysInfo) {
		conf, err = ipmitool.GetLanConfig(cli, lanChannel)
		if err != nil {
			// ignore error
			log.Errorf("ipmitool.GetLanConfig for channel %d fail: %s", lanChannel, err)
		} else {
			channel = lanChannel
			break
		}
	}
	if conf == nil {
		return errors.Wrap(httperrors.ErrNotFound, "no IPMI lan")
	}
	err = self.sendIpmiNicInfo(conf)
	if err != nil {
		return errors.Wrap(err, "self.sendIpmiNicInfo")
	}
	updateInfo := make(map[string]interface{})
	if len(sysInfo.SN) > 0 {
		updateInfo["sn"] = sysInfo.SN
		dmiSysInfo := &types.SDMISystemInfo{
			Manufacture: sysInfo.Manufacture,
			Model:       sysInfo.Model,
			Version:     sysInfo.Version,
			SN:          sysInfo.SN,
		}
		updateInfo["sys_info"] = dmiSysInfo
	}
	// XXX
	// Qemu's IPMI guid is not correct, just ignore it
	if len(guid) > 0 && sysInfo.SN != "" && sysInfo.SN != "Not Specified" {
		updateInfo["uuid"] = guid
	}
	updateInfo["is_baremetal"] = true
	ipmiInfo := self.Baremetal.GetRawIPMIConfig()
	if ipmiInfo == nil {
		ipmiInfo = &types.SIPMIInfo{}
	}
	ipmiInfo.Present = true
	ipmiInfo.Verified = true
	ipmiInfo.RedfishApi = false
	ipmiInfo.CdromBoot = false
	ipmiInfo.PxeBoot = o.Options.EnablePxeBoot
	ipmiInfo.LanChannel = channel
	updateData := jsonutils.Marshal(updateInfo)
	updateData.(*jsonutils.JSONDict).Update(ipmiInfo.ToPrepareParams())
	_, err = modules.Hosts.Update(self.Baremetal.GetClientSession(), self.Baremetal.GetId(), updateData)
	if err != nil {
		return errors.Wrap(err, "modules.Hosts.Update")
	}
	self.Baremetal.SyncStatus("", "Probie IPMI finished")
	SetTaskComplete(self, nil)
	return nil
}
