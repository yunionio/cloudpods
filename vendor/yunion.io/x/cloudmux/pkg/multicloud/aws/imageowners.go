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

package aws

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func getSystemOwnerIds() []string {
	keys := make([]string, len(awsImagePublishers))
	idx := 0
	for key := range awsImagePublishers {
		keys[idx] = key
		idx += 1
	}
	return keys
}

func imageOwnerTypes2Strings(owners []TImageOwnerType, rawIds []string) []string {
	ownerIds := make([]string, 0)
	for i := range owners {
		switch owners[i] {
		case ImageOwnerTypeSelf:
			ownerIds = append(ownerIds, "self")
		case ImageOwnerTypeSystem:
			sysOwnerIds := getSystemOwnerIds()
			ownerIds = append(ownerIds, sysOwnerIds...)
		}
	}
	ownerIds = append(ownerIds, rawIds...)
	return ownerIds
}

type SAWSImagePublisherInfo struct {
	GetOSType     func(image SImage) string
	GetOSDist     func(image SImage) string
	GetOSVersion  func(image SImage) string
	GetOSBuildID  func(image SImage) string
	CompareBuilds func(v1, v2 string) int
}

var rhel = SAWSImagePublisherInfo{
	GetOSType: func(image SImage) string {
		return "Linux"
	},
	GetOSDist: func(image SImage) string {
		return "RHEL"
	},
	GetOSVersion: func(image SImage) string {
		parts := strings.Split(image.ImageName, "-")
		if len(parts) >= 2 {
			parts = strings.Split(parts[1], "_")
			return parts[0]
		}
		return ""
	},
	GetOSBuildID: func(image SImage) string {
		parts := strings.Split(image.ImageName, "-")
		if len(parts) >= 2 {
			return parts[2]
		}
		return ""
	},
}

var (
	debianDatePattern = regexp.MustCompile(`-\d{4}-\d{2}-\d{2}-`)
)

var debian = SAWSImagePublisherInfo{
	GetOSType: func(image SImage) string {
		return "Linux"
	},
	GetOSDist: func(image SImage) string {
		return "Debian"
	},
	GetOSVersion: func(image SImage) string {
		parts := strings.Split(image.ImageName, "-")
		if len(parts) >= 5 {
			return parts[1]
		}
		return ""
	},
	GetOSBuildID: func(image SImage) string {
		dateStr := debianDatePattern.FindString(image.ImageName)
		if len(dateStr) > 2 {
			return dateStr[1 : len(dateStr)-1]
		}
		return ""
	},
}

var (
	centosDatePattern = regexp.MustCompile(`\s+\d{4,8}(_\d+)?`)
)

var centos = SAWSImagePublisherInfo{
	GetOSType: func(image SImage) string {
		return "Linux"
	},
	GetOSDist: func(image SImage) string {
		if strings.Index(image.ImageName, "Atomic") > 0 {
			return "CentOS Atomic"
		} else {
			return "CentOS"
		}
	},
	GetOSVersion: func(image SImage) string {
		parts := strings.Split(image.ImageName, " ")
		if strings.Index(image.ImageName, "Atomic") > 0 {
			if regutils.MatchInteger(parts[3]) {
				return parts[3]
			} else {
				return "7"
			}
		} else if strings.HasPrefix(image.ImageName, "CentOS Linux ") {
			return parts[2]
		} else {
			return parts[1]
		}
	},
	GetOSBuildID: func(image SImage) string {
		build := centosDatePattern.FindString(image.ImageName)
		build = strings.TrimSpace(build)
		if strings.HasPrefix(build, "201") {
			build = build[2:]
		}
		return build
	},
}

