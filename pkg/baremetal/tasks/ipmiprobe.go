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
	baremetalapi "yunion.io/x/onecloud/pkg/apis/compute/baremetal"
	o "yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/profiles"
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/redfish"
)

type SBaremetalIpmiProbeTask struct {
	SBaremetalTaskBase
}

func NewBaremetalIpmiProbeTask(
	userCred mcclient.TokenCredential,
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalIpmiProbeTask{
		SBaremetalTaskBase: newBaremetalTaskBase(userCred, baremetal, taskId, data),
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
	// FIXME: 通过 redfish 探测出来的管理口网卡地址可能和 PXE 启动的网卡地址不一致
	// 所以暂时注释掉这个探测逻辑，使用 doRawIpmiProbe
	/* redfishCli := redfish.NewRedfishDriver(ctx, "https://"+ipmiInfo.IpAddr, ipmiInfo.Username, ipmiInfo.Password, false)
	if redfishCli != nil {
		redfishSuccess, err := self.doRedfishIpmiProbe(ctx, redfishCli)
		if err == nil {
			// success
			return nil
		}
		if redfishSuccess {
			return errors.Wrap(err, "doRedfishIpmiProbe")
		}
		// else, redfish call fails, try IPMI
	} */
	log.Warningf("BMC not redfish-compatible for IPMI: %s, use raw probe", ipmiInfo.IpAddr)
	ipmiTool := ipmitool.NewLanPlusIPMI(ipmiInfo.IpAddr, ipmiInfo.Username, ipmiInfo.Password)
	return self.doRawIpmiProbe(ctx, ipmiTool)
}

// return redfishSuccess, error
// redfishSuccess: does Redfish API call success
func (self *SBaremetalIpmiProbeTask) doRedfishIpmiProbe(ctx context.Context, drv redfish.IRedfishDriver) (bool, error) {
	confs, err := drv.GetLanConfigs(ctx)
	if err != nil {
		return false, errors.Wrap(err, "drv.GetLanConfigs")
	}
	if len(confs) == 0 {
		return false, errors.Wrap(httperrors.ErrNotFound, "no IPMI lan")
	}
	err = self.sendIpmiNicInfo(ctx, &confs[0])
	if err != nil {
		return false, errors.Wrap(err, "self.sendIpmiNicInfo")
	}
	_, sysInfo, err := drv.GetSystemInfo(ctx)
	if err != nil {
		return false, errors.Wrap(err, "drv.GetSystemInfo")
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
	dmiSysInfo := &types.SSystemInfo{
		Manufacture: sysInfo.Manufacturer,
		Model:       sysInfo.Model,
		SN:          sysInfo.SerialNumber,
		OemName:     types.ManufactureOemName(sysInfo.Manufacturer),
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
		return true, errors.Wrap(err, "modules.Hosts.Update")
	}
	for i := range sysInfo.EthernetNICs {
		mac, err := net.ParseMAC(sysInfo.EthernetNICs[i])
		if err == nil {
			err = self.sendNicInfo(ctx, i, mac)
			if err != nil {
				return true, errors.Wrapf(err, "sendNicInfo %d %s", i, mac)
			}
		}
	}
	self.Baremetal.SyncStatus(ctx, "", "Probe Redfish finished")
	SetTaskComplete(self, nil)
	return true, nil
}

func (self *SBaremetalIpmiProbeTask) sendIpmiNicInfo(ctx context.Context, lanConf *types.SIPMILanConfig) error {
	speed := lanConf.SpeedMbps
	if speed <= 0 {
		speed = 100
	}
	up := true
	ipmiNic := &types.SNicDevInfo{
		Mac:   lanConf.Mac,
		Up:    &up,
		Speed: speed,
		Mtu:   1500,
	}
	err := self.Baremetal.SendNicInfo(ctx, ipmiNic, -1, api.NIC_TYPE_IPMI, true, lanConf.IPAddr, true)
	if err != nil {
		return errors.Wrap(err, "SendNicInfo")
	}
	return nil
}

func (self *SBaremetalIpmiProbeTask) sendNicInfo(ctx context.Context, index int, mac net.HardwareAddr) error {
	nicInfo := &types.SNicDevInfo{
		Mac: mac,
	}
	err := self.Baremetal.SendNicInfo(ctx, nicInfo, index, "", false, "", false)
	if err != nil {
		return errors.Wrap(err, "SendNicInfo")
	}
	return nil
}

func (self *SBaremetalIpmiProbeTask) doRawIpmiProbe(ctx context.Context, cli ipmitool.IPMIExecutor) error {
	sysInfo, err := ipmitool.GetSysInfo(cli)
	var profile *baremetalapi.BaremetalProfileSpec
	if err != nil {
		// ignore error for qemu
		log.Errorf("ipmitool.GetSysInfo error %s", err)
	} else {
		profile, err = profiles.GetProfile(ctx, sysInfo)
		if err != nil {
			return errors.Wrap(err, "GetProfile")
		}
	}
	guid := ipmitool.GetSysGuid(cli)
	var conf *types.SIPMILanConfig
	var channel uint8
	var errs []error
	for _, lanChannel := range profile.LanChannels {
		conf, err = ipmitool.GetLanConfig(cli, lanChannel)
		if err != nil {
			// ignore error
			err := errors.Wrapf(err, "ipmitool.GetLanConfig for channel %d failed", lanChannel)
			errs = append(errs, err)
			log.Warningf(err.Error())
		} else if conf.IPAddr == "0.0.0.0" {
			err := errors.Errorf("get 0.0.0.0 ip address of channel %d", lanChannel)
			errs = append(errs, err)
			log.Warningf(err.Error())
			continue
		} else {
			channel = lanChannel
			break
		}
	}
	if conf == nil {
		return errors.Wrapf(httperrors.ErrNotFound, "no IPMI lan: %s", errors.NewAggregate(errs).Error())
	}
	err = self.sendIpmiNicInfo(ctx, conf)
	if err != nil {
		return errors.Wrap(err, "self.sendIpmiNicInfo")
	}
	updateInfo := make(map[string]interface{})
	if len(sysInfo.SN) > 0 {
		updateInfo["sn"] = sysInfo.SN
		dmiSysInfo := &types.SSystemInfo{
			Manufacture: sysInfo.Manufacture,
			Model:       sysInfo.Model,
			Version:     sysInfo.Version,
			SN:          sysInfo.SN,
			OemName:     types.ManufactureOemName(sysInfo.Manufacture),
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
	self.Baremetal.SyncStatus(ctx, "", "Probie IPMI finished")
	SetTaskComplete(self, nil)
	return nil
}
