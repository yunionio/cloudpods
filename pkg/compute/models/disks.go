package models

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/billing"
)

const (
	DISK_INIT                = "init"
	DISK_REBUILD             = "rebuild"
	DISK_ALLOC_FAILED        = "alloc_failed"
	DISK_STARTALLOC          = "start_alloc"
	DISK_BACKUP_STARTALLOC   = "backup_start_alloc"
	DISK_BACKUP_ALLOC_FAILED = "backup_alloc_failed"
	DISK_ALLOCATING          = "allocating"
	DISK_READY               = "ready"
	DISK_RESET               = "reset"
	DISK_DEALLOC             = "deallocating"
	DISK_DEALLOC_FAILED      = "dealloc_failed"
	DISK_UNKNOWN             = "unknown"
	DISK_DETACHING           = "detaching"
	DISK_ATTACHING           = "attaching"

	DISK_START_SAVE = "start_save"
	DISK_SAVING     = "saving"

	DISK_START_RESIZE = "start_resize"
	DISK_RESIZING     = "resizing"

	DISK_START_MIGRATE = "start_migrate"
	DISK_POST_MIGRATE  = "post_migrate"
	DISK_MIGRATING     = "migrating"

	DISK_START_SNAPSHOT = "start_snapshot"
	DISK_SNAPSHOTING    = "snapshoting"

	DISK_TYPE_SYS    = "sys"
	DISK_TYPE_SWAP   = "swap"
	DISK_TYPE_DATA   = "data"
	DISK_TYPE_VOLUME = "volume"

	DISK_BACKING_IMAGE = "image"
)

type SDiskManager struct {
	db.SSharableVirtualResourceBaseManager
}

var DiskManager *SDiskManager

func init() {
	DiskManager = &SDiskManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SDisk{},
			"disks_tbl",
			"disk",
			"disks",
		),
	}
}

type SDisk struct {
	db.SSharableVirtualResourceBase

	SBillingResourceBase

	DiskFormat string `width:"32" charset:"ascii" nullable:"false" default:"qcow2" list:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=False, default='qcow2')
	DiskSize   int    `nullable:"false" list:"user"`                                            // Column(Integer, nullable=False) # in MB
	AccessPath string `width:"256" charset:"ascii" nullable:"true" get:"user"`                  // = Column(VARCHAR(256, charset='ascii'), nullable=True)

	AutoDelete bool `nullable:"false" default:"false" get:"user" update:"user"` // Column(Boolean, nullable=False, default=False)

	StorageId       string `width:"128" charset:"ascii" nullable:"true" list:"admin" create:"required"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)
	BackupStorageId string `width:"128" charset:"ascii" nullable:"true" list:"admin"`

	// # backing template id and type
	TemplateId string `width:"256" charset:"ascii" nullable:"true" list:"user"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=True)
	// backing snapshot id
	SnapshotId string `width:"256" charset:"ascii" nullable:"true" list:"user"`

	// # file system
	FsFormat string `width:"32" charset:"ascii" nullable:"true" list:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	// # disk type, OS, SWAP, DAT, VOLUME
	DiskType string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"admin"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	// # is persistent
	Nonpersistent bool `default:"false" list:"user"` // Column(Boolean, default=False)
	AutoSnapshot  bool `default:"false" nullable:"true" get:"user" update:"user"`
}

func (manager *SDiskManager) GetContextManager() []db.IModelManager {
	return []db.IModelManager{StorageManager}
}

func (manager *SDiskManager) FetchDiskById(diskId string) *SDisk {
	disk, err := manager.FetchById(diskId)
	if err != nil {
		log.Errorf("FetchById fail %s", err)
		return nil
	}
	return disk.(*SDisk)
}

func (manager *SDiskManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	queryDict, ok := query.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("invalid querystring format")
	}

	var err error
	storages := StorageManager.Query().SubQuery()
	q, err = managedResourceFilterByAccount(q, query, "storage_id", func() *sqlchemy.SQuery {
		return storages.Query(storages.Field("id"))
	})
	if err != nil {
		return nil, err
	}
	q = managedResourceFilterByCloudType(q, query, "storage_id", func() *sqlchemy.SQuery {
		return storages.Query(storages.Field("id"))
	})

	billingTypeStr, _ := queryDict.GetString("billing_type")
	if len(billingTypeStr) > 0 {
		if billingTypeStr == BILLING_TYPE_POSTPAID {
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.IsNullOrEmpty(q.Field("billing_type")),
					sqlchemy.Equals(q.Field("billing_type"), billingTypeStr),
				),
			)
		} else {
			q = q.Equals("billing_type", billingTypeStr)
		}
		queryDict.Remove("billing_type")
	}

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	if query.Contains("unused") {
		guestdisks := GuestdiskManager.Query().SubQuery()
		sq := guestdisks.Query(guestdisks.Field("disk_id"))
		if jsonutils.QueryBoolean(query, "unused", false) {
			q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
		} else {
			q = q.Filter(sqlchemy.In(q.Field("id"), sq))
		}
	}

	if jsonutils.QueryBoolean(query, "share", false) {
		sq := storages.Query(storages.Field("id")).Filter(sqlchemy.NotIn(storages.Field("storage_type"), STORAGE_LOCAL_TYPES))
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	/*if jsonutils.QueryBoolean(query, "public_cloud", false) {
		sq :=
		sq = sq.Filter(sqlchemy.In(storages.Field("manager_id"), CloudproviderManager.GetPublicProviderIdsQuery()))

		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	if jsonutils.QueryBoolean(query, "private_cloud", false) {
		sq := storages.Query(storages.Field("id"))
		sq = sq.Filter(
			sqlchemy.OR(
				sqlchemy.In(storages.Field("manager_id"), CloudproviderManager.GetPrivateProviderIdsQuery()),
				sqlchemy.IsNullOrEmpty(storages.Field("manager_id")),
			),
		)
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	if jsonutils.QueryBoolean(query, "is_on_premise", false) {
		sq := storages.Query(storages.Field("id"))
		sq = sq.Filter(
			sqlchemy.OR(
				sqlchemy.In(storages.Field("manager_id"), CloudproviderManager.GetOnPremiseProviderIdsQuery()),
				sqlchemy.IsNullOrEmpty(storages.Field("manager_id")),
			),
		)
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	if jsonutils.QueryBoolean(query, "is_managed", false) {
		sq := storages.Query(storages.Field("id"))
		sq = sq.Filter(sqlchemy.IsNotEmpty(storages.Field("manager_id")))
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}*/

	if jsonutils.QueryBoolean(query, "local", false) {
		sq := storages.Query(storages.Field("id")).Filter(sqlchemy.In(storages.Field("storage_type"), STORAGE_LOCAL_TYPES))
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}

	guestId, _ := queryDict.GetString("guest")
	if len(guestId) != 0 {
		guest := GuestManager.FetchGuestById(guestId)
		if guest == nil {
			return nil, httperrors.NewResourceNotFoundError("guest %q not found", guestId)
		}
		hoststorages := HoststorageManager.Query().SubQuery()
		q = q.Join(hoststorages, sqlchemy.AND(
			sqlchemy.Equals(hoststorages.Field("host_id"), guest.HostId),
			sqlchemy.IsFalse(hoststorages.Field("deleted")))).
			Join(storages, sqlchemy.AND(
				sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id")),
				sqlchemy.IsFalse(storages.Field("deleted")))).
			Filter(sqlchemy.Equals(storages.Field("id"), q.Field("storage_id")))
	}

	storageStr := jsonutils.GetAnyString(queryDict, []string{"storage", "storage_id"})
	if len(storageStr) > 0 {
		storageObj, err := StorageManager.FetchByIdOrName(userCred, storageStr)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("storage %s not found: %s", storageStr, err)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("storage_id"), storageObj.GetId()))
	}

	/* managerStr := jsonutils.GetAnyString(query, []string{"manager", "cloudprovider", "cloudprovider_id", "manager_id"})
	if len(managerStr) > 0 {
		provider, err := CloudproviderManager.FetchByIdOrName(nil, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		subq := storages.Query(storages.Field("id")).Equals("manager_id", provider.GetId())
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), subq.SubQuery()))
	}

	accountStr := jsonutils.GetAnyString(query, []string{"account", "account_id", "cloudaccount", "cloudaccount_id"})
	if len(accountStr) > 0 {
		account, err := CloudaccountManager.FetchByIdOrName(nil, accountStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudaccountManager.Keyword(), accountStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}

		cloudproviders := CloudproviderManager.Query().SubQuery()
		subq := storages.Query(storages.Field("id"))
		subq = subq.Join(cloudproviders, sqlchemy.Equals(cloudproviders.Field("id"), storages.Field("manager_id")))
		subq = subq.Filter(sqlchemy.Equals(cloudproviders.Field("cloudaccount_id"), account.GetId()))

		q = q.Filter(sqlchemy.In(q.Field("storage_id"), subq.SubQuery()))
	}

	if provier, _ := queryDict.GetString("provider"); len(provier) > 0 {
		cloudproviders := CloudproviderManager.Query().SubQuery()
		sq := storages.Query(storages.Field("id"))
		sq = sq.Join(cloudproviders, sqlchemy.Equals(cloudproviders.Field("id"), storages.Field("manager_id")))
		sq = sq.Filter(sqlchemy.Equals(cloudproviders.Field("provider"), provier))

		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq.SubQuery()))
	}*/

	if diskType := jsonutils.GetAnyString(query, []string{"type", "disk_type"}); diskType != "" {
		q = q.Filter(sqlchemy.Equals(q.Field("disk_type"), diskType))
	}
	return q, nil
}

