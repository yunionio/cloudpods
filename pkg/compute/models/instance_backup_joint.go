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
		InstanceBackupJointManager = &SInstanceBackupJointManager{
			SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
				SInstanceBackupJoint{},
				"instancebackupjoints_tbl",
				"instancebackupjoint",
				"instancebackupjoints",
				InstanceBackupManager,
				DiskBackupManager,
			),
		}
		InstanceBackupJointManager.SetVirtualObject(InstanceBackupJointManager)
	})
}

// +onecloud:swagger-gen-ignore
type SInstanceBackupJoint struct {
	db.SVirtualJointResourceBase

	InstanceBackupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	DiskBackupId     string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	DiskIndex        int8   `nullable:"false" default:"0" list:"user" create:"required"`
}

// +onecloud:swagger-gen-ignore
type SInstanceBackupJointManager struct {
	db.SVirtualJointResourceBaseManager
}

func (manager *SInstanceBackupJointManager) GetMasterFieldName() string {
	return "instance_backup_id"
}

func (manager *SInstanceBackupJointManager) GetSlaveFieldName() string {
	return "disk_backup_id"
}

var InstanceBackupJointManager *SInstanceBackupJointManager

func (manager *SInstanceBackupJointManager) CreateJoint(ctx context.Context, instanceBackupId, backupId string, diskIndex int8) error {
	instanceBackupJoint := &SInstanceBackupJoint{}
	instanceBackupJoint.SetModelManager(manager, instanceBackupJoint)

	instanceBackupJoint.InstanceBackupId = instanceBackupId
	instanceBackupJoint.DiskBackupId = backupId
	instanceBackupJoint.DiskIndex = diskIndex
	return manager.TableSpec().Insert(ctx, instanceBackupJoint)
}

func (manager *SInstanceBackupJointManager) IsSubBackup(backupId string) (bool, error) {
	count, err := manager.Query().Equals("disk_backup_id", backupId).CountWithError()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (self *SInstanceBackupJoint) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}
