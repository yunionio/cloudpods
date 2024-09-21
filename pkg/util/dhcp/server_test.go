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

import (
	"testing"

	"golang.org/x/net/bpf"
)

func TestBpfDisemble(t *testing.T) {
	dhcpServerPort := 67
	dhcpRelayPort := 68
	bpf := []bpf.RawInstruction{
		{Op: 0x28, Jt: 0, Jf: 0, K: 0x0000000c},
		{Op: 0x15, Jt: 0, Jf: 13, K: 0x00000800},
		{Op: 0x30, Jt: 0, Jf: 0, K: 0x00000017},
		{Op: 0x15, Jt: 0, Jf: 11, K: 0x00000011},
		{Op: 0x28, Jt: 0, Jf: 0, K: 0x00000014},
		{Op: 0x45, Jt: 9, Jf: 0, K: 0x00001fff},
		{Op: 0xb1, Jt: 0, Jf: 0, K: 0x0000000e},
		{Op: 0x48, Jt: 0, Jf: 0, K: 0x0000000e},
		{Op: 0x15, Jt: 0, Jf: 2, K: uint32(dhcpServerPort)},
		{Op: 0x48, Jt: 0, Jf: 0, K: 0x00000010},
		{Op: 0x15, Jt: 3, Jf: 4, K: uint32(dhcpRelayPort)},
		{Op: 0x15, Jt: 0, Jf: 3, K: uint32(dhcpRelayPort)},
		{Op: 0x48, Jt: 0, Jf: 0, K: 0x00000010},
		{Op: 0x15, Jt: 0, Jf: 1, K: uint32(dhcpServerPort)},
		{Op: 0x6, Jt: 0, Jf: 0, K: 0x00040000},
		{Op: 0x6, Jt: 0, Jf: 0, K: 0x00000000},
	}
	for i := range bpf {
		inst := bpf[i].Disassemble()
		t.Logf("[%d] %#v", i, inst)
	}
}
