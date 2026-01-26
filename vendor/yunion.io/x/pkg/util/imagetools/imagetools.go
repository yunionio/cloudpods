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
	"fmt"
	"regexp"
	"strings"

	"yunion.io/x/pkg/util/osprofile"
)

const (
	OS_DIST_CENTOS        = "CentOS"
	OS_DIST_CENTOS_STREAM = "CentOS Stream"

	OS_DIST_RHEL     = "RHEL"
	OS_DIST_FREE_BSD = "FreeBSD"

	OS_DIST_UBUNTU_SERVER = "Ubuntu Server"
	OS_DIST_UBUNTU        = "Ubuntu"

	OS_DIST_OPEN_SUSE  = "OpenSUSE"
	OS_DIST_SUSE       = "SUSE"
	OS_DIST_DEBIAN     = "Debian"
	OS_DIST_CORE_OS    = "CoreOS"
	OS_DIST_EULER_OS   = "EulerOS"
	OS_DIST_OPEN_EULER = "OpenEuler"
	OS_DIST_ALIYUN     = "Aliyun"
	OS_DIST_DEEPIN     = "Deepin"

	OS_DIST_ALIBABA_CLOUD_LINUX = "Alibaba Cloud Linux"
	OS_DIST_ANOLIS              = "Anolis OS"
	OS_DIST_ROCKY_LINUX         = "Rocky Linux"
	OS_DIST_FEDORA              = "Fedora"
	OS_DIST_ALMA_LINUX          = "AlmaLinux"
	OS_DIST_AMAZON_LINUX        = "Amazon Linux"

	OS_DIST_WINDOWS_SERVER = "Windows Server"
	OS_DIST_WINDOWS        = "Windows"

	OS_DIST_KYLIN = "Kylin"
	OS_DIST_UOS   = "UOS"

	OS_DIST_TENCENTOS_SERVER = "TencentOS Server"
	OS_DIST_OTHER_LINUX      = "Others Linux"
)

func normalizeOsArch(osArch string, imageName string) string {
	if len(osArch) > 0 {
		switch strings.ToLower(osArch) {
		case "x86_64", "amd64", "64", "64bit", "64位":
			return osprofile.OS_ARCH_X86_64
		case "x86", "x86_32", "32", "32bit", "32位":
			return osprofile.OS_ARCH_X86
		case "arm", "arm64", "aarch", "aarch64":
			return osprofile.OS_ARCH_AARCH64
		default:
			return osArch
		}
	} else {
		for _, arch := range []string{
			"64bit", "64位", "amd64", "x86_64",
			"32bit", "32位", "i386", "x86",
			"armv8", "arm64", "aarch64", "aarch",
			"armv6", "armv7", "armv7s", "arm", "aarch32",
		} {
			if strings.Contains(strings.ToLower(imageName), arch) {
				switch arch {
				case "64bit", "64位", "amd64", "x86_64":
					return osprofile.OS_ARCH_X86_64
				case "32bit", "32位", "i386", "x86":
					return osprofile.OS_ARCH_X86_32
				case "armv8", "arm64", "aarch64":
					return osprofile.OS_ARCH_AARCH64
				case "armv6", "armv7", "armv7s", "arm", "aarch32":
					return osprofile.OS_ARCH_AARCH32
				}
			}
		}
		return osprofile.OS_ARCH_X86_64
	}
}

func normalizeOsType(osType string, osDist string) string {
	osType = strings.ToLower(osType)
	if osType == "linux" {
		return osprofile.OS_TYPE_LINUX
	} else if osType == "windows" {
		return osprofile.OS_TYPE_WINDOWS
	} else if strings.HasPrefix(strings.ToLower(osDist), "windows") {
		return osprofile.OS_TYPE_WINDOWS
	} else {
		return osprofile.OS_TYPE_LINUX
	}
}

