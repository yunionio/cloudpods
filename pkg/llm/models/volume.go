package models

import (
	"context"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
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
	// SMountedAppsResourceManager
}

type SVolume struct {
	db.SVirtualResourceBase
	// SMountedAppsResource

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