var ubuntuReleases = map[string]string{
	"warty":    "4.10",
	"hoary":    "5.04",
	"breezy":   "5.10",
	"dapper":   "6.06",
	"edgy":     "6.10",
	"feisty":   "7.04",
	"gutsy":    "7.10",
	"hardy":    "8.04",
	"intrepid": "8.10",
	"jaunty":   "9.04",
	"karmic":   "9.10",
	"lucid":    "10.04",
	"maverick": "10.10",
	"natty":    "11.04",
	"oneiric":  "11.10",
	"precise":  "12.04",
	"quantal":  "12.10",
	"raring":   "13.04",
	"saucy":    "13.10",
	"trusty":   "14.04",
	"utopic":   "14.10",
	"vivid":    "15.04",
	"wily":     "15.10",
	"xenial":   "16.04",
	"yakkety":  "16.10",
	"zesty":    "17.04",
	"artful":   "17.10",
	"bionic":   "18.04",
	"cosmic":   "18.10",
	"disco":    "19.04",
	"focal":    "20.04",
	"jammy":    "22.04",
	"lunar":    "23.04",
}

var ubuntuReleasePattern = regexp.MustCompile(`-\d+\.\d+-`)

var ubuntu = SAWSImagePublisherInfo{
	GetOSType: func(image SImage) string {
		return "Linux"
	},
	GetOSDist: func(image SImage) string {
		if strings.HasPrefix(image.ImageName, "ubuntu-minimal/") {
			return "Ubuntu Minimal"
		}
		if strings.HasPrefix(image.ImageName, "ubuntu/") {
			return "Ubuntu"
		}
		if strings.HasPrefix(image.ImageName, "ubuntu-rolling-") || strings.HasPrefix(image.ImageName, "ubuntu-core") || strings.Index(image.ImageName, "core-edge") > 0 {
			return "Ubuntu Core"
		}
		return "Ubuntu"
	},
	GetOSVersion: func(image SImage) string {
		relStr := ubuntuReleasePattern.FindString(image.ImageName)
		if len(relStr) > 2 {
			return relStr[1 : len(relStr)-1]
		}
		parts := strings.Split(image.ImageName, "/")
		if len(parts) >= 4 {
			parts = strings.Split(parts[3], "-")
			if len(parts) >= 2 {
				relName := strings.ToLower(parts[1])
				if _, ok := ubuntuReleases[relName]; ok {
					return ubuntuReleases[relName]
				} else {
					return relName
				}
			}
		}
		if strings.HasPrefix(image.ImageName, "ubuntu-rolling-") {
			parts := strings.Split(image.ImageName, "-")
			if len(parts) > 3 {
				return strings.ToLower(parts[2])
			}
		}
		return ""
	},
	GetOSBuildID: func(image SImage) string {
		parts := strings.Split(image.ImageName, "-")
		return parts[len(parts)-1]
	},
}

var (
	SUSE_SLES              = regexp.MustCompile(`suse-sles-\d+-v?\d+-`)
	SUSE_SLES_SP           = regexp.MustCompile(`suse-sles-\d+-sp\d+-v?\d+-`)
	SUSE_SLES_RIGHTLINK    = regexp.MustCompile(`suse-sles-\d+-rightscale-v?\d+-`)
	SUSE_SLES_RIGHTLINK_SP = regexp.MustCompile(`suse-sles-\d+-sp\d+-rightscale-v?\d+-`)
	SUSE_SLES_SAPCAL       = regexp.MustCompile(`suse-sles-\d+-sapcal-v?\d+-`)
	SUSE_SLES_SAPCAL_SP    = regexp.MustCompile(`suse-sles-\d+-sp\d+-sapcal-v?\d+-`)
	SUSE_SLES_BYOS         = regexp.MustCompile(`suse-sles-\d+-byos-v?\d+-`)
	SUSE_SLES_BYOS_SP      = regexp.MustCompile(`suse-sles-\d+-sp\d+-byos-v?\d+-`)
	SUSE_SLES_SAP          = regexp.MustCompile(`suse-sles-sap-\d+-v\d+-`)
	SUSE_SLES_SAP_SP       = regexp.MustCompile(`suse-sles-sap-\d+-sp\d+-v\d+-`)
	SUSE_SLES_SAP_BYOS     = regexp.MustCompile(`suse-sles-sap-\d+-byos-v?\d+-`)
	SUSE_SLES_SAP_BYOS_SP  = regexp.MustCompile(`suse-sles-sap-\d+-sp\d+-byos-v?\d+-`)

	SUSE_CAASP_CLUSTER_BYOS = regexp.MustCompile(`suse-caasp-\d+-\d+-cluster-byos-v?\d+-`)
	SUSE_CAASP_ADMIN_BYOS   = regexp.MustCompile(`suse-caasp-\d+-\d+-admin-byos-v?\d+-`)

	SUSE_MANAGER_SERVER_BYOS = regexp.MustCompile(`suse-manager-\d+-\d+-server-byos-v?\d+-`)
	SUSE_MANAGER_PROXY_BYOS  = regexp.MustCompile(`suse-manager-\d+-\d+-proxy-byos-v?\d+-`)
)