func normalizeOsDistribution(osDist string, imageName string) string {
	if len(osDist) == 0 {
		osDist = imageName
	}
	osDist = strings.ToLower(osDist)
	if strings.Contains(osDist, "tencentos") {
		return OS_DIST_TENCENTOS_SERVER
	} else if strings.Contains(osDist, "centos stream") {
		return OS_DIST_CENTOS_STREAM
	} else if strings.Contains(osDist, "centos") {
		return OS_DIST_CENTOS
	} else if strings.Contains(osDist, "redhat") || strings.Contains(osDist, "rhel") {
		return OS_DIST_RHEL
	} else if strings.Contains(osDist, "ubuntu server") {
		return OS_DIST_UBUNTU_SERVER
	} else if strings.Contains(osDist, "ubuntu") {
		return OS_DIST_UBUNTU
	} else if strings.Contains(osDist, "opensuse") {
		return OS_DIST_OPEN_SUSE
	} else if strings.Contains(osDist, "suse") {
		return OS_DIST_SUSE
	} else if strings.Contains(osDist, "debian") {
		return OS_DIST_DEBIAN
	} else if strings.Contains(osDist, "coreos") {
		return OS_DIST_CORE_OS
	} else if strings.Contains(osDist, "aliyun") {
		return OS_DIST_ALIYUN
	} else if strings.Contains(osDist, "freebsd") {
		return OS_DIST_FREE_BSD
	} else if strings.Contains(osDist, "euleros") {
		return OS_DIST_EULER_OS
	} else if strings.Contains(osDist, "openeuler") {
		return OS_DIST_OPEN_EULER
	} else if strings.Contains(osDist, "alibaba cloud linux") {
		return OS_DIST_ALIBABA_CLOUD_LINUX
	} else if strings.Contains(osDist, "anolis") {
		return OS_DIST_ANOLIS
	} else if strings.Contains(osDist, "rocky") {
		return OS_DIST_ROCKY_LINUX
	} else if strings.Contains(osDist, "fedora") {
		return OS_DIST_FEDORA
	} else if strings.Contains(osDist, "alma") {
		return OS_DIST_ALMA_LINUX
	} else if strings.Contains(osDist, "amazon") && strings.Contains(osDist, "linux") {
		return OS_DIST_AMAZON_LINUX
	} else if strings.Contains(osDist, "kylin") {
		return OS_DIST_KYLIN
	} else if strings.Contains(osDist, "uos") {
		return OS_DIST_UOS
	} else if strings.Contains(osDist, "windows") || regexp.MustCompile(".+win(xp|7|8|10|11|2003|2008|2012|2016|2019|2022)*").MatchString(osDist) {
		for _, ver := range []string{"2003", "2008", "2012", "2016", "2019", "2022"} {
			if strings.Contains(osDist, ver) {
				return OS_DIST_WINDOWS_SERVER
			}
		}
		return OS_DIST_WINDOWS
	} else if strings.Contains(osDist, "deepin") {
		return OS_DIST_DEEPIN
	} else {
		return OS_DIST_OTHER_LINUX
	}
}

