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

package usages

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"

	json "yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type sResourceMetric struct {
	Total     int64 `json:"total"`
	Used      int64 `json:"used"`
	Free      int64 `json:"free"`
	Allocated int64 `json:"allocated"`
	Virtual   int64 `json:"virtual,omitempty"`
	VmTotal   int64 `json:"vm_total,omitempty"`
}

type sUsageItem struct {
	Id        string  `json:"id"`
	Name      string  `json:"name"`
	Usage     float64 `json:"usage"`
	Count     int64   `json:"count"`
	Allocated int64   `json:"allocated"`
	Capacity  int64   `json:"capacity"`
}

// sResourceUsagePoint is per-dimension usage for a name (account/project).
// usage is used/total ratio in [0,1].
type sResourceUsagePoint struct {
	Name  string  `json:"name"`
	Usage float64 `json:"usage"`
	Used  int64   `json:"used"`
	Total int64   `json:"total"`
}

type sUsageByResource struct {
	CPU    []sResourceUsagePoint `json:"cpu"`
	Memory []sResourceUsagePoint `json:"memory"`
	Disk   []sResourceUsagePoint `json:"disk"`
	GPU    []sResourceUsagePoint `json:"gpu"`
}

type sCapacityBreakdownItem struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type sCapacityMetric struct {
	Total       int64                    `json:"total"`
	Allocated   int64                    `json:"allocated"`
	Unallocated int64                    `json:"unallocated"`
	Breakdown   []sCapacityBreakdownItem `json:"breakdown"`
}

type sCapacityGroup struct {
	CPU    sCapacityMetric `json:"cpu"`
	Memory sCapacityMetric `json:"memory"`
	Disk   sCapacityMetric `json:"disk"`
	GPU    sCapacityMetric `json:"gpu"`
}

type sGpuModelStat struct {
	Name       string           `json:"name"`
	Count      int64            `json:"count"`
	ByProvider map[string]int64 `json:"by_provider"`
}

type sDistributionStats struct {
	Physical           int64            `json:"physical"`
	VM                 int64            `json:"vm"`
	PhysicalByProvider map[string]int64 `json:"physical_by_provider"`
	VmByProvider       map[string]int64 `json:"vm_by_provider"`
	GpuVmByProvider    map[string]int64 `json:"gpu_vm_by_provider"`
	CPU                sResourceMetric  `json:"cpu"`
	Memory             sResourceMetric  `json:"memory"`
	Disk               sResourceMetric  `json:"disk"`
	GPU                sResourceMetric  `json:"gpu"`
}

type sResourceDistribution struct {
	Stats                  sDistributionStats `json:"stats"`
	ProviderUsage          []sUsageItem       `json:"provider_usage"`
	AccountUsage           []sUsageItem       `json:"account_usage"`
	ProjectUsage           []sUsageItem       `json:"project_usage"`
	AccountUsageByResource sUsageByResource   `json:"account_usage_by_resource"`
	ProjectUsageByResource sUsageByResource   `json:"project_usage_by_resource"`
	Capacity               struct {
		ByProvider sCapacityGroup `json:"by_provider"`
		ByAccount  sCapacityGroup `json:"by_account"`
		ByProject  sCapacityGroup `json:"by_project"`
	} `json:"capacity"`
	GpuModels []sGpuModelStat `json:"gpu_models"`
	Dim       string          `json:"dim"`
	Raw       struct {
		System Usage `json:"system"`
	} `json:"raw"`
}

type sDimKey struct {
	Id   string
	Name string
}

type sAggStat struct {
	Key           sDimKey
	HostCount     int64
	GuestCount    int64
	GpuGuestCount int64
	CPU           int64
	CPUTotal      int64
	CPUUsed       int64
	CPUVirtual    int64
	Memory        int64
	MemoryUsed    int64
	MemoryTotal   int64
	MemoryVirtual int64
	DiskCapacity  int64
	DiskAllocated int64
	GPUTotal      int64
	GPUUsed       int64
	GPUAllocated  int64
}

type sHostRow struct {
	Provider         string
	CloudaccountId   string
	CloudaccountName string
	MemSize          int
	MemoryUsedMb     int64
	PageSizeKB       int
	MemReserved      int
	MemCmtbound      float32
	CpuCount         int
	CpuUsagePercent  float64
	CpuReserved      int
	CpuCmtbound      float32
	StorageSize      int64
}

type sGuestAggRow struct {
	DimId   string
	DimName string
	Count   int64
	CPU     int64
	Memory  int64
	Disk    int64
	GpuVMs  int64
}

type sStorageAggRow struct {
	DimId    string
	DimName  string
	Capacity int64
}

type sGpuAggRow struct {
	DimId   string
	DimName string
	Model   string
	Total   int64
	Used    int64
}

var distributionCache = struct {
	sync.Mutex
	data map[string]*sResourceDistribution
}{data: make(map[string]*sResourceDistribution)}

func resourceDistributionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	ownerId, scope, err, result := db.FetchUsageOwnerScope(ctx, userCred, getQuery(r))
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	query := getQuery(r)
	tags := rbacutils.SPolicyResult{Result: rbacutils.Allow}
	_ = query.Unmarshal(&tags)
	result = result.Merge(tags)

	hostTypes := json.GetQueryStringArray(query, "host_type")
	providers := json.GetQueryStringArray(query, "provider")
	brands := json.GetQueryStringArray(query, "brand")
	cloudEnv, _ := query.GetString("cloud_env")
	includeSystem := json.QueryBoolean(query, "system", false)
	refresh := json.QueryBoolean(query, "refresh", false)
	dim, _ := query.GetString("dim")
	if dim == "" {
		dim = "cpu"
	}
	top := 0
	if topStr, _ := query.GetString("top"); len(topStr) > 0 {
		if v, e := strconv.Atoi(topStr); e == nil {
			top = v
		}
	}

	cacheKey := "resource-distribution:" + getCacheKey(scope, ownerId, false, nil, hostTypes, providers, brands, cloudEnv, includeSystem, result) + fmt.Sprintf("&dim=%s&top=%d", dim, top)
	if !refresh {
		distributionCache.Lock()
		cached := distributionCache.data[cacheKey]
		distributionCache.Unlock()
		if cached != nil {
			appsrv.SendStruct(w, cached)
			return
		}
	}

	ret, err := ReportResourceDistribution(ctx, userCred, scope, ownerId, hostTypes, providers, brands, cloudEnv, includeSystem, result, dim, top)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	distributionCache.Lock()
	distributionCache.data[cacheKey] = ret
	distributionCache.Unlock()
	appsrv.SendStruct(w, ret)
}

