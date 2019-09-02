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
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSnapshotPolicyDiskManager struct {
	db.SVirtualJointResourceBaseManager
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
				"snapshotpolicydisks_tbl",
				"snapshotpolicydisk",
				"snapshotpolicydisks",
				DiskManager,
				SnapshotPolicyManager,
			),
		}
		SnapshotPolicyDiskManager.SetVirtualObject(SnapshotPolicyDiskManager)
	})

}

type SSnapshotPolicyDisk struct {
	db.SVirtualJointResourceBase

	SnapshotpolicyId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	DiskId           string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	Status           string `width:"36" charset:"ascii" nullable:"false" default:"init" list:"user" create:"optional"`
}

func (sd *SSnapshotPolicyDisk) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	if sd.Status == status {
		return nil
	}
	oldStatus := sd.Status
	_, err := db.Update(sd, func() error {
		sd.Status = status
		return nil
	})
	if err != nil {
		return err
	}
	if userCred != nil {
		notes := fmt.Sprintf("%s=>%s", oldStatus, status)
		if len(reason) > 0 {
			notes = fmt.Sprintf("%s: %s", notes, reason)
		}
		db.OpsLog.LogEvent(sd, db.ACT_UPDATE_STATUS, notes, userCred)
	}
	return nil
}

func (self *SSnapshotPolicyDisk) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) *jsonutils.JSONDict {

	ret, _ := self.getMoreDetails(ctx, userCred, query)
	return ret
}

func (self *SSnapshotPolicyDisk) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {

	return self.getMoreDetails(ctx, userCred, query)
}

func (self *SSnapshotPolicyDisk) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {

	disk := DiskManager.FetchDiskById(self.DiskId)
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.Marshal(disk), "disk")
	return ret, nil
}

// ==================================================== fetch ==========================================================

func (m *SSnapshotPolicyDiskManager) FetchBySnapshotPolicyDisk(spId, diskId string) (*SSnapshotPolicyDisk, error) {
	q := m.Query().Equals("snapshotpolicy_id", spId).Equals("disk_id", diskId)
	ret := make([]SSnapshotPolicyDisk, 0, 1)
	err := db.FetchModelObjects(m, q, &ret)
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, fmt.Errorf("Not Found")
	}
	return &ret[0], nil
}

func (m *SSnapshotPolicyDiskManager) FetchAllByDiskID(ctx context.Context, userCred mcclient.TokenCredential,
	diskID string) ([]SSnapshotPolicyDisk, error) {

	return m.fetchAll(ctx, userCred, m.GetMasterFieldName(), diskID)
}

func (m *SSnapshotPolicyDiskManager) FetchAllBySnapshotpolicyID(ctx context.Context, userCred mcclient.TokenCredential,
	snapshotPolicyID string) ([]SSnapshotPolicyDisk, error) {

	return m.fetchAll(ctx, userCred, m.GetSlaveFieldName(), snapshotPolicyID)
}

func (m *SSnapshotPolicyDiskManager) FetchDiskCountBySPID(snapshotpolicyID string) (int, error) {

	q := m.Query().Equals("snapshotpolicy_id", snapshotpolicyID)
	return q.CountWithError()
}

