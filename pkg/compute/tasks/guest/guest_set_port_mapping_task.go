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

package guest

import (
	"context"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

// GuestSetPortMappingTask 设置虚机网卡的端口映射，仅 kvm / pod 支持
//
// 分两步执行：
//  1. 请求宿主机设置端口映射（由宿主机负责分配 host_port，并回写 region 数据库）
//  2. 宿主机完成后再同步配置到宿主机
//
// 之所以把端口分配从 sync 里拆成独立的 task，是因为宿主机 sync 存在两次 task_complete
// 的完成握手（KVM 会先带 task 返回一次，1 秒后再异步返回一次），在 GuestSync 中做端口
// 分配与回写会阻塞首次完成，破坏该时序，导致虚机一直停留在 sync_config 状态。
type GuestSetPortMappingTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GuestSetPortMappingTask{})
}

func (self *GuestSetPortMappingTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	// 进行中状态已由 StartGuestSetPortMappingTask 置好，这里只负责失败时转为 set_portmapping_fail
	input := new(api.ServerSetPortMappingInput)
	if err := self.GetParams().Unmarshal(input); err != nil {
		self.setFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	if _, err := guest.FindGuestnetworkByInfo(input.ServerNetworkInfo); err != nil {
		self.setFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	host, err := guest.GetHost()
	if err != nil {
		self.setFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	drv, err := guest.GetDriver()
	if err != nil {
		self.setFailed(ctx, guest, jsonutils.NewString(err.Error()))
		return
	}
	self.SetStage("OnSetPortMappingComplete", nil)
	if err := drv.RequestSetPortMappingOnHost(ctx, self.UserCred, guest, host, self, *input); err != nil {
		self.setFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

// OnSetPortMappingComplete 宿主机已完成端口设置（含 host_port 分配与回写），开始同步配置
func (self *GuestSetPortMappingTask) OnSetPortMappingComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	self.SetStage("OnSyncComplete", nil)
	if err := guest.StartSyncTask(ctx, self.UserCred, false, self.GetTaskId()); err != nil {
		self.setFailed(ctx, guest, jsonutils.NewString(err.Error()))
	}
}

func (self *GuestSetPortMappingTask) OnSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *GuestSetPortMappingTask) OnSetPortMappingCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.setFailed(ctx, obj.(*models.SGuest), data)
}

func (self *GuestSetPortMappingTask) OnSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.setFailed(ctx, obj.(*models.SGuest), data)
}

func (self *GuestSetPortMappingTask) setFailed(ctx context.Context, guest *models.SGuest, data jsonutils.JSONObject) {
	guest.SetStatus(ctx, self.UserCred, api.VM_SET_PORTMAPPING_FAIL, data.String())
	db.OpsLog.LogEvent(guest, db.ACT_UPDATE, data, self.UserCred)
	logclient.AddActionLogWithStartable(self, guest, logclient.ACT_UPDATE, data, self.UserCred, false)
	self.SetStageFailed(ctx, data)
}
