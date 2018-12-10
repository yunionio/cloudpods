package models

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

const (
	HOST_TYPE_BAREMETAL  = "baremetal"
	HOST_TYPE_HYPERVISOR = "hypervisor" // KVM
	HOST_TYPE_KVM        = "kvm"
	HOST_TYPE_ESXI       = "esxi"    // # VMWare vSphere ESXi
	HOST_TYPE_KUBELET    = "kubelet" // # Kubernetes Kubelet
	HOST_TYPE_HYPERV     = "hyperv"  // # Microsoft Hyper-V
	HOST_TYPE_XEN        = "xen"     // # XenServer

	HOST_TYPE_ALIYUN    = "aliyun"
	HOST_TYPE_AWS       = "aws"
	HOST_TYPE_QCLOUD    = "qcloud"
	HOST_TYPE_AZURE     = "azure"
	HOST_TYPE_HUAWEI    = "huawei"
	HOST_TYPE_OPENSTACK = "openstack"

	HOST_TYPE_DEFAULT = HOST_TYPE_HYPERVISOR

	// # possible status
	HOST_ONLINE   = "online"
	HOST_ENABLED  = "online"
	HOST_OFFLINE  = "offline"
	HOST_DISABLED = "offline"

	NIC_TYPE_IPMI  = "ipmi"
	NIC_TYPE_ADMIN = "admin"
	// #NIC_TYPE_NORMAL = 'normal'

	BAREMETAL_INIT           = "init"
	BAREMETAL_PREPARE        = "prepare"
	BAREMETAL_PREPARE_FAIL   = "prepare_fail"
	BAREMETAL_READY          = "ready"
	BAREMETAL_RUNNING        = "running"
	BAREMETAL_MAINTAINING    = "maintaining"
	BAREMETAL_START_MAINTAIN = "start_maintain"
	BAREMETAL_DELETING       = "deleting"
	BAREMETAL_DELETE         = "delete"
	BAREMETAL_DELETE_FAIL    = "delete_fail"
	BAREMETAL_UNKNOWN        = "unknown"
	BAREMETAL_SYNCING_STATUS = "syncing_status"
	BAREMETAL_SYNC           = "sync"
	BAREMETAL_SYNC_FAIL      = "sync_fail"
	BAREMETAL_START_CONVERT  = "start_convert"
	BAREMETAL_CONVERTING     = "converting"
	BAREMETAL_START_FAIL     = "start_fail"
	BAREMETAL_STOP_FAIL      = "stop_fail"

	HOST_STATUS_RUNNING = BAREMETAL_RUNNING
	HOST_STATUS_READY   = BAREMETAL_READY
	HOST_STATUS_UNKNOWN = BAREMETAL_UNKNOWN
)

const (
	HostResourceTypeShared         = "shared"
	HostResourceTypeDefault        = HostResourceTypeShared
	HostResourceTypePrepaidRecycle = "prepaid"
	HostResourceTypeDedicated      = "dedicated"
)

var HOST_TYPES = []string{HOST_TYPE_BAREMETAL, HOST_TYPE_HYPERVISOR, HOST_TYPE_ESXI, HOST_TYPE_KUBELET, HOST_TYPE_XEN, HOST_TYPE_ALIYUN, HOST_TYPE_AZURE, HOST_TYPE_AWS, HOST_TYPE_QCLOUD, HOST_TYPE_HUAWEI, HOST_TYPE_OPENSTACK}

var NIC_TYPES = []string{NIC_TYPE_IPMI, NIC_TYPE_ADMIN}

type SHostManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var HostManager *SHostManager

func init() {
	HostManager = &SHostManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SHost{},
			"hosts_tbl",
			"host",
			"hosts",
		),
	}
	HostManager.SetAlias("baremetal", "baremetals")
}

type SHost struct {
	db.SEnabledStatusStandaloneResourceBase
	SManagedResourceBase
	SBillingResourceBase

	Rack  string `width:"16" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)
	Slots string `width:"16" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)

	AccessMac  string `width:"32" charset:"ascii" nullable:"false" index:"true" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(32, charset='ascii'), nullable=False, index=True)
	AccessIp   string `width:"16" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`               // Column(VARCHAR(16, charset='ascii'), nullable=True)
	ManagerUri string `width:"256" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`              // Column(VARCHAR(256, charset='ascii'), nullable=True)

	SysInfo jsonutils.JSONObject `nullable:"true" search:"admin" get:"admin" update:"admin" create:"admin_optional"`               // Column(JSONEncodedDict, nullable=True)
	SN      string               `width:"128" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(128, charset='ascii'), nullable=True)

	CpuCount    int8    `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                           // Column(TINYINT, nullable=True) # cpu count
	NodeCount   int8    `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                           // Column(TINYINT, nullable=True)
	CpuDesc     string  `width:"64" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(64, charset='ascii'), nullable=True)
	CpuMhz      int     `nullable:"true" get:"admin" update:"admin" create:"admin_optional"`                            // Column(Integer, nullable=True) # cpu MHz
	CpuCache    int     `nullable:"true" get:"admin" update:"admin" create:"admin_optional"`                            // Column(Integer, nullable=True) # cpu Cache in KB
	CpuReserved int8    `nullable:"true" default:"0" list:"admin" update:"admin" create:"admin_optional"`               // Column(TINYINT, nullable=True, default=0)
	CpuCmtbound float32 `nullable:"true" default:"8" list:"admin" update:"admin" create:"admin_optional"`               // = Column(Float, nullable=True)

	MemSize     int     `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`             // Column(Integer, nullable=True) # memory size in MB
	MemReserved int     `nullable:"true" default:"0" list:"admin" update:"admin" create:"admin_optional"` // Column(Integer, nullable=True, default=0) # memory reserved in MB
	MemCmtbound float32 `nullable:"true" default:"1" list:"admin" update:"admin" create:"admin_optional"` // = Column(Float, nullable=True)

	StorageSize   int                  `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                            // Column(Integer, nullable=True) # storage size in MB
	StorageType   string               `width:"20" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(20, charset='ascii'), nullable=True)
	StorageDriver string               `width:"20" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"`  // Column(VARCHAR(20, charset='ascii'), nullable=True)
	StorageInfo   jsonutils.JSONObject `nullable:"true" get:"admin" update:"admin" create:"admin_optional"`                             // Column(JSONEncodedDict, nullable=True)

	IpmiInfo jsonutils.JSONObject `nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(JSONEncodedDict, nullable=True)

	// Status  string = Column(VARCHAR(16, charset='ascii'), nullable=False, default=baremetalstatus.INIT) # status
	HostStatus string `width:"16" charset:"ascii" nullable:"false" default:"offline" list:"admin"` // Column(VARCHAR(16, charset='ascii'), nullable=False, server_default=HOST_OFFLINE, default=HOST_OFFLINE)

	ZoneId string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_optional"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)

	HostType string `width:"36" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	Version string `width:"64" charset:"ascii" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(64, charset='ascii'))

	IsBaremetal bool `nullable:"true" default:"false" list:"admin" create:"admin_optional"` // Column(Boolean, nullable=True, default=False)

	IsMaintenance bool `nullable:"true" default:"false" list:"admin"` // Column(Boolean, nullable=True, default=False)

	LastPingAt time.Time ``

	ResourceType string `width:"36" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_optional" default:"shared"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	RealExternalId string `width:"256" charset:"utf8" get:"admin"`
}

func (manager *SHostManager) GetContextManager() []db.IModelManager {
	return []db.IModelManager{ZoneManager}
}

func (self *SHostManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowList(userCred, self)
}

func (self *SHostManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowCreate(userCred, self)
}

func (self *SHost) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGet(userCred, self)
}

func (self *SHost) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return db.IsAdminAllowUpdate(userCred, self)
}

func (self *SHost) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowDelete(userCred, self)
}

func (manager *SHostManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	queryDict := query.(*jsonutils.JSONDict)

	resType, _ := query.GetString("resource_type")
	if len(resType) > 0 {
		queryDict.Remove("resource_type")

		switch resType {
		case HostResourceTypeShared:
			q = q.Filter(
				sqlchemy.OR(
					sqlchemy.IsNullOrEmpty(q.Field("resource_type")),
					sqlchemy.Equals(q.Field("resource_type"), HostResourceTypeShared),
				),
			)
		default:
			q = q.Equals("resource_type", resType)
		}
	}

	q, err := manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	anyMac, _ := query.GetString("any_mac")
	if len(anyMac) > 0 {
		anyMac := netutils.FormatMacAddr(anyMac)
		if len(anyMac) == 0 {
			return nil, httperrors.NewInputParameterError("invalid any_mac address")
		}
		netif, _ := NetInterfaceManager.FetchByMac(anyMac)
		if netif != nil && len(netif.BaremetalId) > 0 {
			q = q.Equals("id", netif.BaremetalId)
		} else {
			q = q.Equals("access_mac", anyMac)
		}
	}
	// var scopeQuery *sqlchemy.SSubQuery

	schedTagStr := jsonutils.GetAnyString(query, []string{"schedtag", "schedtag_id"})
	if len(schedTagStr) > 0 {
		schedTag, _ := SchedtagManager.FetchByIdOrName(nil, schedTagStr)
		if schedTag == nil {
			return nil, httperrors.NewResourceNotFoundError("Schedtag %s not found", schedTagStr)
		}
		hostschedtags := HostschedtagManager.Query().SubQuery()
		scopeQuery := hostschedtags.Query(hostschedtags.Field("host_id")).Equals("schedtag_id", schedTag.GetId()).SubQuery()
		q = q.In("id", scopeQuery)
	}

	wireStr := jsonutils.GetAnyString(query, []string{"wire", "wire_id"})
	if len(wireStr) > 0 {
		wire, _ := WireManager.FetchByIdOrName(nil, wireStr)
		if wire == nil {
			return nil, httperrors.NewResourceNotFoundError("Wire %s not found", wireStr)
		}
		hostwires := HostwireManager.Query().SubQuery()
		scopeQuery := hostwires.Query(hostwires.Field("host_id")).Equals("wire_id", wire.GetId()).SubQuery()
		q = q.In("id", scopeQuery)
	}

	storageStr := jsonutils.GetAnyString(query, []string{"storage", "storage_id"})
	if len(storageStr) > 0 {
		storage, _ := StorageManager.FetchByIdOrName(nil, storageStr)
		if storage == nil {
			return nil, httperrors.NewResourceNotFoundError("Storage %s not found", storageStr)
		}
		hoststorages := HoststorageManager.Query().SubQuery()
		scopeQuery := hoststorages.Query(hoststorages.Field("host_id")).Equals("storage_id", storage.GetId()).SubQuery()
		q = q.In("id", scopeQuery)
	}

	zoneStr := jsonutils.GetAnyString(query, []string{"zone", "zone_id"})
	if len(zoneStr) > 0 {
		zone, err := ZoneManager.FetchByIdOrName(nil, zoneStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(ZoneManager.Keyword(), zoneStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("zone_id"), zone.GetId()))

		queryDict.Remove("zone_id")
	}

	regionStr := jsonutils.GetAnyString(query, []string{"region", "region_id"})
	if len(regionStr) > 0 {
		region, err := CloudregionManager.FetchByIdOrName(nil, regionStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudregionManager.Keyword(), regionStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		subq := ZoneManager.Query("id").Equals("cloudregion_id", region.GetId()).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("zone_id"), subq))
	}

	// vcenter
	// zone
	// cachedimage

	managerStr := jsonutils.GetAnyString(query, []string{"manager", "cloudprovider", "cloudprovider_id", "manager_id"})
	if len(managerStr) > 0 {
		provider, err := CloudproviderManager.FetchByIdOrName(nil, managerStr)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), managerStr)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("manager_id"), provider.GetId()))
		queryDict.Remove("manager_id")
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
		subq := CloudproviderManager.Query("id").Equals("cloudaccount_id", account.GetId()).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
	}

	providerStr := jsonutils.GetAnyString(query, []string{"provider"})
	if len(providerStr) > 0 {
		subq := CloudproviderManager.Query("id").Equals("provider", providerStr).SubQuery()
		q = q.Filter(sqlchemy.In(q.Field("manager_id"), subq))
	}

	usable := jsonutils.QueryBoolean(query, "usable", false)
	if usable {
		hostwires := HostwireManager.Query().SubQuery()
		networks := NetworkManager.Query().SubQuery()

		hostQ := hostwires.Query(sqlchemy.DISTINCT("host_id", hostwires.Field("host_id")))
		hostQ = hostQ.Join(networks, sqlchemy.Equals(hostwires.Field("wire_id"), networks.Field("wire_id")))
		hostQ = hostQ.Filter(sqlchemy.Equals(networks.Field("status"), NETWORK_STATUS_AVAILABLE))

		q = q.In("id", hostQ.SubQuery())
	}

	if query.Contains("baremetal") {
		isBaremetal := jsonutils.QueryBoolean(query, "baremetal", false)
		if isBaremetal {
			q = q.Equals("host_type", HOST_TYPE_BAREMETAL)
		} else {
			q = q.NotEquals("host_type", HOST_TYPE_BAREMETAL)
		}
	}

	return q, nil
}

func (self *SHost) GetZone() *SZone {
	if len(self.ZoneId) == 0 {
		return nil
	}
	zone, _ := ZoneManager.FetchById(self.ZoneId)
	if zone != nil {
		return zone.(*SZone)
	}
	return nil
}

func (self *SHost) GetRegion() *SCloudregion {
	zone := self.GetZone()
	if zone != nil {
		return zone.GetRegion()
	}
	return nil
}

func (self *SHost) GetCpuCount() int {
	if self.CpuReserved > 0 && self.CpuReserved < self.CpuCount {
		return int(self.CpuCount - self.CpuReserved)
	} else {
		return int(self.CpuCount)
	}
}

func (self *SHost) GetMemSize() int {
	if self.MemReserved > 0 && self.MemReserved < self.MemSize {
		return self.MemSize - self.MemReserved
	} else {
		return self.MemSize
	}
}

func (self *SHost) GetMemoryOvercommitBound() float32 {
	if self.MemCmtbound > 0 {
		return self.MemCmtbound
	}
	return options.Options.DefaultMemoryOvercommitBound
}

func (self *SHost) GetVirtualMemorySize() float32 {
	return float32(self.GetMemSize()) * self.GetMemoryOvercommitBound()
}

func (self *SHost) GetCPUOvercommitBound() float32 {
	if self.CpuCmtbound > 0 {
		return self.CpuCmtbound
	}
	return options.Options.DefaultCPUOvercommitBound
}

func (self *SHost) GetVirtualCPUCount() float32 {
	return float32(self.GetCpuCount()) * self.GetCPUOvercommitBound()
}

/*func (manager *SHostManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (manager *SHostManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SHost) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
}

func (self *SHost) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin()
}

func (self *SHost) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return userCred.IsSystemAdmin()
}*/