func getBuildId(ver string) string {
	if ver[0] == 'v' {
		return ver[1:]
	} else {
		return ver
	}
}

var suse = SAWSImagePublisherInfo{
	GetOSType: func(image SImage) string {
		return "Linux"
	},
	GetOSDist: func(image SImage) string {
		switch {
		case SUSE_SLES.MatchString(image.ImageName), SUSE_SLES_SP.MatchString(image.ImageName):
			return "SUSE Linux Enterpise Server"
		case SUSE_SLES_RIGHTLINK.MatchString(image.ImageName), SUSE_SLES_RIGHTLINK_SP.MatchString(image.ImageName):
			return "SUSE Linux Enterpise Server with RightLink"
		case SUSE_SLES_SAPCAL.MatchString(image.ImageName), SUSE_SLES_SAPCAL_SP.MatchString(image.ImageName):
			return "SUSE Linux Enterpise Server for SAP CAL"
		case SUSE_SLES_BYOS.MatchString(image.ImageName), SUSE_SLES_BYOS_SP.MatchString(image.ImageName):
			return "SUSE Linux Enterpise Server BYOS"
		case SUSE_SLES_SAP.MatchString(image.ImageName), SUSE_SLES_SAP_SP.MatchString(image.ImageName):
			return "SUSE Linux Enterpise Server for SAP Application"
		case SUSE_SLES_SAP_BYOS.MatchString(image.ImageName), SUSE_SLES_SAP_BYOS_SP.MatchString(image.ImageName):
			return "SUSE Linux Enterpise Server for SAP Application BYOS"
		case SUSE_CAASP_CLUSTER_BYOS.MatchString(image.ImageName):
			return "SUSE CaaSP Cluster Node"
		case SUSE_CAASP_ADMIN_BYOS.MatchString(image.ImageName):
			return "SUSE CaaSP Admin Node"
		case SUSE_MANAGER_SERVER_BYOS.MatchString(image.ImageName):
			return "SUSE Manager Server"
		case SUSE_MANAGER_PROXY_BYOS.MatchString(image.ImageName):
			return "SUSE Manager Proxy"
		}
		return "SUSE"
	},
	GetOSVersion: func(image SImage) string {
		parts := strings.Split(image.ImageName, "-")
		switch {
		case SUSE_SLES.MatchString(image.ImageName):
			return parts[2]
		case SUSE_SLES_SP.MatchString(image.ImageName):
			return fmt.Sprintf("%s.%s", parts[2], parts[3][2:])
		case SUSE_SLES_RIGHTLINK.MatchString(image.ImageName):
			return parts[2]
		case SUSE_SLES_RIGHTLINK_SP.MatchString(image.ImageName):
			return fmt.Sprintf("%s.%s", parts[2], parts[3][2:])
		case SUSE_SLES_SAPCAL.MatchString(image.ImageName):
			return parts[2]
		case SUSE_SLES_SAPCAL_SP.MatchString(image.ImageName):
			return fmt.Sprintf("%s.%s", parts[2], parts[3][2:])
		case SUSE_SLES_BYOS.MatchString(image.ImageName):
			return parts[2]
		case SUSE_SLES_BYOS_SP.MatchString(image.ImageName):
			return fmt.Sprintf("%s.%s", parts[2], parts[3][2:])
		case SUSE_SLES_SAP.MatchString(image.ImageName):
			return parts[3]
		case SUSE_SLES_SAP_SP.MatchString(image.ImageName):
			return fmt.Sprintf("%s.%s", parts[3], parts[4][2:])
		case SUSE_SLES_SAP_BYOS.MatchString(image.ImageName):
			return parts[3]
		case SUSE_SLES_SAP_BYOS_SP.MatchString(image.ImageName):
			return fmt.Sprintf("%s.%s", parts[3], parts[4][2:])
		case SUSE_CAASP_CLUSTER_BYOS.MatchString(image.ImageName):
			return fmt.Sprintf("%s.%s", parts[2], parts[3])
		case SUSE_CAASP_ADMIN_BYOS.MatchString(image.ImageName):
			return fmt.Sprintf("%s.%s", parts[2], parts[3])
		case SUSE_MANAGER_SERVER_BYOS.MatchString(image.ImageName):
			return fmt.Sprintf("%s.%s", parts[2], parts[3])
		case SUSE_MANAGER_PROXY_BYOS.MatchString(image.ImageName):
			return fmt.Sprintf("%s.%s", parts[2], parts[3])
		}
		return ""
	},
	GetOSBuildID: func(image SImage) string {
		parts := strings.Split(image.ImageName, "-")
		switch {
		case SUSE_SLES.MatchString(image.ImageName):
			return getBuildId(parts[3])
		case SUSE_SLES_SP.MatchString(image.ImageName):
			return getBuildId(parts[4])
		case SUSE_SLES_RIGHTLINK.MatchString(image.ImageName):
			return getBuildId(parts[4])
		case SUSE_SLES_RIGHTLINK_SP.MatchString(image.ImageName):
			return getBuildId(parts[5])
		case SUSE_SLES_SAPCAL.MatchString(image.ImageName):
			return getBuildId(parts[4])
		case SUSE_SLES_SAPCAL_SP.MatchString(image.ImageName):
			return getBuildId(parts[5])
		case SUSE_SLES_BYOS.MatchString(image.ImageName):
			return getBuildId(parts[4])
		case SUSE_SLES_BYOS_SP.MatchString(image.ImageName):
			return getBuildId(parts[5])
		case SUSE_SLES_SAP.MatchString(image.ImageName):
			return getBuildId(parts[4])
		case SUSE_SLES_SAP_SP.MatchString(image.ImageName):
			return getBuildId(parts[5])
		case SUSE_SLES_SAP_BYOS.MatchString(image.ImageName):
			return getBuildId(parts[5])
		case SUSE_SLES_SAP_BYOS_SP.MatchString(image.ImageName):
			return getBuildId(parts[6])
		case SUSE_CAASP_CLUSTER_BYOS.MatchString(image.ImageName):
			return getBuildId(parts[6])
		case SUSE_CAASP_ADMIN_BYOS.MatchString(image.ImageName):
			return getBuildId(parts[6])
		case SUSE_MANAGER_SERVER_BYOS.MatchString(image.ImageName):
			return getBuildId(parts[6])
		case SUSE_MANAGER_PROXY_BYOS.MatchString(image.ImageName):
			return getBuildId(parts[6])
		}
		return ""
	},
}

