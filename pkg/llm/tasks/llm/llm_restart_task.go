package llm

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	apis "yunion.io/x/onecloud/pkg/apis/llm"
	notifyapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/tasks/worker"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type LLMRestartTask struct {
	taskman.STask
}

type LLMResetTask struct {
	LLMRestartTask
}

func init() {
	taskman.RegisterTask(LLMRestartTask{})
	taskman.RegisterTask(LLMResetTask{})
}

func (task *LLMRestartTask) taskFailed(ctx context.Context, llm *models.SLLM, err string) {
	task.taskFailedWithStatus(ctx, llm, "", err)
}

func (task *LLMRestartTask) taskFailedWithStatus(ctx context.Context, llm *models.SLLM, status string, err string) {
	if len(status) == 0 {
		input := api.LLMRestartTaskInput{}
		task.GetParams().Unmarshal(&input)

		status = api.LLM_STATUS_START_FAIL
	}
	llm.SetStatus(ctx, task.GetUserCred(), status, err)
	db.OpsLog.LogEvent(llm, "restart", err, task.GetUserCred())
	logclient.AddActionLogWithStartable(task, llm, logclient.ACT_VM_RESTART, err, task.GetUserCred(), false)
	llm.NotifyRequest(ctx, task.GetUserCred(), notifyapi.ActionRestart, nil, false)
	task.SetStageFailed(ctx, jsonutils.NewString(err))
}

func (task *LLMRestartTask) taskComplete(ctx context.Context, llm *models.SLLM) {
	llm.SetStatus(ctx, task.GetUserCred(), api.LLM_STATUS_RUNNING, "restart complete")
	llm.NotifyRequest(ctx, task.GetUserCred(), notifyapi.ActionRestart, nil, true)
	task.SetStageComplete(ctx, nil)
}

func (task *LLMRestartTask) stopComplete(ctx context.Context, d *models.SLLM) {
	d.SetStatus(ctx, task.GetUserCred(), apis.LLM_STATUS_READY, "stop complete")
	d.NotifyRequest(ctx, task.GetUserCred(), notifyapi.ActionRestart, nil, true)
	task.SetStageComplete(ctx, nil)
}

func (task *LLMRestartTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	input := api.LLMRestartTaskInput{}
	task.GetParams().Unmarshal(&input)

	llm.SetStatus(ctx, task.UserCred, api.LLM_STATUS_START_RESTART, "restarting")

	if task.HasParentTask() {
		task.OnSyncLLMInitStatusComplete(ctx, llm, nil)
		return
	}

	// Always do server syncstatus
	task.SetStage("OnSyncLLMInitStatusComplete", nil)
	if err := llm.StartSyncStatusTask(ctx, task.UserCred, task.GetTaskId()); err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "StartSyncStatusTask").Error())
	}
}

func (task *LLMRestartTask) OnSyncLLMInitStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}

func (task *LLMRestartTask) OnSyncLLMInitStatusComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)

	srv, err := llm.GetServer(ctx)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "GetServer").Error())
		return
	}

	switch srv.Status {
	case computeapi.VM_RUNNING:
		task.SetStage("OnServerStopComplete", nil)
		if err := llm.StartLLMStopTask(ctx, task.UserCred, task.GetTaskId()); err != nil {
			task.taskFailed(ctx, llm, errors.Wrap(err, "StartLLMStopTask").Error())
			return
		}
	case computeapi.VM_READY:
		task.OnServerStopComplete(ctx, llm, nil)
	default:
		if strings.Contains(srv.Status, "fail") {
			task.taskFailed(ctx, llm, errors.Wrap(errors.ErrInvalidStatus, srv.Status).Error())
			return
		}
		task.SetStage("OnSyncLLMInitStatusComplete", nil)
		worker.BackupTaskRun(task, func() (jsonutils.JSONObject, error) {
			_, err := llm.WaitServerStatus(ctx, task.UserCred, []string{computeapi.VM_READY, computeapi.VM_RUNNING}, 1800)
			if err != nil {
				return nil, errors.Wrap(err, "WaitServerStatus")
			}
			return nil, nil
		})
	}
}

