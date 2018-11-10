package models

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/sysutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const (
	HOST_TYPE_BAREMETAL  = "baremetal"
	HOST_TYPE_HYPERVISOR = "hypervisor" // KVM
	HOST_TYPE_ESXI       = "esxi"       // # VMWare vSphere ESXi
	HOST_TYPE_KUBELET    = "kubelet"    // # Kubernetes Kubelet
	HOST_TYPE_HYPERV     = "hyperv"     // # Microsoft Hyper-V
	HOST_TYPE_XEN        = "xen"        // # XenServer
	HOST_TYPE_ALIYUN     = "aliyun"
	HOST_TYPE_AZURE      = "azure"

	HOST_TYPE_DEFAULT = HOST_TYPE_HYPERVISOR

	// # possible status
	HOST_ONLINE   = "online"
	HOST_ENABLED  = "online"
	HOST_OFFLINE  = "offline"
	HOST_DISABLED = "offline"

	NIC_TYPE_IPMI  = "ipmi"
	NIC_TYPE_ADMIN = "admin"
	// #NIC_TYPE_NORMAL = 'normal'

	HOST_STATUS_INIT           = "init"
	HOST_STATUS_PREPARE        = "prepare"
	HOST_STATUS_PREPARE_FAIL   = "prepare_fail"
	HOST_STATUS_READY          = "ready"
	HOST_STATUS_RUNNING        = "running"
	HOST_STATUS_MAINTAINING    = "maintaining"
	HOST_STATUS_START_MAINTAIN = "start_maintain"
	HOST_STATUS_DELETING       = "deleting"
	HOST_STATUS_DELETE         = "delete"
	HOST_STATUS_DELETE_FAIL    = "delete_fail"
	HOST_STATUS_UNKNOWN        = "unknown"
	HOST_STATUS_SYNCING_STATUS = "syncing_status"
	HOST_STATUS_SYNC           = "sync"
	HOST_STATUS_SYNC_FAIL      = "sync_fail"
	HOST_STATUS_START_CONVERT  = "start_convert"
	HOST_STATUS_CONVERTING     = "converting"
)

var HOST_TYPES = []string{HOST_TYPE_BAREMETAL, HOST_TYPE_HYPERVISOR, HOST_TYPE_ESXI, HOST_TYPE_KUBELET, HOST_TYPE_XEN, HOST_TYPE_ALIYUN, HOST_TYPE_AZURE}
var NIC_TYPES = []string{NIC_TYPE_IPMI, NIC_TYPE_ADMIN}

type SHostManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	SInfrastructureManager
}

var HostManager *SHostManager

func init() {
	HostManager = &SHostManager{SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(SHost{}, "hosts_tbl", "host", "hosts")}
	HostManager.SetAlias("baremetal", "baremetals")
}