func (self *SHost) ValidateDeleteCondition(ctx context.Context) error {
	if self.IsBaremetal && self.HostType != HOST_TYPE_BAREMETAL {
		return httperrors.NewInvalidStatusError("Host is a converted baremetal, should be unconverted before delete")
	}
	if self.Enabled {
		return httperrors.NewInvalidStatusError("Host is not disabled")
	}
	if self.GetGuestCount() > 0 {
		return httperrors.NewNotEmptyError("Not an empty host")
	}
	for _, hoststorage := range self.GetHoststorages() {
		storage := hoststorage.GetStorage()
		if storage != nil && storage.IsLocal() && storage.GetDiskCount() > 0 {
			return httperrors.NewNotEmptyError("Local host storage is not empty???")
		}
	}
	return self.SEnabledStatusStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func (self *SHost) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Host delete do nothing")
	return nil
}

func (self *SHost) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if self.IsBaremetal {
		return self.StartDeleteBaremetalTask(ctx, userCred, "")
	} else {
		return self.RealDelete(ctx, userCred)
	}
}

func (self *SHost) StartDeleteBaremetalTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalDeleteTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHost) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	for _, hostschedtag := range self.GetHostschedtags() {
		hostschedtag.Delete(ctx, userCred)
	}

	IsolatedDeviceManager.DeleteDevicesByHost(ctx, userCred, self)

	for _, hoststorage := range self.GetHoststorages() {
		storage := hoststorage.GetStorage()
		if storage != nil && storage.IsLocal() && storage.GetDiskCount() > 0 {
			return httperrors.NewNotEmptyError("Inconsistent: local storage is not empty???")
		}
	}
	for _, hoststorage := range self.GetHoststorages() {
		storage := hoststorage.GetStorage()
		if storage != nil && storage.IsLocal() {
			storage.Delete(ctx, userCred)
		}
		hoststorage.Delete(ctx, userCred)
	}
	for _, bn := range self.GetBaremetalnetworks() {
		self.DeleteBaremetalnetwork(ctx, userCred, &bn, false)
	}
	for _, netif := range self.GetNetInterfaces() {
		netif.Remove(ctx, userCred)
	}
	for _, hostwire := range self.GetHostwires() {
		hostwire.Detach(ctx, userCred)
		// hostwire.Delete(ctx, userCred) ???
	}
	baremetalStorage := self.GetBaremetalstorage()
	if baremetalStorage != nil {
		store := baremetalStorage.GetStorage()
		baremetalStorage.Delete(ctx, userCred)
		if store != nil {
			store.Delete(ctx, userCred)
		}
	}
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

func (self *SHost) GetHostschedtags() []SHostschedtag {
	q := HostschedtagManager.Query().Equals("host_id", self.Id)
	hostschedtags := make([]SHostschedtag, 0)
	err := db.FetchModelObjects(HostschedtagManager, q, &hostschedtags)
	if err != nil {
		log.Errorf("GetHostschedtags error %s", err)
		return nil
	}
	return hostschedtags
}

func (self *SHost) GetHoststoragesQuery() *sqlchemy.SQuery {
	return HoststorageManager.Query().Equals("host_id", self.Id)
}

func (self *SHost) GetStorageCount() int {
	return self.GetHoststoragesQuery().Count()
}

func (self *SHost) GetHoststorages() []SHoststorage {
	hoststorages := make([]SHoststorage, 0)
	q := self.GetHoststoragesQuery()
	err := db.FetchModelObjects(HoststorageManager, q, &hoststorages)
	if err != nil {
		log.Errorf("GetHoststorages error %s", err)
		return nil
	}
	return hoststorages
}

func (self *SHost) GetHoststorageOfId(storageId string) *SHoststorage {
	hoststorage := SHoststorage{}
	hoststorage.SetModelManager(HoststorageManager)
	err := self.GetHoststoragesQuery().Equals("storage_id", storageId).First(&hoststorage)
	if err != nil {
		log.Errorf("GetHoststorageOfId fail %s", err)
		return nil
	}
	return &hoststorage
}

func (self *SHost) GetHoststorageByExternalId(extId string) *SHoststorage {
	hoststorage := SHoststorage{}
	hoststorage.SetModelManager(HoststorageManager)

	hoststorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()
	q := hoststorages.Query()
	q = q.Join(storages, sqlchemy.Equals(hoststorages.Field("storage_id"), storages.Field("id")))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("host_id"), self.Id))
	q = q.Filter(sqlchemy.Equals(storages.Field("external_id"), extId))

	err := q.First(&hoststorage)
	if err != nil {
		log.Errorf("GetHoststorageByExternalId fail %s", err)
		return nil
	}

	return &hoststorage
}

func (self *SHost) GetStorageByFilePath(path string) *SStorage {
	hoststorages := self.GetHoststorages()
	if hoststorages == nil {
		return nil
	}
	for i := 0; i < len(hoststorages); i += 1 {
		if strings.HasPrefix(path, hoststorages[i].MountPoint) {
			return hoststorages[i].GetStorage()
		}
	}
	return nil
}

func (self *SHost) GetBaremetalstorage() *SHoststorage {
	if !self.IsBaremetal {
		return nil
	}
	hoststorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()
	q := hoststorages.Query()
	q = q.Join(storages, sqlchemy.AND(sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id")),
		sqlchemy.IsFalse(storages.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(storages.Field("storage_type"), STORAGE_BAREMETAL))
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("host_id"), self.Id))
	if q.Count() == 1 {
		hs := SHoststorage{}
		hs.SetModelManager(HoststorageManager)
		err := q.First(&hs)
		if err != nil {
			log.Errorf("error %s", err)
			return nil
		}
		return &hs
	}
	log.Errorf("Cannof find baremetalstorage??")
	return nil
}

func (self *SHost) SaveCleanUpdates(doUpdate func() error) (map[string]sqlchemy.SUpdateDiff, error) {
	return self.saveUpdates(doUpdate, true)
}

func (self *SHost) SaveUpdates(doUpdate func() error) (map[string]sqlchemy.SUpdateDiff, error) {
	return self.saveUpdates(doUpdate, false)
}

func (self *SHost) saveUpdates(doUpdate func() error, doSchedClean bool) (map[string]sqlchemy.SUpdateDiff, error) {
	diff, err := self.GetModelManager().TableSpec().Update(self, doUpdate)
	if err == nil && doSchedClean {
		self.ClearSchedDescCache()
	}
	return diff, err
}

func (self *SHost) AllowPerformUpdateStorage(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return db.IsAdminAllowPerform(userCred, self, "update-storage")
}

func (self *SHost) PerformUpdateStorage(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	bs := self.GetBaremetalstorage()
	capacity, _ := data.Int("capacity")
	zoneId, _ := data.GetString("zone_id")
	if bs == nil {
		// 1. create storage
		storage := SStorage{}
		storage.Name = fmt.Sprintf("storage%s", self.GetName())
		storage.Capacity = int(capacity)
		storage.StorageType = STORAGE_BAREMETAL
		storage.MediumType = self.StorageType
		storage.Cmtbound = 1.0
		storage.Status = STORAGE_ONLINE
		storage.ZoneId = zoneId
		err := StorageManager.TableSpec().Insert(&storage)
		if err != nil {
			return nil, fmt.Errorf("Create baremetal storage error: %v", err)
		}
		// 2. create host storage
		bmStorage := SHoststorage{}
		bmStorage.HostId = self.Id
		bmStorage.StorageId = storage.Id
		bmStorage.RealCapacity = int(capacity)
		bmStorage.MountPoint = ""
		err = HoststorageManager.TableSpec().Insert(&bmStorage)
		if err != nil {
			return nil, fmt.Errorf("Create baremetal hostStorage error: %v", err)
		}
		return nil, nil
	}
	storage := bs.GetStorage()
	if capacity != int64(storage.Capacity) {
		_, err := storage.GetModelManager().TableSpec().Update(storage, func() error {
			storage.Capacity = int(capacity)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("Update baremetal storage error: %v", err)
		}
	}
	return nil, nil
}

func (self *SHost) GetFetchUrl() string {
	managerUrl, err := url.Parse(self.ManagerUri)
	if err != nil {
		log.Errorf("GetFetchUrl fail to parse url: %s", err)
	}
	portStr := managerUrl.Port()
	var port int
	if len(portStr) > 0 {
		port, _ = strconv.Atoi(portStr)
	} else {
		if managerUrl.Scheme == "https" {
			port = 443
		} else if managerUrl.Scheme == "http" {
			port = 80
		}
	}
	return fmt.Sprintf("%s://%s:%d", managerUrl.Scheme, strings.Split(managerUrl.Host, ":")[0], port+40000)
}

func (self *SHost) GetAttachedStorages(storageType string) []SStorage {
	return self._getAttachedStorages(tristate.False, tristate.True, storageType)
}

func (self *SHost) _getAttachedStorages(isBaremetal tristate.TriState, enabled tristate.TriState, storageType string) []SStorage {
	storages := StorageManager.Query().SubQuery()
	hoststorages := HoststorageManager.Query().SubQuery()
	q := storages.Query()
	q = q.Join(hoststorages, sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id")))
	if enabled.IsTrue() {
		q = q.IsTrue("enabled")
	} else if enabled.IsFalse() {
		q = q.IsFalse("enabled")
	}
	if isBaremetal.IsTrue() {
		q = q.Equals("storage_type", STORAGE_BAREMETAL)
	} else if isBaremetal.IsFalse() {
		q = q.NotEquals("storage_type", STORAGE_BAREMETAL)
	}
	if len(storageType) > 0 {
		q = q.Equals("storage_type", storageType)
	}
	q = q.Filter(sqlchemy.Equals(hoststorages.Field("host_id"), self.Id))
	ret := make([]SStorage, 0)
	err := db.FetchModelObjects(StorageManager, q, &ret)
	if err != nil {
		log.Errorf("GetAttachedStorages fail %s", err)
		return nil
	}
	return ret
}

func (self *SHost) SyncAttachedStorageStatus() {
	storages := self.GetAttachedStorages("")
	if storages != nil {
		for _, storage := range storages {
			storage.SyncStatusWithHosts()
		}
		self.ClearSchedDescCache()
	}
}

func (self *SHostManager) IsNewNameUnique(name string, userCred mcclient.TokenCredential, kwargs *jsonutils.JSONDict) bool {
	q := self.Query().Equals("name", name)
	if kwargs != nil && kwargs.Contains("zone_id") {
		zoneId, _ := kwargs.GetString("zone_id")
		q.Equals("zone_id", zoneId)
	}
	return q.Count() == 0
}

func (self *SHostManager) AllowGetPropertyBmStartRegisterScript(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SHostManager) GetPropertyBmStartRegisterScript(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	regionUri, err := auth.GetServiceURL("compute_v2", options.Options.Region, "", "")
	if err != nil {
		return nil, err
	}
	var script string
	script += fmt.Sprintf("curl -k -fsSL -H 'X-Auth-Token: %s' %s/misc/bm-prepare-script", userCred.GetTokenString(), regionUri)
	res := jsonutils.NewDict()
	res.Add(jsonutils.NewString(script), "script")
	return res, nil
}

func (maanger *SHostManager) ClearAllSchedDescCache() error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
	return modules.SchedManager.CleanCache(s, "")
}

func (maanger *SHostManager) ClearSchedDescCache(hostId string) error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
	return modules.SchedManager.CleanCache(s, hostId)
}

func (self *SHost) ClearSchedDescCache() error {
	return HostManager.ClearSchedDescCache(self.Id)
}

func (self *SHost) GetSpec(statusCheck bool) *jsonutils.JSONDict {
	if statusCheck {
		if utils.IsInStringArray(self.Status, []string{BAREMETAL_INIT, BAREMETAL_PREPARE_FAIL, BAREMETAL_PREPARE}) ||
			self.GetBaremetalServer() != nil {
			return nil
		}
		if self.MemSize == 0 || self.CpuCount == 0 {
			return nil
		}
		if self.ResourceType == HostResourceTypePrepaidRecycle && self.GetGuestCount() > 0 {
			return nil
		}
	}
	spec := self.GetHardwareSpecification()
	spec.Remove("storage_info")
	nifs := self.GetNetInterfaces()
	var nicCount int64
	for _, nif := range nifs {
		if nif.NicType != NIC_TYPE_IPMI {
			nicCount++
		}
	}
	spec.Set("nic_count", jsonutils.NewInt(nicCount))

	var manufacture string
	var model string
	if self.SysInfo != nil {
		manufacture, _ = self.SysInfo.GetString("manufacture")
		model, _ = self.SysInfo.GetString("model")
	}
	if manufacture == "" {
		manufacture = "Unknown"
	}
	if model == "" {
		model = "Unknown"
	}
	spec.Set("manufacture", jsonutils.NewString(manufacture))
	spec.Set("model", jsonutils.NewString(model))
	return spec
}

func (manager *SHostManager) GetSpecIdent(spec *jsonutils.JSONDict) []string {
	nCpu, _ := spec.Int("cpu")
	memSize, _ := spec.Int("mem")
	var memSizeStr string
	if memSize < 1024 {
		memSizeStr = fmt.Sprintf("mem:%dM", memSize)
	} else {
		memGB, err := utils.GetSizeGB(fmt.Sprintf("%d", memSize), "M")
		if err != nil {
			log.Errorf("Get mem size %d GB error: %v", memSize, err)
		}
		memSizeStr = fmt.Sprintf("mem:%dG", memGB)
	}
	nicCount, _ := spec.Int("nic_count")
	manufacture, _ := spec.GetString("manufacture")
	model, _ := spec.GetString("model")

	specKeys := []string{
		fmt.Sprintf("cpu:%d", nCpu),
		memSizeStr,
		fmt.Sprintf("nic:%d", nicCount),
		fmt.Sprintf("manufacture:%s", manufacture),
		fmt.Sprintf("model:%s", model),
	}
	diskSpec, _ := spec.Get("disk")
	if diskSpec != nil {
		driverSpecs, _ := diskSpec.GetMap()
		for driver, driverSpec := range driverSpecs {
			specKeys = append(specKeys, parseDiskDriverSpec(driver, driverSpec)...)
		}
	}
	sort.Strings(specKeys)
	return specKeys
}

func parseDiskDriverSpec(driver string, spec jsonutils.JSONObject) []string {
	ret := make([]string, 0)
	adapterSpecs, _ := spec.GetMap()
	for adapterKey, adapterSpec := range adapterSpecs {
		for _, diskType := range []string{baremetal.HDD_DISK_SPEC_TYPE, baremetal.SSD_DISK_SPEC_TYPE} {
			sizeCountMap, _ := adapterSpec.GetMap(diskType)
			if sizeCountMap == nil {
				continue
			}
			for size, count := range sizeCountMap {
				sizeGB, _ := utils.GetSizeGB(size, "M")
				diskKey := fmt.Sprintf("disk:%s_%s_%s_%dGx%s", driver, adapterKey, diskType, sizeGB, count)
				ret = append(ret, diskKey)
			}
		}
	}
	return ret
}

