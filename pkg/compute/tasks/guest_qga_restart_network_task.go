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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type GuestQgaRestartNetworkTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestQgaRestartNetworkTask{})
}

func (self *GuestQgaRestartNetworkTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.SetStatus(self.GetUserCred(), api.VM_QGA_SET_NETWORK, "use qga restart network")
	self.SetStage("OnRestartNetwork", nil)
	err := guest.StartSyncTask(ctx, self.GetUserCred(), false, self.Id)
	if err != nil {
		self.taskFailed(ctx, guest, "", false, err)
		return
	}
}

func (self *GuestQgaRestartNetworkTask) OnRestartNetwork(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	device, _ := self.Params.GetString("device")
	ipMask, _ := self.Params.GetString("ipMask")
	gateway, _ := self.Params.GetString("gateway")
	prevIp, _ := self.Params.GetString("prevIp")
	inBlockStream, _ := self.Params.Bool("inBlockStream")
	time.Sleep(10 * time.Second)
	_, err := self.PerformSetNetwork(ctx, obj, device, ipMask, gateway)
	if err != nil {
		logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RESTART_NETWORK, err, self.UserCred, false)
		_, err = self.PerformSetNetwork(ctx, obj, device, ipMask, gateway)
	}
	if err != nil {
		logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RESTART_NETWORK, err, self.UserCred, false)
		self.taskFailed(ctx, guest, prevIp, inBlockStream, err)
		return
	}
	err = guest.StartSyncTask(ctx, self.GetUserCred(), false, self.Id)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestQgaRestartNetworkTask) PerformSetNetwork(ctx context.Context, obj db.IStandaloneModel, device string, ipMask string, gateway string) (jsonutils.JSONObject, error) {
	guest := obj.(*models.SGuest)
	inputQgaNet := &api.ServerQgaSetNetworkInput{
		Device:  device,
		Ipmask:  ipMask,
		Gateway: gateway,
	}

	// if success, log network related information
	notesNetwork := jsonutils.NewDict()
	notesNetwork.Add(jsonutils.NewString(inputQgaNet.Device), "Device")
	notesNetwork.Add(jsonutils.NewString(inputQgaNet.Ipmask), "Ipmask")
	notesNetwork.Add(jsonutils.NewString(inputQgaNet.Gateway), "Gateway")
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_QGA_NETWORK_INPUT, notesNetwork, self.UserCred, true)
	return guest.PerformQgaSetNetwork(ctx, self.UserCred, nil, inputQgaNet)
}

func (self *GuestQgaRestartNetworkTask) taskFailed(ctx context.Context, guest *models.SGuest, prevIp string, inBlockStream bool, err error) {
	guest.SetStatus(self.GetUserCred(), api.VM_QGA_SET_NETWORK_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RESTART_NETWORK, jsonutils.NewString(err.Error()), self.UserCred, false)
	if prevIp != "" {
		//use ansible to restart network
		guest.StartRestartNetworkTask(ctx, self.UserCred, "", prevIp, inBlockStream)
	}
	self.SetStageFailed(ctx, nil)
}
