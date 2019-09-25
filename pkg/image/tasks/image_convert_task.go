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

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
)

type ImageConvertTask struct {
	taskman.STask
}

func init() {
	convertWorker := appsrv.NewWorkerManager("ImageConvertTaskWorkerManager", 2, 512, true)
	taskman.RegisterTaskAndWorker(ImageConvertTask{}, convertWorker)
}

func (self *ImageConvertTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)

	self.SetStage("OnConvertComplete", nil)
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		imgOldStatus := image.Status
		image.SetStatus(self.UserCred, api.IMAGE_STATUS_CONVERTING, "start convert")
		err := image.ConvertAllSubformats()
		var msg string
		if err != nil {
			msg = fmt.Sprintf("convert failed: %s", err)
		} else {
			msg = fmt.Sprintf("convert success")
		}

		image.SetStatus(self.UserCred, api.IMAGE_STATUS_ACTIVE, msg)
		if imgOldStatus != api.IMAGE_STATUS_ACTIVE {
			kwargs := jsonutils.NewDict()
			kwargs.Set("name", jsonutils.NewString(image.GetName()))
			osType, err := models.ImagePropertyManager.GetProperty(image.Id, api.IMAGE_OS_TYPE)
			if err == nil {
				kwargs.Set("os_type", jsonutils.NewString(osType.Value))
			}
			notifyclient.SystemNotify(notify.NotifyPriorityNormal, notifyclient.IMAGE_ACTIVED, kwargs)
			notifyclient.NotifyImportant([]string{self.UserCred.GetUserId()}, false, notifyclient.IMAGE_ACTIVED, kwargs)
		}
		return nil, err
	})
}

func (self *ImageConvertTask) OnConvertComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *ImageConvertTask) OnConvertCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}
