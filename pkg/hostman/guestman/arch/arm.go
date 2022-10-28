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
	ARM_MAX_CPUS = 64
	ARM_SOCKETS  = 2
	ARM_CORES    = 32
	ARM_THREADS  = 1

	ARM_MEM_DEFAULT_SLOTS = 4
	ARM_MAX_MEM_MB        = 262144
)

type ARM struct {
	archBase
}

// -device scsi-cd,drive=cd0,share-rw=true
// if=none,file=%s,id=cd0,media=cdrom
func (*ARM) GenerateCdromDesc(osName string, cdrom *desc.SGuestCdrom) {
	scsiDev := desc.NewScsiDevice("", "scsi-cd", "scsi0-cd0")
	scsiDev.Options = map[string]string{"share-rw": "true"}
	driveOptions := map[string]string{
		"if":    "none",
		"media": "cdrom",
	}
	cdrom.Scsi = scsiDev
	cdrom.DriveOptions = driveOptions
	cdrom.Id = "cd0"
}

func (*ARM) GenerateFloppyDesc(osName string, floppy *desc.SGuestFloppy) {

}

func (*ARM) GenerateMachineDesc(accel string) *desc.SGuestMachine {
	gicVersion := "max"
	return &desc.SGuestMachine{
		Accel:      accel,
		GicVersion: &gicVersion,
	}
}

func (*ARM) GenerateMemDesc() *desc.SGuestMem {
	return &desc.SGuestMem{
		Slots:  ARM_MEM_DEFAULT_SLOTS,
		MaxMem: ARM_MAX_MEM_MB,
	}
}

func (*ARM) GenerateCpuDesc(cpus uint, s KVMGuestInstance) (*desc.SGuestCpu, error) {
	cpuMax, err := s.CpuMax()
	if err != nil {
		return nil, err
	}

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
		Sockets: ARM_SOCKETS,
		Cores:   cpuMax / ARM_SOCKETS / ARM_THREADS,
		Threads: ARM_THREADS,
		MaxCpus: cpuMax,
		Model:   cpuType,
		Accel:   accel,
	}, nil
}