var coreosVersionTable = map[string]int{
	"alpha":  0,
	"beta":   1,
	"stable": 2,
}

var coreos = SAWSImagePublisherInfo{
	GetOSType: func(image SImage) string {
		return "Linux"
	},
	GetOSDist: func(image SImage) string {
		return "CoreOS"
	},
	GetOSVersion: func(image SImage) string {
		parts := strings.Split(image.ImageName, "-")
		subparts := strings.Split(parts[2], ".")
		return subparts[0]
	},
	GetOSBuildID: func(image SImage) string {
		parts := strings.Split(image.ImageName, "-")
		return fmt.Sprintf("%s-%s", parts[1], parts[2])
	},
	CompareBuilds: func(v1, v2 string) int {
		parts1 := strings.Split(v1, "-")
		parts2 := strings.Split(v2, "-")
		majorV1 := coreosVersionTable[parts1[0]]
		majorV2 := coreosVersionTable[parts2[0]]
		if majorV1 != majorV2 {
			return majorV1 - majorV2
		}
		parts1 = strings.Split(parts1[1], ".")
		parts2 = strings.Split(parts2[1], ".")
		for i := 0; i < len(parts1) && i < len(parts2); i += 1 {
			n1, _ := strconv.Atoi(parts1[i])
			n2, _ := strconv.Atoi(parts2[i])
			if n1 != n2 {
				return n1 - n2
			}
		}
		return len(parts1) - len(parts2)
	},
}

