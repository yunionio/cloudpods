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

package ksyun

import (
	"fmt"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/jsonutils"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	instances, err := host.zone.region.GetInstances(host.zone.GetName(), []string{})
	if err != nil {
		return nil, err
	}
	ivms := make([]cloudprovider.ICloudVM, len(instances))
	for i := 0; i < len(instances); i += 1 {
		instances[i].host = host
		ivms[i] = &instances[i]
	}
	return ivms, nil
}

func (host *SHost) GetIVMById(vmId string) (cloudprovider.ICloudVM, error) {
	ins, err := host.zone.region.GetInstance(vmId)
	if err != nil {
		return nil, errors.Wrap(err, "GetInstance")
	}
	ins.host = host
	return ins, nil
}

func (host *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vm, err := host.zone.region.CreateVM(opts)
	if err != nil {
		return nil, err
	}
	vm.host = host
	return vm, nil
}

func (region *SRegion) syncKeypair(keyName, publicKey string) (string, error) {
	keypairs, err := region.client.GetKeypairs()
	if err != nil {
		return "", err
	}
	for i := range keypairs {
		if keypairs[i].PublicKey == publicKey {
			return keypairs[i].KeyId, nil
		}
	}
	keypair, err := region.client.CreateKeypair(keyName, publicKey)
	if err != nil {
		return "", err
	}
	return keypair.KeyId, nil
}

func (region *SRegion) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (*SInstance, error) {
	params := map[string]interface{}{
		"InstanceName":        opts.Name,
		"ImageId":             opts.ExternalImageId,
		"InstanceType":        opts.InstanceType,
		"MaxCount":            "1",
		"MinCount":            "1",
		"SubnetId":            opts.ExternalNetworkId,
		"ChargeType":          "HourlyInstantSettlement",
		"SystemDisk.DiskType": opts.SysDisk.StorageType,
		"SystemDisk.DiskSize": fmt.Sprintf("%d", opts.SysDisk.SizeGB),
		"SyncTag":             "true",
	}
	if utils.IsInStringArray(opts.KsyunPostpaidChargeType, []string{"HourlyInstantSettlement", "Daily"}) {
		params["ChargeType"] = opts.KsyunPostpaidChargeType
	}
	if len(opts.Hostname) > 0 {
		params["HostName"] = opts.Hostname
	}
	if len(opts.Password) > 0 {
		params["InstancePassword"] = opts.Password
	}
	if opts.BillingCycle != nil {
		params["ChargeType"] = "Monthly"
		params["PurchaseTime"] = fmt.Sprintf("%d", opts.BillingCycle.GetMonths())
	}
	for i, group := range opts.ExternalSecgroupIds {
		params[fmt.Sprintf("SecurityGroupId.%d", i+1)] = group
	}
	if len(opts.IpAddr) > 0 {
		params["PrivateIpAddress"] = opts.IpAddr
	}
	if len(opts.ProjectId) > 0 {
		params["ProjectId"] = opts.ProjectId
	}
	if len(opts.UserData) > 0 {
		params["UserData"] = opts.UserData
	}
	if len(opts.PublicKey) > 0 {
		var err error
		params["KeyId.1"], err = region.syncKeypair(opts.Name, opts.PublicKey)
		if err != nil {
			return nil, err
		}
	}
	for i, disk := range opts.DataDisks {
		params[fmt.Sprintf("DataDisk.%d.DeleteWithInstance", i+1)] = "true"
		params[fmt.Sprintf("DataDisk.%d.Size", i+1)] = fmt.Sprintf("%d", disk.SizeGB)
		params[fmt.Sprintf("DataDisk.%d.Type", i+1)] = disk.StorageType
	}
	tagIdx := 1
	for k, v := range opts.Tags {
		params[fmt.Sprintf("Tag.%d.Key", tagIdx)] = k
		params[fmt.Sprintf("Tag.%d.Value", tagIdx)] = v
		tagIdx++
	}
	resp, err := region.ecsRequest("RunInstances", params)
	if err != nil {
		return nil, errors.Wrapf(err, "RunInstances")
	}
	ret := []struct {
		InstanceId   string
		InstanceName string
	}{}
	err = resp.Unmarshal(&ret, "InstancesSet")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	if len(ret) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created")
	}
	vmId := ret[0].InstanceId
	err = cloudprovider.Wait(time.Second*3, time.Minute, func() (bool, error) {
		_, err := region.GetInstance(vmId)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after vm %s created", vmId)
	}
	return region.GetInstance(vmId)
}

func (host *SHost) GetAccessIp() string {
	return ""
}

func (host *SHost) IsEmulated() bool {
	return true
}

func (host *SHost) GetAccessMac() string {
	return ""
}

func (host *SHost) GetName() string {
	return host.zone.AvailabilityZone
}

func (host *SHost) GetNodeCount() int8 {
	return 0
}

func (host *SHost) GetSN() string {
	return ""
}

func (host *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (host *SHost) GetCpuCount() int {
	return 0
}

func (host *SHost) GetCpuDesc() string {
	return ""
}

func (host *SHost) GetCpuMhz() int {
	return 0
}

func (host *SHost) GetMemSizeMB() int {
	return 0
}

func (host *SHost) GetStorageSizeMB() int64 {
	return 0
}

func (host *SHost) GetStorageClass() string {
	return ""
}

func (host *SHost) GetStorageType() string {
	return ""
}

func (host *SHost) GetEnabled() bool {
	return true
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) GetGlobalId() string {
	return host.zone.GetId()
}

func (host *SHost) GetId() string {
	return host.zone.GetId()
}

func (host *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_KSYUN
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.zone.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (host *SHost) GetIStorageById(storageId string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(storageId)
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorages()
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_KSYUN_CN), "manufacture")
	return info
}

func (host *SHost) GetVersion() string {
	return ""
}
