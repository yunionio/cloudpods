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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var ScheduledTaskLabelManager *SScheduledTaskLabelManager

func init() {
	db.InitManager(func() {
		ScheduledTaskLabelManager = &SScheduledTaskLabelManager{
			SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
				SScheduledTaskLabel{},
				"scheduledtasklabels_tbl",
				"scheduledtasklabel",
				"scheduledtasklabels",
				ScheduledTaskManager,
				GuestManager,
			),
		}
	})
}

type SScheduledTaskLabelManager struct {
	db.SVirtualJointResourceBaseManager
}

type SScheduledTaskLabel struct {
	db.SVirtualJointResourceBase
	ScheduledTaskId string `width:"36" charset:"ascii" nullable:"false" index:"true"`
	Label           string `width:"64" charset:"utf8" nullable:"false" index:"true"`
}

func (slm *SScheduledTaskLabelManager) GetMasterFieldName() string {
	return "scheduled_task_id"
}

func (slm *SScheduledTaskLabelManager) GetSlaveFieldName() string {
	return "label"
}

func (slm *SScheduledTaskLabelManager) Attach(ctx context.Context, taskId, label string) error {
	sl := &SScheduledTaskLabel{
		ScheduledTaskId: taskId,
		Label:           label,
	}
	return slm.TableSpec().Insert(ctx, sl)
}

func (sl *SScheduledTaskLabel) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, sl)
}
