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

package guestdrivers

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

var (
	containerUseKubectlError error = httperrors.NewUnsupportOperationError("Not supported, please use kubectl")
)

type SContainerDriver struct {
	SVirtualizedGuestDriver
}

func init() {
	driver := SContainerDriver{}
	models.RegisterGuestDriver(&driver)
}

func (self *SContainerDriver) newUnsupportOperationError(option string) error {
	return httperrors.NewUnsupportOperationError("Container not support %s", option)
}

func (self *SContainerDriver) GetHypervisor() string {
	return api.HYPERVISOR_CONTAINER
}

func (self *SContainerDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ONECLOUD
}

func (self *SContainerDriver) GetQuotaPlatformID() []string {
	return []string{
		api.CLOUD_ENV_ON_PREMISE,
		api.CLOUD_PROVIDER_ONECLOUD,
		api.HYPERVISOR_CONTAINER,
	}
}

func (self *SContainerDriver) GetDefaultSysDiskBackend() string {
	return api.STORAGE_LOCAL
}

func (self *SContainerDriver) GetMinimalSysDiskSizeGb() int {
	return options.Options.DefaultDiskSizeMB / 1024
}

func (self *SContainerDriver) RequestGuestCreateAllDisks(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SContainerDriver) RequestGuestHotAddIso(ctx context.Context, guest *models.SGuest, path string, boot bool, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SContainerDriver) RequestStartOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential, task taskman.ITask) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewUnsupportOperationError("")
}

func (self *SContainerDriver) RequestStopOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	return containerUseKubectlError
}

func (self *SContainerDriver) RqeuestSuspendOnHost(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return containerUseKubectlError
}

func (self *SContainerDriver) RequestSoftReset(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return containerUseKubectlError
}

func (self *SContainerDriver) RequestDetachDisk(ctx context.Context, guest *models.SGuest, disk *models.SDisk, task taskman.ITask) error {
	return containerUseKubectlError
}

func (self *SContainerDriver) RequestSyncstatusOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, userCred mcclient.TokenCredential) (jsonutils.JSONObject, error) {
	// always return running
	status := jsonutils.NewDict()
	status.Add(jsonutils.NewString("running"), "status")
	return status, nil
}

func (self *SContainerDriver) CanKeepDetachDisk() bool {
	return false
}

func (self *SContainerDriver) GetGuestVncInfo(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) (*jsonutils.JSONDict, error) {
	return nil, self.newUnsupportOperationError("VNC")
}

func (self *SContainerDriver) OnGuestDeployTaskDataReceived(ctx context.Context, guest *models.SGuest, task taskman.ITask, data jsonutils.JSONObject) error {
	//guest.SaveDeployInfo(ctx, task.GetUserCred(), data)
	// do nothing here
	return nil
}

func (self *SContainerDriver) RequestStopGuestForDelete(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SContainerDriver) RequestDetachDisksFromGuestForDelete(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SContainerDriver) RequestUndeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	url := fmt.Sprintf("%s/servers/%s", host.ManagerUri, guest.Id)
	header := self.getTaskRequestHeader(task)
	_, _, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "DELETE", url, header, nil, false)
	return err
}

func (self *SContainerDriver) GetJsonDescAtHost(ctx context.Context, userCred mcclient.TokenCredential, guest *models.SGuest, host *models.SHost) jsonutils.JSONObject {
	return guest.GetJsonDescAtHypervisor(ctx, host)
}

func (self *SContainerDriver) RequestDeployGuestOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	config, err := guest.GetDeployConfigOnHost(ctx, task.GetUserCred(), host, task.GetParams())
	if err != nil {
		log.Errorf("GetDeployConfigOnHost error: %v", err)
		return err
	}
	config.Add(jsonutils.JSONTrue, "k8s_pod")
	action, err := config.GetString("action")
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/servers/%s/%s", host.ManagerUri, guest.Id, action)
	header := self.getTaskRequestHeader(task)
	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, config, false)
	return err
}

func (self *SContainerDriver) OnDeleteGuestFinalCleanup(ctx context.Context, guest *models.SGuest, userCred mcclient.TokenCredential) error {
	// clean disk records in DB
	return guest.DeleteAllDisksInDB(ctx, userCred)
}

func (self *SContainerDriver) RequestSyncConfigOnHost(ctx context.Context, guest *models.SGuest, host *models.SHost, task taskman.ITask) error {
	// do nothing, call next stage
	task.ScheduleRun(nil)
	return nil
}

func (self *SContainerDriver) DoGuestCreateDisksTask(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	return self.newUnsupportOperationError("create disk")
}

func (self *SContainerDriver) RequestChangeVmConfig(ctx context.Context, guest *models.SGuest, task taskman.ITask, instanceType string, vcpuCount, vmemSize int64) error {
	return self.newUnsupportOperationError("change config")
}

func (self *SContainerDriver) RequestRebuildRootDisk(ctx context.Context, guest *models.SGuest, task taskman.ITask) error {
	// do nothing, call next stage
	return self.newUnsupportOperationError("rebuild root")
}

func (self *SContainerDriver) GetRandomNetworkTypes() []string {
	return []string{api.NETWORK_TYPE_CONTAINER, api.NETWORK_TYPE_GUEST}
}

func (self *SContainerDriver) StartGuestRestartTask(guest *models.SGuest, ctx context.Context, userCred mcclient.TokenCredential, isForce bool, parentTaskId string) error {
	return fmt.Errorf("Not Implement")
}

func (self *SContainerDriver) IsSupportGuestClone() bool {
	return false
}

func (self *SContainerDriver) IsSupportCdrom(guest *models.SGuest) (bool, error) {
	return false, nil
}
