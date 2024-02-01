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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type MountTargetCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(MountTargetCreateTask{})
}

func (self *MountTargetCreateTask) taskFailed(ctx context.Context, mt *models.SMountTarget, err error) {
	mt.SetStatus(ctx, self.UserCred, api.MOUNT_TARGET_STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, mt, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *MountTargetCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	mt := obj.(*models.SMountTarget)

	fs, err := mt.GetFileSystem()
	if err != nil {
		self.taskFailed(ctx, mt, errors.Wrapf(err, "GetFileSystem"))
		return
	}

	ag, err := mt.GetAccessGroup()
	if err != nil {
		self.taskFailed(ctx, mt, errors.Wrapf(err, "mt.GetAccessGroup"))
		return
	}

	opts := cloudprovider.SMountTargetCreateOptions{
		NetworkType:   mt.NetworkType,
		AccessGroupId: ag.ExternalId,
	}

	iFs, err := fs.GetICloudFileSystem(ctx)
	if err != nil {
		self.taskFailed(ctx, mt, errors.Wrapf(err, "fs.GetICloudFileSystem"))
		return
	}

	if opts.NetworkType == api.NETWORK_TYPE_VPC {
		network, err := mt.GetNetwork()
		if err != nil {
			self.taskFailed(ctx, mt, errors.Wrapf(err, "mt.GetNetwork"))
			return
		}
		opts.NetworkId = network.ExternalId
		vpc, err := mt.GetVpc()
		if err != nil {
			self.taskFailed(ctx, mt, errors.Wrapf(err, "mt.GetVpc"))
			return
		}
		opts.VpcId = vpc.ExternalId
	}

	opts.FileSystemId = fs.ExternalId

	iMt, err := iFs.CreateMountTarget(&opts)
	if err != nil {
		self.taskFailed(ctx, mt, errors.Wrapf(err, "iFs.CreateMountTarget"))
		return
	}

	cloudprovider.Wait(time.Second*10, time.Minute*3, func() (bool, error) {
		mts, err := iFs.GetMountTargets()
		if err != nil {
			return false, errors.Wrapf(err, "iFs.GetMountTargets")
		}
		for i := range mts {
			if mts[i].GetGlobalId() == iMt.GetGlobalId() {
				status := mts[i].GetStatus()
				log.Infof("expect mount point status %s current is %s", api.MOUNT_TARGET_STATUS_AVAILABLE, status)
				if status == api.MOUNT_TARGET_STATUS_AVAILABLE {
					iMt = mts[i]
					return true, nil
				}
			}
		}
		return false, nil
	})

	mt.SyncWithMountTarget(ctx, self.GetUserCred(), fs.ManagerId, iMt)
	self.SetStageComplete(ctx, nil)
}
