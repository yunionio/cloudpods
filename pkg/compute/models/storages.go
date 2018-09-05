package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"
)

const (
	STORAGE_LOCAL        = "local"
	STORAGE_BAREMETAL    = "baremetal"
	STORAGE_SHEEPDOG     = "sheepdog"
	STORAGE_RBD          = "rbd"
	STORAGE_DOCKER       = "docker"
	STORAGE_NAS          = "nas"
	STORAGE_VSAN         = "vsan"
	STORAGE_PUBLIC_CLOUD = "cloud"

	STORAGE_ENABLED  = "enabled"
	STORAGE_DISABLED = "disabled"
	STORAGE_OFFLINE  = "offline"
	STORAGE_ONLINE   = "offline"

	DISK_TYPE_ROTATE = "rotate"
	DISK_TYPE_SSD    = "ssd"
	DISK_TYPE_HYBRID = "hybrid"
)

var (
	DISK_TYPES            = []string{DISK_TYPE_ROTATE, DISK_TYPE_SSD, DISK_TYPE_HYBRID}
	STORAGE_LOCAL_TYPES   = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_NAS}
	STORAGE_SUPPORT_TYPES = STORAGE_LOCAL_TYPES
	STORAGE_ALL_TYPES     = []string{
		STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_SHEEPDOG,
		STORAGE_RBD, STORAGE_DOCKER, STORAGE_NAS, STORAGE_VSAN,
	}
)

type SStorageManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	SInfrastructureManager
}

var StorageManager *SStorageManager

func init() {
	StorageManager = &SStorageManager{SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(SStorage{},
		"storages_tbl", "storage", "storages")}
}

type SStorage struct {
	db.SEnabledStatusStandaloneResourceBase
	SInfrastructure
	SManagedResourceBase

	Capacity    int                  `nullable:"false" list:"admin" update:"admin" create:"admin_required"`                           // Column(Integer, nullable=False) # capacity of disk in MB
	Reserved    int                  `nullable:"true" default:"0" list:"admin" update:"admin"`                                        // Column(Integer, nullable=True, default=0)
	StorageType string               `width:"32" charset:"ascii" nullable:"false" list:"user" update:"admin" create:"admin_required"` // Column(VARCHAR(32, charset='ascii'), nullable=False)
	MediumType  string               `width:"32" charset:"ascii" nullable:"false" list:"user" update:"admin" create:"admin_required"` // Column(VARCHAR(32, charset='ascii'), nullable=False)
	Cmtbound    float32              `nullable:"true" list:"admin" update:"admin"`                                                    // Column(Float, nullable=True)
	StorageConf jsonutils.JSONObject `nullable:"true" get:"admin" update:"admin"`                                                     // = Column(JSONEncodedDict, nullable=True)

	ZoneId string `width:"36" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"`

	StoragecacheId string `width:"36" charset:"ascii" nullable:"true" get:"admin"`
}

func (manager *SStorageManager) GetContextManager() []db.IModelManager {
	return []db.IModelManager{ZoneManager, StoragecacheManager}
}

func (self *SStorage) ValidateDeleteCondition(ctx context.Context) error {
	if self.GetHostCount() > 0 || self.GetDiskCount() > 0 {
		return httperrors.NewNotEmptyError("Not an empty storage provider")
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SStorage) GetHostCount() int {
	return HoststorageManager.Query().Equals("storage_id", self.Id).Count()
}

func (self *SStorage) GetDiskCount() int {
	return DiskManager.Query().Equals("storage_id", self.Id).Count()
}

func (self *SStorage) IsLocal() bool {
	return self.StorageType == STORAGE_LOCAL || self.StorageType == STORAGE_BAREMETAL
}

func (manager *SStorageManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SStorage) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	used := self.GetUsedCapacity(tristate.True)
	waste := self.GetUsedCapacity(tristate.False)
	vcapa := float32(self.GetCapacity()) * self.GetOvercommitBound()
	extra.Add(jsonutils.NewInt(int64(used)), "used_capacity")
	extra.Add(jsonutils.NewInt(int64(waste)), "waste_capacity")
	extra.Add(jsonutils.NewFloat(float64(vcapa)), "virtual_capacity")
	extra.Add(jsonutils.NewFloat(float64(vcapa-float32(used)-float32(waste))), "free_capacity")
	if self.GetCapacity() > 0 {
		value := float64(used * 1.0 / self.GetCapacity())
		value = float64(int(value*100+0.5) / 100.0)
		extra.Add(jsonutils.NewFloat(value), "commit_rate")
	} else {
		extra.Add(jsonutils.NewFloat(0.0), "commit_rate")
	}
	extra.Add(jsonutils.NewFloat(float64(self.GetOvercommitBound())), "commit_bound")
	return extra
}