func ConvertStorageInfo2BaremetalStorages(storageInfo jsonutils.JSONObject) []*baremetal.BaremetalStorage {
	storages := []baremetal.BaremetalStorage{}
	err := storageInfo.Unmarshal(&storages)
	if err != nil {
		log.Errorf("Unmarshal to baremetal storage error: %v", err)
		return nil
	}
	ret := make([]*baremetal.BaremetalStorage, len(storages))
	for i := range storages {
		ret[i] = &storages[i]
	}
	return ret
}

func GetDiskSpecV2(storageInfo jsonutils.JSONObject) jsonutils.JSONObject {
	refStorages := ConvertStorageInfo2BaremetalStorages(storageInfo)
	if refStorages == nil {
		return nil
	}
	diskSpec := baremetal.GetDiskSpecV2(refStorages)
	return jsonutils.Marshal(diskSpec)
}

func (self *SHost) GetHardwareSpecification() *jsonutils.JSONDict {
	spec := jsonutils.NewDict()
	spec.Set("cpu", jsonutils.NewInt(int64(self.CpuCount)))
	spec.Set("mem", jsonutils.NewInt(int64(self.MemSize)))
	if self.StorageInfo != nil {
		spec.Set("disk", GetDiskSpecV2(self.StorageInfo))
		spec.Set("driver", jsonutils.NewString(self.StorageDriver))
		spec.Set("storage_info", self.StorageInfo)
	}
	return spec
}

type SStorageCapacity struct {
	Capacity  int
	Used      int
	Wasted    int
	VCapacity int
}

func (capa SStorageCapacity) GetFree() int {
	return capa.VCapacity - capa.Used - capa.Wasted
}

func (self *SHost) GetAttachedStorageCapacity() SStorageCapacity {
	ret := SStorageCapacity{}
	storages := self.GetAttachedStorages("")
	for _, s := range storages {
		ret.Capacity += s.GetCapacity()
		ret.Used += s.GetUsedCapacity(tristate.True)
		ret.Wasted += s.GetUsedCapacity(tristate.False)
		ret.VCapacity += int(float32(s.GetCapacity()) * s.GetOvercommitBound())
	}
	return ret
}

func _getLeastUsedStorage(storages []SStorage, backends []string) *SStorage {
	var best *SStorage
	var bestCap int
	for i := 0; i < len(storages); i++ {
		s := storages[i]
		if len(backends) > 0 {
			in, _ := utils.InStringArray(s.StorageType, backends)
			if !in {
				continue
			}
		}
		capa := s.GetFreeCapacity()
		if best == nil || bestCap < capa {
			bestCap = capa
			best = &s
		}
	}
	return best
}

func getLeastUsedStorage(storages []SStorage, backend string) *SStorage {
	var backends []string
	if backend == STORAGE_LOCAL {
		backends = []string{STORAGE_NAS, STORAGE_LOCAL}
	} else if len(backend) > 0 {
		backends = []string{backend}
	} else {
		backends = []string{}
	}
	return _getLeastUsedStorage(storages, backends)
}

func (self *SHost) GetLeastUsedStorage(backend string) *SStorage {
	storages := self.GetAttachedStorages("")
	if storages != nil {
		return getLeastUsedStorage(storages, backend)
	}
	return nil
}

func (self *SHost) GetWiresQuery() *sqlchemy.SQuery {
	return HostwireManager.Query().Equals("host_id", self.Id)
}

func (self *SHost) GetWireCount() int {
	return self.GetWiresQuery().Count()
}

func (self *SHost) GetHostwires() []SHostwire {
	hw := make([]SHostwire, 0)
	q := self.GetWiresQuery()
	err := db.FetchModelObjects(HostwireManager, q, &hw)
	if err != nil {
		log.Errorf("GetWires error %s", err)
		return nil
	}
	return hw
}

func (self *SHost) getAttachedWires() []SWire {
	wires := WireManager.Query().SubQuery()
	hostwires := HostwireManager.Query().SubQuery()
	q := wires.Query()
	q = q.Join(hostwires, sqlchemy.AND(sqlchemy.IsFalse(hostwires.Field("deleted")),
		sqlchemy.Equals(hostwires.Field("wire_id"), wires.Field("id"))))
	q = q.Filter(sqlchemy.Equals(hostwires.Field("host_id"), self.Id))
	ret := make([]SWire, 0)
	err := db.FetchModelObjects(WireManager, q, &ret)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return ret
}

func (self *SHost) GetMasterHostwire() *SHostwire {
	hw := SHostwire{}
	hw.SetModelManager(HostwireManager)

	q := self.GetWiresQuery().IsTrue("is_master")
	err := q.First(&hw)
	if err != nil {
		log.Errorf("GetMasterHostwire %s", err)
		return nil
	}
	return &hw
}

func (self *SHost) GetMasterWire() *SWire {
	wires := WireManager.Query().SubQuery()
	hostwires := HostwireManager.Query().SubQuery()
	q := wires.Query()
	q = q.Join(hostwires, sqlchemy.AND(sqlchemy.Equals(hostwires.Field("wire_id"), wires.Field("id")),
		sqlchemy.IsFalse(hostwires.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(hostwires.Field("host_id"), self.Id))
	q = q.Filter(sqlchemy.IsTrue(hostwires.Field("is_master")))
	wire := SWire{}
	wire.SetModelManager(WireManager)

	err := q.First(&wire)
	if err != nil {
		log.Errorf("GetMasterWire fail %s", err)
		return nil
	}
	return &wire
}

func (self *SHost) getHostwireOfId(wireId string) *SHostwire {
	hostwire := SHostwire{}
	hostwire.SetModelManager(HostwireManager)

	q := self.GetWiresQuery().Equals("wire_id", wireId)
	err := q.First(&hostwire)
	if err != nil {
		log.Errorf("getHostwireOfId fail %s", err)
		return nil
	}
	return &hostwire
}

func (self *SHost) GetGuestsQuery() *sqlchemy.SQuery {
	return GuestManager.Query().Equals("host_id", self.Id)
}

func (self *SHost) GetGuests() []SGuest {
	q := self.GetGuestsQuery()
	guests := make([]SGuest, 0)
	err := db.FetchModelObjects(GuestManager, q, &guests)
	if err != nil {
		log.Errorf("GetGuests %s", err)
		return nil
	}
	return guests
}

func (self *SHost) GetGuestCount() int {
	q := self.GetGuestsQuery()
	return q.Count()
}

func (self *SHost) GetNonsystemGuestCount() int {
	q := self.GetGuestsQuery()
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))))
	return q.Count()
}

func (self *SHost) GetRunningGuestCount() int {
	q := self.GetGuestsQuery()
	q = q.In("status", VM_RUNNING_STATUS)
	return q.Count()
}

func (self *SHost) GetBaremetalnetworksQuery() *sqlchemy.SQuery {
	return HostnetworkManager.Query().Equals("baremetal_id", self.Id)
}

func (self *SHost) GetBaremetalnetworks() []SHostnetwork {
	q := self.GetBaremetalnetworksQuery()
	hns := make([]SHostnetwork, 0)
	err := db.FetchModelObjects(HostnetworkManager, q, &hns)
	if err != nil {
		log.Errorf("GetBaremetalnetworks error: %s", err)
	}
	return hns
}

func (self *SHost) GetAttach2Network(network *SNetwork) *SHostnetwork {
	q := self.GetBaremetalnetworksQuery()
	q = q.Equals("network_id", network.Id)
	hn := SHostnetwork{}
	hn.SetModelManager(HostnetworkManager)

	err := q.First(&hn)
	if err != nil {
		log.Errorf("GetAttach2Network fail %s", err)
		return nil
	}
	return &hn
}

func (self *SHost) isAttach2Network(network *SNetwork) bool {
	hn := self.GetAttach2Network(network)
	return hn != nil
}

func (self *SHost) GetNetInterfaces() []SNetInterface {
	q := NetInterfaceManager.Query().Equals("baremetal_id", self.Id).Asc("index")
	netifs := make([]SNetInterface, 0)
	err := db.FetchModelObjects(NetInterfaceManager, q, &netifs)
	if err != nil {
		log.Errorf("GetNetInterfaces fail %s", err)
		return nil
	}
	return netifs
}

func (self *SHost) GetAdminNetInterface() *SNetInterface {
	netif := SNetInterface{}
	netif.SetModelManager(NetInterfaceManager)

	q := NetInterfaceManager.Query().Equals("baremetal_id", self.Id).Equals("nic_type", NIC_TYPE_ADMIN)
	err := q.First(&netif)
	if err != nil {
		log.Errorf("GetAdminNetInterface fail %s", err)
		return nil
	}
	return &netif
}

func (self *SHost) GetNetInterface(mac string) *SNetInterface {
	netif, _ := NetInterfaceManager.FetchByMac(mac)
	if netif != nil && netif.BaremetalId == self.Id {
		return netif
	}
	return nil
}

func (self *SHost) DeleteBaremetalnetwork(ctx context.Context, userCred mcclient.TokenCredential, bn *SHostnetwork, reserve bool) {
	net := bn.GetNetwork()
	bn.Delete(ctx, userCred)
	db.OpsLog.LogDetachEvent(ctx, self, net, userCred, nil)
	if reserve && net != nil && len(bn.IpAddr) > 0 && regutils.MatchIP4Addr(bn.IpAddr) {
		ReservedipManager.ReserveIP(userCred, net, bn.IpAddr, "Delete baremetalnetwork to reserve")
	}
}

func (self *SHost) GetHostDriver() IHostDriver {
	if !utils.IsInStringArray(self.HostType, HOST_TYPES) {
		log.Fatalf("Unsupported host type %s", self.HostType)
	}
	return GetHostDriver(self.HostType)
}

func (manager *SHostManager) getHostsByZoneProvider(zone *SZone, provider *SCloudprovider) ([]SHost, error) {
	hosts := make([]SHost, 0)
	q := manager.Query()
	if zone != nil {
		q = q.Equals("zone_id", zone.Id)
	}
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	// exclude prepaid_recycle fake hosts
	q = q.NotEquals("resource_type", HostResourceTypePrepaidRecycle)

	err := db.FetchModelObjects(manager, q, &hosts)
	if err != nil {
		log.Errorf("%s", err)
		return nil, err
	}
	return hosts, nil
}

func (manager *SHostManager) SyncHosts(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, zone *SZone, hosts []cloudprovider.ICloudHost, projectSync bool) ([]SHost, []cloudprovider.ICloudHost, compare.SyncResult) {
	localHosts := make([]SHost, 0)
	remoteHosts := make([]cloudprovider.ICloudHost, 0)
	syncResult := compare.SyncResult{}

	dbHosts, err := manager.getHostsByZoneProvider(zone, provider)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	removed := make([]SHost, 0)
	commondb := make([]SHost, 0)
	commonext := make([]cloudprovider.ICloudHost, 0)
	added := make([]cloudprovider.ICloudHost, 0)

	err = compare.CompareSets(dbHosts, hosts, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].ValidateDeleteCondition(ctx)
		if err != nil { // cannot delete
			err = removed[i].SetStatus(userCred, HOST_OFFLINE, "sync to delete")
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
		err = commondb[i].syncWithCloudHost(commonext[i], projectSync)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localHosts = append(localHosts, commondb[i])
			remoteHosts = append(remoteHosts, commonext[i])
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		new, err := manager.newFromCloudHost(added[i], zone)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localHosts = append(localHosts, *new)
			remoteHosts = append(remoteHosts, added[i])
			syncResult.Add()
		}
	}

	return localHosts, remoteHosts, syncResult
}

