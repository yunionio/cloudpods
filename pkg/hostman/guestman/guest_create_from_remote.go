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

package guestman

import (
	"context"
	"fmt"

	"yunion.io/x/cloudmux/pkg/multicloud/esxi"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
)

func (m *SGuestManager) GuestCreateFromEsxi(
	ctx context.Context, params interface{},
) (jsonutils.JSONObject, error) {
	createConfig, ok := params.(*SGuestCreateFromEsxi)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetKVMServer(createConfig.Sid)
	if err := SaveDesc(guest, createConfig.GuestDesc); err != nil {
		return nil, err
	}

	esxiCli, err := esxi.NewESXiClientFromAccessInfo(ctx, &createConfig.EsxiAccessInfo.Datastore)
	if err != nil {
		return nil, errors.Wrap(err, "new esxi client")
	}
	host, err := esxiCli.FindHostByIp(createConfig.EsxiAccessInfo.HostIp)
	if err != nil {
		return nil, errors.Wrap(err, "esxi client find host by ip")
	}
	ivm, err := host.GetIVMById(createConfig.EsxiAccessInfo.GuestExtId)
	if err != nil {
		return nil, errors.Wrap(err, "get ivm by id")
	}
	vm := ivm.(*esxi.SVirtualMachine)
	disks, err := vm.GetIDisks()
	if err != nil {
		return nil, errors.Wrap(err, "vm get idisk")
	}
	if len(disks) == 0 {
		return nil, fmt.Errorf("no such disks for vm %s", vm.GetId())
	}
	vmref := vm.GetMoid()
	var esxiDisks = new(deployapi.ConnectEsxiDisksParams)
	esxiDisks.VddkInfo = &deployapi.VDDKConInfo{
		Host:   createConfig.EsxiAccessInfo.Datastore.Host,
		Port:   int32(createConfig.EsxiAccessInfo.Datastore.Port),
		User:   createConfig.EsxiAccessInfo.Datastore.Account,
		Passwd: createConfig.EsxiAccessInfo.Datastore.Password,
		Vmref:  vmref,
	}
	esxiDisks.AccessInfo = make([]*deployapi.EsxiDiskInfo, len(disks))
	for i := 0; i < len(disks); i++ {
		esxiDisks.AccessInfo[i] = &deployapi.EsxiDiskInfo{
			DiskPath: disks[i].(*esxi.SVirtualDisk).GetFilename(),
		}
	}
	connections, err := deployclient.GetDeployClient().ConnectEsxiDisks(ctx, esxiDisks)
	if err != nil {
		return nil, errors.Wrap(err, "connect esxi disks")
	}
	log.Infof("Connection disks %v", connections.String())

	var ret = jsonutils.NewDict()
	disksDesc := guest.SourceDesc.Disks
	for i := 0; i < len(disksDesc); i++ {
		storageId := disksDesc[i].StorageId
		if storage := storageman.GetManager().GetStorage(storageId); storage == nil {
			err = errors.Wrapf(err, "get storage %s", storageId)
			break
		} else {
			var diskInfo jsonutils.JSONObject
			diskId := disksDesc[i].DiskId
			iDisk := storage.CreateDisk(diskId)
			diskInfo, err = iDisk.CreateRaw(ctx, 0, "qcow2", "", nil, "", connections.Disks[i].DiskPath)
			if err != nil {
				err = errors.Wrapf(err, "create disk %s failed", diskId)
				log.Errorf(err.Error())
				break
			}
			diskInfo.(*jsonutils.JSONDict).Set("esxi_flat_filepath",
				jsonutils.NewString(connections.Disks[i].DiskPath))
			ret.Set(diskId, diskInfo)
		}
	}
	if err != nil {
		_, e := deployclient.GetDeployClient().DisconnectEsxiDisks(ctx, connections)
		if e != nil {
			log.Errorf("disconnect esxi disks failed %s", e)
		}
		return nil, err
	}
	return ret, nil
}

func (m *SGuestManager) GuestCreateFromCloudpods(
	ctx context.Context, params interface{},
) (jsonutils.JSONObject, error) {
	createConfig, ok := params.(*SGuestCreateFromCloudpods)
	if !ok {
		return nil, hostutils.ParamsError
	}
	guest, _ := m.GetKVMServer(createConfig.Sid)
	if err := SaveDesc(guest, createConfig.GuestDesc); err != nil {
		return nil, err
	}
	var err error
	var ret = jsonutils.NewDict()
	disksDesc := guest.SourceDesc.Disks
	for i := 0; i < len(disksDesc); i++ {
		storageId := disksDesc[i].StorageId
		storage := storageman.GetManager().GetStorage(storageId)
		if storage == nil {
			err = errors.Wrapf(err, "get storage %s", storageId)
			break
		}
		//var diskInfo jsonutils.JSONObject
		diskId := disksDesc[i].DiskId
		iDisk := storage.CreateDisk(diskId)
		diskUrl := fmt.Sprintf("http://%s:48885/disks/%s",
			createConfig.CloudpodsAccessInfo.HostIp, createConfig.CloudpodsAccessInfo.OriginDisksId[i])
		if err = iDisk.CreateFromImageFuse(ctx, diskUrl, 0, nil); err != nil {
			log.Errorf("failed create disk %s from fuse %s", diskUrl, err)
			break
		}
		diskInfo := jsonutils.NewDict()
		diskInfo.Set("origin_disk_url", jsonutils.NewString(diskUrl))
		diskInfo.Set("access_path", jsonutils.NewString(iDisk.GetPath()))
		ret.Set(diskId, diskInfo)
	}
	if err != nil {
		return nil, err
	}
	return ret, nil
}