func (task *LLMRestartTask) OnServerStopCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}

func (task *LLMRestartTask) OnServerStopComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	input := apis.LLMRestartTaskInput{}
	task.GetParams().Unmarshal(&input)

	server, err := llm.GetServer(ctx)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "GetServer").Error())
		return
	}

	skuId := input.SkuId
	if len(skuId) == 0 {
		skuId = llm.LLMSkuId
	}
	sku, err := llm.GetLLMSku(skuId)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "GetLLMSku").Error())
		return
	}

	if len(server.Nics) > 0 {
		// check bandwidth
		// use admin credential
		bandwidth := llm.BandwidthMb
		if bandwidth == 0 {
			bandwidth = sku.Bandwidth
		}
		err := llm.ChangeServerNetworkConfig(ctx, bandwidth, nil, true)
		if err != nil {
			task.taskFailed(ctx, llm, errors.Wrap(err, "changeServerNetworkConfig").Error())
			return
		}
	}

	if sku.Cpu != server.VcpuCount || sku.Memory+1 != server.VmemSize {
		// need to change config
		s := auth.GetSession(ctx, task.UserCred, "")
		params := computeapi.ServerChangeConfigInput{}
		params.VcpuCount = &sku.Cpu
		params.VmemSize = fmt.Sprintf("%dM", sku.Memory+1)
		_, err := compute.Servers.PerformAction(s, llm.CmpId, "change-config", jsonutils.Marshal(params))
		if err != nil {
			task.taskFailed(ctx, llm, errors.Wrap(err, "change config").Error())
			return
		}

		task.SetStage("OnChangeConfigComplete", nil)
		worker.BackupTaskRun(task, func() (jsonutils.JSONObject, error) {
			_, err := llm.WaitServerStatus(ctx, task.UserCred, []string{computeapi.VM_READY}, 1800)
			if err != nil {
				return nil, errors.Wrap(err, "WaitServerStatus")
			}
			return nil, nil
		})
		return
	}

	task.OnChangeConfigComplete(ctx, llm, nil)
}

func (task *LLMRestartTask) OnChangeConfigCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}

func (task *LLMRestartTask) OnChangeConfigComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	input := apis.LLMRestartTaskInput{}
	task.GetParams().Unmarshal(&input)

	if len(input.RebindVolumeId) > 0 {
		// rebind volume
		task.SetStage("OnRebindVolumeComplete", nil)
		_, err := llm.StartBindVolumeTask(ctx, task.UserCred, input.RebindVolumeId, false, task.GetTaskId())
		if err != nil {
			task.taskFailed(ctx, llm, errors.Wrap(err, "StartBindVolumeTask").Error())
			return
		}
	} else {
		task.OnRebindVolumeComplete(ctx, llm, nil)
	}
}

func (task *LLMRestartTask) OnRebindVolumeCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}

func (task *LLMRestartTask) OnRebindVolumeComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	input := apis.LLMRestartTaskInput{}
	task.GetParams().Unmarshal(&input)

	volume, err := llm.GetVolume()
	if err != nil && errors.Cause(err) != errors.ErrNotFound {
		task.taskFailed(ctx, llm, errors.Wrap(err, "GetVolume").Error())
		return
	}

	sku, err := llm.GetLLMSku(input.SkuId)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrapf(err, "GetLLMSku by input sku_id %q", input.SkuId).Error())
		return
	}

	if volume != nil && len(*sku.Volumes) > 0 && (*sku.Volumes)[0].SizeMB > volume.SizeMB {
		task.SetStage("OnDiskResizeComplete", nil)
		_, err := volume.StartResizeTask(ctx, task.UserCred, api.VolumeResizeTaskInput{
			SizeMB:        (*sku.Volumes)[0].SizeMB,
			DesktopStatus: api.STATUS_READY,
		}, task.GetTaskId())
		if err != nil {
			task.taskFailed(ctx, llm, errors.Wrap(err, "StartResizeTask").Error())
			return
		}
	} else {
		task.OnDiskResizeComplete(ctx, llm, nil)
	}
}