var (
	sqlServerPattern  = regexp.MustCompile(`-SQL_(\d+)_(\w+)-`)
	dotnetcorePattern = regexp.MustCompile(`-dotnetcore-`)
)

var windowsServer = SAWSImagePublisherInfo{
	GetOSType: func(image SImage) string {
		if strings.HasPrefix(image.ImageName, "ubuntu-") || strings.HasPrefix(image.ImageName, "amzn-ami-") || strings.HasPrefix(image.ImageName, "amzn2-ami-") {
			return "Linux"
		} else {
			return "Windows"
		}
	},
	GetOSDist: func(image SImage) string {
		osStr := "Windows Server"
		if strings.HasPrefix(image.ImageName, "ubuntu-") {
			osStr = "Ubuntu"
		} else if strings.HasPrefix(image.ImageName, "amzn-ami-") || strings.HasPrefix(image.ImageName, "amzn2-ami-") {
			osStr = "Amazon Linux"
		}
		apps := make([]string, 0)
		matchApp := sqlServerPattern.FindStringSubmatch(image.ImageName)
		if len(matchApp) > 0 {
			apps = append(apps, fmt.Sprintf("SQL Server %s %s", matchApp[1], matchApp[2]))
		}
		if dotnetcorePattern.MatchString(image.ImageName) {
			apps = append(apps, ".Net Core")
		}
		if len(apps) > 0 {
			osStr = fmt.Sprintf("%s with %s", osStr, strings.Join(apps, " "))
		}
		return osStr
	},
	GetOSVersion: func(image SImage) string {
		if strings.HasPrefix(image.ImageName, "ubuntu-") {
			return ubuntu.GetOSVersion(image)
		}
		if strings.HasPrefix(image.ImageName, "amzn-ami-") || strings.HasPrefix(image.ImageName, "amzn2-ami-") {
			return amazon.GetOSVersion(image)
		}
		parts := strings.Split(image.ImageName, "-")
		return strings.Join(parts[1:len(parts)-1], " ")
	},
	GetOSBuildID: func(image SImage) string {
		if strings.HasPrefix(image.ImageName, "ubuntu-") {
			return ubuntu.GetOSBuildID(image)
		}
		if strings.HasPrefix(image.ImageName, "amzn-ami-") || strings.HasPrefix(image.ImageName, "amzn2-ami-") {
			return amazon.GetOSBuildID(image)
		}
		parts := strings.Split(image.ImageName, "-")
		return parts[len(parts)-1]
	},
}

var (
	amazonVersionPattern  = regexp.MustCompile(`-(\d{4})\.(\d{2})\.(rc-\d+|\d+)(\.(\d+))?`)
	amazonVersionPattern2 = regexp.MustCompile(`-(\d{1,2})\.(\d{1,2})\.(\d{8})(\.(\d+))?`)
)

