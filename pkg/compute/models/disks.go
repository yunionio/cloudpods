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
	"yunion.io/x/pkg/util/sysutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	DISK_INIT           = "init"
	DISK_REBUILD        = "rebuild"
	DISK_ALLOC_FAILED   = "alloc_failed"
	DISK_STARTALLOC     = "start_alloc"
	DISK_ALLOCATING     = "allocating"
	DISK_READY          = "ready"
	DISK_DEALLOC        = "deallocating"
	DISK_DEALLOC_FAILED = "dealloc_failed"
	DISK_UNKNOWN        = "unknown"
	DISK_DETACHING      = "detaching"

	DISK_START_SAVE = "start_save"
	DISK_SAVING     = "saving"

	DISK_START_RESIZE = "start_resize"
	DISK_RESIZING     = "resizing"

	DISK_START_MIGRATE = "start_migrate"
	DISK_POST_MIGRATE  = "post_migrate"
	DISK_MIGRATING     = "migrating"

	DISK_TYPE_SYS  = "sys"
	DISK_TYPE_SWAP = "swap"
	DISK_TYPE_DATA = "data"

	DISK_BACKING_IMAGE = "image"
)

type SDiskManager struct {
	db.SSharableVirtualResourceBaseManager
}

var DiskManager *SDiskManager

func init() {
	DiskManager = &SDiskManager{SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(SDisk{}, "disks_tbl", "disk", "disks")}
}

