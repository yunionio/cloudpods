package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	// create by
	MANUAL = "manual"
	AUTO   = "auto"

	SNAPSHOT_CREATING = "creating"
	SNAPSHOT_FAILED   = "create_failed"
	SNAPSHOT_READY    = "ready"
	SNAPSHOT_DELETING = "deleting"
	SNAPSHOT_UNKNOWN  = "unknown"
)

type SSnapshotManager struct {
	db.SVirtualResourceBaseManager
}

type SSnapshot struct {
	db.SVirtualResourceBase
	SManagedResourceBase

	DiskId      string `width:"36" charset:"ascii" nullable:"true" create:"required" key_index:"true" list:"user"`
	StorageId   string `width:"36" charset:"ascii" nullable:"true" list:"admin"`
	CreatedBy   string `width:"36" charset:"ascii" nullable:"false" default:"manual" list:"admin"`
	Location    string `charset:"ascii" nullable:"true" list:"admin"`
	Size        int    `nullable:"false" list:"user"` // MB
	OutOfChain  bool   `nullable:"false" default:"false" index:"true" list:"admin"`
	FakeDeleted bool   `nullable:"false" default:"false" index:"true"`

	CloudregionId string `width:"36" charset:"ascii" nullable:"true" list:"user"`
}

var SnapshotManager *SSnapshotManager

func init() {
	SnapshotManager = &SSnapshotManager{SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SSnapshot{}, "snapshots_tbl", "snapshot", "snapshots")}
}

func ValidateSnapshotName(hypervisor, name string) error {
	if !('A' <= name[0] && name[0] <= 'Z' || 'a' <= name[0] && name[0] <= 'z') {
		return fmt.Errorf("Name must start with letter")
	}
	if len(name) < 2 || len(name) > 128 {
		return fmt.Errorf("Snapshot name length must within 2~128")
	}
	if hypervisor == HYPERVISOR_ALIYUN {
		if strings.HasPrefix(name, "auto") || strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
			return fmt.Errorf("Snapshot name can't start with auto, http:// or https://")
		}
	}
	return nil
}

func (self *SSnapshot) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (manager *SSnapshotManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	if jsonutils.QueryBoolean(query, "fake_deleted", false) {
		q = q.Equals("fake_deleted", true)
	} else {
		q = q.Equals("fake_deleted", false)
	}
	return q, nil
}

func (self *SSnapshot) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SSnapshot) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SSnapshot) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	disk, _ := self.GetDisk()
	if disk != nil {
		extra.Add(jsonutils.NewString(disk.DiskType), "disk_type")
		guests := disk.GetGuests()
		if len(guests) == 1 {
			extra.Add(jsonutils.NewString(guests[0].Id), "guest")
			extra.Add(jsonutils.NewString(guests[0].Status), "guest_status")
		}
	}
	if cloudprovider := self.GetCloudprovider(); cloudprovider != nil {
		extra.Add(jsonutils.NewString(cloudprovider.Provider), "provider")
	}
	return extra
}

func (self *SSnapshot) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SSnapshot) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSnapshot) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SSnapshot) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
}

func (self *SSnapshot) GetGuest() (*SGuest, error) {
	iDisk, err := DiskManager.FetchById(self.DiskId)
	if err != nil {
		return nil, err
	}
	disk := iDisk.(*SDisk)
	guests := disk.GetGuests()
	if len(guests) > 1 {
		return nil, fmt.Errorf("Snapshot disk attach mutil guest")
	} else if len(guests) == 1 {
		return &guests[0], nil
	} else {
		return nil, sql.ErrNoRows
	}
}

func (self *SSnapshot) GetDisk() (*SDisk, error) {
	iDisk, err := DiskManager.FetchById(self.DiskId)
	if err != nil {
		return nil, err
	}
	disk := iDisk.(*SDisk)
	return disk, nil
}

func (self *SSnapshot) GetHost() *SHost {
	iStorage, err := StorageManager.FetchById(self.StorageId)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	storage := iStorage.(*SStorage)
	return storage.GetMasterHost()
}

func (self *SSnapshotManager) GetDiskSnapshotsByCreate(diskId, createdBy string) []SSnapshot {
	dest := make([]SSnapshot, 0)
	q := self.Query().SubQuery()
	err := q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId),
		sqlchemy.Equals(q.Field("created_by"), createdBy),
		sqlchemy.Equals(q.Field("fake_deleted"), false))).All(&dest)
	if err != nil {
		log.Errorf("GetDiskSnapshots error: %s", err)
		return nil
	}
	for i := 0; i < len(dest); i++ {
		dest[i].SetModelManager(self)
	}
	return dest
}

