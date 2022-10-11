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

package imagetools

import (
	"testing"

	"yunion.io/x/pkg/util/osprofile"
)

func TestNormalizeImageInfo(t *testing.T) {
	images := []struct {
		Name      string
		OsDistro  string
		OsType    string
		OsVersion string
		OsLang    string
		OsArch    string
	}{
		{
			Name:      "rhel67_20180816.qcow2",
			OsDistro:  "RHEL",
			OsType:    osprofile.OS_TYPE_LINUX,
			OsVersion: "6.7",
			OsLang:    "",
			OsArch:    "x86_64",
		},
		{
			Name:      "Ubuntu_16.04.3_amd64_qingcloud_20180817.qcow2",
			OsDistro:  "Ubuntu",
			OsType:    osprofile.OS_TYPE_LINUX,
			OsVersion: "16.04.3",
			OsLang:    "",
			OsArch:    "x86_64",
		},
		{
			Name:      "windows-server-2008-dc-cn-20180717",
			OsDistro:  "Windows Server",
			OsType:    osprofile.OS_TYPE_WINDOWS,
			OsVersion: "2008",
			OsLang:    "zh_CN",
			OsArch:    "x86_64",
		},
		{
			Name:      "Ubuntu  14.04 32位",
			OsDistro:  "Ubuntu",
			OsType:    osprofile.OS_TYPE_LINUX,
			OsVersion: "14.04",
			OsLang:    "",
			OsArch:    "x86_32",
		},
		{
			Name:      "CentOS  7.2 64位",
			OsDistro:  "CentOS",
			OsType:    osprofile.OS_TYPE_LINUX,
			OsVersion: "7.2",
			OsLang:    "",
			OsArch:    "x86_64",
		},
		{
			Name:      "Windows Server 2019 数据中心版 64位 中文版 GRID13",
			OsDistro:  "Windows Server",
			OsType:    osprofile.OS_TYPE_WINDOWS,
			OsVersion: "2019",
			OsLang:    "zh_CN",
			OsArch:    "x86_64",
		},
		{
			Name:      "CentOS 8.2(arm64)",
			OsDistro:  "CentOS",
			OsType:    osprofile.OS_TYPE_LINUX,
			OsVersion: "8.2",
			OsLang:    "",
			OsArch:    "aarch64",
		},
		{
			Name:      "Ubuntu Server 22.04 LTS 64位",
			OsDistro:  "Ubuntu Server",
			OsType:    osprofile.OS_TYPE_LINUX,
			OsVersion: "22.04",
			OsLang:    "",
			OsArch:    "x86_64",
		},
		{
			Name:      "CentOS Stream 9 64位",
			OsDistro:  "CentOS Stream",
			OsType:    osprofile.OS_TYPE_LINUX,
			OsVersion: "9",
			OsLang:    "",
			OsArch:    "x86_64",
		},
		{
			Name:      "Debian 11.4 64位",
			OsDistro:  "Debian",
			OsType:    osprofile.OS_TYPE_LINUX,
			OsVersion: "11.4",
			OsLang:    "",
			OsArch:    "x86_64",
		},
	}

	for _, image := range images {
		info := NormalizeImageInfo(image.Name, "", "", "", "")
		if info.OsType != image.OsType {
			t.Errorf("%s osType should be %s", image.Name, image.OsType)
		}
		if info.OsDistro != image.OsDistro {
			t.Errorf("%s osDistro should be %s, but is %s", image.Name, image.OsDistro, info.OsDistro)
		}
		if info.OsVersion != image.OsVersion {
			t.Errorf("%s osVersion should be %s, but is %s", image.Name, image.OsVersion, info.OsVersion)
		}
		if info.OsLang != image.OsLang {
			t.Errorf("%s osLang should be %s, but is %s", image.Name, image.OsLang, info.OsLang)
		}
		if info.OsArch != image.OsArch {
			t.Errorf("%s osArch should be %s, but is %s", image.Name, image.OsArch, info.OsArch)
		}
	}

}
