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
	"fmt"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
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

func (manager *SSnapshotPolicyDiskManager) InitializeData() error {
	disks := DiskManager.Query("id").SubQuery()
	policies := SnapshotPolicyManager.Query("id").SubQuery()
	q := manager.Query().In("disk_id", disks).In("snapshotpolicy_id", policies)

	pds := []SSnapshotPolicyDisk{}
	err := db.FetchModelObjects(manager, q, &pds)
	if err != nil {
		return err
	}
	for i := range pds {
		pd := &pds[i]
		migrateData := &SSnapshotPolicyResource{
			SnapshotpolicyId: pd.SnapshotpolicyId,
			ResourceId:       pd.DiskId,
			ResourceType:     api.SNAPSHOT_POLICY_TYPE_DISK,
		}
		migrateData.SetModelManager(SnapshotPolicyResourceManager, migrateData)
		cnt, err := SnapshotPolicyResourceManager.Query().
			Equals("resource_type", api.SNAPSHOT_POLICY_TYPE_DISK).
			Equals("resource_id", pd.DiskId).
			Equals("snapshotpolicy_id", pd.SnapshotpolicyId).CountWithError()
		if err != nil {
			return errors.Wrapf(err, "Count")
		}
		if cnt == 0 {
			err = SnapshotPolicyResourceManager.TableSpec().Insert(context.Background(), migrateData)
			if err != nil {
				return errors.Wrapf(err, "Insert %s", migrateData.Keyword())
			}
		}
		err = db.Purge(manager, "row_id", []string{fmt.Sprintf("%d", pd.RowId)}, true)
		if err != nil {
			return errors.Wrapf(err, "Purge %d", pd.RowId)
		}
	}
	return nil
}
