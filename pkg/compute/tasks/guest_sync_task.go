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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestSyncConfTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSyncConfTask{})
}

func (self *GuestSyncConfTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_CONF, nil, self.UserCred)
	if host, _ := guest.GetHost(); host == nil {
		self.SetStageFailed(ctx, jsonutils.NewString("No host for sync"))
		return
	} else {
		self.SetStage("on_sync_complete", nil)
		if err := guest.GetDriver().RequestSyncConfigOnHost(ctx, guest, host, self); err != nil {
			self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
			log.Errorf("SyncConfTask faled %v", err)
		}
	}
}

func (self *GuestSyncConfTask) OnSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if restart, _ := self.Params.Bool("restart_network"); restart {
		self.StartRestartNetworkTask(ctx, guest)
	} else if data.Contains("task") {
		// XXX this is only applied to KVM, which will call task_complete twice
		self.SetStage("on_disk_sync_complete", nil)
	} else {
		self.OnDiskSyncComplete(ctx, guest, data)
	}
}

func (self *GuestSyncConfTask) StartRestartNetworkTask(ctx context.Context, guest *models.SGuest) {
	defer self.SetStageComplete(ctx, guest.GetShortDesc(ctx))
	prevIp, err := self.Params.GetString("prev_ip")
	if err != nil {
		log.Errorf("unable to get prev_ip when restart_network is true when sync guest")
		return
	}
	inBlockStream := jsonutils.QueryBoolean(self.Params, "in_block_stream", false)
	preMac, err := self.Params.GetString("prev_mac")
	if err != nil {
		log.Errorf("unable to get prev_mac when restart_network is true when sync guest")
		return
	}
	ipMask, err := self.Params.GetString("ip_mask")
	if err != nil {
		log.Errorf("unable to get ip_mask when restart_network is true when sync guest")
		return
	}
	gateway, err := self.Params.GetString("gateway")
	if err != nil {
		log.Errorf("unable to get gateway when restart_network is true when sync guest")
		return
	}
	isVpcNetwork := jsonutils.QueryBoolean(self.Params, "is_vpc_network", false)

	// try use qga restart network
	err = func() error {
		host, _ := guest.GetHost()
		err = guest.GetDriver().QgaRequestGuestPing(ctx, self.GetTaskRequestHeader(), host, guest, false, &api.ServerQgaTimeoutInput{1000})
		if err != nil {
			return errors.Wrap(err, "qga guest-ping")
		}
		ifnameDevice, err := guest.GetIfNameByMac(ctx, self.UserCred, preMac)
		if err != nil {
			return errors.Wrap(err, "get ifname by mac")
		}
		if ifnameDevice == "" {
			return errors.Errorf("failed find ifname")
		}
		if isVpcNetwork {
			// wait for vpcagent sync network topo
			time.Sleep(10 * time.Second)
		}
		return guest.StartQgaRestartNetworkTask(
			ctx, self.UserCred, "", ifnameDevice, ipMask, gateway, prevIp, inBlockStream)
	}()
	if err != nil {
		log.Errorf("guest %s failed start qga restart network task: %s", guest.GetName(), err)
		guest.SetStatus(self.GetUserCred(), api.VM_QGA_SET_NETWORK_FAILED, err.Error())
		logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RESTART_NETWORK, jsonutils.NewString(err.Error()), self.UserCred, false)
	}
}

func (self *GuestSyncConfTask) OnDiskSyncComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	if jsonutils.QueryBoolean(self.Params, "without_sync_status", false) {
		self.OnSyncStatusComplete(ctx, guest, nil)
	} else {
		self.SetStage("on_sync_status_complete", nil)
		guest.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
	}
}

func (self *GuestSyncConfTask) OnDiskSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_CONF_FAIL, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_SYNC_CONF, data, self.UserCred, false)
	if !jsonutils.QueryBoolean(self.Params, "without_sync_status", false) {
		guest.SetStatus(self.GetUserCred(), api.VM_SYNC_FAIL, data.String())
	}
	self.SetStageFailed(ctx, data)
}

func (self *GuestSyncConfTask) OnSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	if !jsonutils.QueryBoolean(self.Params, "without_sync_status", false) {
		guest.SetStatus(self.GetUserCred(), api.VM_SYNC_FAIL, data.String())
	}
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_VM_SYNC_CONF, data, self.UserCred, false)
	db.OpsLog.LogEvent(guest, db.ACT_SYNC_CONF_FAIL, data, self.UserCred)
	self.SetStageFailed(ctx, data)
}

func (self *GuestSyncConfTask) OnSyncStatusComplete(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSyncConfTask) OnSyncStatusCompleteFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