func ReportResourceDistribution(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	scope rbacscope.TRbacScope,
	ownerId mcclient.IIdentityProvider,
	hostTypes []string,
	providers []string,
	brands []string,
	cloudEnv string,
	includeSystem bool,
	policyResult rbacutils.SPolicyResult,
	dim string,
	top int,
) (*sResourceDistribution, error) {
	systemUsage, err := ReportGeneralUsage(ctx, userCred, scope, ownerId, false, nil, hostTypes, providers, brands, cloudEnv, includeSystem, policyResult)
	if err != nil {
		return nil, errors.Wrap(err, "ReportGeneralUsage")
	}

	hostPolicy := db.UsagePolicyCheck(userCred, models.HostManager, scope).Merge(policyResult)
	guestPolicy := db.UsagePolicyCheck(userCred, models.GuestManager, scope).Merge(policyResult)
	storagePolicy := db.UsagePolicyCheck(userCred, models.StorageManager, scope).Merge(policyResult)
	devicePolicy := db.UsagePolicyCheck(userCred, models.IsolatedDeviceManager, scope).Merge(policyResult)

	resourceTypes := []string{api.HostResourceTypeShared}

	var (
		hostByProvider, hostByAccount                   map[string]*sAggStat
		guestByProvider, guestByAccount, guestByProject map[string]*sAggStat
		storageByProvider, storageByAccount             map[string]*sAggStat
		gpuByProvider, gpuByAccount                     map[string]*sAggStat
		gpuModels                                       []sGpuModelStat
	)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	setErr := func(e error) {
		if e == nil {
			return
		}
		mu.Lock()
		if firstErr == nil {
			firstErr = e
		}
		mu.Unlock()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		hp, ha, e := aggregateHostsByDims(ctx, ownerId, scope, hostTypes, resourceTypes, providers, brands, cloudEnv, hostPolicy)
		setErr(e)
		mu.Lock()
		hostByProvider, hostByAccount = hp, ha
		mu.Unlock()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		gp, ga, gj, e := aggregateGuestsByDims(ctx, ownerId, scope, hostTypes, resourceTypes, providers, brands, cloudEnv, includeSystem, guestPolicy)
		setErr(e)
		mu.Lock()
		guestByProvider, guestByAccount, guestByProject = gp, ga, gj
		mu.Unlock()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sp, sa, e := aggregateStoragesByDims(ctx, ownerId, scope, hostTypes, resourceTypes, providers, brands, cloudEnv, storagePolicy)
		setErr(e)
		mu.Lock()
		storageByProvider, storageByAccount = sp, sa
		mu.Unlock()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		gp, ga, models, e := aggregateGpusByDims(ctx, ownerId, scope, hostTypes, resourceTypes, providers, brands, cloudEnv, devicePolicy)
		setErr(e)
		mu.Lock()
		gpuByProvider, gpuByAccount = gp, ga
		gpuModels = models
		mu.Unlock()
	}()

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	ret := &sResourceDistribution{Dim: dim}
	ret.Raw.System = pickUsageKeys(systemUsage,
		"hosts", "hosts.cpu", "hosts.cpu.used", "hosts.cpu.virtual",
		"hosts.memory", "hosts.memory.used", "hosts.memory.virtual",
		"enabled_hosts", "enabled_hosts.cpu", "enabled_hosts.memory",
		"all.servers", "all.servers.cpu", "all.servers.memory", "all.servers.disk",
		"storages", "storages.virtual", "all.disks", "disks",
		"isolated_devices.gpu", "isolated_devices.gpu.used",
	)
	ret.GpuModels = gpuModels

	ret.Stats = buildDistributionStats(systemUsage, hostByProvider, guestByProvider)
	accountMerged := mergeAggClean(hostByAccount, guestByAccount, storageByAccount, gpuByAccount)
	projectMerged := mergeAggClean(nil, guestByProject, nil, nil)
	ret.ProviderUsage = buildUsageItems(mergeAggClean(hostByProvider, guestByProvider, storageByProvider, gpuByProvider), dim, false, usageInt64(systemUsage, "hosts.cpu.virtual"), top)
	ret.AccountUsage = buildUsageItems(accountMerged, dim, false, usageInt64(systemUsage, "hosts.cpu.virtual"), top)
	ret.ProjectUsage = buildUsageItems(projectMerged, dim, true, dimCapacity(systemUsage, dim), top)
	ret.AccountUsageByResource = buildUsageByResource(accountMerged, false, systemUsage, top)
	ret.ProjectUsageByResource = buildUsageByResource(projectMerged, true, systemUsage, top)

	ret.Capacity.ByProvider = buildCapacityGroup(mergeAggClean(hostByProvider, guestByProvider, storageByProvider, gpuByProvider), false)
	ret.Capacity.ByAccount = buildCapacityGroup(accountMerged, false)
	ret.Capacity.ByProject = buildCapacityGroup(projectMerged, true)
	applyGlobalCapacityTotals(&ret.Capacity.ByProject, systemUsage)

	return ret, nil
}

