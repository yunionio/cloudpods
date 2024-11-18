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

func init() {
	taskman.RegisterTask(ContainerBatchStartTask{})
}

type ContainerBatchStartTask struct {
	taskman.STask
}

func (t *ContainerBatchStartTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStage("OnContainersRestartComplete", nil)
	for i := range objs {
		ctr := objs[i].(*models.SContainer)
		if err := ctr.StartStartTask(ctx, t.GetUserCred(), t.GetId()); err != nil {
			t.SetStageFailed(ctx, jsonutils.NewString(fmt.Sprintf("start container %s: %s", ctr.GetName(), err.Error())))
			return
		}
	}
}

func (t *ContainerBatchStartTask) OnContainersRestartComplete(ctx context.Context, objs []db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}
