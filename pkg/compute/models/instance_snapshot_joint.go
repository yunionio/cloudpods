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

func init() {
	db.InitManager(func() {
		InstanceSnapshotJointManager = &SInstanceSnapshotJointManager{
			SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
				SInstanceSnapshotJoint{},
				"instancesnapshotjoints_tbl",
				"instancesnapshotjoint",
				"instancesnapshotjoints",
				InstanceSnapshotManager,
				SnapshotManager,
			),
		}
		InstanceSnapshotJointManager.SetVirtualObject(InstanceSnapshotJointManager)
	})
}

// +onecloud:swagger-gen-ignore
type SInstanceSnapshotJoint struct {
	db.SVirtualJointResourceBase

	InstanceSnapshotId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	SnapshotId         string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	DiskIndex          int8   `nullable:"false" default:"0" list:"user" create:"required"`
}

// +onecloud:swagger-gen-ignore
type SInstanceSnapshotJointManager struct {
	db.SVirtualJointResourceBaseManager
}

func (manager *SInstanceSnapshotJointManager) GetMasterFieldName() string {
	return "instance_snapshot_id"
}

func (manager *SInstanceSnapshotJointManager) GetSlaveFieldName() string {
	return "snapshot_id"
}

var InstanceSnapshotJointManager *SInstanceSnapshotJointManager

func (manager *SInstanceSnapshotJointManager) CreateJoint(ctx context.Context, instanceSnapshotId, snapshotId string, diskIndex int8) error {
	instanceSnapshotJoint := &SInstanceSnapshotJoint{}
	instanceSnapshotJoint.SetModelManager(manager, instanceSnapshotJoint)

	instanceSnapshotJoint.InstanceSnapshotId = instanceSnapshotId
	instanceSnapshotJoint.SnapshotId = snapshotId
	instanceSnapshotJoint.DiskIndex = diskIndex
	return manager.TableSpec().Insert(ctx, instanceSnapshotJoint)
}

func (manager *SInstanceSnapshotJointManager) IsSubSnapshot(snapshotId string) (bool, error) {
	count, err := manager.Query().Equals("snapshot_id", snapshotId).CountWithError()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (self *SInstanceSnapshotJoint) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}
