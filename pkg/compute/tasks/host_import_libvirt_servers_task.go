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
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(HostImportLibvirtServersTask{})
	taskman.RegisterTask(CreateImportedLibvirtGuestTask{})
}

type HostImportLibvirtServersTask struct {
	taskman.STask
}

func (self *HostImportLibvirtServersTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject,
) {
	host := obj.(*models.SHost)
	self.SetStage("OnRequestHostPrepareImport", nil)
	self.RequestHostPrepareImport(ctx, host)
}

func (self *HostImportLibvirtServersTask) RequestHostPrepareImport(
	ctx context.Context, host *models.SHost,
) {
	header := self.GetTaskRequestHeader()
	if _, err := host.Request(ctx, self.UserCred, "POST",
		"/servers/prepare-import-from-libvirt", header, self.Params); err != nil {
		self.TaskFailed(ctx, host, jsonutils.NewString(err.Error()))
	}
}

func (self *HostImportLibvirtServersTask) OnRequestHostPrepareImport(
	ctx context.Context, host *models.SHost, body jsonutils.JSONObject,
) {
	serversNotMatch, _ := body.GetArray("servers_not_match")
	if len(serversNotMatch) > 0 {
		db.OpsLog.LogEvent(host, db.ACT_HOST_IMPORT_LIBVIRT_SERVERS,
			fmt.Sprintf("Servers %s not match host xml description", serversNotMatch), self.UserCred)
		logclient.AddActionLogWithContext(ctx, host, logclient.ACT_HOST_IMPORT_LIBVIRT_SERVERS,
			fmt.Sprintf("Servers %s not match host xml description", serversNotMatch), self.UserCred, false)
	}

	serversMatched, _ := body.Get("servers_matched")
	if serversMatched == nil {
		self.TaskFailed(ctx, host, jsonutils.NewString("No matched server found"))
		return
	}

	guestsDesc := []compute.SImportGuestDesc{}
	if err := serversMatched.Unmarshal(&guestsDesc); err != nil {
		self.TaskFailed(ctx, host, jsonutils.NewString(fmt.Sprintf("Unmarshal matched servers failed: %s", err)))
		return
	}
	self.StartImportServers(ctx, host, guestsDesc)
}