func (self *SStorage) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SStorage) GetUsedCapacity(isReady tristate.TriState) int {
	disks := DiskManager.Query().SubQuery()
	q := disks.Query(sqlchemy.SUM("sum", disks.Field("disk_size"))).Equals("storage_id", self.Id)
	switch isReady {
	case tristate.True:
		q = q.Equals("status", DISK_READY)
	case tristate.False:
		q = q.NotEquals("status", DISK_READY)
	}
	if q.Count() == 0 {
		return 0
	}
	row := q.Row()
	var sum int
	err := row.Scan(&sum)
	if err != nil {
		log.Errorf("GetUsedCapacity fail: %s", err)
		return 0
	}
	return sum
}

func (self *SStorage) GetOvercommitBound() float32 {
	if self.Cmtbound > 0 {
		return self.Cmtbound
	} else {
		return options.Options.DefaultStorageOvercommitBound
	}
}

func (self *SStorage) GetMasterHost() *SHost {
	hosts := HostManager.Query().SubQuery()
	hoststorages := HoststorageManager.Query().SubQuery()

	q := hosts.Query().Join(hoststorages, sqlchemy.AND(sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")),
		sqlchemy.IsFalse(hoststorages.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), self.Id))
	q = q.IsTrue("enabled")
	q = q.Equals("host_status", HOST_ONLINE).Asc("id")
	host := SHost{}
	host.SetModelManager(HostManager)
	err := q.First(&host)
	if err != nil {
		log.Errorf("GetMasterHost fail %s", err)
		return nil
	}
	return &host
}

func (self *SStorage) GetZoneId() string {
	if len(self.ZoneId) > 0 {
		return self.ZoneId
	}
	host := self.GetMasterHost()
	if host != nil {
		_, err := StorageManager.TableSpec().Update(self, func() error {
			self.ZoneId = host.ZoneId
			return nil
		})
		if err != nil {
			log.Errorf("%s", err)
			return ""
		}
		return self.ZoneId
	} else {
		log.Errorf("No mater host for storage")
		return ""
	}
}

func (self *SStorage) getZone() *SZone {
	zoneId := self.GetZoneId()
	if len(zoneId) > 0 {
		return ZoneManager.FetchZoneById(zoneId)
	}
	return nil
}

func (self *SStorage) GetReserved() int {
	return self.Reserved
}

func (self *SStorage) GetCapacity() int {
	return self.Capacity - self.GetReserved()
}

func (self *SStorage) GetFreeCapacity() int {
	return int(float32(self.GetCapacity())*self.GetOvercommitBound()) - self.GetUsedCapacity(tristate.None)
}

func (self *SStorage) GetAttachedHosts() []SHost {
	hosts := HostManager.Query().SubQuery()
	hoststorages := HoststorageManager.Query().SubQuery()

	q := hosts.Query()
	q = q.Join(hoststorages, sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), self.Id))

	hostList := make([]SHost, 0)
	err := db.FetchModelObjects(HostManager, q, &hostList)
	if err != nil {
		log.Errorf("GetAttachedHosts fail %s", err)
		return nil
	}
	return hostList
}

func (self *SStorage) SyncStatusWithHosts() {
	hosts := self.GetAttachedHosts()
	if hosts == nil {
		return
	}
	total := 0
	online := 0
	offline := 0
	for _, h := range hosts {
		if h.HostStatus == HOST_ONLINE {
			online += 1
		} else {
			offline += 1
		}
		total += 1
	}
	var status string
	if !self.IsLocal() {
		status = STORAGE_ENABLED
	} else if online > 0 {
		status = STORAGE_ENABLED
	} else if offline > 0 {
		status = STORAGE_OFFLINE
	} else {
		status = STORAGE_DISABLED
	}
	if status != self.Status {
		self.SetStatus(nil, status, "SyncStatusWithHosts")
	}
}

