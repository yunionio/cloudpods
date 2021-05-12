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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type ImageConvertTask struct {
	taskman.STask
}

func init() {
	convertWorker := appsrv.NewWorkerManager("ImageConvertTaskWorkerManager", 2, 512, true)
	putWorker := appsrv.NewWorkerManager("PutImageTaskWorkerManager", 4, 512, true)
	taskman.RegisterTaskAndWorker(ImageConvertTask{}, convertWorker)
	taskman.RegisterTaskAndWorker(PutImageTask{}, putWorker)
}

func (self *ImageConvertTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)
	self.Params.Set("old_status", jsonutils.NewString(image.Status))
	self.SetStage("OnConvertComplete", nil)
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		image.SetStatus(self.UserCred, api.IMAGE_STATUS_CONVERTING, "start convert")
		return nil, image.ConvertAllSubformats()
	})
}

func (self *ImageConvertTask) OnConvertComplete(ctx context.Context, image *models.SImage, data jsonutils.JSONObject) {
	image.StartPutImageTask(ctx, self.UserCred, "")
	self.SetStageComplete(ctx, nil)
}

func (self *ImageConvertTask) OnConvertCompleteFailed(ctx context.Context, image *models.SImage, data jsonutils.JSONObject) {
	image.StartPutImageTask(ctx, self.UserCred, "")
	self.SetStageFailed(ctx, data)
}

type PutImageTask struct {
	taskman.STask
}

func (self *PutImageTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)
	oldStatus := image.Status
	if strings.HasPrefix(image.Location, models.LocalFilePrefix) {
		imagePath := image.GetLocalLocation()
		image.SetStatus(self.UserCred, api.IMAGE_STATUS_SAVING, "save image to specific storage")
		storage := models.GetStorage()
		location, err := storage.SaveImage(imagePath)
		if err != nil {
			log.Errorf("Failed save image to specific storage %s", err)
			errStr := fmt.Sprintf("save image to storage %s: %v", storage.Type(), err)
			image.SetStatus(self.UserCred, api.IMAGE_STATUS_SAVE_FAIL, errStr)
			self.SetStageFailed(ctx, jsonutils.NewString(errStr))
			return
		} else if location != image.Location {
			_, err = db.Update(image, func() error {
				image.Location = location
				return nil
			})
			if err != nil {
				log.Errorf("failed update image location %s", err)
			} else {
				if err = procutils.NewCommand("rm", "-f", imagePath).Run(); err != nil {
					log.Errorf("failed remove file %s: %s", imagePath, err)
				}
			}
		}
	}
	image.SetStatus(self.UserCred, api.IMAGE_STATUS_ACTIVE, "save image to specific storage complete")
	if oldStatus != api.IMAGE_STATUS_ACTIVE {
		kwargs := jsonutils.NewDict()
		kwargs.Set("name", jsonutils.NewString(image.GetName()))
		osType, err := models.ImagePropertyManager.GetProperty(image.Id, api.IMAGE_OS_TYPE)
		if err == nil {
			kwargs.Set("os_type", jsonutils.NewString(osType.Value))
		}
		notifyclient.SystemNotifyWithCtx(ctx, notify.NotifyPriorityNormal, notifyclient.IMAGE_ACTIVED, kwargs)
		notifyclient.NotifyImportantWithCtx(ctx, []string{self.UserCred.GetUserId()}, false, notifyclient.IMAGE_ACTIVED, kwargs)
	}

	subimgs := models.ImageSubformatManager.GetAllSubImages(image.Id)
	for i := 0; i < len(subimgs); i++ {
		if !strings.HasPrefix(subimgs[i].Location, models.LocalFilePrefix) {
			continue
		}
		if subimgs[i].Format == image.DiskFormat {
			_, err := db.Update(&subimgs[i], func() error {
				subimgs[i].Location = image.Location
				subimgs[i].Status = api.IMAGE_STATUS_ACTIVE
				return nil
			})
			if err != nil {
				log.Errorf("failed update subimg %s", err)
			}
		} else {
			imagePath := subimgs[i].GetLocalLocation()
			storage := models.GetStorage()
			location, err := models.GetStorage().SaveImage(imagePath)
			if err != nil {
				log.Errorf("Failed save image to sepcific storage %s", err)
				errStr := fmt.Sprintf("save sub image %s to storage %s: %v", subimgs[i].Format, storage.Type(), err)
				subimgs[i].SetStatus(api.IMAGE_STATUS_SAVE_FAIL)
				self.SetStageFailed(ctx, jsonutils.NewString(errStr))
				return
			} else if subimgs[i].Location != location {
				_, err := db.Update(&subimgs[i], func() error {
					subimgs[i].Location = location
					return nil
				})
				if err != nil {
					log.Errorf("failed update subimg %s", err)
				}
				if err = procutils.NewCommand("rm", "-f", imagePath).Run(); err != nil {
					log.Errorf("failed remove file %s: %s", imagePath, err)
				}
			}
			db.Update(&subimgs[i], func() error {
				subimgs[i].Status = api.IMAGE_STATUS_ACTIVE
				return nil
			})
		}
	}

	self.SetStageComplete(ctx, nil)
}