func (self *HostImportLibvirtServersTask) StartImportServers(
	ctx context.Context, host *models.SHost, guestsDesc []compute.SImportGuestDesc,
) {
	var (
		note    string
		success bool
	)
	for i := 0; i < len(guestsDesc); i++ {
		var (
			guest    *models.SGuest = nil
			originId string         = guestsDesc[i].Id
		)

		func() {
			lockman.LockRawObject(ctx, models.GuestManager.Keyword(), "name")
			defer lockman.ReleaseRawObject(ctx, models.GuestManager.Keyword(), "name")

			err := self.FillLibvirtGuestDesc(ctx, host, &guestsDesc[i])
			if err != nil {
				note = fmt.Sprintf("Guest %s desc fill failed: %s", guestsDesc[i].Id, err)
				success = false
			} else {
				guest, err = models.GuestManager.DoImport(ctx, self.UserCred, &guestsDesc[i])
				if err != nil {
					note = fmt.Sprintf("Guest %s import failed: %s", guestsDesc[i].Id, err)
					success = false
				} else {
					meta := map[string]interface{}{
						"__is_import":    "true",
						"__origin_id":    originId,
						"__monitor_path": guestsDesc[i].MonitorPath,
					}
					guest.SetAllMetadata(ctx, meta, self.UserCred)
					if err := self.CreateImportedLibvirtGuestOnHost(ctx, host, guest, &guestsDesc[i]); err != nil {
						note = fmt.Sprintf("Guest  %s create on host failed: %s", guestsDesc[i].Id, err)
						success = false
					} else {
						note = fmt.Sprintf("Guest %s import success, started create on host", guestsDesc[i].Id)
						success = true
					}
				}
			}
		}()

		if success {
			db.OpsLog.LogEvent(host, db.ACT_HOST_IMPORT_LIBVIRT_SERVERS, note, self.UserCred)
		} else {
			log.Errorln(note)
			if guest != nil {
				guest.SetStatus(self.UserCred, compute.VM_IMPORT_FAILED, note)
			}
			db.OpsLog.LogEvent(host, db.ACT_HOST_IMPORT_LIBVIRT_SERVERS_FAIL, note, self.UserCred)
		}
		logclient.AddActionLogWithContext(ctx, host,
			logclient.ACT_HOST_IMPORT_LIBVIRT_SERVERS, note, self.UserCred, success)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *HostImportLibvirtServersTask) FillLibvirtGuestDesc(
	ctx context.Context, host *models.SHost, guestDesc *compute.SImportGuestDesc,
) error {
	// Generate new uuid for guest to prevent duplicate
	guestDesc.Id = stringutils.UUID4()
	guestDesc.HostId = host.Id
	newName, err := db.GenerateName(ctx, models.GuestManager, self.UserCred, guestDesc.Name)
	if err != nil {
		return err
	}
	guestDesc.Name = newName
	for i := 0; i < len(guestDesc.Disks); i++ {
		guestDesc.Disks[i].DiskId = stringutils.UUID4()
		if len(guestDesc.Disks[i].Backend) == 0 {
			guestDesc.Disks[i].Backend = api.STORAGE_LOCAL
		}
	}
	return nil
}

// Create sub task to create guest on host, and feedback disk real access path
func (self *HostImportLibvirtServersTask) CreateImportedLibvirtGuestOnHost(
	ctx context.Context, host *models.SHost, guest *models.SGuest, guestDesc *compute.SImportGuestDesc,
) error {
	task, err := taskman.TaskManager.NewTask(
		ctx, "CreateImportedLibvirtGuestTask", guest, self.UserCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	disksPath := jsonutils.NewDict()
	for _, disk := range guestDesc.Disks {
		disksPath.Set(disk.DiskId, jsonutils.NewString(disk.AccessPath))
	}
	body := jsonutils.NewDict()
	body.Set("desc", guest.GetJsonDescAtHypervisor(ctx, host))
	body.Set("disks_path", disksPath)
	if len(guestDesc.MonitorPath) > 0 {
		body.Set("monitor_path", jsonutils.NewString(guestDesc.MonitorPath))
	}

	_, err = host.Request(ctx, self.UserCred, "POST",
		fmt.Sprintf("/servers/%s/create-from-libvirt", guest.Id),
		task.GetTaskRequestHeader(), body)
	return err
}

func (self *HostImportLibvirtServersTask) TaskFailed(
	ctx context.Context, host *models.SHost, reason jsonutils.JSONObject,
) {
	self.SetStageFailed(ctx, reason)
	db.OpsLog.LogEvent(host, db.ACT_HOST_IMPORT_LIBVIRT_SERVERS_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, host,
		logclient.ACT_HOST_IMPORT_LIBVIRT_SERVERS, reason, self.UserCred, false)
}

type CreateImportedLibvirtGuestTask struct {
	taskman.STask
}

func (self *CreateImportedLibvirtGuestTask) OnInit(
	ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject,
) {
	guest := obj.(*models.SGuest)
	disksPath, err := body.GetMap("disks_path")
	if err != nil {
		self.TaskFailed(ctx, guest, jsonutils.NewString("guest create on host no disk access path feedback"))
	}
	guestDisks := guest.GetDisks()
	for i := 0; i < len(guestDisks); i++ {
		disk := guestDisks[i].GetDisk()
		if accessPath, ok := disksPath[disk.Id]; !ok {
			self.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Guest missing disk %s access path", disk.Id)))
			return
		} else {
			_, err := db.Update(disk, func() error {
				disk.AccessPath = accessPath.String()
				if guestDisks[i].Index == 0 {
					disk.DiskType = api.DISK_TYPE_SYS
				}
				return nil
			})
			if err != nil {
				self.TaskFailed(ctx, guest, jsonutils.NewString(fmt.Sprintf("Guest set disk access path error %s", err)))
				return
			}
		}
	}

	self.SetStage("OnGuestSync", nil)
	guest.StartSyncTask(ctx, self.UserCred, false, self.GetTaskId())
}

func (self *CreateImportedLibvirtGuestTask) OnGuestSync(
	ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject,
) {
	db.OpsLog.LogEvent(guest, db.ACT_GUEST_CREATE_FROM_IMPORT_SUCC, "", self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest,
		logclient.ACT_GUEST_CREATE_FROM_IMPORT, "", self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

func (self *CreateImportedLibvirtGuestTask) OnGuestSyncFailed(
	ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject,
) {
	self.TaskFailed(ctx, guest, body)
}

func (self *CreateImportedLibvirtGuestTask) OnInitFailed(
	ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject,
) {
	self.TaskFailed(ctx, guest, body)
}

func (self *CreateImportedLibvirtGuestTask) TaskFailed(
	ctx context.Context, guest *models.SGuest, reason jsonutils.JSONObject,
) {
	guest.SetStatus(self.UserCred, compute.VM_IMPORT_FAILED, reason.String())
	db.OpsLog.LogEvent(guest, db.ACT_GUEST_CREATE_FROM_IMPORT_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest,
		logclient.ACT_GUEST_CREATE_FROM_IMPORT, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}