var imageVersions = map[string][]string{
	// CentOS：补充停更版本和完整迭代，CentOS 8已EOL，Stream补充最新版本
	OS_DIST_CENTOS:        {"4", "5", "6", "7", "8", "9"},
	OS_DIST_CENTOS_STREAM: {"8", "9", "10"},

	// RHEL：补充完整主版本，覆盖从5到最新的10
	OS_DIST_RHEL: {"5", "6", "7", "8", "9", "10"},
	// FreeBSD：补充最新稳定版，覆盖10到15
	OS_DIST_FREE_BSD: {"10", "11", "12", "13", "14", "15"},

	// Ubuntu：Server版补充LTS版本（每2年一个），Desktop版补充所有主要版本
	OS_DIST_UBUNTU_SERVER: {"10.04", "12.04", "14.04", "16.04", "18.04", "20.04", "22.04", "24.04",
		"10", "12", "14", "16", "18", "20", "22", "24"},
	OS_DIST_UBUNTU: {"10.04", "12.04", "14.04", "15.04", "16.04", "17.04", "18.04", "19.04", "20.04", "21.04", "22.04", "23.04", "24.04",
		"10", "12", "14", "16", "17", "18", "19", "20", "21", "22", "23", "24"},

	// OpenSUSE：补充Leap版本，SUSE补充SLES主版本
	OS_DIST_OPEN_SUSE: {"11", "12", "13", "42", "15.0", "15.1", "15.2", "15.3", "15.4", "15.5", "15.6"},
	OS_DIST_SUSE:      {"10", "11", "12", "15", "15 SP1", "15 SP2", "15 SP3", "15 SP4", "15 SP5"},
	// Debian：补充从6到最新的12，覆盖所有稳定版
	OS_DIST_DEBIAN: {"6", "7", "8", "9", "10", "11", "12"},
	// CoreOS：补充Container Linux和Fedora CoreOS的主要版本
	OS_DIST_CORE_OS: {"7", "200", "213", "224", "234", "246", "251", "3033"},
	// 欧拉OS：补充openEuler和EulerOS完整版本
	OS_DIST_OPEN_EULER: {"2.0 SP1", "2.0 SP2", "2.0 SP3", "2.0 SP8", "3.0", "22.03", "23.09"},
	OS_DIST_EULER_OS:   {"2"},
	// 阿里云Linux：补充1代和2/3代版本
	OS_DIST_ALIYUN: {"1", "2.1903", "3.2104", "3.2304"},

	// 阿里云轻量版：补充完整版本
	OS_DIST_ALIBABA_CLOUD_LINUX: {"2.1903", "3.2104", "3.2304", "3.2404"},
	// 龙蜥OS：补充7/8系列完整小版本
	OS_DIST_ANOLIS: {"7.6", "7.9", "8.2", "8.4", "8.6", "8.8", "9.0", "9.2"},
	// Rocky Linux：补充8/9全系列小版本
	OS_DIST_ROCKY_LINUX: {"8.5", "8.6", "8.7", "8.8", "8.9", "8.10", "9.0", "9.1", "9.2", "9.3", "9.4", "9.5"},
	// Fedora：补充近年主流版本（33到40）
	OS_DIST_FEDORA: {"33", "34", "35", "36", "37", "38", "39", "40"},
	// AlmaLinux：补充8/9全系列
	OS_DIST_ALMA_LINUX: {"8.5", "8.6", "8.7", "8.8", "8.9", "8.10", "9.0", "9.1", "9.2", "9.3", "9.4", "9.5"},
	// Amazon Linux：补充1/2/2023版本
	OS_DIST_AMAZON_LINUX: {"2022", "2023", "1", "2"},

	// Windows Server：补充完整服务器版本
	OS_DIST_WINDOWS_SERVER: {"2003", "2008", "2008 R2", "2012", "2012 R2", "2016", "2019", "2022"},
	// Windows 桌面版：补充完整版本
	OS_DIST_WINDOWS: {"XP", "Vista", "7", "8", "8.1", "10", "11"},

	// 麒麟OS：补充V10各版本和V11
	OS_DIST_KYLIN: {"V10", "V10 SP1", "V10 SP2", "V10 SP3", "V11", "Nile"},
	// UOS：补充统信UOS完整版本
	OS_DIST_UOS: {"V20 1050", "20 1050", "1050", "V20 1060", "1060", "V20 1070", "1070", "V20 1080", "V20 1090", "V23", "Eagle", "V20"},

	// 腾讯云OS：补充完整版本
	OS_DIST_TENCENTOS_SERVER: {"2.4", "3.1", "3.2", "3.3", "4.0", "4"},
	// 其他Linux：预留空列表，可根据实际场景补充

	OS_DIST_DEEPIN:      {"20", "20.9", "21", "21.9", "22", "22.9", "23", "23.9", "Crimson"},
	OS_DIST_OTHER_LINUX: {},
}

