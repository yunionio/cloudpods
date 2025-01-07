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

package guestman

import (
	"fmt"
	"path"
	"sort"
	"sync"

	"github.com/jaypipes/ghw/pkg/topology"

	"yunion.io/x/cloudmux/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cpuset"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type SBaseParams struct {
	Sid  string
	Body jsonutils.JSONObject
}

type SGuestDeploy struct {
	UserCred mcclient.TokenCredential

	Sid    string
	Body   jsonutils.JSONObject
	IsInit bool
}

type SSrcPrepareMigrate struct {
	Sid               string
	LiveMigrate       bool
	LiveMigrateUseTLS bool
}

type SDestPrepareMigrate struct {
	Sid          string
	ServerUrl    string
	QemuVersion  string
	MigrateCerts map[string]string
	EnableTLS    bool
	SnapshotsUri string
	DisksUri     string
	// TargetStorageId string
	TargetStorageIds []string
	LiveMigrate      bool
	RebaseDisks      bool

	Desc               *desc.SGuestDesc
	SrcDesc            *desc.SGuestDesc
	DisksBackingFile   jsonutils.JSONObject
	DiskSnapsChain     jsonutils.JSONObject
	OutChainSnaps      jsonutils.JSONObject
	SysDiskHasTemplate bool

	MemorySnapshotsUri string
	SrcMemorySnapshots []string

	UserCred mcclient.TokenCredential
}

type SLiveMigrate struct {
	Sid            string
	DestPort       int
	NbdServerPort  int
	DestIp         string
	IsLocal        bool
	EnableTLS      bool
	MaxBandwidthMB *int64
	QuicklyFinish  bool
}

type SDriverMirror struct {
	Sid          string
	NbdServerUri string
	Desc         *desc.SGuestDesc
}

type SGuestHotplugCpuMem struct {
	Sid         string
	AddCpuCount int64
	AddMemSize  int64

	CpuNumaPin []*desc.SCpuNumaPin
}

type SReloadDisk struct {
	Sid  string
	Disk storageman.IDisk
}

type SBackupDiskConfig struct {
	compute.DiskConfig
	Name        string                        `json:"name"`
	BackupAsTar *compute.DiskBackupAsTarInput `json:"backup_as_tar"`
}

type SDiskSnapshot struct {
	UserCred         mcclient.TokenCredential
	Sid              string
	SnapshotId       string
	BackupDiskConfig *SBackupDiskConfig
	Disk             storageman.IDisk
}

type SMemorySnapshot struct {
	*hostapi.GuestMemorySnapshotRequest
	Sid string
}

type SMemorySnapshotReset struct {
	*hostapi.GuestMemorySnapshotResetRequest
	Sid string
}

type SMemorySnapshotDelete struct {
	*hostapi.GuestMemorySnapshotDeleteRequest
}

type SDiskBackup struct {
	Sid        string
	SnapshotId string
	BackupId   string
	Disk       storageman.IDisk
}

type SDeleteDiskSnapshot struct {
	Sid             string
	DeleteSnapshot  string
	Disk            storageman.IDisk
	ConvertSnapshot string
	BlockStream     bool
	EncryptInfo     apis.SEncryptInfo

	TotalDeleteSnapshotCount int
	DeletedSnapshotCount     int
}

type SLibvirtServer struct {
	Uuid  string
	MacIp map[string]string
}

type SLibvirtDomainImportConfig struct {
	LibvritDomainXmlDir string
	Servers             []SLibvirtServer
}

type SGuestCreateFromLibvirt struct {
	Sid         string
	MonitorPath string
	GuestDesc   *desc.SGuestDesc
	DisksPath   *jsonutils.JSONDict
}

type SGuestIoThrottle struct {
	Sid   string
	Input *compute.ServerSetDiskIoThrottleInput
}

type SGuestCreateFromEsxi struct {
	Sid            string
	GuestDesc      *desc.SGuestDesc
	EsxiAccessInfo SEsxiAccessInfo
}

type SEsxiAccessInfo struct {
	Datastore  vcenter.SVCenterAccessInfo
	HostIp     string
	GuestExtId string
}

type SGuestCreateFromCloudpods struct {
	Sid                 string
	GuestDesc           *desc.SGuestDesc
	CloudpodsAccessInfo SCloudpodsAccessInfo
}

type SCloudpodsAccessInfo struct {
	HostIp        string
	OriginDisksId []string
}

type SQgaGuestSetPassword struct {
	*hostapi.GuestSetPasswordRequest
	Sid string
}

type SQgaGuestSetNetwork struct {
	Timeout int
	Sid     string
	Device  string
	Ipmask  string
	Gateway string
}

type CpuSetCounter struct {
	Nodes       []*NumaNode
	NumaEnabled bool
	CPUCmtbound float32
	MEMCmtbound float32

	GuestIds map[string]struct{}
	Lock     sync.Mutex
}

func NewGuestCpuSetCounter(
	info *hostapi.HostTopology, reservedCpus cpuset.CPUSet, numaAllocate, isContainerHost bool,
	hugepageSizeKB int, cpuCmtbound, memCmtBound float32, reservedMemMb int,
) (*CpuSetCounter, error) {
	cpuSetCounter := new(CpuSetCounter)
	cpuSetCounter.Nodes = make([]*NumaNode, len(info.Nodes))
	cpuSetCounter.NumaEnabled = numaAllocate
	cpuSetCounter.CPUCmtbound = cpuCmtbound
	cpuSetCounter.MEMCmtbound = memCmtBound
	cpuSetCounter.GuestIds = map[string]struct{}{}
	hasL3Cache := false
	nodeReserveMem := reservedMemMb / len(info.Nodes) * 1024
	for i := 0; i < len(info.Nodes); i++ {
		node, err := NewNumaNode(
			info.Nodes[i],
			cpuSetCounter.NumaEnabled,
			isContainerHost,
			hugepageSizeKB,
			memCmtBound,
			nodeReserveMem,
		)
		if err != nil {
			return nil, err
		}
		reservedCpuCnt := 0
		cpuDies := make([]*CPUDie, 0)
		for j := 0; j < len(info.Nodes[i].Caches); j++ {
			if info.Nodes[i].Caches[j].Level != 3 {
				continue
			}
			hasL3Cache = true
			cpuDie := new(CPUDie)
			dieBuilder := cpuset.NewBuilder()
			for k := 0; k < len(info.Nodes[i].Caches[j].LogicalProcessors); k++ {
				if reservedCpus.Contains(int(info.Nodes[i].Caches[j].LogicalProcessors[k])) {
					reservedCpuCnt += 1
					continue
				}
				dieBuilder.Add(int(info.Nodes[i].Caches[j].LogicalProcessors[k]))
			}
			cpuDie.LogicalProcessors = dieBuilder.Result()
			node.CpuCount += cpuDie.LogicalProcessors.Size()
			node.LogicalProcessors = node.LogicalProcessors.Union(cpuDie.LogicalProcessors)
			cpuDie.initCpuFree(cpuCmtbound)

			cpuDies = append(cpuDies, cpuDie)
		}
		if !hasL3Cache {
			cpuDie := new(CPUDie)
			dieBuilder := cpuset.NewBuilder()
			for j := 0; j < len(info.Nodes[i].Cores); j++ {
				for k := 0; k < len(info.Nodes[i].Cores[j].LogicalProcessors); k++ {
					if reservedCpus.Contains(info.Nodes[i].Cores[j].LogicalProcessors[k]) {
						reservedCpuCnt += 1
						continue
					}
					dieBuilder.Add(info.Nodes[i].Cores[j].LogicalProcessors[k])
				}
			}
			cpuDie.LogicalProcessors = dieBuilder.Result()
			node.CpuCount += cpuDie.LogicalProcessors.Size()
			node.LogicalProcessors = node.LogicalProcessors.Union(cpuDie.LogicalProcessors)
			cpuDie.initCpuFree(cpuCmtbound)

			cpuDies = append(cpuDies, cpuDie)
		}

		hasL3Cache = false
		node.CpuDies = cpuDies
		node.ReserveCpuCount = reservedCpuCnt
		sort.Sort(node.CpuDies)
		cpuSetCounter.Nodes[i] = node
	}
	sort.Sort(cpuSetCounter)
	log.Infof("cpusetcounter %s", jsonutils.Marshal(cpuSetCounter))
	return cpuSetCounter, nil
}

func (pq *CpuSetCounter) AllocCpusetWithNodeCount(vcpuCount int, memSizeKB int64, nodeCount int, guestId string) (map[int]SAllocNumaCpus, error) {
	if !pq.NumaEnabled {
		return pq.AllocCpuset(vcpuCount, memSizeKB, nil, guestId)
	}
	if len(pq.Nodes) < nodeCount {
		return nil, nil
	}
	pq.GuestIds[guestId] = struct{}{}

	pq.Lock.Lock()
	defer pq.Lock.Unlock()
	var res = map[int]SAllocNumaCpus{}
	var nodeAllocSize = memSizeKB / int64(nodeCount)
	if pq.nodesEnough(nodeCount, vcpuCount, int(memSizeKB)) {
		var pcpuCount = vcpuCount / nodeCount
		var remPcpuCount = vcpuCount % nodeCount

		for i := 0; i < nodeCount; i++ {
			var npcpuCount = pcpuCount
			if remPcpuCount > 0 {
				npcpuCount += 1
				remPcpuCount -= 1
			}
			res[pq.Nodes[i].NodeId] = SAllocNumaCpus{
				Cpuset:    pq.Nodes[i].AllocCpuset(npcpuCount),
				MemSizeKB: nodeAllocSize,
				Unregular: false,
			}
			pq.Nodes[i].NumaNodeFreeMemSizeKB -= nodeAllocSize
			pq.Nodes[i].VcpuCount += npcpuCount
		}
	}
	return res, nil
}

type SAllocNumaCpus struct {
	Cpuset    []int
	MemSizeKB int64

	Unregular bool
}

func (pq *CpuSetCounter) IsNumaEnabled() bool {
	return pq.NumaEnabled
}

func (pq *CpuSetCounter) AllocCpuset(vcpuCount int, memSizeKB int64, preferNumaNodes []int8, guestId string) (map[int]SAllocNumaCpus, error) {
	pq.Lock.Lock()
	defer pq.Lock.Unlock()

	if len(pq.Nodes) == 0 {
		return nil, nil
	}
	pq.GuestIds[guestId] = struct{}{}

	if pq.NumaEnabled && len(preferNumaNodes) > 0 {
		sortedNumaDistance := pq.getDistancesSeqByPreferNodes(preferNumaNodes)
		for nodeCount := 1; nodeCount <= len(pq.Nodes); nodeCount *= 2 {
			ret := pq.allocCpuNumaNodesByPreferNodes(vcpuCount, int(memSizeKB), nodeCount, sortedNumaDistance)
			if ret != nil {
				for i := range pq.Nodes {
					if cpupin, ok := ret[pq.Nodes[i].NodeId]; ok {
						pq.Nodes[i].VcpuCount += vcpuCount
						pq.Nodes[i].NumaNodeFreeMemSizeKB -= cpupin.MemSizeKB
					}
				}
				sort.Sort(pq)
				return ret, nil
			}
		}
	}

	res := map[int]SAllocNumaCpus{}
	sourceVcpuCount := vcpuCount
	if pq.NumaEnabled {
		err := pq.AllocNumaNodes(vcpuCount, memSizeKB, res)
		return res, err
	} else {
		for vcpuCount > 0 {
			count := vcpuCount
			if vcpuCount > pq.Nodes[0].CpuCount {
				count = vcpuCount/2 + vcpuCount%2
			}
			res[pq.Nodes[0].NodeId] = SAllocNumaCpus{
				Cpuset: pq.Nodes[0].AllocCpuset(count),
			}
			pq.Nodes[0].VcpuCount += sourceVcpuCount
			vcpuCount -= count
			sort.Sort(pq)
		}
		return res, nil
	}
}

func (pq *CpuSetCounter) allocCpuNumaNodesByPreferNodes(
	vcpuCount, memSizeKB, nodeCount int, sortedNumaDistance []SSortedNumaDistance,
) map[int]SAllocNumaCpus {
	res := map[int]SAllocNumaCpus{}
	var nodeAllocSize = memSizeKB / nodeCount
	var pcpuCount = vcpuCount / nodeCount
	var remPcpuCount = vcpuCount % nodeCount

	allocatedNode := 0
	for i := range sortedNumaDistance {
		if allocatedNode >= nodeCount {
			break
		}

		var npcpuCount = pcpuCount
		if remPcpuCount > 0 {
			npcpuCount += 1
			remPcpuCount -= 1
		}
		nodeIdx := sortedNumaDistance[i].NodeIndex
		if pq.Nodes[nodeIdx].nodeEnough(vcpuCount, memSizeKB, pq.CPUCmtbound, pq.NumaEnabled) {
			cpuNumaPin := SAllocNumaCpus{
				Cpuset: pq.Nodes[nodeIdx].AllocCpuset(npcpuCount),
			}
			cpuNumaPin.MemSizeKB = int64(nodeAllocSize)
			res[pq.Nodes[nodeIdx].NodeId] = cpuNumaPin
			allocatedNode += 1
		} else {
			log.Infof("node %v not enough", pq.Nodes[i])
		}
		log.Infof("node %d, free mems %d, vcpuCount %d, GuestCounts %v", pq.Nodes[nodeIdx].NodeId, pq.Nodes[nodeIdx].NumaNodeFreeMemSizeKB, pq.Nodes[nodeIdx].VcpuCount, len(pq.GuestIds))
	}

	if allocatedNode < nodeCount {
		return nil
	}
	return res
}

type SSortedNumaDistance struct {
	NodeIndex   int
	Distance    int
	FreeMemSize int
	UsedRate    float32
	CpuReserved bool
}

func (pq *CpuSetCounter) getDistancesSeqByPreferNodes(preferNumaNodes []int8) []SSortedNumaDistance {
	sortedNumaDistance := make([]SSortedNumaDistance, len(pq.Nodes))
	for i := range pq.Nodes {
		distance := 0
		for j := range preferNumaNodes {
			distance += pq.Nodes[i].Distances[preferNumaNodes[j]]
		}

		var useableCpuRate float32 = 1.0
		if pq.Nodes[i].ReserveCpuCount > 0 {
			useableCpuRate = float32(pq.Nodes[i].CpuCount) / float32(pq.Nodes[i].CpuCount+pq.Nodes[i].ReserveCpuCount)
		}

		usedMems := float32(pq.Nodes[i].NumaNodeMemSizeKB - pq.Nodes[i].NumaNodeFreeMemSizeKB)
		usedRate := usedMems / (float32(pq.Nodes[i].MemTotalSizeKB) * pq.MEMCmtbound * useableCpuRate)

		//memCmt := float32(usedMems / pq.Nodes[i].NumaNodeMemSizeKB)
		//cpuPro := float32(pq.Nodes[i].CpuCount) * pq.CPUCmtbound / (float32(pq.Nodes[i].CpuCount)*pq.CPUCmtbound - float32(pq.Nodes[i].VcpuCount))
		sortedNumaDistance[i] = SSortedNumaDistance{
			NodeIndex:   i,
			Distance:    distance,
			FreeMemSize: int(pq.Nodes[i].NumaNodeFreeMemSizeKB),
			UsedRate:    usedRate,
			CpuReserved: pq.Nodes[i].ReserveCpuCount > 0,
		}
	}
	sort.Slice(sortedNumaDistance, func(i, j int) bool {
		// 7 is tolerant max distances
		if sortedNumaDistance[i].Distance > (7 + sortedNumaDistance[j].Distance) {
			return false
		} else if (sortedNumaDistance[i].Distance + 7) < sortedNumaDistance[j].Distance {
			return true
		}

		if sortedNumaDistance[i].CpuReserved {
			return sortedNumaDistance[i].UsedRate < sortedNumaDistance[j].UsedRate
		}
		return sortedNumaDistance[i].FreeMemSize > sortedNumaDistance[j].FreeMemSize
	})
	return sortedNumaDistance
}

func (pq *CpuSetCounter) AllocNumaNodes(vcpuCount int, memSizeKB int64, res map[int]SAllocNumaCpus) error {
	var allocated = false

	// alloc numa nodes in order 1, 2, 4, ...
	if !allocated {
		for nodeCount := 1; nodeCount <= len(pq.Nodes); nodeCount *= 2 {
			if nodeCount > vcpuCount {
				break
			}
			if ok := pq.nodesEnough(nodeCount, vcpuCount, int(memSizeKB)); !ok {
				log.Infof("node count %d not enough", nodeCount)
				continue
			}
			var nodeAllocSize = memSizeKB / int64(nodeCount)
			if nodeAllocSize/1024%1024 > 0 {
				continue
			}

			var pcpuCount = vcpuCount / nodeCount
			var remPcpuCount = vcpuCount % nodeCount
			for i := 0; i < nodeCount; i++ {
				var npcpuCount = pcpuCount
				if remPcpuCount > 0 {
					npcpuCount += 1
					remPcpuCount -= 1
				}
				res[pq.Nodes[i].NodeId] = SAllocNumaCpus{
					Cpuset:    pq.Nodes[i].AllocCpuset(npcpuCount),
					MemSizeKB: nodeAllocSize,
					Unregular: false,
				}
				pq.Nodes[i].NumaNodeFreeMemSizeKB -= nodeAllocSize
				pq.Nodes[i].VcpuCount += npcpuCount
			}
			allocated = true
			break
		}
	}

	// alloc numa nodes in order free numa node size
	//if !allocated {
	//	if ok := pq.nodesFreeMemSizeEnough(len(pq.Nodes), memSizeKB); !ok {
	//		return errors.Errorf("free hugepage is not enough")
	//	}
	//}
	sort.Sort(pq)
	return nil
}

func (pq *CpuSetCounter) nodesEnough(nodeCount, vcpuCount int, memSizeKB int) bool {
	var leastFree = memSizeKB / nodeCount
	var leastCpuCount = vcpuCount / nodeCount
	var remPcpuCount = vcpuCount % nodeCount

	for i := 0; i < nodeCount; i++ {
		if pq.NumaEnabled {
			if int(pq.Nodes[i].NumaNodeFreeMemSizeKB) < leastFree {
				return false
			}
		}

		requireCpuCount := leastCpuCount
		if remPcpuCount > 0 {
			requireCpuCount += 1
			remPcpuCount -= 1
		}
		if (pq.Nodes[i].VcpuCount + requireCpuCount) > int(float32(pq.Nodes[i].CpuCount)*pq.CPUCmtbound) {
			return false
		}

	}
	return true
}

func (pq *CpuSetCounter) nodesFreeMemSizeEnough(nodeCount int, memSizeKB int64) bool {
	var freeMem int64 = 0
	var leastFree = memSizeKB / int64(nodeCount)
	log.Debugf("request memsize %d, least free %d", memSizeKB, leastFree)
	for i := 0; i < nodeCount; i++ {
		log.Debugf("index %d node %d free size %d", i, pq.Nodes[i].NodeId, pq.Nodes[i].NumaNodeFreeMemSizeKB)
		if pq.Nodes[i].NumaNodeFreeMemSizeKB < leastFree {
			return false
		}
		freeMem += pq.Nodes[i].NumaNodeFreeMemSizeKB
	}
	return freeMem >= memSizeKB
}

func (pq *CpuSetCounter) setNumaNodes(numaMaps map[int]int, vcpuCount int64) map[int]SAllocNumaCpus {
	res := map[int]SAllocNumaCpus{}
	for i := range pq.Nodes {
		if size, ok := numaMaps[pq.Nodes[i].NodeId]; ok {
			allocMem := int64(size) * 1024
			//npcpuCount := int(vcpuCount*allocMem/memSizeKB + (vcpuCount*allocMem)%memSizeKB)
			res[pq.Nodes[i].NodeId] = SAllocNumaCpus{
				Cpuset:    pq.Nodes[i].AllocCpuset(int(vcpuCount)),
				MemSizeKB: allocMem,
				Unregular: true,
			}

			pq.Nodes[i].NumaNodeFreeMemSizeKB -= allocMem
			pq.Nodes[i].VcpuCount += int(vcpuCount)
		}
	}
	sort.Sort(pq)
	return res
}

func (pq *CpuSetCounter) ReleaseCpus(cpus []int, vcpuCount int) {
	var numaCpuCount = map[int][]int{}
	for i := 0; i < len(cpus); i++ {
		for j := 0; j < len(pq.Nodes); j++ {
			if pq.Nodes[j].LogicalProcessors.Contains(cpus[i]) {
				if numaCpus, ok := numaCpuCount[pq.Nodes[j].NodeId]; !ok {
					numaCpuCount[pq.Nodes[j].NodeId] = []int{cpus[i]}
				} else {
					numaCpuCount[pq.Nodes[j].NodeId] = append(numaCpus, cpus[i])
				}
				break
			}
		}
	}
	for i := 0; i < len(pq.Nodes); i++ {
		if numaCpus, ok := numaCpuCount[pq.Nodes[i].NodeId]; ok {
			pq.Nodes[i].CpuDies.ReleaseCpus(numaCpus, vcpuCount)
			pq.Nodes[i].VcpuCount -= vcpuCount
		}
	}
	sort.Sort(pq)
}

func (pq *CpuSetCounter) ReleaseNumaCpus(memSizeMb int64, hostNode int, cpus []int, vcpuCount int) {
	for i := 0; i < len(pq.Nodes); i++ {
		if pq.Nodes[i].NodeId != hostNode {
			continue
		}
		pq.Nodes[i].CpuDies.ReleaseCpus(cpus, vcpuCount)
		pq.Nodes[i].VcpuCount -= vcpuCount
		pq.Nodes[i].NumaNodeFreeMemSizeKB += memSizeMb * 1024
	}
	sort.Sort(pq)
}

func (pq *CpuSetCounter) LoadNumaCpus(memSizeMb int64, hostNode int, cpus []int, vcpuCount int) {
	for i := 0; i < len(pq.Nodes); i++ {
		if pq.Nodes[i].NodeId != hostNode {
			continue
		}
		pq.Nodes[i].CpuDies.LoadCpus(cpus, vcpuCount)
		pq.Nodes[i].VcpuCount += vcpuCount
		pq.Nodes[i].NumaNodeFreeMemSizeKB -= memSizeMb * 1024
	}
	sort.Sort(pq)
}

func (pq *CpuSetCounter) LoadCpus(cpus []int, vcpuCpunt int) {
	var numaCpuCount = map[int][]int{}
	for i := 0; i < len(cpus); i++ {
		for j := 0; j < len(pq.Nodes); j++ {
			if pq.Nodes[j].LogicalProcessors.Contains(cpus[i]) {
				if numaCpus, ok := numaCpuCount[pq.Nodes[j].NodeId]; !ok {
					numaCpuCount[pq.Nodes[j].NodeId] = []int{cpus[i]}
				} else {
					numaCpuCount[pq.Nodes[j].NodeId] = append(numaCpus, cpus[i])
				}
				break
			}
		}
	}
	for i := 0; i < len(pq.Nodes); i++ {
		if numaCpus, ok := numaCpuCount[pq.Nodes[i].NodeId]; ok {
			pq.Nodes[i].CpuDies.LoadCpus(numaCpus, vcpuCpunt)
			pq.Nodes[i].VcpuCount += vcpuCpunt
		}
	}
	sort.Sort(pq)
}

func (pq CpuSetCounter) Len() int { return len(pq.Nodes) }

func (pq CpuSetCounter) Less(i, j int) bool {
	freeCpuI := int(float32(pq.Nodes[i].CpuCount)*pq.CPUCmtbound) - pq.Nodes[i].VcpuCount
	freeCpuJ := int(float32(pq.Nodes[i].CpuCount)*pq.CPUCmtbound) - pq.Nodes[j].VcpuCount
	if pq.NumaEnabled {
		if pq.Nodes[i].NumaNodeFreeMemSizeKB == pq.Nodes[j].NumaNodeFreeMemSizeKB {
			return freeCpuI > freeCpuJ
		}
		return pq.Nodes[i].NumaNodeFreeMemSizeKB > pq.Nodes[j].NumaNodeFreeMemSizeKB
	} else {
		return freeCpuI > freeCpuJ
	}
}

func (pq CpuSetCounter) Swap(i, j int) {
	pq.Nodes[i], pq.Nodes[j] = pq.Nodes[j], pq.Nodes[i]
}

func (pq *CpuSetCounter) Push(item interface{}) {
	(*pq).Nodes = append((*pq).Nodes, item.(*NumaNode))
}

func (pq *CpuSetCounter) Pop() interface{} {
	old := *pq
	n := len(old.Nodes)
	item := old.Nodes[n-1]
	old.Nodes[n-1] = nil // avoid memory leak
	(*pq).Nodes = old.Nodes[0 : n-1]
	return item
}

type NumaNode struct {
	CpuDies           SorttedCPUDie
	LogicalProcessors cpuset.CPUSet
	VcpuCount         int
	CpuCount          int
	ReserveCpuCount   int

	NodeId                int
	Distances             []int
	NumaNodeMemSizeKB     int64
	MemTotalSizeKB        int64
	NumaNodeFreeMemSizeKB int64
}

func NewNumaNode(
	nodeInfo *topology.Node,
	numaAllocate, isContainerHost bool,
	hugepageSizeKB int, memCmtBound float32,
	reservedMemSizeKB int,
) (*NumaNode, error) {
	n := new(NumaNode)
	n.LogicalProcessors = cpuset.NewCPUSet()
	n.NodeId = nodeInfo.ID
	n.Distances = nodeInfo.Distances

	if !numaAllocate {
		return n, nil
	}
	if isContainerHost {
		if nodeInfo.Memory == nil {
			return nil, errors.Errorf("node %d no memory info: %#v", nodeInfo.ID, nodeInfo)
		}
		n.NumaNodeMemSizeKB = int64(float32(nodeInfo.Memory.TotalUsableBytes/1024-int64(reservedMemSizeKB)) * memCmtBound)
		n.MemTotalSizeKB = nodeInfo.Memory.TotalUsableBytes / 1024
	} else {
		nodeHugepagePath := fmt.Sprintf("/sys/devices/system/node/node%d/hugepages/hugepages-%dkB", n.NodeId, hugepageSizeKB)
		if !fileutils2.Exists(nodeHugepagePath) {
			return n, nil
		}
		nrHugepage, err := fileutils2.FileGetIntContent(path.Join(nodeHugepagePath, "nr_hugepages"))
		if err != nil {
			log.Errorf("failed get node %d nr hugepage %s", n.NodeId, err)
			return nil, errors.Wrap(err, "get numa node nr hugepage")
		}
		n.NumaNodeMemSizeKB = int64(nrHugepage) * int64(hugepageSizeKB)
	}
	n.NumaNodeFreeMemSizeKB = n.NumaNodeMemSizeKB

	return n, nil
}

func (n *NumaNode) nodeEnough(vcpuCount, memSizeKB int, cmtBound float32, enableNumaAlloc bool) bool {
	if int(float32(n.CpuCount)*cmtBound)-n.VcpuCount < vcpuCount {
		return false
	}
	if enableNumaAlloc {
		if int(n.NumaNodeFreeMemSizeKB) < memSizeKB {
			return false
		}
	}
	return true
}

func (n *NumaNode) AllocCpuset(vcpuCount int) []int {
	if options.HostOptions.EnableStrictCpuBind {
		return n.allocCpusetStrict(vcpuCount)
	}
	return n.allocCpusetOnNode(vcpuCount)
}

func (n *NumaNode) allocCpusetStrict(vcpuCount int) []int {
	var allocCount = vcpuCount
	var dieCnt = 0

	// If request vcpu count great then node cpucount,
	// vcpus should evenly distributed to all dies.
	// Otherwise figure out how many dies can hold
	// all of vcpus at first, and evenly distributed
	// to selected dies.
	if vcpuCount > n.CpuCount {
		dieCnt = len(n.CpuDies)
	} else {
		var pcpuCount = 0
		for dieCnt < len(n.CpuDies) {
			pcpuCount += n.CpuDies[dieCnt].LogicalProcessors.Size()
			dieCnt += 1

			if pcpuCount >= vcpuCount {
				break
			}
		}
	}

	var perDieCpuCount = vcpuCount / dieCnt
	var allocCpuCountMap = make([]int, dieCnt)
	for allocCount > 0 {
		for i := 0; i < dieCnt; i++ {
			var allocNum = perDieCpuCount
			if allocCount < allocNum {
				allocNum = allocCount
			}
			allocCount -= allocNum
			allocCpuCountMap[i] += allocNum
		}
	}

	defer sort.Sort(n.CpuDies)
	var ret = make([]int, 0)
	for i := 0; i < len(allocCpuCountMap); i++ {
		var allocCpuCount = allocCpuCountMap[i]
		for allocCpuCount > 0 {
			pcpus := n.CpuDies[i].LogicalProcessors.ToSliceNoSort()
			for j := 0; j < len(pcpus); j++ {
				if n.CpuDies[i].CpuFree[pcpus[j]] > 0 {
					ret = append(ret, n.CpuDies[i].CpuFree[pcpus[j]])
					n.CpuDies[i].CpuFree[pcpus[j]] -= 1
				}
				allocCpuCount -= 1
				if allocCpuCount <= 0 {
					break
				}
			}
		}
	}
	return ret
}

func (n *NumaNode) allocCpusetOnNode(vcpuCount int) []int {
	cpus := make([]int, 0)

	var allocCount = vcpuCount
	for i := range n.CpuDies {
		n.CpuDies[i].VcpuCount += vcpuCount
		cpus = append(cpus, n.CpuDies[i].LogicalProcessors.ToSliceNoSort()...)
		if allocCount > n.CpuDies[i].LogicalProcessors.Size() {
			allocCount -= n.CpuDies[i].LogicalProcessors.Size()
		} else {
			break
		}
	}

	sort.Sort(n.CpuDies)
	return cpus
}

type CPUDie struct {
	CpuFree           map[int]int
	LogicalProcessors cpuset.CPUSet
	VcpuCount         int
}

func (d *CPUDie) initCpuFree(cpuCmtbound float32) {
	cpuFree := map[int]int{}
	for _, cpuId := range d.LogicalProcessors.ToSliceNoSort() {
		cpuFree[cpuId] = int(cpuCmtbound)
	}
	d.CpuFree = cpuFree
}

type SorttedCPUDie []*CPUDie

func (pq SorttedCPUDie) Len() int { return len(pq) }

func (pq SorttedCPUDie) Less(i, j int) bool {
	return pq[i].VcpuCount < pq[j].VcpuCount
}

func (pq SorttedCPUDie) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *SorttedCPUDie) Push(item interface{}) {
	*pq = append(*pq, item.(*CPUDie))
}