func (m *SSnapshotPolicyDiskManager) fetchAll(ctx context.Context, userCred mcclient.TokenCredential,
	fieldName string, fieldValue string) ([]SSnapshotPolicyDisk, error) {

	q := m.Query()
	q.Equals(fieldName, fieldValue)
	ret := make([]SSnapshotPolicyDisk, 0)
	err := db.FetchModelObjects(m, q, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// ==================================================== sync ===========================================================

// SyncDetachByDisk will detach all snapshot policies fo cloudRelations
// if cloudRelations is nil, it will detach all shapshot policies which has been attached to disk.
func (m *SSnapshotPolicyDiskManager) SyncDetachByDisk(ctx context.Context, userCred mcclient.TokenCredential,
	cloudRelations []SSnapshotPolicyDisk, disk *SDisk) error {
	snapshotPolicyDisks := cloudRelations
	var err error
	if snapshotPolicyDisks == nil {
		snapshotPolicyDisks, err = m.FetchAllByDiskID(ctx, userCred, disk.GetId())
		if err != nil {
			return errors.Wrapf(err, "Fetach allsnapshotpolicy of disk %s in database", disk.GetId())
		}
	}
	failResult := make([]string, 0, 1)
	for i := range snapshotPolicyDisks {
		err = snapshotPolicyDisks[i].DetachByDisk(ctx, userCred, disk)
		if err != nil {
			failResult = append(failResult, snapshotPolicyDisks[i].GetId())
		}
	}
	if len(failResult) != 0 {
		errInfo := "detach failed which IDs are: "
		return errors.Error(errInfo + strings.Join(failResult, ", "))
	}
	return nil
}

// SyncDetachBySnapshotpolicy detach all sn
func (m *SSnapshotPolicyDiskManager) SyncDetachBySnapshotpolicy(ctx context.Context,
	userCred mcclient.TokenCredential, cloudRelations []SSnapshotPolicyDisk, snapshotPolicy *SSnapshotPolicy) error {

	snapshotPolicyDisks := cloudRelations
	var err error
	if snapshotPolicyDisks == nil {
		snapshotPolicyDisks, err = m.FetchAllBySnapshotpolicyID(ctx, userCred, snapshotPolicy.GetId())
		if err != nil {
			return errors.Wrapf(err, "Fetach all bysnapshotpolicy %s in database", snapshotPolicy.GetId())
		}
	}
	failResult := make([]string, 0, 1)
	for i := range snapshotPolicyDisks {
		err = snapshotPolicyDisks[i].DetachBySnapshotpolicy(ctx, userCred, snapshotPolicy)
		if err != nil {
			failResult = append(failResult, snapshotPolicyDisks[i].GetId())
		}
	}
	if len(failResult) != 0 {
		errInfo := "detach failed which IDs are "
		return errors.Error(errInfo + strings.Join(failResult, ", "))
	}
	return nil
}

func (m *SSnapshotPolicyDiskManager) SyncByDisk(ctx context.Context, userCred mcclient.TokenCredential,
	extSnapshotpolicies []string, syncOwnerID mcclient.IIdentityProvider, disk *SDisk, storage *SStorage) error {

	sds, err := m.FetchAllByDiskID(ctx, userCred, disk.GetId())
	if err != nil {
		return errors.Wrapf(err, "Fetach allsnapshotpolicy of disk %s in database", disk.GetId())
	}

	//fetch snapshotPolicy Cache to find the snapshotpolicyID corresponding to extSnapshotpolicyID
	spCaches, err := SnapshotPolicyCacheManager.FetchAllByExtIds(extSnapshotpolicies, storage.GetRegion().GetId(),
		storage.ManagerId)
	if err != nil {
		return errors.Wrapf(err, "fetachsnapshotpolicy caches failed")
	}
	cloudRelationsSet := make(map[string]struct{})
	for i := range spCaches {
		cloudRelationsSet[spCaches[i].SnapshotpolicyId] = struct{}{}
	}

	removed := make([]SSnapshotPolicyDisk, 0, 1)
	added := make([]string, 0, 1)
	for i := range sds {
		if _, ok := cloudRelationsSet[sds[i].SnapshotpolicyId]; !ok {
			removed = append(removed, sds[i])
		}
		delete(cloudRelationsSet, sds[i].SnapshotpolicyId)
	}
	for k := range cloudRelationsSet {
		added = append(added, k)
	}
	err = m.SyncDetachByDisk(ctx, userCred, removed, disk)
	if err != nil {
		return err
	}
	err = m.SyncAttachDisk(ctx, userCred, added, syncOwnerID, disk)
	if err != nil {
		return err
	}
	return nil
}

func (m *SSnapshotPolicyDiskManager) SyncAttachDisk(ctx context.Context, userCred mcclient.TokenCredential,
	Snapshotpolicies []string, syncOwnerID mcclient.IIdentityProvider, disk *SDisk) error {

	lockman.LockClass(ctx, m, db.GetLockClassKey(m, syncOwnerID))
	defer lockman.ReleaseClass(ctx, m, db.GetLockClassKey(m, syncOwnerID))

	failRecord := make([]string, 0, 1)
	for _, spId := range Snapshotpolicies {
		snapshotpolicyDisk, err := db.FetchJointByIds(m, disk.GetId(), spId, jsonutils.JSONNull)
		if err != nil && err != sql.ErrNoRows {
			failRecord = append(failRecord,
				fmt.Sprintf("Get SnapshotpolicyDisk whose diskid %s snapshotpolicyid %s failed",
					disk.GetId(), spId))
			continue
		}
		if snapshotpolicyDisk != nil {
			continue
		}

		sd := SSnapshotPolicyDisk{}
		sd.DiskId = disk.GetId()
		sd.SnapshotpolicyId = spId
		sd.Status = compute.SNAPSHOT_POLICY_DISK_READY
		err = m.TableSpec().Insert(&sd)
		if err != nil {
			failRecord = append(failRecord, fmt.Sprintf("attachsnapshotpolicy %s to disk %s failed",
				spId, disk.GetId()))
			continue
		}
	}
	if len(failRecord) == 0 {
		return nil
	}
	buf := bytes.NewBufferString("sync attach extSnapshotpolicies to disk ")
	buf.WriteString(disk.GetId())
	buf.WriteString("failed because that ")
	for i := range failRecord {
		buf.WriteString(failRecord[i])
		buf.WriteString(", ")
	}
	buf.Truncate(buf.Len() - 2)
	return errors.Error(buf.String())
}

//
func (m *SSnapshotPolicyDiskManager) SyncAttachDiskExt(ctx context.Context, userCred mcclient.TokenCredential,
	extSnapshotpolicies []string, syncOwnerID mcclient.IIdentityProvider, disk *SDisk, storage *SStorage) error {

	//fetch snapshotPolicy Cache to find the snapshotpolicyID corresponding to extSnapshotpolicyID
	spCaches, err := SnapshotPolicyCacheManager.FetchAllByExtIds(extSnapshotpolicies, storage.GetRegion().GetId(),
		storage.ManagerId)
	if err != nil {
		return errors.Wrapf(err, "fetachsnapshotpolicy caches failed")
	}

	snapshotPolicie := make([]string, 0, 1)
	for i := range spCaches {
		snapshotPolicie = append(snapshotPolicie, spCaches[i].SnapshotpolicyId)
	}

	return m.SyncAttachDisk(ctx, userCred, snapshotPolicie, syncOwnerID, disk)
}

// ==================================================== detach =========================================================

func (sd *SSnapshotPolicyDisk) RealDetach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, sd)
}

