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
	"context"
	"encoding/json"
	"sort"
	gosync "sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/apis/scheduler"
	computedb "yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cpuset"
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

func (h *hostGetter) NumaAllocateEnabled() bool {
	return h.h.HostTopo.NumaEnabled || h.h.HostType == computeapi.HOST_TYPE_CONTAINER
}

func (h *hostGetter) GetFreeCpuNuma() []*scheduler.SFreeNumaCpuMem {
	return h.h.GetFreeCpuNuma()
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

func (h *hostGetter) GetFreeStorageSizeOfType(storageType string, mediumType string, useRsvd bool, reqMaxSize int64) (int64, int64, error) {
	return h.h.GetFreeStorageSizeOfType(storageType, mediumType, useRsvd, reqMaxSize)
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

	EnableCpuNumaAllocate bool       `json:"enable_cpu_numa_allocate"`
	HostTopo              *SHostTopo `json:"host_topo"`

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
	Metadata                  map[string]string `json:"metadata"`
	IsMaintenance             bool              `json:"is_maintenance"`
	GuestReservedResource     *ReservedResource `json:"guest_reserved_resource"`
	GuestReservedResourceUsed *ReservedResource `json:"guest_reserved_used"`
}

type CPUFree struct {
	Cpu  int
	Free int
}
type SorttedCPUFree []*CPUFree

func (pq *SorttedCPUFree) LoadCpu(cpuId int) {
	for i := range *pq {
		if (*pq)[i].Cpu == cpuId {
			(*pq)[i].Free -= 1
		}
	}
}

func (pq SorttedCPUFree) Len() int { return len(pq) }

func (pq SorttedCPUFree) Less(i, j int) bool {
	return pq[i].Free > pq[j].Free
}

func (pq SorttedCPUFree) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *SorttedCPUFree) Push(item interface{}) {
	*pq = append(*pq, item.(*CPUFree))
}

func (pq *SorttedCPUFree) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*pq = old[0 : n-1]
	return item
}

type CPUDie struct {
	LogicalProcessors cpuset.CPUSet
	// Core thread id maps
	CoreThreadIdMaps map[int]int

	CpuFree   SorttedCPUFree
	VcpuCount int
}

func (d *CPUDie) initCpuFree(cpuCmtbound int) {
	cpuFree := make([]*CPUFree, 0)
	for _, cpuId := range d.LogicalProcessors.ToSliceNoSort() {
		cpuFree = append(cpuFree, &CPUFree{cpuId, cpuCmtbound})
	}
	d.CpuFree = cpuFree
	sort.Sort(d.CpuFree)
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
				d.CpuFree.LoadCpu(cpu)
			}
			d.VcpuCount += vcpuCount

			sort.Sort(d.CpuFree)
		}
	}
	sort.Sort(pq)
}

type NumaNode struct {
	CpuDies           SorttedCPUDie
	LogicalProcessors cpuset.CPUSet
	VcpuCount         int
	CpuCount          int

	NodeId                int
	Distances             []int
	NumaNodeMemSizeKB     int
	NumaNodeFreeMemSizeKB int
}

func (n *NumaNode) nodeEnough(vcpuCount, memSizeKB int, cmtBound float32, enableNumaAlloc bool) bool {
	if int(float32(n.CpuCount)*cmtBound)-n.VcpuCount < vcpuCount {
		return false
	}
	if enableNumaAlloc {
		if n.NumaNodeFreeMemSizeKB < memSizeKB {
			return false
		}
	}
	return true
}