func (self *SDisk) GetGuestDiskCount() int {
	guestdisks := GuestdiskManager.Query()
	return guestdisks.Equals("disk_id", self.Id).Count()
}

func (self *SDisk) isAttached() bool {
	return GuestdiskManager.Query().Equals("disk_id", self.Id).Count() > 0
}

func (self *SDisk) GetGuestdisks() []SGuestdisk {
	guestdisks := make([]SGuestdisk, 0)
	q := GuestdiskManager.Query().Equals("disk_id", self.Id)
	err := q.All(&guestdisks)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return guestdisks
}
func (self *SDisk) GetGuests() []SGuest {
	result := make([]SGuest, 0)
	query := GuestManager.Query()
	guestdisks := GuestdiskManager.Query().SubQuery()
	q := query.Join(guestdisks, sqlchemy.AND(
		sqlchemy.Equals(guestdisks.Field("guest_id"), query.Field("id")))).
		Filter(sqlchemy.Equals(guestdisks.Field("disk_id"), self.Id))
	// q.DebugQuery()
	err := db.FetchModelObjects(GuestManager, q, &result)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	return result
}

func (self *SDisk) GetGuestsCount() int {
	guests := GuestManager.Query().SubQuery()
	guestdisks := GuestdiskManager.Query().SubQuery()
	return guests.Query().Join(guestdisks, sqlchemy.AND(
		sqlchemy.Equals(guestdisks.Field("guest_id"), guests.Field("id")))).
		Filter(sqlchemy.Equals(guestdisks.Field("disk_id"), self.Id)).Count()
}

func (self *SDisk) GetRuningGuestCount() int {
	guests := GuestManager.Query().SubQuery()
	guestdisks := GuestdiskManager.Query().SubQuery()
	return guests.Query().Join(guestdisks, sqlchemy.AND(
		sqlchemy.Equals(guestdisks.Field("guest_id"), guests.Field("id")))).
		Filter(sqlchemy.Equals(guestdisks.Field("disk_id"), self.Id)).
		Filter(sqlchemy.Equals(guests.Field("status"), VM_RUNNING)).Count()
}

func (self *SDisk) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	diskConfig := SDiskConfig{}
	if err := data.Unmarshal(&diskConfig, "disk"); err != nil {
		return err
	} else {
		self.fetchDiskInfo(&diskConfig)
	}
	return self.SSharableVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerProjId, query, data)
}

func (self *SDisk) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	storage := self.GetStorage()
	if storage == nil {
		return nil, httperrors.NewNotFoundError("failed to find storage for disk %s", self.Name)
	}

	host := storage.GetMasterHost()
	if host == nil {
		return nil, httperrors.NewNotFoundError("failed to find host for storage %s with disk %s", storage.Name, self.Name)
	}

	if diskType, _ := data.GetString("disk_type"); diskType != "" {
		if !utils.IsInStringArray(diskType, []string{DISK_TYPE_DATA, DISK_TYPE_VOLUME}) {
			return nil, httperrors.NewInputParameterError("not support update disk_type %s", diskType)
		}
	}

	data, err := host.GetHostDriver().ValidateUpdateDisk(ctx, userCred, data)
	if err != nil {
		return nil, err
	}

	return self.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (manager *SDiskManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	disk, err := data.Get("disk")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("disk")
	}

	diskConfig, err := parseDiskInfo(ctx, userCred, disk)
	if err != nil {
		return nil, err
	}

	storageID := jsonutils.GetAnyString(data, []string{"storage_id", "storage"})
	if storageID != "" {
		storageObj, err := StorageManager.FetchByIdOrName(nil, storageID)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("Storage %s not found", storageID)
		}
		storage := storageObj.(*SStorage)

		if len(diskConfig.Backend) == 0 {
			diskConfig.Backend = storage.StorageType
		}
		err = manager.validateDiskOnStorage(diskConfig, storage)
		if err != nil {
			return nil, err
		}
		data.Add(jsonutils.NewString(storage.Id), "storage_id")
	} else {
		diskConfig.Backend = STORAGE_LOCAL
		hypervisor, _ := data.GetString("hypervisor")
		data, err = ValidateScheduleCreateData(ctx, userCred, data, hypervisor)
		if err != nil {
			return nil, err
		}
	}
	data.Add(jsonutils.Marshal(diskConfig), "disk")

	if _, err := manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data); err != nil {
		return nil, err
	}
	pendingUsage := SQuota{Storage: diskConfig.SizeMb}
	if err := QuotaManager.CheckSetPendingQuota(ctx, userCred, userCred.GetProjectId(), &pendingUsage); err != nil {
		return nil, httperrors.NewOutOfQuotaError("%s", err)
	}
	return data, nil
}

func (manager *SDiskManager) validateDiskOnStorage(diskConfig *SDiskConfig, storage *SStorage) error {
	if !storage.Enabled {
		return httperrors.NewInputParameterError("Cannot create disk with disabled storage[%s]", storage.Name)
	}
	if !utils.IsInStringArray(storage.Status, []string{STORAGE_ENABLED, STORAGE_ONLINE}) {
		return httperrors.NewInputParameterError("Cannot create disk with offline storage[%s]", storage.Name)
	}
	if storage.StorageType != diskConfig.Backend {
		return httperrors.NewInputParameterError("Storage type[%s] not match backend %s", storage.StorageType, diskConfig.Backend)
	}
	if host := storage.GetMasterHost(); host != nil {
		//公有云磁盘大小检查。
		if err := host.GetHostDriver().ValidateDiskSize(storage, diskConfig.SizeMb>>10); err != nil {
			return httperrors.NewInputParameterError(err.Error())
		}
	}
	hoststorages := HoststorageManager.Query().SubQuery()
	hoststorage := make([]SHoststorage, 0)
	if err := hoststorages.Query().Equals("storage_id", storage.Id).All(&hoststorage); err != nil {
		return err
	}
	if len(hoststorage) == 0 {
		return httperrors.NewInputParameterError("Storage[%s] must attach to a host", storage.Name)
	}
	if diskConfig.SizeMb > storage.GetFreeCapacity() && !storage.IsEmulated {
		return httperrors.NewInputParameterError("Not enough free space")
	}
	return nil
}

