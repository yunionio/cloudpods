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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type ProjectCleanTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ProjectCleanTask{})
}

func (task *ProjectCleanTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	project := obj.(*models.SProject)

	empty, err := project.GetEmptyProjects()
	if err != nil {
		logclient.AddActionLogWithStartable(task, project, logclient.ACT_CLEAN_PROJECT, errors.Wrapf(err, "GetEmptyProjects"), task.UserCred, false)
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}

	cnt, failed := 0, []error{}
	for i := range empty {
		lockman.LockObject(ctx, &empty[i])
		defer lockman.ReleaseObject(ctx, &empty[i])

		err = empty[i].Delete(ctx, task.UserCred)
		if err != nil {
			failed = append(failed, err)
			continue
		}
		cnt++
	}
	logclient.AddActionLogWithStartable(task, project, logclient.ACT_CLEAN_PROJECT, map[string]interface{}{"clean": cnt, "failed": errors.NewAggregate(failed).Error()}, task.UserCred, true)
	task.SetStageComplete(ctx, nil)
}