func (self *SHost) syncWithCloudHost(extHost cloudprovider.ICloudHost, projectSync bool) error {
	_, err := self.SaveUpdates(func() error {
		self.Name = extHost.GetName()

		self.Status = extHost.GetStatus()
		self.HostStatus = extHost.GetHostStatus()
		self.AccessIp = extHost.GetAccessIp()
		self.AccessMac = extHost.GetAccessMac()
		self.SN = extHost.GetSN()
		self.SysInfo = extHost.GetSysInfo()
		self.CpuCount = extHost.GetCpuCount()
		self.NodeCount = extHost.GetNodeCount()
		self.CpuDesc = extHost.GetCpuDesc()
		self.CpuMhz = extHost.GetCpuMhz()
		self.MemSize = extHost.GetMemSizeMB()
		self.StorageSize = extHost.GetStorageSizeMB()
		self.StorageType = extHost.GetStorageType()
		self.HostType = extHost.GetHostType()

		self.ManagerId = extHost.GetManagerId()
		self.IsEmulated = extHost.IsEmulated()
		self.Enabled = extHost.GetEnabled()

		self.IsMaintenance = extHost.GetIsMaintenance()
		self.Version = extHost.GetVersion()

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
	}

	if projectSync {
		if err := HostManager.ClearSchedDescCache(self.Id); err != nil {
			log.Errorf("ClearSchedDescCache for host %s error %v", self.Name, err)
		}
	}

	return err
}

func (self *SHost) syncWithCloudPrepaidVM(extVM cloudprovider.ICloudVM, host *SHost, projectSync bool) error {
	_, err := self.SaveUpdates(func() error {

		self.CpuCount = extVM.GetVcpuCount()
		self.MemSize = extVM.GetVmemSizeMB()

		self.BillingType = extVM.GetBillingType()
		self.ExpiredAt = extVM.GetExpiredAt()

		self.ExternalId = host.ExternalId

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
	}

	if projectSync {
		if err := HostManager.ClearSchedDescCache(self.Id); err != nil {
			log.Errorf("ClearSchedDescCache for host %s error %v", self.Name, err)
		}
	}

	return err
}

func (manager *SHostManager) newFromCloudHost(extHost cloudprovider.ICloudHost, izone *SZone) (*SHost, error) {
	host := SHost{}
	host.SetModelManager(manager)

	if izone == nil {
		wire, err := WireManager.GetWireOfIp(extHost.GetAccessIp())
		if err != nil {
			msg := fmt.Sprintf("fail to find wire for host %s %s: %s", extHost.GetName(), extHost.GetAccessIp(), err)
			log.Errorf(msg)
			return nil, fmt.Errorf(msg)
		}
		izone = wire.GetZone()
	}

	host.Name = extHost.GetName()
	host.ExternalId = extHost.GetGlobalId()
	host.ZoneId = izone.Id
	// host.ManagerId = extHost.GetManagerId()
	host.HostType = extHost.GetHostType()

	host.Status = extHost.GetStatus()
	host.HostStatus = extHost.GetHostStatus()
	host.Enabled = extHost.GetEnabled()

	host.AccessIp = extHost.GetAccessIp()
	host.AccessMac = extHost.GetAccessMac()
	host.SN = extHost.GetSN()
	host.SysInfo = extHost.GetSysInfo()
	host.CpuCount = extHost.GetCpuCount()
	host.NodeCount = extHost.GetNodeCount()
	host.CpuDesc = extHost.GetCpuDesc()
	host.CpuMhz = extHost.GetCpuMhz()
	host.MemSize = extHost.GetMemSizeMB()
	host.StorageSize = extHost.GetStorageSizeMB()
	host.StorageType = extHost.GetStorageType()
	host.CpuCmtbound = 8.0
	host.MemCmtbound = 1.0

	host.ManagerId = extHost.GetManagerId()
	host.IsEmulated = extHost.IsEmulated()

	host.IsMaintenance = extHost.GetIsMaintenance()
	host.Version = extHost.GetVersion()

	err := manager.TableSpec().Insert(&host)
	if err != nil {
		log.Errorf("newFromCloudHost fail %s", err)
		return nil, err
	}

	if err := manager.ClearSchedDescCache(host.Id); err != nil {
		log.Errorf("ClearSchedDescCache for host %s error %v", host.Name, err)
	}

	return &host, nil
}

func (self *SHost) SyncHostStorages(ctx context.Context, userCred mcclient.TokenCredential, storages []cloudprovider.ICloudStorage) ([]SStorage, []cloudprovider.ICloudStorage, compare.SyncResult) {
	localStorages := make([]SStorage, 0)
	remoteStorages := make([]cloudprovider.ICloudStorage, 0)
	syncResult := compare.SyncResult{}

	dbStorages := make([]SStorage, 0)

	hostStorages := self.GetHoststorages()
	for i := 0; i < len(hostStorages); i += 1 {
		storage := hostStorages[i].GetStorage()
		if storage == nil {
			hostStorages[i].Delete(ctx, userCred)
		} else {
			dbStorages = append(dbStorages, *storage)
		}
	}

	// dbStorages := self._getAttachedStorages(tristate.None, tristate.None)

	removed := make([]SStorage, 0)
	commondb := make([]SStorage, 0)
	commonext := make([]cloudprovider.ICloudStorage, 0)
	added := make([]cloudprovider.ICloudStorage, 0)

	err := compare.CompareSets(dbStorages, storages, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		log.Infof("host %s not connected with %s any more, to detach...", self.Id, removed[i].Id)
		hs := self.GetHoststorageOfId(removed[i].Id)
		err := hs.Detach(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		log.Infof("host %s is still connected with %s, to update ...", self.Id, commondb[i].Id)
		err := self.syncWithCloudHostStorage(&commondb[i], commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
		localStorages = append(localStorages, commondb[i])
		remoteStorages = append(remoteStorages, commonext[i])
	}

	for i := 0; i < len(added); i += 1 {
		log.Infof("host %s is found connected with %s, to add ...", self.Id, added[i].GetId())
		local, err := self.newCloudHostStorage(ctx, userCred, added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			localStorages = append(localStorages, *local)
			remoteStorages = append(remoteStorages, added[i])
			syncResult.Add()
		}
	}
	return localStorages, remoteStorages, syncResult
}

func (self *SHost) syncWithCloudHostStorage(localStorage *SStorage, extStorage cloudprovider.ICloudStorage) error {
	// do nothing
	hs := self.GetHoststorageOfId(localStorage.Id)
	return hs.syncWithCloudHostStorage(extStorage)
}

func (self *SHost) isAttach2Storage(storage *SStorage) bool {
	hs := self.GetHoststorageOfId(storage.Id)
	return hs != nil
}

func (self *SHost) Attach2Storage(ctx context.Context, userCred mcclient.TokenCredential, storage *SStorage, mountPoint string) error {
	if self.isAttach2Storage(storage) {
		return nil
	}

	hs := SHoststorage{}
	hs.SetModelManager(HoststorageManager)

	hs.StorageId = storage.Id
	hs.HostId = self.Id
	hs.MountPoint = mountPoint
	err := HoststorageManager.TableSpec().Insert(&hs)
	if err != nil {
		return err
	}

	db.OpsLog.LogAttachEvent(ctx, self, storage, userCred, nil)

	return nil
}

func (self *SHost) newCloudHostStorage(ctx context.Context, userCred mcclient.TokenCredential, extStorage cloudprovider.ICloudStorage) (*SStorage, error) {
	storageObj, err := StorageManager.FetchByExternalId(extStorage.GetGlobalId())
	if err != nil {
		if err == sql.ErrNoRows {
			// no cloud storage found, this may happen for on-premise host
			// create the storage right now
			storageObj, err = StorageManager.newFromCloudStorage(extStorage, self.GetZone())
			if err != nil {
				log.Errorf("create by cloud storage fail %s", err)
				return nil, err
			}
		} else {
			log.Errorf("%s", err)
			return nil, err
		}
	}
	storage := storageObj.(*SStorage)
	err = self.Attach2Storage(ctx, userCred, storage, extStorage.GetMountPoint())
	return storage, err
}

func (self *SHost) SyncHostWires(ctx context.Context, userCred mcclient.TokenCredential, wires []cloudprovider.ICloudWire) compare.SyncResult {
	syncResult := compare.SyncResult{}

	dbWires := make([]SWire, 0)

	hostWires := self.GetHostwires()
	for i := 0; i < len(hostWires); i += 1 {
		wire := hostWires[i].GetWire()
		if wire == nil {
			hostWires[i].Delete(ctx, userCred)
		} else {
			dbWires = append(dbWires, *wire)
		}
	}

	// dbWires := self.getAttachedWires()

	removed := make([]SWire, 0)
	commondb := make([]SWire, 0)
	commonext := make([]cloudprovider.ICloudWire, 0)
	added := make([]cloudprovider.ICloudWire, 0)

	err := compare.CompareSets(dbWires, wires, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		log.Infof("host %s not connected with %s any more, to detach...", self.Id, removed[i].Id)
		hw := self.getHostwireOfId(removed[i].Id)
		err := hw.Detach(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		log.Infof("host %s is still connected with %s, to update...", self.Id, commondb[i].Id)
		err := self.syncWithCloudHostWire(commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		log.Infof("host %s is found connected with %s, to add...", self.Id, added[i].GetId())
		err := self.newCloudHostWire(ctx, userCred, added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}

	return syncResult
}

func (self *SHost) syncWithCloudHostWire(extWire cloudprovider.ICloudWire) error {
	// do nothing
	return nil
}

func (self *SHost) Attach2Wire(ctx context.Context, userCred mcclient.TokenCredential, wire *SWire) error {
	hs := SHostwire{}
	hs.SetModelManager(HostwireManager)

	hs.WireId = wire.Id
	hs.HostId = self.Id
	err := HostwireManager.TableSpec().Insert(&hs)
	if err != nil {
		return err
	}
	db.OpsLog.LogAttachEvent(ctx, self, wire, userCred, nil)
	return nil
}

func (self *SHost) newCloudHostWire(ctx context.Context, userCred mcclient.TokenCredential, extWire cloudprovider.ICloudWire) error {
	wireObj, err := WireManager.FetchByExternalId(extWire.GetGlobalId())
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	wire := wireObj.(*SWire)
	err = self.Attach2Wire(ctx, userCred, wire)
	return err
}

func (self *SHost) SyncHostVMs(ctx context.Context, userCred mcclient.TokenCredential, vms []cloudprovider.ICloudVM, projectId string, projectSync bool) ([]SGuest, []cloudprovider.ICloudVM, compare.SyncResult) {
	localVMs := make([]SGuest, 0)
	remoteVMs := make([]cloudprovider.ICloudVM, 0)
	syncResult := compare.SyncResult{}

	dbVMs := self.GetGuests()

	removed := make([]SGuest, 0)
	commondb := make([]SGuest, 0)
	commonext := make([]cloudprovider.ICloudVM, 0)
	added := make([]cloudprovider.ICloudVM, 0)

	err := compare.CompareSets(dbVMs, vms, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return nil, nil, syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err := removed[i].SetStatus(userCred, VM_UNKNOWN, "Sync lost")
		if err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		err := commondb[i].syncWithCloudVM(ctx, userCred, self, commonext[i], projectId, projectSync)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localVMs = append(localVMs, commondb[i])
			remoteVMs = append(remoteVMs, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		if added[i].GetBillingType() == BILLING_TYPE_PREPAID {
			vhost := HostManager.GetHostByRealExternalId(added[i].GetGlobalId())
			if vhost != nil {
				// this recycle vm is not build yet, skip synchronize
				continue
			}
		}
		new, err := GuestManager.newCloudVM(ctx, userCred, self, added[i], projectId)
		if err != nil {
			syncResult.AddError(err)
		} else {
			localVMs = append(localVMs, *new)
			remoteVMs = append(remoteVMs, added[i])
			syncResult.Add()
		}
	}

	return localVMs, remoteVMs, syncResult
}

func (self *SHost) getNetworkOfIPOnHost(ipAddr string) (*SNetwork, error) {
	net, err := NetworkManager.GetNetworkOfIP(ipAddr, "", tristate.None)
	if err != nil {
		return nil, err
	}
	hw := self.getHostwireOfId(net.WireId)
	if hw == nil {
		return nil, fmt.Errorf("IP %s not reachable on this host", ipAddr)
	}
	return net, nil
}

func (self *SHost) GetNetinterfaceWithNetworkAndCredential(netId string, userCred mcclient.TokenCredential, reserved bool) (*SNetInterface, *SNetwork) {
	netif, net := self.getNetifWithNetworkAndCredential(netId, userCred, true, reserved)
	if netif != nil {
		return netif, net
	}
	return self.getNetifWithNetworkAndCredential(netId, userCred, false, reserved)
}

func (self *SHost) getNetifWithNetworkAndCredential(netId string, userCred mcclient.TokenCredential, isPublic bool, reserved bool) (*SNetInterface, *SNetwork) {
	netifs := self.GetNetInterfaces()
	var maxFreeCnt = 0
	var maxFreeNet *SNetwork
	var maxFreeNetif *SNetInterface
	for i := 0; i < len(netifs); i++ {
		if !netifs[i].IsUsableServernic() {
			continue
		}
		wire := netifs[i].GetWire()
		if wire != nil {
			if isPublic {
				nets, _ := wire.getPublicNetworks()
				for _, net := range nets {
					if net.Id == netId || net.GetName() == netId {
						freeCnt := net.getFreeAddressCount()
						if maxFreeNet == nil || maxFreeCnt < freeCnt {
							maxFreeNetif = &netifs[i]
							maxFreeNet = &net
						}
					}
				}
			} else {
				nets, _ := wire.getPrivateNetworks(userCred)
				for _, net := range nets {
					if net.Id == netId || net.GetName() == netId {
						freeCnt := net.getFreeAddressCount()
						if maxFreeNet == nil || maxFreeCnt < freeCnt {
							maxFreeNetif = &netifs[i]
							maxFreeNet = &net
						}
					}
				}
			}
		}
	}
	return maxFreeNetif, maxFreeNet
}

func (self *SHost) GetNetworkWithIdAndCredential(netId string, userCred mcclient.TokenCredential, reserved bool) (*SNetwork, error) {
	net, err := self.getNetworkWithIdAndCredential(netId, userCred, true, reserved)
	if err == nil {
		return net, nil
	}
	return self.getNetworkWithIdAndCredential(netId, userCred, false, reserved)
}

func (self *SHost) getNetworkWithIdAndCredential(netId string, userCred mcclient.TokenCredential, isPublic bool, reserved bool) (*SNetwork, error) {
	networks := NetworkManager.Query().SubQuery()
	hostwires := HostwireManager.Query().SubQuery()
	hosts := HostManager.Query().SubQuery()

	q := networks.Query()
	q = q.Join(hostwires, sqlchemy.AND(sqlchemy.Equals(hostwires.Field("wire_id"), networks.Field("wire_id")),
		sqlchemy.IsFalse(hostwires.Field("deleted"))))
	q = q.Join(hosts, sqlchemy.AND(sqlchemy.Equals(hosts.Field("id"), hostwires.Field("host_id")),
		sqlchemy.IsFalse(hosts.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(hosts.Field("id"), self.Id))
	q = q.Filter(sqlchemy.OR(sqlchemy.Equals(networks.Field("id"), netId),
		sqlchemy.Equals(networks.Field("name"), netId)))
	if isPublic {
		q = q.Filter(sqlchemy.IsTrue(networks.Field("is_public")))
	} else {
		q = q.Filter(sqlchemy.Equals(networks.Field("tenant_id"), userCred.GetProjectId()))
	}

	nets := make([]SNetwork, 0)
	err := db.FetchModelObjects(NetworkManager, q, &nets)
	if err != nil {
		return nil, err
	}
	var maxFreeNet *SNetwork
	maxFrees := 0
	for i := 0; i < len(nets); i += 1 {
		freeCnt := nets[i].getFreeAddressCount()
		if maxFreeNet == nil || maxFrees < freeCnt {
			maxFrees = freeCnt
			maxFreeNet = &nets[i]
		}
	}
	if reserved || maxFrees > 0 {
		return maxFreeNet, nil
	}
	return nil, fmt.Errorf("No IP address")
}

func (manager *SHostManager) FetchHostById(hostId string) *SHost {
	host := SHost{}
	host.SetModelManager(manager)
	err := manager.Query().Equals("id", hostId).First(&host)
	if err != nil {
		log.Errorf("fetchHostById fail %s", err)
		return nil
	} else {
		return &host
	}
}

func (manager *SHostManager) totalCountQ(
	userCred mcclient.TokenCredential,
	rangeObj db.IStandaloneModel,
	hostStatus, status string,
	hostTypes []string,
	resourceTypes []string,
	providers []string,
	enabled, isBaremetal tristate.TriState,
) *sqlchemy.SQuery {
	hosts := manager.Query().SubQuery()
	q := hosts.Query()
	if len(status) > 0 {
		q = q.Filter(sqlchemy.Equals(hosts.Field("status"), status))
	}
	if len(hostStatus) > 0 {
		q = q.Filter(sqlchemy.Equals(hosts.Field("host_status"), hostStatus))
	}
	if !enabled.IsNone() {
		cond := sqlchemy.IsFalse
		if enabled.Bool() {
			cond = sqlchemy.IsTrue
		}
		q = q.Filter(cond(hosts.Field("enabled")))
	}
	if !isBaremetal.IsNone() {
		cond := sqlchemy.IsFalse
		if isBaremetal.Bool() {
			cond = sqlchemy.IsTrue
		}
		q = q.Filter(cond(hosts.Field("is_baremetal")))
	}
	q = AttachUsageQuery(q, hosts, hostTypes, resourceTypes, providers, rangeObj)
	return q
}

type HostStat struct {
	MemSize     int
	MemReserved int
	MemCmtbound float32
	CpuCount    int8
	CpuReserved int8
	CpuCmtbound float32
	StorageSize int
}

type HostsCountStat struct {
	StorageSize   int64
	Count         int64
	Memory        int64
	MemoryVirtual float64
	CPU           int64
	CPUVirtual    float64
}

func (manager *SHostManager) calculateCount(q *sqlchemy.SQuery) HostsCountStat {
	usableSize := func(act, reserved int) int {
		aSize := 0
		if reserved > 0 && reserved < act {
			aSize = act - reserved
		} else {
			aSize = act
		}
		return aSize
	}
	var (
		tStore int64   = 0
		tCnt   int64   = 0
		tMem   int64   = 0
		tVmem  float64 = 0.0
		tCPU   int64   = 0
		tVCPU  float64 = 0.0
	)
	stats := make([]HostStat, 0)
	err := q.All(&stats)
	if err != nil {
		log.Errorf("%v", err)
	}
	for _, stat := range stats {
		if stat.MemSize == 0 {
			continue
		}
		tCnt += 1
		if stat.StorageSize > 0 {
			tStore += int64(stat.StorageSize)
		}
		aMem := usableSize(stat.MemSize, stat.MemReserved)
		aCpu := usableSize(int(stat.CpuCount), int(stat.CpuReserved))
		tMem += int64(aMem)
		tCPU += int64(aCpu)
		if stat.MemCmtbound <= 0.0 {
			stat.MemCmtbound = options.Options.DefaultMemoryOvercommitBound
		}
		if stat.CpuCmtbound <= 0.0 {
			stat.CpuCmtbound = options.Options.DefaultCPUOvercommitBound
		}
		tVmem += float64(float32(aMem) * stat.MemCmtbound)
		tVCPU += float64(float32(aCpu) * stat.CpuCmtbound)
	}
	return HostsCountStat{
		StorageSize:   tStore,
		Count:         tCnt,
		Memory:        tMem,
		MemoryVirtual: tVmem,
		CPU:           tCPU,
		CPUVirtual:    tVCPU,
	}
}

func (manager *SHostManager) TotalCount(
	userCred mcclient.TokenCredential,
	rangeObj db.IStandaloneModel,
	hostStatus, status string,
	hostTypes []string,
	resourceTypes []string,
	providers []string,
	enabled, isBaremetal tristate.TriState,
) HostsCountStat {
	return manager.calculateCount(manager.totalCountQ(userCred, rangeObj, hostStatus, status, hostTypes, resourceTypes, providers, enabled, isBaremetal))
}

/*
func (self *SHost) GetIZone() (cloudprovider.ICloudZone, error) {
	provider, err := self.GetCloudProvider()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovide for host: %s", err)
	}
	zone := self.GetZone()
	if zone == nil {
		return nil, fmt.Errorf("no zone for host???")
	}
	region := zone.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("No region for zone???")
	}
	iregion, err := provider.GetIRegionById(region.ExternalId)
	if err != nil {
		return nil, fmt.Errorf("fail to find iregion by id %s", err)
	}
	izone, err := iregion.GetIZoneById(zone.ExternalId)
	if err != nil {
		return nil, fmt.Errorf("fail to find izone by id %s", err)
	}
	return izone, nil
}
*/

func (self *SHost) GetIHost() (cloudprovider.ICloudHost, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovide for host: %s", err)
	}
	var iregion cloudprovider.ICloudRegion
	if provider.IsOnPremiseInfrastructure() {
		iregion, err = provider.GetOnPremiseIRegion()
	} else {
		region := self.GetRegion()
		if region == nil {
			msg := "fail to find region of host???"
			log.Errorf(msg)
			return nil, fmt.Errorf(msg)
		}
		iregion, err = provider.GetIRegionById(region.ExternalId)
	}
	if err != nil {
		log.Errorf("fail to find iregion: %s", err)
		return nil, err
	}
	ihost, err := iregion.GetIHostById(self.ExternalId)
	if err != nil {
		log.Errorf("fail to find ihost by id %s %s", self.ExternalId, err)
		return nil, fmt.Errorf("fail to find ihost by id %s", err)
	}
	return ihost, nil
}

func (self *SHost) GetIRegion() (cloudprovider.ICloudRegion, error) {
	provider, err := self.GetDriver()
	if err != nil {
		return nil, fmt.Errorf("No cloudprovide for host %s: %s", self.Name, err)
	}
	region := self.GetRegion()
	if region == nil {
		return nil, fmt.Errorf("failed to find host %s region info", self.Name)
	}
	iregion, err := provider.GetIRegionById(region.ExternalId)
	if err != nil {
		msg := fmt.Sprintf("fail to find iregion by id %s: %v", region.ExternalId, err)
		return nil, fmt.Errorf(msg)
	}
	return iregion, nil
}

func (self *SHost) getDiskConfig() jsonutils.JSONObject {
	bs := self.GetBaremetalstorage()
	if bs != nil {
		return bs.Config
	}
	return nil
}

func (self *SHost) GetBaremetalServer() *SGuest {
	if !self.IsBaremetal {
		return nil
	}
	guestObj, err := db.NewModelObject(GuestManager)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	q := GuestManager.Query().Equals("host_id", self.Id).Equals("hypervisor", HOST_TYPE_BAREMETAL)
	err = q.First(guestObj)
	if err != nil {
		log.Errorf("query fail %s", err)
		return nil
	}
	return guestObj.(*SGuest)
}

func (self *SHost) getSchedtags() []SSchedtag {
	tags := make([]SSchedtag, 0)
	schedtags := SchedtagManager.Query().SubQuery()
	hostschedtags := HostschedtagManager.Query().SubQuery()
	q := schedtags.Query()
	q = q.Join(hostschedtags, sqlchemy.AND(sqlchemy.Equals(hostschedtags.Field("schedtag_id"), schedtags.Field("id")),
		sqlchemy.IsFalse(hostschedtags.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(hostschedtags.Field("host_id"), self.Id))
	err := db.FetchModelObjects(SchedtagManager, q, &tags)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return tags
}

type SHostGuestResourceUsage struct {
	GuestCount     int
	GuestVcpuCount int
	GuestVmemSize  int
}

func (self *SHost) getGuestsResource(status string) *SHostGuestResourceUsage {
	guests := GuestManager.Query().SubQuery()
	q := guests.Query(sqlchemy.COUNT("guest_count"),
		sqlchemy.SUM("guest_vcpu_count", guests.Field("vcpu_count")),
		sqlchemy.SUM("guest_vmem_size", guests.Field("vmem_size")))
	cond := sqlchemy.OR(sqlchemy.Equals(q.Field("host_id"), self.Id),
		sqlchemy.Equals(q.Field("backup_host_id"), self.Id))
	q = q.Filter(cond)
	if len(status) > 0 {
		q = q.Equals("status", status)
	}
	stat := SHostGuestResourceUsage{}
	err := q.First(&stat)
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	return &stat
}

func (self *SHost) getMoreDetails(ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	/*zone := self.GetZone()
	if zone != nil {
		extra.Add(jsonutils.NewString(zone.Id), "zone_id")
		extra.Add(jsonutils.NewString(zone.Name), "zone")
		if len(zone.ExternalId) > 0 {
			extra.Add(jsonutils.NewString(zone.ExternalId), "")
		}
		region := zone.GetRegion()
		if region != nil {
			extra.Add(jsonutils.NewString(zone.GetRegion().GetName()), "region")
			extra.Add(jsonutils.NewString(zone.GetRegion().GetId()), "region_id")
		}
	}*/

	info := self.getCloudProviderInfo()
	extra.Update(jsonutils.Marshal(&info))

	server := self.GetBaremetalServer()
	if server != nil {
		extra.Add(jsonutils.NewString(server.Id), "server_id")
		extra.Add(jsonutils.NewString(server.Name), "server")
	}
	netifs := self.GetNetInterfaces()
	if netifs != nil && len(netifs) > 0 {
		nicInfos := []jsonutils.JSONObject{}
		for i := 0; i < len(netifs); i += 1 {
			nicInfo := netifs[i].getBaremetalJsonDesc()
			if nicInfo == nil {
				log.Errorf("netif %s get baremetal desc failed", netifs[i].GetId())
				continue
			}
			nicInfos = append(nicInfos, nicInfo)
		}
		extra.Add(jsonutils.NewInt(int64(len(nicInfos))), "nic_count")
		extra.Add(jsonutils.NewArray(nicInfos...), "nic_info")
	}
	schedtags := self.getSchedtags()
	if schedtags != nil && len(schedtags) > 0 {
		info := make([]jsonutils.JSONObject, len(schedtags))
		for i := 0; i < len(schedtags); i += 1 {
			info[i] = schedtags[i].GetShortDesc(ctx)
		}
		extra.Add(jsonutils.NewArray(info...), "schedtags")
	}
	var usage *SHostGuestResourceUsage
	if options.Options.IgnoreNonrunningGuests {
		usage = self.getGuestsResource(VM_RUNNING)
	} else {
		usage = self.getGuestsResource("")
	}
	if usage != nil {
		extra.Add(jsonutils.NewInt(int64(usage.GuestVcpuCount)), "cpu_commit")
		extra.Add(jsonutils.NewInt(int64(usage.GuestVmemSize)), "mem_commit")
	}
	extra.Add(jsonutils.NewInt(int64(self.GetGuestCount())), "guests")
	extra.Add(jsonutils.NewInt(int64(self.GetNonsystemGuestCount())), "nonsystem_guests")
	extra.Add(jsonutils.NewInt(int64(self.GetRunningGuestCount())), "running_guests")
	totalCpu := self.GetCpuCount()
	cpuCommitRate := 0.0
	if totalCpu > 0 && usage.GuestVcpuCount > 0 {
		cpuCommitRate = float64(usage.GuestVcpuCount) * 1.0 / float64(totalCpu)
	}
	extra.Add(jsonutils.NewFloat(cpuCommitRate), "cpu_commit_rate")
	totalMem := self.GetMemSize()
	memCommitRate := 0.0
	if totalMem > 0 && usage.GuestVmemSize > 0 {
		memCommitRate = float64(usage.GuestVmemSize) * 1.0 / float64(totalMem)
	}
	extra.Add(jsonutils.NewFloat(memCommitRate), "mem_commit_rate")
	capa := self.GetAttachedStorageCapacity()
	extra.Add(jsonutils.NewInt(int64(capa.Capacity)), "storage")
	extra.Add(jsonutils.NewInt(int64(capa.Used)), "storage_used")
	extra.Add(jsonutils.NewInt(int64(capa.Wasted)), "storage_waste")
	extra.Add(jsonutils.NewInt(int64(capa.VCapacity)), "storage_virtual")
	extra.Add(jsonutils.NewInt(int64(capa.GetFree())), "storage_free")
	extra.Add(self.GetHardwareSpecification(), "spec")

	// extra = self.SManagedResourceBase.getExtraDetails(ctx, extra)

	if self.IsPrepaidRecycle() {
		extra.Add(jsonutils.JSONTrue, "is_prepaid_recycle")
	} else {
		extra.Add(jsonutils.JSONFalse, "is_prepaid_recycle")
	}

	return extra
}

func (self *SHost) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(ctx, extra)
}

func (self *SHost) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(ctx, extra), nil
}

func (self *SHost) AllowGetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "vnc")
}

func (self *SHost) GetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{BAREMETAL_READY, BAREMETAL_RUNNING}) {
		retval := jsonutils.NewDict()
		retval.Set("host_id", jsonutils.NewString(self.Id))
		zone := self.GetZone()
		retval.Set("zone", jsonutils.NewString(zone.GetName()))
		return retval, nil
	}
	return jsonutils.NewDict(), nil
}

func (self *SHost) AllowGetDetailsIpmi(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowGetSpec(userCred, self, "ipmi")
}

func (self *SHost) GetDetailsIpmi(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret, ok := self.IpmiInfo.(*jsonutils.JSONDict)
	if !ok {
		return nil, httperrors.NewNotFoundError("No ipmi information was found for host %s", self.Name)
	}
	password, err := ret.GetString("password")
	if err != nil {
		return nil, httperrors.NewNotFoundError("IPMI has no password information")
	}
	descryptedPassword, err := utils.DescryptAESBase64(self.Id, password)
	if err != nil {
		return nil, err
	}
	ret.Set("password", jsonutils.NewString(descryptedPassword))
	return ret, nil
}

func (manager *SHostManager) GetHostsByManagerAndRegion(managerId string, regionId string) []SHost {
	zones := ZoneManager.Query().Equals("cloudregion_id", regionId).SubQuery()
	hosts := HostManager.Query()
	q := hosts.Equals("manager_id", managerId)
	q = q.Join(zones, sqlchemy.Equals(zones.Field("id"), hosts.Field("zone_id")))
	ret := make([]SHost, 0)
	err := db.FetchModelObjects(HostManager, q, &ret)
	if err != nil {
		log.Errorf("GetHostsByManagerAndRegion fail %s", err)
		return nil
	}
	return ret
}

/*func (self *SHost) StartImageCacheTask(ctx context.Context, userCred mcclient.TokenCredential, imageId, parentTaskId string, isForce bool) error {
	//Todo
	// HostcachedimagesManager.Register(userCred, self, imageId)
	data := jsonutils.NewDict()
	data.Set("image_id", jsonutils.NewString(imageId))
	if isForce {
		data.Set("is_force", jsonutils.JSONTrue)
	}
	task, err := taskman.TaskManager.NewTask(ctx, "StorageCacheImageTask", self, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}*/

func (self *SHost) Request(ctx context.Context, userCred mcclient.TokenCredential, method httputils.THttpMethod, url string, headers http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	s := auth.GetSession(ctx, userCred, "", "")
	_, ret, err := s.JSONRequest(self.ManagerUri, "", method, url, headers, body)
	return ret, err
}

func (self *SHost) GetLocalStoragecache() *SStoragecache {
	localStorages := self.GetAttachedStorages(STORAGE_LOCAL)
	for i := 0; i < len(localStorages); i += 1 {
		sc := localStorages[i].GetStoragecache()
		if sc != nil {
			return sc
		}
	}
	return nil
}

func (self *SHost) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	kwargs := data.(*jsonutils.JSONDict)
	ipmiInfo, err := self.FetchIpmiInfo(kwargs)
	if err != nil {
		log.Errorln(err.Error())
		return
	}
	if ipmiInfo.Length() > 0 {
		_, err := self.SaveUpdates(func() error {
			self.IpmiInfo = ipmiInfo
			return nil
		})
		if err != nil {
			log.Errorln(err.Error())
		}
	}
}

func (manager *SHostManager) ValidateSizeParams(data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	memStr, _ := data.GetString("mem_size")
	if len(memStr) > 0 {
		if !regutils.MatchSize(memStr) {
			return nil, fmt.Errorf("Memory size must be number[+unit], like 256M, 1G or 256")
		}
		memSize, err := fileutils.GetSizeMb(memStr, 'M', 1024)
		if err != nil {
			return nil, err
		}
		data.Set("mem_size", jsonutils.NewInt(int64(memSize)))
	}
	memReservedStr, _ := data.GetString("mem_reserved")
	if len(memReservedStr) > 0 {
		if !regutils.MatchSize(memReservedStr) {
			return nil, fmt.Errorf("Memory size must be number[+unit], like 256M, 1G or 256")
		}
		memSize, err := fileutils.GetSizeMb(memReservedStr, 'M', 1024)
		if err != nil {
			return nil, err
		}
		data.Set("mem_reserved", jsonutils.NewInt(int64(memSize)))
	}
	cpuCacheStr, _ := data.GetString("cpu_cache")
	if len(cpuCacheStr) > 0 {
		if !regutils.MatchSize(cpuCacheStr) {
			return nil, fmt.Errorf("Illegal cpu cache size %s", cpuCacheStr)
		}
		cpuCache, err := fileutils.GetSizeKb(cpuCacheStr, 'K', 1024)
		if err != nil {
			return nil, err
		}
		data.Set("cpu_cache", jsonutils.NewInt(int64(cpuCache)))
	}
	return data, nil
}

func (manager *SHostManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	zoneId, _ := data.GetString("zone_id")
	if len(zoneId) > 0 && ZoneManager.Query().Equals("id", zoneId).Count() == 0 {
		return nil, httperrors.NewInputParameterError("Zone id %s not found", zoneId)
	}
	mangerUri, err := data.GetString("manager_uri")
	if err == nil {
		count := manager.Query().Equals("manager_uri", mangerUri).Count()
		if count > 0 {
			return nil, httperrors.NewConflictError("Conflict manager_uri %s", mangerUri)
		}
	}
	accessIp, err := data.GetString("access_ip")
	if err == nil {
		count := manager.Query().Equals("access_ip", accessIp).Count()
		if count > 0 {
			return nil, httperrors.NewDuplicateResourceError("Duplicate access_ip %s", accessIp)
		}
	}
	accessMac, err := data.GetString("access_mac")
	if err == nil {
		count := HostManager.Query().Equals("access_mac", accessMac).Count()
		if count > 0 {
			return nil, httperrors.NewDuplicateResourceError("Duplicate access_mac %s", accessMac)
		}
	}
	data, err = manager.ValidateSizeParams(data)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	memReserved, err := data.Int("mem_reserved")
	if err != nil {
		hostType, _ := data.GetString("host_type")
		if hostType != HOST_TYPE_BAREMETAL {
			memSize, _ := data.Int("mem_size")
			memReserved = memSize / 8
			if memReserved > 4096 {
				memReserved = 4096
			}
			data.Set("mem_reserved", jsonutils.NewInt(memReserved))
		} else {
			data.Set("mem_reserved", jsonutils.NewInt(0))
		}
	}
	return manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SHost) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	mangerUri, err := data.GetString("manager_uri")
	if err == nil {
		count := HostManager.Query().Equals("manager_uri", mangerUri).
			NotEquals("id", self.Id).Equals("zone_id", self.ZoneId).Count()
		if count > 0 {
			return nil, httperrors.NewConflictError("Conflict manager_uri %s", mangerUri)
		}
	}
	accessIp, err := data.GetString("access_ip")
	if err == nil {
		count := HostManager.Query().Equals("access_ip", accessIp).
			NotEquals("id", self.Id).Equals("zone_id", self.ZoneId).Count()
		if count > 0 {
			return nil, httperrors.NewDuplicateResourceError("Duplicate access_ip %s", accessIp)
		}
	}
	accessMac, err := data.GetString("access_mac")
	if err == nil {
		accessMac = netutils.FormatMacAddr(accessMac)
		if len(accessMac) == 0 {
			return nil, httperrors.NewInputParameterError("invalid access_mac address")
		}
		q := HostManager.Query().Equals("access_mac", accessMac).NotEquals("id", self.Id)
		if q.Count() > 0 {
			return nil, httperrors.NewDuplicateResourceError("Duplicate access_mac %s", accessMac)
		}
	}
	data, err = HostManager.ValidateSizeParams(data)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	ipmiInfo, err := self.FetchIpmiInfo(data)
	if err != nil {
		return nil, err
	}
	if ipmiInfo.Length() > 0 {
		val := jsonutils.NewDict()
		val.Update(self.IpmiInfo)
		val.Update(ipmiInfo)
		data.Set("ipmi_info", val)
	}
	data, err = self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, data)
	if err != nil {
		return nil, err
	}
	if data.Contains("name") {
		self.UpdateDnsRecords(false)
	}
	return data, nil
}

func (self *SHost) UpdateDnsRecords(isAdd bool) {
	for _, netif := range self.GetNetInterfaces() {
		self.UpdateDnsRecord(&netif, isAdd)
	}
}

func (self *SHost) UpdateDnsRecord(netif *SNetInterface, isAdd bool) {
	name := self.GetNetifName(netif)
	if len(name) == 0 {
		return
	}
	bn := netif.GetBaremetalNetwork()
	if bn == nil {
		log.Errorf("Interface %s not enable", netif.GetId())
		return
	}
	net := bn.GetNetwork()
	if net == nil {
		log.Errorf("BaremetalNetwoke %s not found network", bn.GetId())
	}
	net._updateDnsRecord(name, bn.IpAddr, isAdd)
}

func (self *SHost) GetNetifName(netif *SNetInterface) string {
	if netif.NicType == NIC_TYPE_IPMI {
		return self.GetName()
	} else if netif.NicType == NIC_TYPE_ADMIN {
		return self.GetName() + "-admin"
	}
	return ""
}

func (self *SHost) FetchIpmiInfo(data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	IPMI_KEY_PERFIX := "ipmi_"
	ipmiInfo := jsonutils.NewDict()
	kv, _ := data.GetMap()
	var err error
	for key := range kv {
		if strings.HasPrefix(key, IPMI_KEY_PERFIX) {
			value, _ := data.GetString(key)
			subkey := key[len(IPMI_KEY_PERFIX):]
			data.Remove(key)
			if subkey == "password" {
				value, err = utils.EncryptAESBase64(self.Id, value)
				if err != nil {
					log.Errorf("encrypt password failed %s", err)
					return nil, err
				}
			} else if subkey == "ip_addr" {
				if !regutils.MatchIP4Addr(value) {
					log.Errorf("%s: %s not match ip address", key, value)
					continue
				}
			}
			ipmiInfo.Set(subkey, jsonutils.NewString(value))
		}
	}
	return ipmiInfo, nil
}

func (self *SHost) AllowPerformStart(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "start")
}

func (self *SHost) PerformStart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.IsBaremetal {
		return nil, httperrors.NewBadRequestError("Cannot start a non-baremetal host")
	}
	if !utils.IsInStringArray(self.Status, []string{BAREMETAL_READY}) {
		return nil, httperrors.NewInvalidStatusError("Cannot start baremetal with active guest")
	}
	guest := self.GetBaremetalServer()
	if guest != nil {
		if self.HostType != HOST_TYPE_BAREMETAL {
			if !utils.IsInStringArray(guest.Status, []string{VM_ADMIN}) {
				return nil, httperrors.NewBadRequestError("Cannot start baremetal with active guest")
			}
		} else {
			if utils.ToBool(guest.GetMetadata("is_fake_baremetal_server", userCred)) {
				return nil, self.InitializedGuestStart(ctx, userCred, guest)
			}
			self.SetStatus(userCred, BAREMETAL_START_MAINTAIN, "")
			return guest.PerformStart(ctx, userCred, query, data)
		}
	}
	params := jsonutils.NewDict()
	params.Set("force_reboot", jsonutils.NewBool(false))
	params.Set("action", jsonutils.NewString("start"))
	return self.PerformMaintenance(ctx, userCred, nil, params)
}