func (disk *SDisk) SetStorageByHost(hostId string, diskConfig *SDiskConfig) error {
	host := HostManager.FetchHostById(hostId)
	backend := diskConfig.Backend
	if backend == "" {
		return fmt.Errorf("Backend is empty")
	}
	var storage *SStorage
	if utils.IsInStringArray(backend, STORAGE_LIMITED_TYPES) {
		storage = host.GetLeastUsedStorage(backend)
	} else {
		// unlimited pulic cloud storages
		storages := host.GetAttachedStorages("")
		for _, s := range storages {
			if s.StorageType == backend {
				tmpS := s
				storage = &tmpS
			}
		}
	}
	if storage == nil {
		return fmt.Errorf("Not found host %s backend %s storage", host.Name, backend)
	}
	err := DiskManager.validateDiskOnStorage(diskConfig, storage)
	if err != nil {
		return err
	}
	_, err = db.Update(disk, func() error {
		disk.StorageId = storage.Id
		return nil
	})
	return err
}

func getDiskResourceRequirements(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject, count int) SQuota {
	diskSize, _ := data.Int("disk", "size")
	return SQuota{
		Storage: int(diskSize) * count,
	}
}

func (manager *SDiskManager) convertToBatchCreateData(data jsonutils.JSONObject) *jsonutils.JSONDict {
	diskConfig, _ := data.Get("disk")
	newData := data.(*jsonutils.JSONDict).CopyExcludes("disk")
	newData.Add(diskConfig, "disk.0")
	return newData
}

func (manager *SDiskManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	pendingUsage := getDiskResourceRequirements(ctx, userCred, data, len(items))
	RunBatchCreateTask(ctx, items, userCred, manager.convertToBatchCreateData(data), pendingUsage, "DiskBatchCreateTask", "")
}

func (self *SDisk) StartDiskCreateTask(ctx context.Context, userCred mcclient.TokenCredential, rebuild bool, snapshot string, parentTaskId string) error {
	kwargs := jsonutils.NewDict()
	if rebuild {
		kwargs.Add(jsonutils.JSONTrue, "rebuild")
	}
	if len(snapshot) > 0 {
		kwargs.Add(jsonutils.NewString(snapshot), "snapshot")
	}
	if task, err := taskman.TaskManager.NewTask(ctx, "DiskCreateTask", self, userCred, kwargs, parentTaskId, "", nil); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) GetSnapshotCount() int {
	q := SnapshotManager.Query()
	count := q.Filter(sqlchemy.AND(sqlchemy.Equals(q.Field("disk_id"), self.Id),
		sqlchemy.Equals(q.Field("fake_deleted"), false))).Count()
	return count
}

func (self *SDisk) StartAllocate(ctx context.Context, host *SHost, storage *SStorage, taskId string, userCred mcclient.TokenCredential, rebuild bool, snapshot string, task taskman.ITask) error {
	log.Infof("Allocating disk on host %s ...", host.GetName())

	templateId := self.GetTemplateId()
	fsFormat := self.GetFsFormat()

	content := jsonutils.NewDict()
	content.Add(jsonutils.NewString(self.DiskFormat), "format")
	content.Add(jsonutils.NewInt(int64(self.DiskSize)), "size")
	if len(snapshot) > 0 {
		content.Add(jsonutils.NewString(snapshot), "snapshot")
		SnapshotManager.AddRefCount(self.SnapshotId, 1)
		self.SetMetadata(ctx, "merge_snapshot", jsonutils.JSONTrue, userCred)
	} else if len(templateId) > 0 {
		content.Add(jsonutils.NewString(templateId), "image_id")
	}
	if len(fsFormat) > 0 {
		content.Add(jsonutils.NewString(fsFormat), "fs_format")
		if fsFormat == "ext4" {
			name := strings.ToLower(self.GetName())
			for _, key := range []string{"encrypt", "secret", "cipher", "private"} {
				if strings.Index(key, name) > 0 {
					content.Add(jsonutils.JSONTrue, "encryption")
					break
				}
			}
		}
	}
	if rebuild {
		return host.GetHostDriver().RequestRebuildDiskOnStorage(ctx, host, storage, self, task, content)
	} else {
		return host.GetHostDriver().RequestAllocateDiskOnStorage(ctx, host, storage, self, task, content)
	}
}

func (self *SDisk) AllowGetDetailsConvertSnapshot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "convert-snapshot")
}

func (self *SDisk) GetDetailsConvertSnapshot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	deleteSnapshot := SnapshotManager.GetDiskFirstSnapshot(self.Id)
	if deleteSnapshot == nil {
		return nil, httperrors.NewNotFoundError("Can not get disk snapshot")
	}
	convertSnapshot, err := SnapshotManager.GetConvertSnapshot(deleteSnapshot)
	if err != nil {
		return nil, httperrors.NewBadRequestError("Get convert snapshot failed: %s", err.Error())
	}
	if convertSnapshot == nil {
		return nil, httperrors.NewBadRequestError("Snapshot %s dose not have convert snapshot", deleteSnapshot.Id)
	}
	var FakeDelete bool
	if deleteSnapshot.CreatedBy == MANUAL && !deleteSnapshot.FakeDeleted {
		FakeDelete = true
	}
	ret := jsonutils.NewDict()
	ret.Set("delete_snapshot", jsonutils.NewString(deleteSnapshot.Id))
	ret.Set("convert_snapshot", jsonutils.NewString(convertSnapshot.Id))
	ret.Set("pending_delete", jsonutils.NewBool(FakeDelete))
	return ret, nil
}

// On disk reset, auto delete snapshots after the reset snapshot(reserve manualed snapshot)
func (self *SDisk) CleanUpDiskSnapshots(ctx context.Context, userCred mcclient.TokenCredential, snapshot *SSnapshot) error {
	dest := make([]SSnapshot, 0)
	query := SnapshotManager.Query()
	query.Filter(sqlchemy.Equals(query.Field("disk_id"), self.Id)).
		GT("created_at", snapshot.CreatedAt).Asc("created_at").All(&dest)
	if len(dest) == 0 {
		return nil
	}
	convertSnapshots := jsonutils.NewArray()
	deleteSnapshots := jsonutils.NewArray()
	for i := 0; i < len(dest); i++ {
		if !dest[i].FakeDeleted && !dest[i].OutOfChain {
			convertSnapshots.Add(jsonutils.NewString(dest[i].Id))
		} else {
			deleteSnapshots.Add(jsonutils.NewString(dest[i].Id))
		}
	}
	params := jsonutils.NewDict()
	params.Set("convert_snapshots", convertSnapshots)
	params.Set("delete_snapshots", deleteSnapshots)
	task, err := taskman.TaskManager.NewTask(ctx, "DiskCleanUpSnapshotsTask", self, userCred, params, "", "", nil)
	if err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) AllowPerformCreateSnapshot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "create-snapshot")
}

func (self *SDisk) PerformCreateSnapshot(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	guests := self.GetGuests()
	if len(guests) != 1 {
		return nil, httperrors.NewBadRequestError("Disk dosen't attach guest??")
	}
	dataDict := data.(*jsonutils.JSONDict)
	dataDict.Set("disk_id", jsonutils.NewString(self.Id))
	return guests[0].PerformDiskSnapshot(ctx, userCred, query, dataDict)
}

func (self *SDisk) AllowPerformDiskReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "disk-reset")
}

