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

package profiles

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

type IPMIProfile struct {
	LanChannel []int
	RootName   string
	RootId     int
	StrongPass bool
}

func DefaultProfile() IPMIProfile {
	return IPMIProfile{
		LanChannel: []int{1},
		RootName:   "root",
		RootId:     2,
	}
}

func InspurProfile() IPMIProfile {
	return IPMIProfile{
		LanChannel: []int{8, 1},
		RootName:   "admin",
		RootId:     2,
	}
}

func LenovoProfile() IPMIProfile {
	return IPMIProfile{
		LanChannel: []int{8},
		RootName:   "root",
		RootId:     2,
	}
}

func HpProfile() IPMIProfile {
	return IPMIProfile{
		LanChannel: []int{2},
		RootName:   "root",
		RootId:     1,
	}
}

func HuaweiProfile() IPMIProfile {
	return IPMIProfile{
		LanChannel: []int{1},
		RootName:   "root",
		RootId:     2,
		StrongPass: true,
	}
}

func FoxconnProfile() IPMIProfile {
	return IPMIProfile{
		LanChannel: []int{1},
		RootName:   "root",
		RootId:     2,
		StrongPass: true,
	}
}

func QemuProfile() IPMIProfile {
	return IPMIProfile{
		LanChannel: []int{8, 1},
		RootName:   "root",
		RootId:     2,
		StrongPass: true,
	}
}

var (
	PROFILES map[string]IPMIProfile = map[string]IPMIProfile{
		"inspur":  InspurProfile(),
		"lenovo":  LenovoProfile(),
		"hp":      HpProfile(),
		"huawei":  HuaweiProfile(),
		"foxconn": FoxconnProfile(),
		"qemu":    QemuProfile(),
	}
)

func GetProfile(sysinfo *types.SSystemInfo) IPMIProfile {
	profile, ok := PROFILES[sysinfo.OemName]
	if ok {
		return profile
	}
	return DefaultProfile()
}

func GetLanChannel(sysinfo *types.SSystemInfo) []int {
	return GetProfile(sysinfo).LanChannel
}

func GetRootId(sysinfo *types.SSystemInfo) int {
	return GetProfile(sysinfo).RootId
}

func GetRootName(sysinfo *types.SSystemInfo) string {
	return GetProfile(sysinfo).RootName
}

func IsStrongPass(sysinfo *types.SSystemInfo) bool {
	return GetProfile(sysinfo).StrongPass
}
