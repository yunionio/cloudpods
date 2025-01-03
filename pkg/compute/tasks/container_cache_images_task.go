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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
)

func init() {
	taskman.RegisterTask(ContainerCacheImagesTask{})
}

type ContainerCacheImagesTask struct {
	ContainerBaseTask
}

func (t *ContainerCacheImagesTask) getInput() (*api.ContainerCacheImagesInput, error) {
	input := new(api.ContainerCacheImagesInput)
	if err := t.GetParams().Unmarshal(input); err != nil {
		return nil, err
	}
	return input, nil
}

func (t *ContainerCacheImagesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	if err := t.startCacheImages(ctx, obj); err != nil {
		t.onError(ctx, obj.(*models.SContainer), jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerCacheImagesTask) onError(ctx context.Context, ctr *models.SContainer, reason jsonutils.JSONObject) {
	ctr.SetStatus(ctx, t.GetUserCred(), api.CONTAINER_STATUS_CACHE_IMAGE_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *ContainerCacheImagesTask) startCacheImages(ctx context.Context, obj db.IStandaloneModel) error {
	input, err := t.getInput()
	if err != nil {
		return errors.Wrapf(err, "getInput")
	}
	//caches := make([]db.IStandaloneModel, 0)
	//params := []*api.CacheImageInput{}
	t.SetStage("OnStorageCacheImageComplete", nil)
	for i, img := range input.Images {
		disk := models.DiskManager.FetchDiskById(img.DiskId)
		if disk == nil {
			return errors.Wrapf(httperrors.ErrNotFound, "disk not found %s", img.DiskId)
		}
		storage, _ := disk.GetStorage()
		storagecache := storage.GetStoragecache()
		if storagecache == nil {
			return errors.Wrapf(httperrors.ErrNotFound, "storage cache not found by %s", storage.GetId())
		}
		//caches = append(caches, storagecache)
		param := input.Images[i].Image
		param.ParentTaskId = t.GetTaskId()
		//params = append(params, param)
		if err := storagecache.StartImageCacheTask(ctx, t.GetUserCred(), *param); err != nil {
			return errors.Wrapf(err, "startImageCacheTask of param: %s", jsonutils.Marshal(param))
		}
	}
	return nil
}

func (t *ContainerCacheImagesTask) OnStorageCacheImageComplete(ctx context.Context, ctr *models.SContainer, data jsonutils.JSONObject) {
	if t.IsSubtask() {
		t.SetStageComplete(ctx, nil)
		return
	}
	ctr.StartSyncStatusTask(ctx, t.GetUserCred(), t.GetTaskId())
	t.SetStageComplete(ctx, nil)
}

func (t *ContainerCacheImagesTask) OnStorageCacheImageCompleteFailed(ctx context.Context, ctr *models.SContainer, data jsonutils.JSONObject) {
	t.onError(ctx, ctr, data)
}