func (self *SDisk) PerformDiskReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != DISK_READY {
		return nil, httperrors.NewInvalidStatusError("Cannot reset disk in status %s", self.Status)
	}
	snapshotId, err := data.GetString("snapshot_id")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("snapshot_id")
	}
	guests := self.GetGuests()
	if len(guests) > 1 {
		return nil, httperrors.NewBadRequestError("Disk attach muti guests")
	} else if len(guests) == 1 {
		if guests[0].Status != VM_READY {
			return nil, httperrors.NewServerStatusError("Disk attached guest status must be ready")
		}
	}
	iSnapshot, err := SnapshotManager.FetchById(snapshotId)
	if err != nil {
		return nil, httperrors.NewNotFoundError("Snapshot %s not found", snapshotId)
	}
	snapshot := iSnapshot.(*SSnapshot)
	if snapshot.Status != SNAPSHOT_READY {
		return nil, httperrors.NewBadRequestError("Cannot reset disk with snapshot in status %s", snapshot.Status)
	}
	autoStart := jsonutils.QueryBoolean(data, "auto_start", false)
	self.StartResetDisk(ctx, userCred, snapshotId, autoStart)
	return nil, nil
}

func (self *SDisk) StartResetDisk(ctx context.Context, userCred mcclient.TokenCredential, snapshotId string, autoStart bool) error {
	self.SetStatus(userCred, DISK_RESET, "")
	params := jsonutils.NewDict()
	params.Set("snapshot_id", jsonutils.NewString(snapshotId))
	params.Set("auto_start", jsonutils.NewBool(autoStart))
	task, err := taskman.TaskManager.NewTask(ctx, "DiskResetTask", self, userCred, params, "", "", nil)
	if err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) AllowPerformResize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "resize")
}

func (self *SDisk) PerformResize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	sizeStr, err := data.GetString("size")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("size")
	}
	sizeMb, err := fileutils.GetSizeMb(sizeStr, 'M', 1024)
	if err != nil {
		return nil, err
	}
	if self.Status != DISK_READY {
		return nil, httperrors.NewResourceNotReadyError("Resize disk when disk is READY")
	}
	if sizeMb < self.DiskSize {
		return nil, httperrors.NewUnsupportOperationError("Disk cannot be thrink")
	}
	if sizeMb == self.DiskSize {
		return nil, nil
	}
	addDisk := sizeMb - self.DiskSize
	storage := self.GetStorage()
	if host := storage.GetMasterHost(); host != nil {
		if err := host.GetHostDriver().ValidateDiskSize(storage, sizeMb>>10); err != nil {
			return nil, httperrors.NewInputParameterError(err.Error())
		}
	}
	if addDisk > storage.GetFreeCapacity() && !storage.IsEmulated {
		return nil, httperrors.NewOutOfResourceError("Not enough free space")
	}
	if guests := self.GetGuests(); len(guests) > 0 {
		if err := guests[0].ValidateResizeDisk(self, storage); err != nil {
			return nil, httperrors.NewInputParameterError(err.Error())
		}
	}
	pendingUsage := SQuota{Storage: int(addDisk)}
	if err := QuotaManager.CheckSetPendingQuota(ctx, userCred, userCred.GetProjectId(), &pendingUsage); err != nil {
		return nil, httperrors.NewOutOfQuotaError(err.Error())
	}

	guests := self.GetGuests()

	var guest *SGuest
	if len(guests) == 1 {
		guest = &guests[0]
	}

	return nil, self.StartDiskResizeTask(ctx, userCred, int64(sizeMb), "", &pendingUsage, guest)
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	storage := self.GetStorage()
	if storage == nil {
		return nil, httperrors.NewResourceNotFoundError("fail to find storage for disk %s", self.GetName())
	}
	istorage, err := storage.GetIStorage()
	if err != nil {
		return nil, err
	}
	return istorage, nil
}

func (self *SDisk) GetIDisk() (cloudprovider.ICloudDisk, error) {
	iStorage, err := self.GetIStorage()
	if err != nil {
		log.Errorf("fail to find iStorage: %v", err)
		return nil, err
	}
	return iStorage.GetIDiskById(self.GetExternalId())
}

func (self *SDisk) GetZone() *SZone {
	if storage := self.GetStorage(); storage != nil {
		return storage.getZone()
	}
	return nil
}

func (self *SDisk) PrepareSaveImage(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict) (string, error) {
	if zone := self.GetZone(); zone == nil {
		return "", httperrors.NewResourceNotFoundError("No zone for this disk")
	}
	data.Add(jsonutils.NewString(self.DiskFormat), "disk_format")
	name, _ := data.GetString("name")
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	if imageList, err := modules.Images.List(s, jsonutils.Marshal(map[string]string{"name": name, "admin": "true"})); err != nil {
		return "", err
	} else if imageList.Total > 0 {
		return "", httperrors.NewConflictError("Duplicate image name %s", name)
	}
	/*
		no need to check quota anymore
		session := auth.GetSession(userCred, options.Options.Region, "v2")
		quota := image_models.SQuota{Image: 1}
		if _, err := modules.ImageQuotas.DoQuotaCheck(session, jsonutils.Marshal(&quota)); err != nil {
			return "", err
		}*/
	data.Add(jsonutils.NewInt(int64(self.DiskSize)), "virtual_size")
	if result, err := modules.Images.Create(s, data); err != nil {
		return "", err
	} else if imageId, err := result.GetString("id"); err != nil {
		return "", err
	} else {
		return imageId, nil
	}
}

func (self *SDisk) AllowPerformSave(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "save")
}

func (self *SDisk) PerformSave(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != DISK_READY {
		return nil, httperrors.NewResourceNotReadyError("Save disk when disk is READY")

	}
	if self.GetRuningGuestCount() > 0 {
		return nil, httperrors.NewResourceNotReadyError("Save disk when not being USED")
	}

	if name, err := data.GetString("name"); err != nil || len(name) == 0 {
		return nil, httperrors.NewInputParameterError("Image name is required")
	}
	kwargs := data.(*jsonutils.JSONDict)
	if imageId, err := self.PrepareSaveImage(ctx, userCred, kwargs); err != nil {
		return nil, err
	} else {
		kwargs.Add(jsonutils.NewString(imageId), "image_id")
		return nil, self.StartDiskSaveTask(ctx, userCred, kwargs, "")
	}
}

func (self *SDisk) StartDiskSaveTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	self.SetStatus(userCred, DISK_START_SAVE, "")
	if task, err := taskman.TaskManager.NewTask(ctx, "DiskSaveTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorf("Start DiskSaveTask failed:%v", err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) ValidateDeleteCondition(ctx context.Context) error {
	return self.validateDeleteCondition(ctx, false)
}

func (self *SDisk) ValidatePurgeCondition(ctx context.Context) error {
	return self.validateDeleteCondition(ctx, true)
}

func (self *SDisk) validateDeleteCondition(ctx context.Context, isPurge bool) error {
	if self.GetGuestDiskCount() > 0 {
		return httperrors.NewNotEmptyError("Virtual disk used by virtual servers")
	}
	if !isPurge && self.IsValidPrePaid() {
		return httperrors.NewForbiddenError("not allow to delete prepaid disk in valid status")
	}
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SDisk) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	overridePendingDelete := false
	purge := false
	if query != nil {
		overridePendingDelete = jsonutils.QueryBoolean(query, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(query, "purge", false)
	}
	if (overridePendingDelete || purge) && !db.IsAdminAllowDelete(userCred, self) {
		return false
	}
	return self.IsOwner(userCred) || db.IsAdminAllowDelete(userCred, self)
}

func (self *SDisk) GetTemplateId() string {
	imageObj, err := CachedimageManager.FetchById(self.TemplateId)
	if err != nil || imageObj == nil {
		log.Errorf("failed to found disk %s(%s) templateId", self.Name, self.Id)
		return ""
	}
	return self.TemplateId
}

