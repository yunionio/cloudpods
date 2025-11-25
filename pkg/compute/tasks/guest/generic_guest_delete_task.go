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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type GenericGuestDeleteTask struct {
	SGuestBaseTask
}

func init() {
	taskman.RegisterTask(GenericGuestDeleteTask{})
}

func (t *GenericGuestDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	t.startDeleteKvm(ctx, obj.(*models.SGuest))
}

func (t *GenericGuestDeleteTask) startDeleteKvm(ctx context.Context, kvm *models.SGuest) {
	t.SetStage("OnBaseGuestDeleted", nil)
	err := kvm.StartBaseDeleteTask(ctx, t)
	if err != nil {
		t.OnBaseGuestDeletedFailed(ctx, kvm, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *GenericGuestDeleteTask) OnBaseGuestDeleted(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guest := obj.(*models.SGuest)
	guest.FinalizeDeleteTask(ctx, t.GetUserCred(), t, data)
	t.SetStageComplete(ctx, nil)
}

func (t *GenericGuestDeleteTask) OnBaseGuestDeletedFailed(ctx context.Context, obj db.IStandaloneModel, reason jsonutils.JSONObject) {
	t.SetStageFailed(ctx, reason)
}
