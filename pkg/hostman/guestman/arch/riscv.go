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

package arch

import (
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

const (
	RISCV_MAX_CPUS = 128
	RISCV_SOCKETS  = 1
	RISCV_THREADS  = 1

	RISCV_MEM_DEFAULT_SLOTS = 4
)

var RISCV_MAX_MEM_MB uint = 262144

type RISCV struct {
	archBase
	otherArchBase
}

func (*RISCV) GenerateMachineDesc(accel string) *desc.SGuestMachine {
	return &desc.SGuestMachine{
		Accel: accel,
	}
}

func (*RISCV) GenerateMemDesc() *desc.SGuestMem {
	return &desc.SGuestMem{
		Slots:  RISCV_MEM_DEFAULT_SLOTS,
		MaxMem: RISCV_MAX_MEM_MB,
	}
}

func (*RISCV) GenerateCpuDesc(cpus uint, cpuMax uint, s KVMGuestInstance) (*desc.SGuestCpu, error) {
	var hostCPUPassthrough = options.HostOptions.HostCpuPassthrough
	var accel, cpuType string
	if s.IsKvmSupport() {
		accel = "kvm"
		if hostCPUPassthrough {
			cpuType = "host"
		} else {
			// * under KVM, -cpu max is the same as -cpu host
			// * under TCG, -cpu max means "emulate with as many features as possible"
			cpuType = "max"
		}
	} else {
		accel = "tcg"
		cpuType = "max"
	}
	return &desc.SGuestCpu{
		Cpus:    cpus,
		Sockets: RISCV_SOCKETS,
		Cores:   cpuMax / RISCV_SOCKETS / RISCV_THREADS,
		Threads: RISCV_THREADS,
		MaxCpus: cpuMax,
		Model:   cpuType,
		Accel:   accel,
	}, nil
}