func (self *SDisk) IsLocal() bool {
	storage := self.GetStorage()
	if storage != nil {
		return storage.IsLocal()
	}
	return false
}

func (self *SDisk) GetStorage() *SStorage {
	store, _ := StorageManager.FetchById(self.StorageId)
	if store != nil {
		return store.(*SStorage)
	}
	return nil
}

func (self *SDisk) GetCloudprovider() *SCloudprovider {
	if storage := self.GetStorage(); storage != nil {
		return storage.GetCloudprovider()
	}
	return nil
}

func (self *SDisk) GetPathAtHost(host *SHost) string {
	hostStorage := host.GetHoststorageOfId(self.StorageId)
	if hostStorage != nil {
		return path.Join(hostStorage.MountPoint, self.Id)
	} else if len(self.BackupStorageId) > 0 {
		hostStorage = host.GetHoststorageOfId(self.BackupStorageId)
		if hostStorage != nil {
			return path.Join(hostStorage.MountPoint, self.Id)
		}
	}
	return ""
}

func (self *SDisk) GetFetchUrl() string {
	storage := self.GetStorage()
	host := storage.GetMasterHost()
	return fmt.Sprintf("%s/disks/%s", host.GetFetchUrl(), self.Id)
}

func (self *SDisk) GetFsFormat() string {
	return self.FsFormat
}

func (manager *SDiskManager) getDisksByStorage(storage *SStorage) ([]SDisk, error) {
	disks := make([]SDisk, 0)
	q := manager.Query().Equals("storage_id", storage.Id)
	err := db.FetchModelObjects(manager, q, &disks)
	if err != nil {
		log.Errorf("%s", err)
		return nil, err
	}
	return disks, nil
}

func (manager *SDiskManager) syncCloudDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, vdisk cloudprovider.ICloudDisk, index int, projectId string) (*SDisk, error) {
	ownerProjId := projectId

	lockman.LockClass(ctx, manager, ownerProjId)
	defer lockman.ReleaseClass(ctx, manager, ownerProjId)

	diskObj, err := manager.FetchByExternalId(vdisk.GetGlobalId())
	if err != nil {
		if err == sql.ErrNoRows {
			vstorage, _ := vdisk.GetIStorage()

			storageObj, err := StorageManager.FetchByExternalId(vstorage.GetGlobalId())
			if err != nil {
				log.Errorf("cannot find storage of vdisk %s", err)
				return nil, err
			}
			storage := storageObj.(*SStorage)
			return manager.newFromCloudDisk(ctx, userCred, provider, vdisk, storage, -1, ownerProjId)
		} else {
			return nil, err
		}
	} else {
		disk := diskObj.(*SDisk)
		err = disk.syncWithCloudDisk(ctx, userCred, provider, vdisk, index, ownerProjId)
		if err != nil {
			return nil, err
		}
		return disk, nil
	}
}

func (manager *SDiskManager) SyncDisks(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, storage *SStorage, disks []cloudprovider.ICloudDisk, projectId string) ([]SDisk, []cloudprovider.ICloudDisk, compare.SyncResult) {
	syncOwnerId := projectId

	lockman.LockClass(ctx, manager, syncOwnerId)
	defer lockman.ReleaseClass(ctx, manager, syncOwnerId)

	localDisks := make([]SDisk, 0)
	remoteDisks := make([]cloudprovider.ICloudDisk, 0)
	syncResult := compare.SyncResult{}

	dbDisks, err := manager.getDisksByStorage(storage)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SDisk, 0)
	commondb := make([]SDisk, 0)
	commonext := make([]cloudprovider.ICloudDisk, 0)
	added := make([]cloudprovider.ICloudDisk, 0)

	err = compare.CompareSets(dbDisks, disks, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudDisk(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudDisk(ctx, userCred, provider, commonext[i], -1, projectId)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncMetadata(ctx, userCred, &commondb[i], commonext[i])
			localDisks = append(localDisks, commondb[i])
			remoteDisks = append(remoteDisks, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		extId := added[i].GetGlobalId()
		_disk, err := manager.FetchByExternalId(extId)
		if err != nil && err != sql.ErrNoRows {
			//主要是显示duplicate err及 general err,方便排错
			msg := fmt.Errorf("failed to found disk by external Id %s error: %v", extId, err)
			syncResult.Error(msg)
			continue
		}
		if _disk != nil {
			disk := _disk.(*SDisk)
			err = disk.syncDiskStorage(ctx, userCred, added[i])
			if err != nil {
				syncResult.UpdateError(err)
			} else {
				syncResult.Update()
			}
			continue
		}
		new, err := manager.newFromCloudDisk(ctx, userCred, provider, added[i], storage, -1, syncOwnerId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncMetadata(ctx, userCred, new, added[i])
			localDisks = append(localDisks, *new)
			remoteDisks = append(remoteDisks, added[i])
			syncResult.Add()
		}
	}

	return localDisks, remoteDisks, syncResult
}

func (self *SDisk) syncDiskStorage(ctx context.Context, userCred mcclient.TokenCredential, idisk cloudprovider.ICloudDisk) error {
	extId := idisk.GetGlobalId()
	istorage, err := idisk.GetIStorage()
	if err != nil {
		log.Errorf("failed to get istorage for disk %s error: %v", extId, err)
		return err
	}
	storageExtId := istorage.GetGlobalId()
	storage, err := StorageManager.FetchByExternalId(storageExtId)
	if err != nil {
		log.Errorf("failed to found storage by istorage %s error: %v", storageExtId, err)
		return err
	}
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.StorageId = storage.GetId()
		self.Status = idisk.GetStatus()
		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudDisk error %s", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SDisk) syncRemoveCloudDisk(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	return self.SetStatus(userCred, DISK_UNKNOWN, "missing original disk after sync")
}

func (self *SDisk) syncWithCloudDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, extDisk cloudprovider.ICloudDisk, index int, projectId string) error {
	recycle := false
	guests := self.GetGuests()
	if provider.GetFactory().IsSupportPrepaidResources() && len(guests) == 1 && guests[0].IsPrepaidRecycle() {
		recycle = true
	}
	extDisk.Refresh()

	storage := self.GetStorage()
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		// self.Name = extDisk.GetName()
		self.Status = extDisk.GetStatus()
		self.DiskFormat = extDisk.GetDiskFormat()
		self.DiskSize = extDisk.GetDiskSizeMB()
		self.AccessPath = extDisk.GetAccessPath()
		if extDisk.GetIsAutoDelete() {
			self.AutoDelete = true
		}
		// self.TemplateId = extDisk.GetTemplateId() no sync template ID
		self.DiskType = extDisk.GetDiskType()
		if index == 0 {
			self.DiskType = DISK_TYPE_SYS
		}
		// self.FsFormat = extDisk.GetFsFormat()
		self.Nonpersistent = extDisk.GetIsNonPersistent()

		self.IsEmulated = extDisk.IsEmulated()

		if provider.GetFactory().IsSupportPrepaidResources() && !recycle {
			self.BillingType = extDisk.GetBillingType()
			self.ExpiredAt = extDisk.GetExpiredAt()
		}

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudDisk error %s", err)
		return err
	}

	db.OpsLog.LogSyncUpdate(self, diff, userCred)

	SyncCloudProject(userCred, self, projectId, extDisk, storage.ManagerId)

	return nil
}

