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
	if strings.HasPrefix(osDist, "centos") || strings.HasPrefix(osDist, "redhat") || strings.HasPrefix(osDist, "rhel") {
		return "CentOS"
	} else if strings.HasPrefix(osDist, "ubuntu") {
		return "Ubuntu"
	} else if strings.HasPrefix(osDist, "suse") {
		return "SUSE"
	} else if strings.HasPrefix(osDist, "opensuse") {
		return "OpenSUSE"
	} else if strings.HasPrefix(osDist, "debian") {
		return "Debian"
	} else if strings.HasPrefix(osDist, "coreos") {
		return "CoreOS"
	} else if strings.HasPrefix(osDist, "aliyun") {
		return "Aliyun"
	} else if strings.HasPrefix(osDist, "windows") {
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

type ImageInfo struct {
	Name     string
	OsArch   string
	OsType   string
	OsDistro string
}

func NormalizeImageInfo(imageName, osArch, osType, osDist string) ImageInfo {
	info := ImageInfo{}
	info.Name = imageName
	info.OsDistro = normalizeOsDistribution(osDist, imageName)
	info.OsType = normalizeOsType(osType, info.OsDistro)
	info.OsArch = normalizeOsArch(osArch, info.OsType, info.OsDistro)
	return info
}