func (task *LLMRestartTask) OnDiskResizeCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}

func (task *LLMRestartTask) OnDiskResizeComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	input := apis.LLMRestartTaskInput{}
	task.GetParams().Unmarshal(&input)

	// TODO: 处理多个 volumes 的情况
	volume, err := llm.GetVolume()
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "GetVolume").Error())
		return
	}

	disk, err := volume.GetDisk(ctx)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "GetDisk").Error())
		return
	}

	sku, err := llm.GetLLMSku(input.SkuId)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrapf(err, "GetLLMSku by input sku_id %q", input.SkuId).Error())
		return
	}

	var templateId *string
	var backupId *string
	if volume.TemplateId != (*sku.Volumes)[0].TemplateId || volume.TemplateId != disk.TemplateId {
		// reset disk template
		input.ResetDataDisk = true
		templateId = &(*sku.Volumes)[0].TemplateId
	}
	var llmBackup *models.SLLMBackup
	sizeGB := (*sku.Volumes)[0].SizeMB / 1024
	if len(input.BackupName) > 0 {
		llmBackupObj, err := models.GetLLMBackupManager().FetchById(input.BackupName)
		if err != nil {
			task.taskFailed(ctx, llm, errors.Wrapf(err, "desktopBackupManager.FetchById %s", input.BackupName).Error())
			return
		}
		llmBackup = llmBackupObj.(*models.SLLMBackup)
		backupId = &llmBackup.DiskbackupId
		input.ResetDataDisk = true
		sizeGB = llmBackup.VolumeSizeMB / 1024
	}

	if len(input.RebindVolumeId) == 0 && input.ResetDataDisk {
		// 恢复出厂设置
		err := volume.DoReset(ctx, task.UserCred, templateId, backupId, sizeGB)
		if err != nil {
			task.taskFailed(ctx, llm, errors.Wrap(err, "volume.DoReset").Error())
			return
		}

		task.SetStage("OnResetDiskComplete", jsonutils.Marshal(input).(*jsonutils.JSONDict))

		worker.BackupTaskRun(task, func() (jsonutils.JSONObject, error) {
			_, err := volume.WaitDiskStatus(ctx, task.UserCred, []string{computeapi.DISK_READY}, 7200)
			if err != nil {
				return nil, errors.Wrap(err, "WaitDiskStatus")
			}

			_, err = llm.WaitServerStatus(ctx, task.UserCred, []string{computeapi.VM_READY}, 7200)
			if err != nil {
				return nil, errors.Wrap(err, "WaitServerStatus")
			}

			var mountedModels []string
			if llmBackup != nil {
				mountedModels = llmBackup.MountedModels
			}
			err = llm.UpdateMountedModelFullNames(ctx, task.GetUserCred(), mountedModels, true, input.ImageId, input.SkuId)
			if err != nil {
				return nil, errors.Wrap(err, "UpdateMountedAppVersions")
			}

			return nil, nil
		})
	} else {
		if len(input.RebindVolumeId) == 0 {
			// if not rebind volume and reset disk, should reset mounted apps according to phone model also preserve the current mounted apps
			err := llm.UpdateMountedModelFullNames(ctx, task.GetUserCred(), nil, false, input.ImageId, input.SkuId)
			if err != nil {
				task.taskFailed(ctx, llm, errors.Wrap(err, "UpdateMountedAppVersions").Error())
				return
			}
		}
		task.OnResetDiskComplete(ctx, llm, nil)
	}
}

func (task *LLMRestartTask) OnResetDiskCompleteFailed(ctx context.Context, obj db.IStandaloneModel, errJson jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, errJson.String())
}

