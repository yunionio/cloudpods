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

package candidate

import (
	"encoding/json"
	"fmt"
	"strings"
	gosync "sync"
	"sync/atomic"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/util/workqueue"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	computedb "yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
)

type hostGetter struct {
	*baseHostGetter
	h *HostDesc
}

func newHostGetter(h *HostDesc) *hostGetter {
	return &hostGetter{
		baseHostGetter: newBaseHostGetter(h.BaseHostDesc),
		h:              h,
	}
}

func (h *hostGetter) CreatingGuestCount() int {
	return int(h.h.CreatingGuestCount)
}

func (h *hostGetter) RunningCPUCount() int64 {
	return h.h.RunningCPUCount
}

func (h *hostGetter) TotalCPUCount(useRsvd bool) int64 {
	return h.h.GetTotalCPUCount(useRsvd)
}

func (h *hostGetter) FreeCPUCount(useRsvd bool) int64 {
	return h.h.GetFreeCPUCount(useRsvd)
}

func (h *hostGetter) FreeMemorySize(useRsvd bool) int64 {
	return h.h.GetFreeMemSize(useRsvd)
}

func (h *hostGetter) RunningMemorySize() int64 {
	return h.h.RunningMemSize
}

func (h *hostGetter) TotalMemorySize(useRsvd bool) int64 {
	return h.h.GetTotalMemSize(useRsvd)
}

func (h *hostGetter) IsEmpty() bool {
	return h.h.GuestCount == 0
}

func (h *hostGetter) StorageInfo() []*baremetal.BaremetalStorage {
	return nil
}

func (h *hostGetter) GetFreeStorageSizeOfType(storageType string, useRsvd bool) int64 {
	return h.h.GetFreeStorageSizeOfType(storageType, useRsvd)
}

func (h *hostGetter) GetFreePort(netId string) int {
	return h.h.GetFreePort(netId)
}

func (h *hostGetter) OvnCapable() bool {
	return len(h.h.OvnVersion) > 0
}

