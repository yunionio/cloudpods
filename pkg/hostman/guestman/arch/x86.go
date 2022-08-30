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
	"yunion.io/x/onecloud/pkg/hostman/guestman/qemu"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

const (
	X86_MAX_CPUS = 128
	X86_SOCKETS  = 2
	X86_CORES    = 64
	X86_THREADS  = 1

	X86_MEM_DEFAULT_SLOTS = 4
	X86_MAX_MEM_MB        = 524288
)

type X86 struct {
	archBase
}

func (*X86) GeneratePvpanicDesc() *desc.SGuestPvpanic {
	return &desc.SGuestPvpanic{
		Ioport: 1285, //default ioport
		Id:     "pvpanic",
	}
}

func (*X86) GenerateIsaSerialDesc() *desc.SGuestIsaSerial {
	return &desc.SGuestIsaSerial{
		Pty: desc.NewCharDev("pty", "charserial0", ""),
		Id:  "serial0",
	}
}

func (*X86) GenerateCdromDesc(osName string, cdrom *desc.SGuestCdrom) {
	var id, devType string
	var driveOpts map[string]string

	switch osName {
	case qemu.OS_NAME_MACOS:
		id = "MacDVD"
		devType = "ide-drive"
		driveOpts = map[string]string{
			"if":       "none",
			"snapshot": "on",
		}
	default:
		id = "ide0-cd0"
		devType = "ide-cd"
		driveOpts = map[string]string{
			"if":    "none",
			"media": "cdrom",
		}
	}

	cdrom.Ide = &desc.IDEDevice{
		DevType: devType,
	}
	cdrom.DriveOptions = driveOpts
	cdrom.Id = id
}

func (*X86) GenerateMachineDesc(accel string) *desc.SGuestMachine {
	return &desc.SGuestMachine{
		Accel: accel,
	}
}

func (*X86) GenerateMemDesc() *desc.SGuestMem {
	return &desc.SGuestMem{
		Slots:  X86_MEM_DEFAULT_SLOTS,
		MaxMem: X86_MAX_MEM_MB,
	}
}

func (*X86) GenerateCpuDesc(cpus uint, osName string, enableKVM, hideKVM bool) *desc.SGuestCpu {
	var hostCPUPassthrough = options.HostOptions.HostCpuPassthrough
	var isCPUIntel = sysutils.IsProcessorIntel()
	var isCPUAMD = sysutils.IsProcessorAmd()

	var accel, cpuType, vendor, level string
	var features = make(map[string]bool, 0)
	if enableKVM {
		accel = "kvm"
		if osName == qemu.OS_NAME_MACOS {
			cpuType = "Penryn"
			vendor = "GenuineIntel"
		} else if hostCPUPassthrough {
			cpuType = "host"
			// https://unix.stackexchange.com/questions/216925/nmi-received-for-unknown-reason-20-do-you-have-a-strange-power-saving-mode-ena
			features["kvm_pv_eoi"] = true
		} else {
			cpuType = "qemu64"
			features["kvm_pv_eoi"] = true
			if isCPUIntel {
				for _, feat := range []string{
					"vmx", "ssse3", "sse4.1", "sse4.2", "aes", "avx",
					"vme", "pat", "ss", "pclmulqdq", "xsave",
				} {
					features[feat] = true
				}
				features["x2apic"] = false
				level = "13"
			} else if isCPUAMD {
				features["svm"] = true
			}
		}

		if !hideKVM {
			features["kvm"] = false
		}
	} else {
		accel = "tcg"
		cpuType = "qemu64"
	}
	return &desc.SGuestCpu{
		Cpus:     cpus,
		Sockets:  X86_SOCKETS,
		Cores:    X86_CORES,
		Threads:  X86_THREADS,
		MaxCpus:  X86_MAX_CPUS,
		Model:    cpuType,
		Vendor:   vendor,
		Level:    level,
		Features: features,
		Accel:    accel,
	}
}
