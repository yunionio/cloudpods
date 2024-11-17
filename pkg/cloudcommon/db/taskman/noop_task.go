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

package taskman

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type NoopTask struct {
	STask
}

func init() {
	RegisterTask(NoopTask{})
}

func (task *NoopTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	task.SetStageComplete(ctx, nil)
}

func StartNoopTask(ctx context.Context, userCred mcclient.TokenCredential, obj db.IStandaloneModel, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := TaskManager.NewTask(ctx, "NoopTask", obj, userCred, params, parentTaskId, "")
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	err = task.ScheduleRun(nil)
	if err != nil {
		return errors.Wrap(err, "ScheduleRun")
	}
	return nil
}
