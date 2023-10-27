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
	"path/filepath"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

/*
ID 256 gen 32 top level 5 path @
ID 257 gen 2426 top level 256 path @/var
ID 258 gen 870 top level 256 path @/usr/local
ID 259 gen 2426 top level 256 path @/tmp
ID 260 gen 35 top level 256 path @/srv
ID 261 gen 2426 top level 256 path @/root
ID 262 gen 874 top level 256 path @/opt
ID 263 gen 35 top level 256 path @/home
ID 264 gen 26 top level 256 path @/boot/grub2/x86_64-efi
ID 265 gen 64 top level 256 path @/boot/grub2/i386-pc
ID 266 gen 881 top level 256 path @/.snapshots
ID 267 gen 2426 top level 266 path @/.snapshots/1/snapshot
ID 272 gen 65 top level 266 path @/.snapshots/2/snapshot
*/
func parseBtrfsSubvols(lines string) []string {
	log.Debugf("parseBtrfsSubvols %s", lines)
	var ret []string
	for _, l := range strings.Split(lines, "\n") {
		parts := strings.Split(l, " ")
		if len(parts) >= 9 && strings.HasPrefix(parts[8], "@/") {
			ret = append(ret, strings.TrimSpace(parts[8]))
		}
	}
	return ret
}

func MountSubvols(dev string, mnt string) error {
	output, err := procutils.NewCommand("btrfs", "subvolume", "list", mnt).Output()
	if err != nil {
		return errors.Wrap(err, "btrfs subvolume list")
	}
	subvols := parseBtrfsSubvols(string(output))
	log.Debugf("mount btrfs subvols %s", strings.Join(subvols, ","))
	for _, subvol := range subvols {
		err := procutils.NewCommand("mount", dev, filepath.Join(mnt, subvol[2:]), "-o", "subvol=/"+subvol).Run()
		if err != nil {
			return errors.Wrapf(err, "btrfs mount subvolume %s", subvol)
		}
	}
	return nil
}

func UnmountSubvols(mnt string) error {
	output, err := procutils.NewCommand("btrfs", "subvolume", "list", mnt).Output()
	if err != nil {
		return errors.Wrap(err, "btrfs subvolume list")
	}
	subvols := parseBtrfsSubvols(string(output))
	log.Debugf("unmount btrfs subvols %s", strings.Join(subvols, ","))
	// scan in reverse order
	var errs []error
	for i := len(subvols) - 1; i >= 0; i-- {
		subvol := subvols[i]
		err := procutils.NewCommand("umount", filepath.Join(mnt, subvol[2:])).Run()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "btrfs unmount subvolume %s", subvol))
		}
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return nil
}
