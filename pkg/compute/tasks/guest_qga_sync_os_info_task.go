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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

func init() {
	taskman.RegisterTask(GuestQgaSyncOsInfoTask{})
}

type GuestQgaSyncOsInfoTask struct {
	SGuestBaseTask
}

func (self *GuestQgaSyncOsInfoTask) guestPing(ctx context.Context, guest *models.SGuest) error {
	host, err := guest.GetHost()
	if err != nil {
		return err
	}
	drv, err := guest.GetDriver()
	if err != nil {
		return err
	}
	return drv.QgaRequestGuestPing(ctx, self.GetTaskRequestHeader(), host, guest, true, nil)
}

func (self *GuestQgaSyncOsInfoTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	host, _ := guest.GetHost()
	drv, err := guest.GetDriver()
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
		return
	}
	res, err := drv.QgaRequestGetOsInfo(ctx, self.UserCred, nil, host, guest)
	if err != nil {
		self.taskFailed(ctx, guest, err.Error())
		return
	}
	self.updateOsInfo(ctx, guest, res)
}

type GuestOsInfo struct {
	Id            string `json:"id"`
	KernelRelease string `json:"kernel-release"`
	KernelVersion string `json:"kernel-version"`
	Machine       string `json:"machine"`
	Name          string `json:"name"`
	PrettyName    string `json:"pretty-name"`
	Version       string `json:"version"`
	VersionId     string `json:"version-id"`
}

func (self *GuestQgaSyncOsInfoTask) updateOsInfo(ctx context.Context, guest *models.SGuest, res jsonutils.JSONObject) {
	osInfo := new(GuestOsInfo)
	err := res.Unmarshal(osInfo)
	if err != nil {
		self.taskFailed(ctx, guest, fmt.Sprintf("failed unmarshal osinfo %s", err))
		return
	}
	osType := "Linux"
	if osInfo.Id == "mswindows" {
		osType = "Windows"
	}
	osDistribution := osInfo.PrettyName
	switch osInfo.Id {
	case "centos":
		osDistribution = "CentOS"
	case "debian":
		osDistribution = "Debian"
	case "ubuntu":
		osDistribution = "Ubuntu"
	case "fedora":
		osDistribution = "Fedora"
	case "openEuler":
		osDistribution = "OpenEuler"
	case "gentoo":
		osDistribution = "Gentoo"
	case "cirros":
		osDistribution = "Cirros"
	case "archLinux":
		osDistribution = "ArchLinux"
	case "kylin":
		osDistribution = "Kylin"
	case "anolis":
		osDistribution = "Anolis"
	}

	osInput := api.ServerSetOSInfoInput{
		Type:         osType,
		Distribution: osDistribution,
		Version:      osInfo.VersionId,
		Arch:         osInfo.Machine,
	}
	_, err = guest.PerformSetOsInfo(ctx, self.UserCred, nil, osInput)
	if err != nil {
		self.taskFailed(ctx, guest, fmt.Sprintf("failed set osinfo %s", err))
		return
	}
	self.OnUpdateOsInfoComplete(ctx, guest, osInput)
}

func (self *GuestQgaSyncOsInfoTask) taskFailed(ctx context.Context, guest *models.SGuest, reason string) {
	guest.SetStatus(ctx, self.UserCred, api.VM_QGA_EXEC_COMMAND_FAILED, reason)
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_OS_INFO_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SET_USER_PASSWORD, reason, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(reason))
}

func (self *GuestQgaSyncOsInfoTask) OnUpdateOsInfoComplete(ctx context.Context, guest *models.SGuest, osInput api.ServerSetOSInfoInput) {
	guest.SetStatus(ctx, self.UserCred, api.VM_RUNNING, "on qga set user password success")
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_OS_INFO, jsonutils.Marshal(osInput), self.UserCred)
	logclient.AddActionLogWithContext(ctx, guest, logclient.ACT_SET_USER_PASSWORD, jsonutils.Marshal(osInput), self.UserCred, false)
	self.SetStageComplete(ctx, nil)
}