func (self *SHost) AllowPerformStop(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "stop")
}

func (self *SHost) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.IsBaremetal {
		return nil, httperrors.NewBadRequestError("Cannot stop a non-baremetal host")
	}
	if !utils.IsInStringArray(self.Status, []string{BAREMETAL_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot stop baremetal with non-active guest")
	}
	guest := self.GetBaremetalServer()
	if guest != nil {
		if self.HostType != HOST_TYPE_BAREMETAL {
			if !utils.IsInStringArray(guest.Status, []string{VM_ADMIN}) {
				return nil, httperrors.NewBadRequestError("Cannot stop baremetal with active guest")
			}
		} else {
			if utils.ToBool(guest.GetMetadata("is_fake_baremetal_server", userCred)) {
				return nil, self.InitializedGuestStop(ctx, userCred, guest)
			}
			self.SetStatus(userCred, BAREMETAL_START_MAINTAIN, "")
			return guest.PerformStop(ctx, userCred, query, data)
		}
	}
	return nil, self.StartBaremetalUnmaintenanceTask(ctx, userCred, false, "stop")
}

func (self *SHost) InitializedGuestStart(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalServerStartTask", guest, userCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHost) InitializedGuestStop(ctx context.Context, userCred mcclient.TokenCredential, guest *SGuest) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalServerStopTask", guest, userCred, nil, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHost) AllowPerformMaintenance(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "maintenance")

}

