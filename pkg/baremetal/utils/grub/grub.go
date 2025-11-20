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

package grub

import (
	"fmt"
	"strings"
)

func GetYunionOSConfig(sleepTime int, httpSite, kernel string, kernelArgs string, initrd string, useTftpDownload bool) string {
	if useTftpDownload {
		site := strings.Split(httpSite, ":")[0]
		kernel = fmt.Sprintf("(tftp,%s)/%s", site, kernel)
		initrd = fmt.Sprintf("(tftp,%s)/%s", site, initrd)
	} else {
		kernel = fmt.Sprintf("(http,%s)/tftp/%s", httpSite, kernel)
		initrd = fmt.Sprintf("(http,%s)/tftp/%s", httpSite, initrd)
	}
	return fmt.Sprintf(`
set timeout=%d
menuentry 'YunionOS for PXE' --class os {
	echo "Loading linux %s ..."
	linux %s %s

	echo "Loading initrd %s ..."
	initrd %s
}
`, sleepTime, kernel, kernel, kernelArgs, initrd, initrd)
}

var (
	BOOT_EFI_MATCHER = "(hd*,gpt*)/EFI/BOOT/BOOTX64.EFI (hd*,msdos*)/EFI/BOOT/BOOTX64.EFI (hd*,gpt*)/EFI/BOOT/BOOTAA64.EFI (hd*,msdos*)/EFI/BOOT/BOOTAA64.EFI"
	GRUB_EFI_MATCHER = "(md*)/EFI/*/shimx64.efi (md*)/EFI/*/grubx64.efi (hd*,gpt*)/EFI/*/shimx64.efi (hd*,msdos*)/EFI/*/shimx64.efi (hd*,gpt*)/EFI/*/grubx64.efi (hd*,msdos*)/EFI/*/grubx64.efi (hd*,gpt*)/EFI/*/shimaa64.efi (hd*,msdos*)/EFI/*/shimaa64.efi (hd*,gpt*)/EFI/*/grubaa64.efi (hd*,msdos*)/EFI/*/grubaa64.efi"
	GRUB_CFG_MATCHER = "(md*)/boot/*/grub.cfg (md*)/*/grub.cfg (md*)/grub.cfg (md*)/EFI/*/grub.cfg (md*)/EFI/*/*/grub.cfg (hd*,gpt*)/boot/*/grub.cfg (hd*,gpt*)/*/grub.cfg (hd*,gpt*)/grub.cfg (hd*,gpt*)/EFI/*/grub.cfg (hd*,gpt*)/EFI/*/*/grub.cfg (hd*,msdos*)/boot/*/grub.cfg (hd*,msdos*)/*/grub.cfg (hd*,msdos*)/grub.cfg (hd*,msdos*)/EFI/*/grub.cfg (hd*,msdos*)/EFI/*/*/grub.cfg"
)

// REF: https://github.com/bluebanquise/infrastructure/blob/master/packages/bluebanquise-ipxe/grub2-efi-autofind.cfg
const autoFindCfg = `
echo "Loading modules..."
insmod part_gpt
insmod fat
insmod chain
insmod part_msdos
insmod ext2
insmod regexp
insmod xfs
insmod mdraid1x
insmod mdraid09

echo
echo "======================= lsmod ======================================="
lsmod

echo
echo "======================= devices ====================================="
ls

echo
echo "======================= Searching for the BOOT EFI executable ======="
echo "Scanning, 1st pass"
for efi in %s; do
	regexp --set=1:root '^\(([^)]+)\)/' "${efi}"
	echo "- Scanning: $efi"
	if [ -e "$efi" ] ; then
		echo "	Found: $efi"
	else
		echo "		$efi does not exist"
	fi
done

echo
echo "Scanning, 2nd pass..."
for efi in %s; do
	echo "- Scanning: $efi"
	if [ -e "$efi" ] ; then
		regexp --set 1:root '^\(([^)]+)\)/' "${efi}"
		echo "	Found: $efi"
		echo "	Root: $root"
		echo "	Chainloading $efi"
		chainloader "$efi"
		boot
	fi
done

echo
echo "		Found no BOOT EFI executable.  Falling back to shell..."

echo
echo "======================= Searching for Grub EFI executables =========="
echo "Scanning, 1st pass"
for efi in %s ; do
	regexp --set=1:root '^\(([^)]+)\)/' "${efi}"
	echo "- Scanning: $efi"
	if [ -e "$efi" ] ; then
		echo "	Found: $efi"
	else
		echo "		$efi does not exist"
	fi
done

echo
echo "Scanning, 2nd pass..."
for efi in %s ; do
	echo "- Scanning: $efi"
	if [ -e "$efi" ] ; then
		regexp --set 1:root '^\(([^)]+)\)/' "${efi}"
		echo "	Found: $efi"
		echo "	Root: $root"
		echo "	Chainloading $efi"
		chainloader "$efi"
		boot
	fi
done

echo
echo "		Found no Grub EFI executable to load an OS."

echo
echo "======================= Searching for grub.cfg on local disks ======="
echo "Scanning, 1st pass..."
for grubcfg in %s ; do
	regexp --set=1:root '^\(([^)]+)\)/' "${grubcfg}"
	if [ -e "$grubcfg" ] ; then
		echo "	Found: $grubcfg"
	else
		echo "		$grubcfg does not exist"
	fi
done

echo
echo "Scanning, 2nd pass..."
for grubcfg in %s ; do
	echo "- Scanning: $grubcfg"
	if [ -e "${grubcfg}" ]; then
		regexp --set=1:root '^\(([^)]+)\)/' "${grubcfg}"
		echo "	Found: $grubcfg"
		echo "	Root: $root"
#		echo "	Contents:"
#		cat "$grubcfg"
		configfile "${grubcfg}"
		boot
	fi
done

echo
echo "		Found no grub.cfg configuration file to load."

sleep 4s
`

func GetAutoFindConfig() string {
	return fmt.Sprintf(autoFindCfg, BOOT_EFI_MATCHER, BOOT_EFI_MATCHER, GRUB_EFI_MATCHER, GRUB_EFI_MATCHER, GRUB_CFG_MATCHER, GRUB_CFG_MATCHER)
}

// REF: https://archived.forum.manjaro.org/t/detecting-efi-files-and-booting-them-from-grub/38083
const efiDetectMenuCfg = `
menuentry "Detect EFI bootloaders "  {
	insmod part_gpt
	insmod fat
	insmod chain
	insmod part_msdos
	insmod ext2
	insmod xfs

	for efi in (*,gpt*)/efi/*/*.efi (*,gpt*)/efi/*/*/*.efi (*,gpt*)/*.efi (*,gpt*)/*/*.efi; do
		regexp --set=1:efi_device '^\((.*)\)/' "${efi}"
		if [ -e "${efi}" ]; then
			efi_found=true

			menuentry --class=efi "${efi}" "${efi_device}" {
				root="${2}"
				chainloader "${1}"
			}
		fi
	done

	if [ "${efi_found}" != true ]; then
		menuentry --hotkey=q --class=find.none "No EFI files detected." {menu_reload}
	else
		menuentry --hotkey=q --class=cancel "Cancel" {menu_reload}
	fi
}
`

func GetEFIDetectMenuConfig() string {
	return efiDetectMenuCfg
}
