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

package hostdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaremetalHostDriver struct {
	SBaseHostDriver
}

func init() {
	driver := SBaremetalHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SBaremetalHostDriver) GetHostType() string {
	return api.HOST_TYPE_BAREMETAL
}

func (self *SBaremetalHostDriver) GetHypervisor() string {
	return api.HYPERVISOR_BAREMETAL
}

func (self *SBaremetalHostDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (self *SBaremetalHostDriver) RequestBaremetalUnmaintence(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("/baremetals/%s/unmaintenance", baremetal.Id)
	headers := task.GetTaskRequestHeader()
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, task.GetParams())
	return err
}

func (self *SBaremetalHostDriver) RequestBaremetalMaintence(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("/baremetals/%s/maintenance", baremetal.Id)
	headers := task.GetTaskRequestHeader()
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, task.GetParams())
	return err
}

func (self *SBaremetalHostDriver) RequestSyncBaremetalHostStatus(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("/baremetals/%s/syncstatus", baremetal.Id)
	headers := task.GetTaskRequestHeader()
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, nil)
	return err
}

func (self *SBaremetalHostDriver) RequestSyncBaremetalHostConfig(ctx context.Context, userCred mcclient.TokenCredential, baremetal *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("/baremetals/%s/sync-config", baremetal.Id)
	headers := task.GetTaskRequestHeader()
	_, err := baremetal.BaremetalSyncRequest(ctx, "POST", url, headers, nil)
	return err
}

func (self *SBaremetalHostDriver) IsDisableImageCache(host *models.SHost) (bool, error) {
	agent := host.GetAgent(api.AgentTypeBaremetal)
	if agent == nil {
		return false, errors.Wrapf(errors.ErrNotFound, "get host %s(%s) agent", host.GetName(), host.GetId())
	}
	return agent.DisableImageCache, nil
}

func (self *SBaremetalHostDriver) CheckAndSetCacheImage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	input := api.CacheImageInput{}
	task.GetParams().Unmarshal(&input)
	_, err := models.CachedimageManager.FetchById(input.ImageId)
	if err != nil {
		return errors.Wrapf(err, "fetch cachedimage by image_id %s", input.ImageId)
	}

	disableCache, err := self.IsDisableImageCache(host)
	if err != nil {
		return errors.Wrapf(err, "check disable image cache by host %s(%s)", host.GetName(), host.GetId())
	}

	// iso must be cached to use
	if disableCache && input.Format != "iso" {
		task.ScheduleRun(nil)
		return nil
	}

	url := "/disks/image_cache"
	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&input), "disk")

	header := task.GetTaskRequestHeader()
	_, err = host.BaremetalSyncRequest(ctx, "POST", url, header, body)
	if err != nil {
		return err
	}
	return nil
}

func (self *SBaremetalHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, userCred mcclient.TokenCredential, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, input api.DiskAllocateInput) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestDeallocateDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, cleanSnapshots bool, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestRebuildDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, input api.DiskAllocateInput) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestResizeDiskOnHost(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, sizeMb int64, task taskman.ITask) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestSaveUploadImageOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error {
	return fmt.Errorf("not supported")
}

func (self *SBaremetalHostDriver) RequestUncacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask, deactivateImage bool) error {
	return fmt.Errorf("not supported")
}
