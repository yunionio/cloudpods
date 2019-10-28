package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

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

type SInstanceSnapshotJoint struct {
	db.SVirtualJointResourceBase

	InstanceSnapshotId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	SnapshotId         string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	DiskIndex          int8   `nullable:"false" default:"0" list:"user" create:"required"`
}

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

func (manager *SInstanceSnapshotJointManager) CreateJoint(instanceSnapshotId, snapshotId string, diskIndex int8) error {
	instanceSnapshotJoint := &SInstanceSnapshotJoint{}
	instanceSnapshotJoint.SetModelManager(manager, instanceSnapshotJoint)

	instanceSnapshotJoint.InstanceSnapshotId = instanceSnapshotId
	instanceSnapshotJoint.SnapshotId = snapshotId
	instanceSnapshotJoint.DiskIndex = diskIndex
	return manager.TableSpec().Insert(instanceSnapshotJoint)
}

func (manager *SInstanceSnapshotJointManager) IsSubSnapshot(snapshotId string) (bool, error) {
	count, err := manager.Query().Equals("snapshot_id", snapshotId).CountWithError()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