func (self *SHost) PerformMaintenance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{BAREMETAL_READY, BAREMETAL_RUNNING}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do maintenance in status %s", self.Status)
	}
	guest := self.GetBaremetalServer()
	if guest != nil && !utils.IsInStringArray(guest.Status, []string{VM_READY, VM_RUNNING, VM_ADMIN}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do maintenance while guest status %s", guest.Status)
	}
	params := jsonutils.NewDict()
	if guest != nil {
		if guest.Status == VM_RUNNING {
			params.Set("guest_running", jsonutils.NewBool(true))
		}
		guest.SetStatus(userCred, VM_ADMIN, "")
	}
	if self.Status == BAREMETAL_RUNNING && jsonutils.QueryBoolean(data, "force_reboot", false) {
		params.Set("force_reboot", jsonutils.NewBool(true))
	}
	action := "maintenance"
	if data.Contains("action") {
		action, _ = data.GetString("action")
	}
	params.Set("action", jsonutils.NewString(action))
	self.SetStatus(userCred, BAREMETAL_START_MAINTAIN, "")
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalMaintenanceTask", self, userCred, params, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (self *SHost) AllowPerformUnmaintenance(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "unmaintenance")
}

func (self *SHost) PerformUnmaintenance(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{BAREMETAL_RUNNING, BAREMETAL_READY}) {
		return nil, httperrors.NewInvalidStatusError("Cannot do unmaintenance in status %s", self.Status)
	}
	guest := self.GetBaremetalServer()
	if guest != nil && guest.Status != VM_ADMIN {
		return nil, httperrors.NewInvalidStatusError("Wrong guest status %s", guest.Status)
	}
	action, _ := data.GetString("action")
	if len(action) == 0 {
		action = "unmaintenance"
	}
	guestRunning := self.GetMetadata("__maint_guest_running", userCred)
	var startGuest = false
	if utils.ToBool(guestRunning) {
		startGuest = true
	}
	return nil, self.StartBaremetalUnmaintenanceTask(ctx, userCred, startGuest, action)
}

func (self *SHost) StartBaremetalUnmaintenanceTask(ctx context.Context, userCred mcclient.TokenCredential, startGuest bool, action string) error {
	self.SetStatus(userCred, BAREMETAL_START_MAINTAIN, "")
	params := jsonutils.NewDict()
	params.Set("guest_running", jsonutils.NewBool(startGuest))
	if len(action) == 0 {
		action = "unmaintenance"
	}
	params.Set("action", jsonutils.NewString(action))
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalUnmaintenanceTask", self, userCred, params, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHost) IsBaremetalAgentReady() bool {
	url, err := auth.GetServiceURL("baremetal", options.Options.Region, self.GetZone().GetName(), "")
	if err != nil {
		log.Errorln("is baremetal agent ready: false")
		return false
	}
	log.Infof("baremetal url:%s", url)
	return true
}

func (self *SHost) BaremetalSyncRequest(ctx context.Context, method httputils.THttpMethod, url string, headers http.Header, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	serviceUrl, err := auth.GetServiceURL("baremetal", options.Options.Region, self.GetZone().GetName(), "")
	if err != nil {
		return nil, err
	}
	url = serviceUrl + url
	_, data, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, method, url, headers, body, false)
	return data, err
}

func (self *SHost) StartSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalSyncStatusTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SHost) AllowPerformOffline(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "offline")
}

func (self *SHost) PerformOffline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.HostStatus != HOST_OFFLINE {
		_, err := self.SaveUpdates(func() error {
			self.HostStatus = HOST_OFFLINE
			return nil
		})
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_OFFLINE, "", userCred)
		logclient.AddActionLog(self, logclient.ACT_ONLINE, nil, userCred, true)
		self.SyncAttachedStorageStatus()
	}
	return nil, nil
}

func (self *SHost) AllowPerformOnline(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "online")
}

func (self *SHost) PerformOnline(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.HostStatus != HOST_ONLINE {
		_, err := self.SaveUpdates(func() error {
			self.LastPingAt = time.Now()
			self.HostStatus = HOST_ONLINE
			self.Status = BAREMETAL_RUNNING
			return nil
		})
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(self, db.ACT_ONLINE, "", userCred)
		logclient.AddActionLog(self, logclient.ACT_ONLINE, nil, userCred, true)
		self.SyncAttachedStorageStatus()
		self.StartSyncAllGuestsStatusTask(ctx, userCred)
	}
	return nil, nil
}

func (self *SHost) StartSyncAllGuestsStatusTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalSyncAllGuestsStatusTask", self, userCred, nil, "", "", nil); err != nil {
		log.Errorf(err.Error())
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SHost) AllowPerformPing(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "ping")
}

func (self *SHost) PerformPing(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.HostStatus != HOST_ONLINE {
		self.PerformOnline(ctx, userCred, query, data)
	} else {
		self.SaveUpdates(func() error {
			self.LastPingAt = time.Now()
			return nil
		})
	}
	result := jsonutils.NewDict()
	result.Set("name", jsonutils.NewString(self.GetName()))
	dependSvcs := []string{"ntpd", "kafka", "influxdb", "elasticsearch"}
	catalog := auth.GetCatalogData(dependSvcs, options.Options.Region)
	if catalog == nil {
		return nil, fmt.Errorf("Get catalog error")
	}
	result.Set("catalog", catalog)

	appParams := appsrv.AppContextGetParams(ctx)
	if appParams != nil {
		// skip log&trace, when everything is normal
		appParams.SkipTrace = true
		appParams.SkipLog = true
	}

	return result, nil
}

func (self *SHost) AllowPerformPrepare(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "prepare")
}

func (self *SHost) PerformPrepare(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if utils.IsInStringArray(self.Status, []string{BAREMETAL_READY, BAREMETAL_RUNNING, BAREMETAL_PREPARE_FAIL}) {
		var onfinish string
		if self.GetBaremetalServer() != nil {
			if self.Status == BAREMETAL_RUNNING {
				onfinish = "restart"
			} else if self.Status == BAREMETAL_READY {
				onfinish = "shotdown"
			}
		}
		return nil, self.StartPrepareTask(ctx, userCred, onfinish, "")
	}
	return nil, httperrors.NewInvalidStatusError("Cannot prepare baremetal in status %s", self.Status)
}

