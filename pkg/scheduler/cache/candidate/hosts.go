package candidate

import (
	"encoding/json"
	"fmt"
	"strings"
	gosync "sync"
	"sync/atomic"
	"time"

	"yunion.io/x/log"
	o "yunion.io/x/onecloud/cmd/scheduler/options"
	"yunion.io/x/onecloud/pkg/scheduler/cache"
	"yunion.io/x/onecloud/pkg/scheduler/cache/db"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
	"yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/util/workqueue"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type HostDesc struct {
	*baseHostDesc

	// cpu
	CPUMHZ              int64    `json:"cpu_mhz"`
	CPUCmtbound         float64  `json:"cpu_cmtbound"`
	CPUDesc             string   `json:"cpu_desc"`
	CPUCache            int64    `json:"cpu_cache"`
	CPUReserved         int64    `json:"cpu_reserved"`
	CPUBoundCount       int64    `json:"cpu_bound_count"`
	CPULoad             *float64 `json:"cpu_load"`
	TotalCPUCount       int64    `json:"total_cpu_count"`
	RunningCPUCount     int64    `json:"running_cpu_count"`
	CreatingCPUCount    int64    `json:"creating_cpu_count"`
	RequiredCPUCount    int64    `json:"required_cpu_count"`
	FakeDeletedCPUCount int64    `json:"fake_deleted_cpu_count"`
	FreeCPUCount        int64    `json:"free_cpu_count"`

	// memory
	MemCmtbound        float64 `json:"mem_cmtbound"`
	MemReserved        int64   `json:"mem_reserved"`
	TotalMemSize       int64   `json:"total_mem_size"`
	FreeMemSize        int64   `json:"free_mem_size"`
	RunningMemSize     int64   `json:"running_mem_size"`
	CreatingMemSize    int64   `json:"creating_mem_size"`
	RequiredMemSize    int64   `json:"required_mem_size"`
	FakeDeletedMemSize int64   `json:"fake_deleted_mem_size"`

	// storage
	Storages     []*Storage `json:"storages"`
	StorageTypes []string   `json:"storage_types"`

	// IO
	IOBoundCount int64    `json:"io_bound_count"`
	IOLoad       *float64 `json:"io_load"`

	// server
	GuestCount         int64 `json:"guest_count"`
	CreatingGuestCount int64 `json:"creating_guest_count"`
	RunningGuestCount  int64 `json:"running_guest_count"`

	Groups                    *GroupCounts          `json:"groups"`
	Metadata                  map[string]string     `json:"metadata"`
	IsolatedDevices           []*IsolatedDeviceDesc `json:"isolated_devices"`
	IsMaintenance             bool                  `json:"is_maintenance"`
	GuestReservedResource     *ReservedResource     `json:"guest_reserved_resource"`
	GuestReservedResourceUsed *ReservedResource     `json:"guest_reserved_used"`
}

type ReservedResource struct {
	CPUCount    int64 `json:"cpu_count"`
	MemorySize  int64 `json:"memory_size"`
	StorageSize int64 `json:"storage_size"`
}

func NewReservedResource(cpu, mem, storage int64) *ReservedResource {
	return &ReservedResource{
		CPUCount:    cpu,
		MemorySize:  mem,
		StorageSize: storage,
	}
}

func CpuIsolatedDevReservedCount() int64 {
	return o.GetOptions().CpuReservedPerIsolatedDevice
}

func MemIsolatedDevReservedSize() int64 {
	return o.GetOptions().MemoryReservedPerIsolatedDevice
}

func StorageIsolatedDevReservedSize() int64 {
	return o.GetOptions().StorageReservedPerIsolatedDevice
}

func NewGuestReservedResourceByBuilder(b *HostBuilder, host *models.Host) (ret *ReservedResource) {
	ret = NewReservedResource(0, 0, 0)
	//isoDevs := b.getUnusedIsolatedDevices(host.ID)
	isoDevs := b.getIsolatedDevices(host.ID)
	hostDevsCount := int64(len(isoDevs))
	if hostDevsCount == 0 {
		return
	}

	cpuPerDevRsvd := CpuIsolatedDevReservedCount()
	memPerDevRsvd := MemIsolatedDevReservedSize()
	storagePerDevRsvd := StorageIsolatedDevReservedSize()
	ret.CPUCount = hostDevsCount * cpuPerDevRsvd
	ret.MemorySize = hostDevsCount * memPerDevRsvd
	ret.StorageSize = hostDevsCount * storagePerDevRsvd

	return
}

func NewGuestReservedResourceUsedByBuilder(b *HostBuilder, host *models.Host) (ret *ReservedResource, err error) {
	ret = NewReservedResource(0, 0, 0)
	gst := b.getIsolatedDeviceGuests(host.ID)
	if len(gst) == 0 {
		return
	}
	var (
		cpu  int64 = 0
		mem  int64 = 0
		disk int64 = 0
	)
	for _, g := range gst {
		dSize, err2 := g.DiskSize(true)
		if err2 != nil {
			err = err2
			return
		}
		disk += dSize
		if o.GetOptions().IgnoreNonRunningGuests && !g.IsRunning() {
			continue
		}
		cpu += g.VCPUCount
		mem += g.VMemSize
	}
	ret.CPUCount = cpu
	ret.MemorySize = mem
	ret.StorageSize = disk
	return
}

type Storage struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Capacity      int64    `json:"capacity"`
	StorageType   string   `json:"type"`
	UsedCapacity  int64    `json:"used"`
	WasteCapacity int64    `json:"waste"`
	FreeCapacity  int64    `json:"free"`
	VCapacity     int64    `json:"vcapacity"`
	Cmtbound      float64  `json:"cmtbound"`
	StorageDriver string   `json:"driver"`
	Adapter       string   `json:"adapter"`
	Splits        []string `json:"splits"`
	Range         string   `json:"range"`
	Conf          string   `json:"conf"`
	MinStripSize  int      `json:"min_strip_size"`
	MaxStripSize  int      `json:"max_strip_size"`
	Size          int      `json:"size"`
}