type SDisk struct {
	db.SSharableVirtualResourceBase

	DiskFormat string `width:"32" charset:"ascii" nullable:"false" default:"qcow2" list:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=False, default='qcow2')
	DiskSize   int    `nullable:"false" list:"user"`                                            // Column(Integer, nullable=False) # in MB
	AccessPath string `width:"256" charset:"ascii" nullable:"true" get:"user"`                  // = Column(VARCHAR(256, charset='ascii'), nullable=True)

	AutoDelete bool `nullable:"false" default:"false" get:"user" update:"user"` // Column(Boolean, nullable=False, default=False)

	StorageId string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"required"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)

	// # backing template id and type
	TemplateId string `width:"128" charset:"ascii" nullable:"true" list:"user"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=True)
	// # file system
	FsFormat string `width:"32" charset:"ascii" nullable:"true" list:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	// # disk type, OS, SWAP, DAT
	DiskType string `width:"32" charset:"ascii" nullable:"true" list:"user"` // Column(VARCHAR(32, charset='ascii'), nullable=True)
	// # is persistent
	Nonpersistent bool `default:"false" list:"user"` // Column(Boolean, default=False)
}

func (manager *SDiskManager) GetContextManager() []db.IModelManager {
	return []db.IModelManager{StorageManager}
}

func (manager *SDiskManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	queryDict, ok := query.(*jsonutils.JSONDict)
	if !ok {
		return nil, fmt.Errorf("Invalid querystring formst: %v", query)
	}
	if jsonutils.QueryBoolean(query, "unused", false) {
		guestdisks := GuestdiskManager.Query().SubQuery()
		sq := guestdisks.Query(guestdisks.Field("disk_id"))
		q = q.Filter(sqlchemy.NotIn(q.Field("id"), sq))
	}
	storages := StorageManager.Query().SubQuery()
	if jsonutils.QueryBoolean(query, "share", false) {
		sq := storages.Query(storages.Field("id")).Filter(sqlchemy.NotIn(storages.Field("storage_type"), STORAGE_LOCAL_TYPES))
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}
	if jsonutils.QueryBoolean(query, "local", false) {
		sq := storages.Query(storages.Field("id")).Filter(sqlchemy.In(storages.Field("storage_type"), STORAGE_LOCAL_TYPES))
		q = q.Filter(sqlchemy.In(q.Field("storage_id"), sq))
	}
	if provier, _ := queryDict.GetString("provider"); len(provier) > 0 {
		cloudprovider := CloudproviderManager.Query().SubQuery()
		sq := storages.Query(storages.Field("id")).Join(cloudprovider,
			sqlchemy.AND(
				sqlchemy.Equals(cloudprovider.Field("id"), storages.Field("manager_id")),
				sqlchemy.Equals(cloudprovider.Field("provider"), provier)))
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
		storageObj, err := StorageManager.FetchByIdOrName(userCred.GetProjectId(), storageStr)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("storage %s not found: %s", storageStr, err)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("storage_id"), storageObj.GetId()))
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
		log.Errorf(err.Error())
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

func (manager *SDiskManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if disk, err := data.Get("disk"); err != nil {
		return nil, err
	} else {
		if diskConfig, err := parseDiskInfo(ctx, userCred, disk); err != nil {
			return nil, err
		} else {
			data.Add(jsonutils.Marshal(diskConfig), "disk")
			if storageID, err := data.GetString("storage_id"); err != nil {
				return nil, err
			} else {
				storages := StorageManager.Query().SubQuery()
				storage := SStorage{}
				storage.SetModelManager(StorageManager)
				if err := storages.Query().Equals("id", storageID).First(&storage); err != nil {
					return nil, err
				}
				if !storage.Enabled {
					return nil, httperrors.NewInputParameterError("Cannot create disk with disabled storage[%s]", storage.Name)
				}
				if !utils.IsInStringArray(storage.Status, []string{STORAGE_ENABLED, STORAGE_ONLINE}) {
					return nil, httperrors.NewInputParameterError("Cannot create disk with offline storage[%s]", storage.Name)
				}
				if storage.StorageType != diskConfig.Backend {
					return nil, httperrors.NewInputParameterError("Storage type[%s] not match backend %s", storage.StorageType, diskConfig.Backend)
				}
				size := diskConfig.Size >> 10
				if storage.StorageType == STORAGE_RBD {
					diskConfig.Format = "raw"
					data.Add(jsonutils.Marshal(diskConfig), "disk")
				} else if storage.StorageType == STORAGE_CLOUD_EFFICIENCY || storage.StorageType == STORAGE_CLOUD_SSD {
					if size < 20 || size > 32768 {
						return nil, httperrors.NewInputParameterError("cloud_ssd or cloud_efficiency disk only support 20G ~ 32768G")
					}
				} else if storage.StorageType == STORAGE_PUBLIC_CLOUD {
					if size < 5 || size > 2000 {
						return nil, httperrors.NewInputParameterError("cloud disk only support 5G ~ 2000G")
					}
				}
				hoststorages := HoststorageManager.Query().SubQuery()
				hoststorage := make([]SHoststorage, 0)
				if err := hoststorages.Query().Equals("storage_id", storage.Id).All(&hoststorage); err != nil {
					return nil, err
				}
				if len(hoststorage) == 0 {
					return nil, httperrors.NewInputParameterError("Storage[%s] must attach to a host", storage.Name)
				}
				if diskConfig.Size > storage.GetFreeCapacity() && !storage.IsEmulated {
					return nil, httperrors.NewInputParameterError("Not enough free space")
				}
				if _, err := manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data); err != nil {
					return nil, err
				}
				pendingUsage := SQuota{Storage: diskConfig.Size}
				if err := QuotaManager.CheckSetPendingQuota(ctx, userCred, userCred.GetProjectId(), &pendingUsage); err != nil {
					return nil, err
				}
			}
		}
	}
	return data, nil
}

func (disk *SDisk) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	disk.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	disk.StartDiskCreateTask(ctx, userCred, false, "", "")
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

func (self *SDisk) StartAllocate(host *SHost, storage *SStorage, taskId string, userCred mcclient.TokenCredential, rebuild bool, snapshot string, task taskman.ITask) error {
	log.Infof("Allocating disk on host %s ...", host.GetName())

	templateId := self.GetTemplateId()
	fsFormat := self.GetFsFormat()

	content := jsonutils.NewDict()
	content.Add(jsonutils.NewString(self.DiskFormat), "format")
	content.Add(jsonutils.NewInt(int64(self.DiskSize)), "size")
	if len(snapshot) > 0 {
		content.Add(jsonutils.NewString(snapshot), "snapshot")
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
		content.Add(jsonutils.JSONTrue, "rebuild")
	}
	return host.GetHostDriver().RequestAllocateDiskOnStorage(host, storage, self, task, content)
}

func (self *SDisk) AllowPerformResize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred)
}

func (self *SDisk) PerformResize(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if sizeStr, err := data.GetString("size"); err != nil {
		return nil, err
	} else if size, err := fileutils.GetSizeMb(sizeStr, 'M', 1024); err != nil {
		return nil, err
	} else if self.Status != DISK_READY {
		return nil, httperrors.NewResourceNotReadyError("Resize disk when disk is READY")
	} else if size < self.DiskSize {
		return nil, httperrors.NewUnsupportOperationError("Disk cannot be thrink")
	} else if size == self.DiskSize {
		return nil, nil
	} else {
		addDisk := size - self.DiskSize
		storage := self.GetStorage()
		if addDisk > storage.GetFreeCapacity() && !storage.IsEmulated {
			return nil, httperrors.NewOutOfResourceError("Not enough free space")
		}
		pendingUsage := SQuota{Storage: int(addDisk)}
		if err := QuotaManager.CheckSetPendingQuota(ctx, userCred, userCred.GetProjectId(), &pendingUsage); err != nil {
			return nil, httperrors.NewOutOfQuotaError(err.Error())
		}
		return nil, self.StartDiskResizeTask(ctx, userCred, int64(size), "", &pendingUsage)
	}
}

func (self *SDisk) ValidateDeleteCondition(ctx context.Context) error {
	if self.GetGuestDiskCount() > 0 {
		return httperrors.NewNotEmptyError("Virtual disk used by virtual servers")
	}
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SDisk) GetTemplateId() string {
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

func (self *SDisk) GetPathAtHost(host *SHost) string {
	storage := self.GetStorage()
	if storage.StorageType == STORAGE_RBD {
		pool, _ := storage.StorageConf.GetString("pool")
		monHost, _ := storage.StorageConf.GetString("mon_host")
		key, _ := storage.StorageConf.GetString("key")
		for _, keyword := range []string{"@", ":", "="} {
			monHost = strings.Replace(monHost, keyword, fmt.Sprintf("\\%s", keyword), -1)
			key = strings.Replace(key, keyword, fmt.Sprintf("\\%s", keyword), -1)
		}
		return fmt.Sprintf("rbd:%s/%s:mon_host=%s:key=%s", pool, self.Id, monHost, key)
	} else if storage.StorageType == STORAGE_LOCAL || storage.StorageType == STORAGE_NAS {
		hostStorage := host.GetHoststorageOfId(self.StorageId)
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

func (manager *SDiskManager) syncCloudDisk(userCred mcclient.TokenCredential, vdisk cloudprovider.ICloudDisk) (*SDisk, error) {
	diskObj, err := manager.FetchByExternalId(vdisk.GetGlobalId())
	if err != nil {
		if err == sql.ErrNoRows {
			vstorage := vdisk.GetIStorge()
			storageObj, err := StorageManager.FetchByExternalId(vstorage.GetGlobalId())
			if err != nil {
				log.Errorf("cannot find storage of vdisk %s", err)
				return nil, err
			}
			storage := storageObj.(*SStorage)
			return manager.newFromCloudDisk(userCred, vdisk, storage)
		} else {
			return nil, err
		}
	} else {
		disk := diskObj.(*SDisk)
		err = disk.syncWithCloudDisk(userCred, vdisk)
		if err != nil {
			return nil, err
		}
		return disk, nil
	}
}

func (manager *SDiskManager) SyncDisks(ctx context.Context, userCred mcclient.TokenCredential, storage *SStorage, disks []cloudprovider.ICloudDisk) ([]SDisk, []cloudprovider.ICloudDisk, compare.SyncResult) {
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
		removed[i].SetStatus(userCred, DISK_UNKNOWN, "missing original disk after sync")
		if err != nil { // cannot delete
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudDisk(userCred, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localDisks = append(localDisks, commondb[i])
			remoteDisks = append(remoteDisks, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudDisk(userCred, added[i], storage)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localDisks = append(localDisks, *new)
			remoteDisks = append(remoteDisks, added[i])
			syncResult.Add()
		}
	}

	return localDisks, remoteDisks, syncResult
}

func (self *SDisk) syncWithCloudDisk(userCred mcclient.TokenCredential, extDisk cloudprovider.ICloudDisk) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		extDisk.Refresh()
		self.Name = extDisk.GetName()
		self.Status = extDisk.GetStatus()
		self.DiskFormat = extDisk.GetDiskFormat()
		self.DiskSize = extDisk.GetDiskSizeMB()
		self.AutoDelete = extDisk.GetIsAutoDelete()
		// self.TemplateId = extDisk.GetTemplateId() no sync template ID
		self.DiskType = extDisk.GetDiskType()
		// self.FsFormat = extDisk.GetFsFormat()
		self.Nonpersistent = extDisk.GetIsNonPersistent()

		self.IsEmulated = extDisk.IsEmulated()

		self.ProjectId = userCred.GetProjectId()

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudDisk error %s", err)
	}
	return err
}

func (manager *SDiskManager) newFromCloudDisk(userCred mcclient.TokenCredential, extDisk cloudprovider.ICloudDisk, storage *SStorage) (*SDisk, error) {
	disk := SDisk{}
	disk.SetModelManager(manager)

	disk.Name = extDisk.GetName()
	disk.Status = extDisk.GetStatus()
	disk.ExternalId = extDisk.GetGlobalId()
	disk.StorageId = storage.Id
	disk.ProjectId = userCred.GetProjectId()

	disk.DiskFormat = extDisk.GetDiskFormat()
	disk.DiskSize = extDisk.GetDiskSizeMB()
	disk.AutoDelete = extDisk.GetIsAutoDelete()
	disk.DiskType = extDisk.GetDiskType()
	disk.Nonpersistent = extDisk.GetIsNonPersistent()

	disk.IsEmulated = extDisk.IsEmulated()

	err := manager.TableSpec().Insert(&disk)
	if err != nil {
		log.Errorf("newFromCloudZone fail %s", err)
		return nil, err
	}
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
			q = q.Filter(sqlchemy.Equals(storages.Field("status"), STORAGE_ENABLED))
		} else {
			q = q.Filter(sqlchemy.NotEquals(storages.Field("status"), STORAGE_ENABLED))
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
	ImageId string
	// ImageDiskFormat string
	Size            int    // MB
	Fs              string // file system
	Format          string //
	Driver          string //
	Cache           string //
	Mountpoint      string //
	Backend         string // stroageType
	Medium          string
	ImageProperties map[string]string
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
	diskConfig.Backend = STORAGE_LOCAL
	diskConfig.Medium = DISK_TYPE_HYBRID

	diskStr, err := info.GetString()
	if err != nil {
		log.Errorf("invalid diskinfo format %s", err)
		return nil, err
	}
	parts := strings.Split(diskStr, ":")
	for _, p := range parts {
		if regutils.MatchSize(p) {
			diskConfig.Size, _ = fileutils.GetSizeMb(p, 'M', 1024)
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
		} else if p[0] == '/' {
			diskConfig.Mountpoint = p
		} else if p == "autoextend" {
			diskConfig.Size = -1
		} else if utils.IsInStringArray(p, sysutils.STORAGE_TYPES) {
			diskConfig.Backend = p
		} else if len(p) > 0 {
			if userCred == nil {
				diskConfig.ImageId = p
			} else {
				image, err := CachedimageManager.getImageInfo(ctx, userCred, p, false)
				if err != nil {
					log.Errorf("getImageInfo fail %s", err)
					return nil, err
				}
				if image.Status != IMAGE_STATUS_ACTIVE {
					return nil, httperrors.NewInvalidStatusError("Image status is not active")
				}
				diskConfig.ImageId = image.Id
				diskConfig.ImageProperties = image.Properties
				if len(diskConfig.Format) == 0 {
					diskConfig.Format = image.DiskFormat
				}
				// diskConfig.ImageDiskFormat = image.DiskFormat
				CachedimageManager.ImageAddRefCount(image.Id)
				if diskConfig.Size == 0 {
					diskConfig.Size = image.MinDisk // MB
				}
			}
		}
	}
	if len(diskConfig.ImageId) > 0 && diskConfig.Size == 0 {
		diskConfig.Size = options.Options.DefaultDiskSize // MB
	} else if len(diskConfig.ImageId) == 0 && diskConfig.Size == 0 {
		return nil, httperrors.NewInputParameterError("Diskinfo not contains either imageID or size")
	}
	return &diskConfig, nil
}

func parseIsoInfo(ctx context.Context, userCred mcclient.TokenCredential, info string) (string, error) {
	image, err := CachedimageManager.getImageInfo(ctx, userCred, info, false)
	if err != nil {
		log.Errorf("getImageInfo fail %s", err)
		return "", err
	}
	if image.Status != IMAGE_STATUS_ACTIVE {
		return "", httperrors.NewInvalidStatusError("Image status is not active")
	}
	return image.Id, nil
}

func (self *SDisk) fetchDiskInfo(diskConfig *SDiskConfig) {
	if len(diskConfig.ImageId) > 0 {
		self.TemplateId = diskConfig.ImageId
		self.DiskType = DISK_TYPE_SYS
	}
	if len(diskConfig.Fs) > 0 {
		self.FsFormat = diskConfig.Fs
	}
	if self.FsFormat == "swap" {
		self.DiskType = DISK_TYPE_SWAP
		self.Nonpersistent = true
	} else {
		if len(self.DiskType) == 0 {
			self.DiskType = DISK_TYPE_DATA
		}
		self.Nonpersistent = false
	}
	self.DiskFormat = diskConfig.Format
	self.DiskSize = diskConfig.Size
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
	return self.IsOwner(userCred)
}

func (self *SDisk) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := self.ValidateDeleteCondition(ctx)
	if err != nil {
		return nil, err
	}
	return nil, self.StartDiskDeleteTask(ctx, userCred, "", true)
}

func (self *SDisk) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDiskDeleteTask(ctx, userCred, "", false)
}

func (self *SDisk) StartDiskResizeTask(ctx context.Context, userCred mcclient.TokenCredential, size int64, parentTaskId string, pendingUsage quotas.IQuota) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewInt(size), "size")
	if task, err := taskman.TaskManager.NewTask(ctx, "DiskResizeTask", self, userCred, params, parentTaskId, "", pendingUsage); err != nil {
		return err
	} else {
		task.ScheduleRun(nil)
	}
	return nil
}

func (self *SDisk) StartDiskDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, isPurge bool) error {
	params := jsonutils.NewDict()
	if isPurge {
		params.Add(jsonutils.JSONTrue, "purge")
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
	err := q.All(&ret)
	if err != nil {
		log.Errorf("%s", err)
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

func (self *SDisk) GetShortDesc() *jsonutils.JSONDict {
	desc := self.SSharableVirtualResourceBase.GetShortDesc()
	desc.Add(jsonutils.NewInt(int64(self.DiskSize)), "size")
	storage := self.GetStorage()
	desc.Add(jsonutils.NewString(storage.StorageType), "storage_type")
	desc.Add(jsonutils.NewString(storage.MediumType), "medium_type")
	fs := self.GetFsFormat()
	if len(fs) > 0 {
		desc.Add(jsonutils.NewString(fs), "fs_format")
	}
	tid := self.GetTemplateId()
	if len(tid) > 0 {
		desc.Add(jsonutils.NewString(tid), "template_id")
	}
	return desc
}

func (self *SDisk) getDev() string {
	return self.GetMetadata("dev", nil)
}

func (self *SDisk) isReady() bool {
	return self.Status == DISK_READY
}

func (self *SDisk) isInit() bool {
	return self.Status == DISK_INIT
}

func (model *SDisk) AllowPerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SDisk) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.PendingDeleted {
		err := self.DoCancelPendingDelete(ctx, userCred)
		return nil, err
	}
	return nil, nil
}

func (manager *SDiskManager) getExpiredPendingDeleteDisks() []SDisk {
	deadline := time.Now().Add(time.Duration(options.Options.PendingDeleteExpireSeconds) * time.Second)

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

func (manager *SDiskManager) CleanPendingDeleteDisks(ctx context.Context, userCred mcclient.TokenCredential) {
	disks := manager.getExpiredPendingDeleteDisks()
	if disks == nil {
		return
	}
	for i := 0; i < len(disks); i += 1 {
		disks[i].StartDiskDeleteTask(ctx, userCred, "", false)
	}
}