func (manager *SStorageManager) getStoragesByZoneId(zoneId string) ([]SStorage, error) {
	storages := make([]SStorage, 0)
	q := manager.Query().Equals("zone_id", zoneId)
	err := db.FetchModelObjects(manager, q, &storages)
	if err != nil {
		log.Errorf("getStoragesByZoneId fail %s", err)
		return nil, err
	}
	return storages, nil
}

func (manager *SStorageManager) scanLegacyStorages() error {
	storages := make([]SStorage, 0)
	table := manager.Query().SubQuery()
	q := table.Query().Filter(sqlchemy.OR(sqlchemy.IsNull(table.Field("zone_id")), sqlchemy.IsEmpty(table.Field("zone_id"))))
	err := db.FetchModelObjects(manager, q, &storages)
	if err != nil {
		log.Errorf("getLegacyStoragesByZoneId fail %s", err)
		return err
	}
	for i := 0; i < len(storages); i += 1 {
		storages[i].GetZoneId()
	}
	return nil
}

func (manager *SStorageManager) SyncStorages(ctx context.Context, userCred mcclient.TokenCredential, zone *SZone, storages []cloudprovider.ICloudStorage) ([]SStorage, []cloudprovider.ICloudStorage, compare.SyncResult) {
	localStorages := make([]SStorage, 0)
	remoteStorages := make([]cloudprovider.ICloudStorage, 0)
	syncResult := compare.SyncResult{}

	err := manager.scanLegacyStorages()
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	dbStorages, err := manager.getStoragesByZoneId(zone.Id)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SStorage, 0)
	commondb := make([]SStorage, 0)
	commonext := make([]cloudprovider.ICloudStorage, 0)
	added := make([]cloudprovider.ICloudStorage, 0)

	err = compare.CompareSets(dbStorages, storages, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].ValidateDeleteCondition(ctx)
		if err != nil { // cannot delete
			err = removed[i].SetStatus(userCred, STORAGE_DISABLED, "sync to delete")
			if err == nil {
				_, err = removed[i].PerformDisable(ctx, userCred, nil, nil)
			}
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		} else {
			err = removed[i].Delete(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		}
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].syncWithCloudStorage(commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localStorages = append(localStorages, commondb[i])
			remoteStorages = append(remoteStorages, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudStorage(added[i], zone)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localStorages = append(localStorages, *new)
			remoteStorages = append(remoteStorages, added[i])
			syncResult.Add()
		}
	}

	return localStorages, remoteStorages, syncResult
}

func (self *SStorage) syncWithCloudStorage(extStorage cloudprovider.ICloudStorage) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Name = extStorage.GetName()
		self.Status = extStorage.GetStatus()
		self.StorageType = extStorage.GetStorageType()
		self.MediumType = extStorage.GetMediumType()
		self.Capacity = extStorage.GetCapacityMB()
		self.StorageConf = extStorage.GetStorageConf()

		self.Enabled = extStorage.GetEnabled()

		self.IsEmulated = extStorage.IsEmulated()
		self.ManagerId = extStorage.GetManagerId()

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
	}
	return err
}

func (manager *SStorageManager) newFromCloudStorage(extStorage cloudprovider.ICloudStorage, zone *SZone) (*SStorage, error) {
	storage := SStorage{}
	storage.SetModelManager(manager)

	storage.Name = extStorage.GetName()
	storage.Status = extStorage.GetStatus()
	storage.ExternalId = extStorage.GetGlobalId()
	storage.ZoneId = zone.Id
	storage.StorageType = extStorage.GetStorageType()
	storage.MediumType = extStorage.GetMediumType()
	storage.StorageConf = extStorage.GetStorageConf()
	storage.Capacity = extStorage.GetCapacityMB()

	storage.Enabled = extStorage.GetEnabled()

	storage.IsEmulated = extStorage.IsEmulated()
	storage.ManagerId = extStorage.GetManagerId()

	err := manager.TableSpec().Insert(&storage)
	if err != nil {
		log.Errorf("newFromCloudStorage fail %s", err)
		return nil, err
	}
	return &storage, nil
}

type StorageCapacityStat struct {
	TotalSize        int64
	TotalSizeVirtual float64
}

func (manager *SStorageManager) disksReadyQ() *sqlchemy.SSubQuery {
	disks := DiskManager.Query().SubQuery()
	q := disks.Query(
		disks.Field("storage_id"),
		sqlchemy.SUM("used_capacity", disks.Field("disk_size")),
	).Equals("status", DISK_READY).GroupBy(disks.Field("storage_id")).SubQuery()
	return q
}

