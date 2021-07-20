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
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type NetworksUnderWireMergeTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(NetworksUnderWireMergeTask{})
}

func (self *NetworksUnderWireMergeTask) taskFailed(ctx context.Context, wire *models.SWire, desc string, err error) {
	d := jsonutils.NewDict()
	d.Set("description", jsonutils.NewString(desc))
	if err != nil {
		d.Set("error", jsonutils.NewString(err.Error()))
	}
	wire.SetStatus(self.UserCred, api.WIRE_STATUS_MERGE_NETWORK_FAILED, d.PrettyString())
	db.OpsLog.LogEvent(wire, db.ACT_MERGE_NETWORK_FAILED, d, self.UserCred)
	logclient.AddActionLogWithStartable(self, wire, logclient.ACT_MERGE_NETWORK, d, self.UserCred, false)
	self.SetStageFailed(ctx, nil)
}

func (self *NetworksUnderWireMergeTask) taskSuccess(ctx context.Context, wire *models.SWire, desc string) {
	d := jsonutils.NewString(desc)
	wire.SetStatus(self.UserCred, api.WIRE_STATUS_AVAILABLE, "")
	db.OpsLog.LogEvent(wire, db.ACT_MERGE_NETWORK, d, self.UserCred)
	logclient.AddActionLogWithStartable(self, wire, logclient.ACT_MERGE_NETWORK, d, self.UserCred, true)
	self.SetStageComplete(ctx, nil)
}

type Net struct {
	*models.SNetwork
	StartIp netutils.IPV4Addr
}

func (self *NetworksUnderWireMergeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	w := obj.(*models.SWire)
	w.SetStatus(self.UserCred, api.WIRE_STATUS_MERGE_NETWORK, "")

	lockman.LockClass(ctx, models.NetworkManager, db.GetLockClassKey(models.NetworkManager, self.UserCred))
	defer lockman.ReleaseClass(ctx, models.NetworkManager, db.GetLockClassKey(models.NetworkManager, self.UserCred))
	networks, err := w.GetNetworks(self.UserCred, rbacutils.ScopeDomain)
	if err != nil {
		self.taskFailed(ctx, w, "unable to GetNetworks", err)
		return
	}
	if len(networks) <= 1 {
		self.taskSuccess(ctx, w, fmt.Sprintf("num of networks under wire is %d", len(networks)))
	}
	nets := make([]Net, len(networks))
	for i := range nets {
		startIp, _ := netutils.NewIPV4Addr(networks[i].GuestIpStart)
		nets[i] = Net{
			SNetwork: &networks[i],
			StartIp:  startIp,
		}
	}
	sort.Slice(nets, func(i, j int) bool {
		if nets[i].VlanId == nets[j].VlanId {
			return nets[i].StartIp < nets[j].StartIp
		}
		return nets[i].VlanId < nets[j].VlanId
	})
	log.Infof("nets sorted: %s", jsonutils.Marshal(nets))
	for i := 0; i < len(nets)-1; i++ {
		if nets[i].VlanId != nets[i+1].VlanId {
			continue
		}
		// preparenets
		wireNets := make([]*models.SNetwork, 0, len(nets)-2)
		for j := range nets {
			if j != i && j != i+1 {
				wireNets = append(wireNets, nets[i].SNetwork)
			}
		}
		ok, err := self.mergeNetwork(ctx, nets[i].SNetwork, nets[i+1].SNetwork, wireNets)
		if err != nil {
			self.taskFailed(ctx, w, fmt.Sprintf("unable to merge network %q to %q", nets[i].GetId(), nets[i+1].GetId()), err)
			return
		}
		if ok {
			continue
		}
		// Try to merge in the opposite direction
		ok, err = self.mergeNetwork(ctx, nets[i+1].SNetwork, nets[i].SNetwork, wireNets)
		if err != nil {
			self.taskFailed(ctx, w, fmt.Sprintf("unable to merge network %q to %q", nets[i+1].GetId(), nets[i].GetId()), err)
			return
		}
		if ok {
			// Swap position
			nets[i], nets[i+1] = nets[i+1], nets[i]
		}
	}
	self.taskSuccess(ctx, w, "")
}

func (self *NetworksUnderWireMergeTask) mergeNetwork(ctx context.Context, source, target *models.SNetwork, wireNets []*models.SNetwork) (bool, error) {
	startIp, endIp, err := source.CheckInvalidToMerge(ctx, target, wireNets)
	if err != nil {
		log.Debugf("unable to merge network %q to %q: %v", source.GetId(), target.GetId(), err)
		return false, nil
	}
	err = source.MergeToNetworkAfterCheck(ctx, self.UserCred, target, startIp, endIp)
	if err != nil {
		return false, err
	}
	return true, nil
}
