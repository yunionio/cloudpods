package models

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
)

func init() {
	GetVolumeManager()
}

var volumeManager *SVolumeManager

func GetVolumeManager() *SVolumeManager {
	if volumeManager != nil {
		return volumeManager
	}
	volumeManager = &SVolumeManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SVolume{},
			"volumes_tbl",
			"llm_volume",
			"llm_volumes",
		),
	}
	volumeManager.SetVirtualObject(volumeManager)
	return volumeManager
}

type SVolumeManager struct {
	db.SVirtualResourceBaseManager
	SMountedModelsResourceManager
}

type SVolume struct {
	db.SVirtualResourceBase
	SMountedModelsResource

	LLMId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"user"`
	// 存储类型
	StorageType string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"user"`
	// 模板ID
	TemplateId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"admin_optional" update:"user"`
	// size in MB
	SizeMB     int                          `nullable:"false" default:"0" create:"optional" list:"user" update:"user"`
	CmpId      string                       `width:"128" charset:"ascii" nullable:"true" list:"user"`
	Containers api.ContainerVolumeRelations `charset:"utf8" nullable:"true" list:"user" create:"optional"`
}

func (volume *SVolume) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "VolumeDeleteTask", volume, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	volume.SetStatus(ctx, userCred, commonapi.STATUS_DELETING, "")
	task.ScheduleRun(nil)
	return nil
}

func (volume *SVolume) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return volume.SVirtualResourceBase.Delete(ctx, userCred)
}

func (volume *SVolume) StartResizeTask(ctx context.Context, userCred mcclient.TokenCredential, input api.VolumeResizeTaskInput, parentTaskId string) (*taskman.STask, error) {
	volume.SetStatus(ctx, userCred, computeapi.DISK_START_RESIZE, "StartResizeTask")
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "VolumeResizeTask", volume, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "NewTask")
	}
	if err := task.ScheduleRun(nil); err != nil {
		return nil, errors.Wrap(err, "ScheduleRun")
	}
	return task, nil
}

func fetchImage(ctx context.Context, userCred mcclient.TokenCredential, imageId string) (*imageapi.ImageDetails, error) {
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	imgObj, err := image.Images.Get(s, imageId, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Image.Get %s", imageId)
	}
	img := imageapi.ImageDetails{}
	err = imgObj.Unmarshal(&img)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &img, nil
}

func (volume *SVolume) UpdateMountedModelFullNames(mountModels []string) error {
	_, err := db.Update(volume, func() error {
		volume.MountedModels = mountModels
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "update volume mounted_apps")
	}
	return nil
}

func (volume *SVolume) GetDisk(ctx context.Context) (*computeapi.DiskDetails, error) {
	if len(volume.CmpId) == 0 {
		return nil, errors.ErrInvalidStatus
	}
	s := auth.GetAdminSession(ctx, "")
	disk := computeapi.DiskDetails{}
	resp, err := compute.Disks.GetById(s, volume.CmpId, jsonutils.Marshal(map[string]interface{}{
		"scope": "max",
	}))
	if err != nil {
		if httputils.ErrorCode(err) == 404 {
			return nil, errors.Wrapf(errors.ErrNotFound, "GetById %s", volume.CmpId)
		}
		return nil, errors.Wrap(err, "fetch disk")
	}
	resp.Unmarshal(&disk)
	return &disk, nil
}

func (volume *SVolume) WaitDiskStatus(ctx context.Context, userCred mcclient.TokenCredential, targetStatus []string, timeoutSecs int) (*computeapi.DiskDetails, error) {
	expire := time.Now().Add(time.Second * time.Duration(timeoutSecs))
	for time.Now().Before(expire) {
		disk, err := volume.GetDisk(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "GetDisk")
		}
		if utils.IsInArray(disk.Status, targetStatus) {
			return disk, nil
		}
		if strings.Contains(disk.Status, "fail") {
			return nil, errors.Wrap(errors.ErrInvalidStatus, disk.Status)
		}
		time.Sleep(2 * time.Second)
	}
	return nil, errors.Wrapf(httperrors.ErrTimeout, "wait disk status %s timeout", targetStatus)
}

func (volume *SVolume) GetLLM() *SLLM {
	if len(volume.LLMId) == 0 {
		return nil
	}
	obj, err := GetLLMManager().FetchById(volume.LLMId)
	if err != nil {
		log.Errorf("Volume %s fetch llm %s error %s", volume.Id, volume.LLMId, err)
		return nil
	}
	return obj.(*SLLM)
}

func (volume *SVolume) PerformReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.VolumePerformResetInput) (*api.LLMBatchPerformOutput, error) {
	if volume.Status != computeapi.DISK_READY {
		return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "invalid status %s", volume.Status)
	}
	task, err := volume.StartResetTask(ctx, userCred, input, "")
	if err != nil {
		return nil, errors.Wrap(err, "StartResetTask")
	}
	return &api.LLMBatchPerformOutput{
		Data: []api.LLMPerformOutput{
			{
				Id:            volume.Id,
				Name:          volume.Name,
				RequestStatus: http.StatusOK,
				TaskId:        task.Id,
			},
		},
		Task: task,
	}, nil
}

func (volume *SVolume) StartResetTask(ctx context.Context, userCred mcclient.TokenCredential, input api.VolumePerformResetInput, parentTaskId string) (*taskman.STask, error) {
	volume.SetStatus(ctx, userCred, api.VOLUME_STATUS_START_RESET, "StartResetTask")
	params := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	task, err := taskman.TaskManager.NewTask(ctx, "VolumeResetTask", volume, userCred, params, parentTaskId, "")
	if err != nil {
		return nil, errors.Wrap(err, "NewTask")
	}
	err = task.ScheduleRun(nil)
	if err != nil {
		return nil, errors.Wrap(err, "ScheduleRun")
	}
	return task, nil
}

func (volume *SVolume) DoReset(ctx context.Context, userCred mcclient.TokenCredential, templateId, backupId *string, sizeGb int) error {
	// clear desktop app list
	{
		llm := volume.GetLLM()
		if llm != nil {
			llm.purgeModelList()
		}
	}
	// clear mounts
	{
		_, err := db.Update(volume, func() error {
			volume.MountedModels = nil
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "update volume mounted_apps")
		}
	}

	// reset disk
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	params := computeapi.DiskRebuildInput{}
	params.TemplateId = templateId
	params.BackupId = backupId
	emptyStr := ""
	if params.BackupId == nil {
		params.BackupId = &emptyStr
	}
	if sizeGb > 0 {
		size := fmt.Sprintf("%dG", sizeGb)
		params.Size = &size
	}

	log.Debugf("need to reset phone data disk ... %s %dG", jsonutils.Marshal(params), sizeGb)

	_, err := compute.Disks.PerformAction(s, volume.CmpId, "rebuild", jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrap(err, "rebuild disk")
	}

	return nil
}
