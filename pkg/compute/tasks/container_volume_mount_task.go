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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(ContainerAddVolumeMountPostOverlayTask{})
	taskman.RegisterTask(ContainerRemoveVolumeMountPostOverlayTask{})
}

type ContainerVolumeMountTaskPostOverlay struct {
	ContainerBaseTask
}

func (t *ContainerVolumeMountTaskPostOverlay) UpdateContainerVolume(c *models.SContainer, index int, vm *apis.ContainerVolumeMount) error {
	if _, err := db.Update(c, func() error {
		c.Spec.VolumeMounts[index] = vm
		return nil
	}); err != nil {
		return errors.Wrapf(err, "UpdateContainerVolume %d", index)
	}
	return nil
}

type ContainerAddVolumeMountPostOverlayTask struct {
	ContainerVolumeMountTaskPostOverlay
}

func (t *ContainerAddVolumeMountPostOverlayTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	ctr := obj.(*models.SContainer)
	if err := t.startCacheImage(ctx, ctr); err != nil {
		t.OnAddedFailed(ctx, ctr, jsonutils.NewString(err.Error()))
	}
}

func (t *ContainerAddVolumeMountPostOverlayTask) startCacheImage(ctx context.Context, ctr *models.SContainer) error {
	input, err := t.getInput()
	if err != nil {
		return errors.Wrap(err, "getInput")
	}
	volIdx := input.Index
	vol := ctr.Spec.VolumeMounts[volIdx]
	if vol.Disk == nil || vol.Disk.Id == "" {
		return errors.Wrapf(err, "invalid volume mount disk %s", jsonutils.Marshal(vol.Disk).String())
	}
	diskId := vol.Disk.Id
	taskInput := &api.ContainerCacheImagesInput{
		Images: make([]*api.ContainerCacheImageInput, 0),
	}
	for i := range input.PostOverlay {
		po := input.PostOverlay[i]
		if po.GetType() == apis.CONTAINER_VOLUME_MOUNT_DISK_POST_OVERLAY_IMAGE {
			if err := taskInput.Add(diskId, po.Image.Id, imageapi.IMAGE_DISK_FORMAT_TGZ); err != nil {
				return errors.Wrap(err, "add cached image to input")
			}
		}
	}
	if len(taskInput.Images) != 0 {
		t.SetStage("OnCachedImagesComplete", nil)
		return ctr.StartCacheImagesTask(ctx, t.GetUserCred(), taskInput, t.GetTaskId())
	} else {
		t.OnCachedImagesComplete(ctx, ctr, nil)
		return nil
	}
}

func (t *ContainerAddVolumeMountPostOverlayTask) OnCachedImagesComplete(ctx context.Context, ctr *models.SContainer, data jsonutils.JSONObject) {
	t.requestAdd(ctx, ctr)
}

func (t *ContainerAddVolumeMountPostOverlayTask) OnCachedImagesCompleteFailed(ctx context.Context, ctr *models.SContainer, data jsonutils.JSONObject) {
	t.OnAddedFailed(ctx, ctr, jsonutils.NewString(data.String()))
}

func (t *ContainerAddVolumeMountPostOverlayTask) getInput() (*api.ContainerVolumeMountAddPostOverlayInput, error) {
	input := new(api.ContainerVolumeMountAddPostOverlayInput)
	if err := t.GetParams().Unmarshal(input); err != nil {
		return nil, err
	}
	return input, nil
}

