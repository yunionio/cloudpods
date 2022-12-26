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

	"yunion.io/x/cloudmux/pkg/multicloud/esxi"
	"yunion.io/x/cloudmux/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type SAgentDisk struct {
	SLocalDisk
}

func NewAgentDisk(storage IStorage, id string) *SAgentDisk {
	return &SAgentDisk{*NewLocalDisk(storage, id)}
}

type PrepareSaveToGlanceParams struct {
	TaskId   string
	DiskInfo jsonutils.JSONObject
}

func (sd *SAgentDisk) PrepareSaveToGlance(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	p, ok := params.(PrepareSaveToGlanceParams)
	if !ok {
		return nil, errors.Wrap(hostutils.ParamsError, "Resize params format error")
	}
	storage := sd.Storage.(*SAgentStorage)
	return storage.PrepareSaveToGlance(ctx, p.TaskId, p.DiskInfo)
}

func (sd *SAgentDisk) Resize(ctx context.Context, diskInfo interface{}) (jsonutils.JSONObject, error) {
	body, ok := diskInfo.(*jsonutils.JSONDict)
	if !ok {
		return nil, errors.Wrap(hostutils.ParamsError, "PrepareSaveToGlance params format error")
	}

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

	esxiClient, err := esxi.NewESXiClientFromAccessInfo(ctx, &resize.HostInfo)
	if err != nil {
		return nil, httperrors.NewInputParameterError("info of host_info error")
	}
	host, err := esxiClient.FindHostByIp(resize.HostInfo.PrivateId)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to find host by ip %s", resize.HostInfo.PrivateId)
	}
	ivm, err := host.GetIVMById(resize.VMId)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to find vm by ID %s", resize.VMId)
	}
	vm := ivm.(*esxi.SVirtualMachine)
	iDisk, err := vm.GetIDiskById(resize.DiskId)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get idisk  %q of vm %s", resize.DiskId, resize.VMId)
	}
	disk := iDisk.(*esxi.SVirtualDisk)
	err = disk.Resize(ctx, resize.SizeMb)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to resize to %dMB", resize.SizeMb)
	}

	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewInt(resize.SizeMb), "disk_size")
	// try to resize partition
	err = resizeESXiPartition(ctx, vm, iDisk.(*esxi.SVirtualDisk), resize.HostInfo)
	if err != nil {
		log.Errorf("unable to ResizePartition: %v", err)
	}
	return desc, nil
}

func resizeESXiPartition(ctx context.Context, vm *esxi.SVirtualMachine, disk *esxi.SVirtualDisk, accessInfo vcenter.SVCenterAccessInfo) error {
	diskPath := disk.GetFilename()
	vmref := vm.GetMoid()
	diskInfo := deployapi.DiskInfo{
		Path: diskPath,
	}
	vddkInfo := deployapi.VDDKConInfo{
		Host:   accessInfo.Host,
		Port:   int32(accessInfo.Port),
		User:   accessInfo.Account,
		Passwd: accessInfo.Password,
		Vmref:  vmref,
	}
	_, err := deployclient.GetDeployClient().ResizeFs(ctx, &deployapi.ResizeFsParams{
		DiskInfo:   &diskInfo,
		Hypervisor: compute.HYPERVISOR_ESXI,
		VddkInfo:   &vddkInfo,
	})
	if err != nil {
		return errors.Wrap(err, "unable to ResizeFs")
	}
	return nil
}
