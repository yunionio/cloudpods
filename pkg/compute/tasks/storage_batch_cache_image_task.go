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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(StorageBatchCacheImageTask{})
}

type StorageBatchCacheImageTask struct {
	taskman.STask
}

func (t *StorageBatchCacheImageTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStage("OnStorageCacheImageComplete", nil)
	params := make([]api.CacheImageInput, 0)
	t.GetParams().Unmarshal(&params, "params")
	for i := range objs {
		sc := objs[i].(*models.SStoragecache)
		input := params[i]
		input.ParentTaskId = t.GetTaskId()
		if err := sc.StartImageCacheTask(ctx, t.GetUserCred(), input); err != nil {
			t.SetStageFailed(ctx, jsonutils.NewString(fmt.Sprintf("start image cache task %s failed: %s", sc.GetId(), err)))
			return
		}
	}
}

func (t *StorageBatchCacheImageTask) OnStorageCacheImageComplete(ctx context.Context, obj []db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}