func (self *SSnapshotManager) GetDiskSnapshots(diskId string) []SSnapshot {
	dest := make([]SSnapshot, 0)
	q := self.Query().SubQuery()
	err := q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId))).All(&dest)
	if err != nil {
		log.Errorf("GetDiskSnapshots error: %s", err)
		return nil
	}
	for i := 0; i < len(dest); i++ {
		dest[i].SetModelManager(self)
	}
	return dest
}

func (self *SSnapshotManager) GetDiskFirstSnapshot(diskId string) *SSnapshot {
	dest := &SSnapshot{}
	q := self.Query().SubQuery()
	err := q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId),
		sqlchemy.In(q.Field("status"), []string{SNAPSHOT_READY, SNAPSHOT_DELETING}),
		sqlchemy.Equals(q.Field("out_of_chain"), false))).Asc("created_at").First(dest)
	if err != nil {
		log.Errorf("Get Disk First snapshot error: %s", err.Error())
		return nil
	}
	dest.SetModelManager(self)
	return dest
}

func (self *SSnapshotManager) GetDiskSnapshotCount(diskId string) int {
	q := self.Query().SubQuery()
	return q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), diskId),
		sqlchemy.Equals(q.Field("fake_deleted"), false))).Count()
}

func (self *SSnapshotManager) CreateSnapshot(ctx context.Context, userCred mcclient.TokenCredential, createdBy, diskId, guestId, location, name string) (*SSnapshot, error) {
	iDisk, err := DiskManager.FetchById(diskId)
	if err != nil {
		return nil, err
	}
	disk := iDisk.(*SDisk)
	snapshot := &SSnapshot{}
	snapshot.SetModelManager(self)
	snapshot.ProjectId = userCred.GetProjectId()
	snapshot.DiskId = disk.Id
	snapshot.StorageId = disk.StorageId
	snapshot.Size = disk.DiskSize
	snapshot.Location = location
	snapshot.CreatedBy = createdBy
	snapshot.Name = name
	snapshot.Status = SNAPSHOT_CREATING
	err = SnapshotManager.TableSpec().Insert(snapshot)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (self *SSnapshot) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SSnapshot) StartSnapshotDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, reloadDisk bool, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Set("reload_disk", jsonutils.NewBool(reloadDisk))
	task, err := taskman.TaskManager.NewTask(ctx, "SnapshotDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf(err.Error())
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SSnapshot) ValidateDeleteCondition(ctx context.Context) error {
	return nil
}

func (self *SSnapshot) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if self.Status == SNAPSHOT_DELETING {
		return fmt.Errorf("Cannot delete snapshot in status %s", self.Status)
	}
	if self.Status == SNAPSHOT_UNKNOWN {
		return self.RealDelete(ctx, userCred)
	}
	if len(self.ExternalId) == 0 {
		if utils.IsInStringArray(self.Status, []string{SNAPSHOT_FAILED}) {
			return self.RealDelete(ctx, userCred)
		}
		if self.CreatedBy == MANUAL {
			if !self.FakeDeleted {
				return self.FakeDelete()
			} else {
				_, err := SnapshotManager.GetConvertSnapshot(self)
				if err != nil {
					return fmt.Errorf("Cannot delete snapshot: %s, disk need at least one of snapshot as backing file", err.Error())
				}
				return self.StartSnapshotDeleteTask(ctx, userCred, false, "")
			}
		} else {
			return fmt.Errorf("Cannot delete snapshot created by %s", self.CreatedBy)
		}
	} else {
		return self.StartSnapshotDeleteTask(ctx, userCred, false, "")
	}
}

func (self *SSnapshot) AllowPerformDeleted(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SSnapshot) PerformDeleted(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	self.GetModelManager().TableSpec().Update(self, func() error {
		self.OutOfChain = true
		return nil
	})
	err := self.StartSnapshotDeleteTask(ctx, userCred, true, "")
	return nil, err
}

func (self *SSnapshotManager) AllowGetPropertyMaxCount(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSnapshotManager) GetPropertyMaxCount(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	ret.Set("max_count", jsonutils.NewInt(int64(options.Options.DefaultMaxSnapshotCount)))
	return ret, nil
}

