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

package storageman

import (
	"context"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/cloudmux/pkg/multicloud/proxmox"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SProxmoxDisk struct {
	SLocalDisk
}

func NewProxmoxDisk(storage IStorage, id string) *SProxmoxDisk {
	return &SProxmoxDisk{*NewLocalDisk(storage, id)}
}

func (sd *SProxmoxDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	p, ok := params.(PrepareSaveToGlanceParams)
	if !ok {
		return nil, errors.Wrap(hostutils.ParamsError, "Resize params format error")
	}
	storage := sd.Storage.(*SAgentStorage)
	return storage.PrepareSaveToGlance(ctx, p.TaskId, p.DiskInfo)
}

func (sd *SProxmoxDisk) Resize(ctx context.Context, params *SDiskResizeInput) (jsonutils.JSONObject, error) {
	body := params.DiskInfo
	type sResize struct {
		SizeMb   int64 `json:"size_mb"`
		HostInfo vcenter.SVCenterAccessInfo
		VMId     string `json:"vm_id"`
		DiskId   string `json:"disk_id"`
	}
	resize := sResize{}
	err := body.Unmarshal(&resize)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: unmarshal to sResize", hostutils.ParamsError)
	}

	client, err := proxmox.NewProxmoxClientFromAccessInfo(&resize.HostInfo)
	if err != nil {
		return nil, httperrors.NewInputParameterError("info of host_info error")
	}
	host, err := client.GetHost(resize.HostInfo.PrivateId)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to find host by ip %s", resize.HostInfo.PrivateId)
	}
	ivm, err := host.GetIVMById(resize.VMId)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to find vm by ID %s", resize.VMId)
	}
	disks, err := ivm.GetIDisks()
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get idisks of vm %s", resize.VMId)
	}
	var disk cloudprovider.ICloudDisk = nil
	for i := range disks {
		if disks[i].GetGlobalId() == resize.DiskId {
			disk = disks[i]
			break
		}
	}
	if disk == nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "fail to find disk by ID %s", resize.DiskId)
	}
	err = disk.Resize(ctx, resize.SizeMb)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to resize disk %s", resize.DiskId)
	}

	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewInt(resize.SizeMb), "disk_size")
	// try to resize partition
	err = resizeProxmoxPartition(ctx, host, disk, resize.HostInfo)
	if err != nil {
		log.Errorf("unable to ResizePartition: %v", err)
	}
	return desc, nil
}

func resizeProxmoxPartition(ctx context.Context, host *proxmox.SHost, disk cloudprovider.ICloudDisk, accessInfo vcenter.SVCenterAccessInfo) error {
	diskPath := disk.GetAccessPath()
	diskInfo := deployapi.DiskInfo{
		Path: diskPath,
	}
	vddkInfo := deployapi.VDDKConInfo{
		Host:   host.GetAccessIp(),
		Port:   22,
		User:   strings.TrimSuffix(accessInfo.Account, "@pam"),
		Passwd: accessInfo.Password,
	}
	_, err := deployclient.GetDeployClient().ResizeFs(ctx, &deployapi.ResizeFsParams{
		DiskInfo:   &diskInfo,
		Hypervisor: compute.HYPERVISOR_PROXMOX,
		VddkInfo:   &vddkInfo,
	})
	if err != nil {
		return errors.Wrap(err, "unable to ResizeFs")
	}
	return nil
}
