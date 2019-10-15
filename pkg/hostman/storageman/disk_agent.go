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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/hostman/hostutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
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
		return nil, hostutils.ParamsError
	}
	storage := sd.Storage.(*SAgentStorage)
	return storage.PrePareSaveToGlance(ctx, p.TaskId, p.DiskInfo)
}

func (sd *SAgentDisk) ReSize(ctx context.Context, diskInfo interface{}) (jsonutils.JSONObject, error) {
	body, ok := diskInfo.(*jsonutils.JSONDict)
	if !ok {
		return nil, hostutils.ParamsError
	}
	// check parameters
	params := []string{"size", "host_info", "vm_private_id", "disk_private_id"}
	for _, param := range params {
		if !body.Contains(param) {
			return nil, httperrors.NewMissingParameterError(param)
		}
	}

	sizeMb, _ := body.Int("size")
	hostInfo, _ := body.Get("host_info")
	vmId, _ := body.GetString("vm_private_id")
	diskId, _ := body.GetString("disk_private_id")

	esxiClient, accessInfo, err := esxi.NewESXiClientFromJson(ctx, hostInfo)
	if err != nil {
		return nil, httperrors.NewInputParameterError("info of host_info error")
	}
	host, err := esxiClient.FindHostByIp(accessInfo.PrivateId)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to find host by ip %s", accessInfo.PrivateId)
	}
	ivm, err := host.GetIVMById(vmId)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to find vm by ID %s", vmId)
	}
	idisks, err := ivm.GetIDisks()
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get idisks of vm %s", vmId)
	}
	var (
		idisk   cloudprovider.ICloudDisk
		hasDisk bool
	)
	for i := range idisks {
		if idisks[i].GetId() == diskId {
			idisk = idisks[i]
			hasDisk = true
		}
	}
	if !hasDisk {
		return nil, errors.Wrapf(err, "no such disk %s", diskId)
	}
	url := idisk.GetAccessPath()
	vm, disk := ivm.(*esxi.SVirtualMachine), idisk.(*esxi.SVirtualDisk)
	online := jsonutils.QueryBoolean(body, "online", false)
	if online {
		esxiClient.DoExtendDiskOnline(vm, disk, sizeMb)
	} else {
		esxiClient.ExtendDisk(url, sizeMb)
	}
	desc := jsonutils.NewDict()
	desc.Add(jsonutils.NewInt(sizeMb), "disk_size")
	return desc, nil
}
