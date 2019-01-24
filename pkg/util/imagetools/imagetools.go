package imagetools

import "strings"

func normalizeOsArch(osArch string, osType string, osDist string) string {
	if len(osArch) > 0 {
		if strings.ToLower(osArch) == "x86_64" {
			return "x86_64"
		} else {
			return "i386"
		}
	} else {
		if osType == "linux" {
			return "x86_64"
		} else if osDist == "Windows Server 2003" {
			return "i386"
		} else {
			return "x86_64"
		}
	}
}

func normalizeOsType(osType string, osDist string) string {
	osType = strings.ToLower(osType)
	if osType == "linux" {
		return "linux"
	} else if osType == "windows" {
		return "windows"
	} else if strings.HasPrefix(osDist, "Windows") {
		return "windows"
	} else {
		return "linux"
	}
}

func normalizeOsDistribution(osDist string, imageName string) string {
	if len(osDist) == 0 {
		osDist = imageName
	}
	osDist = strings.ToLower(osDist)
	if strings.Contains(osDist, "centos") || strings.Contains(osDist, "redhat") || strings.Contains(osDist, "rhel") {
		return "CentOS"
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
	} else if strings.Contains(osDist, "windows") {
		if strings.Contains(osDist, "2003") {
			return "Windows Server 2003"
		} else if strings.Contains(osDist, "2008") {
			return "Windows Server 2008"
		} else if strings.Contains(osDist, "2012") {
			return "Windows Server 2012"
		} else if strings.Contains(osDist, "2016") {
			return "Windows Server 2016"
		} else {
			return "Windows Server 2008"
		}
	} else {
		return "Others Linux"
	}
}

var imageVersions = map[string][]string{
	"CentOS":   {"5", "6", "7"},
	"FreeBSD":  {"10"},
	"Ubuntu":   {"10", "12", "14", "16"},
	"OpenSUSE": {"11", "12"},
	"SUSE":     {"10", "11", "12", "13"},
	"Debian":   {"6", "7", "8", "9"},
	"CoreOS":   {"7"},
	"Aliyun":   {},
}

func normalizeOsVersion(imageName string, osDist string, osVersion string) string {
	if versions, ok := imageVersions[osDist]; ok {
		for _, version := range versions {
			if strings.HasPrefix(osVersion, version) {
				return version
			}
		}
		if len(versions) > 0 {
			return versions[0]
		}
	}
	return "-"
}

type ImageInfo struct {
	Name          string
	OsArch        string
	OsType        string
	OsDistro      string
	OsVersion     string
	OsFullVersion string
}

func NormalizeImageInfo(imageName, osArch, osType, osDist, osVersion string) ImageInfo {
	info := ImageInfo{}
	info.Name = imageName
	info.OsDistro = normalizeOsDistribution(osDist, imageName)
	info.OsType = normalizeOsType(osType, info.OsDistro)
	info.OsArch = normalizeOsArch(osArch, info.OsType, info.OsDistro)
	info.OsVersion = normalizeOsVersion(imageName, info.OsDistro, osVersion)
	info.OsFullVersion = osVersion
	return info
}