func (self *SHost) StartPrepareTask(ctx context.Context, userCred mcclient.TokenCredential, onfinish, parentTaskId string) error {
	data := jsonutils.NewDict()
	if len(onfinish) > 0 {
		data.Set("on_finish", jsonutils.NewString(onfinish))
	}
	self.SetStatus(userCred, BAREMETAL_PREPARE, "")
	if task, err := taskman.TaskManager.NewTask(ctx, "BaremetalPrepareTask", self, userCred, data, parentTaskId, "", nil); err != nil {
		log.Errorf(err.Error())
		return err
	} else {
		task.ScheduleRun(nil)
		return nil
	}
}

func (self *SHost) AllowPerformAddNetif(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "add-netif")
}

func (self *SHost) PerformAddNetif(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	mac, _ := data.GetString("mac")
	if len(mac) == 0 || len(netutils.FormatMacAddr(mac)) == 0 {
		return nil, httperrors.NewBadRequestError("Invaild mac address")
	}
	wire, _ := data.GetString("wire")
	ipAddr, _ := data.GetString("ip_addr")
	rate, _ := data.Int("rate")
	nicType, _ := data.GetString("nic_type")
	index, _ := data.Int("index")
	linkUp, _ := data.GetString("link_up")
	mtu, _ := data.Int("mtu")
	reset := jsonutils.QueryBoolean(data, "reset", false)
	strInterface, _ := data.GetString("interface")
	bridge, _ := data.GetString("bridge")
	reserve := jsonutils.QueryBoolean(data, "reserve", false)
	requireDesignatedIp := jsonutils.QueryBoolean(data, "require_designated_ip", false)

	isLinkUp := tristate.None
	if linkUp != "" {
		if utils.ToBool(linkUp) {
			isLinkUp = tristate.True
		} else {
			isLinkUp = tristate.False
		}
	}

	err := self.addNetif(ctx, userCred, mac, wire, ipAddr, int(rate), nicType, int8(index), isLinkUp,
		int16(mtu), reset, strInterface, bridge, reserve, requireDesignatedIp)
	return nil, err
}

func (self *SHost) addNetif(ctx context.Context, userCred mcclient.TokenCredential,
	mac string, wire string, ipAddr string,
	rate int, nicType string, index int8, linkUp tristate.TriState, mtu int16,
	reset bool, strInterface string, bridge string,
	reserve bool, requireDesignatedIp bool,
) error {
	var sw *SWire
	if len(wire) > 0 && len(ipAddr) == 0 {
		iWire, err := WireManager.FetchByIdOrName(userCred, wire)
		if err != nil {
			return httperrors.NewBadRequestError("Wire %s not found", wire)
		}
		sw = iWire.(*SWire)
	} else if len(ipAddr) > 0 && len(wire) == 0 {
		ipWire, err := WireManager.GetWireOfIp(ipAddr)
		if err != nil {
			return httperrors.NewBadRequestError("IP %s not attach to any wire", ipAddr)
		}
		sw = ipWire
	} else if len(wire) > 0 && len(ipAddr) > 0 {
		ipWire, err := WireManager.GetWireOfIp(ipAddr)
		if err != nil {
			return httperrors.NewBadRequestError("IP %s not attach to any wire", ipAddr)
		}
		if ipWire.Id != wire && ipWire.GetName() != wire {
			return httperrors.NewBadRequestError("IP %s not attach to wire %s", ipAddr, wire)
		}
		sw = ipWire
	}
	netif, err := NetInterfaceManager.FetchByMac(mac)
	if err != nil {
		netif = &SNetInterface{}
		netif.Mac = mac
		netif.BaremetalId = self.Id
		if sw != nil {
			netif.WireId = sw.Id
		}
		netif.Rate = rate
		netif.NicType = nicType
		netif.Index = index
		if !linkUp.IsNone() {
			netif.LinkUp = linkUp.Bool()
		}
		netif.Mtu = mtu
		err = NetInterfaceManager.TableSpec().Insert(netif)
		if err != nil {
			return err
		}
	} else {
		var changed = false
		_, err := NetInterfaceManager.TableSpec().Update(netif, func() error {
			if netif.BaremetalId != self.Id {
				changed = true
				netif.BaremetalId = self.Id
			}
			if sw != nil && netif.WireId != sw.Id {
				changed = true
				netif.WireId = sw.Id
			}
			if rate > 0 && rate != netif.Rate {
				netif.Rate = rate
			}
			if nicType != "" && nicType != netif.NicType {
				netif.NicType = nicType
			}
			if index >= 0 && index != netif.Index {
				netif.Index = index
			}
			if !linkUp.IsNone() && linkUp.Bool() != netif.LinkUp {
				netif.LinkUp = linkUp.Bool()
			}
			if mtu > 0 && mtu != netif.Mtu {
				netif.Mtu = mtu
			}
			return nil
		})
		if err != nil {
			return err
		}
		if changed || reset {
			self.DisableNetif(ctx, userCred, netif, false)
		}
	}
	sw = netif.GetWire()
	if sw != nil {
		if len(strInterface) == 0 {
			strInterface = fmt.Sprintf("eth%d", netif.Index)
		}
		if len(strInterface) > 0 {
			if len(bridge) == 0 {
				bridge = fmt.Sprintf("br%s", sw.GetName())
			}
			var isMaster = netif.NicType == NIC_TYPE_ADMIN
			ihw, err := HostwireManager.FetchByIds(self.Id, sw.Id)
			if err != nil {
				hw := &SHostwire{}
				hw.Bridge = bridge
				hw.Interface = strInterface
				hw.HostId = self.Id
				hw.WireId = sw.Id
				hw.IsMaster = isMaster
				hw.MacAddr = mac
				err := HostwireManager.TableSpec().Insert(hw)
				if err != nil {
					return err
				}
			} else {
				hw := ihw.(*SHostwire)
				HostwireManager.TableSpec().Update(hw, func() error {
					hw.Bridge = bridge
					hw.Interface = strInterface
					hw.MacAddr = mac
					hw.IsMaster = isMaster
					return nil
				})
			}
		}
	}
	if len(ipAddr) > 0 {
		err = self.EnableNetif(ctx, userCred, netif, "", ipAddr, "", reserve, requireDesignatedIp)
		if err != nil {
			return httperrors.NewBadRequestError(err.Error())
		}
	}
	return nil
}

func (self *SHost) AllowPerformEnableNetif(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "enable-netif")
}

func (self *SHost) PerformEnableNetif(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	mac, _ := data.GetString("mac")
	netif := self.GetNetInterface(mac)
	if netif == nil {
		return nil, httperrors.NewBadRequestError("Interface %s not exist", mac)
	}
	if !utils.IsInStringArray(netif.NicType, NIC_TYPES) {
		return nil, httperrors.NewBadRequestError("Only ADMIN and IPMI nic can be enable")
	}
	network, _ := data.GetString("network")
	ipAddr, _ := data.GetString("ip_addr")
	allocDir, _ := data.GetString("alloc_dir")
	reserve := jsonutils.QueryBoolean(data, "reserve", false)
	requireDesignatedIp := jsonutils.QueryBoolean(data, "require_designated_ip", false)
	err := self.EnableNetif(ctx, userCred, netif, network, ipAddr, allocDir, reserve, requireDesignatedIp)
	if err != nil {
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	return nil, nil
}

func (self *SHost) EnableNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, network, ipAddr, allocDir string, reserve, requireDesignatedIp bool) error {
	bn := netif.GetBaremetalNetwork()
	if bn != nil {
		return nil
	}
	log.Errorf("==========EnableNetif %#v, net: %s, ipAddr: %s, allocDir: %s, reserve: %v, requireDesignatedIp: %v", netif, network, ipAddr, allocDir, reserve, requireDesignatedIp)
	var net *SNetwork
	var err error
	if len(ipAddr) > 0 {
		net, err = netif.GetCandidateNetworkForIp(userCred, ipAddr)
		if net != nil {
			log.Infof("find network %s for ip %s", net.GetName(), ipAddr)
		} else if requireDesignatedIp {
			log.Errorf("Cannot allocate IP %s, not reachable", ipAddr)
			return fmt.Errorf("Cannot allocate IP %s, not reachable", ipAddr)
		}
	}
	wire := netif.GetWire()
	if wire == nil {
		return fmt.Errorf("No wire attached")
	}
	hw, err := HostwireManager.FetchByIds(self.Id, wire.Id)
	if hw == nil {
		return fmt.Errorf("host not attach to this wire")
	}
	if net == nil {
		if len(network) > 0 {
			iNet, err := NetworkManager.FetchByIdOrName(userCred, network)
			if err != nil {
				return fmt.Errorf("Network %s not found: %s", network, err)
			}
			net = iNet.(*SNetwork)
			if len(net.WireId) == 0 || net.WireId != wire.Id {
				return fmt.Errorf("Network %s not reacheable on mac %s", network, netif.Mac)
			}
		} else {
			net, err = wire.GetCandidatePrivateNetwork(userCred, false, SERVER_TYPE_BAREMETAL)
			if err != nil || net == nil {
				return fmt.Errorf("No network found")
			}
		}
	} else if net.WireId != wire.Id {
		return fmt.Errorf("conflict??? candiate net is not on wire")
	}
	return self.Attach2Network(ctx, userCred, netif, net, ipAddr, allocDir, reserve, requireDesignatedIp)
}

func (self *SHost) AllowPerformDisableNetif(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "disable-netif")
}

func (self *SHost) PerformDisableNetif(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	mac, _ := data.GetString("mac")
	netif := self.GetNetInterface(mac)
	if netif == nil {
		return nil, httperrors.NewBadRequestError("Interface %s not exists", mac)
	}
	reserve := jsonutils.QueryBoolean(data, "reserve", false)
	err := self.DisableNetif(ctx, userCred, netif, reserve)
	if err != nil {
		return nil, httperrors.NewBadRequestError(err.Error())
	}
	return nil, nil
}

func (self *SHost) DisableNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, reserve bool) error {
	bn := netif.GetBaremetalNetwork()
	if bn != nil {
		self.UpdateDnsRecord(netif, false)
		self.DeleteBaremetalnetwork(ctx, userCred, bn, reserve)
	}
	return nil
}

func (self *SHost) Attach2Network(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, net *SNetwork, ipAddr, allocDir string, reserved, requireDesignatedIp bool) error {
	lockman.LockObject(ctx, net)
	defer lockman.ReleaseObject(ctx, net)
	usedAddr := net.GetUsedAddresses()
	freeIp, err := net.GetFreeIP(ctx, userCred, usedAddr, nil, ipAddr, IPAddlocationDirection(allocDir), reserved)
	if err != nil {
		log.Errorf("attach2network: %s", err)
		return err
	}
	if len(ipAddr) > 0 && ipAddr != freeIp && requireDesignatedIp {
		return fmt.Errorf("IP address %s is occupied", ipAddr)
	}
	bn := &SHostnetwork{}
	bn.BaremetalId = self.Id
	bn.NetworkId = net.Id
	bn.IpAddr = freeIp
	bn.MacAddr = netif.Mac
	err = HostnetworkManager.TableSpec().Insert(bn)
	if err != nil {
		log.Errorf("HostnetworkManager.TableSpec().Insert fail %s", err)
		return err
	}
	db.OpsLog.LogAttachEvent(ctx, self, net, userCred, jsonutils.NewString(freeIp))
	self.UpdateDnsRecord(netif, true)
	net.UpdateBaremetalNetmap(bn, self.GetNetifName(netif))
	return nil
}

func (self *SHost) AllowPerformRemoveNetif(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "remove-netif")
}

func (self *SHost) PerformRemoveNetif(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	mac, _ := data.GetString("mac")
	mac = netutils.FormatMacAddr(mac)
	if len(mac) == 0 {
		return nil, httperrors.NewBadRequestError("Invalid mac address")
	}
	netif, err := NetInterfaceManager.FetchByMac(mac)
	if err != nil {
		return nil, httperrors.NewBadRequestError("Fetch netif error %s", err)
	}
	return nil, self.RemoveNetif(ctx, userCred, netif, jsonutils.QueryBoolean(data, "reserve", false))
}

func (self *SHost) RemoveNetif(ctx context.Context, userCred mcclient.TokenCredential, netif *SNetInterface, reserve bool) error {
	wire := netif.GetWire()
	self.DisableNetif(ctx, userCred, netif, reserve)
	log.Infof("Remove wire")
	err := netif.Remove(ctx, userCred)
	if err != nil {
		return err
	}
	if wire != nil {
		log.Infof("Remove wire")
		others := self.GetNetifsOnWire(wire)
		if len(others) == 0 {
			hw, _ := HostwireManager.FetchByIds(self.Id, wire.Id)
			if hw != nil {
				db.OpsLog.LogDetachEvent(ctx, self, wire, userCred, jsonutils.NewString(fmt.Sprintf("disable netif %s", self.AccessMac)))
				log.Infof("Detach host wire because of remove netif %s", netif.Mac)
				return hw.Delete(ctx, userCred)
			}
		}
	}
	return nil
}

func (self *SHost) GetNetifsOnWire(wire *SWire) []SNetInterface {
	dest := make([]SNetInterface, 0)
	q := NetInterfaceManager.Query()
	err := q.Filter(sqlchemy.Equals(q.Field("baremetal_id"), self.Id)).Filter(sqlchemy.Equals(q.Field("wire_id"), wire.Id)).Desc(q.Field("index")).All(&dest)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	return dest
}

func (self *SHost) AllowPerformSyncstatus(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "syncstatus")
}

func (self *SHost) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	self.SetStatus(userCred, BAREMETAL_SYNCING_STATUS, "")
	return nil, self.StartSyncstatus(ctx, userCred, "")
}

func (self *SHost) AllowPerformReset(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "reset")
}

func (self *SHost) PerformReset(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.IsBaremetal {
		return nil, httperrors.NewBadRequestError("Cannot start a non-baremetal host")
	}
	if self.Status != BAREMETAL_RUNNING {
		return nil, httperrors.NewBadRequestError("Cannot reset baremetal in status %s", self.Status)
	}
	guest := self.GetBaremetalServer()
	if guest != nil {
		if self.HostType == HOST_TYPE_BAREMETAL {
			if guest.Status != VM_ADMIN {
				return nil, httperrors.NewBadRequestError("Cannot reset baremetal with active guest")
			}
		} else {
			return guest.PerformReset(ctx, userCred, query, data)
		}
	}
	kwargs := jsonutils.NewDict()
	kwargs.Set("force_reboot", jsonutils.JSONTrue)
	kwargs.Set("action", jsonutils.NewString("reset"))
	return self.PerformMaintenance(ctx, userCred, query, kwargs)
}

func (self *SHost) AllowPerformRemoveAllNetifs(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "remove-all-netifs")
}