func (manager *SStorageManager) disksFailedQ() *sqlchemy.SSubQuery {
	disks := DiskManager.Query().SubQuery()
	q := disks.Query(
		disks.Field("storage_id"),
		sqlchemy.SUM("failed_capacity", disks.Field("disk_size")),
	).NotEquals("status", DISK_READY).GroupBy(disks.Field("storage_id")).SubQuery()
	return q
}

func (manager *SStorageManager) totalCapacityQ(
	rangeObj db.IStandaloneModel, hostTypes []string,
) *sqlchemy.SQuery {
	stmt := manager.disksReadyQ()
	stmt2 := manager.disksFailedQ()
	storages := manager.Query().SubQuery()
	q := storages.Query(
		storages.Field("capacity"),
		storages.Field("reserved"),
		storages.Field("cmtbound"),
		stmt.Field("used_capacity"),
		stmt2.Field("failed_capacity")).
		LeftJoin(stmt, sqlchemy.Equals(stmt.Field("storage_id"), storages.Field("id"))).
		LeftJoin(stmt2, sqlchemy.Equals(stmt2.Field("storage_id"), storages.Field("id")))

	hosts := HostManager.Query().SubQuery()
	hostStorages := HoststorageManager.Query().SubQuery()

	q = q.Join(hostStorages, sqlchemy.AND(
		sqlchemy.Equals(hostStorages.Field("storage_id"), storages.Field("id")),
		sqlchemy.IsFalse(hostStorages.Field("deleted")),
	)).Join(
		hosts, sqlchemy.AND(
			sqlchemy.Equals(hosts.Field("id"), hostStorages.Field("host_id")),
			sqlchemy.IsFalse(hosts.Field("deleted")),
			sqlchemy.IsTrue(hosts.Field("enabled")),
			sqlchemy.Equals(hosts.Field("host_status"), HOST_ONLINE)))

	return AttachUsageQuery(q, hosts, hostStorages.Field("host_id"), hostTypes, rangeObj)
}

type StorageStat struct {
	Capacity       int
	Reserved       int
	Cmtbound       float32
	UsedCapacity   int
	FailedCapacity int
}

type StoragesCapacityStat struct {
	Capacity        int64
	CapacityVirtual float64
	CapacityUsed    int64
	CapacityUnread  int64
}

func (manager *SStorageManager) calculateCapacity(q *sqlchemy.SQuery) StoragesCapacityStat {
	stats := make([]StorageStat, 0)
	err := q.All(&stats)
	if err != nil {
		log.Errorf("calculateCapacity: %v", err)
	}
	var (
		tCapa   int64   = 0
		tVCapa  float64 = 0
		tUsed   int64   = 0
		tFailed int64   = 0
	)
	for _, stat := range stats {
		tCapa += int64(stat.Capacity - stat.Reserved)
		if stat.Cmtbound == 0 {
			stat.Cmtbound = options.Options.DefaultStorageOvercommitBound
		}
		tVCapa += float64(stat.Capacity-stat.Reserved) * float64(stat.Cmtbound)
		tUsed += int64(stat.UsedCapacity)
		tFailed += int64(stat.FailedCapacity)
	}
	return StoragesCapacityStat{
		Capacity:        tCapa,
		CapacityVirtual: tVCapa,
		CapacityUsed:    tUsed,
		CapacityUnread:  tFailed,
	}
}

func (manager *SStorageManager) TotalCapacity(rangeObj db.IStandaloneModel, hostTypes []string) StoragesCapacityStat {
	res1 := manager.calculateCapacity(manager.totalCapacityQ(rangeObj, hostTypes))
	return res1
}

func (self *SStorage) createDisk(name string, diskConfig *SDiskConfig, userCred mcclient.TokenCredential, ownerProjId string, autoDelete bool, isSystem bool) (*SDisk, error) {
	disk := SDisk{}
	disk.SetModelManager(DiskManager)

	disk.Name = name
	disk.fetchDiskInfo(diskConfig)

	disk.StorageId = self.Id
	disk.AutoDelete = autoDelete
	disk.ProjectId = ownerProjId
	disk.IsSystem = isSystem

	err := disk.GetModelManager().TableSpec().Insert(&disk)
	if err != nil {
		return nil, err
	}
	db.OpsLog.LogEvent(&disk, db.ACT_CREATE, nil, userCred)
	return &disk, nil
}

