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
	Arch_x86_64  string = "x86_64"
	Arch_aarch64 string = "aarch64"
	Arch_riscv64 string = "riscv64"
)

type Arch interface {
	GenerateCpuDesc(cpus uint, cpuMax uint, s KVMGuestInstance) (*desc.SGuestCpu, error)
	GenerateMemDesc() *desc.SGuestMem
	GenerateMachineDesc(accel string) *desc.SGuestMachine
	GenerateCdromDesc(osName string, cdrom *desc.SGuestCdrom)
	GenerateFloppyDesc(osName string, floppy *desc.SGuestFloppy)
	GenerateQgaDesc(qgaPath string) *desc.SGuestQga
	GeneratePvpanicDesc() *desc.SGuestPvpanic
	GenerateIsaSerialDesc() *desc.SGuestIsaSerial
}

type KVMGuestInstance interface {
	IsOldWindows() bool
	GetOsName() string

	GetKernelVersion() string
	CpuMax() (uint, error)
	IsNestedVirt() bool
	IsKvmSupport() bool
	HideKVM() bool
	HideHypervisor() bool
}

func NewArch(arch string) Arch {
	switch arch {
	case Arch_x86_64:
		return &X86{}
	case Arch_aarch64:
		return &ARM{}
	case Arch_riscv64:
		return &RISCV{}
	}
	return nil
}

type archBase struct {
}

func (*archBase) GenerateQgaDesc(qgaPath string) *desc.SGuestQga {
	charDev := "qga0"
	socket := &desc.CharDev{
		Backend: "socket",
		Id:      charDev,
		Options: map[string]string{
			"path":   qgaPath,
			"server": "",
			"nowait": "",
		},
	}

	serialPort := &desc.VirtSerialPort{
		Chardev: charDev,
		Name:    "org.qemu.guest_agent.0",
	}

	return &desc.SGuestQga{
		Socket:     socket,
		SerialPort: serialPort,
	}
}

func (*archBase) GeneratePvpanicDesc() *desc.SGuestPvpanic {
	return nil
}

func (*archBase) GenerateIsaSerialDesc() *desc.SGuestIsaSerial {
	return nil
}

type otherArchBase struct {
}

// -device scsi-cd,drive=cd0,share-rw=true
// if=none,file=%s,id=cd0,media=cdrom
func (*otherArchBase) GenerateCdromDesc(osName string, cdrom *desc.SGuestCdrom) {
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

func (*otherArchBase) GenerateFloppyDesc(osName string, floppy *desc.SGuestFloppy) {
}