func normalizeOsVersion(imageName string, osDist string, osVersion string) string {
	if versions, ok := imageVersions[osDist]; ok {
		for _, version := range versions {
			if len(osVersion) > 0 {
				if strings.HasPrefix(osVersion, version) {
					return osVersion
				}
			} else {
				parts := strings.Split(strings.ToLower(osDist), " ")
				parts = append(parts, fmt.Sprintf(`(?P<verstr>%s[.\d]*)`, version), "")
				regexpStr := strings.Join(parts, `[\s-_]*`)
				m := regexp.MustCompile(regexpStr).FindAllStringSubmatch(strings.ToLower(imageName), -1)
				if m != nil && len(m) > 0 && len(m[0]) > 1 {
					verStr := m[0][1]
					if strings.HasPrefix(verStr, version) && len(verStr) > len(version) && !strings.HasPrefix(verStr, version+".") {
						verStr = version + "." + verStr[len(version):]
					}
					return verStr
				}
			}
		}
		for i := len(versions) - 1; i > 0; i-- {
			if strings.Contains(imageName, versions[i]) {
				return versions[i]
			}
		}
	}
	return ""
}

func normalizeOsLang(imageName string) string {
	for _, sep := range []string{" ", "-", "_"} {
		lang := normalizeOsLang2(imageName, sep)
		if len(lang) > 0 {
			return lang
		}
	}
	return ""
}

func normalizeOsLang2(imageName string, sep string) string {
	parts := strings.Split(imageName, sep)
	for _, o := range parts {
		switch strings.ToLower(o) {
		case "en":
			return "en_US"
		case "cn":
			return "zh_CN"
		case "zh":
			return "zh_CN"
		case "中文版":
			return "zh_CN"
		case "英文版":
			return "en_US"
		}
	}
	return ""
}

func normalizeOsBios(imageName string, osArch string) string {
	if osArch != "" {
		switch osArch {
		case osprofile.OS_ARCH_ARM, osprofile.OS_ARCH_AARCH32, osprofile.OS_ARCH_AARCH64:
			return osprofile.OS_BOOT_UEFI
		}
	}
	for _, sep := range []string{" ", "-", "_"} {
		lang := normalizeOsBios2(imageName, sep)
		if len(lang) > 0 {
			return lang
		}
	}
	return ""
}

func normalizeOsBios2(imageName string, sep string) string {
	parts := strings.Split(imageName, sep)
	for _, o := range parts {
		switch strings.ToLower(o) {
		case "uefi":
			return osprofile.OS_BOOT_UEFI
		}
	}
	return osprofile.OS_BOOT_BIOS
}

type ImageInfo struct {
	Name          string
	OsArch        string
	OsType        string
	OsDistro      string
	OsVersion     string
	OsFullVersion string
	OsLang        string
	OsBios        string
}

func (i ImageInfo) GetFullOsName() string {
	parts := make([]string, 0)
	if len(i.OsDistro) > 0 {
		parts = append(parts, i.OsDistro)
	} else if len(i.OsType) > 0 {
		parts = append(parts, string(i.OsType))
	}
	if len(i.OsVersion) > 0 {
		parts = append(parts, i.OsVersion)
	}
	if len(i.OsArch) > 0 {
		parts = append(parts, i.OsArch)
	}
	if len(i.OsBios) > 0 {
		parts = append(parts, i.OsBios)
	}
	if len(i.OsLang) > 0 {
		parts = append(parts, i.OsLang)
	}
	return strings.Join(parts, " ")
}

func NormalizeImageInfo(imageName, osArch, osType, osDist, osVersion string) ImageInfo {
	info := ImageInfo{}
	info.Name = imageName
	info.OsDistro = normalizeOsDistribution(osDist, imageName)
	info.OsType = normalizeOsType(osType, info.OsDistro)
	info.OsArch = normalizeOsArch(osArch, imageName)
	info.OsVersion = normalizeOsVersion(imageName, info.OsDistro, osVersion)
	info.OsFullVersion = osVersion
	info.OsLang = normalizeOsLang(imageName)
	info.OsBios = normalizeOsBios(imageName, info.OsArch)
	return info
}