func (self *SSnapshotManager) GetConvertSnapshot(deleteSnapshot *SSnapshot) (*SSnapshot, error) {
	dest := &SSnapshot{}
	q := self.Query().SubQuery()
	err := q.Query().Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), deleteSnapshot.DiskId),
		sqlchemy.In(q.Field("status"), []string{SNAPSHOT_READY, SNAPSHOT_DELETING}),
		sqlchemy.Equals(q.Field("out_of_chain"), false),
		sqlchemy.GT(q.Field("created_at"), deleteSnapshot.CreatedAt))).
		Asc("created_at").First(dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func (self *SSnapshotManager) AllowPerformDeleteDiskSnapshots(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsAdmin()
}

func (self *SSnapshotManager) PerformDeleteDiskSnapshots(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	diskId, err := data.GetString("disk_id")
	if err != nil {
		return nil, err
	}
	disk, err := DiskManager.FetchById(diskId)
	if disk != nil {
		return nil, httperrors.NewBadRequestError("Cannot Delete disk %s snapshots, disk exist", diskId)
	}
	snapshots := self.GetDiskSnapshots(diskId)
	if snapshots == nil || len(snapshots) == 0 {
		return nil, httperrors.NewNotFoundError("Disk %s dose not have snapshot", diskId)
	}
	for i := 0; i < len(snapshots); i++ {
		if snapshots[i].CreatedBy == MANUAL && snapshots[i].FakeDeleted == false {
			return nil, httperrors.NewBadRequestError("Can not delete disk snapshots, have manual snapshot")
		}
	}
	err = snapshots[0].StartSnapshotsDeleteTask(ctx, userCred, "")
	return nil, err
}

func (self *SSnapshot) StartSnapshotsDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BatchSnapshotsDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		log.Errorf(err.Error())
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SSnapshot) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SSnapshot) FakeDelete() error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.FakeDeleted = true
		return nil
	})
	return err
}

func (self *SSnapshot) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func totalSnapshotCount(projectId string) int {
	q := SnapshotManager.Query()
	count := q.Equals("tenant_id", projectId).Equals("fake_deleted", false).Count()
	return count
}

// Only sync snapshot status
func (self *SSnapshot) SyncWithCloudSnapshot(userCred mcclient.TokenCredential, ext cloudprovider.ICloudSnapshot, projectId string, projectSync bool) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Status = ext.GetStatus()
		if projectSync && len(projectId) > 0 {
			self.ProjectId = projectId
		}
		return nil
	})
	if err != nil {
		log.Errorf("SyncWithCloudSnapshot fail %s", err)
	}
	return err
}

func (manager *SSnapshotManager) newFromCloudSnapshot(userCred mcclient.TokenCredential, extSnapshot cloudprovider.ICloudSnapshot, region *SCloudregion, projectId string) (*SSnapshot, error) {
	snapshot := SSnapshot{}
	snapshot.SetModelManager(manager)

	snapshot.Name = extSnapshot.GetName()
	snapshot.Status = extSnapshot.GetStatus()
	snapshot.ExternalId = extSnapshot.GetGlobalId()
	if len(extSnapshot.GetDiskId()) > 0 {
		disk, err := DiskManager.FetchByExternalId(extSnapshot.GetDiskId())
		if err != nil {
			log.Errorf("snapshot %s missing disk?", snapshot.Name)
		} else {
			snapshot.DiskId = disk.GetId()
		}
	}

	snapshot.Size = int(extSnapshot.GetSize()) * 1024
	snapshot.ManagerId = extSnapshot.GetManagerId()
	snapshot.CloudregionId = region.Id

	snapshot.ProjectId = userCred.GetProjectId()
	if len(projectId) > 0 {
		snapshot.ProjectId = projectId
	}
	err := manager.TableSpec().Insert(&snapshot)
	if err != nil {
		log.Errorf("newFromCloudEip fail %s", err)
		return nil, err
	}
	return &snapshot, nil
}

func (manager *SSnapshotManager) getProviderSnapshotsByRegion(region *SCloudregion, provider *SCloudprovider) ([]SSnapshot, error) {
	if region == nil || provider == nil {
		return nil, fmt.Errorf("Region is nil or provider is nil")
	}
	snapshots := make([]SSnapshot, 0)
	q := manager.Query().Equals("cloudregion_id", region.Id).Equals("manager_id", provider.Id).NotEquals("status", SNAPSHOT_UNKNOWN)
	err := db.FetchModelObjects(manager, q, &snapshots)
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (manager *SSnapshotManager) SyncSnapshots(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, snapshots []cloudprovider.ICloudSnapshot, projectId string, projectSync bool) compare.SyncResult {
	syncResult := compare.SyncResult{}
	dbSnapshots, err := manager.getProviderSnapshotsByRegion(region, provider)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	removed := make([]SSnapshot, 0)
	commondb := make([]SSnapshot, 0)
	commonext := make([]cloudprovider.ICloudSnapshot, 0)
	added := make([]cloudprovider.ICloudSnapshot, 0)

	err = compare.CompareSets(dbSnapshots, snapshots, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	for i := 0; i < len(removed); i += 1 {
		err = removed[i].SetStatus(userCred, SNAPSHOT_UNKNOWN, "sync to delete")
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudSnapshot(userCred, commonext[i], projectId, projectSync)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		_, err := manager.newFromCloudSnapshot(userCred, added[i], region, projectId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}

func (self *SSnapshot) GetRegion() *SCloudregion {
	return CloudregionManager.FetchRegionById(self.CloudregionId)
}

func (self *SSnapshot) GetISnapshotRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, err
	}

	region := self.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("fail to find region for snapshot")
	}
	return provider.GetIRegionById(region.GetExternalId())
}
