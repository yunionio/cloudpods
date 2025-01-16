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
	"database/sql"
	"fmt"
	"sync/atomic"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type DiskCleanOverduedSnapshots struct {
	SDiskBaseTask
}

func init() {
	taskman.RegisterTask(SnapshotCleanupTask{})
}

type SnapshotCleanupTask struct {
	taskman.STask
}

func (self *SnapshotCleanupTask) taskFailed(ctx context.Context, reason *jsonutils.JSONString) {
	log.Infof("SnapshotCleanupTask failed %s", reason)
	atomic.CompareAndSwapInt32(&models.SnapshotCleanupTaskRunning, 1, 0)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotCleanupTask) taskCompleted(ctx context.Context, data jsonutils.JSONObject) {
	log.Infof("SnapshotCleanupTask completed %s", data)
	atomic.CompareAndSwapInt32(&models.SnapshotCleanupTaskRunning, 1, 0)
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotCleanupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	atomic.CompareAndSwapInt32(&models.SnapshotCleanupTaskRunning, 0, 1)
	now, err := self.Params.GetTime("tick")
	if err != nil {
		self.taskFailed(ctx, jsonutils.NewString("failed get tick"))
		return
	}
	var snapshots = make([]models.SSnapshot, 0)
	err = models.SnapshotManager.Query().
		Equals("fake_deleted", false).
		Equals("created_by", compute.SNAPSHOT_AUTO).
		LE("expired_at", now).All(&snapshots)
	if err == sql.ErrNoRows {
		self.taskCompleted(ctx, nil)
		return
	} else if err != nil {
		self.taskFailed(ctx, jsonutils.NewString(fmt.Sprintf("failed get snapshot %s", err)))
		return
	}
	snapshotIds := make([]string, len(snapshots))
	for i := range snapshots {
		snapshotIds[i] = snapshots[i].Id
	}
	self.StartSnapshotsDelete(ctx, snapshotIds)
}

func (self *SnapshotCleanupTask) OnInitFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Errorf("delete snapshots failed %s", data)
	self.OnInit(ctx, obj, data)
}

func (self *SnapshotCleanupTask) StartSnapshotsDelete(ctx context.Context, snapshotIds []string) {
	snapshotId := snapshotIds[0]
	snapshotIds = snapshotIds[1:]
	if len(snapshotIds) > 0 {
		self.Params.Set("snapshots", jsonutils.Marshal(snapshotIds))
	}
	self.SetStage("OnDeleteSnapshot", nil)

	iSnapshot, err := models.SnapshotManager.FetchById(snapshotId)
	if err != nil {
		self.OnDeleteSnapshotFailed(ctx, self.GetObject(), nil)
		return
	}
	snapshot := iSnapshot.(*models.SSnapshot)
	if snapshot.Status == compute.SNAPSHOT_DELETING {
		self.OnDeleteSnapshot(ctx, snapshot, nil)
		return
	}

	snapshot.SetModelManager(models.SnapshotManager, snapshot)
	err = snapshot.StartSnapshotDeleteTask(ctx, self.UserCred, false, self.GetId(), 0, 0)
	if err != nil {
		self.OnDeleteSnapshotFailed(ctx, self.GetObject(), nil)
	}
}

func (self *SnapshotCleanupTask) OnDeleteSnapshot(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	var snapshotIds = make([]string, 0)
	err := self.Params.Unmarshal(&snapshotIds, "snapshots")
	if err != nil {
		self.taskFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	if len(snapshotIds) > 0 {
		self.StartSnapshotsDelete(ctx, snapshotIds)
	} else {
		self.taskCompleted(ctx, nil)
	}
}

func (self *SnapshotCleanupTask) OnDeleteSnapshotFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	reason := fmt.Sprintf("snapshot delete faield %s", data)
	self.taskFailed(ctx, jsonutils.NewString(reason))
}
