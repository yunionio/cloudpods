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
	if strings.Contains(osDist, "centos stream") {
		return "CentOS Stream"
	} else if strings.Contains(osDist, "centos") {
		return "CentOS"
	} else if strings.Contains(osDist, "redhat") || strings.Contains(osDist, "rhel") {
		return "RHEL"
	} else if strings.Contains(osDist, "ubuntu server") {
		return "Ubuntu Server"
	} else if strings.Contains(osDist, "ubuntu") {
		return "Ubuntu"
	} else if strings.Contains(osDist, "suse") {
		return "SUSE"
	} else if strings.Contains(osDist, "opensuse") {
		return "OpenSUSE"
	} else if strings.Contains(osDist, "debian") {
		return "Debian"
	} else if strings.Contains(osDist, "coreos") {
		return "CoreOS"
	} else if strings.Contains(osDist, "aliyun") {
		return "Aliyun"
	} else if strings.Contains(osDist, "freebsd") {
		return "FreeBSD"
	} else if strings.Contains(osDist, "euleros") {
		return "EulerOS"
	} else if strings.Contains(osDist, "alibaba cloud linux") {
		return "Alibaba Cloud Linux"
	} else if strings.Contains(osDist, "anolis") {
		return "Anolis OS"
	} else if strings.Contains(osDist, "rocky") {
		return "Rocky Linux"
	} else if strings.Contains(osDist, "fedora") {
		return "Fedora"
	} else if strings.Contains(osDist, "alma") {
		return "AlmaLinux"
	} else if strings.Contains(osDist, "amazon") && strings.Contains(osDist, "linux") {
		return "Amazon Linux"
	} else if strings.Contains(osDist, "windows") {
		for _, ver := range []string{"2003", "2008", "2012", "2016", "2019", "2022"} {
			if strings.Contains(osDist, ver) {
				return "Windows Server"
			}
		}
		return "Windows"
	} else {
		return "Others Linux"
	}
}

var imageVersions = map[string][]string{
	"CentOS":        {"5", "6", "7", "8"},
	"CentOS Stream": {"8", "9"},

	"RHEL":    {"5", "6", "7", "8", "9"},
	"FreeBSD": {"10", "11", "12"},

	"Ubuntu Server": {"10", "12", "14", "16", "18", "20", "22"},
	"Ubuntu":        {"10", "12", "14", "16", "17", "18", "19", "20", "21", "22"},

	"OpenSUSE": {"11", "12"},
	"SUSE":     {"10", "11", "12", "13"},
	"Debian":   {"6", "7", "8", "9", "10", "11"},
	"CoreOS":   {"7"},
	"EulerOS":  {"2"},
	"Aliyun":   {},

	"Alibaba Cloud Linux": {"2.1903", "3.2104"},
	"Anolis OS":           {"7.9", "8.2", "8.4"},
	"Rocky Linux":         {"8.5", "8.6", "8.7", "8.8", "8.9", "9.0", "9.1", "9.2"},
	"Fedora":              {"33", "34", "35"},
	"AlmaLinux":           {"8.5"},
	"Amazon Linux":        {"2023", "2"},

	"Windows Server": {"2003", "2008", "2012", "2016", "2019", "2022"},
	"Windows":        {"XP", "7", "8", "Vista", "10", "11"},

	"Kylin": {"V10"},
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
