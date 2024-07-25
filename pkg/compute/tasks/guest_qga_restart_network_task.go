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
	guest.SetStatus(ctx, self.UserCred, api.VM_QGA_SET_NETWORK, "")
	self.OnRestartNetwork(ctx, guest)
}

func (self *GuestQgaRestartNetworkTask) OnRestartNetwork(ctx context.Context, guest *models.SGuest) {
	device, _ := self.Params.GetString("device")
	ipMask, _ := self.Params.GetString("ip_mask")
	gateway, _ := self.Params.GetString("gateway")
	prevIp, _ := self.Params.GetString("prev_ip")
	inBlockStream, _ := self.Params.Bool("in_block_stream")

	_, err := self.requestSetNetwork(ctx, guest, device, ipMask, gateway)
	if err != nil {
		logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RESTART_NETWORK, err, self.UserCred, false)
		self.taskFailed(ctx, guest, prevIp, inBlockStream, err)
	}
}

func (self *GuestQgaRestartNetworkTask) requestSetNetwork(ctx context.Context, guest *models.SGuest, device string, ipMask string, gateway string) (jsonutils.JSONObject, error) {
	host, err := guest.GetHost()
	if err != nil {
		self.taskFailed(ctx, guest, "", false, err)
		return nil, err
	}
	inputQgaNet := &api.ServerQgaSetNetworkInput{
		Device:  device,
		Ipmask:  ipMask,
		Gateway: gateway,
	}

	// if success, log network related information
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_QGA_NETWORK_INPUT, inputQgaNet, self.UserCred, true)
	drv, err := guest.GetDriver()
	if err != nil {
		return nil, err
	}

	self.SetStage("OnSetNetwork", nil)
	return drv.QgaRequestSetNetwork(ctx, self, jsonutils.Marshal(inputQgaNet), host, guest)
}

func (self *GuestQgaRestartNetworkTask) OnSetNetwork(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_QGA_NETWORK_SUCCESS, "qga restart network success", self.UserCred, true)
	guest.StartSyncstatus(ctx, self.UserCred, "")
	self.SetStageComplete(ctx, nil)
}

func (self *GuestQgaRestartNetworkTask) OnSetNetworkFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	prevIp, _ := self.Params.GetString("prev_ip")
	inBlockStream, _ := self.Params.Bool("in_block_stream")
	self.taskFailed(ctx, guest, prevIp, inBlockStream, errors.Errorf(data.String()))
}

func (self *GuestQgaRestartNetworkTask) taskFailed(ctx context.Context, guest *models.SGuest, prevIp string, inBlockStream bool, err error) {
	guest.SetStatus(ctx, self.GetUserCred(), api.VM_QGA_SET_NETWORK_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_RESTART_NETWORK, jsonutils.NewString(err.Error()), self.UserCred, false)
	self.SetStageFailed(ctx, nil)
}