func (sd *SSnapshotPolicyDisk) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (sd *SSnapshotPolicyDisk) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

// syncDetach should lock before
func (sd *SSnapshotPolicyDisk) DetachByDisk(ctx context.Context, userCred mcclient.TokenCredential, disk *SDisk) error {
	snapshotPolicy := SSnapshotPolicy{}
	snapshotPolicy.Id = sd.SnapshotpolicyId
	snapshotPolicy.SetModelManager(SnapshotPolicyManager, &snapshotPolicy)
	lockman.LockJointObject(ctx, disk, &snapshotPolicy)
	defer lockman.ReleaseJointObject(ctx, disk, &snapshotPolicy)
	// todo call real Detach
	return sd.RealDetach(ctx, userCred)
}

func (sd *SSnapshotPolicyDisk) DetachBySnapshotpolicy(ctx context.Context, userCred mcclient.TokenCredential,
	snapshotPolicy *SSnapshotPolicy) error {

	disk := SDisk{}
	disk.Id = sd.DiskId
	disk.SetModelManager(DiskManager, &disk)
	lockman.LockJointObject(ctx, &disk, snapshotPolicy)
	defer lockman.ReleaseJointObject(ctx, &disk, snapshotPolicy)
	return sd.RealDetach(ctx, userCred)
}