func (pq *SorttedCPUDie) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*pq = old[0 : n-1]
	return item
}

func (pq *SorttedCPUDie) ReleaseCpus(cpus []int, vcpuCount int) {
	var cpuDies = map[int][]int{}
	for i := 0; i < len(cpus); i++ {
		for j := 0; j < len(*pq); j++ {
			if (*pq)[j].LogicalProcessors.Contains(cpus[i]) {
				if cpuDie, ok := cpuDies[j]; !ok {
					cpuDies[j] = []int{cpus[i]}
				} else {
					cpuDies[j] = append(cpuDie, cpus[i])
				}
				break
			}
		}
	}

	for i := 0; i < len(*pq); i++ {
		if _, ok := cpuDies[i]; ok {
			d := (*pq)[i]
			for _, cpu := range cpus {
				d.CpuFree[cpu] += 1
			}
			d.VcpuCount -= vcpuCount
		}
	}
	sort.Sort(pq)
}

func (pq *SorttedCPUDie) LoadCpus(cpus []int, vcpuCount int) {
	var cpuDies = map[int][]int{}
	for i := 0; i < len(cpus); i++ {
		for j := 0; j < len(*pq); j++ {
			if (*pq)[j].LogicalProcessors.Contains(cpus[i]) {
				if cpuDie, ok := cpuDies[j]; !ok {
					cpuDies[j] = []int{cpus[i]}
				} else {
					cpuDies[j] = append(cpuDie, cpus[i])
				}
				break
			}
		}
	}

	for i := 0; i < len(*pq); i++ {
		if cpus, ok := cpuDies[i]; ok {
			d := (*pq)[i]
			for _, cpu := range cpus {
				d.CpuFree[cpu] -= 1
			}
			d.VcpuCount += vcpuCount
		}
	}
	sort.Sort(pq)
}