func (n *NumaNode) allocCpusetSequenceN(vcpuCount int, usedCpu map[int]int) {
	var seqNumber = o.Options.GuestCpusetAllocSequenceInterval
	if vcpuCount%seqNumber != 0 || n.CpuCount/len(n.CpuDies) < vcpuCount {
		n._allocCpuset(vcpuCount, usedCpu)
		return
	}

	for i := range n.CpuDies {
		detectedSet := cpuset.NewCPUSet()
		for j := range n.CpuDies[i].CpuFree {
			if detectedSet.Contains(n.CpuDies[i].CpuFree[j].Cpu) {
				continue
			}

			cpuIdBase := n.CpuDies[i].CpuFree[j].Cpu - n.CpuDies[i].CpuFree[j].Cpu%vcpuCount
			lo, hi := cpuIdBase, cpuIdBase+vcpuCount-1
			cpuIds := make([]int, hi-lo+1)
			for m := range cpuIds {
				cpuIds[m] = m + lo
			}

			var matched = true
			cpuIdSet := cpuset.NewCPUSet(cpuIds...)
			detectedSet = detectedSet.Union(cpuIdSet)
			for k := range n.CpuDies[i].CpuFree {
				if !cpuIdSet.Contains(n.CpuDies[i].CpuFree[k].Cpu) {
					continue
				}
				if n.CpuDies[i].CpuFree[k].Free <= 0 {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}

			for m := range cpuIds {
				usedCpu[cpuIds[m]] = 1
			}
			return
		}
	}

	n._allocCpuset(vcpuCount, usedCpu)
}

func (n *NumaNode) allocCpuset(vcpuCount int, usedCpu map[int]int) {
	if o.Options.GuestCpusetAllocSequence {
		n.allocCpusetSequenceN(vcpuCount, usedCpu)
		return
	}
	n._allocCpuset(vcpuCount, usedCpu)
}

func (n *NumaNode) _allocCpuset(vcpuCount int, usedCpu map[int]int) {
	for i := range n.CpuDies {
		for j := range n.CpuDies[i].CpuFree {
			cpuId, nFree := n.CpuDies[i].CpuFree[j].Cpu, n.CpuDies[i].CpuFree[j].Free
			if cnt, ok := usedCpu[cpuId]; ok {
				if cnt < nFree {
					usedCpu[cpuId] = cnt + 1
					vcpuCount -= 1
					if vcpuCount <= 0 {
						return
					}
				}
			} else {
				if nFree > 0 {
					usedCpu[cpuId] = 1
					vcpuCount -= 1
					if vcpuCount <= 0 {
						return
					}
				}
			}
			if pairCpuId, ok := n.CpuDies[i].CoreThreadIdMaps[cpuId]; ok {
				for k := range n.CpuDies[i].CpuFree {
					if n.CpuDies[i].CpuFree[k].Cpu == pairCpuId {
						pairNFree := n.CpuDies[i].CpuFree[k].Free
						if cnt, ok := usedCpu[pairCpuId]; ok {
							if cnt < pairNFree {
								usedCpu[pairCpuId] = cnt + 1
								vcpuCount -= 1
								if vcpuCount <= 0 {
									return
								}
							}
						} else {
							if pairNFree > 0 {
								usedCpu[pairCpuId] = 1
								vcpuCount -= 1
								if vcpuCount <= 0 {
									return
								}
							}
						}
					}
				}
			}
		}
		//sort.Sort(n.CpuDies[i].CpuFree)
	}
	n._allocCpuset(vcpuCount, usedCpu)
}

func (n *NumaNode) AllocCpuset(vcpuCount int) []int {
	if vcpuCount <= 0 {
		return nil
	}

	var usedCpuCount = make(map[int]int)
	n.allocCpuset(vcpuCount, usedCpuCount)

	var ret = make([]int, 0)
	for cpuId, cnt := range usedCpuCount {
		for cnt > 0 {
			ret = append(ret, cpuId)
			cnt -= 1
		}
	}
	return ret
}

func NewNumaNode(nodeId int, nodeDistances []int, hugepageSizeKb int, nodeHugepages []hostapi.HostNodeHugepageNr, memSizeKB int, memCmtBound float32) *NumaNode {
	n := new(NumaNode)
	n.LogicalProcessors = cpuset.NewCPUSet()
	n.NodeId = nodeId
	n.Distances = nodeDistances

	if len(nodeHugepages) > 0 {
		for i := range nodeHugepages {
			if nodeHugepages[i].NodeId == nodeId {
				n.NumaNodeMemSizeKB = nodeHugepages[i].HugepageNr * hugepageSizeKb
			}
		}
	} else {
		n.NumaNodeMemSizeKB = int(float32(memSizeKB) * memCmtBound)
	}

	n.NumaNodeFreeMemSizeKB = n.NumaNodeMemSizeKB
	return n
}

type SHostTopo struct {
	Nodes       []*NumaNode
	NumaEnabled bool
	CPUCmtbound float32
	HostName    string
}

func HostTopoSubPendingUsage(topo *SHostTopo, cpuUsage map[int]int, numaMemUsage map[int]int) *SHostTopo {
	res := new(SHostTopo)
	res.NumaEnabled = topo.NumaEnabled
	res.CPUCmtbound = topo.CPUCmtbound
	res.Nodes = make([]*NumaNode, len(topo.Nodes))
	for i := range topo.Nodes {
		res.Nodes[i] = new(NumaNode)
		res.Nodes[i].LogicalProcessors = topo.Nodes[i].LogicalProcessors.Clone()
		res.Nodes[i].VcpuCount = topo.Nodes[i].VcpuCount
		res.Nodes[i].CpuCount = topo.Nodes[i].CpuCount
		res.Nodes[i].NodeId = topo.Nodes[i].NodeId
		res.Nodes[i].NumaNodeMemSizeKB = topo.Nodes[i].NumaNodeMemSizeKB
		res.Nodes[i].NumaNodeFreeMemSizeKB = topo.Nodes[i].NumaNodeFreeMemSizeKB
		res.Nodes[i].Distances = topo.Nodes[i].Distances

		if memUsed, ok := numaMemUsage[topo.Nodes[i].NodeId]; ok {
			res.Nodes[i].NumaNodeFreeMemSizeKB -= memUsed * 1024
		}
		res.Nodes[i].CpuDies = make([]*CPUDie, len(topo.Nodes[i].CpuDies))
		for j := range topo.Nodes[i].CpuDies {
			res.Nodes[i].CpuDies[j] = &CPUDie{
				LogicalProcessors: topo.Nodes[i].CpuDies[j].LogicalProcessors.Clone(),
				CpuFree:           make(SorttedCPUFree, 0),
				VcpuCount:         topo.Nodes[i].CpuDies[j].VcpuCount,
			}

			for k := range topo.Nodes[i].CpuDies[j].CpuFree {
				cpuFree := topo.Nodes[i].CpuDies[j].CpuFree[k]
				cpuId := cpuFree.Cpu
				free := cpuFree.Free
				if pending, ok := cpuUsage[cpuId]; ok {
					res.Nodes[i].CpuDies[j].CpuFree = append(res.Nodes[i].CpuDies[j].CpuFree, &CPUFree{cpuId, free - pending})

					res.Nodes[i].CpuDies[j].VcpuCount += pending
					res.Nodes[i].VcpuCount += pending
				} else {
					res.Nodes[i].CpuDies[j].CpuFree = append(res.Nodes[i].CpuDies[j].CpuFree, &CPUFree{cpuId, free})
				}
			}

			sort.Sort(res.Nodes[i].CpuDies[j].CpuFree)
		}
	}
	sort.Sort(res)
	return res
}

func (pq SHostTopo) Len() int { return len(pq.Nodes) }

func (pq SHostTopo) Less(i, j int) bool {
	if pq.NumaEnabled {
		if pq.Nodes[i].NumaNodeFreeMemSizeKB == pq.Nodes[j].NumaNodeFreeMemSizeKB {
			return pq.Nodes[i].VcpuCount < pq.Nodes[j].VcpuCount
		}
		return pq.Nodes[i].NumaNodeFreeMemSizeKB > pq.Nodes[j].NumaNodeFreeMemSizeKB
	} else {
		return pq.Nodes[i].NumaNodeFreeMemSizeKB > pq.Nodes[j].NumaNodeFreeMemSizeKB
	}
}

func (pq SHostTopo) Swap(i, j int) {
	pq.Nodes[i], pq.Nodes[j] = pq.Nodes[j], pq.Nodes[i]
}

func (pq *SHostTopo) Push(item interface{}) {
	(*pq).Nodes = append((*pq).Nodes, item.(*NumaNode))
}

func (h *SHostTopo) LoadCpuNumaPin(guestsCpuNumaPin []scheduler.SCpuNumaPin) {
	for _, gCpuNumaPin := range guestsCpuNumaPin {
		var node *NumaNode
		for i := range h.Nodes {
			if h.Nodes[i].NodeId == gCpuNumaPin.NodeId {
				node = h.Nodes[i]
			}
		}

		cpus := gCpuNumaPin.CpuPin
		node.CpuDies.LoadCpus(cpus, len(cpus))
		if gCpuNumaPin.MemSizeMB != nil {
			node.NumaNodeFreeMemSizeKB -= *gCpuNumaPin.MemSizeMB * 1024
		}
		node.VcpuCount += len(cpus)
	}
	sort.Sort(h)
}

func (h *SHostTopo) nodesEnough(nodeCount, vcpuCount int, memSizeKB int) bool {
	var leastFree = memSizeKB / nodeCount
	var leastCpuCount = vcpuCount / nodeCount
	var remPcpuCount = vcpuCount % nodeCount

	for i := 0; i < nodeCount; i++ {
		if h.NumaEnabled {
			if h.Nodes[i].NumaNodeFreeMemSizeKB < leastFree {
				return false
			}
		}

		requireCpuCount := leastCpuCount
		if remPcpuCount > 0 {
			requireCpuCount += 1
			remPcpuCount -= 1
		}
		if (h.Nodes[i].VcpuCount + requireCpuCount) > int(float32(h.Nodes[i].CpuCount)*h.CPUCmtbound) {
			return false
		}

	}
	return true
}

func (h *SHostTopo) allocCpuNumaNodesByPreferNodes(
	vcpuCount, memSizeKB, nodeCount int, sortedNumaDistance []SSortedNumaDistance,
) []scheduler.SCpuNumaPin {
	res := make([]scheduler.SCpuNumaPin, 0)
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
		if h.Nodes[nodeIdx].nodeEnough(vcpuCount, memSizeKB, h.CPUCmtbound, h.NumaEnabled) {
			cpuNumaPin := scheduler.SCpuNumaPin{
				CpuPin: h.Nodes[nodeIdx].AllocCpuset(npcpuCount),
				NodeId: h.Nodes[nodeIdx].NodeId,
			}
			allocSize := nodeAllocSize / 1024
			cpuNumaPin.MemSizeMB = &allocSize
			res = append(res, cpuNumaPin)
			allocatedNode += 1
		} else {
			log.Infof("%s node %v not enough", h.HostName, h.Nodes[i])
		}

		log.Infof("node %d, free mems %d", h.Nodes[nodeIdx].NodeId, h.Nodes[nodeIdx].NumaNodeFreeMemSizeKB)
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
}

func (h *SHostTopo) getDistancesSeqByPreferNodes(preferNumaNodes []int) []SSortedNumaDistance {
	sortedNumaDistance := make([]SSortedNumaDistance, len(h.Nodes))
	for i := range h.Nodes {
		distance := 0
		for j := range preferNumaNodes {
			log.Infof("node distance %v", h.Nodes[i].Distances)
			distance += h.Nodes[i].Distances[preferNumaNodes[j]]
		}
		sortedNumaDistance[i] = SSortedNumaDistance{
			NodeIndex:   i,
			Distance:    distance,
			FreeMemSize: h.Nodes[i].NumaNodeFreeMemSizeKB,
		}
	}
	sort.Slice(sortedNumaDistance, func(i, j int) bool {
		// 7 is tolerant max distances
		if sortedNumaDistance[i].Distance > (7 + sortedNumaDistance[j].Distance) {
			return false
		} else if (sortedNumaDistance[i].Distance + 7) < sortedNumaDistance[j].Distance {
			return true
		}

		return sortedNumaDistance[i].FreeMemSize > sortedNumaDistance[j].FreeMemSize
	})
	return sortedNumaDistance
}

func (h *SHostTopo) AllocCpuNumaNodes(vcpuCount, memSizeKB int, ignoreMemSingular bool, preferNumaNodes []int) []scheduler.SCpuNumaPin {
	if h.NumaEnabled && len(preferNumaNodes) > 0 {
		log.Infof("preferNumaNodes %v", preferNumaNodes)
		sortedNumaDistance := h.getDistancesSeqByPreferNodes(preferNumaNodes)
		for nodeCount := 1; nodeCount <= len(h.Nodes); nodeCount *= 2 {
			ret := h.allocCpuNumaNodesByPreferNodes(vcpuCount, memSizeKB, nodeCount, sortedNumaDistance)
			if ret != nil {
				return ret
			}
		}
	}

	res := make([]scheduler.SCpuNumaPin, 0)
	for nodeCount := 1; nodeCount <= len(h.Nodes); nodeCount *= 2 {
		if ok := h.nodesEnough(nodeCount, vcpuCount, memSizeKB); !ok {
			log.Infof("host %s node count %d not enough", h.HostName, nodeCount)
			continue
		}
		log.Infof("use node count %d", nodeCount)

		var nodeAllocSize = memSizeKB / nodeCount
		if h.NumaEnabled && !ignoreMemSingular {
			if nodeAllocSize/1024%1024 > 0 {
				log.Infof("host %s node alloc size singular %d", h.HostName, nodeAllocSize)
				continue
			}
		}

		var pcpuCount = vcpuCount / nodeCount
		var remPcpuCount = vcpuCount % nodeCount
		for i := 0; i < nodeCount; i++ {
			var npcpuCount = pcpuCount
			if remPcpuCount > 0 {
				npcpuCount += 1
				remPcpuCount -= 1
			}
			cpuNumaPin := scheduler.SCpuNumaPin{
				CpuPin: h.Nodes[i].AllocCpuset(npcpuCount),
				NodeId: h.Nodes[i].NodeId,
			}
			if h.NumaEnabled {
				allocSize := nodeAllocSize / 1024
				cpuNumaPin.MemSizeMB = &allocSize
			}
			res = append(res, cpuNumaPin)
		}
		break
	}

	return res
}

func (h *SHostTopo) AllocCpuNumaNodesWithNodeCount(vcpuCount, memSizeKB, nodeCount int) []scheduler.SCpuNumaPin {
	res := make([]scheduler.SCpuNumaPin, 0)
	var nodeAllocSize = memSizeKB / nodeCount
	var pcpuCount = vcpuCount / nodeCount
	var remPcpuCount = vcpuCount % nodeCount
	for i := 0; i < nodeCount; i++ {
		var npcpuCount = pcpuCount
		if remPcpuCount > 0 {
			npcpuCount += 1
			remPcpuCount -= 1
		}

		cpuNumaPin := scheduler.SCpuNumaPin{
			CpuPin: h.Nodes[i].AllocCpuset(npcpuCount),
			NodeId: h.Nodes[i].NodeId,
		}
		if h.NumaEnabled {
			allocSize := nodeAllocSize / 1024
			cpuNumaPin.MemSizeMB = &allocSize
		}
		res = append(res, cpuNumaPin)
	}
	return res
}

func (b *HostBuilder) buildHostTopo(
	desc *HostDesc, reservedCpus *cpuset.CPUSet,
	hugepageSizeKb int, nodeHugepages []hostapi.HostNodeHugepageNr,
	info *hostapi.HostTopology,
) error {
	var numaEnabled = len(nodeHugepages) > 0
	hostTopo := new(SHostTopo)
	hostTopo.Nodes = make([]*NumaNode, len(info.Nodes))

	hasL3Cache := false
	for i := 0; i < len(info.Nodes); i++ {
		nodoMemSizeKB := 0
		if info.Nodes[i].Memory != nil {
			nodoMemSizeKB = int(info.Nodes[i].Memory.TotalUsableBytes/1024) - (desc.MemReserved * 1024 / len(info.Nodes))
			if desc.HostType == computeapi.HOST_TYPE_CONTAINER && o.Options.ContainerNumaAllocate {
				numaEnabled = true
				log.Infof("host %s ignore singular", desc.Name)
			}
		}

		node := NewNumaNode(info.Nodes[i].ID, info.Nodes[i].Distances, hugepageSizeKb, nodeHugepages, nodoMemSizeKB, desc.MemCmtbound)

		cpuDies := make([]*CPUDie, 0)
		for j := 0; j < len(info.Nodes[i].Caches); j++ {
			if info.Nodes[i].Caches[j].Level != 3 {
				continue
			}
			hasL3Cache = true
			cpuDie := new(CPUDie)
			cpuDie.CoreThreadIdMaps = make(map[int]int)
			dieBuilder := cpuset.NewBuilder()
			for k := 0; k < len(info.Nodes[i].Caches[j].LogicalProcessors); k++ {
				if reservedCpus != nil && reservedCpus.Contains(int(info.Nodes[i].Caches[j].LogicalProcessors[k])) {
					continue
				}
				dieBuilder.Add(int(info.Nodes[i].Caches[j].LogicalProcessors[k]))
			}
			cpuDie.LogicalProcessors = dieBuilder.Result()
			cpuDie.initCpuFree(int(desc.CPUCmtbound))

			for _, c := range info.Nodes[i].Cores {
				if len(c.LogicalProcessors) != 2 {
					continue
				}
				if cpuDie.LogicalProcessors.Contains(c.LogicalProcessors[0]) {
					cpuDie.CoreThreadIdMaps[c.LogicalProcessors[0]] = c.LogicalProcessors[1]
					cpuDie.CoreThreadIdMaps[c.LogicalProcessors[1]] = c.LogicalProcessors[0]
				}
			}

			node.CpuCount += cpuDie.LogicalProcessors.Size()
			node.LogicalProcessors = node.LogicalProcessors.Union(cpuDie.LogicalProcessors)
			cpuDies = append(cpuDies, cpuDie)

			// TODO: add cpu core builder
		}
		if !hasL3Cache {
			cpuDie := new(CPUDie)
			dieBuilder := cpuset.NewBuilder()
			for j := 0; j < len(info.Nodes[i].Cores); j++ {
				for k := 0; k < len(info.Nodes[i].Cores[j].LogicalProcessors); k++ {
					if reservedCpus != nil && reservedCpus.Contains(info.Nodes[i].Cores[j].LogicalProcessors[k]) {
						continue
					}
					dieBuilder.Add(info.Nodes[i].Cores[j].LogicalProcessors[k])
				}
			}
			cpuDie.LogicalProcessors = dieBuilder.Result()
			node.CpuCount += cpuDie.LogicalProcessors.Size()
			node.LogicalProcessors = node.LogicalProcessors.Union(cpuDie.LogicalProcessors)
			cpuDies = append(cpuDies, cpuDie)
		}

		hasL3Cache = false
		node.CpuDies = cpuDies
		hostTopo.Nodes[i] = node
	}
	hostTopo.CPUCmtbound = desc.CPUCmtbound
	hostTopo.NumaEnabled = numaEnabled
	hostTopo.HostName = desc.Name

	desc.HostTopo = hostTopo
	//log.Infof("host topo %s", jsonutils.Marshal(hostTopo))

	sort.Sort(desc.HostTopo)
	desc.EnableCpuNumaAllocate = true
	return nil
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

func NewGuestReservedResourceByBuilder(b *HostBuilder, host *computemodels.SHost) (ret *ReservedResource) {
	ret = NewReservedResource(0, 0, 0)
	//isoDevs := b.getUnusedIsolatedDevices(host.ID)
	isoDevs := b.getIsolatedDevices(host.Id)
	if len(isoDevs) == 0 {
		return
	}
	reservedResource := host.GetDevsReservedResource(isoDevs)
	if reservedResource != nil {
		ret.CPUCount = int64(*reservedResource.ReservedCpu)
		ret.MemorySize = int64(*reservedResource.ReservedMemory)
		ret.StorageSize = int64(*reservedResource.ReservedStorage)
	}
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
		disks, _ := g.GetDisks()
		for _, disk := range disks {
			if !onlyLocal || disk.IsLocal() {
				size += disk.DiskSize
			}
		}
		return size
	}
	for _, g := range gst {
		dSize := guestDiskSize(&g, true)
		disk += int64(dSize)
		if o.Options.IgnoreNonrunningGuests && (g.Status == computeapi.VM_READY) {
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
			total += int64(storage.FreeCapacity)
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

func (h *HostDesc) GetFreeStorageSizeOfType(sType string, mediumType string, useRsvd bool, reqMaxSize int64) (int64, int64, error) {
	return h.freeStorageSizeOfType(sType, mediumType, useRsvd, reqMaxSize)
}

func (h *HostDesc) freeStorageSizeOfType(storageType string, mediumType string, useRsvd bool, reqMaxSize int64) (int64, int64, error) {
	var total int64
	var actualTotal int64
	foundLEReqStore := false
	errs := make([]error, 0)

	for _, storage := range h.Storages {
		if IsStorageBackendMediumMatch(storage, storageType, mediumType) {
			total += int64(storage.FreeCapacity)
			actualTotal += int64(storage.ActualFreeCapacity)
			if err := checkStorageSize(storage, reqMaxSize, useRsvd); err != nil {
				errs = append(errs, err)
			} else {
				foundLEReqStore = true
			}
		}
	}
	if utils.IsLocalStorage(storageType) {
		total = total + h.GuestReservedResourceUsed.StorageSize - h.GetReservedStorageSize()
		sizeSub := h.GuestReservedResource.StorageSize - h.GuestReservedResourceUsed.StorageSize
		if sizeSub < 0 {
			total += sizeSub
		}
	}
	if !foundLEReqStore {
		return 0, 0, errors.NewAggregate(errs)
	}

	if useRsvd {
		return reservedResourceAddCal(total, h.GuestReservedStorageSizeFree(), useRsvd), actualTotal, nil
	}

	return total - int64(h.GetPendingUsage().DiskUsage.Get(storageType)), actualTotal, nil
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

func (h *HostDesc) GetFreeCpuNuma() scheduler.SortedFreeNumaCpuMam {
	if !h.EnableCpuNumaAllocate {
		return nil
	}

	res := make(scheduler.SortedFreeNumaCpuMam, 0)
	cpuPin := h.GetPendingUsage().CpuPin
	numaPin := h.GetPendingUsage().NumaMemPin
	for i := range h.HostTopo.Nodes {
		nodeFree := new(scheduler.SFreeNumaCpuMem)
		nodeFree.NodeId = h.HostTopo.Nodes[i].NodeId
		nodeFree.CpuCount = h.HostTopo.Nodes[i].CpuCount
		nodeFree.MemSize = h.HostTopo.Nodes[i].NumaNodeFreeMemSizeKB / 1024
		nodeFree.EnableNumaAllocate = h.HostTopo.NumaEnabled
		nodeFree.FreeCpuCount = int(float32(h.HostTopo.Nodes[i].CpuCount)*h.CPUCmtbound) - h.HostTopo.Nodes[i].VcpuCount
		for cpuId, pending := range cpuPin {
			if h.HostTopo.Nodes[i].LogicalProcessors.Contains(cpuId) {
				nodeFree.FreeCpuCount -= pending
			}
		}

		if memSize, ok := numaPin[h.HostTopo.Nodes[i].NodeId]; ok {
			nodeFree.MemSize -= memSize
		}
		res = append(res, nodeFree)
	}
	sort.Sort(res)
	return res
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

func (h *HostDesc) AllocCpuNumaPin(vcpuCount, memSizeKB int, preferNumaNodes []int) []scheduler.SCpuNumaPin {
	if !h.EnableCpuNumaAllocate {
		return nil
	}

	hostTopo := h.HostTopo
	pendingUsage := h.GetPendingUsage()
	if len(pendingUsage.CpuPin) > 0 || len(pendingUsage.NumaMemPin) > 0 {
		hostTopo = HostTopoSubPendingUsage(h.HostTopo, pendingUsage.CpuPin, pendingUsage.NumaMemPin)
	}
	ignoreMemSingular := h.HostType == computeapi.HOST_TYPE_CONTAINER && o.Options.ContainerNumaAllocate
	return hostTopo.AllocCpuNumaNodes(vcpuCount, memSizeKB, ignoreMemSingular, preferNumaNodes)
}

func (h *HostDesc) AllocCpuNumaPinWithNodeCount(vcpuCount, memSizeKB, nodeCount int) []scheduler.SCpuNumaPin {
	if !h.EnableCpuNumaAllocate {
		return nil
	}
	hostTopo := h.HostTopo
	pendingUsage := h.GetPendingUsage()
	if len(pendingUsage.CpuPin) > 0 || len(pendingUsage.NumaMemPin) > 0 {
		hostTopo = HostTopoSubPendingUsage(h.HostTopo, pendingUsage.CpuPin, pendingUsage.NumaMemPin)
	}

	return hostTopo.AllocCpuNumaNodesWithNodeCount(vcpuCount, memSizeKB, nodeCount)
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

type HostBuilder struct {
	*baseBuilder

	residentTenantDict map[string]map[string]interface{}

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
	// isolatedDevicesDict map[string][]interface{}

	cpuIOLoads map[string]map[string]float64
}

func newHostBuilder() *HostBuilder {
	builder := new(HostBuilder)
	builder.baseBuilder = newBaseBuilder(HostDescBuilder, builder)
	return builder
}

func (b *HostBuilder) FetchHosts(ids []string) ([]computemodels.SHost, error) {
	hosts := computemodels.HostManager.Query()
	q := hosts.In("id", ids).NotEquals("host_type", computeapi.HOST_TYPE_BAREMETAL)
	hostObjs := make([]computemodels.SHost, 0)
	err := computedb.FetchModelObjects(computemodels.HostManager, q, &hostObjs)
	return hostObjs, err
}

func (b *HostBuilder) setGuests(hosts []computemodels.SHost, errMessageChannel chan error) {
	idsQuery := b.AllIDsQuery()
	guests, err := FetchGuestByHostIDsQuery(idsQuery)
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
	return newHostBuilder()
}

func (b *HostBuilder) AllIDs() ([]string, error) {
	q := computemodels.HostManager.Query("id")
	q = q.Filter(sqlchemy.NotEquals(q.Field("host_type"), computeapi.HOST_TYPE_BAREMETAL))
	return FetchModelIds(q)
}

func (b *HostBuilder) AllIDsQuery() sqlchemy.IQuery {
	q := computemodels.HostManager.Query("id")
	q = q.Filter(sqlchemy.NotEquals(q.Field("host_type"), computeapi.HOST_TYPE_BAREMETAL))
	return q
}

func (b *HostBuilder) InitFuncs() []InitFunc {
	return []InitFunc{
		// b.setSchedtags,
		b.setGuests,
	}
}

// build host desc
func (b *HostBuilder) BuildOne(host *computemodels.SHost, getter *networkGetter, baseDesc *BaseHostDesc) (interface{}, error) {
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
		//b.fillResidentGroups,
		b.fillMetadata,
		b.fillCPUIOLoads,
		b.fillGuestsCpuNumaPin,
		b.fillGuestsResourceInfo,
	}

	for _, f := range fillFuncs {
		err := f(desc, host)
		if err != nil {
			return nil, err
		}
	}

	return desc, nil
}

func (b *HostBuilder) fillGuestsCpuNumaPin(desc *HostDesc, host *computemodels.SHost) error {
	if !host.EnableNumaAllocate {
		return nil
	}

	topoObj, err := host.SysInfo.Get("topology")
	if err != nil {
		return errors.Wrap(err, "get topology from host sys_info")
	}
	hostTopo := new(hostapi.HostTopology)
	if err := topoObj.Unmarshal(hostTopo); err != nil {
		return errors.Wrap(err, "Unmarshal host topology struct")
	}
	var reservedCpus *cpuset.CPUSet
	reservedCpusStr := host.GetMetadata(context.Background(), computeapi.HOSTMETA_RESERVED_CPUS_INFO, nil)
	if reservedCpusStr != "" {
		reservedCpusJson, err := jsonutils.ParseString(reservedCpusStr)
		if err != nil {
			return errors.Wrap(err, "parse reserved cpus info failed")
		}
		reservedCpusInfo := computeapi.HostReserveCpusInput{}
		err = reservedCpusJson.Unmarshal(&reservedCpusInfo)
		if err != nil {
			return errors.Wrap(err, "unmarshal host reserved cpus info failed")
		}
		reservedCpuset, err := cpuset.Parse(reservedCpusInfo.Cpus)
		if err != nil {
			return errors.Wrap(err, "cpuset parse reserved cpus")
		}
		reservedCpus = &reservedCpuset
	}
	pinnedCpuset, err := host.GetPinnedCpusetCores(context.Background(), nil, nil)
	if err != nil {
		return err
	}
	if pinnedCpuset != nil {
		if reservedCpus == nil {
			reservedCpus = pinnedCpuset
		} else {
			newset := reservedCpus.Union(*pinnedCpuset)
			reservedCpus = &newset
		}
	}

	nodeHugepages := make([]hostapi.HostNodeHugepageNr, 0)
	if host.SysInfo.Contains("node_hugepages") {
		err = host.SysInfo.Unmarshal(&nodeHugepages, "node_hugepages")
		if err != nil {
			return errors.Wrap(err, "unmarshal node hugepages")
		}
	}

	hugepageSizeKb, err := host.SysInfo.Int("hugepage_size_kb")
	if err != nil {
		return errors.Wrap(err, "unmarshal hugepage size kb")
	}

	return b.buildHostTopo(desc, reservedCpus, int(hugepageSizeKb), nodeHugepages, hostTopo)
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
		guestsCpuNumaPin    = make([]scheduler.SCpuNumaPin, 0)
	)
	guestsOnHost, ok := b.hostGuests[host.Id]
	if !ok {
		guestsOnHost = []interface{}{}
	}
	backupGuestsOnHost, ok := b.hostBackupGuests[host.Id]
	if ok {
		guestsOnHost = append(guestsOnHost, backupGuestsOnHost...)
	}

	pendingUsage := desc.GetPendingUsage()
	desc.Tenants = make(map[string]int64)
	for _, gst := range guestsOnHost {
		guest := gst.(computemodels.SGuest)
		projectId := guest.ProjectId
		if count, ok := desc.Tenants[projectId]; ok {
			desc.Tenants[projectId] = count + 1
		} else {
			desc.Tenants[projectId] = 1
		}
		if IsGuestPendingDelete(guest) {
			memFakeDeletedSize += int64(guest.VmemSize)
			cpuFakeDeletedCount += int64(guest.VcpuCount)
		} else {
			if _, ok := pendingUsage.PendingGuestIds[guest.Id]; ok {
				log.Infof("fillGuestsResourceInfo guest %s in pending usage", guest.Id)
				continue
			}
			if IsGuestCreating(guest) {
				creatingGuestCount++
				creatingMemSize += int64(guest.VmemSize)
				creatingCPUCount += int64(guest.VcpuCount)
				if host.EnableNumaAllocate && guest.CpuNumaPin != nil {
					cpuNumaPin := make([]scheduler.SCpuNumaPin, 0)
					if err := guest.CpuNumaPin.Unmarshal(&cpuNumaPin); err != nil {
						return errors.Wrap(err, "unmarshal cpu numa pin")
					}
					for i := range cpuNumaPin {
						if cpuNumaPin[i].ExtraCpuCount > 0 {
							creatingCPUCount += int64(cpuNumaPin[i].ExtraCpuCount)
						}
					}
					guestsCpuNumaPin = append(guestsCpuNumaPin, cpuNumaPin...)
				}
			} else if !IsGuestStoppedStatus(guest) {
				// running status
				runningCount++
				memSize += int64(guest.VmemSize)
				cpuCount += int64(guest.VcpuCount)
				if host.EnableNumaAllocate && guest.CpuNumaPin != nil {
					cpuNumaPin := make([]scheduler.SCpuNumaPin, 0)
					if err := guest.CpuNumaPin.Unmarshal(&cpuNumaPin); err != nil {
						return errors.Wrap(err, "unmarshal cpu numa pin")
					}
					for i := range cpuNumaPin {
						if cpuNumaPin[i].ExtraCpuCount > 0 {
							creatingCPUCount += int64(cpuNumaPin[i].ExtraCpuCount)
						}
					}
					guestsCpuNumaPin = append(guestsCpuNumaPin, cpuNumaPin...)
				}
			}
		}
		//if IsGuestRunning(guest) {
		//	runningCount++
		//	memSize += int64(guest.VmemSize)
		//	cpuCount += int64(guest.VcpuCount)
		//	if host.EnableNumaAllocate && guest.CpuNumaPin != nil {
		//		cpuNumaPin := make([]scheduler.SCpuNumaPin, 0)
		//		if err := guest.CpuNumaPin.Unmarshal(&cpuNumaPin); err != nil {
		//			return errors.Wrap(err, "unmarshal cpu numa pin")
		//		}
		//		guestsCpuNumaPin = append(guestsCpuNumaPin, cpuNumaPin...)
		//	}
		//} else if IsGuestCreating(guest) {
		//	creatingGuestCount++
		//	creatingMemSize += int64(guest.VmemSize)
		//	creatingCPUCount += int64(guest.VcpuCount)
		//	if host.EnableNumaAllocate && guest.CpuNumaPin != nil {
		//		cpuNumaPin := make([]scheduler.SCpuNumaPin, 0)
		//		if err := guest.CpuNumaPin.Unmarshal(&cpuNumaPin); err != nil {
		//			return errors.Wrap(err, "unmarshal cpu numa pin")
		//		}
		//		guestsCpuNumaPin = append(guestsCpuNumaPin, cpuNumaPin...)
		//	}
		//} else if IsGuestPendingDelete(guest) {
		//	memFakeDeletedSize += int64(guest.VmemSize)
		//	cpuFakeDeletedCount += int64(guest.VcpuCount)
		//}

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

	if host.EnableNumaAllocate && len(guestsCpuNumaPin) > 0 {
		desc.HostTopo.LoadCpuNumaPin(guestsCpuNumaPin)
	}
	//log.Infof("host %s topo %s", desc.Name, jsonutils.Marshal(desc.HostTopo))

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
	if o.Options.IgnoreNonrunningGuests {
		memFreeSize = desc.TotalMemSize - desc.RunningMemSize - desc.CreatingMemSize
		cpuFreeCount = desc.TotalCPUCount - desc.RunningCPUCount - desc.CreatingCPUCount
	} else {
		memFreeSize = desc.TotalMemSize - desc.RequiredMemSize
		cpuFreeCount = desc.TotalCPUCount - desc.RequiredCPUCount
		if o.Options.IgnoreFakeDeletedGuests {
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
	metadata, err := host.GetAllMetadata(nil, nil)
	if err != nil {
		log.Errorf("Get host %s metadata: %v", desc.GetId(), err)
		return nil
	}
	desc.Metadata = metadata
	return nil
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
