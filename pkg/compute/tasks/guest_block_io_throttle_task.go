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
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GuestBlockIoThrottleTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestBlockIoThrottleTask{})
}

func (self *GuestBlockIoThrottleTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	url := fmt.Sprintf("/servers/%s/io-throttle", guest.Id)
	headers := self.GetTaskRequestHeader()
	host := guest.GetHost()
	self.SetStage("OnIoThrottle", nil)
	_, err := host.Request(ctx, self.UserCred, "POST", url, headers, self.Params)
	if err != nil {
		self.SetStageFailed(ctx, err.Error())
	}
}

func (self *GuestBlockIoThrottleTask) OnIoThrottle(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetMetadata(ctx, "io-throttle", self.Params.String(), self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *GuestBlockIoThrottleTask) OnIoThrottleFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data.String())
}
