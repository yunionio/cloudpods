package models

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

const (
	STORAGE_LOCAL     = "local"
	STORAGE_BAREMETAL = "baremetal"
	STORAGE_SHEEPDOG  = "sheepdog"
	STORAGE_RBD       = "rbd"
	STORAGE_DOCKER    = "docker"
	STORAGE_NAS       = "nas"
	STORAGE_VSAN      = "vsan"
	STORAGE_NFS       = "nfs"

	STORAGE_PUBLIC_CLOUD     = "cloud"
	STORAGE_CLOUD_EFFICIENCY = "cloud_efficiency"
	STORAGE_CLOUD_SSD        = "cloud_ssd"
	STORAGE_CLOUD_ESSD       = "cloud_essd"    //增强型(Enhanced)SSD 云盘
	STORAGE_EPHEMERAL_SSD    = "ephemeral_ssd" //单块本地SSD盘, 容量最大不能超过800 GiB

	//Azure hdd and ssd storagetype
	STORAGE_STANDARD_LRS    = "standard_lrs"
	STORAGE_STANDARDSSD_LRS = "standardssd_lrs"
	STORAGE_PREMIUM_LRS     = "premium_lrs"

	// aws storage type
	STORAGE_GP2_SSD      = "gp2"      // aws general purpose ssd
	STORAGE_IO1_SSD      = "io1"      // aws Provisioned IOPS SSD
	STORAGE_ST1_HDD      = "st1"      // aws Throughput Optimized HDD
	STORAGE_SC1_SSD      = "sc1"      // aws Cold HDD
	STORAGE_STANDARD_SSD = "standard" // aws Magnetic volumes

	// qcloud storage type
	// STORAGE_CLOUD_SSD ="cloud_ssd"
	STORAGE_LOCAL_BASIC   = "local_basic"
	STORAGE_LOCAL_SSD     = "local_ssd"
	STORAGE_CLOUD_BASIC   = "cloud_basic"
	STORAGE_CLOUD_PREMIUM = "cloud_premium"
)

const (
	STORAGE_ENABLED  = "enabled"
	STORAGE_DISABLED = "disabled"
	STORAGE_OFFLINE  = "offline"
	STORAGE_ONLINE   = "online"

	DISK_TYPE_ROTATE = "rotate"
	DISK_TYPE_SSD    = "ssd"
	DISK_TYPE_HYBRID = "hybrid"
)

var (
	DISK_TYPES            = []string{DISK_TYPE_ROTATE, DISK_TYPE_SSD, DISK_TYPE_HYBRID}
	STORAGE_LOCAL_TYPES   = []string{STORAGE_LOCAL, STORAGE_BAREMETAL}
	STORAGE_SUPPORT_TYPES = STORAGE_LOCAL_TYPES
	STORAGE_ALL_TYPES     = []string{
		STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_SHEEPDOG,
		STORAGE_RBD, STORAGE_DOCKER, STORAGE_NAS, STORAGE_VSAN,
		STORAGE_NFS,
	}
	STORAGE_TYPES = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_SHEEPDOG,
		STORAGE_RBD, STORAGE_DOCKER, STORAGE_NAS, STORAGE_VSAN, STORAGE_NFS,
		STORAGE_PUBLIC_CLOUD, STORAGE_CLOUD_SSD, STORAGE_CLOUD_ESSD, STORAGE_EPHEMERAL_SSD, STORAGE_CLOUD_EFFICIENCY,
		STORAGE_STANDARD_LRS, STORAGE_STANDARDSSD_LRS, STORAGE_PREMIUM_LRS,
		STORAGE_GP2_SSD, STORAGE_IO1_SSD, STORAGE_ST1_HDD, STORAGE_SC1_SSD, STORAGE_STANDARD_SSD,
		STORAGE_LOCAL_BASIC, STORAGE_LOCAL_SSD, STORAGE_CLOUD_BASIC, STORAGE_CLOUD_PREMIUM,
	}

	STORAGE_LIMITED_TYPES = []string{STORAGE_LOCAL, STORAGE_BAREMETAL, STORAGE_NAS, STORAGE_RBD, STORAGE_NFS}
)