func (storage *Storage) GetFreeSize() int64 {
	return storage.GetTotalSize() - storage.UsedCapacity - storage.WasteCapacity
}

func (storage *Storage) GetTotalSize() int64 {
	return int64(float64(storage.Capacity) * storage.Cmtbound)
}

func (storage *Storage) IsLocal() bool {
	return utils.IsLocalStorage(storage.StorageType)
}

type HostBuilder struct {
	clusters cache.Cache

	residentTenantDict map[string]map[string]interface{}

	hosts    []interface{}
	hostDict map[string]interface{}

	guests    []interface{}
	guestDict map[string]interface{}
	guestIDs  []string

	hostStorages          []interface{}
	hostStoragesDict      map[string][]interface{}
	storages              []interface{}
	storageDict           map[string]interface{}
	storageStatesSizeDict map[string]map[string]interface{}

	hostGuests map[string][]interface{}

	groupGuests        []interface{}
	groups             []interface{}
	groupDict          map[string]interface{}
	hostGroupCountDict HostGroupCountDict

	hostMetadatas      []interface{}
	hostMetadatasDict  map[string][]interface{}
	guestMetadatas     []interface{}
	guestMetadatasDict map[string][]interface{}

	diskStats           []models.StorageCapacity
	isolatedDevicesDict map[string][]interface{}

	cpuIOLoads map[string]map[string]float64
}

func (h *HostDesc) String() string {
	s, _ := json.Marshal(h)
	return string(s)
}

func (h *HostDesc) Type() int {
	// Guest type
	return 0
}

func (h *HostDesc) Get(key string) interface{} {
	switch key {
	case "ID":
		return h.ID

	case "Name":
		return h.Name

	case "CPUCount":
		return h.CPUCount

	case "MemSize":
		return h.MemSize

	case "PoolID":
		return h.PoolID

	case "ZoneID":
		return h.ZoneID

	case "ClusterID":
		return h.ClusterID

	case "TotalCPUCount":
		return h.GetTotalCPUCount(true)

	case "FreeCPUCount":
		return h.GetFreeCPUCount(false)

	case "TotalMemSize":
		return h.GetTotalMemSize(true)

	case "FreeMemSize":
		return h.GetFreeMemSize(false)

	case "Groups":
		return h.Groups

	case "IsolatedDevices":
		return h.IsolatedDevices

	case "Status":
		return h.Status

	case "TotalStorageSize":
		return h.totalStorageSize(false, true)

	case "TotalLocalStorageSize":
		return h.totalStorageSize(true, true)

	case "FreeStorageSize":
		return h.freeStorageSize(false, false)

	case "FreeLocalStorageSize":
		return h.freeStorageSize(true, false)

	case "StorageTypes":
		return h.StorageTypes

	case "HostStatus":
		return h.HostStatus

	case "EnableStatus":
		return h.GetEnableStatus()

	case "HostType":
		return h.HostType

	case "IsBaremetal":
		return h.IsBaremetal

	default:
		index := strings.Index(key, ":")
		if index >= 0 {
			masterKey := key[0:index]
			slaveKey := key[index+1:]

			switch masterKey {
			case "FreeStorageSize":
				storageType := slaveKey
				return h.freeStorageSizeOfType(storageType, false)
			}
		}
		return nil
	}
}

func (h *HostDesc) XGet(key string, kind core.Kind) interface{} {
	return core.XGetCalculator(h, key, kind)
}

func (h *HostDesc) GetTotalLocalStorageSize(useRsvd bool) int64 {
	return h.totalStorageSize(true, useRsvd)
}

func (h *HostDesc) GetFreeLocalStorageSize(useRsvd bool) int64 {
	return h.freeStorageSize(true, useRsvd)
}

func (h *HostDesc) totalStorageSize(onlyLocal, useRsvd bool) int64 {

	total := int64(0)
	for _, storage := range h.Storages {
		if !onlyLocal || storage.IsLocal() {
			total += storage.GetTotalSize()
		}
	}

	if onlyLocal {
		return reservedResourceMinusCal(total, h.GuestReservedResource.StorageSize, useRsvd)
	}
	return total
}