func (t *ContainerAddVolumeMountPostOverlayTask) requestAdd(ctx context.Context, c *models.SContainer) {
	t.SetStage("OnAdded", nil)
	if err := t.GetPodDriver().RequestAddVolumeMountPostOverlay(ctx, t.GetUserCred(), t); err != nil {
		t.OnAddedFailed(ctx, c, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerAddVolumeMountPostOverlayTask) OnAdded(ctx context.Context, c *models.SContainer, _ jsonutils.JSONObject) {
	if err := t.updateVolume(ctx, c); err != nil {
		t.OnAddedFailed(ctx, c, jsonutils.NewString(err.Error()))
		return
	}
	t.SetStage("OnSynced", nil)
	c.GetPod().StartSyncTask(ctx, t.GetUserCred(), false, t.GetTaskId())
}

func (t *ContainerAddVolumeMountPostOverlayTask) updateVolume(ctx context.Context, c *models.SContainer) error {
	input, err := t.getInput()
	if err != nil {
		return errors.Wrap(err, "getInput")
	}
	vm, err := c.GetAddPostOverlayVolumeMount(input.Index, input.PostOverlay)
	if err != nil {
		return errors.Wrap(err, "GetAddPostOverlayVolumeMount")
	}
	if err := t.UpdateContainerVolume(c, input.Index, vm); err != nil {
		return errors.Wrap(err, "UpdateContainerVolume")
	}
	return nil
}

func (t *ContainerAddVolumeMountPostOverlayTask) OnAddedFailed(ctx context.Context, c *models.SContainer, reason jsonutils.JSONObject) {
	c.SetStatus(ctx, t.GetUserCred(), api.CONTAINER_STATUS_ADD_POST_OVERLY_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *ContainerAddVolumeMountPostOverlayTask) OnSynced(ctx context.Context, c *models.SContainer, _ jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

type ContainerRemoveVolumeMountPostOverlayTask struct {
	ContainerVolumeMountTaskPostOverlay
}

func (t *ContainerRemoveVolumeMountPostOverlayTask) getInput() (*api.ContainerVolumeMountRemovePostOverlayInput, error) {
	input := new(api.ContainerVolumeMountRemovePostOverlayInput)
	if err := t.GetParams().Unmarshal(input); err != nil {
		return nil, err
	}
	return input, nil
}

func (t *ContainerRemoveVolumeMountPostOverlayTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.requestRemove(ctx, obj.(*models.SContainer))
}

func (t *ContainerRemoveVolumeMountPostOverlayTask) requestRemove(ctx context.Context, c *models.SContainer) {
	t.SetStage("OnRemoved", nil)
	// 如果是关机情况，并且还要 clear layers ，需要 mount disk 起来，然后清理
	if err := t.GetPodDriver().RequestRemoveVolumeMountPostOverlay(ctx, t.GetUserCred(), t); err != nil {
		t.OnRemovedFailed(ctx, c, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ContainerRemoveVolumeMountPostOverlayTask) updateVolume(ctx context.Context, c *models.SContainer) error {
	input, err := t.getInput()
	if err != nil {
		return errors.Wrap(err, "getInput")
	}
	vm, err := c.GetRemovePostOverlayVolumeMount(input.Index, input.PostOverlay)
	if err != nil {
		return errors.Wrap(err, "GetAddPostOverlayVolumeMount")
	}
	if err := t.UpdateContainerVolume(c, input.Index, vm); err != nil {
		return errors.Wrap(err, "UpdateContainerVolume")
	}
	return nil
}

func (t *ContainerRemoveVolumeMountPostOverlayTask) OnRemoved(ctx context.Context, c *models.SContainer, _ jsonutils.JSONObject) {
	if err := t.updateVolume(ctx, c); err != nil {
		t.OnRemovedFailed(ctx, c, jsonutils.NewString(err.Error()))
		return
	}
	t.SetStage("OnSynced", nil)
	c.GetPod().StartSyncTask(ctx, t.GetUserCred(), false, t.GetTaskId())
}

func (t *ContainerRemoveVolumeMountPostOverlayTask) OnRemovedFailed(ctx context.Context, c *models.SContainer, reason jsonutils.JSONObject) {
	c.SetStatus(ctx, t.GetUserCred(), api.CONTAINER_STATUS_REMOVE_POST_OVERLY_FAILED, reason.String())
	t.SetStageFailed(ctx, reason)
}

func (t *ContainerRemoveVolumeMountPostOverlayTask) OnSynced(ctx context.Context, c *models.SContainer, _ jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}
