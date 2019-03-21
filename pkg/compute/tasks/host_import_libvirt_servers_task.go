package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/stringutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type HostImportLibvirtServersTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(HostImportLibvirtServersTask{})
	taskman.RegisterTask(CreateImportedLibvirtGuestTask{})
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
		self.TaskFailed(ctx, host, err.Error())
	}
}

func (self *HostImportLibvirtServersTask) OnRequestHostPrepareImport(
	ctx context.Context, host *models.SHost, body jsonutils.JSONObject,
) {
	serversNotMatch, _ := body.Get("servers_not_match")
	if !serversNotMatch.IsZero() {
		db.OpsLog.LogEvent(host, db.ACT_HOST_IMPORT_SERVERS_FROM_LIBVIRT,
			fmt.Sprintf("Servers %s not match host xml description", serversNotMatch), self.UserCred)
		logclient.AddActionLogWithContext(ctx, host, logclient.ACT_HOST_IMPORT_SERVERS_FROM_LIBVIRT,
			fmt.Sprintf("Servers %s not match host xml description", serversNotMatch), self.UserCred, false)
	}

	serversMatched, _ := body.Get("servers_matched")
	if serversMatched == nil {
		self.TaskFailed(ctx, host, "No matched server found")
		return
	}

	guestsDesc := []compute.SImportGuestDesc{}
	if err := serversMatched.Unmarshal(guestsDesc); err != nil {
		self.TaskFailed(ctx, host, fmt.Sprintf("Unmarshal matched servers failed: %s", err))
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
		self.FillLibvirtGuestDesc(ctx, host, &guestsDesc[i])
		guest, err := models.GuestManager.DoImport(ctx, self.UserCred, &guestsDesc[i])
		if err != nil {
			note = fmt.Sprintf("Guest %s import failed: %s", guestsDesc[i].Id, err)
			success = false
			goto end
		}
		if err := self.CreateImportedLibvirtGuestOnHost(ctx, host, guest, &guestsDesc[i]); err != nil {
			note = fmt.Sprintf("Guest  %s create on host failed: %s", guestsDesc[i].Id, err)
			success = false
			goto end
		}
		note = fmt.Sprintf("Guest %s import success, started create on host", guestsDesc[i].Id)
		success = true

	end:
		if success {
			db.OpsLog.LogEvent(host, db.ACT_HOST_IMPORT_SERVERS_FROM_LIBVIRT, note, self.UserCred)
		} else {
			db.OpsLog.LogEvent(host, db.ACT_HOST_IMPORT_SERVERS_FROM_LIBVIRT_FAIL, note, self.UserCred)
		}
		logclient.AddActionLogWithContext(ctx, host,
			logclient.ACT_HOST_IMPORT_SERVERS_FROM_LIBVIRT, note, self.UserCred, success)
	}
	self.SetStageComplete(ctx, nil)
}

func (self *HostImportLibvirtServersTask) FillLibvirtGuestDesc(
	ctx context.Context, host *models.SHost, guestDesc *compute.SImportGuestDesc,
) {
	guestDesc.HostId = host.Id
	for i := 0; i < len(guestDesc.Disks); i++ {
		guestDesc.Disks[i].DiskId = stringutils.UUID4()
		if len(guestDesc.Disks[i].Backend) == 0 {
			guestDesc.Disks[i].Backend = models.STORAGE_LOCAL
		}
	}
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

	_, err = host.Request(ctx, self.UserCred, "POST",
		fmt.Sprintf("/servers/%s/create-from-libvirt", guest.Id),
		task.GetTaskRequestHeader(), body)
	return err
}

func (self *HostImportLibvirtServersTask) TaskFailed(
	ctx context.Context, host *models.SHost, reason string,
) {
	self.SetStageFailed(ctx, reason)
	db.OpsLog.LogEvent(host, db.ACT_HOST_IMPORT_SERVERS_FROM_LIBVIRT_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, host,
		logclient.ACT_HOST_IMPORT_SERVERS_FROM_LIBVIRT, reason, self.UserCred, false)
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
		self.TaskFailed(ctx, guest, "guest create on host no disk access path feedback")
	}
	guestDisks := guest.GetDisks()
	for i := 0; i < len(guestDisks); i++ {
		disk := guestDisks[i].GetDisk()
		if accessPath, ok := disksPath[disk.Id]; !ok {
			self.TaskFailed(ctx, guest, fmt.Sprintf("Guest missing disk %s access path", disk.Id))
			return
		} else {
			_, err := db.Update(disk, func() error {
				disk.AccessPath = accessPath.String()
				return nil
			})
			if err != nil {
				self.TaskFailed(ctx, guest, fmt.Sprintf("Guest set disk access path error %s", err))
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
	self.TaskFailed(ctx, guest, body.String())
}

func (self *CreateImportedLibvirtGuestTask) OnInitFailed(
	ctx context.Context, guest *models.SGuest, body jsonutils.JSONObject,
) {
	self.TaskFailed(ctx, guest, body.String())
}

func (self *CreateImportedLibvirtGuestTask) TaskFailed(
	ctx context.Context, guest *models.SGuest, reason string,
) {
	guest.SetStatus(self.UserCred, compute.VM_IMPORT_FAILED, reason)
	db.OpsLog.LogEvent(guest, db.ACT_GUEST_CREATE_FROM_IMPORT_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest,
		logclient.ACT_GUEST_CREATE_FROM_IMPORT, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}