func (h *HostDesc) freeStorageSize(onlyLocal, useRsvd bool) int64 {
	total := int64(0)
	for _, storage := range h.Storages {
		if !onlyLocal || storage.IsLocal() {
			total += storage.GetFreeSize()
		}
	}

	total = total + h.GuestReservedResourceUsed.StorageSize - h.GetReservedStorageSize()
	sizeSub := h.GuestReservedResource.StorageSize - h.GuestReservedResourceUsed.StorageSize
	if sizeSub < 0 {
		total += sizeSub
	}
	if useRsvd {
		return reservedResourceAddCal(total, h.GuestReservedStorageSizeFree(), useRsvd)
	}

	return total
}

func (h *HostDesc) GetFreeStorageSizeOfType(sType string, useRsvd bool) int64 {
	return h.freeStorageSizeOfType(sType, useRsvd)
}

func (h *HostDesc) freeStorageSizeOfType(storageType string, useRsvd bool) int64 {
	total := int64(0)
	for _, storage := range h.Storages {
		if storage.StorageType == storageType {
			total += storage.GetFreeSize()
		}
	}
	if utils.IsLocalStorage(storageType) {
		total = total + h.GuestReservedResourceUsed.StorageSize - h.GetReservedStorageSize()
		sizeSub := h.GuestReservedResource.StorageSize - h.GuestReservedResourceUsed.StorageSize
		if sizeSub < 0 {
			total += sizeSub
		}
	}
	if useRsvd {
		return reservedResourceAddCal(total, h.GuestReservedStorageSizeFree(), useRsvd)
	}

	return total
}

func reservedResourceCal(
	curRes, rsvdRes int64,
	useRsvd, minusRsvd bool,
) int64 {
	actRes := curRes
	if useRsvd {
		if minusRsvd {
			actRes -= rsvdRes
		} else {
			actRes += rsvdRes
		}
	}
	return actRes
}

func reservedResourceAddCal(curRes, rsvdRes int64, useRsvd bool) int64 {
	return reservedResourceCal(curRes, rsvdRes, useRsvd, false)
}

func reservedResourceMinusCal(curRes, rsvdRes int64, useRsvd bool) int64 {
	return reservedResourceCal(curRes, rsvdRes, useRsvd, true)
}

func (h *HostDesc) GetTotalMemSize(useRsvd bool) int64 {
	return reservedResourceMinusCal(h.TotalMemSize, h.GuestReservedResource.MemorySize, useRsvd)
}

func (h *HostDesc) GetFreeMemSize(useRsvd bool) int64 {
	return reservedResourceAddCal(h.FreeMemSize, h.GuestReservedMemSizeFree(), useRsvd)
}

func (h *HostDesc) GuestReservedMemSizeFree() int64 {
	return h.GuestReservedResource.MemorySize - h.GuestReservedResourceUsed.MemorySize
}

func (h *HostDesc) GuestReservedCPUCountFree() int64 {
	return h.GuestReservedResource.CPUCount - h.GuestReservedResourceUsed.CPUCount
}

func (h *HostDesc) GuestReservedStorageSizeFree() int64 {
	return h.GuestReservedResource.StorageSize - h.GuestReservedResourceUsed.StorageSize
}

func (h *HostDesc) GetReservedMemSize() int64 {
	return h.GuestReservedResource.MemorySize + h.MemReserved
}

func (h *HostDesc) GetReservedCPUCount() int64 {
	return h.GuestReservedResource.CPUCount + h.CPUReserved
}

func (h *HostDesc) GetReservedStorageSize() int64 {
	return h.GuestReservedResource.StorageSize
}

func (h *HostDesc) GetTotalCPUCount(useRsvd bool) int64 {
	return reservedResourceMinusCal(h.TotalCPUCount, h.GuestReservedResource.CPUCount, useRsvd)
}

func (h *HostDesc) GetFreeCPUCount(useRsvd bool) int64 {
	return reservedResourceAddCal(h.FreeCPUCount, h.GuestReservedCPUCountFree(), useRsvd)
}

func (h *HostDesc) IndexKey() string {
	return h.ID
}

func (h *HostDesc) UnusedIsolatedDevices() []*IsolatedDeviceDesc {
	ret := make([]*IsolatedDeviceDesc, 0)
	for _, dev := range h.IsolatedDevices {
		if len(dev.GuestID) == 0 {
			ret = append(ret, dev)
		}
	}
	return ret
}

func (h *HostDesc) UnusedIsolatedDevicesByType(devType string) []*IsolatedDeviceDesc {
	ret := make([]*IsolatedDeviceDesc, 0)
	for _, dev := range h.UnusedIsolatedDevices() {
		if dev.DevType == devType {
			ret = append(ret, dev)
		}
	}
	return ret
}

func (h *HostDesc) UnusedIsolatedDevicesByVendorModel(vendorModel string) []*IsolatedDeviceDesc {
	ret := make([]*IsolatedDeviceDesc, 0)
	for _, dev := range h.UnusedIsolatedDevices() {
		if strings.Contains(dev.VendorModel(), vendorModel) {
			ret = append(ret, dev)
		}
	}
	return ret
}