func (self *SStorage) GetAllAttachingHosts() []SHost {
	hosts := HostManager.Query().SubQuery()
	hoststorages := HoststorageManager.Query().SubQuery()

	q := hosts.Query()
	q = q.Join(hoststorages, sqlchemy.AND(sqlchemy.Equals(hoststorages.Field("host_id"), hosts.Field("id")),
		sqlchemy.IsFalse(hoststorages.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("storage_id"), self.Id))
	q = q.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))
	q = q.Filter(sqlchemy.Equals(hosts.Field("host_status"), HOST_ONLINE))

	ret := make([]SHost, 0)
	err := q.All(&ret)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return ret
}

func (self *SStorage) SetStoragecache(cache *SStoragecache) error {
	if self.StoragecacheId == cache.Id {
		return nil
	}
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.StoragecacheId = cache.Id
		return nil
	})
	return err
}

func (self *SStorage) AllowPerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SStorage) GetStoragecache() *SStoragecache {
	obj, err := StoragecacheManager.FetchById(self.StoragecacheId)
	if err != nil {
		log.Errorf("cannot find storage cache??? %s", err)
		return nil
	}
	return obj.(*SStoragecache)
}

func (self *SStorage) PerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cache := self.GetStoragecache()
	if cache == nil {
		return nil, httperrors.NewInternalServerError("storage cache is missing")
	}

	return cache.PerformCacheImage(ctx, userCred, query, data)
}

func (self *SStorage) AllowPerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SStorage) PerformUncacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cache := self.GetStoragecache()
	if cache == nil {
		return nil, httperrors.NewInternalServerError("storage cache is missing")
	}

	return cache.PerformUncacheImage(ctx, userCred, query, data)
}

func (self *SStorage) GetIStorage() (cloudprovider.ICloudStorage, error) {
	provider, err := self.GetDriver()
	if err != nil {
		log.Errorf("fail to find cloud provider")
		return nil, err
	}
	return provider.GetIStorageById(self.GetExternalId())
}

func (manager *SStorageManager) FetchStorageById(storageId string) *SStorage {
	obj, err := manager.FetchById(storageId)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return obj.(*SStorage)
}

func (manager *SStorageManager) InitializeData() error {
	storages := make([]SStorage, 0)
	q := manager.Query()
	err := db.FetchModelObjects(manager, q, &storages)
	if err != nil {
		return err
	}
	for _, s := range storages {
		if len(s.ZoneId) == 0 {
			zoneId := ""
			hosts := s.GetAttachedHosts()
			if hosts != nil && len(hosts) > 0 {
				zoneId = hosts[0].ZoneId
			} else {
				log.Fatalf("Cannot locate zoneId for storage %s", s.Name)
			}
			manager.TableSpec().Update(&s, func() error {
				s.ZoneId = zoneId
				return nil
			})
		}
		if len(s.StoragecacheId) == 0 && s.StorageType == STORAGE_RBD {
			storagecache := &SStoragecache{}
			storagecache.SetModelManager(StoragecacheManager)
			storagecache.Name = "rbd-" + s.Id
			if pool, err := s.StorageConf.GetString("pool"); err != nil {
				log.Fatalf("Get storage %s pool info error", s.Name)
			} else {
				storagecache.Path = "rbd:" + pool
				if err := StoragecacheManager.TableSpec().Insert(storagecache); err != nil {
					log.Fatalf("Cannot Add storagecache for %s", s.Name)
				} else {
					manager.TableSpec().Update(&s, func() error {
						s.StoragecacheId = storagecache.Id
						return nil
					})
				}
			}
		}
	}
	return nil
}

func (manager *SStorageManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	regionStr, _ := query.GetString("region")
	if len(regionStr) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(userCred.GetProjectId(), regionStr)
		if err != nil {
			return nil, httperrors.NewNotFoundError("Region %s not found: %s", regionStr, err)
		}
		sq := ZoneManager.Query("id").Equals("cloudregion_id", regionObj.GetId())
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), sq.SubQuery()))
	}

	managerStr := jsonutils.GetAnyString(query, []string{"manager", "provider", "manager_id", "provider_id"})
	if len(managerStr) > 0 {
		provider := CloudproviderManager.FetchCloudproviderByIdOrName(managerStr)
		if provider == nil {
			return nil, httperrors.NewResourceNotFoundError("provider %s not found", managerStr)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("manager_id"), provider.GetId()))
	}

	return q, err
}