func buildDistributionStats(u Usage, hostByProvider, guestByProvider map[string]*sAggStat) sDistributionStats {
	stats := sDistributionStats{
		Physical:           usageInt64(u, "hosts"),
		VM:                 usageInt64(u, "all.servers"),
		PhysicalByProvider: map[string]int64{},
		VmByProvider:       map[string]int64{},
		GpuVmByProvider:    map[string]int64{},
	}
	for name, st := range hostByProvider {
		stats.PhysicalByProvider[name] = st.HostCount
	}
	stats.CPU = sResourceMetric{
		Total:     usageInt64(u, "hosts.cpu"),
		Used:      usageInt64(u, "hosts.cpu.used"),
		Allocated: usageInt64(u, "all.servers.cpu"),
		Virtual:   usageInt64(u, "hosts.cpu.virtual"),
	}
	stats.CPU.Free = stats.CPU.Total - stats.CPU.Used
	if stats.CPU.Free < 0 {
		stats.CPU.Free = 0
	}
	stats.Memory = sResourceMetric{
		Total:     usageInt64(u, "hosts.memory"),
		Used:      usageInt64(u, "hosts.memory.used"),
		Allocated: usageInt64(u, "all.servers.memory"),
		Virtual:   usageInt64(u, "hosts.memory.virtual"),
	}
	stats.Memory.Free = stats.Memory.Total - stats.Memory.Used
	if stats.Memory.Free < 0 {
		stats.Memory.Free = 0
	}
	diskUsed := usageInt64(u, "all.disks")
	if diskUsed == 0 {
		diskUsed = usageInt64(u, "disks")
	}
	stats.Disk = sResourceMetric{
		Total:     usageInt64(u, "storages"),
		Used:      diskUsed,
		Allocated: usageInt64(u, "all.servers.disk"),
	}
	stats.Disk.Free = stats.Disk.Total - stats.Disk.Used
	if stats.Disk.Free < 0 {
		stats.Disk.Free = 0
	}
	stats.GPU = sResourceMetric{
		Total:   usageInt64(u, "isolated_devices.gpu"),
		Used:    usageInt64(u, "isolated_devices.gpu.used"),
		VmTotal: 0,
	}
	stats.GPU.Free = stats.GPU.Total - stats.GPU.Used
	if stats.GPU.Free < 0 {
		stats.GPU.Free = 0
	}

	for name, st := range guestByProvider {
		stats.VmByProvider[name] = st.GuestCount
		stats.GpuVmByProvider[name] = st.GpuGuestCount
		stats.GPU.VmTotal += st.GpuGuestCount
	}
	return stats
}

func applyGlobalCapacityTotals(group *sCapacityGroup, u Usage) {
	group.CPU.Total = usageInt64(u, "hosts.cpu.virtual")
	group.Memory.Total = usageInt64(u, "hosts.memory.virtual")
	group.Disk.Total = usageInt64(u, "storages")
	group.GPU.Total = usageInt64(u, "isolated_devices.gpu")
	group.CPU.Unallocated = maxInt64(0, group.CPU.Total-group.CPU.Allocated)
	group.Memory.Unallocated = maxInt64(0, group.Memory.Total-group.Memory.Allocated)
	group.Disk.Unallocated = maxInt64(0, group.Disk.Total-group.Disk.Allocated)
	group.GPU.Unallocated = maxInt64(0, group.GPU.Total-group.GPU.Allocated)
}

func dimCapacity(u Usage, dim string) int64 {
	switch dim {
	case "memory":
		return usageInt64(u, "hosts.memory.virtual")
	case "disk":
		return usageInt64(u, "storages")
	case "gpu":
		return usageInt64(u, "isolated_devices.gpu")
	default:
		return usageInt64(u, "hosts.cpu.virtual")
	}
}

func mergeAggClean(hosts, guests, storages, gpus map[string]*sAggStat) map[string]*sAggStat {
	out := map[string]*sAggStat{}
	put := func(k string, key sDimKey) *sAggStat {
		st, ok := out[k]
		if !ok {
			st = &sAggStat{Key: key}
			out[k] = st
		}
		if (st.Key.Name == "" || st.Key.Name == st.Key.Id) && key.Name != "" {
			st.Key = key
		}
		return st
	}
	for k, v := range hosts {
		st := put(k, v.Key)
		st.HostCount += v.HostCount
		st.CPU += v.CPU
		st.CPUUsed += v.CPUUsed
		st.CPUVirtual += v.CPUVirtual
		st.Memory += v.Memory
		st.MemoryUsed += v.MemoryUsed
		st.MemoryVirtual += v.MemoryVirtual
	}
	for k, v := range guests {
		st := put(k, v.Key)
		st.GuestCount += v.GuestCount
		st.GpuGuestCount += v.GpuGuestCount
		st.CPUTotal += v.CPUTotal       // guest vcpu allocated
		st.MemoryTotal += v.MemoryTotal // guest vmem allocated
		st.DiskAllocated += v.DiskAllocated
		st.GPUAllocated += v.GPUAllocated
	}
	for k, v := range storages {
		st := put(k, v.Key)
		st.DiskCapacity += v.DiskCapacity
	}
	for k, v := range gpus {
		st := put(k, v.Key)
		st.GPUTotal += v.GPUTotal
		st.GPUUsed += v.GPUUsed
	}
	return out
}