func (manager *SDiskManager) newFromCloudDisk(ctx context.Context, userCred mcclient.TokenCredential, provider cloudprovider.ICloudProvider, extDisk cloudprovider.ICloudDisk, storage *SStorage, index int, projectId string) (*SDisk, error) {
	disk := SDisk{}
	disk.SetModelManager(manager)

	disk.Name = db.GenerateName(manager, projectId, extDisk.GetName())
	disk.Status = extDisk.GetStatus()
	disk.ExternalId = extDisk.GetGlobalId()
	disk.StorageId = storage.Id

	disk.DiskFormat = extDisk.GetDiskFormat()
	disk.DiskSize = extDisk.GetDiskSizeMB()
	disk.AutoDelete = extDisk.GetIsAutoDelete()
	disk.DiskType = extDisk.GetDiskType()
	if index == 0 {
		disk.DiskType = DISK_TYPE_SYS
	}
	disk.Nonpersistent = extDisk.GetIsNonPersistent()

	disk.IsEmulated = extDisk.IsEmulated()

	if provider.GetFactory().IsSupportPrepaidResources() {
		disk.BillingType = extDisk.GetBillingType()
		disk.ExpiredAt = extDisk.GetExpiredAt()
	}

	err := manager.TableSpec().Insert(&disk)
	if err != nil {
		log.Errorf("newFromCloudZone fail %s", err)
		return nil, err
	}

	SyncCloudProject(userCred, &disk, projectId, extDisk, storage.ManagerId)

	db.OpsLog.LogEvent(&disk, db.ACT_CREATE, disk.GetShortDesc(ctx), userCred)

	return &disk, nil
}

func totalDiskSize(projectId string, active tristate.TriState, ready tristate.TriState, includeSystem bool) int {
	disks := DiskManager.Query().SubQuery()
	q := disks.Query(sqlchemy.SUM("total", disks.Field("disk_size")))
	if !active.IsNone() {
		storages := StorageManager.Query().SubQuery()
		q = q.Join(storages, sqlchemy.AND(sqlchemy.IsFalse(storages.Field("deleted")),
			sqlchemy.Equals(storages.Field("id"), disks.Field("storage_id"))))
		if active.IsTrue() {
			q = q.Filter(sqlchemy.In(storages.Field("status"), []string{STORAGE_ENABLED, STORAGE_ONLINE}))
		} else {
			q = q.Filter(sqlchemy.NotIn(storages.Field("status"), []string{STORAGE_ENABLED, STORAGE_ONLINE}))
		}
	}
	if len(projectId) > 0 {
		q = q.Filter(sqlchemy.OR(sqlchemy.Equals(disks.Field("tenant_id"), projectId), sqlchemy.IsTrue(disks.Field("is_public"))))
	}
	if !ready.IsNone() {
		if ready.IsTrue() {
			q = q.Filter(sqlchemy.Equals(disks.Field("status"), DISK_READY))
		} else {
			q = q.Filter(sqlchemy.NotEquals(disks.Field("status"), DISK_READY))
		}
	}
	if !includeSystem {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(disks.Field("is_system")),
			sqlchemy.IsFalse(disks.Field("is_system"))))
	}
	row := q.Row()
	size := 0
	err := row.Scan(&size)
	if err != nil {
		log.Errorf("totalDiskSize error %s", err)
	}
	return size
}

type SDiskConfig struct {
	ImageId string `json:"image_id"`

	SnapshotId string `json:"snapshot_id"`
	DiskType   string `json:"disk_type"` // sys, data, swap, volume

	// ImageDiskFormat string
	SizeMb          int               `json:"size"`       // MB
	Fs              string            `json:"fs"`         // file system
	Format          string            `json:"format"`     //
	Driver          string            `json:"driver"`     //
	Cache           string            `json:"cache"`      //
	Mountpoint      string            `json:"mountpoint"` //
	Backend         string            `json:"backend"`    // stroageType
	Medium          string            `json:"medium"`
	ImageProperties map[string]string `json:"image_properties"`

	Storage string `json:"storage_id"`
	DiskId  string `json:"-"` // import only
}

func parseDiskInfo(ctx context.Context, userCred mcclient.TokenCredential, info jsonutils.JSONObject) (*SDiskConfig, error) {
	diskConfig := SDiskConfig{}

	diskJson, ok := info.(*jsonutils.JSONDict)
	if ok {
		err := diskJson.Unmarshal(&diskConfig)
		if err != nil {
			return nil, err
		}
		return &diskConfig, nil
	}

	// default backend and medium type
	diskConfig.Backend = "" // STORAGE_LOCAL
	diskConfig.Medium = DISK_TYPE_HYBRID

	diskStr, err := info.GetString()
	if err != nil {
		log.Errorf("invalid diskinfo format %s", err)
		return nil, err
	}
	parts := strings.Split(diskStr, ":")
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		if regutils.MatchSize(p) {
			diskConfig.SizeMb, _ = fileutils.GetSizeMb(p, 'M', 1024)
		} else if utils.IsInStringArray(p, osprofile.FS_TYPES) {
			diskConfig.Fs = p
		} else if utils.IsInStringArray(p, osprofile.IMAGE_FORMAT_TYPES) {
			diskConfig.Format = p
		} else if utils.IsInStringArray(p, osprofile.DISK_DRIVERS) {
			diskConfig.Driver = p
		} else if utils.IsInStringArray(p, osprofile.DISK_CACHE_MODES) {
			diskConfig.Cache = p
		} else if utils.IsInStringArray(p, DISK_TYPES) {
			diskConfig.Medium = p
		} else if utils.IsInStringArray(p, []string{DISK_TYPE_VOLUME}) {
			diskConfig.DiskType = p
		} else if p[0] == '/' {
			diskConfig.Mountpoint = p
		} else if p == "autoextend" {
			diskConfig.SizeMb = -1
		} else if storageType, exist := StorageManager.IsStorageTypeExist(p); exist {
			diskConfig.Backend = storageType
		} else if strings.HasPrefix(p, "snapshot-") {
			// HACK: use snapshot creat disk format snapshot-id
			// example: snapshot-3140cecb-ccc4-4865-abae-3a5ba8c69d9b
			if err := fillDiskConfigBySnapshot(userCred, &diskConfig, p[len("snapshot-"):]); err != nil {
				return nil, err
			}
		} else if len(p) > 0 {
			if err := fillDiskConfigByImage(ctx, userCred, &diskConfig, p); err != nil {
				return nil, err
			}
		}
	}
	// XXX: do not set default disk size here, set it by each hypervisor driver
	// if len(diskConfig.ImageId) > 0 && diskConfig.SizeMb == 0 {
	// 	diskConfig.SizeMb = options.Options.DefaultDiskSize // MB
	// else
	if len(diskConfig.ImageId) == 0 && diskConfig.SizeMb == 0 {
		return nil, httperrors.NewInputParameterError("Diskinfo not contains either imageID or size")
	}
	return &diskConfig, nil
}

func fillDiskConfigBySnapshot(userCred mcclient.TokenCredential, diskConfig *SDiskConfig, snapshotId string) error {
	iSnapshot, err := SnapshotManager.FetchByIdOrName(userCred, snapshotId)
	if err != nil {
		if err == sql.ErrNoRows {
			return httperrors.NewNotFoundError("Snapshot %s not found", snapshotId)
		}
		return err
	}
	var snapshot = iSnapshot.(*SSnapshot)
	if storage := StorageManager.FetchStorageById(snapshot.StorageId); storage == nil {
		return httperrors.NewBadRequestError("Snapshot %s storage %s not found, is public cloud?",
			snapshotId, snapshot.StorageId)
	} else {
		if disk := DiskManager.FetchDiskById(snapshot.DiskId); disk != nil {
			diskConfig.Fs = disk.FsFormat
			if len(diskConfig.Format) == 0 {
				diskConfig.Format = disk.DiskFormat
			}
		}
		diskConfig.SnapshotId = snapshot.Id
		diskConfig.DiskType = snapshot.DiskType
		diskConfig.SizeMb = snapshot.Size
		diskConfig.Backend = storage.StorageType
	}
	return nil
}

