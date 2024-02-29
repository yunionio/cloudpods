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

// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or fsreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific langufse governing permissions and
// limitations under the License.

package tasks

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type FileSystemCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(FileSystemCreateTask{})
}

func (self *FileSystemCreateTask) taskFailed(ctx context.Context, fs *models.SFileSystem, err error) {
	fs.SetStatus(ctx, self.UserCred, api.NAS_STATUS_CREATE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, fs, logclient.ACT_ALLOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *FileSystemCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	fs := obj.(*models.SFileSystem)

	iRegion, err := fs.GetIRegion(ctx)
	if err != nil {
		self.taskFailed(ctx, fs, errors.Wrapf(err, "fs.GetIRegion"))
		return
	}

	zone, _ := fs.GetZone()

	opts := &cloudprovider.FileSystemCraeteOptions{
		Name:           fs.Name,
		Desc:           fs.Description,
		Capacity:       fs.Capacity,
		StorageType:    fs.StorageType,
		Protocol:       fs.Protocol,
		FileSystemType: fs.FileSystemType,
		ZoneId:         strings.TrimPrefix(zone.ExternalId, iRegion.GetGlobalId()+"/"),
	}

	netId := jsonutils.GetAnyString(self.GetParams(), []string{"network_id"})
	if len(netId) > 0 {
		net, err := models.NetworkManager.FetchById(netId)
		if err != nil {
			self.taskFailed(ctx, fs, errors.Wrapf(err, "NetworkManager.FetchById(%s)", netId))
			return
		}
		network := net.(*models.SNetwork)
		opts.NetworkId = network.ExternalId
		vpc, _ := network.GetVpc()
		opts.VpcId = vpc.ExternalId
	}

	log.Infof("nas create params: %s", jsonutils.Marshal(opts).String())

	iFs, err := iRegion.CreateICloudFileSystem(opts)
	if err != nil {
		self.taskFailed(ctx, fs, errors.Wrapf(err, "iRegion.CreateICloudFileSystem"))
		return
	}
	db.SetExternalId(fs, self.GetUserCred(), iFs.GetGlobalId())

	cloudprovider.WaitMultiStatus(iFs, []string{api.NAS_STATUS_AVAILABLE, api.NAS_STATUS_CREATE_FAILED}, time.Second*5, time.Minute*10)

	tags, _ := fs.GetAllUserMetadata()
	if len(tags) > 0 {
		err = iFs.SetTags(tags, true)
		if err != nil {
			logclient.AddActionLogWithStartable(self, fs, logclient.ACT_UPDATE, errors.Wrapf(err, "SetTags"), self.UserCred, false)
		}
	}

	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    self,
		Action: notifyclient.ActionCreate,
	})

	self.SetStage("OnSyncstatusComplete", nil)
	fs.StartSyncstatus(ctx, self.GetUserCred(), self.GetTaskId())
}

func (self *FileSystemCreateTask) OnSyncstatusComplete(ctx context.Context, fs *models.SFileSystem, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *FileSystemCreateTask) OnSyncstatusCompleteFailed(ctx context.Context, fs *models.SFileSystem, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
