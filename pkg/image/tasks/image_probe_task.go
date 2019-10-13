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
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ImageProbeTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ImageProbeTask{})
}

func (self *ImageProbeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)
	if err, ok := image.IsIso(); err != nil {
		self.OnProbeFailed(ctx, image, err.Error())
	} else if !ok {
		self.StartImageProbe(ctx, image)
	} else {
		// iso do not probe
		self.OnProbeSuccess(ctx, image)
	}
}

func (self *ImageProbeTask) StartImageProbe(ctx context.Context, image *models.SImage) {
	probeErr := self.doProbe(ctx, image)

	// save chksum first
	// if failed, set image unavailable, because size and chksum changed
	if err := self.updateImageMetadata(ctx, image); err != nil {
		image.SetStatus(self.UserCred, api.IMAGE_STATUS_KILLED,
			fmt.Sprintf("Image update failed after probe %s", err))
		self.SetStageFailed(ctx, err.Error())
		return
	}

	if probeErr == nil {
		self.OnProbeSuccess(ctx, image)
	} else {
		self.OnProbeFailed(ctx, image, probeErr.Error())
	}
}

func (self *ImageProbeTask) doProbe(ctx context.Context, image *models.SImage) error {
	diskPath := image.GetPath("")
	if deployclient.GetDeployClient() == nil {
		return fmt.Errorf("deploy client not init")
	}
	imageInfo, err := deployclient.GetDeployClient().ProbeImageInfo(ctx, &deployapi.ProbeImageInfoPramas{DiskPath: diskPath})
	if err != nil {
		return err
	}
	self.updateImageInfo(ctx, image, imageInfo)
	return nil
}

func (self *ImageProbeTask) updateImageMetadata(
	ctx context.Context, image *models.SImage,
) error {
	imagePath := image.GetPath("")
	fp, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer fp.Close()

	stat, err := fp.Stat()
	if err != nil {
		return err
	}

	chksum, err := fileutils2.MD5(imagePath)
	if err != nil {
		return err
	}

	fastchksum, err := fileutils2.FastCheckSum(imagePath)
	if err != nil {
		return err
	}

	_, err = db.Update(image, func() error {
		image.Size = stat.Size()
		image.Checksum = chksum
		image.FastHash = fastchksum
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (self *ImageProbeTask) updateImageInfo(ctx context.Context, image *models.SImage, imageInfo *deployapi.ImageInfo) {
	imageProperties := jsonutils.Marshal(imageInfo.OsInfo).(*jsonutils.JSONDict)
	if len(imageInfo.OsType) > 0 {
		imageProperties.Set(api.IMAGE_OS_TYPE, jsonutils.NewString(imageInfo.OsType))
	}
	if imageInfo.IsUefiSupport {
		imageProperties.Set(api.IMAGE_UEFI_SUPPORT, jsonutils.JSONTrue)
	}
	if imageInfo.IsLvmPartition {
		imageProperties.Set(api.IMAGE_IS_LVM_PARTITION, jsonutils.JSONTrue)
	}
	if imageInfo.IsReadonly {
		imageProperties.Set(api.IMAGE_IS_READONLY, jsonutils.JSONTrue)
	}
	if imageInfo.IsInstalledCloudInit {
		imageProperties.Set(api.IMAGE_INSTALLED_CLOUDINIT, jsonutils.JSONTrue)

	}
	if len(imageInfo.PhysicalPartitionType) > 0 {
		imageProperties.Set(api.IMAGE_PARTITION_TYPE, jsonutils.NewString(imageInfo.PhysicalPartitionType))
	}
	models.ImagePropertyManager.SaveProperties(ctx, self.UserCred, image.Id, imageProperties)
}

type sImageInfo struct {
	osInfo *deployapi.ReleaseInfo

	OsType                string
	IsUEFISupport         bool
	IsLVMPartition        bool
	IsReadonly            bool
	PhysicalPartitionType string // mbr or gbt or unknown
	IsInstalledCloudInit  bool
}

func (self *ImageProbeTask) OnProbeFailed(ctx context.Context, image *models.SImage, reason string) {
	log.Infof("Image %s Probe Failed: %s", image.Name, reason)
	db.OpsLog.LogEvent(image, db.ACT_PROBE_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, image, logclient.ACT_IMAGE_PROBE, reason, self.UserCred, false)

	if jsonutils.QueryBoolean(self.Params, "do_convert", false) {
		self.SetStage("OnConvertComplete", nil)
		image.StartImageConvertTask(ctx, self.UserCred, self.GetId())
	} else {
		self.SetStageFailed(ctx, reason)
	}
}

func (self *ImageProbeTask) OnProbeSuccess(ctx context.Context, image *models.SImage) {
	log.Infof("Image %s Probe Success ...", image.Name)
	db.OpsLog.LogEvent(image, db.ACT_PROBE, "Image Probe Success", self.UserCred)
	logclient.AddActionLogWithContext(
		ctx, image, logclient.ACT_IMAGE_PROBE, "Image Probe Success", self.UserCred, true)

	if jsonutils.QueryBoolean(self.Params, "do_convert", false) {
		self.SetStage("OnConvertComplete", nil)
		image.StartImageConvertTask(ctx, self.UserCred, self.GetId())
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *ImageProbeTask) OnConvertComplete(ctx context.Context, image *models.SImage, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
