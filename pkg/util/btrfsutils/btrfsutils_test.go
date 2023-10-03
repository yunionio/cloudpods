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

package btrfsutils

import (
	"reflect"
	"testing"
)

const (
	lines = `ID 256 gen 32 top level 5 path @
ID 257 gen 872 top level 256 path @/var
ID 258 gen 869 top level 256 path @/usr/local
ID 259 gen 872 top level 256 path @/tmp
ID 260 gen 35 top level 256 path @/srv
ID 261 gen 872 top level 256 path @/root
ID 262 gen 871 top level 256 path @/opt
ID 263 gen 35 top level 256 path @/home
ID 264 gen 26 top level 256 path @/boot/grub2/x86_64-efi
ID 265 gen 64 top level 256 path @/boot/grub2/i386-pc
ID 266 gen 66 top level 256 path @/.snapshots
ID 267 gen 873 top level 266 path @/.snapshots/1/snapshot
ID 272 gen 65 top level 266 path @/.snapshots/2/snapshot
`
)

func TestParseBtrfsSubvols(t *testing.T) {
	subvols := parseBtrfsSubvols(lines)
	want := []string{"@/var", "@/usr/local", "@/tmp", "@/srv", "@/root", "@/opt", "@/home",
		"@/boot/grub2/x86_64-efi", "@/boot/grub2/i386-pc", "@/.snapshots",
		"@/.snapshots/1/snapshot", "@/.snapshots/2/snapshot",
	}
	if !reflect.DeepEqual(subvols, want) {
		t.Errorf("expect %s got %s", want, subvols)
	}
}
