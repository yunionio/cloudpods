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

//go:build linux
// +build linux

package isoutils

import (
	"fmt"

	"github.com/Microsoft/go-winio/wim"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/imagetools"
)

// ========== 7. 保留Windows版本识别函数（适配新结构） ==========
func DetectWindowsEdition(r *ISOFileReader) (*ISOInfo, error) {
	wimFile, err := r.GetFile("sources/install.wim")
	if err != nil {
		return nil, err
	}
	wim, err := wim.NewReader(wimFile.NewReader())
	if err != nil {
		return nil, err
	}
	result := &ISOInfo{}
	for _, image := range wim.Image {
		version := fmt.Sprintf("%d.%d.%d", image.Windows.Version.Major, image.Windows.Version.Minor, image.Windows.Version.Build)
		if image.Windows != nil {
			if image.Windows.Arch == 9 {
				result.Arch = "x86_64"
			} else if image.Windows.Arch == 12 {
				result.Arch = "arm64"
			} else if image.Windows.Arch == 0 {
				result.Arch = "x86"
			}
			result.Distro = imagetools.OS_DIST_WINDOWS
			result.Language = image.Windows.DefaultLanguage
			switch fmt.Sprintf("%d.%d", image.Windows.Version.Major, image.Windows.Version.Minor) {
			case "6.0":
				result.Version = "Windows Vista"
			case "6.1":
				result.Version = "Windows 7"
			case "6.2":
				result.Version = "Windows 8"
			case "6.3":
				result.Version = "Windows 8.1"
			case "10.0":
				if image.Windows.Version.Build >= 27500 {
					result.Version = "Windows 12"
				} else if image.Windows.Version.Build >= 22000 {
					result.Version = "Windows 11"
				} else {
					result.Version = "Windows 10"
				}
			}
			if image.Windows.ProductType == "ServerNT" {
				result.Distro = imagetools.OS_DIST_WINDOWS_SERVER
				switch fmt.Sprintf("%d.%d", image.Windows.Version.Major, image.Windows.Version.Minor) {
				case "6.0":
					result.Version = "Windows Server 2008"
				case "6.1":
					result.Version = "Windows Server 2008 R2"
				case "6.2":
					result.Version = "Windows Server 2012"
				case "6.3":
					result.Version = "Windows Server 2012 R2"
				case "10.0":
					if image.Windows.Version.Build >= 26040 {
						result.Version = "Windows Server 2025"
					} else if image.Windows.Version.Build >= 20348 {
						result.Version = "Windows Server 2022"
					} else if image.Windows.Version.Build >= 17763 {
						result.Version = "Windows Server 2019"
					} else if image.Windows.Version.Build >= 14393 {
						result.Version = "Windows Server 2016"
					}
				}
			}
			log.Debugf("识别到 %s 版本: %s -> %s", result.Distro, version, result.Version)
			break
		}
	}
	return result, nil
}
