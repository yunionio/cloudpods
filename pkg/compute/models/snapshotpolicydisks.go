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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SSnapshotPolicyDiskManager struct {
	db.SVirtualJointResourceBaseManager
	SSnapshotPolicyResourceBaseManager
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

	SSnapshotPolicyResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	SDiskResourceBase           `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	Status                      string `width:"36" charset:"ascii" nullable:"false" default:"init" list:"user" create:"optional"`
	NextSyncTime                time.Time
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

func (self *SSnapshotPolicyDisk) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (api.SnapshotPolicyDiskDetails, error) {
	return api.SnapshotPolicyDiskDetails{}, nil
}

func (manager *SSnapshotPolicyDiskManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.SnapshotPolicyDiskDetails {
	rows := make([]api.SnapshotPolicyDiskDetails, len(objs))

	virtRows := manager.SVirtualJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	snapIds := make([]string, len(rows))
	diskIds := make([]string, len(rows))

	for i := range rows {
		rows[i] = api.SnapshotPolicyDiskDetails{
			VirtualJointResourceBaseDetails: virtRows[i],
		}
		snapIds[i] = objs[i].(*SSnapshotPolicyDisk).SnapshotpolicyId
		diskIds[i] = objs[i].(*SSnapshotPolicyDisk).DiskId
	}

	snapIdMaps, err := db.FetchIdNameMap2(SnapshotPolicyManager, snapIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail for snapshot Ids %s", err)
		return rows
	}

	disks := make(map[string]SDisk)
	err = db.FetchStandaloneObjectsByIds(DiskManager, diskIds, &disks)
	if err != nil {
		log.Errorf("FetchStandaloneObjectsByIds for disks fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := snapIdMaps[snapIds[i]]; ok {
			rows[i].Snapshotpolicy = name
		}
		if disk, ok := disks[diskIds[i]]; ok {
			rows[i].Disk, _ = disk.GetExtraDetails(ctx, userCred, query, isList)
		}
	}

	return rows
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
		return nil, nil
	}
	return &ret[0], nil
}

func (sdm *SSnapshotPolicyDiskManager) InitializeData() error {
	diskQ := DiskManager.Query("id").SubQuery()
	sdQ := sdm.Query().NotIn("disk_id", diskQ)

	var sds []SSnapshotPolicyDisk
	err := db.FetchModelObjects(sdm, sdQ, &sds)
	if err != nil {
		return errors.Wrap(err, "unable to FetchModelObjects")
	}
	for i := range sds {
		sd := &sds[i]
		_, err := db.Update(sd, func() error {
			return sd.MarkDelete()
		})
		if err != nil {
			return err
		}
	}

	sds = make([]SSnapshotPolicyDisk, 0)
	q := sdm.Query().IsNullOrEmpty("next_sync_time")
	err = db.FetchModelObjects(sdm, q, &sds)
	if err != nil {
		return err
	}

	// fetch all snapshotpolicy
	spIdSet := sets.NewString()
	for i := range sds {
		spIdSet.Insert(sds[i].SnapshotpolicyId)
	}
	sps, err := SnapshotPolicyManager.FetchAllByIds(spIdSet.UnsortedList())
	if err != nil {
		return errors.Wrap(err, "FetchAllByIds")
	}
	spMap := make(map[string]*SSnapshotPolicy, len(sps))
	for i := range sps {
		spMap[sps[i].GetId()] = &sps[i]
	}
	now := time.Now()
	for i := range sds {
		sd := &sds[i]
		_, err := db.Update(sd, func() error {
			sd.NextSyncTime = spMap[sd.SnapshotpolicyId].ComputeNextSyncTime(now)
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update")
		}
	}
	return nil
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
	disksq := DiskManager.Query("id").IsFalse("pending_deleted").SubQuery()
	sdsq := m.Query().SubQuery()
	q := sdsq.Query().Join(disksq, sqlchemy.Equals(disksq.Field("id"),
		sdsq.Field("disk_id"))).Equals("snapshotpolicy_id", snapshotpolicyID)
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

	lockman.LockRawObject(ctx, "snapshot-policies", disk.Id)
	defer lockman.ReleaseRawObject(ctx, "snapshot-policies", disk.Id)

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
		sd.Status = api.SNAPSHOT_POLICY_DISK_READY
		err = m.TableSpec().Insert(ctx, &sd)
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

const ErrExistSD = errors.Error("snapshotpolicy disk has been exist")

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

	spd := SSnapshotPolicyDisk{}
	spd.SnapshotpolicyId = sp.GetId()
	spd.DiskId = disk.GetId()
	now := time.Now()
	spd.NextSyncTime = sp.ComputeNextSyncTime(now)
	spd.SetModelManager(self, &spd)

	lockman.LockJointObject(ctx, disk, sp)
	defer lockman.ReleaseJointObject(ctx, disk, sp)
	return &spd, self.TableSpec().Insert(ctx, &spd)

}
func (self *SSnapshotPolicyDiskManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	diskId, _ := data.GetString(self.GetMasterFieldName())
	snapshotPolicyId, _ := data.GetString(self.GetSlaveFieldName())
	disk := DiskManager.FetchDiskById(diskId)
	snapshotPolicy, err := SnapshotPolicyManager.FetchSnapshotPolicyById(snapshotPolicyId)
	if err != nil {
		return nil, err
	}
	err = disk.GetStorage().GetRegion().GetDriver().ValidateCreateSnapshopolicyDiskData(ctx, userCred, disk, snapshotPolicy)
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

func (sd *SSnapshotPolicyDisk) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	sp, err := SnapshotPolicyManager.FetchSnapshotPolicyById(sd.SnapshotpolicyId)
	if err != nil {
		return err
	}
	now := time.Now()
	sd.NextSyncTime = sp.ComputeNextSyncTime(now)
	return nil
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
	sd.SetStatus(userCred, api.SNAPSHOT_POLICY_DISK_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotPolicyCancelTask", disk, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrapf(err, "SnapshotPolicyCancelTask newTask error %s", err)
	} else {
		task.ScheduleRun(taskData)
	}
	return nil
}

func (manager *SSnapshotPolicyDiskManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SnapshotPolicyDiskListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualJointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SSnapshotPolicyResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SnapshotPolicyFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSnapshotPolicyResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SDiskResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DiskFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDiskResourceBaseManager.ListItemFilter")
	}

	if len(query.Status) > 0 {
		q = q.In("status", query.Status)
	}

	return q, nil
}

func (manager *SSnapshotPolicyDiskManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.SnapshotPolicyDiskListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualJointResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SSnapshotPolicyResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SnapshotPolicyFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSnapshotPolicyResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SDiskResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DiskFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDiskResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SSnapshotPolicyDiskManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SSnapshotPolicyResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SDiskResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SSnapshotPolicyDiskManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualJointResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualJointResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SSnapshotPolicyResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SSnapshotPolicyResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SSnapshotPolicyResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SDiskResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SDiskResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SDiskResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
