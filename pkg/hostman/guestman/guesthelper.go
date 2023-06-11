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
	"container/heap"
	"sync"

	"yunion.io/x/cloudmux/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/cgrouputils/cpuset"
)

type SBaseParms struct {
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
}

type SReloadDisk struct {
	Sid  string
	Disk storageman.IDisk
}

type SDiskSnapshot struct {
	UserCred   mcclient.TokenCredential
	Sid        string
	SnapshotId string
	Disk       storageman.IDisk
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
	PendingDelete   bool
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
	Sid  string
	BPS  int64
	IOPS int64
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

type SQgaGuestSetPassword struct {
	*hostapi.GuestSetPasswordRequest
	Sid string
}

type CpuSetCounter struct {
	Nodes []*NumaNode
	Lock  sync.Mutex
}

func NewGuestCpuSetCounter(info *hostapi.HostTopology, reservedCpus *cpuset.CPUSet) *CpuSetCounter {
	log.Infof("NewGuestCpuSetCounter from topo: %s", jsonutils.Marshal(info))
	cpuSetCounter := new(CpuSetCounter)
	cpuSetCounter.Nodes = make([]*NumaNode, len(info.Nodes))
	hasL3Cache := false
	for i := 0; i < len(info.Nodes); i++ {
		node := new(NumaNode)
		node.LogicalProcessors = cpuset.NewCPUSet()
		node.NodeId = info.Nodes[i].ID
		cpuDies := make([]*CPUDie, 0)
		for j := 0; j < len(info.Nodes[i].Caches); j++ {
			if info.Nodes[i].Caches[j].Level != 3 {
				continue
			}
			hasL3Cache = true
			cpuDie := new(CPUDie)
			dieBuilder := cpuset.NewBuilder()
			for k := 0; k < len(info.Nodes[i].Caches[j].LogicalProcessors); k++ {
				if reservedCpus != nil && reservedCpus.Contains(int(info.Nodes[i].Caches[j].LogicalProcessors[k])) {
					continue
				}
				dieBuilder.Add(int(info.Nodes[i].Caches[j].LogicalProcessors[k]))
			}
			cpuDie.LogicalProcessors = dieBuilder.Result()
			node.CpuCount += cpuDie.LogicalProcessors.Size()
			node.LogicalProcessors = node.LogicalProcessors.Union(cpuDie.LogicalProcessors)
			cpuDies = append(cpuDies, cpuDie)
		}
		if !hasL3Cache {
			cpuDie := new(CPUDie)
			dieBuilder := cpuset.NewBuilder()
			for j := 0; j < len(info.Nodes[i].Cores); j++ {
				for k := 0; k < len(info.Nodes[i].Cores[j].LogicalProcessors); k++ {
					if reservedCpus != nil && reservedCpus.Contains(int(info.Nodes[i].Cores[j].LogicalProcessors[k])) {
						continue
					}
					dieBuilder.Add(int(info.Nodes[i].Cores[j].LogicalProcessors[k]))
				}
			}
			cpuDie.LogicalProcessors = dieBuilder.Result()
			node.CpuCount += cpuDie.LogicalProcessors.Size()
			node.LogicalProcessors = node.LogicalProcessors.Union(cpuDie.LogicalProcessors)
			cpuDies = append(cpuDies, cpuDie)
		}

		hasL3Cache = false
		node.CpuDies = cpuDies
		cpuSetCounter.Nodes[i] = node
	}
	heap.Init(cpuSetCounter)
	return cpuSetCounter
}

func (pq *CpuSetCounter) AllocCpuset(vcpuCount int) map[int][]int {
	res := map[int][]int{}
	sourceVcpuCount := vcpuCount
	pq.Lock.Lock()
	defer pq.Lock.Unlock()
	for vcpuCount > 0 {
		count := vcpuCount
		if vcpuCount > pq.Nodes[0].CpuCount {
			count = vcpuCount/2 + vcpuCount%2
		}
		res[pq.Nodes[0].NodeId] = pq.Nodes[0].AllocCpuset(count)
		pq.Nodes[0].VcpuCount += sourceVcpuCount
		heap.Fix(pq, 0)
		vcpuCount -= count
	}
	return res
}

func (pq *CpuSetCounter) ReleaseCpus(cpus []int, vcpuCount int) {
	pq.Lock.Lock()
	defer pq.Lock.Unlock()
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
			heap.Fix(pq, i)
		}
	}
}

func (pq *CpuSetCounter) LoadCpus(cpus []int, vcpuCpunt int) {
	pq.Lock.Lock()
	defer pq.Lock.Unlock()
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
			heap.Fix(pq, i)
		}
	}
}

func (pq CpuSetCounter) Len() int { return len(pq.Nodes) }

func (pq CpuSetCounter) Less(i, j int) bool {
	return pq.Nodes[i].VcpuCount < pq.Nodes[j].VcpuCount
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
	NodeId            int
}

func (n *NumaNode) AllocCpuset(vcpuCount int) []int {
	cpus := make([]int, 0)
	for vcpuCount > 0 {
		dies := n.CpuDies
		count := vcpuCount
		if vcpuCount > dies[0].LogicalProcessors.Size() {
			count = dies[0].LogicalProcessors.Size()
		}
		dies[0].VcpuCount += count
		heap.Fix(&n.CpuDies, 0)
		vcpuCount -= count
		cpus = append(cpus, dies[0].LogicalProcessors.ToSliceNoSort()...)
	}
	return cpus
}

type CPUDie struct {
	LogicalProcessors cpuset.CPUSet
	VcpuCount         int
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
			(*pq)[i].VcpuCount -= vcpuCount
			heap.Fix(pq, i)
		}
	}
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
		if _, ok := cpuDies[i]; ok {
			(*pq)[i].VcpuCount += vcpuCount
			heap.Fix(pq, i)
		}
	}
}
