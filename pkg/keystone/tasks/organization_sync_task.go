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
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type OrganizationSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(OrganizationSyncTask{})
}

func (task *OrganizationSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	org := obj.(*models.SOrganization)

	resourceType, _ := task.Params.GetString("resource_type")

	log.Debugf("start sync tags to org %s %s resourceType: %s", org.Name, org.Keys, resourceType)

	err := org.SyncTags(ctx, task.UserCred, resourceType)
	if err != nil {
		log.Errorf("SyncTags fail %s", err)
		task.SetStageFailed(ctx, jsonutils.Marshal(err))
		org.SetStatus(ctx, task.UserCred, api.OrganizationStatusSyncFailed, err.Error())
		return
	}

	org.SetStatus(ctx, task.UserCred, api.OrganizationStatusReady, "sync success")

	notes := jsonutils.NewDict()
	notes.Set("organization", org.GetShortDesc(ctx))
	logclient.AddActionLogWithStartable(task, org, logclient.ACT_BIND_DISK, notes, task.UserCred, true)
	db.OpsLog.LogEvent(org, db.ACT_BIND, notes, task.UserCred)

	task.SetStageComplete(ctx, notes)
}