func buildUsageByResource(stats map[string]*sAggStat, projectMode bool, systemUsage Usage, top int) sUsageByResource {
	globalCPU := usageInt64(systemUsage, "hosts.cpu.virtual")
	globalMem := usageInt64(systemUsage, "hosts.memory.virtual")
	globalDisk := usageInt64(systemUsage, "storages")
	globalGPU := usageInt64(systemUsage, "isolated_devices.gpu")

	out := sUsageByResource{
		CPU:    make([]sResourceUsagePoint, 0, len(stats)),
		Memory: make([]sResourceUsagePoint, 0, len(stats)),
		Disk:   make([]sResourceUsagePoint, 0, len(stats)),
		GPU:    make([]sResourceUsagePoint, 0, len(stats)),
	}
	for _, st := range stats {
		name := preferName(st.Key)
		cpuUsed, cpuTotal := st.CPUTotal, st.CPUVirtual
		memUsed, memTotal := st.MemoryTotal, st.MemoryVirtual
		diskUsed, diskTotal := st.DiskAllocated, st.DiskCapacity
		gpuUsed := st.GPUAllocated
		if gpuUsed == 0 {
			gpuUsed = st.GpuGuestCount
		}
		gpuTotal := st.GPUTotal
		if projectMode || cpuTotal == 0 {
			cpuTotal = globalCPU
		}
		if projectMode || memTotal == 0 {
			memTotal = globalMem
		}
		if projectMode || diskTotal == 0 {
			diskTotal = globalDisk
		}
		if projectMode || gpuTotal == 0 {
			gpuTotal = globalGPU
		}
		out.CPU = append(out.CPU, makeResourceUsagePoint(name, cpuUsed, cpuTotal))
		out.Memory = append(out.Memory, makeResourceUsagePoint(name, memUsed, memTotal))
		out.Disk = append(out.Disk, makeResourceUsagePoint(name, diskUsed, diskTotal))
		out.GPU = append(out.GPU, makeResourceUsagePoint(name, gpuUsed, gpuTotal))
	}
	sortResourceUsagePoints := func(items []sResourceUsagePoint) []sResourceUsagePoint {
		sort.Slice(items, func(i, j int) bool {
			if items[i].Used == items[j].Used {
				return items[i].Usage > items[j].Usage
			}
			return items[i].Used > items[j].Used
		})
		if top > 0 && len(items) > top {
			items = items[:top]
		}
		return items
	}
	out.CPU = sortResourceUsagePoints(out.CPU)
	out.Memory = sortResourceUsagePoints(out.Memory)
	out.Disk = sortResourceUsagePoints(out.Disk)
	out.GPU = sortResourceUsagePoints(out.GPU)
	return out
}

func makeResourceUsagePoint(name string, used, total int64) sResourceUsagePoint {
	usage := 0.0
	if total > 0 {
		usage = float64(used) / float64(total)
	}
	return sResourceUsagePoint{
		Name:  name,
		Usage: utils.FloatRound(usage, 4),
		Used:  used,
		Total: total,
	}
}

