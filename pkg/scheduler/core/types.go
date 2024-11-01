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

package core

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core/score"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/sku"
	schedmodels "yunion.io/x/onecloud/pkg/scheduler/models"
)

const (
	PriorityStep int = 1
)

var ErrInstanceGroupNotFound = errors.Error("InstanceGroupNotFound")

type FailedCandidate struct {
	Stage     string
	Candidate Candidater
	Reasons   []PredicateFailureReason
}

type FailedCandidates struct {
	Candidates []FailedCandidate
}

type SelectPlugin interface {
	Name() string
	OnPriorityEnd(*Unit, Candidater)
	OnSelectEnd(u *Unit, c Candidater, count int64)
}

type Kind int

const (
	KindFree Kind = iota
	KindRaw
	KindReserved
)

type CandidatePropertyGetter interface {
	Id() string
	Name() string
	Zone() *computemodels.SZone
	Host() *computemodels.SHost
	Cloudprovider() *computemodels.SCloudprovider
	IsPublic() bool
	DomainId() string
	PublicScope() string
	SharedDomains() []string
	Region() *computemodels.SCloudregion
	HostType() string
	Sku(string) *sku.ServerSku
	Storages() []*api.CandidateStorage
	Networks() []*api.CandidateNetwork
	OvnCapable() bool
	Status() string
	HostStatus() string
	Enabled() bool
	IsEmpty() bool
	ResourceType() string
	NetInterfaces() map[string][]computemodels.SNetInterface
	ProjectGuests() map[string]int64
	CreatingGuestCount() int

	CPUArch() string
	IsArmHost() bool
	RunningCPUCount() int64
	TotalCPUCount(useRsvd bool) int64
	FreeCPUCount(useRsvd bool) int64

	RunningMemorySize() int64
	TotalMemorySize(useRsvd bool) int64
	FreeMemorySize(useRsvd bool) int64
	GetFreeCpuNuma() []*schedapi.SFreeNumaCpuMem
	NumaAllocateEnabled() bool

	StorageInfo() []*baremetal.BaremetalStorage
	GetFreeStorageSizeOfType(storageType string, mediumType string, useRsvd bool, reqMaxSize int64) (int64, int64, error)

	GetFreePort(netId string) int

	InstanceGroups() map[string]*api.CandidateGroup
	GetFreeGroupCount(groupId string) (int, error)

	GetAllClassMetadata() (map[string]string, error)

	GetIpmiInfo() types.SIPMIInfo

	GetNics() []*types.SNic

	GetQuotaKeys(s *api.SchedInfo) computemodels.SComputeResourceKeys

	GetPendingUsage() *schedmodels.SPendingUsage

	// isloatedDevices
	UnusedIsolatedDevices() []*IsolatedDeviceDesc
	UnusedIsolatedDevicesByType(devType string) []*IsolatedDeviceDesc
	UnusedIsolatedDevicesByVendorModel(vendorModel string) []*IsolatedDeviceDesc
	UnusedIsolatedDevicesByModel(model string) []*IsolatedDeviceDesc
	UnusedIsolatedDevicesByModelAndWire(model, wire string) []*IsolatedDeviceDesc
	UnusedIsolatedDevicesByDevicePath(devPath string) []*IsolatedDeviceDesc
	GetIsolatedDevice(devID string) *IsolatedDeviceDesc
	UnusedGpuDevices() []*IsolatedDeviceDesc
	GetIsolatedDevices() []*IsolatedDeviceDesc

	db.IResource
}

// Candidater replace host Candidate resource info
type Candidater interface {
	Getter() CandidatePropertyGetter
	// IndexKey return candidate cache item's ident, usually host ID
	IndexKey() string
	Type() int

	GetSchedDesc() *jsonutils.JSONDict
	GetGuestCount() int64
	GetResourceType() string
	AllocCpuNumaPin(vcpuCount, memSizeKB int, preferNumaNodes []int) []schedapi.SCpuNumaPin
	AllocCpuNumaPinWithNodeCount(vcpuCount, memSizeKB, nodeCount int) []schedapi.SCpuNumaPin
}

