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

package models

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IStatusBase interface {
	SetStatus(ctx context.Context, userCred mcclient.TokenCredential, status string, reason string) error
	GetStatus() string
}

type IStatusStandaloneBase interface {
	db.IStandaloneModel
	IStatusBase
}

func StartResourceSyncStatusTask(ctx context.Context, userCred mcclient.TokenCredential, obj IStatusStandaloneBase, taskName string, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(obj.GetStatus()), "origin_status")
	task, err := taskman.TaskManager.NewTask(ctx, taskName, obj, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	obj.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "perform_syncstatus")
	return task.ScheduleRun(nil)
}