type SStorageManager struct {
	db.SStandaloneResourceBaseManager
}

var StorageManager *SStorageManager

func init() {
	StorageManager = &SStorageManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SStorage{},
			"storages_tbl",
			"storage",
			"storages",
		),
	}
}

type SStorage struct {
	db.SStandaloneResourceBase
	SManagedResourceBase

	Capacity    int                  `nullable:"false" list:"admin" update:"admin" create:"admin_required"`                           // Column(Integer, nullable=False) # capacity of disk in MB
	Reserved    int                  `nullable:"true" default:"0" list:"admin" update:"admin"`                                        // Column(Integer, nullable=True, default=0)
	StorageType string               `width:"32" charset:"ascii" nullable:"false" list:"user" update:"admin" create:"admin_required"` // Column(VARCHAR(32, charset='ascii'), nullable=False)
	MediumType  string               `width:"32" charset:"ascii" nullable:"false" list:"user" update:"admin" create:"admin_required"` // Column(VARCHAR(32, charset='ascii'), nullable=False)
	Cmtbound    float32              `nullable:"true" list:"admin" update:"admin"`                                                    // Column(Float, nullable=True)
	StorageConf jsonutils.JSONObject `nullable:"true" get:"admin" update:"admin"`                                                     // = Column(JSONEncodedDict, nullable=True)

	ZoneId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"admin_required"`

	StoragecacheId string `width:"36" charset:"ascii" nullable:"true" list:"admin" get:"admin"`

	Enabled bool   `nullable:"false" default:"true" list:"user" create:"optional"`
	Status  string `width:"36" charset:"ascii" nullable:"false" default:"offline" list:"user" create:"optional"`
}

func (manager *SStorageManager) GetContextManager() []db.IModelManager {
	return []db.IModelManager{ZoneManager, StoragecacheManager}
}

func (manager *SStorageManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SStorageManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SStorage) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SStorage) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SStorage) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SStorageManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	storageType, _ := data.GetString("storage_type")
	mediumType, _ := data.GetString("medium_type")
	capacity, _ := data.Int("capacity")
	if capacity <= 0 {
		return nil, httperrors.NewInputParameterError("Invalid capacity")
	}
	data.Set("capacity", jsonutils.NewInt(capacity))
	if !utils.IsInStringArray(storageType, STORAGE_TYPES) {
		return nil, httperrors.NewInputParameterError("Invalid storage type %s", storageType)
	}
	if !utils.IsInStringArray(mediumType, DISK_TYPES) {
		return nil, httperrors.NewInputParameterError("Invalid medium type %s", mediumType)
	}
	zoneId, err := data.GetString("zone")
	if err != nil {
		return nil, httperrors.NewMissingParameterError("zone")
	}
	zone, _ := ZoneManager.FetchByIdOrName(userCred, zoneId)
	if zone == nil {
		return nil, httperrors.NewResourceNotFoundError("zone %s", zoneId)
	}
	data.Set("zone_id", jsonutils.NewString(zone.GetId()))
	if storageType == STORAGE_RBD {
		conf, err := manager.ValidateRbdConfData(data)
		if err != nil {
			return nil, httperrors.NewBadRequestError("Vaildata rbd conf error: %s", err.Error())
		}
		data.Set("storage_conf", conf)
		// data.Set("capacity", rbdConf)
	} else if storageType == STORAGE_NFS {
		conf, err := manager.ValidataNfsConfdata(data)
		if err != nil {
			return nil, httperrors.NewBadRequestError("Vaildata nfs conf error: %s", err.Error())
		}
		data.Set("storage_conf", conf)
	}
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (manager *SStorageManager) ValidataNfsConfdata(data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	conf := jsonutils.NewDict()
	if nfsHost, err := data.GetString("nfs_host"); err != nil {
		return nil, httperrors.NewInputParameterError("Get nfs conf host error: %s", err.Error())
	} else {
		conf.Set("nfs_host", jsonutils.NewString(nfsHost))
	}
	if nfsSharedDir, err := data.GetString("nfs_shared_dir"); err != nil {
		return nil, httperrors.NewInputParameterError("Get nfs conf shared dir error: %s", err.Error())
	} else {
		conf.Set("nfs_shared_dir", jsonutils.NewString(nfsSharedDir))
	}
	return conf, nil
}

func (manager *SStorageManager) ValidateRbdConfData(data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	conf := jsonutils.NewDict()
	for k, v := range data.Value() {
		if strings.HasPrefix(k, fmt.Sprintf("%s_", STORAGE_RBD)) {
			k = k[len(STORAGE_RBD)+1:]
			if len(k) > 0 {
				conf.Set(k, v)
			}
		}
	}
	requireFields := []string{"mon_host", "key", "pool"}
	for _, field := range requireFields {
		if !conf.Contains(field) {
			return nil, httperrors.NewMissingParameterError(field)
		}
	}
	storages := make([]SStorage, 0)
	err := manager.Query().Equals("storage_type", STORAGE_RBD).All(&storages)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storages); i++ {
		if conf.Equals(storages[i].StorageConf) {
			return nil, httperrors.NewDuplicateResourceError("This RBD Storage[%s/%s] has already exist", storages[i].Name, conf.String())
		}
	}
	// TODO??? ensure rbd pool can use and get capacity
	return conf, nil
}

func (self *SStorage) ValidateDeleteCondition(ctx context.Context) error {
	if self.GetHostCount() > 0 || self.GetDiskCount() > 0 || self.GetSnapshotCount() > 0 {
		return httperrors.NewNotEmptyError("Not an empty storage provider")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SStorage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SStandaloneResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	storageConf, _ := data.Get("storage_conf")
	if storageConf != nil {
		_, err := self.GetModelManager().TableSpec().Update(self, func() error {
			self.StorageConf = storageConf
			return nil
		})
		if err != nil {
			log.Errorln(err)
			return
		}
	}
	if self.StorageType == STORAGE_RBD {
		var storages = make([]SStorage, 0)
		err := StorageManager.Query().Equals("storage_type", STORAGE_RBD).All(&storages)
		if err != nil {
			log.Errorln(err)
			return
		}
		nMonHost, _ := storageConf.GetString("mon_host")
		nKey, _ := storageConf.GetString("key")
		for i := 0; i < len(storages); i++ {
			monHost, _ := storages[i].StorageConf.GetString("mon_host")
			key, _ := storages[i].StorageConf.GetString("key")
			if monHost == nMonHost && nKey == key {
				_, err := self.GetModelManager().TableSpec().Update(self, func() error {
					self.StoragecacheId = storages[i].StoragecacheId
					return nil
				})
				if err != nil {
					log.Errorln(err)
					return
				}
				break
			}
		}
		if len(self.StoragecacheId) == 0 {
			sc := &SStoragecache{}
			sc.SetModelManager(StoragecacheManager)
			sc.Name = fmt.Sprintf("imagecache-%s", self.Id)
			pool, _ := storageConf.GetString("pool")
			sc.Path = fmt.Sprintf("rbd:%s", pool)
			err := StorageManager.TableSpec().Insert(sc)
			if err != nil {
				log.Errorln(err)
			}
		}
	} else if self.StorageType == STORAGE_NFS {
		sc := &SStoragecache{}
		sc.Path = options.Options.NfsDefaultImageCacheDir
		sc.ExternalId = self.Id
		sc.Name = "nfs-" + self.Name + time.Now().Format("2006-01-02 15:04:05")
		if err := StoragecacheManager.TableSpec().Insert(sc); err != nil {
			log.Errorln(err)
			return
		}
		if err := StoragecacheManager.Query().Equals("external_id", self.Id).First(sc); err != nil {
			log.Errorln(err)
			return
		}
		_, err := self.GetModelManager().TableSpec().Update(self, func() error {
			self.StoragecacheId = sc.Id
			self.Status = STORAGE_ONLINE
			return nil
		})
		if err != nil {
			log.Errorln(err)
		}
	}
}

func (self *SStorage) SetStatus(userCred mcclient.TokenCredential, status string, reason string) error {
	if self.Status == status {
		return nil
	}
	oldStatus := self.Status
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
		self.Status = status
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
		db.OpsLog.LogEvent(self, db.ACT_UPDATE_STATUS, notes, userCred)
		if strings.Contains(notes, "fail") {
			logclient.AddActionLog(self, logclient.ACT_VM_SYNC_STATUS, notes, userCred, false)
		}
	}
	return nil
}

func (self *SStorage) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "enable")
}

func (self *SStorage) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		_, err := self.GetModelManager().TableSpec().Update(self, func() error {
			self.Enabled = true
			return nil
		})
		if err != nil {
			log.Errorf("PerformEnable save update fail %s", err)
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_ENABLE, "", userCred)
	}
	return nil, nil
}

func (self *SStorage) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "disable")
}

func (self *SStorage) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Enabled {
		_, err := self.GetModelManager().TableSpec().Update(self, func() error {
			self.Enabled = false
			return nil
		})
		if err != nil {
			log.Errorf("PerformDisable save update fail %s", err)
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_DISABLE, "", userCred)
	}
	return nil, nil
}

func (self *SStorage) AllowPerformOnline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "online")
}

func (self *SStorage) PerformOnline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != STORAGE_ONLINE {
		err := self.SetStatus(userCred, STORAGE_ONLINE, "")
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_ONLINE, "", userCred)
	}
	return nil, nil
}

func (self *SStorage) AllowPerformOffline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "offline")
}

func (self *SStorage) PerformOffline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Status != STORAGE_OFFLINE {
		err := self.SetStatus(userCred, STORAGE_OFFLINE, "")
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_OFFLINE, "", userCred)
	}
	return nil, nil
}

func (self *SStorage) GetHostCount() int {
	return HoststorageManager.Query().Equals("storage_id", self.Id).Count()
}

func (self *SStorage) GetDiskCount() int {
	return DiskManager.Query().Equals("storage_id", self.Id).Count()
}

func (self *SStorage) GetSnapshotCount() int {
	return SnapshotManager.Query().Equals("storage_id", self.Id).Count()
}

func (self *SStorage) IsLocal() bool {
	return self.StorageType == STORAGE_LOCAL || self.StorageType == STORAGE_BAREMETAL
}

func (self *SStorage) GetStorageCachePath(mountPoint, imageCachePath string) string {
	if self.StorageType == STORAGE_NFS {
		return path.Join(mountPoint, imageCachePath)
	} else {
		return imageCachePath
	}
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
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
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

func (self *SStorage) GetRegion() *SCloudregion {
	zone := self.getZone()
	if zone == nil {
		return nil
	}
	return zone.GetRegion()
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

func (manager *SStorageManager) getStoragesByZoneId(zoneId string, provider *SCloudprovider) ([]SStorage, error) {
	storages := make([]SStorage, 0)
	q := manager.Query().Equals("zone_id", zoneId)
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
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

func (manager *SStorageManager) SyncStorages(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, zone *SZone, storages []cloudprovider.ICloudStorage) ([]SStorage, []cloudprovider.ICloudStorage, compare.SyncResult) {
	localStorages := make([]SStorage, 0)
	remoteStorages := make([]cloudprovider.ICloudStorage, 0)
	syncResult := compare.SyncResult{}

	err := manager.scanLegacyStorages()
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	dbStorages, err := manager.getStoragesByZoneId(zone.Id, provider)
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

func (manager *SStorageManager) diskIsAttachedQ(isAttached bool) *sqlchemy.SSubQuery {
	sumKey := "attached_used_capacity"
	cond := sqlchemy.In
	if !isAttached {
		sumKey = "detached_used_capacity"
		cond = sqlchemy.NotIn
	}
	sq := GuestdiskManager.Query("disk_id").SubQuery()
	disks := DiskManager.Query().SubQuery()
	disks = disks.Query().Filter(cond(disks.Field("id"), sq)).SubQuery()
	q := disks.Query(
		disks.Field("storage_id"),
		sqlchemy.SUM(sumKey, disks.Field("disk_size")),
	).Equals("status", DISK_READY).GroupBy(disks.Field("storage_id"))
	return q.SubQuery()
}

func (manager *SStorageManager) diskAttachedQ() *sqlchemy.SSubQuery {
	return manager.diskIsAttachedQ(true)
}

func (manager *SStorageManager) diskDetachedQ() *sqlchemy.SSubQuery {
	return manager.diskIsAttachedQ(false)
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
	resourceTypes []string, providers []string,
) *sqlchemy.SQuery {
	stmt := manager.disksReadyQ()
	stmt2 := manager.disksFailedQ()
	attachedDisks := manager.diskAttachedQ()
	detachedDisks := manager.diskDetachedQ()
	storages := manager.Query().SubQuery()
	q := storages.Query(
		storages.Field("capacity"),
		storages.Field("reserved"),
		storages.Field("cmtbound"),
		stmt.Field("used_capacity"),
		stmt2.Field("failed_capacity"),
		attachedDisks.Field("attached_used_capacity"),
		detachedDisks.Field("detached_used_capacity"),
	)
	q = q.LeftJoin(stmt, sqlchemy.Equals(stmt.Field("storage_id"), storages.Field("id")))
	q = q.LeftJoin(stmt2, sqlchemy.Equals(stmt2.Field("storage_id"), storages.Field("id")))
	q = q.LeftJoin(attachedDisks, sqlchemy.Equals(attachedDisks.Field("storage_id"), storages.Field("id")))
	q = q.LeftJoin(detachedDisks, sqlchemy.Equals(detachedDisks.Field("storage_id"), storages.Field("id")))

	if len(hostTypes) > 0 || len(resourceTypes) > 0 || rangeObj != nil {
		hosts := HostManager.Query().SubQuery()
		hostStorages := HoststorageManager.Query().SubQuery()

		q = q.Join(hostStorages, sqlchemy.Equals(hostStorages.Field("storage_id"), storages.Field("id")))
		q = q.Join(hosts, sqlchemy.Equals(hosts.Field("id"), hostStorages.Field("host_id")))
		q = q.Filter(sqlchemy.IsTrue(hosts.Field("enabled")))
		q = q.Filter(sqlchemy.Equals(hosts.Field("host_status"), HOST_ONLINE))

		q = AttachUsageQuery(q, hosts, hostTypes, resourceTypes, nil, rangeObj)
	}

	if len(providers) > 0 {
		cloudproviders := CloudproviderManager.Query().SubQuery()
		subq := cloudproviders.Query(cloudproviders.Field("id"))
		subq = subq.Filter(sqlchemy.In(cloudproviders.Field("provider"), providers))
		q = q.Filter(sqlchemy.In(storages.Field("manager_id"), subq.SubQuery()))
	}

	return q
}

type StorageStat struct {
	Capacity             int
	Reserved             int
	Cmtbound             float32
	UsedCapacity         int
	FailedCapacity       int
	AttachedUsedCapacity int
	DetachedUsedCapacity int
}

type StoragesCapacityStat struct {
	Capacity         int64
	CapacityVirtual  float64
	CapacityUsed     int64
	CapacityUnread   int64
	AttachedCapacity int64
	DetachedCapacity int64
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
		atCapa  int64   = 0
		dtCapa  int64   = 0
	)
	for _, stat := range stats {
		tCapa += int64(stat.Capacity - stat.Reserved)
		if stat.Cmtbound == 0 {
			stat.Cmtbound = options.Options.DefaultStorageOvercommitBound
		}
		tVCapa += float64(stat.Capacity-stat.Reserved) * float64(stat.Cmtbound)
		tUsed += int64(stat.UsedCapacity)
		tFailed += int64(stat.FailedCapacity)
		atCapa += int64(stat.AttachedUsedCapacity)
		dtCapa += int64(stat.DetachedUsedCapacity)
	}
	return StoragesCapacityStat{
		Capacity:         tCapa,
		CapacityVirtual:  tVCapa,
		CapacityUsed:     tUsed,
		CapacityUnread:   tFailed,
		AttachedCapacity: atCapa,
		DetachedCapacity: dtCapa,
	}
}

func (manager *SStorageManager) TotalCapacity(rangeObj db.IStandaloneModel, hostTypes []string, resourceTypes []string, providers []string) StoragesCapacityStat {
	res1 := manager.calculateCapacity(manager.totalCapacityQ(rangeObj, hostTypes, resourceTypes, providers))
	return res1
}

func (self *SStorage) createDisk(name string, diskConfig *SDiskConfig, userCred mcclient.TokenCredential,
	ownerProjId string, autoDelete bool, isSystem bool,
	billingType string, billingCycle string,
) (*SDisk, error) {
	disk := SDisk{}
	disk.SetModelManager(DiskManager)

	disk.Name = name
	disk.fetchDiskInfo(diskConfig)

	disk.StorageId = self.Id
	disk.AutoDelete = autoDelete
	disk.ProjectId = ownerProjId
	disk.IsSystem = isSystem

	disk.BillingType = billingType
	disk.BillingCycle = billingCycle

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
	return db.IsAdminAllowPerform(userCred, self, "cache-image")
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
	return db.IsAdminAllowPerform(userCred, self, "uncache-image")
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
	var iRegion cloudprovider.ICloudRegion
	if provider.IsOnPremiseInfrastructure() {
		iRegion, err = provider.GetOnPremiseIRegion()
	} else {
		region := self.GetRegion()
		if region == nil {
			msg := "cannot find region for storage???"
			log.Errorf(msg)
			return nil, fmt.Errorf(msg)
		}
		iRegion, err = provider.GetIRegionById(region.ExternalId)
	}
	if err != nil {
		log.Errorf("provider.GetIRegionById fail %s", err)
		return nil, err
	}
	istore, err := iRegion.GetIStorageById(self.GetExternalId())
	if err != nil {
		log.Errorf("iRegion.GetIStorageById fail %s", err)
		return nil, err
	}
	return istore, nil
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
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	regionStr, _ := query.GetString("region")
	if len(regionStr) > 0 {
		regionObj, err := CloudregionManager.FetchByIdOrName(userCred, regionStr)
		if err != nil {
			return nil, httperrors.NewNotFoundError("Region %s not found: %s", regionStr, err)
		}
		sq := ZoneManager.Query("id").Equals("cloudregion_id", regionObj.GetId())
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), sq.SubQuery()))
	}

	if jsonutils.QueryBoolean(query, "share", false) {
		q = q.Filter(sqlchemy.NotIn(q.Field("storage_type"), STORAGE_LOCAL_TYPES))
	}

	if jsonutils.QueryBoolean(query, "local", false) {
		q = q.Filter(sqlchemy.In(q.Field("storage_type"), STORAGE_LOCAL_TYPES))
	}

	if jsonutils.QueryBoolean(query, "usable", false) {
		hostStorageTable := HoststorageManager.Query().SubQuery()
		hostTable := HostManager.Query().SubQuery()
		sq := hostStorageTable.Query(hostStorageTable.Field("storage_id")).Join(hostTable,
			sqlchemy.Equals(hostTable.Field("id"), hostStorageTable.Field("host_id"))).
			Filter(sqlchemy.Equals(hostTable.Field("host_status"), HOST_ONLINE))

		q = q.Filter(sqlchemy.In(q.Field("id"), sq)).
			Filter(sqlchemy.In(q.Field("status"), []string{STORAGE_ENABLED, STORAGE_ONLINE})).
			Filter(sqlchemy.IsTrue(q.Field("enabled")))
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

func (self *SStorage) ClearSchedDescCache() error {
	hosts := self.GetAllAttachingHosts()
	if hosts == nil {
		msg := "get attaching host error"
		log.Errorf(msg)
		return fmt.Errorf(msg)
	}
	for i := 0; i < len(hosts); i += 1 {
		err := hosts[i].ClearSchedDescCache()
		if err != nil {
			log.Errorf("host CleanHostSchedCache error: %v", err)
			return err
		}
	}
	return nil
}

func (self *SStorage) getCloudBillingInfo() SCloudBillingInfo {
	var region *SCloudregion
	zone := self.getZone()
	if zone != nil {
		region = zone.GetRegion()
	}
	provider := self.GetCloudprovider()
	return MakeCloudBillingInfo(region, zone, provider)
}

func (self *SStorage) GetShortDesc() *jsonutils.JSONDict {
	info := self.getCloudBillingInfo()
	return jsonutils.Marshal(&info).(*jsonutils.JSONDict)
}
