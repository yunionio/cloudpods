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
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type SSnapshotPolicyDiskManager struct {
	db.SVirtualJointResourceBaseManager
	SDiskResourceBaseManager
}

func (m *SSnapshotPolicyDiskManager) GetMasterFieldName() string {
	return "disk_id"
}

func (m *SSnapshotPolicyDiskManager) GetSlaveFieldName() string {
	return "snapshotpolicy_id"
}

var SnapshotPolicyDiskManager *SSnapshotPolicyDiskManager

func init() {
	db.InitManager(func() {
		SnapshotPolicyDiskManager = &SSnapshotPolicyDiskManager{
			SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
				SSnapshotPolicyDisk{},
				"snapshot_policy_disks_tbl",
				"snapshot_policy_disk",
				"snapshot_policy_disks",
				DiskManager,
				SnapshotPolicyManager,
			),
		}
		SnapshotPolicyDiskManager.SetVirtualObject(SnapshotPolicyDiskManager)
	})

}

type SSnapshotPolicyDisk struct {
	db.SVirtualJointResourceBase

	SnapshotpolicyId  string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	SDiskResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func (self *SSnapshotPolicyDisk) GetDisk() (*SDisk, error) {
	disk, err := DiskManager.FetchById(self.DiskId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", self.DiskId)
	}
	return disk.(*SDisk), nil
}

func (self *SSnapshotPolicyDisk) GetSnapshotPolicy() (*SSnapshotPolicy, error) {
	policy, err := SnapshotPolicyManager.FetchById(self.SnapshotpolicyId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", self.SnapshotpolicyId)
	}
	return policy.(*SSnapshotPolicy), nil
}

func (man *SSnapshotPolicyDiskManager) RemoveByDisk(id string) error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where disk_id = ?",
			man.TableSpec().Name(),
		), id,
	)
	return err
}

func (man *SSnapshotPolicyDiskManager) RemoveBySnapshotpolicy(id string) error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where snapshotpolicy_id = ?",
			man.TableSpec().Name(),
		), id,
	)
	return err
}