func buildUsageItems(stats map[string]*sAggStat, dim string, projectMode bool, globalCap int64, top int) []sUsageItem {
	items := make([]sUsageItem, 0, len(stats))
	for _, st := range stats {
		var allocated, capacity int64
		switch dim {
		case "memory":
			allocated = st.MemoryTotal
			capacity = st.MemoryVirtual
			if projectMode || capacity == 0 {
				capacity = globalCap
			}
		case "disk":
			allocated = st.DiskAllocated
			capacity = st.DiskCapacity
			if projectMode || capacity == 0 {
				capacity = globalCap
			}
		case "gpu":
			allocated = st.GPUAllocated
			if allocated == 0 {
				allocated = st.GpuGuestCount
			}
			capacity = st.GPUTotal
			if projectMode || capacity == 0 {
				capacity = globalCap
			}
		default: // cpu
			allocated = st.CPUTotal
			capacity = st.CPUVirtual
			if projectMode || capacity == 0 {
				capacity = globalCap
			}
		}
		usage := 0.0
		if capacity > 0 {
			usage = float64(allocated) * 100.0 / float64(capacity)
		}
		items = append(items, sUsageItem{
			Id:        st.Key.Id,
			Name:      preferName(st.Key),
			Usage:     utils.FloatRound(usage, 1),
			Count:     st.GuestCount,
			Allocated: allocated,
			Capacity:  capacity,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Allocated == items[j].Allocated {
			return items[i].Usage > items[j].Usage
		}
		return items[i].Allocated > items[j].Allocated
	})
	if top > 0 && len(items) > top {
		items = items[:top]
	}
	return items
}

func buildCapacityGroup(stats map[string]*sAggStat, projectMode bool) sCapacityGroup {
	group := sCapacityGroup{
		CPU:    sCapacityMetric{Breakdown: []sCapacityBreakdownItem{}},
		Memory: sCapacityMetric{Breakdown: []sCapacityBreakdownItem{}},
		Disk:   sCapacityMetric{Breakdown: []sCapacityBreakdownItem{}},
		GPU:    sCapacityMetric{Breakdown: []sCapacityBreakdownItem{}},
	}
	for _, st := range stats {
		name := preferName(st.Key)
		guestCPU := st.CPUTotal
		guestMem := st.MemoryTotal
		guestDisk := st.DiskAllocated
		guestGPU := st.GPUAllocated
		if guestGPU == 0 {
			guestGPU = st.GpuGuestCount
		}
		if !projectMode {
			group.CPU.Total += st.CPUVirtual
			group.Memory.Total += st.MemoryVirtual
			group.Disk.Total += st.DiskCapacity
			group.GPU.Total += st.GPUTotal
		}
		group.CPU.Allocated += guestCPU
		group.Memory.Allocated += guestMem
		group.Disk.Allocated += guestDisk
		group.GPU.Allocated += guestGPU
		if guestCPU > 0 {
			group.CPU.Breakdown = append(group.CPU.Breakdown, sCapacityBreakdownItem{Name: name, Value: guestCPU})
		}
		if guestMem > 0 {
			group.Memory.Breakdown = append(group.Memory.Breakdown, sCapacityBreakdownItem{Name: name, Value: guestMem})
		}
		if guestDisk > 0 {
			group.Disk.Breakdown = append(group.Disk.Breakdown, sCapacityBreakdownItem{Name: name, Value: guestDisk})
		}
		if guestGPU > 0 {
			group.GPU.Breakdown = append(group.GPU.Breakdown, sCapacityBreakdownItem{Name: name, Value: guestGPU})
		}
	}
	sortBreakdown := func(items []sCapacityBreakdownItem) []sCapacityBreakdownItem {
		sort.Slice(items, func(i, j int) bool { return items[i].Value > items[j].Value })
		return items
	}
	group.CPU.Breakdown = sortBreakdown(group.CPU.Breakdown)
	group.Memory.Breakdown = sortBreakdown(group.Memory.Breakdown)
	group.Disk.Breakdown = sortBreakdown(group.Disk.Breakdown)
	group.GPU.Breakdown = sortBreakdown(group.GPU.Breakdown)
	if !projectMode {
		group.CPU.Unallocated = maxInt64(0, group.CPU.Total-group.CPU.Allocated)
		group.Memory.Unallocated = maxInt64(0, group.Memory.Total-group.Memory.Allocated)
		group.Disk.Unallocated = maxInt64(0, group.Disk.Total-group.Disk.Allocated)
		group.GPU.Unallocated = maxInt64(0, group.GPU.Total-group.GPU.Allocated)
	}
	return group
}

func preferName(k sDimKey) string {
	if k.Name != "" {
		return k.Name
	}
	if k.Id != "" {
		return k.Id
	}
	return api.CLOUD_PROVIDER_ONECLOUD
}

func aggregateHostsByDims(
	ctx context.Context,
	ownerId mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	policyResult rbacutils.SPolicyResult,
) (map[string]*sAggStat, map[string]*sAggStat, error) {
	rows, err := fetchHostRows(ctx, ownerId, scope, hostTypes, resourceTypes, providers, brands, cloudEnv, policyResult)
	if err != nil {
		return nil, nil, err
	}
	byProvider := map[string]*sAggStat{}
	byAccount := map[string]*sAggStat{}
	for _, row := range rows {
		provider := row.Provider
		if provider == "" {
			provider = api.CLOUD_PROVIDER_ONECLOUD
		}
		accountId := row.CloudaccountId
		accountName := row.CloudaccountName
		if accountId == "" {
			accountId = api.CLOUD_PROVIDER_ONECLOUD
			accountName = api.CLOUD_PROVIDER_ONECLOUD
		}
		addHostRow(byProvider, provider, provider, row)
		addHostRow(byAccount, accountId, accountName, row)
	}
	return byProvider, byAccount, nil
}

func addHostRow(dst map[string]*sAggStat, id, name string, row sHostRow) {
	st, ok := dst[id]
	if !ok {
		st = &sAggStat{Key: sDimKey{Id: id, Name: name}}
		dst[id] = st
	}
	if row.MemSize == 0 {
		return
	}
	st.HostCount++
	cpuUsable := usableSize(row.CpuCount, row.CpuReserved)
	memUsable := usableSize(row.MemSize, row.MemReserved)
	st.CPU += int64(cpuUsable)
	st.CPUTotal += int64(row.CpuCount)
	st.CPUUsed += int64(row.CpuUsagePercent * float64(row.CpuCount) / 100)
	st.Memory += int64(memUsable)
	st.MemoryTotal += int64(row.MemSize)
	st.MemoryUsed += row.MemoryUsedMb
	cpuCmt := row.CpuCmtbound
	if cpuCmt <= 0 {
		cpuCmt = options.Options.DefaultCPUOvercommitBound
	}
	memCmt := row.MemCmtbound
	if row.PageSizeKB > 4 {
		memCmt = 1.0
	} else if memCmt <= 0 {
		memCmt = options.Options.DefaultMemoryOvercommitBound
	}
	st.CPUVirtual += int64(float64(float32(cpuUsable) * cpuCmt))
	st.MemoryVirtual += int64(float64(float32(memUsable) * memCmt))
}

func usableSize(act, reserved int) int {
	if reserved > 0 && reserved < act {
		return act - reserved
	}
	return act
}

func fetchHostRows(
	ctx context.Context,
	ownerId mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	policyResult rbacutils.SPolicyResult,
) ([]sHostRow, error) {
	hosts := models.HostManager.Query().SubQuery()
	cps := models.CloudproviderManager.Query().SubQuery()
	accounts := models.CloudaccountManager.Query().SubQuery()

	q := hosts.Query(
		hosts.Field("mem_size"),
		hosts.Field("memory_used_mb"),
		hosts.Field("page_size_kb"),
		hosts.Field("mem_reserved"),
		hosts.Field("mem_cmtbound"),
		hosts.Field("cpu_count"),
		hosts.Field("cpu_usage_percent"),
		hosts.Field("cpu_reserved"),
		hosts.Field("cpu_cmtbound"),
		hosts.Field("storage_size"),
		cps.Field("provider"),
		accounts.Field("id").Label("cloudaccount_id"),
		accounts.Field("name").Label("cloudaccount_name"),
	)
	q = q.LeftJoin(cps, sqlchemy.Equals(hosts.Field("manager_id"), cps.Field("id")))
	q = q.LeftJoin(accounts, sqlchemy.Equals(cps.Field("cloudaccount_id"), accounts.Field("id")))
	if scope != rbacscope.ScopeSystem && ownerId != nil {
		q = q.Filter(sqlchemy.Equals(hosts.Field("domain_id"), ownerId.GetProjectDomainId()))
	}
	q = q.Filter(sqlchemy.OR(
		sqlchemy.IsFalse(hosts.Field("is_baremetal")),
		sqlchemy.NotEquals(hosts.Field("host_type"), api.HOST_TYPE_BAREMETAL),
	))
	q = db.ObjectIdQueryWithPolicyResult(ctx, q, models.HostManager, policyResult)
	q = models.AttachUsageQuery(q, hosts, hostTypes, resourceTypes, providers, brands, cloudEnv, nil)

	rows := make([]sHostRow, 0)
	err := q.All(&rows)
	if err != nil {
		return nil, errors.Wrap(err, "fetchHostRows")
	}
	return rows, nil
}

func aggregateGuestsByDims(
	ctx context.Context,
	ownerId mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	includeSystem bool,
	policyResult rbacutils.SPolicyResult,
) (map[string]*sAggStat, map[string]*sAggStat, map[string]*sAggStat, error) {
	byProvider, err := fetchGuestAgg(ctx, ownerId, scope, hostTypes, resourceTypes, providers, brands, cloudEnv, includeSystem, policyResult, "provider")
	if err != nil {
		return nil, nil, nil, err
	}
	byAccount, err := fetchGuestAgg(ctx, ownerId, scope, hostTypes, resourceTypes, providers, brands, cloudEnv, includeSystem, policyResult, "account")
	if err != nil {
		return nil, nil, nil, err
	}
	byProject, err := fetchGuestAgg(ctx, ownerId, scope, hostTypes, resourceTypes, providers, brands, cloudEnv, includeSystem, policyResult, "project")
	if err != nil {
		return nil, nil, nil, err
	}
	return guestRowsToAgg(byProvider), guestRowsToAgg(byAccount), guestRowsToAgg(byProject), nil
}

func guestRowsToAgg(rows []sGuestAggRow) map[string]*sAggStat {
	out := map[string]*sAggStat{}
	for _, row := range rows {
		id := row.DimId
		name := row.DimName
		if id == "" {
			id = api.CLOUD_PROVIDER_ONECLOUD
			name = api.CLOUD_PROVIDER_ONECLOUD
		}
		if name == "" {
			name = id
		}
		out[id] = &sAggStat{
			Key:           sDimKey{Id: id, Name: name},
			GuestCount:    row.Count,
			GpuGuestCount: row.GpuVMs,
			CPUTotal:      row.CPU,
			MemoryTotal:   row.Memory,
			DiskAllocated: row.Disk,
			GPUAllocated:  row.GpuVMs,
		}
	}
	return out
}

func fetchGuestAgg(
	ctx context.Context,
	ownerId mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	includeSystem bool,
	policyResult rbacutils.SPolicyResult,
	dim string,
) ([]sGuestAggRow, error) {
	hypervisors := sets.NewString(api.HYPERVISORS...)
	hypervisors.Delete(api.HYPERVISOR_POD, api.HYPERVISOR_BAREMETAL)

	gq := models.GuestManager.Query()
	gq = gq.Filter(sqlchemy.OR(sqlchemy.IsNull(gq.Field("pending_deleted")), sqlchemy.IsFalse(gq.Field("pending_deleted"))))
	if !includeSystem {
		gq = gq.Filter(sqlchemy.OR(sqlchemy.IsNull(gq.Field("is_system")), sqlchemy.IsFalse(gq.Field("is_system"))))
	}
	gq = gq.In("hypervisor", hypervisors.List())
	switch scope {
	case rbacscope.ScopeDomain:
		if ownerId != nil {
			gq = gq.Equals("domain_id", ownerId.GetProjectDomainId())
		}
	case rbacscope.ScopeProject:
		if ownerId != nil {
			gq = gq.Equals("tenant_id", ownerId.GetProjectId())
		}
	}
	gq = db.ObjectIdQueryWithPolicyResult(ctx, gq, models.GuestManager, policyResult)
	if len(hostTypes) > 0 || len(resourceTypes) > 0 || len(providers) > 0 || len(brands) > 0 || len(cloudEnv) > 0 {
		hosts := models.HostManager.Query().SubQuery()
		subq := hosts.Query(hosts.Field("id"))
		subq = models.AttachUsageQuery(subq, hosts, hostTypes, resourceTypes, providers, brands, cloudEnv, nil)
		gq = gq.In("host_id", subq.SubQuery())
	}
	guests := gq.SubQuery()

	guestdisks := models.GuestdiskManager.Query().SubQuery()
	disks := models.DiskManager.Query().SubQuery()
	diskQ := guestdisks.Query(
		guestdisks.Field("guest_id"),
		sqlchemy.SUM("guest_disk_size", disks.Field("disk_size")),
	).Join(disks, sqlchemy.Equals(guestdisks.Field("disk_id"), disks.Field("id"))).GroupBy(guestdisks.Field("guest_id")).SubQuery()

	guestIdevs := models.GuestIsolatedDeviceManager.Query().SubQuery()
	idevs := models.IsolatedDeviceManager.Query().Equals("dev_type", api.GPU_TYPE).SubQuery()
	gpuQ := guestIdevs.Query(
		guestIdevs.Field("guest_id"),
		sqlchemy.COUNT("gpu_cnt"),
	).Join(idevs, sqlchemy.Equals(guestIdevs.Field("isolated_device_id"), idevs.Field("id"))).GroupBy(guestIdevs.Field("guest_id")).SubQuery()

	hosts := models.HostManager.Query().SubQuery()
	cps := models.CloudproviderManager.Query().SubQuery()
	accounts := models.CloudaccountManager.Query().SubQuery()

	var q *sqlchemy.SQuery
	switch dim {
	case "account":
		q = guests.Query(
			accounts.Field("id").Label("dim_id"),
			accounts.Field("name").Label("dim_name"),
			sqlchemy.COUNT("count", guests.Field("id")),
			sqlchemy.SUM("cpu", guests.Field("vcpu_count")),
			sqlchemy.SUM("memory", guests.Field("vmem_size")),
			sqlchemy.SUM("disk", diskQ.Field("guest_disk_size")),
			sqlchemy.COUNT("gpu_vms", gpuQ.Field("guest_id")),
		)
		q = q.Join(hosts, sqlchemy.Equals(guests.Field("host_id"), hosts.Field("id")))
		q = q.LeftJoin(cps, sqlchemy.Equals(hosts.Field("manager_id"), cps.Field("id")))
		q = q.LeftJoin(accounts, sqlchemy.Equals(cps.Field("cloudaccount_id"), accounts.Field("id")))
		q = q.LeftJoin(diskQ, sqlchemy.Equals(guests.Field("id"), diskQ.Field("guest_id")))
		q = q.LeftJoin(gpuQ, sqlchemy.Equals(guests.Field("id"), gpuQ.Field("guest_id")))
		q = q.GroupBy(accounts.Field("id"), accounts.Field("name"))
	case "project":
		q = guests.Query(
			guests.Field("tenant_id").Label("dim_id"),
			sqlchemy.COUNT("count", guests.Field("id")),
			sqlchemy.SUM("cpu", guests.Field("vcpu_count")),
			sqlchemy.SUM("memory", guests.Field("vmem_size")),
			sqlchemy.SUM("disk", diskQ.Field("guest_disk_size")),
			sqlchemy.COUNT("gpu_vms", gpuQ.Field("guest_id")),
		)
		q = q.LeftJoin(diskQ, sqlchemy.Equals(guests.Field("id"), diskQ.Field("guest_id")))
		q = q.LeftJoin(gpuQ, sqlchemy.Equals(guests.Field("id"), gpuQ.Field("guest_id")))
		q = q.GroupBy(guests.Field("tenant_id"))
	default: // provider
		q = guests.Query(
			cps.Field("provider").Label("dim_id"),
			sqlchemy.COUNT("count", guests.Field("id")),
			sqlchemy.SUM("cpu", guests.Field("vcpu_count")),
			sqlchemy.SUM("memory", guests.Field("vmem_size")),
			sqlchemy.SUM("disk", diskQ.Field("guest_disk_size")),
			sqlchemy.COUNT("gpu_vms", gpuQ.Field("guest_id")),
		)
		q = q.Join(hosts, sqlchemy.Equals(guests.Field("host_id"), hosts.Field("id")))
		q = q.LeftJoin(cps, sqlchemy.Equals(hosts.Field("manager_id"), cps.Field("id")))
		q = q.LeftJoin(diskQ, sqlchemy.Equals(guests.Field("id"), diskQ.Field("guest_id")))
		q = q.LeftJoin(gpuQ, sqlchemy.Equals(guests.Field("id"), gpuQ.Field("guest_id")))
		q = q.GroupBy(cps.Field("provider"))
	}

	rows := make([]sGuestAggRow, 0)
	err := q.All(&rows)
	if err != nil {
		return nil, errors.Wrapf(err, "fetchGuestAgg %s", dim)
	}
	if dim == "project" {
		for i := range rows {
			rows[i].DimName = fetchProjectName(ctx, rows[i].DimId)
		}
	} else if dim == "provider" {
		for i := range rows {
			if rows[i].DimId == "" {
				rows[i].DimId = api.CLOUD_PROVIDER_ONECLOUD
			}
			rows[i].DimName = rows[i].DimId
		}
	}
	return rows, nil
}

func fetchProjectName(ctx context.Context, id string) string {
	if id == "" {
		return ""
	}
	tenant, err := db.TenantCacheManager.FetchTenantById(ctx, id)
	if err != nil || tenant == nil {
		return id
	}
	return tenant.GetName()
}

func aggregateStoragesByDims(
	ctx context.Context,
	ownerId mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	policyResult rbacutils.SPolicyResult,
) (map[string]*sAggStat, map[string]*sAggStat, error) {
	byProvider, err := fetchStorageAgg(ctx, ownerId, scope, hostTypes, resourceTypes, providers, brands, cloudEnv, policyResult, "provider")
	if err != nil {
		return nil, nil, err
	}
	byAccount, err := fetchStorageAgg(ctx, ownerId, scope, hostTypes, resourceTypes, providers, brands, cloudEnv, policyResult, "account")
	if err != nil {
		return nil, nil, err
	}
	return storageRowsToAgg(byProvider), storageRowsToAgg(byAccount), nil
}

func storageRowsToAgg(rows []sStorageAggRow) map[string]*sAggStat {
	out := map[string]*sAggStat{}
	for _, row := range rows {
		id := row.DimId
		name := row.DimName
		if id == "" {
			id = api.CLOUD_PROVIDER_ONECLOUD
			name = api.CLOUD_PROVIDER_ONECLOUD
		}
		if name == "" {
			name = id
		}
		out[id] = &sAggStat{
			Key:          sDimKey{Id: id, Name: name},
			DiskCapacity: row.Capacity,
		}
	}
	return out
}

func fetchStorageAgg(
	ctx context.Context,
	ownerId mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	policyResult rbacutils.SPolicyResult,
	dim string,
) ([]sStorageAggRow, error) {
	storages := models.StorageManager.Query().IsTrue("enabled").SubQuery()
	cps := models.CloudproviderManager.Query().SubQuery()
	accounts := models.CloudaccountManager.Query().SubQuery()

	// usable capacity ~= capacity - reserved (same idea as usage)
	capaExpr := sqlchemy.SUB("", storages.Field("capacity"), storages.Field("reserved"))

	var q *sqlchemy.SQuery
	if dim == "account" {
		q = storages.Query(
			accounts.Field("id").Label("dim_id"),
			accounts.Field("name").Label("dim_name"),
			sqlchemy.SUM("capacity", capaExpr),
		)
		q = q.LeftJoin(cps, sqlchemy.Equals(storages.Field("manager_id"), cps.Field("id")))
		q = q.LeftJoin(accounts, sqlchemy.Equals(cps.Field("cloudaccount_id"), accounts.Field("id")))
		q = q.GroupBy(accounts.Field("id"), accounts.Field("name"))
	} else {
		q = storages.Query(
			cps.Field("provider").Label("dim_id"),
			sqlchemy.SUM("capacity", capaExpr),
		)
		q = q.LeftJoin(cps, sqlchemy.Equals(storages.Field("manager_id"), cps.Field("id")))
		q = q.GroupBy(cps.Field("provider"))
	}
	if scope != rbacscope.ScopeSystem && ownerId != nil {
		q = q.Filter(sqlchemy.Equals(storages.Field("domain_id"), ownerId.GetProjectDomainId()))
	}
	q = db.ObjectIdQueryWithPolicyResult(ctx, q, models.StorageManager, policyResult)
	q = models.CloudProviderFilter(q, storages.Field("manager_id"), providers, brands, cloudEnv)
	_ = hostTypes
	_ = resourceTypes

	rows := make([]sStorageAggRow, 0)
	err := q.All(&rows)
	if err != nil {
		return nil, errors.Wrapf(err, "fetchStorageAgg %s", dim)
	}
	if dim == "provider" {
		for i := range rows {
			if rows[i].DimId == "" {
				rows[i].DimId = api.CLOUD_PROVIDER_ONECLOUD
			}
			rows[i].DimName = rows[i].DimId
		}
	}
	return rows, nil
}

func aggregateGpusByDims(
	ctx context.Context,
	ownerId mcclient.IIdentityProvider,
	scope rbacscope.TRbacScope,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	policyResult rbacutils.SPolicyResult,
) (map[string]*sAggStat, map[string]*sAggStat, []sGpuModelStat, error) {
	devs := models.IsolatedDeviceManager.Query().Equals("dev_type", api.GPU_TYPE).SubQuery()
	hosts := models.HostManager.Query().SubQuery()
	cps := models.CloudproviderManager.Query().SubQuery()
	accounts := models.CloudaccountManager.Query().SubQuery()
	guestIdevs := models.GuestIsolatedDeviceManager.Query().SubQuery()

	base := devs.Query(
		devs.Field("id"),
		devs.Field("model"),
		devs.Field("host_id"),
		guestIdevs.Field("guest_id"),
		cps.Field("provider"),
		accounts.Field("id").Label("cloudaccount_id"),
		accounts.Field("name").Label("cloudaccount_name"),
	)
	base = base.Join(hosts, sqlchemy.Equals(devs.Field("host_id"), hosts.Field("id")))
	base = base.LeftJoin(cps, sqlchemy.Equals(hosts.Field("manager_id"), cps.Field("id")))
	base = base.LeftJoin(accounts, sqlchemy.Equals(cps.Field("cloudaccount_id"), accounts.Field("id")))
	base = base.LeftJoin(guestIdevs, sqlchemy.Equals(devs.Field("id"), guestIdevs.Field("isolated_device_id")))
	if scope != rbacscope.ScopeSystem && ownerId != nil {
		base = base.Filter(sqlchemy.Equals(hosts.Field("domain_id"), ownerId.GetProjectDomainId()))
	}
	base = db.ObjectIdQueryWithPolicyResult(ctx, base, models.IsolatedDeviceManager, policyResult)
	base = models.AttachUsageQuery(base, hosts, hostTypes, resourceTypes, providers, brands, cloudEnv, nil)

	type gpuRow struct {
		Id               string
		Model            string
		HostId           string
		GuestId          string
		Provider         string
		CloudaccountId   string
		CloudaccountName string
	}
	rows := make([]gpuRow, 0)
	if err := base.All(&rows); err != nil {
		return nil, nil, nil, errors.Wrap(err, "aggregateGpusByDims")
	}

	byProvider := map[string]*sAggStat{}
	byAccount := map[string]*sAggStat{}
	modelMap := map[string]*sGpuModelStat{}

	inc := func(dst map[string]*sAggStat, id, name string, used bool) {
		st, ok := dst[id]
		if !ok {
			st = &sAggStat{Key: sDimKey{Id: id, Name: name}}
			dst[id] = st
		}
		st.GPUTotal++
		if used {
			st.GPUUsed++
		}
	}

	for _, row := range rows {
		provider := row.Provider
		if provider == "" {
			provider = api.CLOUD_PROVIDER_ONECLOUD
		}
		accountId := row.CloudaccountId
		accountName := row.CloudaccountName
		if accountId == "" {
			accountId = api.CLOUD_PROVIDER_ONECLOUD
			accountName = api.CLOUD_PROVIDER_ONECLOUD
		}
		used := row.GuestId != ""
		inc(byProvider, provider, provider, used)
		inc(byAccount, accountId, accountName, used)

		model := row.Model
		if model == "" {
			model = "unknown"
		}
		ms, ok := modelMap[model]
		if !ok {
			ms = &sGpuModelStat{Name: model, ByProvider: map[string]int64{}}
			modelMap[model] = ms
		}
		ms.Count++
		ms.ByProvider[provider]++
	}

	modelsOut := make([]sGpuModelStat, 0, len(modelMap))
	for _, v := range modelMap {
		modelsOut = append(modelsOut, *v)
	}
	sort.Slice(modelsOut, func(i, j int) bool { return modelsOut[i].Count > modelsOut[j].Count })
	return byProvider, byAccount, modelsOut, nil
}

func usageInt64(u Usage, key string) int64 {
	v := u.Get(key)
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	case json.JSONObject:
		f, _ := n.Float()
		return int64(f)
	default:
		s := fmt.Sprintf("%v", n)
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return int64(f)
		}
		log.Debugf("usageInt64 unknown type %T for %s", v, key)
		return 0
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func pickUsageKeys(u Usage, keys ...string) Usage {
	out := Usage{}
	for _, k := range keys {
		if v := u.Get(k); v != nil {
			out[k] = v
		}
	}
	return out
}
