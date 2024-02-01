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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ImageSyncClassMetadataTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ImageSyncClassMetadataTask{})
}

func (self *ImageSyncClassMetadataTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	img := obj.(*models.SImage)
	cm, err := img.GetAllClassMetadata()
	if err != nil {
		self.taskFailed(ctx, img, jsonutils.NewString(err.Error()))
		return
	}
	session := auth.GetAdminSession(ctx, "")
	_, err = compute.Cachedimages.PerformAction(session, img.Id, "set-class-metadata", jsonutils.Marshal(cm))
	if err == nil {
		self.taskSuccess(ctx, img)
		return
	}
	if errors.Cause(err) != httperrors.ErrResourceNotFound {
		self.taskFailed(ctx, img, jsonutils.NewString(err.Error()))
		return
	}
	params := jsonutils.NewDict()
	params.Set("image_id", jsonutils.NewString(img.Id))
	_, err = compute.Cachedimages.PerformClassAction(session, "cache-image", params)
	if err != nil {
		self.taskFailed(ctx, img, jsonutils.NewString(err.Error()))
		return
	}
	self.taskSuccess(ctx, img)
}

func (self *ImageSyncClassMetadataTask) taskFailed(ctx context.Context, image *models.SImage, reason jsonutils.JSONObject) {
	reasonStr, _ := reason.GetString()
	image.SetStatus(ctx, self.UserCred, api.IMAGE_STATUS_SYNC_CLASS_METADATA_FAILEd, reasonStr)
	logclient.AddActionLogWithStartable(self, image, logclient.ACT_SYNC_CLASS_METADATA, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *ImageSyncClassMetadataTask) taskSuccess(ctx context.Context, image *models.SImage) {
	logclient.AddActionLogWithStartable(self, image, logclient.ACT_SYNC_CLASS_METADATA, nil, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}
