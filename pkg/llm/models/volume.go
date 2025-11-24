package models

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
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
	SvrId      string                       `width:"128" charset:"ascii" nullable:"true" list:"user"`
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
	if len(volume.SvrId) == 0 {
		return nil, errors.ErrInvalidStatus
	}
	s := auth.GetAdminSession(ctx, "")
	disk := computeapi.DiskDetails{}
	resp, err := compute.Disks.GetById(s, volume.SvrId, jsonutils.Marshal(map[string]interface{}{
		"scope": "max",
	}))
	if err != nil {
		if httputils.ErrorCode(err) == 404 {
			return nil, errors.Wrapf(errors.ErrNotFound, "GetById %s", volume.SvrId)
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
