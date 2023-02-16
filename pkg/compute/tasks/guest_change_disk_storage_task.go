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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(GuestChangeDiskStorageTask{})
}

type GuestChangeDiskStorageTask struct {
	SGuestBaseTask
}

func (t *GuestChangeDiskStorageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	t.ChangeDiskStorage(ctx, guest)
}

func (t *GuestChangeDiskStorageTask) GetInputParams() (*api.ServerChangeDiskStorageInternalInput, error) {
	input := new(api.ServerChangeDiskStorageInternalInput)
	err := t.GetParams().Unmarshal(input)
	return input, err
}

func (t *GuestChangeDiskStorageTask) getDiskById(id string) (*models.SDisk, error) {
	obj, err := models.DiskManager.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*models.SDisk), nil
}

func (t *GuestChangeDiskStorageTask) GetSourceDisk() (*models.SDisk, error) {
	input, err := t.GetInputParams()
	if err != nil {
		return nil, errors.Wrap(err, "GetInputParams")
	}
	return t.getDiskById(input.DiskId)
}

func (t *GuestChangeDiskStorageTask) GetTargetDisk() (*models.SDisk, error) {
	input, err := t.GetInputParams()
	if err != nil {
		return nil, errors.Wrap(err, "GetInputParams")
	}
	return t.getDiskById(input.TargetDiskId)
}

func (t *GuestChangeDiskStorageTask) ChangeDiskStorage(ctx context.Context, guest *models.SGuest) {
	input, err := t.GetInputParams()
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("GetInputParams error: %v", err)))
		return
	}

	targetDisk, err := t.GetTargetDisk()
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("GetTargetDisk error: %v", err)))
		return
	}
	sourceDisk, err := t.GetSourceDisk()
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("GetSourceDisk error: %v", err)))
		return
	}

	// set target disk's status to clone
	targetDisk.SetStatus(t.GetUserCred(), api.DISK_CLONE, "")

	err = sourceDisk.SetMetadata(ctx, api.DISK_CLONE_TASK_ID, t.GetId(), t.GetUserCred())
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("SetMetadata clone task id: %v", err)))
		return
	}

	if input.GuestRunning {
		t.SetStage("OnDiskLiveChangeStorageReady", nil)
	} else {
		t.SetStage("OnDiskChangeStorageComplete", nil)
	}

	// create target disk
	if err := guest.GetDriver().RequestChangeDiskStorage(ctx, t.GetUserCred(), guest, input, t); err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("RequestChangeDiskStorage: %s", err)))
		return
	}
}

func (t *GuestChangeDiskStorageTask) OnDiskLiveChangeStorageReady(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	if !jsonutils.QueryBoolean(data, "block_jobs_ready", false) {
		log.Infof("OnDiskLiveChangeStorageReady block jobs not ready")
		resp := new(hostapi.ServerCloneDiskFromStorageResponse)
		if err := data.Unmarshal(resp); err != nil {
			t.TaskFailed(ctx, guest,
				jsonutils.NewString(fmt.Sprintf("unmarshal OnDiskLiveChangeStorageReady resp failed %s", err)),
			)
			return
		}
		targetDisk, err := t.GetTargetDisk()
		if err != nil {
			t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("failed get target disk %s", err)))
			return
		}
		if _, err := db.UpdateWithLock(ctx, targetDisk, func() error {
			targetDisk.AccessPath = resp.TargetAccessPath
			targetDisk.DiskFormat = resp.TargetFormat
			return nil
		}); err != nil {
			t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Update target disk attributes error: %v", err)))
			return
		}
		return
	}

	input, err := t.GetInputParams()
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("GetInputParams error: %v", err)))
		return
	}
	guestdisk := guest.GetGuestDisk(input.DiskId)
	if guestdisk == nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString("failed get guest disk"))
		return
	}
	host, err := guest.GetHost()
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("failed get host %s", err)))
		return
	}
	targetDisk, err := t.GetTargetDisk()
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("failed get target disk %s", err)))
		return
	}
	input.TargetDiskDesc = guestdisk.GetDiskJsonDescAtHost(ctx, host, targetDisk)

	t.SetStage("OnDiskChangeStorageComplete", nil)
	// block job ready, start switch to target storage disk
	err = guest.GetDriver().RequestSwitchToTargetStorageDisk(ctx, t.UserCred, guest, input, t)
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("OnDiskLiveChangeStorageReady: %s", err)))
		return
	}
}

func (t *GuestChangeDiskStorageTask) OnDiskLiveChangeStorageReadyFailed(
	ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject,
) {
	targetDisk, _ := t.GetTargetDisk()
	targetDisk.SetStatus(t.GetUserCred(), api.DISK_CLONE_FAIL, data.String())
	t.TaskFailed(ctx, guest, data)
}