func (self *SHost) PerformRemoveAllNetifs(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	netifs := self.GetNetInterfaces()
	for i := 0; i < len(netifs); i++ {
		if !utils.IsInStringArray(netifs[i].NicType, NIC_TYPES) {
			self.RemoveNetif(ctx, userCred, &netifs[i], false)
		}
	}
	return nil, nil
}

func (self *SHost) AllowPerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return self.SEnabledStatusStandaloneResourceBase.AllowPerformEnable(ctx, userCred, query, data)
}

func (self *SHost) PerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if !self.Enabled {
		_, err := self.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, data)
		if err != nil {
			return nil, err
		}
		self.SyncAttachedStorageStatus()
	}
	return nil, nil
}

func (self *SHost) AllowPerformDisable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) bool {
	return self.SEnabledStatusStandaloneResourceBase.AllowPerformDisable(ctx, userCred, query, data)
}

func (self *SHost) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.Enabled {
		_, err := self.SEnabledStatusStandaloneResourceBase.PerformDisable(ctx, userCred, query, data)
		if err != nil {
			return nil, err
		}
		self.SyncAttachedStorageStatus()
	}
	return nil, nil
}

func (self *SHost) AllowPerformCacheImage(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "cache-image")
}

func (self *SHost) PerformCacheImage(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.HostStatus != HOST_ONLINE {
		return nil, httperrors.NewInvalidStatusError("Cannot perform cache image in status %s", self.Status)
	}
	imageId, _ := data.GetString("image")
	img, err := CachedimageManager.getImageInfo(ctx, userCred, imageId, false)
	if err != nil {
		log.Errorf(err.Error())
		return nil, httperrors.NewNotFoundError("image %s not found", imageId)
	}
	if len(img.Checksum) != 0 && regutils.MatchUUID(img.Checksum) {
		return nil, httperrors.NewInvalidStatusError("Cannot cache image with no checksum")
	}
	isForce := jsonutils.QueryBoolean(data, "is_force", false)
	format, _ := data.GetString("format")
	return nil, self.StartImageCacheTask(ctx, userCred, img.Id, format, isForce)
}

func (self *SHost) StartImageCacheTask(ctx context.Context, userCred mcclient.TokenCredential, imageId string, format string, isForce bool) error {
	sc := self.GetLocalStoragecache()
	if sc == nil {
		return fmt.Errorf("No local storage cache found")
	}
	return sc.StartImageCacheTask(ctx, userCred, imageId, format, isForce, "")
}

func (self *SHost) AllowPerformConvertHypervisor(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "convert-hypervisor")
}

func (self *SHost) isAlterNameUnique(name string) bool {
	if self.GetModelManager().Query().Equals("name", name).NotEquals("id", self.Id).Equals("zone_id", self.ZoneId).Count() == 0 {
		return true
	}
	return false
}

func (self *SHost) PerformConvertHypervisor(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	hostType, err := data.GetString("host_type")
	if err != nil {
		return nil, httperrors.NewNotAcceptableError("host_type must be specified")
	}
	if self.HostType != HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewNotAcceptableError("Must be a baremetal host")
	}
	if self.GetBaremetalServer() != nil {
		return nil, httperrors.NewNotAcceptableError("Baremetal host is aleady occupied")
	}
	if !utils.IsInStringArray(self.Status, []string{BAREMETAL_READY, BAREMETAL_RUNNING}) {
		return nil, httperrors.NewNotAcceptableError("Connot convert hypervisor in status %s", self.Status)
	}
	driver := GetHostDriver(hostType)
	if driver == nil {
		return nil, httperrors.NewNotAcceptableError("Unsupport driver type %s", hostType)
	}
	// err := driver.CheckConvertConfig() do nothing
	// if err != nil {
	// return nil, httperrors.NewNotAcceptableError("Need more configuration: %s", err.Error())
	// }
	if data.Contains("name") {
		name, _ := data.GetString("name")
		err := self.GetModelManager().ValidateName(name)
		if err != nil {
			return nil, err
		}
		if !self.isAlterNameUnique(name) {
			return nil, httperrors.NewInputParameterError("Invalid name %s", name)
		}
	}
	image, _ := data.GetString("image")
	raid, _ := data.GetString("raid")
	params, err := driver.PrepareConvert(self, image, raid, data)
	if err != nil {
		return nil, httperrors.NewNotAcceptableError("Convert error: %s", err.Error())
	}
	guest, err := GuestManager.DoCreate(ctx, userCred, data, params, GuestManager)
	if err != nil {
		return nil, err
	}
	log.Infof("Host convert to %s", guest.GetName())
	db.OpsLog.LogEvent(self, db.ACT_CONVERT_START, "", userCred)
	db.OpsLog.LogEvent(guest, db.ACT_CREATE, "Convert hypervisor", userCred)
	params.Set("__task__", jsonutils.NewString(taskman.CONVERT_TASK))
	params.Set("__convert_host_type__", jsonutils.NewString(hostType))
	GuestManager.OnCreateComplete(ctx, []db.IModel{guest}, userCred, nil, params)
	self.SetStatus(userCred, BAREMETAL_START_CONVERT, "")
	return nil, nil
}

func (self *SHost) AllowPerformUndoConvert(ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, self, "undo-convert")
}

func (self *SHost) PerformUndoConvert(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !self.IsBaremetal {
		return nil, httperrors.NewNotAcceptableError("Not a baremetal")
	}
	if self.HostType == HOST_TYPE_BAREMETAL {
		return nil, httperrors.NewNotAcceptableError("Not being convert to hypervisor")
	}
	if self.Enabled {
		return nil, httperrors.NewNotAcceptableError("Host should be disabled")
	}
	if !utils.IsInStringArray(self.Status, []string{BAREMETAL_READY, BAREMETAL_RUNNING}) {
		return nil, httperrors.NewNotAcceptableError("Cannot unconvert in status %s", self.Status)
	}
	driver := self.GetDriverWithDefault()
	if driver == nil {
		return nil, httperrors.NewNotAcceptableError("Unsupport driver type %s", self.HostType)
	}
	err := driver.PrepareUnconvert(self)
	if err != nil {
		return nil, httperrors.NewNotAcceptableError(err.Error())
	}
	guests := self.GetGuests()
	if len(guests) > 1 {
		return nil, httperrors.NewNotAcceptableError("Not an empty host")
	} else if len(guests) == 1 {
		guest := guests[0]
		if guest.Hypervisor != HYPERVISOR_BAREMETAL {
			return nil, httperrors.NewNotAcceptableError("Not an converted hypervisor")
		}
		_, err := guest.GetModelManager().TableSpec().Update(&guest, func() error {
			guest.DisableDelete = tristate.False
			return nil
		})
		if err != nil {
			return nil, err
		}
		db.OpsLog.LogEvent(&guest, db.ACT_DELETE, "Unconvert baremetal", userCred)
	}
	db.OpsLog.LogEvent(self, db.ACT_UNCONVERT_START, "", userCred)
	task, err := taskman.TaskManager.NewTask(ctx, "BaremetalUnconvertHypervisorTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return nil, err
	}
	task.ScheduleRun(nil)
	return nil, nil
}

func (self *SHost) GetDriverWithDefault() IHostDriver {
	hostType := self.HostType
	if len(hostType) == 0 {
		hostType = HOST_TYPE_DEFAULT
	}
	return GetHostDriver(hostType)
}

func (self *SHost) UpdateDiskConfig(layouts []baremetal.Layout) error {
	bs := self.GetBaremetalstorage()
	if bs != nil {
		_, err := bs.GetModelManager().TableSpec().Update(bs, func() error {
			if len(layouts) != 0 {
				bs.Config = jsonutils.Marshal(layouts).(*jsonutils.JSONArray)
				var size int64
				for i := 0; i < len(layouts); i++ {
					size += layouts[i].Size
				}
				bs.RealCapacity = int(size)
			} else {
				bs.Config = jsonutils.NewArray()
				bs.RealCapacity = bs.GetStorage().Capacity
			}
			return nil
		})
		if err != nil {
			log.Errorln(err)
			return err
		}
	}
	return nil
}

func (host *SHost) SyncHostExternalNics(ctx context.Context, userCred mcclient.TokenCredential, ihost cloudprovider.ICloudHost) compare.SyncResult {
	result := compare.SyncResult{}

	netIfs := host.GetNetInterfaces()
	extNics, err := ihost.GetIHostNics()
	if err != nil {
		result.Error(err)
		return result
	}

	disables := make([]*SNetInterface, 0)
	enables := make([]cloudprovider.ICloudHostNetInterface, 0)

	type sRemoveNetInterface struct {
		netif     *SNetInterface
		reserveIp bool
	}

	type sAddNetInterface struct {
		netif     cloudprovider.ICloudHostNetInterface
		reserveIp bool
	}

	removes := make([]sRemoveNetInterface, 0)
	adds := make([]sAddNetInterface, 0)

	nicMax := len(netIfs)
	if nicMax < len(extNics) {
		nicMax = len(extNics)
	}
	for i := 0; i < nicMax; i += 1 {
		if i < len(netIfs) && i < len(extNics) {
			obn := netIfs[i].GetBaremetalNetwork()
			var oip string
			if obn != nil {
				oip = obn.IpAddr
			}
			nip := extNics[i].GetIpAddr()
			if netIfs[i].Mac == extNics[i].GetMac() {
				if oip != nip {
					if obn != nil {
						disables = append(disables, &netIfs[i])
					}
					if len(nip) > 0 {
						enables = append(enables, extNics[i])
					}
				} else {
					// do nothing, in sync
				}
			} else {
				reserveIp := false
				if len(oip) > 0 && oip == nip {
					// # mac change case
					reserveIp = true
				}
				removes = append(removes, sRemoveNetInterface{netif: &netIfs[i], reserveIp: reserveIp})
				adds = append(adds, sAddNetInterface{netif: extNics[i], reserveIp: reserveIp})
			}
		} else if i < len(netIfs) && i >= len(extNics) {
			removes = append(removes, sRemoveNetInterface{netif: &netIfs[i], reserveIp: false})
		} else if i >= len(netIfs) && i < len(extNics) {
			adds = append(adds, sAddNetInterface{netif: extNics[i], reserveIp: false})
		}
	}

	for i := len(removes) - 1; i >= 0; i -= 1 {
		err = host.RemoveNetif(ctx, userCred, removes[i].netif, removes[i].reserveIp)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := len(disables) - 1; i >= 0; i -= 1 {
		err = host.DisableNetif(ctx, userCred, disables[i], false)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}

	for i := 0; i < len(enables); i += 1 {
		netif := host.GetNetInterface(enables[i].GetMac())
		err = host.EnableNetif(ctx, userCred, netif, "", enables[i].GetIpAddr(), "", false, true)
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}

	for i := 0; i < len(adds); i += 1 {
		extNic := adds[i].netif
		err = host.addNetif(ctx, userCred, extNic.GetMac(), "", extNic.GetIpAddr(), 0, extNic.GetNicType(), extNic.GetIndex(),
			extNic.IsLinkUp(), extNic.GetMtu(), false, "", "", false, true)
		if err != nil {
			result.AddError(err)
		} else {
			result.Add()
		}
	}

	return result
}

// func (manager *SHostManager) GetEsxiAgentHostId(key string) (string, error) {
// 	q := HostManager.Query("id")
// 	q = q.Equals("host_status", HOST_ONLINE)
// 	q = q.Equals("host_type", HOST_TYPE_HYPERVISOR)
// 	q = q.IsTrue("enabled")
//
// 	rows, err := q.Rows()
// 	if err != nil {
// 		return "", err
// 	}
// 	defer rows.Close()
//
// 	var hostId string
// 	hostIds := make([]string, 0)
// 	for rows.Next() {
// 		err = rows.Scan(&hostId)
// 		if err != nil {
// 			return "", err
// 		}
// 		hostIds = append(hostIds, hostId)
// 	}
//
// 	ring := hashring.New(hostIds)
// 	ret, _ := ring.GetNode(key)
// 	return ret, nil
// }
//
// func (manager *SHostManager) GetEsxiAgentHost(key string) (*SHost, error) {
// 	hostId, err := manager.GetEsxiAgentHostId(key)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return manager.FetchHostById(hostId), nil
// }
//
// func (host *SHost) GetEsxiAgentHost() (*SHost, error) {
// 	return HostManager.GetEsxiAgentHost(host.Id)
// }

func (self *SHost) IsEsxiAgentReady() bool {
	url, err := auth.GetServiceURL("esxiagent", options.Options.Region, self.GetZone().GetName(), "")
	if err != nil {
		log.Errorln("is esxi agent ready: false")
		return false
	}
	log.Infof("esxi agent url:%s", url)
	return true
}

func (self *SHost) EsxiRequest(ctx context.Context, method httputils.THttpMethod, url string, headers http.Header, body *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	serviceUrl, err := auth.GetServiceURL("esxiagent", options.Options.Region, self.GetZone().GetName(), "")
	if err != nil {
		return nil, err
	}
	url = serviceUrl + url
	_, data, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, method, url, headers, body, false)
	return data, err
}

func (manager *SHostManager) GetHostByIp(hostIp string) (*SHost, error) {
	q := manager.Query()
	q = q.Equals("access_ip", hostIp)

	host, err := db.NewModelObject(manager)
	if err != nil {
		return nil, err
	}
	err = q.First(host)
	if err != nil {
		return nil, err
	}

	return host.(*SHost), nil
}

func (self *SHost) getCloudProviderInfo() SCloudProviderInfo {
	var region *SCloudregion
	zone := self.GetZone()
	if zone != nil {
		region = zone.GetRegion()
	}
	provider := self.GetCloudprovider()
	return MakeCloudProviderInfo(region, zone, provider)
}

func (self *SHost) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SEnabledStatusStandaloneResourceBase.GetShortDesc(ctx)
	info := self.getCloudProviderInfo()
	desc.Update(jsonutils.Marshal(&info))
	return desc
}

func (self *SHost) MarkGuestUnknown(userCred mcclient.TokenCredential) {
	log.Errorln(self.GetGuests())
	for _, guest := range self.GetGuests() {
		guest.SetStatus(userCred, VM_UNKNOWN, "host offline")
	}
}

func (manager *SHostManager) PingDetectionTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	deadline := time.Now().Add(-1 * time.Duration(options.Options.HostOfflineMaxSeconds) * time.Second)

	q := manager.Query().Equals("host_status", HOST_ONLINE).
		Equals("host_type", HOST_TYPE_HYPERVISOR)
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("last_ping_at")),
		sqlchemy.LT(q.Field("last_ping_at"), deadline)))

	rows, err := q.Rows()
	if err != nil {
		log.Errorln(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var host = SHost{}
		q.Row2Struct(rows, &host)
		host.SetModelManager(manager)
		host.PerformOffline(ctx, userCred, nil, nil)
		host.MarkGuestUnknown(userCred)
	}
}