func fillDiskConfigByImage(ctx context.Context, userCred mcclient.TokenCredential,
	diskConfig *SDiskConfig, imageId string) error {
	if userCred == nil {
		diskConfig.ImageId = imageId
	} else {
		image, err := CachedimageManager.getImageInfo(ctx, userCred, imageId, false)
		if err != nil {
			log.Errorf("getImageInfo fail %s", err)
			return err
		}
		if image.Status != cloudprovider.IMAGE_STATUS_ACTIVE {
			return httperrors.NewInvalidStatusError("Image status is not active")
		}
		diskConfig.ImageId = image.Id
		diskConfig.ImageProperties = image.Properties
		if len(diskConfig.Format) == 0 {
			diskConfig.Format = image.DiskFormat
		}
		// diskConfig.ImageDiskFormat = image.DiskFormat
		CachedimageManager.ImageAddRefCount(image.Id)
		if diskConfig.SizeMb < image.MinDiskMB {
			diskConfig.SizeMb = image.MinDiskMB // MB
		}
	}
	return nil
}

func parseIsoInfo(ctx context.Context, userCred mcclient.TokenCredential, imageId string) (*cloudprovider.SImage, error) {
	image, err := CachedimageManager.getImageInfo(ctx, userCred, imageId, false)
	if err != nil {
		log.Errorf("getImageInfo fail %s", err)
		return nil, err
	}
	if image.Status != cloudprovider.IMAGE_STATUS_ACTIVE {
		return nil, httperrors.NewInvalidStatusError("Image status is not active")
	}
	return image, nil
}

func (self *SDisk) fetchDiskInfo(diskConfig *SDiskConfig) {
	if len(diskConfig.ImageId) > 0 {
		self.TemplateId = diskConfig.ImageId
		self.DiskType = DISK_TYPE_SYS
	} else if len(diskConfig.SnapshotId) > 0 {
		self.SnapshotId = diskConfig.SnapshotId
		self.DiskType = diskConfig.DiskType
	}
	if len(diskConfig.Fs) > 0 {
		self.FsFormat = diskConfig.Fs
	}
	if self.FsFormat == "swap" {
		self.DiskType = DISK_TYPE_SWAP
		self.Nonpersistent = true
	} else {
		if len(self.DiskType) == 0 {
			diskType := DISK_TYPE_DATA
			if diskConfig.DiskType == DISK_TYPE_VOLUME {
				diskType = DISK_TYPE_VOLUME
			}
			self.DiskType = diskType
		}
		self.Nonpersistent = false
	}
	if len(diskConfig.DiskId) > 0 && utils.IsMatchUUID(diskConfig.DiskId) {
		self.Id = diskConfig.DiskId
	}
	self.DiskFormat = diskConfig.Format
	self.DiskSize = diskConfig.SizeMb
}

type DiskInfo struct {
	ImageId    string
	Fs         string
	MountPoint string
	Format     string
	Size       int64
	Storage    string
	Backend    string
	MediumType string
	Driver     string
	Cache      string
	DiskType   string
}

func (self *SDisk) ToDiskInfo() DiskInfo {
	ret := DiskInfo{
		ImageId:    self.GetTemplateId(),
		Fs:         self.GetFsFormat(),
		MountPoint: self.GetMountPoint(),
		Format:     self.DiskFormat,
		Size:       int64(self.DiskSize),
		DiskType:   self.DiskType,
	}
	storage := self.GetStorage()
	if storage == nil {
		return ret
	}
	ret.Storage = storage.Id
	ret.Backend = storage.StorageType
	ret.MediumType = storage.MediumType
	return ret
}

func (self *SDisk) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	// override
	log.Infof("disk delete do nothing")
	return nil
}

func (self *SDisk) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	guestdisks := self.GetGuestdisks()
	if guestdisks != nil {
		for _, guestdisk := range guestdisks {
			guestdisk.Detach(ctx, userCred)
		}
	}
	return self.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SDisk) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, self, "purge")
}

func (self *SDisk) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidatePurgeCondition(ctx)
	if err != nil {
		return nil, err
	}

	provider := self.GetCloudprovider()
	if provider != nil && provider.Provider == CLOUD_PROVIDER_HUAWEI && self.GetSnapshotCount() > 0 {
		return nil, httperrors.NewForbiddenError("not allow to purge. Virtual disk must not have snapshots")
	}

	return nil, self.StartDiskDeleteTask(ctx, userCred, "", true, false)
}

func (self *SDisk) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	provider := self.GetCloudprovider()
	if provider != nil && provider.Provider == CLOUD_PROVIDER_HUAWEI && self.GetSnapshotCount() > 0 {
		return httperrors.NewForbiddenError("not allow to delete. Virtual disk must not have snapshots")
	}

	return self.StartDiskDeleteTask(ctx, userCred, "", false,
		jsonutils.QueryBoolean(query, "override_pending_delete", false))
}

func (self *SDisk) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	if cloudprovider := self.GetCloudprovider(); cloudprovider != nil {
		extra.Add(jsonutils.NewString(cloudprovider.Provider), "provider")
	}
	if storage := self.GetStorage(); storage != nil {
		extra.Add(jsonutils.NewString(storage.GetName()), "storage")
		extra.Add(jsonutils.NewString(storage.StorageType), "storage_type")
		extra.Add(jsonutils.NewString(storage.MediumType), "medium_type")
		/*extra.Add(jsonutils.NewString(storage.ZoneId), "zone_id")
		if zone := storage.getZone(); zone != nil {
			extra.Add(jsonutils.NewString(zone.Name), "zone")
			extra.Add(jsonutils.NewString(zone.CloudregionId), "region_id")
			if region := zone.GetRegion(); region != nil {
				extra.Add(jsonutils.NewString(region.Name), "region")
			}
		}*/

		info := storage.getCloudProviderInfo()
		extra.Update(jsonutils.Marshal(&info))
	}
	guestArray := jsonutils.NewArray()
	guests, guest_status := []string{}, []string{}
	for _, guest := range self.GetGuests() {
		guests = append(guests, guest.Name)
		guest_status = append(guest_status, guest.Status)
		guestArray.Add(jsonutils.Marshal(map[string]string{"name": guest.Name, "id": guest.Id, "status": guest.Status}))
	}
	extra.Add(guestArray, "guests")
	extra.Add(jsonutils.NewString(strings.Join(guests, ",")), "guest")
	extra.Add(jsonutils.NewInt(int64(len(guests))), "guest_count")
	extra.Add(jsonutils.NewString(strings.Join(guest_status, ",")), "guest_status")

	if self.PendingDeleted {
		pendingDeletedAt := self.PendingDeletedAt.Add(time.Second * time.Duration(options.Options.PendingDeleteExpireSeconds))
		extra.Add(jsonutils.NewString(timeutils.FullIsoTime(pendingDeletedAt)), "auto_delete_at")
	}
	return extra
}

func (self *SDisk) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SSharableVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(extra), nil
}