// ==================================================== create =========================================================

var ErrExistSD = fmt.Errorf("snapshotpolicy disk has been exist")

func (self *SSnapshotPolicyDiskManager) newSnapshotpolicyDisk(ctx context.Context, userCred mcclient.TokenCredential,
	sp *SSnapshotPolicy, disk *SDisk) (*SSnapshotPolicyDisk, error) {

	q := self.Query().Equals("snapshotpolicy_id", sp.GetId()).Equals("disk_id", disk.GetId())
	count, err := q.CountWithError()
	if err != nil {
		return nil, nil
	}
	if count > 0 {
		spd := SSnapshotPolicyDisk{}
		q.First(&spd)
		spd.SetModelManager(self, &spd)
		return &spd, ErrExistSD
	}

	spd := SSnapshotPolicyDisk{SnapshotpolicyId: sp.GetId(), DiskId: disk.GetId()}
	spd.SetModelManager(self, &spd)

	lockman.LockJointObject(ctx, disk, sp)
	defer lockman.ReleaseJointObject(ctx, disk, sp)
	return &spd, self.TableSpec().Insert(&spd)

}
func (self *SSnapshotPolicyDiskManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	diskId, _ := data.GetString(self.GetMasterFieldName())
	snapshotPolicyId, _ := data.GetString(self.GetSlaveFieldName())
	disk := DiskManager.FetchDiskById(diskId)
	snapshotPolicy := SnapshotPolicyManager.FetchSnapshotPolicyById(snapshotPolicyId)
	err := disk.GetStorage().GetRegion().GetDriver().ValidateCreateSnapshopolicyDiskData(ctx, userCred, disk, snapshotPolicy)
	if err != nil {
		return nil, err
	}
	//to control that one disk should only bind one snapshot policy
	spds, err := SnapshotPolicyDiskManager.FetchAllByDiskID(ctx, userCred, diskId)
	if err != nil {
		return nil, err
	}
	if len(spds) > 1 {
		return nil, httperrors.NewInputParameterError("disk %s has too many snapshot policy attached", diskId)
	}
	if len(spds) == 1 {
		data.Add(jsonutils.NewString(spds[0].SnapshotpolicyId), "need_detach")
	}
	// I don't want to request to database again behind
	data.Add(jsonutils.Marshal(snapshotPolicy), "snapshotPolicy")
	data.Add(jsonutils.Marshal(disk), "disk")
	return data, nil
}

func (sd *SSnapshotPolicyDisk) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.
	IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {

	taskdata := data.(*jsonutils.JSONDict)
	taskdata.Add(jsonutils.Marshal(sd), "snapshotPolicyDisk")
	disk := &SDisk{}
	data.Unmarshal(disk, "disk")
	disk.SetModelManager(DiskManager, disk)
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyApplyTask", disk, userCred, nil, "", "", nil)
	if err != nil {
		log.Errorf("SnapshotPolicyApplyTask newTask error %s", err)
	} else {
		task.ScheduleRun(taskdata)
	}
}

// ==================================================== delete =========================================================

func (sd *SSnapshotPolicyDisk) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) error {
	diskID := sd.DiskId
	model, err := DiskManager.FetchById(diskID)
	if err != nil {
		return errors.Wrapf(err, "Fetch disk by ID %s failed", diskID)
	}
	disk := model.(*SDisk)
	snapshotPolicyID := sd.SnapshotpolicyId
	taskData := jsonutils.NewDict()
	taskData.Add(jsonutils.NewString(snapshotPolicyID), "snapshot_policy_id")
	taskData.Add(jsonutils.Marshal(sd), "snapshotPolicyDisk")
	sd.SetStatus(userCred, compute.SNAPSHOT_POLICY_DISK_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyCancelTask", disk, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "SnapshotPolicyCancelTask newTask error %s", err)
	} else {
		task.ScheduleRun(taskData)
	}
	return nil
}