func (t *GuestChangeDiskStorageTask) OnDiskChangeStorageComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	srcDisk, err := t.GetSourceDisk()
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("GetSourceDisk: %v", err)))
		return
	}

	err = srcDisk.SetMetadata(ctx, api.DISK_CLONE_TASK_ID, "", t.GetUserCred())
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("SetMetadata clone task id: %v", err)))
		return
	}

	input, err := t.GetInputParams()
	if err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("GetInputParams error: %v", err)))
		return
	}

	if !input.GuestRunning {
		resp := new(hostapi.ServerCloneDiskFromStorageResponse)
		err = data.Unmarshal(resp)
		if err != nil {
			t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Unmarshal response: %v", err)))
			return
		}
		if len(resp.TargetFormat) == 0 {
			resp.TargetFormat = srcDisk.DiskFormat
		}

		targetDisk, err := t.GetTargetDisk()
		if err != nil {
			t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("GetTargetDisk error: %v", err)))
			return
		}
		if _, err := db.UpdateWithLock(ctx, targetDisk, func() error {
			targetDisk.AccessPath = resp.TargetAccessPath
			targetDisk.DiskFormat = resp.TargetFormat
			return nil
		}); err != nil {
			t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Update target disk attributes error: %v", err)))
			return
		}
	}

	guestSrcDisk := guest.GetGuestDisk(srcDisk.GetId())
	if guestSrcDisk == nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Source disk %s not attached", srcDisk.GetId())))
		return
	}

	conf := guestSrcDisk.ToDiskConfig()
	t.Params.Set("src_disk_conf", jsonutils.Marshal(conf))
	t.SetStage("OnSourceDiskDetachComplete", nil)
	if err := t.detachSourceDisk(ctx, guest, srcDisk); err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("detachSourceDisk: %s", err)))
		return
	}
}

func (t *GuestChangeDiskStorageTask) OnDiskChangeStorageCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	// set target disk's status to clone
	targetDisk, _ := t.GetTargetDisk()
	targetDisk.SetStatus(t.GetUserCred(), api.DISK_CLONE_FAIL, err.String())
	t.TaskFailed(ctx, guest, err)
}

func (t *GuestChangeDiskStorageTask) detachSourceDisk(ctx context.Context, guest *models.SGuest, srcDisk *models.SDisk) error {
	input, err := t.GetInputParams()
	if err != nil {
		return errors.Wrap(err, "GetInputParams")
	}
	return guest.StartGuestDetachdiskTask(ctx, t.GetUserCred(),
		srcDisk, input.KeepOriginDisk, t.GetTaskId(), false, true)
}

func (t *GuestChangeDiskStorageTask) OnSourceDiskDetachComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	t.SetStage("OnTargetDiskAttachComplete", data.(*jsonutils.JSONDict))
	conf := new(api.DiskConfig)
	if err := t.Params.Unmarshal(conf, "src_disk_conf"); err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("unmarshal %s to api.DiskConfig: %s", data, err)))
		return
	}
	if err := t.attachTargetDisk(ctx, guest, conf); err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("attachTargetDisk: %s", err)))
		return
	}
}

func (t *GuestChangeDiskStorageTask) OnSourceDiskDetachCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	t.TaskFailed(ctx, guest, err)
}

func (t *GuestChangeDiskStorageTask) attachTargetDisk(ctx context.Context, guest *models.SGuest, conf *api.DiskConfig) error {
	targetDisk, err := t.GetTargetDisk()
	if err != nil {
		return errors.Wrap(err, "GetTargetDisk")
	}
	confData := map[string]interface{}{
		"index":          conf.Index,
		"mountpoint":     conf.Mountpoint,
		"driver":         conf.Driver,
		"cache":          conf.Cache,
		"sync_desc_only": true,
	}
	attachData := jsonutils.Marshal(confData).(*jsonutils.JSONDict)
	attachData.Add(jsonutils.NewString(targetDisk.GetId()), "disk_id")

	return guest.GetDriver().StartGuestAttachDiskTask(ctx, t.GetUserCred(), guest, attachData, t.GetTaskId())
}

func (t *GuestChangeDiskStorageTask) OnTargetDiskAttachComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	t.SetStage("OnGuestSyncStatus", nil)
	if err := guest.StartSyncstatus(ctx, t.UserCred, t.Id); err != nil {
		t.TaskFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (t *GuestChangeDiskStorageTask) OnGuestSyncStatus(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	t.TaskComplete(ctx, guest, nil)
}

func (t *GuestChangeDiskStorageTask) OnGuestSyncStatusFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	t.TaskFailed(ctx, guest, data)
}

func (t *GuestChangeDiskStorageTask) OnTargetDiskAttachCompleteFailed(ctx context.Context, guest *models.SGuest, err jsonutils.JSONObject) {
	t.TaskFailed(ctx, guest, err)
}

func (t *GuestChangeDiskStorageTask) TaskComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(t, guest, logclient.ACT_DISK_CHANGE_STORAGE, nil, t.GetUserCred(), true)
	t.SetStageComplete(ctx, nil)
}

func (t *GuestChangeDiskStorageTask) TaskFailed(ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject) {
	guest.SetStatus(t.GetUserCred(), api.VM_DISK_CHANGE_STORAGE_FAIL, reason.String())
	logclient.AddActionLogWithStartable(t, guest, logclient.ACT_DISK_CHANGE_STORAGE, reason, t.GetUserCred(), false)
	t.SetStageFailed(ctx, reason)
}
