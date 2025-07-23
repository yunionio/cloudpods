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
	"fmt"

	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
)

const (
	LOONGARCH64_MAX_CPUS = 8
	LOONGARCH64_SOCKETS  = 1
	LOONGARCH64_CORES    = 8
	LOONGARCH64_THREADS  = 1

	LOONGARCH64_MEM_DEFAULT_SLOTS = 4
	LOONGARCH64_MAX_MEM_MB        = 262144
)

type LOONGARCH64 struct {
	archBase
}

// -device scsi-cd,drive=cd0,share-rw=true
// if=none,file=%s,id=cd0,media=cdrom
func (*LOONGARCH64) GenerateCdromDesc(osName string, cdrom *desc.SGuestCdrom) {
	id := fmt.Sprintf("scsi%d-cd0", cdrom.Ordinal)
	scsiDev := desc.NewScsiDevice("", "scsi-cd", id)
	scsiDev.Options = map[string]string{"share-rw": "true"}
	driveOptions := map[string]string{
		"if":    "none",
		"media": "cdrom",
	}
	cdrom.Scsi = scsiDev
	cdrom.DriveOptions = driveOptions
	cdrom.Id = id
}

func (*LOONGARCH64) GenerateFloppyDesc(osName string, floppy *desc.SGuestFloppy) {

}

func (*LOONGARCH64) GenerateMachineDesc(accel string) *desc.SGuestMachine {
	return &desc.SGuestMachine{
		Accel: accel,
	}
}

func (*LOONGARCH64) GenerateMemDesc() *desc.SGuestMem {
	return &desc.SGuestMem{
		Slots:  LOONGARCH64_MEM_DEFAULT_SLOTS,
		MaxMem: LOONGARCH64_MAX_MEM_MB,
	}
}

func (*LOONGARCH64) GenerateCpuDesc(cpus uint, cpuMax uint, s KVMGuestInstance) (*desc.SGuestCpu, error) {
	var accel, cpuType string
	if s.IsKvmSupport() {
		accel = "kvm"

		// * under KVM, -cpu max is the same as -cpu host
		// * under TCG, -cpu max means "emulate with as many features as possible"
		cpuType = "max"
	} else {
		accel = "tcg"
		cpuType = "max"
	}
	return &desc.SGuestCpu{
		Cpus:  cpus,
		Model: cpuType,
		Accel: accel,
	}, nil
}