type HostDesc struct {
	*BaseHostDesc

	// cpu
	CPUCmtbound         float32  `json:"cpu_cmtbound"`
	CPUBoundCount       int64    `json:"cpu_bound_count"`
	CPULoad             *float64 `json:"cpu_load"`
	TotalCPUCount       int64    `json:"total_cpu_count"`
	RunningCPUCount     int64    `json:"running_cpu_count"`
	CreatingCPUCount    int64    `json:"creating_cpu_count"`
	RequiredCPUCount    int64    `json:"required_cpu_count"`
	FakeDeletedCPUCount int64    `json:"fake_deleted_cpu_count"`
	FreeCPUCount        int64    `json:"free_cpu_count"`

	// memory
	MemCmtbound        float32 `json:"mem_cmtbound"`
	TotalMemSize       int64   `json:"total_mem_size"`
	FreeMemSize        int64   `json:"free_mem_size"`
	RunningMemSize     int64   `json:"running_mem_size"`
	CreatingMemSize    int64   `json:"creating_mem_size"`
	RequiredMemSize    int64   `json:"required_mem_size"`
	FakeDeletedMemSize int64   `json:"fake_deleted_mem_size"`

	// storage
	StorageTypes []string `json:"storage_types"`

	// IO
	IOBoundCount int64    `json:"io_bound_count"`
	IOLoad       *float64 `json:"io_load"`

	// server
	GuestCount         int64 `json:"guest_count"`
	CreatingGuestCount int64 `json:"creating_guest_count"`
	RunningGuestCount  int64 `json:"running_guest_count"`

	//Groups                    *GroupCounts          `json:"groups"`
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

func NewGuestReservedResourceByBuilder(b *HostBuilder, host *computemodels.SHost) (ret *ReservedResource) {
	ret = NewReservedResource(0, 0, 0)
	//isoDevs := b.getUnusedIsolatedDevices(host.ID)
	isoDevs := b.getIsolatedDevices(host.Id)
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

func NewGuestReservedResourceUsedByBuilder(b *HostBuilder, host *computemodels.SHost, free *ReservedResource) (ret *ReservedResource, err error) {
	ret = NewReservedResource(0, 0, 0)
	gst := b.getIsolatedDeviceGuests(host.Id)
	if len(gst) == 0 {
		return
	}
	var (
		cpu  int64 = 0
		mem  int64 = 0
		disk int64 = 0
	)
	guestDiskSize := func(g *computemodels.SGuest, onlyLocal bool) int {
		size := 0
		for _, gd := range g.GetDisks() {
			disk := gd.GetDisk()
			if !onlyLocal || disk.IsLocal() {
				size += disk.DiskSize
			}
		}
		return size
	}
	for _, g := range gst {
		dSize := guestDiskSize(&g, true)
		disk += int64(dSize)
		if o.GetOptions().IgnoreNonrunningGuests && !utils.IsInStringArray(g.Status, computeapi.VM_RUNNING_STATUS) {
			continue
		}
		cpu += int64(g.VcpuCount)
		mem += int64(g.VmemSize)
	}
	usedF := func(used, free int64) int64 {
		if used <= free {
			return used
		}
		return free
	}
	ret.CPUCount = usedF(cpu, free.CPUCount)
	ret.MemorySize = usedF(mem, free.MemorySize)
	ret.StorageSize = usedF(disk, free.StorageSize)
	return
}

type HostBuilder struct {
	*baseBuilder

	residentTenantDict map[string]map[string]interface{}

	hosts    []computemodels.SHost
	hostDict map[string]interface{}

	guests    []computemodels.SGuest
	guestDict map[string]interface{}
	guestIDs  []string

	hostStorages []computemodels.SHoststorage
	//hostStoragesDict      map[string][]*computemodels.SStorage
	storages              []interface{}
	storageStatesSizeDict map[string]map[string]interface{}

	hostGuests       map[string][]interface{}
	hostBackupGuests map[string][]interface{}

	//groupGuests        []interface{}
	//groups             []interface{}
	//groupDict          map[string]interface{}
	//hostGroupCountDict HostGroupCountDict

	//hostMetadatas      []interface{}
	//hostMetadatasDict  map[string][]interface{}
	//guestMetadatas     []interface{}
	//guestMetadatasDict map[string][]interface{}

	//diskStats           []models.StorageCapacity
	isolatedDevicesDict map[string][]interface{}

	cpuIOLoads map[string]map[string]float64

	schedtags []computemodels.SSchedtag
	zoneSkus  map[string][]computemodels.SServerSku
}

func newHostBuilder() *HostBuilder {
	return &HostBuilder{
		baseBuilder: newBaseBuilder(HostDescBuilder),
	}
}

func (h *HostDesc) String() string {
	s, _ := json.Marshal(h)
	return string(s)
}

func (h *HostDesc) Type() int {
	// Guest type
	return 0
}

func (h *HostDesc) Getter() core.CandidatePropertyGetter {
	return newHostGetter(h)
}

func (h *HostDesc) GetGuestCount() int64 {
	return h.GuestCount
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
			total += int64(storage.GetCapacity())
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
			total += int64(storage.GetFreeCapacity())
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
	var total int64
	for _, storage := range h.Storages {
		if storage.StorageType == storageType {
			total += int64(storage.GetFreeCapacity())
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

	return total - int64(h.GetPendingUsage().DiskUsage.Get(storageType))
}

func (h *HostDesc) GetFreePort(netId string) int {
	freeCnt := h.BaseHostDesc.GetFreePort(netId)
	return freeCnt - h.GetPendingUsage().NetUsage.Get(netId)
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
	return reservedResourceAddCal(h.FreeMemSize, h.GuestReservedMemSizeFree(), useRsvd) - int64(h.GetPendingUsage().Memory)
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
	return h.GuestReservedResource.MemorySize + int64(h.MemReserved)
}

func (h *HostDesc) GetReservedCPUCount() int64 {
	return h.GuestReservedResource.CPUCount + int64(h.CpuReserved)
}

func (h *HostDesc) GetReservedStorageSize() int64 {
	return h.GuestReservedResource.StorageSize
}

func (h *HostDesc) GetTotalCPUCount(useRsvd bool) int64 {
	return reservedResourceMinusCal(h.TotalCPUCount, h.GuestReservedResource.CPUCount, useRsvd)
}

func (h *HostDesc) GetFreeCPUCount(useRsvd bool) int64 {
	return reservedResourceAddCal(h.FreeCPUCount, h.GuestReservedCPUCountFree(), useRsvd) - int64(h.GetPendingUsage().Cpu)
}

func (h *HostDesc) IndexKey() string {
	return h.Id
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
	vm := NewVendorModelByStr(vendorModel)
	for _, dev := range h.UnusedIsolatedDevices() {
		if dev.GetVendorModel().IsMatch(vm) {
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

func (b *HostBuilder) init(ids []string) error {
	wg := &WaitGroupWrapper{}
	errMessageChannel := make(chan error, 12)
	defer close(errMessageChannel)
	setFuncs := []func(){
		func() { b.setHosts(ids, errMessageChannel) },
		func() { b.setSchedtags(ids, errMessageChannel) },
		func() {
			b.setGuests(ids, errMessageChannel)
			b.setIsolatedDevs(ids, errMessageChannel)
		},
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

func (b *HostBuilder) setHosts(ids []string, errMessageChannel chan error) {
	hosts := computemodels.HostManager.Query()
	q := hosts.In("id", ids).NotEquals("host_type", computeapi.HOST_TYPE_BAREMETAL)
	hostObjs := make([]computemodels.SHost, 0)
	err := computedb.FetchModelObjects(computemodels.HostManager, q, &hostObjs)
	if err != nil {
		errMessageChannel <- err
		return
	}

	hostDict, err := utils.ToDict(hostObjs, func(obj interface{}) (string, error) {
		host, ok := obj.(computemodels.SHost)
		if !ok {
			return "", utils.ConvertError(obj, "computemodels.Host")
		}
		return host.Id, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.hosts = hostObjs
	b.hostDict = hostDict
	return
}

func (b *HostBuilder) setSchedtags(ids []string, errMessageChannel chan error) {
	tags := make([]computemodels.SSchedtag, 0)
	if err := computemodels.SchedtagManager.Query().All(&tags); err != nil {
		errMessageChannel <- err
		return
	}
	b.schedtags = tags
}

func (b *HostBuilder) setGuests(ids []string, errMessageChannel chan error) {
	guests, err := FetchGuestByHostIDs(ids)
	if err != nil {
		errMessageChannel <- err
		return
	}
	guestIDs := make([]string, len(guests))
	func() {
		for i, gst := range guests {
			guestIDs[i] = gst.GetId()
		}
	}()

	hostGuests, err := utils.GroupBy(guests, func(obj interface{}) (string, error) {
		gst, ok := obj.(computemodels.SGuest)
		if !ok {
			return "", utils.ConvertError(obj, "computemodels.SGuest")
		}
		return gst.HostId, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}

	hostBackupGuests, err := utils.GroupBy(guests, func(obj interface{}) (string, error) {
		gst, ok := obj.(computemodels.SGuest)
		if !ok {
			return "", utils.ConvertError(obj, "computemodels.SGuest")
		}
		return gst.BackupHostId, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}

	guestDict, err := utils.ToDict(guests, func(obj interface{}) (string, error) {
		gst, ok := obj.(computemodels.SGuest)
		if !ok {
			return "", utils.ConvertError(obj, "computemodels.SGuest")
		}
		return gst.GetId(), nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.guestIDs = guestIDs
	b.guests = guests
	b.hostGuests = hostGuests
	b.hostBackupGuests = hostBackupGuests
	b.guestDict = guestDict
	return
}

//func (b *HostBuilder) setGroupInfo(errMessageChannel chan error) {
//groupGuests, err := models.FetchByGuestIDs(models.GroupGuests, b.guestIDs)
//if err != nil {
//errMessageChannel <- err
//return
//}

//groupIds, err := utils.SelectDistinct(groupGuests, func(obj interface{}) (string, error) {
//g, ok := obj.(*models.GroupGuest)
//if !ok {
//return "", utils.ConvertError(obj, "*models.GroupGuest")
//}
//return g.GroupID, nil
//})

//if err != nil {
//errMessageChannel <- err
//return
//}

//groups, err := models.FetchGroupByIDs(groupIds)
//if err != nil {
//errMessageChannel <- err
//return
//}

//groupDict, err := utils.ToDict(groups, func(obj interface{}) (string, error) {
//grp, ok := obj.(*models.Group)
//if !ok {
//return "", utils.ConvertError(obj, "*models.Group")
//}
//return grp.ID, nil
//})
//if err != nil {
//errMessageChannel <- err
//return
//}
//b.groups = groups
//b.groupDict = groupDict
//b.groupGuests = groupGuests
//hostGroupCountDict, err := b.toHostGroupCountDict(groupGuests)
//if err != nil {
//errMessageChannel <- err
//return
//}
//b.hostGroupCountDict = hostGroupCountDict
//return
//}

//type GroupCount struct {
//ID    string `json:"id"`    // group id
//Name  string `json:"name"`  // group name
//Count int64  `json:"count"` // guest count
//}

//type GroupCounts struct {
//Data map[string]*GroupCount `json:"data"` // group_id: group_count
//}

//func NewGroupCounts() *GroupCounts {
//return &GroupCounts{
//Data: make(map[string]*GroupCount),
//}
//}

//type HostGroupCountDict map[string]*GroupCounts

//func (b *HostBuilder) toHostGroupCountDict(groupGuests []interface{}) (HostGroupCountDict, error) {
//d := make(map[string]*GroupCounts)
//for _, groupGuestObj := range groupGuests {
//groupGuest := groupGuestObj.(*models.GroupGuest)
//groupObj, grpOK := b.groupDict[groupGuest.GroupID]
//guestObj, gstOK := b.guestDict[*groupGuest.GuestID]
//if !grpOK || !gstOK {
//continue
//}
//hostObj, ok := b.hostDict[guestObj.(*models.Guest).HostID]
//if !ok {
//continue
//}
//host := hostObj.(*models.Host)
//group := groupObj.(*models.Group)

//counts, ok := d[host.ID]
//if !ok {
//counts = NewGroupCounts()
//d[host.ID] = counts
//}
//count, ok := counts.Data[group.ID]
//if !ok {
//count = &GroupCount{ID: group.ID, Name: group.Name, Count: 1}
//counts.Data[group.ID] = count
//} else {
//count.Count++
//}
//counts.Data[host.ID] = count
//}
//return d, nil
//}

//func (b *HostBuilder) setMetadataInfo(hostIDs []string, errMessageChannel chan error) {
//hostMetadataNames := []string{"dynamic_load_cpu_percent", "dynamic_load_io_util",
//"enable_sriov", "bridge_driver"}
//hostMetadataNames = append(hostMetadataNames, models.HostExtraFeature...)
//hostMetadatas, err := models.FetchMetadatas(models.HostResourceName, hostIDs, hostMetadataNames)
//if err != nil {
//errMessageChannel <- err
//return
//}
//guestMetadataNames := []string{"app_tags"}
//guestMetadatas, err := models.FetchMetadatas(models.GuestResourceName, b.guestIDs, guestMetadataNames)
//if err != nil {
//errMessageChannel <- err
//return
//}
//idFunc := func(obj interface{}) (string, error) {
//metadata, ok := obj.(*models.Metadata)
//if !ok {
//return "", utils.ConvertError(obj, "*models.Metadata")
//}
//id := strings.Split(metadata.ID, "::")[1]
//return id, nil
//}
//hostMetadatasDict, err := utils.GroupBy(hostMetadatas, idFunc)
//if err != nil {
//errMessageChannel <- err
//return
//}
//guestMetadatasDict, err := utils.GroupBy(guestMetadatas, idFunc)
//if err != nil {
//errMessageChannel <- err
//return
//}
//b.hostMetadatas = hostMetadatas
//b.hostMetadatasDict = hostMetadatasDict
//b.guestMetadatas = guestMetadatas
//b.guestMetadatasDict = guestMetadatasDict
//return
//}

func (b *HostBuilder) setIsolatedDevs(ids []string, errMessageChannel chan error) {
	devs := computemodels.IsolatedDeviceManager.FindByHosts(ids)
	dict, err := utils.GroupBy(devs, func(obj interface{}) (string, error) {
		dev, ok := obj.(computemodels.SIsolatedDevice)
		if !ok {
			return "", utils.ConvertError(obj, "computemodels.SIsolatedDevice")
		}
		return dev.HostId, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.isolatedDevicesDict = dict
}

/*func (b *HostBuilder) setDiskStats(errMessageChannel chan error) {
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
}*/

func (b *HostBuilder) Clone() BuildActor {
	return &HostBuilder{
		baseBuilder: newBaseBuilder(HostDescBuilder),
	}
}

func (b *HostBuilder) AllIDs() ([]string, error) {
	q := computemodels.HostManager.Query("id")
	q = q.Filter(sqlchemy.NotEquals(q.Field("host_type"), computeapi.HOST_TYPE_BAREMETAL))
	return FetchModelIds(q)
}

func (b *HostBuilder) Do(ids []string) ([]interface{}, error) {
	err := b.init(ids)
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
		desc, err := b.buildOne(&host)
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
		log.V(4).Warningf("Build schedule descs error: %s", err)
	}

	return schedDescs, nil
}

func (b *HostBuilder) buildOne(host *computemodels.SHost) (interface{}, error) {
	baseDesc, err := newBaseHostDesc(host)
	if err != nil {
		return nil, err
	}
	desc := &HostDesc{
		BaseHostDesc: baseDesc,
	}

	desc.Metadata = make(map[string]string)

	desc.CPUCmtbound = host.GetCPUOvercommitBound()
	desc.MemCmtbound = host.GetMemoryOvercommitBound()

	desc.GuestReservedResource = NewGuestReservedResourceByBuilder(b, host)
	guestRsvdUsed, err := NewGuestReservedResourceUsedByBuilder(b, host, desc.GuestReservedResource)
	if err != nil {
		return nil, err
	}
	desc.GuestReservedResourceUsed = guestRsvdUsed

	fillFuncs := []func(*HostDesc, *computemodels.SHost) error{
		b.fillGuestsResourceInfo,
		//b.fillResidentGroups,
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

func (b *HostBuilder) fillGuestsResourceInfo(desc *HostDesc, host *computemodels.SHost) error {
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
	guestsOnHost, ok := b.hostGuests[host.Id]
	if !ok {
		guestsOnHost = []interface{}{}
	}
	backupGuestsOnHost, ok := b.hostBackupGuests[host.Id]
	if ok {
		guestsOnHost = append(guestsOnHost, backupGuestsOnHost...)
	}

	for _, gst := range guestsOnHost {
		guest := gst.(computemodels.SGuest)
		if IsGuestRunning(guest) {
			runningCount++
			memSize += int64(guest.VmemSize)
			cpuCount += int64(guest.VcpuCount)
		} else if IsGuestCreating(guest) {
			creatingGuestCount++
			creatingMemSize += int64(guest.VmemSize)
			creatingCPUCount += int64(guest.VcpuCount)
		} else if IsGuestPendingDelete(guest) {
			memFakeDeletedSize += int64(guest.VmemSize)
			cpuFakeDeletedCount += int64(guest.VcpuCount)
		}
		guestCount++
		cpuReqCount += int64(guest.VcpuCount)
		memReqSize += int64(guest.VmemSize)

		//appTags := b.guestAppTags(guest)
		//for _, tag := range appTags {
		//if tag == "cpu_bound" {
		//cpuBoundCount += int64(guest.VcpuCount)
		//} else if tag == "io_bound" {
		//ioBoundCount++
		//}
		//}
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

	desc.TotalMemSize = int64(float32(desc.MemSize) * desc.MemCmtbound)
	desc.TotalCPUCount = int64(float32(desc.CpuCount) * desc.CPUCmtbound)

	var memFreeSize int64
	var cpuFreeCount int64
	if o.GetOptions().IgnoreNonrunningGuests {
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

/*func (b *HostBuilder) guestAppTags(guest computemodels.SGuest) []string {
	metadatas, ok := b.guestMetadatasDict[guest.GetId()]
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

func (b *HostBuilder) fillResidentGroups(desc *HostDesc, host *computemodels.SHost) error {
	groups, ok := b.hostGroupCountDict[host.Id]
	if !ok {
		desc.Groups = nil
		return nil
	}
	desc.Groups = groups
	return nil
}*/

func (b *HostBuilder) fillMetadata(desc *HostDesc, host *computemodels.SHost) error {
	metadata, err := host.GetAllMetadata(nil)
	if err != nil {
		log.Errorf("Get host %s metadata: %v", desc.GetId(), err)
		return nil
	}
	desc.Metadata = metadata
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

type VendorModel struct {
	Vendor string
	Model  string
}

func NewVendorModelByStr(desc string) *VendorModel {
	vm := new(VendorModel)
	// desc format is '<vendor>:<model>'
	parts := strings.Split(desc, ":")
	if len(parts) == 1 {
		vm.Model = parts[0]
	} else if len(parts) == 2 {
		vm.Vendor = parts[0]
		vm.Model = parts[1]
	}
	return vm
}

func (vm *VendorModel) IsMatch(target *VendorModel) bool {
	if vm.Model == "" || target.Model == "" {
		return false
	}
	vendorMatch := false
	modelMatch := false
	if target.Vendor != "" {
		if vm.Vendor == target.Vendor {
			vendorMatch = true
		} else if computeapi.ID_VENDOR_MAP[vm.Vendor] == target.Vendor {
			vendorMatch = true
		}
	} else {
		vendorMatch = true
	}
	if vm.Model == target.Model {
		modelMatch = true
	}
	return vendorMatch && modelMatch
}

func (i *IsolatedDeviceDesc) GetVendorModel() *VendorModel {
	return &VendorModel{
		Vendor: i.VendorID(),
		Model:  i.Model,
	}
}

func (b *HostBuilder) getIsolatedDevices(hostID string) (devs []computemodels.SIsolatedDevice) {
	devObjs, ok := b.isolatedDevicesDict[hostID]
	devs = make([]computemodels.SIsolatedDevice, 0)
	if !ok {
		return
	}
	for _, obj := range devObjs {
		dev := obj.(computemodels.SIsolatedDevice)
		devs = append(devs, dev)
	}
	return
}

func (b *HostBuilder) getUsedIsolatedDevices(hostID string) (devs []computemodels.SIsolatedDevice) {
	devs = make([]computemodels.SIsolatedDevice, 0)
	for _, dev := range b.getIsolatedDevices(hostID) {
		if len(dev.GuestId) != 0 {
			devs = append(devs, dev)
		}
	}
	return
}

func (b *HostBuilder) getIsolatedDeviceGuests(hostID string) (guests []computemodels.SGuest) {
	guests = make([]computemodels.SGuest, 0)
	usedDevs := b.getUsedIsolatedDevices(hostID)
	if len(usedDevs) == 0 {
		return
	}
	ids := sets.NewString()
	for _, dev := range usedDevs {
		g, ok := b.guestDict[dev.GuestId]
		if !ok {
			continue
		}
		guest := g.(computemodels.SGuest)
		if !ids.Has(guest.Id) {
			ids.Insert(guest.Id)
			guests = append(guests, guest)
		}
	}
	return
}

func (b *HostBuilder) getUnusedIsolatedDevices(hostID string) (devs []computemodels.SIsolatedDevice) {
	devs = make([]computemodels.SIsolatedDevice, 0)
	for _, dev := range b.getIsolatedDevices(hostID) {
		if len(dev.GuestId) == 0 {
			devs = append(devs, dev)
		}
	}
	return
}

func (b *HostBuilder) fillIsolatedDevices(desc *HostDesc, host *computemodels.SHost) error {

	allDevs := b.getIsolatedDevices(host.Id)
	if len(allDevs) == 0 {
		return nil
	}

	devs := make([]*IsolatedDeviceDesc, len(allDevs))
	for index, devModel := range allDevs {
		dev := &IsolatedDeviceDesc{
			ID:             devModel.Id,
			GuestID:        devModel.GuestId,
			HostID:         devModel.HostId,
			DevType:        devModel.DevType,
			Model:          devModel.Model,
			Addr:           devModel.Addr,
			VendorDeviceID: devModel.VendorDeviceId,
		}
		devs[index] = dev
	}
	desc.IsolatedDevices = devs

	return nil
}

func (b *HostBuilder) fillCPUIOLoads(desc *HostDesc, host *computemodels.SHost) error {
	desc.CPULoad = b.loadByName(host.Id, "cpu_load")
	desc.IOLoad = b.loadByName(host.Id, "io_load")
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
