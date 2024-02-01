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

package api

import (
	"fmt"

	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/compute/models"
)

const (
	HostTypeHost         = "host"
	HostTypeBaremetal    = "baremetal"
	SchedTypeBaremetal   = "baremetal"
	SchedTypeKvm         = "kvm"
	HostHypervisorForKvm = "hypervisor"

	AggregateStrategyRequire = "require"
	AggregateStrategyExclude = "exclude"
	AggregateStrategyPrefer  = "prefer"
	AggregateStrategyAvoid   = "avoid"

	// passthrough device type
	DIRECT_PCI_TYPE = "PCI"
	GPU_HPC_TYPE    = "GPU-HPC"
	GPU_VGA_TYPE    = "GPU-VGA"
	USB_TYPE        = "USB"
	NIC_TYPE        = "NIC"

	// Hard code vendor const
	NVIDIA           = "NVIDIA"
	AMD              = "AMD"
	NVIDIA_VENDOR_ID = "10de"
	AMD_VENDOR_ID    = "1002"
)

var (
	AggregateStrategySets = sets.NewString(
		AggregateStrategyRequire,
		AggregateStrategyExclude,
		AggregateStrategyPrefer,
		AggregateStrategyAvoid,
	)

	ValidGpuTypes = sets.NewString(
		GPU_HPC_TYPE,
		GPU_VGA_TYPE,
	)

	ValidPassthroughTypes = sets.NewString(
		DIRECT_PCI_TYPE,
		USB_TYPE,
		NIC_TYPE,
	).Union(ValidGpuTypes)

	IsolatedVendorIDMap = map[string]string{
		NVIDIA: NVIDIA_VENDOR_ID,
		AMD:    AMD_VENDOR_ID,
	}

	IsolatedIDVendorMap = map[string]string{}
)

func init() {
	for k, v := range IsolatedVendorIDMap {
		IsolatedIDVendorMap[v] = k
	}
}

func SchedtagStrategyCheck(strategy string) (err error) {
	if !AggregateStrategySets.Has(strategy) {
		err = fmt.Errorf("Strategy %q must in set %v", strategy, AggregateStrategySets.List())
	}
	return
}

type CandidateStorage struct {
	*models.SStorage
	FreeCapacity       int64 `json:"free_capacity"`
	ActualFreeCapacity int64 `json:"actual_free_capacity"`
}

type CandidateNetwork struct {
	*models.SNetwork
	FreePort int `json:"free_port"`

	Provider string
	VpcId    string
}

type CandidateGroup struct {
	*models.SGroup
	ReferCount int
}