func (self *SDisk) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SSharableVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SDisk) StartDiskResizeTask(ctx context.Context, userCred mcclient.TokenCredential, sizeMb int64, parentTaskId string, pendingUsage quotas.IQuota, guest *SGuest) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(sizeMb), "size")
	if guest != nil {
		params.Add(jsonutils.NewString(guest.Id), "guest_id")
	}
	if task, err := taskman.TaskManager.NewTask(ctx, "DiskResizeTask", self, userCred, params, parentTaskId, "", pendingUsage); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) StartDiskDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, isPurge, overridePendingDelete bool) error {
	params := jsonutils.NewDict()
	if isPurge {
		params.Add(jsonutils.JSONTrue, "purge")
	}
	if overridePendingDelete {
		params.Add(jsonutils.JSONTrue, "override_pending_delete")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "DiskDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		log.Errorf("%s", err)
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SDisk) GetAttachedGuests() []SGuest {
	guests := GuestManager.Query().SubQuery()
	guestdisks := GuestdiskManager.Query().SubQuery()

	q := guests.Query()
	q = q.Join(guestdisks, sqlchemy.AND(sqlchemy.Equals(guestdisks.Field("guest_id"), guests.Field("id")),
		sqlchemy.IsFalse(guestdisks.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(guestdisks.Field("disk_id"), self.Id))

	ret := make([]SGuest, 0)
	if err := db.FetchModelObjects(GuestManager, q, &ret); err != nil {
		log.Errorf("Fetch Geusts Objects %v", err)
		return nil
	}
	return ret
}

func (self *SDisk) SetDiskReady(ctx context.Context, userCred mcclient.TokenCredential, reason string) {
	self.SetStatus(userCred, DISK_READY, reason)
	guests := self.GetAttachedGuests()
	if guests != nil {
		for _, guest := range guests {
			guest.StartSyncstatus(ctx, userCred, "")
		}
	}
}

func (self *SDisk) SwitchToBackup(userCred mcclient.TokenCredential) error {
	diff, err := db.Update(self, func() error {
		self.StorageId, self.BackupStorageId = self.BackupStorageId, self.StorageId
		return nil
	})
	if err != nil {
		log.Errorf("SwitchToBackup fail %s", err)
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	return nil
}

func (self *SDisk) ClearHostSchedCache() error {
	storage := self.GetStorage()
	hosts := storage.GetAllAttachingHosts()
	if hosts == nil {
		return fmt.Errorf("get attaching host error")
	}
	for _, h := range hosts {
		err := h.ClearSchedDescCache()
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SDisk) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SSharableVirtualResourceBase.GetShortDesc(ctx)
	desc.Add(jsonutils.NewInt(int64(self.DiskSize)), "size")
	storage := self.GetStorage()
	if storage != nil {
		desc.Add(jsonutils.NewString(storage.StorageType), "storage_type")
		desc.Add(jsonutils.NewString(storage.MediumType), "medium_type")
	}

	if hypervisor := self.GetMetadata("hypervisor", nil); len(hypervisor) > 0 {
		desc.Add(jsonutils.NewString(hypervisor), "hypervisor")
	}

	if len(self.ExternalId) > 0 {
		desc.Add(jsonutils.NewString(self.ExternalId), "externalId")
	}

	fs := self.GetFsFormat()
	if len(fs) > 0 {
		desc.Add(jsonutils.NewString(fs), "fs_format")
	}
	tid := self.GetTemplateId()
	if len(tid) > 0 {
		desc.Add(jsonutils.NewString(tid), "template_id")
	}

	var billingInfo SCloudBillingInfo

	if storage != nil {
		billingInfo.SCloudProviderInfo = storage.getCloudProviderInfo()
	}

	if priceKey := self.GetMetadata("ext:price_key", nil); len(priceKey) > 0 {
		billingInfo.PriceKey = priceKey
	}

	billingInfo.SBillingBaseInfo = self.getBillingBaseInfo()

	desc.Update(jsonutils.Marshal(billingInfo))

	return desc
}

func (self *SDisk) getDev() string {
	return self.GetMetadata("dev", nil)
}

func (self *SDisk) GetMountPoint() string {
	return self.GetMetadata("mountpoint", nil)
}

func (self *SDisk) isReady() bool {
	return self.Status == DISK_READY
}

func (self *SDisk) isInit() bool {
	return self.Status == DISK_INIT
}

func (self *SDisk) AllowPerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "cancel-delete")
}

func (self *SDisk) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.PendingDeleted {
		err := self.DoCancelPendingDelete(ctx, userCred)
		return nil, err
	}
	return nil, nil
}

func (manager *SDiskManager) getExpiredPendingDeleteDisks() []SDisk {
	deadline := time.Now().Add(time.Duration(options.Options.PendingDeleteExpireSeconds*-1) * time.Second)

	q := manager.Query()
	q = q.IsTrue("pending_deleted").LT("pending_deleted_at", deadline).Limit(options.Options.PendingDeleteMaxCleanBatchSize)

	disks := make([]SDisk, 0)
	err := db.FetchModelObjects(DiskManager, q, &disks)
	if err != nil {
		log.Errorf("fetch disks error %s", err)
		return nil
	}

	return disks
}

func (manager *SDiskManager) CleanPendingDeleteDisks(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	disks := manager.getExpiredPendingDeleteDisks()
	if disks == nil {
		return
	}
	for i := 0; i < len(disks); i += 1 {
		disks[i].StartDiskDeleteTask(ctx, userCred, "", false, false)
	}
}

func (manager *SDiskManager) getAutoSnapshotDisks() []SDisk {
	q := manager.Query().SubQuery()
	dest := make([]SDisk, 0)
	err := q.Query().Filter(sqlchemy.Equals(q.Field("auto_snapshot"), true)).All(&dest)
	if err != nil {
		return nil
	}
	return dest
}

func (manager *SDiskManager) AutoDiskSnapshot(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	disks := manager.getAutoSnapshotDisks()
	if disks == nil {
		return
	}
	for _, disk := range disks {
		snapCount := disk.GetSnapshotCount()
		if snapCount >= options.Options.DefaultMaxSnapshotCount {
			continue
		}
		guests := disk.GetGuests()
		if len(guests) != 1 {
			log.Errorf("Disk %s(%s) is attached to %d guest(s)", disk.Name, disk.Id, len(guests))
			continue
		}
		if !utils.IsInStringArray(guests[0].Status, []string{VM_RUNNING, VM_READY}) {
			log.Errorf("Guest(%s) in status(%s) cannot do snapshot action", guests[0].Id, guests[0].Status)
			continue
		}
		// name
		name := "Auto-" + guests[0].Name + time.Now().Format("2006-01-02#15:04:05")
		snap, err := SnapshotManager.CreateSnapshot(ctx, userCred, AUTO, disk.Id, guests[0].Id, "", name)
		if err != nil {
			log.Errorln(err)
			continue
		}
		guests[0].StartDiskSnapshot(ctx, userCred, disk.Id, snap.Id)
	}
}

func (disk *SDisk) StratCreateBackupTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "DiskCreateBackupTask", disk, userCred, nil, parentTaskId, "", nil); err != nil {
		log.Errorln(err)
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) SaveRenewInfo(ctx context.Context, userCred mcclient.TokenCredential, bc *billing.SBillingCycle, expireAt *time.Time) error {
	_, err := db.Update(self, func() error {
		if self.BillingType != BILLING_TYPE_PREPAID {
			self.BillingType = BILLING_TYPE_PREPAID
		}
		if expireAt != nil && !expireAt.IsZero() {
			self.ExpiredAt = *expireAt
		} else if bc != nil {
			self.BillingCycle = bc.String()
			self.ExpiredAt = bc.EndAt(self.ExpiredAt)
		}
		return nil
	})
	if err != nil {
		log.Errorf("Update error %s", err)
		return err
	}
	db.OpsLog.LogEvent(self, db.ACT_RENEW, self.GetShortDesc(ctx), userCred)
	return nil
}

func (self *SDisk) IsDetachable() bool {
	storage := self.GetStorage()
	if storage == nil {
		return true
	}
	if storage.IsLocal() {
		return false
	}
	if self.BillingType == BILLING_TYPE_PREPAID {
		return false
	}
	if utils.IsInStringArray(self.DiskType, []string{DISK_TYPE_SYS, DISK_TYPE_SWAP}) {
		return false
	}
	if self.AutoDelete {
		return false
	}
	return true
}