type SHost struct {
	db.SEnabledStatusStandaloneResourceBase
	SInfrastructure
	SManagedResourceBase

	Rack  string `width:"16" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)
	Slots string `width:"16" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(16, charset='ascii'), nullable=True)

	AccessMac  string `width:"32" charset:"ascii" nullable:"false" index:"true" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(32, charset='ascii'), nullable=False, index=True)
	AccessIp   string `width:"16" charset:"ascii" nillable:"true" list:"admin" update:"admin" create:"admin_optional"`               // Column(VARCHAR(16, charset='ascii'), nullable=True)
	ManagerUri string `width:"256" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"`              // Column(VARCHAR(256, charset='ascii'), nullable=True)

	SysInfo jsonutils.JSONObject `nullable:"true" search:"admin" get:"admin" update:"admin" create:"admin_optional"`               // Column(JSONEncodedDict, nullable=True)
	SN      string               `width:"128" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(128, charset='ascii'), nullable=True)

	CpuCount    int8    `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                           // Column(TINYINT, nullable=True) # cpu count
	NodeCount   int8    `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                           // Column(TINYINT, nullable=True)
	CpuDesc     string  `width:"64" charset:"ascii" nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(64, charset='ascii'), nullable=True)
	CpuMhz      int     `nullable:"true" get:"admin" update:"admin" create:"admin_optional"`                            // Column(Integer, nullable=True) # cpu MHz
	CpuCache    int     `nullable:"true" get:"admin" update:"admin" create:"admin_optional"`                            // Column(Integer, nullable=True) # cpu Cache in KB
	CpuReserved int8    `nullable:"true" default:"0" list:"admin" update:"admin" create:"admin_optional"`               // Column(TINYINT, nullable=True, default=0)
	CpuCmtbound float32 `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                           // = Column(Float, nullable=True)

	MemSize     int     `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`             // Column(Integer, nullable=True) # memory size in MB
	MemReserved int     `nullable:"true" default:"0" list:"admin" update:"admin" create:"admin_optional"` // Column(Integer, nullable=True, default=0) # memory reserved in MB
	MemCmtbound float32 `nullable:"true" update:"admin" create:"admin_optional"`                          // = Column(Float, nullable=True)

	StorageSize   int                  `nullable:"true" list:"admin" update:"admin" create:"admin_optional"`                            // Column(Integer, nullable=True) # storage size in MB
	StorageType   string               `width:"20" charset:"ascii" nullable:"true" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(20, charset='ascii'), nullable=True)
	StorageDriver string               `width:"20" charset:"ascii" nullable:"true" update:"admin" create:"admin_optional"`              // Column(VARCHAR(20, charset='ascii'), nullable=True)
	StorageInfo   jsonutils.JSONObject `nullable:"true" get:"admin" update:"admin" create:"admin_optional"`                             // Column(JSONEncodedDict, nullable=True)

	IpmiInfo jsonutils.JSONObject `nullable:"true" get:"admin" update:"admin" create:"admin_optional"` // Column(JSONEncodedDict, nullable=True)

	// Status  string = Column(VARCHAR(16, charset='ascii'), nullable=False, default=baremetalstatus.INIT) # status
	HostStatus string `width:"16" charset:"ascii" nullable:"false" default:"offline" list:"admin"` // Column(VARCHAR(16, charset='ascii'), nullable=False, server_default=HOST_OFFLINE, default=HOST_OFFLINE)

	ZoneId string `width:"128" charset:"ascii" nullable:"false" list:"admin" create:"admin_required"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=False)

	HostType string `width:"36" charset:"ascii" nullable:"false" list:"admin" update:"admin" create:"admin_required"` // Column(VARCHAR(36, charset='ascii'), nullable=False)

	Version string `width:"64" charset:"ascii" list:"admin" update:"admin" create:"admin_optional"` // Column(VARCHAR(64, charset='ascii'))

	IsBaremetal bool `nullable:"true" default:"false" list:"admin" update:"true" create:"admin_optional"` // Column(Boolean, nullable=True, default=False)

	IsMaintenance bool `nullable:"true" default:"false" list:"admin"` // Column(Boolean, nullable=True, default=False)
}

func (manager *SHostManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
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
	var scopeQuery *sqlchemy.SSubQuery

	schedTagStr := jsonutils.GetAnyString(query, []string{"schedtag", "schedtag_id"})
	if len(schedTagStr) > 0 {
		schedTag, _ := SchedtagManager.FetchByIdOrName("", schedTagStr)
		if schedTag == nil {
			return nil, httperrors.NewResourceNotFoundError("Schedtag %s not found", schedTagStr)
		}
		hostschedtags := HostschedtagManager.Query().SubQuery()
		scopeQuery = hostschedtags.Query(hostschedtags.Field("host_id")).Equals("schedtag_id", schedTag.GetId()).SubQuery()
	}

	wireStr := jsonutils.GetAnyString(query, []string{"wire", "wire_id"})
	if len(wireStr) > 0 {
		wire, _ := WireManager.FetchByIdOrName("", wireStr)
		if wire == nil {
			return nil, httperrors.NewResourceNotFoundError("Wire %s not found", wireStr)
		}
		hostwires := HostwireManager.Query().SubQuery()
		scopeQuery = hostwires.Query(hostwires.Field("host_id")).Equals("wire_id", wire.GetId()).SubQuery()
	}

	storageStr := jsonutils.GetAnyString(query, []string{"storage", "storage_id"})
	if len(storageStr) > 0 {
		storage, _ := StorageManager.FetchByIdOrName("", storageStr)
		if storage == nil {
			return nil, httperrors.NewResourceNotFoundError("Storage %s not found", storageStr)
		}
		hoststorages := HoststorageManager.Query().SubQuery()
		scopeQuery = hoststorages.Query(hoststorages.Field("host_id")).Equals("storage_id", storage.GetId()).SubQuery()
	}

	zoneStr := jsonutils.GetAnyString(query, []string{"zone", "zone_id"})
	if len(zoneStr) > 0 {
		zone, _ := ZoneManager.FetchByIdOrName("", zoneStr)
		if zone == nil {
			return nil, httperrors.NewResourceNotFoundError("Zone %s not found", zoneStr)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("zone_id"), zone.GetId()))
	}
	// vcenter
	// zone
	// cachedimage

	managerStr := jsonutils.GetAnyString(query, []string{"manager", "provider", "manager_id", "provider_id"})
	if len(managerStr) > 0 {
		provider := CloudproviderManager.FetchCloudproviderByIdOrName(managerStr)
		if provider == nil {
			return nil, httperrors.NewResourceNotFoundError("provider %s not found", managerStr)
		}
		q = q.Filter(sqlchemy.Equals(q.Field("manager_id"), provider.GetId()))
	}

	if scopeQuery != nil {
		q = q.In("id", scopeQuery)
	}
	return q, nil
}

func (self *SHost) GetZone() *SZone {
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
		return self.StartDeleteBaremetalTask(userCred)
	} else {
		return self.RealDelete(ctx, userCred)
	}
}

func (self *SHost) StartDeleteBaremetalTask(userCred mcclient.TokenCredential) error {
	log.Debugf("StartDeleteBaremetalTask")
	return nil
}

func (self *SHost) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	for _, hostschedtag := range self.GetHostschedtags() {
		hostschedtag.Delete(ctx, userCred)
	}
	// XXX: TODO
	// IsolatedDevices.delete_devices_by_host(self)
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
	// XXX: TODO
	// for hc in self.get_hostcachedimages():
	//	hc.delete()
	for _, bn := range self.GetBaremetalnetworks() {
		self.DeleteBaremetalnetwork(ctx, userCred, &bn, false)
	}
	for _, netif := range self.GetNetInterfaces() {
		netif.Remove(ctx, userCred)
	}
	for _, hostwire := range self.GetHostwires() {
		hostwire.Detach(ctx, userCred)
		// hostwire.Delete(ctx, userCred)
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

func (self *SHost) GetBaremetalstorage() *SHoststorage {
	if !self.IsBaremetal {
		return nil
	}
	hoststorages := HoststorageManager.Query().SubQuery()
	storages := StorageManager.Query().SubQuery()
	q := hoststorages.Query()
	q = q.Join(storages, sqlchemy.AND(sqlchemy.Equals(storages.Field("id"), hoststorages.Field("storage_id")),
		sqlchemy.IsFalse(storages.Field("deleted"))))
	q = q.Filter(sqlchemy.Equals(storages.Field("storage_type"), sysutils.STORAGE_BAREMETAL))
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
	return fmt.Sprintf("%s://%s:%d", managerUrl.Scheme, managerUrl.Host, port+40000)
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

func (self *SHostManager) AllowGetPropertyBmStartRegisterScript(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SHostManager) GetPropertyBmStartRegisterScript(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	regionUri, err := auth.GetServiceURL("compute", options.Options.Region, "", "")
	if err != nil {
		return nil, err
	}
	var script string
	script += fmt.Sprintf("curl -fsSL -H 'X-Auth-Token: %s' %s/misc/bm-prepare-script", userCred.GetTokenString(), regionUri)
	res := jsonutils.NewDict()
	res.Add(jsonutils.NewString(script), "script")
	return res, nil
}

func (maanger *SHostManager) ClearAllSchedDescCache() error {
	s := auth.GetAdminSession(options.Options.Region, "")
	return modules.SchedManager.CleanCache(s, "")
}

func (maanger *SHostManager) ClearSchedDescCache(hostId string) error {
	s := auth.GetAdminSession(options.Options.Region, "")
	return modules.SchedManager.CleanCache(s, hostId)
}

func (self *SHost) ClearSchedDescCache() error {
	return HostManager.ClearSchedDescCache(self.Id)
}

func (self *SHost) GetSpec(statusCheck bool) *jsonutils.JSONDict {
	if statusCheck {
		if utils.IsInStringArray(self.Status, []string{BAREMETAL_INIT, BAREMETAL_PREPARE_FAIL, BAREMETAL_PREPARE}) ||
			self.getBaremetalServer() != nil {
			return nil
		}
		if self.MemSize == 0 || self.CpuCount == 0 {
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
	manufacture, err := self.SysInfo.Get("manufacture")
	if err != nil {
		manufacture = jsonutils.NewString("Unknown")
	}
	spec.Set("manufacture", manufacture)
	model, err := self.SysInfo.Get("model")
	if err != nil {
		model = jsonutils.NewString("Unknown")
	}
	spec.Set("model", model)
	return spec
}

func (manager *SHostManager) GetSpecIdent(spec *jsonutils.JSONDict) []string {
	nCpu, _ := spec.Int("cpu")
	memSize, _ := spec.Int("mem")
	memGB, err := utils.GetSizeGB(fmt.Sprintf("%d", memSize), "M")
	if err != nil {
		log.Errorf("Get mem size %d GB error: %v", memSize, err)
	}
	nicCount, _ := spec.Int("nic_count")
	manufacture, _ := spec.GetString("manufacture")
	model, _ := spec.GetString("model")

	specKeys := []string{
		fmt.Sprintf("cpu:%d", nCpu),
		fmt.Sprintf("mem:%dG", memGB),
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

func GetDiskSpecV2(storageInfo jsonutils.JSONObject) jsonutils.JSONObject {
	storages := []baremetal.BaremetalStorage{}
	err := storageInfo.Unmarshal(&storages)
	if err != nil {
		log.Errorf("Unmarshal to baremetal storage error: %v", err)
		return nil
	}
	refStorages := make([]*baremetal.BaremetalStorage, len(storages))
	for i := range storages {
		refStorages[i] = &storages[i]
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

func (self *SHost) GetAttachedStorageCapacity() SStorageCapacity {
	ret := SStorageCapacity{}
	storages := self.GetAttachedStorages("")
	if storages != nil {
		for _, s := range storages {
			ret.Capacity += s.GetCapacity()
			ret.Used += s.GetUsedCapacity(tristate.True)
			ret.Wasted += s.GetUsedCapacity(tristate.False)
			ret.VCapacity += int(float32(s.GetCapacity()) * s.GetOvercommitBound())
		}
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
	db.OpsLog.LogDetachEvent(self, net, userCred, nil)
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

func (manager *SHostManager) getHostsByZone(zone *SZone, provider *SCloudprovider) ([]SHost, error) {
	hosts := make([]SHost, 0)
	q := manager.Query().Equals("zone_id", zone.Id)
	if provider != nil {
		q = q.Equals("manager_id", provider.Id)
	}
	err := db.FetchModelObjects(manager, q, &hosts)
	if err != nil {
		log.Errorf("%s", err)
		return nil, err
	}
	return hosts, nil
}

func (manager *SHostManager) SyncHosts(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, zone *SZone, hosts []cloudprovider.ICloudHost) ([]SHost, []cloudprovider.ICloudHost, compare.SyncResult) {
	localHosts := make([]SHost, 0)
	remoteHosts := make([]cloudprovider.ICloudHost, 0)
	syncResult := compare.SyncResult{}

	dbHosts, err := manager.getHostsByZone(zone, provider)
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
		err = commondb[i].syncWithCloudHost(commonext[i])
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

func (self *SHost) syncWithCloudHost(extHost cloudprovider.ICloudHost) error {
	_, err := self.GetModelManager().TableSpec().Update(self, func() error {
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

		return nil
	})
	if err != nil {
		log.Errorf("syncWithCloudZone error %s", err)
	}
	return err
}

func (manager *SHostManager) newFromCloudHost(extHost cloudprovider.ICloudHost, zone *SZone) (*SHost, error) {
	host := SHost{}
	host.SetModelManager(manager)

	host.Name = extHost.GetName()
	host.ExternalId = extHost.GetGlobalId()
	host.ZoneId = zone.Id
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

	host.ManagerId = extHost.GetManagerId()
	host.IsEmulated = extHost.IsEmulated()

	err := manager.TableSpec().Insert(&host)
	if err != nil {
		log.Errorf("newFromCloudHost fail %s", err)
		return nil, err
	}
	return &host, nil
}

func (self *SHost) SyncHostStorages(ctx context.Context, userCred mcclient.TokenCredential, storages []cloudprovider.ICloudStorage) compare.SyncResult {
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
		return syncResult
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
		err := self.syncWithCloudHostStorage(commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		log.Infof("host %s is found connected with %s, to add ...", self.Id, added[i].GetId())
		err := self.newCloudHostStorage(ctx, userCred, added[i])
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}

func (self *SHost) syncWithCloudHostStorage(extStorage cloudprovider.ICloudStorage) error {
	// do nothing
	return nil
}

func (self *SHost) Attach2Storage(ctx context.Context, userCred mcclient.TokenCredential, storage *SStorage, mountPoint string) error {
	hs := SHoststorage{}
	hs.SetModelManager(HoststorageManager)

	hs.StorageId = storage.Id
	hs.HostId = self.Id
	hs.MountPoint = mountPoint
	err := HoststorageManager.TableSpec().Insert(&hs)
	if err != nil {
		return err
	}
	db.OpsLog.LogAttachEvent(self, storage, userCred, nil)

	return nil
}

func (self *SHost) newCloudHostStorage(ctx context.Context, userCred mcclient.TokenCredential, extStorage cloudprovider.ICloudStorage) error {
	storageObj, err := StorageManager.FetchByExternalId(extStorage.GetGlobalId())
	if err != nil {
		log.Errorf("%s", err)
		return nil
	}
	storage := storageObj.(*SStorage)
	log.Debugf("Storage: %s", storage)
	log.Debugf("Host: %s", self)
	err = self.Attach2Storage(ctx, userCred, storage, "")
	return err
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
	db.OpsLog.LogAttachEvent(self, wire, userCred, nil)
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

func (self *SHost) SyncHostVMs(ctx context.Context, userCred mcclient.TokenCredential, vms []cloudprovider.ICloudVM) ([]SGuest, []cloudprovider.ICloudVM, compare.SyncResult) {
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
		err := commondb[i].syncWithCloudVM(ctx, userCred, self, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			localVMs = append(localVMs, commondb[i])
			remoteVMs = append(remoteVMs, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		new, err := GuestManager.newCloudVM(ctx, userCred, self, added[i])
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
	if userCred != nil && !userCred.IsSystemAdmin() {
		zones := ZoneManager.Query().SubQuery()
		q = q.Join(zones, sqlchemy.AND(
			sqlchemy.Equals(zones.Field("id"), hosts.Field("zone_id")),
			sqlchemy.IsFalse(zones.Field("deleted")))).
			Filter(sqlchemy.Equals(zones.Field("admin_id"), userCred.GetProjectId()))
	} else {
		q = AttachUsageQuery(q, hosts, hosts.Field("id"), hostTypes, rangeObj)
	}
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

func (manager *SHostManager) totalCount(
	userCred mcclient.TokenCredential,
	rangeObj db.IStandaloneModel,
	hostStatus, status string,
	hostTypes []string,
	enabled, isBaremetal tristate.TriState,
) HostsCountStat {
	return manager.calculateCount(manager.totalCountQ(userCred, rangeObj, hostStatus, status, hostTypes, enabled, isBaremetal))
}

func (manager *SHostManager) TotalCount(
	userCred mcclient.TokenCredential,
	rangeObj db.IStandaloneModel,
	hostStatus, status string,
	hostTypes []string,
	enabled, isBaremetal tristate.TriState,
) HostsCountStat {
	stat1 := manager.totalCount(userCred, rangeObj, hostStatus, status, hostTypes, enabled, isBaremetal)
	return stat1
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
	ihost, err := provider.GetIHostById(self.ExternalId)
	if err != nil {
		log.Errorf("fail to find ihost by id %s %s", self.ExternalId, err)
		return nil, fmt.Errorf("fail to find ihost by id %s", err)
	}
	return ihost, nil
}

func (self *SHost) getDiskConfig() jsonutils.JSONObject {
	bs := self.GetBaremetalstorage()
	if bs != nil {
		return bs.Config
	}
	return nil
}

func (self *SHost) getBaremetalServer() *SGuest {
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
	err := q.All(&tags)
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
	q = q.Equals("host_id", self.Id)
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

func (self *SHost) getMoreDetails(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	zone := self.GetZone()
	if zone != nil {
		extra.Add(jsonutils.NewString(zone.Id), "zone_id")
		extra.Add(jsonutils.NewString(zone.Name), "zone")
		extra.Add(jsonutils.NewString(zone.GetRegion().GetName()), "region")
		extra.Add(jsonutils.NewString(zone.GetRegion().GetId()), "region_id")
	}
	server := self.getBaremetalServer()
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
			info[i] = schedtags[i].GetShortDesc()
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
	return extra
}

func (self *SHost) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SHost) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SEnabledStatusStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	return self.getMoreDetails(extra)
}

func (self *SHost) AllowGetDetailsVnc(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsSystemAdmin()
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

func (manager *SHostManager) GetHostsByManagerAndRegion(managerId string, regionId string) []SHost {
	hosts := HostManager.Query().SubQuery()
	zones := ZoneManager.Query().SubQuery()
	q := hosts.Query()
	q = q.Join(zones, sqlchemy.Equals(hosts.Field("zone_id"), zones.Field("id")))
	q = q.Filter(sqlchemy.Equals(hosts.Field("manager_id"), managerId))
	q = q.Filter(sqlchemy.Equals(zones.Field("cloudregion_id"), regionId))
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

func (self *SHost) Request(userCred mcclient.TokenCredential, method string, url string, headers http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	s := auth.GetSession(userCred, "", "")
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