var amazon = SAWSImagePublisherInfo{
	GetOSType: func(image SImage) string {
		return "Linux"
	},
	GetOSDist: func(image SImage) string {
		if strings.HasPrefix(image.ImageName, "amzn-ami-minimal-") || strings.HasPrefix(image.ImageName, "amzn2-ami-minimal-") {
			return "Amazon Linux Minimal"
		} else {
			return "Amazon Linux"
		}
	},
	GetOSVersion: func(image SImage) string {
		verStrs := amazonVersionPattern2.FindStringSubmatch(image.ImageName)
		if len(verStrs) > 3 {
			return fmt.Sprintf("%s.%s.%s", verStrs[1], verStrs[2], verStrs[3][:6])
		}
		verStrs = amazonVersionPattern.FindStringSubmatch(image.ImageName)
		if len(verStrs) > 3 {
			return fmt.Sprintf("%s.%s.%s", verStrs[1], verStrs[2], verStrs[3])
		}
		return ""
	},
	GetOSBuildID: func(image SImage) string {
		verStrs := amazonVersionPattern2.FindStringSubmatch(image.ImageName)
		if len(verStrs) > 5 && len(verStrs[5]) > 0 {
			return fmt.Sprintf("%s.%s", verStrs[3], verStrs[5])
		} else if len(verStrs) > 3 {
			return verStrs[3]
		}
		verStrs = amazonVersionPattern.FindStringSubmatch(image.ImageName)
		if len(verStrs) > 5 {
			return verStrs[5]
		}
		return ""
	},
}

var awsImagePublishers = map[string]SAWSImagePublisherInfo{
	"841258680906": rhel,          // china
	"309956199498": rhel,          // international
	"673060587306": debian,        // china
	"379101102735": debian,        // international
	"718707510307": centos,        // china
	"410186602215": centos,        // international
	"837727238323": ubuntu,        // china
	"099720109477": ubuntu,        // internaltional
	"841869936221": suse,          // china
	"013907871322": suse,          // international
	"280032941352": coreos,        // china
	"595879546273": coreos,        // international
	"016951021795": windowsServer, // china
	"801119661308": windowsServer, // international
	"141808717104": amazon,        // china
	"137112412989": amazon,        // international
}

func getImageOSType(image SImage) string {
	ownerInfo, ok := awsImagePublishers[image.OwnerId]
	if ok {
		return ownerInfo.GetOSType(image)
	}
	if strings.Contains(strings.ToLower(image.Platform), "windows") {
		return string(cloudprovider.OsTypeWindows)
	}
	return string(cloudprovider.OsTypeLinux)
}

func getImageOSDist(image SImage) string {
	ownerInfo, ok := awsImagePublishers[image.OwnerId]
	if ok {
		return ownerInfo.GetOSDist(image)
	}
	return ""
}

func getImageOSVersion(image SImage) string {
	ownerInfo, ok := awsImagePublishers[image.OwnerId]
	if ok {
		return ownerInfo.GetOSVersion(image)
	}
	return ""
}

func getImageOSBuildID(image SImage) string {
	ownerInfo, ok := awsImagePublishers[image.OwnerId]
	if ok {
		return ownerInfo.GetOSBuildID(image)
	}
	return ""
}

func comapreImageBuildIds(ver1 string, img2 SImage) int {
	ownerInfo, ok := awsImagePublishers[img2.OwnerId]
	if ok && ownerInfo.CompareBuilds != nil {
		return ownerInfo.CompareBuilds(ver1, getImageOSBuildID(img2))
	}
	return strings.Compare(ver1, getImageOSBuildID(img2))
}

func getImageType(image *SImage) cloudprovider.TImageType {
	_, ok := awsImagePublishers[image.OwnerId]
	if ok {
		return cloudprovider.ImageTypeSystem
	}
	if !image.Public {
		return cloudprovider.ImageTypeCustomized
	}
	return cloudprovider.ImageTypeMarket
}
