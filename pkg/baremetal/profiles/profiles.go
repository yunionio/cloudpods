package profiles

import (
	"strings"

	"yunion.io/x/onecloud/pkg/baremetal/types"
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

func GetProfile(sysinfo *types.IPMISystemInfo) IPMIProfile {
	profile, ok := PROFILES[strings.ToLower(sysinfo.Manufacture)]
	if ok {
		return profile
	}
	return DefaultProfile()
}

func GetLanChannel(sysinfo *types.IPMISystemInfo) []int {
	return GetProfile(sysinfo).LanChannel
}

func GetRootId(sysinfo *types.IPMISystemInfo) int {
	return GetProfile(sysinfo).RootId
}

func GetRootName(sysinfo *types.IPMISystemInfo) string {
	return GetProfile(sysinfo).RootName
}

func IsStrongPass(sysinfo *types.IPMISystemInfo) bool {
	return GetProfile(sysinfo).StrongPass
}