// HostPriority represents the priority of scheduling to particular host, higher priority is better.
type HostPriority struct {
	// Name of the host
	Host string
	// Score associated with the host
	Score Score
	// Resource wraps Candidate host info
	Candidate Candidater
}

type HostPriorityList []HostPriority

func (h HostPriorityList) Len() int {
	return len(h)
}

func (h HostPriorityList) compareNormalScore(i, j int) bool {
	s1 := h[i].Score.ScoreBucket
	s2 := h[j].Score.ScoreBucket
	normalScorei, normalScorej := s1.NormalScore(), s2.NormalScore()
	if normalScorei == normalScorej {
		return h[i].Host < h[j].Host
	}
	return normalScorei < normalScorej
}

func (h HostPriorityList) Less(i, j int) bool {
	si := h[i].Score.ScoreBucket
	sj := h[j].Score.ScoreBucket

	if si.PreferScore() > 0 || sj.PreferScore() > 0 {
		if si.PreferScore() != 0 && sj.PreferScore() != 0 {
			// both have prefer tags
			return h.compareNormalScore(i, j)
		}
		if si.PreferScore() <= 0 {
			return true
		}
		if sj.PreferScore() <= 0 {
			return false
		}
	}

	if si.AvoidScore() > 0 || sj.AvoidScore() > 0 {
		if si.AvoidScore() != 0 && sj.AvoidScore() != 0 {
			// both have avoid tags
			return h.compareNormalScore(i, j)
		}
		if si.AvoidScore() > 0 {
			return true
		}
		if sj.AvoidScore() > 0 {
			return false
		}
	}

	return h.compareNormalScore(i, j)
}

func (h HostPriorityList) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

type FitPredicate interface {
	// Get filter's name
	Name() string
	Clone() FitPredicate
	PreExecute(context.Context, *Unit, []Candidater) (bool, error)
	Execute(context.Context, *Unit, Candidater) (bool, []PredicateFailureReason, error)
}

type PredicateFailureError interface {
	GetReason() string
}

type PredicateFailureReason interface {
	PredicateFailureError
	GetType() string
}

type PriorityPreFunction func(*Unit, []Candidater) (bool, []PredicateFailureReason, error)

// PriorityMapFunction is a function that computes per-resource results for a given resource.
type PriorityMapFunction func(*Unit, Candidater) (HostPriority, error)

// PriorityReduceFunction is a function that aggregated per-resource results and computes
// final scores for all hosts.
type PriorityReduceFunction func(*Unit, []Candidater, HostPriorityList) error

type PriorityConfig struct {
	Name   string
	Pre    PriorityPreFunction
	Map    PriorityMapFunction
	Reduce PriorityReduceFunction
	Weight int
}

type Priority interface {
	Name() string
	Clone() Priority
	Map(*Unit, Candidater) (HostPriority, error)
	Reduce(*Unit, []Candidater, HostPriorityList) error
	PreExecute(*Unit, []Candidater) (bool, []PredicateFailureReason, error)

	// Score intervals
	ScoreIntervals() score.Intervals
}

type AllocatedResource struct {
	Disks []*schedapi.CandidateDiskV2 `json:"disks"`
	Nets  []*schedapi.CandidateNet    `json:"nets"`
}

func NewAllocatedResource() *AllocatedResource {
	return &AllocatedResource{
		Disks: make([]*schedapi.CandidateDiskV2, 0),
		Nets:  make([]*schedapi.CandidateNet, 0),
	}
}

type IsolatedDeviceDesc struct {
	ID             string
	GuestID        string
	HostID         string
	DevType        string
	Model          string
	Addr           string
	VendorDeviceID string
	WireId         string
	DevicePath     string
}

func (i *IsolatedDeviceDesc) VendorID() string {
	return strings.Split(i.VendorDeviceID, ":")[0]
}

func (i *IsolatedDeviceDesc) GetVendorModel() *VendorModel {
	return &VendorModel{
		Vendor: i.VendorID(),
		Model:  i.Model,
	}
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