func (task *LLMRestartTask) OnResetDiskComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	input := apis.LLMRestartTaskInput{}
	task.GetParams().Unmarshal(&input)

	imageId := llm.LLMImageId
	if len(input.ImageId) > 0 {
		imageId = input.ImageId
	}
	llmImageObj, err := models.GetLLMImageManager().FetchById(imageId)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "fetch image").Error())
		return
	}
	llmImage := llmImageObj.(*models.SLLMImage)

	skuId := input.SkuId
	if len(skuId) == 0 {
		skuId = llm.LLMSkuId
	}
	sku, err := llm.GetLLMSku(skuId)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrapf(err, "GetLLMSku by input sku_id %q", skuId).Error())
		return
	}

	volume, err := llm.GetVolume()
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "GetVolume").Error())
		return
	}

	disk, err := volume.GetDisk(ctx)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "volume.GetDisk").Error())
		return
	}

	/*diskCaseInsensitive := false
	if disk.FsFeatures != nil && disk.FsFeatures.Ext4 != nil && disk.FsFeatures.Ext4.CaseInsensitive {
		diskCaseInsensitive = true
	}*/

	srvDetails, err := llm.GetServer(ctx)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "GetServer").Error())
		return
	}

	// 处理多个比较 app 主容器的变化
	ctrs, err := models.GetLLMContainers(ctx, llm)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "GetLLMContainers").Error())
		return
	}
	drv := llm.GetLLMContainerDriver()
	containersToUpdateInput := models.NewContainersToUpdateInput(llm, &input, sku, llmImage, srvDetails, disk, ctrs)
	updates, err := models.GetContainersToUpdate(drv, ctx, containersToUpdateInput)
	if err != nil {
		task.taskFailed(ctx, llm, errors.Wrap(err, "drv.GetContainersToUpdate").Error())
		return
	}
	for ctrId, ctrInput := range updates {
		if err := UpdateContainerIfNeeded(ctx, task.UserCred, ctrId, ctrInput); err != nil {
			task.taskFailed(ctx, llm, errors.Wrapf(err, "update container %s %s", ctrId, ctrInput.Name).Error())
			return
		}
	}

	if (len(input.ImageId) > 0 && llm.LLMImageId != input.ImageId) || (len(input.SkuId) > 0 && llm.LLMSkuId != input.SkuId) {
		diff, err := db.Update(llm, func() error {
			if len(input.ImageId) > 0 {
				llm.LLMImageId = input.ImageId
				llm.SyncImageRequest = false
			}
			if len(input.SkuId) > 0 {
				llm.LLMSkuId = input.SkuId
			}
			return nil
		})
		if err != nil {
			task.taskFailed(ctx, llm, errors.Wrap(err, "update llm_image_id or llm_sku_id fail").Error())
			return
		}
		if diff != nil {
			db.OpsLog.LogEvent(llm, db.ACT_UPDATE, diff, task.UserCred)
			logclient.AddActionLogWithStartable(task, llm, logclient.ACT_UPDATE, diff.String(), task.UserCred, true)
		}
	}

	if input.OnlyStop {
		task.stopComplete(ctx, llm)
		return
	}

	task.SetStage("OnStartComplete", nil)
	llm.StartStartTaskInternal(ctx, task.UserCred, task.GetTaskId())
}

func (task *LLMRestartTask) OnStartComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskComplete(ctx, llm)
}

func (task *LLMRestartTask) OnStartCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	llm := obj.(*models.SLLM)
	task.taskFailed(ctx, llm, err.String())
}

func getGuestContainerNameToId(ctx context.Context, guestId string) (map[string]string, error) {
	s := auth.GetAdminSession(ctx, "")
	resp, err := compute.Containers.List(s, jsonutils.Marshal(map[string]string{
		"guest_id": guestId,
		"scope":    "max",
	}))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(resp.Data))
	for i := range resp.Data {
		id, _ := resp.Data[i].GetString("id")
		name, _ := resp.Data[i].GetString("name")
		if id == "" || name == "" {
			continue
		}
		out[name] = id
	}
	if len(out) == 0 {
		return nil, errors.Wrapf(errors.ErrNotFound, "no containers for guest %s", guestId)
	}
	return out, nil
}

func UpdateContainerIfNeeded(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ctrId string,
	ctrInput *computeapi.PodContainerCreateInput,
) error {
	_, err := models.UpdateContainerIfNeeded(ctx, userCred, ctrId, ctrInput)
	if err != nil {
		return errors.Wrap(err, "update container")
	}
	return nil
}
