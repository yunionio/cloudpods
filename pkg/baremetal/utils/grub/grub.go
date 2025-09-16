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

// REF: https://github.com/bluebanquise/infrastructure/blob/master/packages/ipxe-bluebanquise/grub2-efi-autofind.cfg
const autoFindCfg = `
echo "Loading modules..."
insmod part_gpt
insmod fat
insmod chain
insmod part_msdos
insmod ext2
insmod xfs
echo
echo "Scanning, first pass..."
for cfg in (*,gpt*)/efi/*/grub.cfg (*,gpt*)/efi/*/*/grub.cfg (*,gpt*)/grub.cfg (*,gpt*)/*/grub.cfg (*,gpt*)/*/*/grub.cfg (*,msdos*)/grub.cfg (*,msdos*)/*/grub.cfg (*,mosdos*)/*/*/grub.cfg; do
	regexp --set=1:cfg_device '^\((.*)\)/' "${cfg}"
done

echo "Scanning, second pass..."
for cfg in (*,gpt*)/efi/*/grub.cfg (*,gpt*)/efi/*/*/grub.cfg (*,gpt*)/grub.cfg (*,gpt*)/*/grub.cfg (*,gpt*)/*/*/grub.cfg (*,msdos*)/grub.cfg (*,msdos*)/*/grub.cfg (*,mosdos*)/*/*/grub.cfg; do
	regexp --set=1:cfg_device '^\((.*)\)/' "${cfg}"
	echo "Try configfile ${cfg}"
	if [ -e "${cfg}" ]; then
		cfg_found=true
		echo " >> Found operating system grub config! <<"
		echo " Path: ${cfg}"
		echo " Booting in 5s..."
		sleep --interruptible --verbose 5
		configfile "${cfg}"
		boot
	fi
done

echo "No grub.cfg known OS found. Fall back on shell after 5s."
sleep 5s
`

func GetAutoFindConfig() string {
	return autoFindCfg
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
