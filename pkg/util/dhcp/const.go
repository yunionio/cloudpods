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

package dhcp

const (
	// DHCP client architecture values
	// - https://datatracker.ietf.org/doc/html/rfc4578#section-2.1
	// - https://github.com/ipxe/ipxe/blob/236299baa32452c79a59138c44eca5fcf4a918f9/src/include/ipxe/dhcp.h#L275-L305
	// 0. Intel x86 PC
	CLIENT_ARCH_INTEL_X86PC = iota
	// 1. NEC/PC98
	CLIENT_ARCH_NEC_PC98
	// 2. EFI Itenium
	CLIENT_ARCH_EFI_ITANIUM
	// 3. DEC alpha
	CLIENT_ARCH_DEC_ALPHA
	// 4. Arc x86
	CLIENT_ARCH_ARC_X86
	// 5. Intel Lean Client
	CLIENT_ARCH_INTEL_LEAN_CLIENT
	// 6. EFI IA32
	CLIENT_ARCH_EFI_IA32
	// 7. EFI BC
	CLIENT_ARCH_EFI_BC
	// 8. EFI Xscale
	CLIENT_ARCH_EFI_XSCALE
	// 9. EFI x86_64
	CLIENT_ARCH_EFI_X86_64
	// 10. EFI 32-bit ARM
	CLIENT_ARCH_EFI_ARM32
	// 11. EFI 64-bit ARM
	CLIENT_ARCH_EFI_ARM64
)

const (
	icmpRAFakePort = int(-1111)
)

func IsUEFIPxeArch(arch uint16) bool {
	switch arch {
	case CLIENT_ARCH_EFI_IA32:
		return true
	case CLIENT_ARCH_EFI_BC, CLIENT_ARCH_EFI_XSCALE, CLIENT_ARCH_EFI_X86_64:
		return true
	case CLIENT_ARCH_EFI_ARM32, CLIENT_ARCH_EFI_ARM64:
		return true
	}
	return false
}
