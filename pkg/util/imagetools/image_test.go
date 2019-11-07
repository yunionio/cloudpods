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

import "testing"

func TestNormalizeImageInfo(t *testing.T) {
	images := []struct {
		Name      string
		OsDistro  string
		OsType    string
		OsVersion string
	}{
		{
			Name:      "rhel67_20180816.qcow2",
			OsDistro:  "RHEL",
			OsType:    "linux",
			OsVersion: "-",
		},
		{
			Name:      "Ubuntu_16.04.3_amd64_qingcloud_20180817.qcow2",
			OsDistro:  "Ubuntu",
			OsType:    "linux",
			OsVersion: "16",
		},
		{
			Name:      "windows-server-2008-dc-cn-20180717",
			OsDistro:  "Windows Server 2008",
			OsType:    "windows",
			OsVersion: "-",
		},
		{
			Name:      "Ubuntu  14.04 32位",
			OsDistro:  "Ubuntu",
			OsType:    "linux",
			OsVersion: "14",
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
	}

}