func (h *HostDesc) UnusedIsolatedDevicesByModel(model string) []*IsolatedDeviceDesc {
	ret := make([]*IsolatedDeviceDesc, 0)
	for _, dev := range h.UnusedIsolatedDevices() {
		if strings.Contains(dev.Model, model) {
			ret = append(ret, dev)
		}
	}
	return ret
}

func (h *HostDesc) GetIsolatedDevice(devID string) *IsolatedDeviceDesc {
	for _, dev := range h.IsolatedDevices {
		if dev.ID == devID {
			return dev
		}
	}
	return nil
}

func (h *HostDesc) UnusedGpuDevices() []*IsolatedDeviceDesc {
	ret := make([]*IsolatedDeviceDesc, 0)
	for _, dev := range h.UnusedIsolatedDevices() {
		if strings.HasPrefix(dev.DevType, "GPU") {
			ret = append(ret, dev)
		}
	}
	return ret
}

type WaitGroupWrapper struct {
	gosync.WaitGroup
}

func (w *WaitGroupWrapper) Wrap(cb func()) {
	w.Add(1)
	go func() {
		cb()
		w.Done()
	}()
}

func waitTimeOut(wg *WaitGroupWrapper, timeout time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (b *HostBuilder) init(ids []string, dbCache DBGroupCacher, syncCache SyncGroupCacher) error {
	wg := &WaitGroupWrapper{}
	errMessageChannel := make(chan error, 12)
	defer close(errMessageChannel)
	setFuncs := []func(){
		func() { b.setClusters(dbCache, errMessageChannel) },
		func() { b.setHosts(ids, errMessageChannel) },
		func() {
			b.setGuests(ids, errMessageChannel)
			b.setGroupInfo(errMessageChannel)
			b.setMetadataInfo(ids, errMessageChannel)
		},
		func() {
			b.setStorages(ids, errMessageChannel)
			b.setDiskStats(errMessageChannel)
		},
		func() { b.setMetadataInfo(ids, errMessageChannel) },
		func() { b.setIsolatedDevs(ids, errMessageChannel) },
		func() { b.setCPUIOLoadInfo(errMessageChannel) },
	}

	for _, f := range setFuncs {
		wg.Wrap(f)
	}

	if ok := waitTimeOut(wg, time.Duration(20*time.Second)); !ok {
		log.Errorln("HostBuilder waitgroup timeout.")
	}

	if len(errMessageChannel) != 0 {
		errMessages := make([]string, 0)
		lengthChan := len(errMessageChannel)
		for ; lengthChan > 0; lengthChan-- {
			msg := fmt.Sprintf("%s", <-errMessageChannel)
			log.Errorf("Get error from chan: %s", msg)
			errMessages = append(errMessages, msg)
		}
		return fmt.Errorf("%s\n", strings.Join(errMessages, ";"))
	}

	return nil
}

func (b *HostBuilder) setClusters(dbCache DBGroupCacher, errMessageChannel chan error) {
	clusters, err := dbCache.Get(db.ClusterDBCache)
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.clusters = clusters
	return
}

func (b *HostBuilder) setHosts(ids []string, errMessageChannel chan error) {
	hosts, err := models.FetchHypervisorHostByIDs(ids)
	if err != nil {
		errMessageChannel <- err
		return
	}
	hostDict, err := utils.ToDict(hosts, func(obj interface{}) (string, error) {
		host, ok := obj.(*models.Host)
		if !ok {
			return "", utils.ConvertError(obj, "*models.Host")
		}
		return host.ID, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.hosts = hosts
	b.hostDict = hostDict
	return
}

func (b *HostBuilder) setStorages(ids []string, errMessageChannel chan error) {
	hostStorages, err := models.FetchByHostIDs(models.HostStorages, ids)
	if err != nil {
		errMessageChannel <- err
		return
	}
	storageIDs := make([]string, len(hostStorages))
	func() {
		for i, s := range hostStorages {
			storageIDs[i] = s.(*models.HostStorage).StorageID
		}
	}()
	storages, err := models.FetchByIDs(models.Storages, storageIDs)
	if err != nil {
		errMessageChannel <- err
		return
	}

	hostStoragesDict, err := utils.GroupBy(hostStorages, func(obj interface{}) (string, error) {
		storage, ok := obj.(*models.HostStorage)
		if !ok {
			return "", utils.ConvertError(obj, "*models.HostStorage")
		}
		return storage.HostID, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}
	storageDict, err := utils.ToDict(storages, func(obj interface{}) (string, error) {
		storage, ok := obj.(*models.Storage)
		if !ok {
			return "", utils.ConvertError(obj, "*models.Storage")
		}
		return storage.ID, nil
	})

	if err != nil {
		errMessageChannel <- err
		return
	}
	b.hostStorages = hostStorages
	b.hostStoragesDict = hostStoragesDict
	b.storages = storages
	b.storageDict = storageDict
	return
}

func (b *HostBuilder) setGuests(ids []string, errMessageChannel chan error) {
	guests, err := models.FetchGuestByHostIDs(ids)
	if err != nil {
		errMessageChannel <- err
		return
	}
	guestIDs := make([]string, len(guests))
	func() {
		for i, gst := range guests {
			guestIDs[i] = gst.(*models.Guest).ID
		}
	}()

	hostGuests, err := utils.GroupBy(guests, func(obj interface{}) (string, error) {
		gst, ok := obj.(*models.Guest)
		if !ok {
			return "", utils.ConvertError(obj, "*models.Guest")
		}
		return gst.HostID, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}
	guestDict, err := utils.ToDict(guests, func(obj interface{}) (string, error) {
		gst, ok := obj.(*models.Guest)
		if !ok {
			return "", utils.ConvertError(obj, "*models.Guest")
		}
		return gst.ID, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.guestIDs = guestIDs
	b.guests = guests
	b.hostGuests = hostGuests
	b.guestDict = guestDict
	return
}

func (b *HostBuilder) setGroupInfo(errMessageChannel chan error) {
	groupGuests, err := models.FetchByGuestIDs(models.GroupGuests, b.guestIDs)
	if err != nil {
		errMessageChannel <- err
		return
	}

	groupIds, err := utils.SelectDistinct(groupGuests, func(obj interface{}) (string, error) {
		g, ok := obj.(*models.GroupGuest)
		if !ok {
			return "", utils.ConvertError(obj, "*models.GroupGuest")
		}
		return g.GroupID, nil
	})

	if err != nil {
		errMessageChannel <- err
		return
	}

	groups, err := models.FetchGroupByIDs(groupIds)
	if err != nil {
		errMessageChannel <- err
		return
	}

	groupDict, err := utils.ToDict(groups, func(obj interface{}) (string, error) {
		grp, ok := obj.(*models.Group)
		if !ok {
			return "", utils.ConvertError(obj, "*models.Group")
		}
		return grp.ID, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.groups = groups
	b.groupDict = groupDict
	b.groupGuests = groupGuests
	hostGroupCountDict, err := b.toHostGroupCountDict(groupGuests)
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.hostGroupCountDict = hostGroupCountDict
	return
}

type GroupCount struct {
	ID    string `json:"id"`    // group id
	Name  string `json:"name"`  // group name
	Count int64  `json:"count"` // guest count
}

type GroupCounts struct {
	Data map[string]*GroupCount `json:"data"` // group_id: group_count
}

func NewGroupCounts() *GroupCounts {
	return &GroupCounts{
		Data: make(map[string]*GroupCount),
	}
}

type HostGroupCountDict map[string]*GroupCounts

func (b *HostBuilder) toHostGroupCountDict(groupGuests []interface{}) (HostGroupCountDict, error) {
	d := make(map[string]*GroupCounts)
	for _, groupGuestObj := range groupGuests {
		groupGuest := groupGuestObj.(*models.GroupGuest)
		groupObj, grpOK := b.groupDict[groupGuest.GroupID]
		guestObj, gstOK := b.guestDict[*groupGuest.GuestID]
		if !grpOK || !gstOK {
			continue
		}
		hostObj, ok := b.hostDict[guestObj.(*models.Guest).HostID]
		if !ok {
			continue
		}
		host := hostObj.(*models.Host)
		group := groupObj.(*models.Group)

		counts, ok := d[host.ID]
		if !ok {
			counts = NewGroupCounts()
			d[host.ID] = counts
		}
		count, ok := counts.Data[group.ID]
		if !ok {
			count = &GroupCount{ID: group.ID, Name: group.Name, Count: 1}
			counts.Data[group.ID] = count
		} else {
			count.Count++
		}
		counts.Data[host.ID] = count
	}
	return d, nil
}

func (b *HostBuilder) setMetadataInfo(hostIDs []string, errMessageChannel chan error) {
	hostMetadataNames := []string{"dynamic_load_cpu_percent", "dynamic_load_io_util",
		"enable_sriov", "bridge_driver"}
	hostMetadataNames = append(hostMetadataNames, models.HostExtraFeature...)
	hostMetadatas, err := models.FetchMetadatas(models.HostResourceName, hostIDs, hostMetadataNames)
	if err != nil {
		errMessageChannel <- err
		return
	}
	guestMetadataNames := []string{"app_tags"}
	guestMetadatas, err := models.FetchMetadatas(models.GuestResourceName, b.guestIDs, guestMetadataNames)
	if err != nil {
		errMessageChannel <- err
		return
	}
	idFunc := func(obj interface{}) (string, error) {
		metadata, ok := obj.(*models.Metadata)
		if !ok {
			return "", utils.ConvertError(obj, "*models.Metadata")
		}
		id := strings.Split(metadata.ID, "::")[1]
		return id, nil
	}
	hostMetadatasDict, err := utils.GroupBy(hostMetadatas, idFunc)
	if err != nil {
		errMessageChannel <- err
		return
	}
	guestMetadatasDict, err := utils.GroupBy(guestMetadatas, idFunc)
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.hostMetadatas = hostMetadatas
	b.hostMetadatasDict = hostMetadatasDict
	b.guestMetadatas = guestMetadatas
	b.guestMetadatasDict = guestMetadatasDict
	return
}

func (b *HostBuilder) setIsolatedDevs(ids []string, errMessageChannel chan error) {
	devs, err := models.FetchByHostIDs(models.IsolatedDevices, ids)
	if err != nil {
		errMessageChannel <- err
		return
	}
	dict, err := utils.GroupBy(devs, func(obj interface{}) (string, error) {
		dev, ok := obj.(*models.IsolatedDevice)
		if !ok {
			return "", utils.ConvertError(obj, "*models.IsolatedDevice")
		}
		return dev.HostID, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.isolatedDevicesDict = dict
}

func (b *HostBuilder) setDiskStats(errMessageChannel chan error) {
	storageIDs := make([]string, len(b.storages))
	func() {
		for i, s := range b.storages {
			storageIDs[i] = s.(*models.Storage).ID
		}
	}()
	capacities, err := models.GetStorageCapacities(storageIDs)
	stat3 := make([]utils.StatItem3, len(capacities))
	for i, item := range capacities {
		stat3[i] = item
	}
	if err != nil {
		errMessageChannel <- err
		return
	}
	storageStatesSizeDict, _ := utils.ToStatDict3(stat3)
	b.storageStatesSizeDict = storageStatesSizeDict
	b.diskStats = capacities
	return
}

func (b *HostBuilder) setCPUIOLoadInfo(errMessageChannel chan error) {
	return
}

func (b *HostBuilder) Clone() BuildActor {
	return &HostBuilder{}
}

func (b *HostBuilder) Type() string {
	return HostDescBuilder
}

func (b *HostBuilder) AllIDs() ([]string, error) {
	return models.AllHostIDs()
}

func (b *HostBuilder) Do(ids []string, dbCache DBGroupCacher, syncCache SyncGroupCacher) ([]interface{}, error) {
	err := b.init(ids, dbCache, syncCache)
	if err != nil {
		return nil, err
	}
	descs, err := b.build()
	if err != nil {
		return nil, err
	}
	return descs, nil
}

func (b *HostBuilder) build() ([]interface{}, error) {
	schedDescs := make([]interface{}, len(b.hosts))
	errs := []error{}
	var descResultLock gosync.Mutex
	var descedLen int32

	buildOne := func(i int) {
		if i >= len(b.hosts) {
			log.Errorf("invalid host index[%d] in b.hosts:%v\n", i, b.hosts)
			return
		}
		host := b.hosts[i]
		desc, err := b.buildOne(host.(*models.Host))
		if err != nil {
			descResultLock.Lock()
			errs = append(errs, err)
			descResultLock.Unlock()
			return
		}
		descResultLock.Lock()
		schedDescs[atomic.AddInt32(&descedLen, 1)-1] = desc
		descResultLock.Unlock()
	}

	workqueue.Parallelize(o.GetOptions().HostBuildParallelizeSize, len(b.hosts), buildOne)
	schedDescs = schedDescs[:descedLen]
	if len(errs) > 0 {
		//return nil, errors.NewAggregate(errs)
		err := errors.NewAggregate(errs)
		log.Warningf("Build schedule descs error: %s", err)
	}

	return schedDescs, nil
}

func (b *HostBuilder) buildOne(host *models.Host) (interface{}, error) {
	baseDesc, err := newBaseHostDesc(host)
	if err != nil {
		return nil, err
	}
	desc := &HostDesc{
		baseHostDesc: baseDesc,
	}

	desc.Metadata = make(map[string]string)
	desc.ManagerID = host.ManagerID

	desc.CPUCmtbound = host.CPUOverCommitBound()
	desc.CPUDesc = host.CPUDesc
	desc.CPUCache = host.CPUCache
	desc.CPUReserved = host.CPUReserved
	desc.NodeCount = host.NodeCount
	desc.CPUMHZ = host.CPUMHZ

	desc.MemCmtbound = host.MemOverCommitBound()
	desc.MemReserved = host.MemReserved

	desc.GuestReservedResource = NewGuestReservedResourceByBuilder(b, host)
	guestRsvdUsed, err := NewGuestReservedResourceUsedByBuilder(b, host)
	if err != nil {
		return nil, err
	}
	desc.GuestReservedResourceUsed = guestRsvdUsed

	fillFuncs := []func(*HostDesc, *models.Host) error{
		b.fillGuestsResourceInfo,
		b.fillStorages,
		b.fillResidentGroups,
		b.fillMetadata,
		b.fillIsolatedDevices,
		b.fillCPUIOLoads,
	}

	for _, f := range fillFuncs {
		err := f(desc, host)
		if err != nil {
			return nil, err
		}
	}

	return desc, nil
}

func _in(s string, ss []string) bool {
	for _, str := range ss {
		if s == str {
			return true
		}
	}
	return false
}

func (b *HostBuilder) fillGuestsResourceInfo(desc *HostDesc, host *models.Host) error {
	var (
		guestCount          int64
		runningCount        int64
		memSize             int64
		memReqSize          int64
		memFakeDeletedSize  int64
		cpuCount            int64
		cpuReqCount         int64
		cpuBoundCount       int64
		cpuFakeDeletedCount int64
		ioBoundCount        int64
		creatingMemSize     int64
		creatingCPUCount    int64
		creatingGuestCount  int64
	)
	guestsOnHost, ok := b.hostGuests[host.ID]
	if !ok {
		guestsOnHost = []interface{}{}
	}

	for _, gst := range guestsOnHost {
		guest := gst.(*models.Guest)
		if guest.IsRunning() {
			runningCount++
			memSize += guest.VMemSize
			cpuCount += guest.VCPUCount
		} else if guest.IsCreating() {
			creatingGuestCount++
			creatingMemSize += guest.VMemSize
			creatingCPUCount += guest.VCPUCount
		} else if guest.IsGuestFakeDeleted() && _in(guest.Status, []string{models.VmReady}) {
			memFakeDeletedSize += guest.VMemSize
			cpuFakeDeletedCount += guest.VCPUCount
		}
		guestCount++
		cpuReqCount += guest.VCPUCount
		memReqSize += guest.VMemSize

		appTags := b.guestAppTags(guest)
		for _, tag := range appTags {
			if tag == "cpu_bound" {
				cpuBoundCount += guest.VCPUCount
			} else if tag == "io_bound" {
				ioBoundCount++
			}
		}
	}
	desc.GuestCount = guestCount
	desc.CreatingGuestCount = creatingGuestCount
	desc.RunningGuestCount = runningCount
	desc.RunningMemSize = memSize
	desc.RequiredMemSize = memReqSize
	desc.CreatingMemSize = creatingMemSize
	desc.FakeDeletedMemSize = memFakeDeletedSize
	desc.RunningCPUCount = cpuCount
	desc.RequiredCPUCount = cpuReqCount
	desc.CreatingCPUCount = creatingCPUCount
	desc.FakeDeletedCPUCount = cpuFakeDeletedCount

	desc.TotalMemSize = int64(float64(desc.MemSize) * desc.MemCmtbound)
	desc.TotalCPUCount = int64(float64(desc.CPUCount) * desc.CPUCmtbound)

	var memFreeSize int64
	var cpuFreeCount int64
	if o.GetOptions().IgnoreNonRunningGuests {
		memFreeSize = desc.TotalMemSize - desc.RunningMemSize - desc.CreatingMemSize
		cpuFreeCount = desc.TotalCPUCount - desc.RunningCPUCount - desc.CreatingCPUCount
	} else {
		memFreeSize = desc.TotalMemSize - desc.RequiredMemSize
		cpuFreeCount = desc.TotalCPUCount - desc.RequiredCPUCount
		if o.GetOptions().IgnoreFakeDeletedGuests {
			memFreeSize += memFakeDeletedSize
			cpuFreeCount += cpuFakeDeletedCount
		}
	}

	// free memory size calculate
	rsvdUseMem := desc.GuestReservedResourceUsed.MemorySize
	memFreeSize = memFreeSize + rsvdUseMem - desc.GetReservedMemSize()
	memSub := desc.GuestReservedResource.MemorySize - desc.GuestReservedResourceUsed.MemorySize
	if memSub < 0 {
		memFreeSize += memSub
	}
	desc.FreeMemSize = memFreeSize

	// free cpu count calculate
	rsvdUseCPU := desc.GuestReservedResourceUsed.CPUCount
	cpuFreeCount = cpuFreeCount + rsvdUseCPU - desc.GetReservedCPUCount()
	cpuSub := desc.GuestReservedResource.CPUCount - desc.GuestReservedResourceUsed.CPUCount
	if cpuSub < 0 {
		cpuFreeCount += cpuSub
	}
	desc.FreeCPUCount = cpuFreeCount

	desc.CPUBoundCount = cpuBoundCount
	desc.IOBoundCount = ioBoundCount

	return nil
}

func (b *HostBuilder) guestAppTags(guest *models.Guest) []string {
	metadatas, ok := b.guestMetadatasDict[guest.ID]
	if !ok {
		return []string{}
	}
	for _, obj := range metadatas {
		metadata, ok := obj.(*models.Metadata)
		if !ok {
			log.Errorf("%v", utils.ConvertError(obj, "*models.Metadata"))
			return []string{}
		}
		if metadata.Key == "app_tags" {
			tagsStr := metadata.Value
			if len(tagsStr) > 0 {
				return strings.Split(tagsStr, ",")
			}
		}
	}
	return []string{}
}

func (b *HostBuilder) fillStorages(desc *HostDesc, host *models.Host) error {
	objs, ok := b.hostStoragesDict[host.ID]
	if !ok {
		return nil
	}

	var (
		rets           = make([]*Storage, 0)
		storageTypeMap = make(map[string]int, 0)
	)
	for _, obj := range objs {
		hostStorage, ok := obj.(*models.HostStorage)
		if !ok {
			return utils.ConvertError(obj, "*models.HostStorage")
		}
		storageID := hostStorage.StorageID
		storageObj, ok := b.storageDict[storageID]
		if !ok {
			log.Warningf("Storage ID: %q not found when fill it", storageID)
			return nil
		}
		storageModel := storageObj.(*models.Storage)
		storage := new(Storage)
		storage.ID = storageModel.ID
		storage.Name = storageModel.Name
		storage.Capacity = storageModel.Capacity
		storage.StorageType = storageModel.StorageType
		storage.UsedCapacity = b.storageUsedCapacity(storageModel, true)
		storage.WasteCapacity = b.storageUsedCapacity(storageModel, false)
		storage.Cmtbound = storageModel.OverCommitBound()
		storage.VCapacity = storage.GetTotalSize()
		storage.FreeCapacity = storage.GetFreeSize()
		rets = append(rets, storage)

		storageTypeMap[storage.StorageType] = 0
	}

	desc.Storages = rets

	for storageType := range storageTypeMap {
		desc.StorageTypes = append(desc.StorageTypes, storageType)
	}

	return nil
}

func (b *HostBuilder) storageUsedCapacity(storage *models.Storage, ready bool) int64 {
	d, ok := b.storageStatesSizeDict[storage.ID]
	if !ok {
		return 0
	}
	if ready {
		obj, ok := d[models.DiskReady]
		if !ok {
			return 0
		}
		return obj.(int64)
	}
	var total int64
	for status, sizeObj := range d {
		if (status == models.DiskReady && ready) || (status != models.DiskReady && !ready) {
			total += sizeObj.(int64)
		}
	}
	return total
}

func (b *HostBuilder) fillResidentGroups(desc *HostDesc, host *models.Host) error {
	groups, ok := b.hostGroupCountDict[host.ID]
	if !ok {
		desc.Groups = nil
		return nil
	}
	desc.Groups = groups
	return nil
}

func (b *HostBuilder) fillMetadata(desc *HostDesc, host *models.Host) error {
	metadataObjs, ok := b.hostMetadatasDict[host.ID]
	if !ok {
		return nil
	}
	for _, obj := range metadataObjs {
		metadata, ok := obj.(*models.Metadata)
		if !ok {
			return utils.ConvertError(obj, "*models.Metadata")
		}
		desc.Metadata[metadata.Key] = metadata.Value
	}
	return nil
}

type IsolatedDeviceDesc struct {
	ID             string
	GuestID        string
	HostID         string
	DevType        string
	Model          string
	Addr           string
	VendorDeviceID string
}

func (i *IsolatedDeviceDesc) VendorID() string {
	return strings.Split(i.VendorDeviceID, ":")[0]
}

func (i *IsolatedDeviceDesc) VendorModel() string {
	return fmt.Sprintf("%s:%s", i.VendorID(), i.Model)
}

func (b *HostBuilder) getIsolatedDevices(hostID string) (devs []*models.IsolatedDevice) {
	devObjs, ok := b.isolatedDevicesDict[hostID]
	devs = make([]*models.IsolatedDevice, 0)
	if !ok {
		return
	}
	for _, obj := range devObjs {
		dev := obj.(*models.IsolatedDevice)
		devs = append(devs, dev)
	}
	return
}

func (b *HostBuilder) getUsedIsolatedDevices(hostID string) (devs []*models.IsolatedDevice) {
	devs = make([]*models.IsolatedDevice, 0)
	for _, dev := range b.getIsolatedDevices(hostID) {
		if len(dev.GuestID) != 0 {
			devs = append(devs, dev)
		}
	}
	return
}

func (b *HostBuilder) getIsolatedDeviceGuests(hostID string) (guests []*models.Guest) {
	guests = make([]*models.Guest, 0)
	usedDevs := b.getUsedIsolatedDevices(hostID)
	if len(usedDevs) == 0 {
		return
	}
	ids := sets.NewString()
	for _, dev := range usedDevs {
		g, ok := b.guestDict[dev.GuestID]
		if !ok {
			continue
		}
		guest := g.(*models.Guest)
		if !ids.Has(guest.ID) {
			ids.Insert(guest.ID)
			guests = append(guests, guest)
		}
	}
	return
}

func (b *HostBuilder) getUnusedIsolatedDevices(hostID string) (devs []*models.IsolatedDevice) {
	devs = make([]*models.IsolatedDevice, 0)
	for _, dev := range b.getIsolatedDevices(hostID) {
		if len(dev.GuestID) == 0 {
			devs = append(devs, dev)
		}
	}
	return
}

func (b *HostBuilder) fillIsolatedDevices(desc *HostDesc, host *models.Host) error {

	allDevs := b.getIsolatedDevices(host.ID)
	if len(allDevs) == 0 {
		return nil
	}

	devs := make([]*IsolatedDeviceDesc, len(allDevs))
	for index, devModel := range allDevs {
		dev := &IsolatedDeviceDesc{
			ID:             devModel.ID,
			GuestID:        devModel.GuestID,
			HostID:         devModel.HostID,
			DevType:        devModel.DevType,
			Model:          devModel.Model,
			Addr:           devModel.Addr,
			VendorDeviceID: devModel.VendorDeviceID,
		}
		devs[index] = dev
	}
	desc.IsolatedDevices = devs

	return nil
}

func (b *HostBuilder) fillCPUIOLoads(desc *HostDesc, host *models.Host) error {
	desc.CPULoad = b.loadByName(host.ID, "cpu_load")
	desc.IOLoad = b.loadByName(host.ID, "io_load")
	return nil
}

func (b *HostBuilder) loadByName(hostID, name string) *float64 {
	if b.cpuIOLoads == nil {
		return nil
	}
	loads, ok := b.cpuIOLoads[hostID]
	if !ok {
		return nil
	}
	value := loads[name]
	if value >= 0.0 && value <= 1.0 {
		return &value
	}
	return nil
}
