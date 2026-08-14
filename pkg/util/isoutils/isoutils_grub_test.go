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

package isoutils

import (
	"testing"

	"yunion.io/x/pkg/util/imagetools"
)

func TestGetOsInfoByGrubIgnoresClassFedora(t *testing.T) {
	content := `
set default="1"

function load_video {
  insmod efi_gop
  insmod efi_uga
  insmod video_bochs
  insmod video_cirrus
  insmod all_video
}

load_video
set gfxpayload=keep
insmod gzio
insmod part_gpt
insmod ext2

set timeout=60
### END /etc/grub.d/00_header ###

search --no-floppy --set=root -l 'Kylin-Server-10'

### BEGIN /etc/grub.d/10_linux ###
menuentry 'Install Kylin Linux Advanced Server V10' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Kylin-Server-10 quiet
	initrdefi /images/pxeboot/initrd.img
}
menuentry 'Test this media & install Kylin Linux Advanced Server V10' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Kylin-Server-10 rd.live.check quiet
	initrdefi /images/pxeboot/initrd.img
}
### END /etc/grub.d/10_linux ###
`
	info := getOsInfoByGrub(content)
	if info.Distro != imagetools.OS_DIST_KYLIN {
		t.Fatalf("expected distro %s, got %s", imagetools.OS_DIST_KYLIN, info.Distro)
	}
	if info.Version == "" {
		t.Fatalf("expected version from menuentry, got empty")
	}
}

func TestGetOsInfoByGrubRealFedora(t *testing.T) {
	content := `
menuentry 'Install Fedora Linux 40' --class fedora --class gnu-linux --class gnu --class os {
	linuxefi /images/pxeboot/vmlinuz inst.stage2=hd:LABEL=Fedora-40 quiet
	initrdefi /images/pxeboot/initrd.img
}
`
	info := getOsInfoByGrub(content)
	if info.Distro != imagetools.OS_DIST_FEDORA {
		t.Fatalf("expected distro %s, got %s", imagetools.OS_DIST_FEDORA, info.Distro)
	}
}

func TestExtractGrubMenuEntryTitles(t *testing.T) {
	content := `menuentry 'test this media & install kylin linux advanced server v10' --class fedora --class gnu-linux --class gnu --class os {`
	titles := extractGrubMenuEntryTitles(content)
	if len(titles) != 1 {
		t.Fatalf("expected 1 title, got %v", titles)
	}
	if titles[0] != "test this media & install kylin linux advanced server v10" {
		t.Fatalf("unexpected title: %q", titles[0])
	}
}
