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

package azure

import (
	"fmt"
	"strings"

	"yunion.io/x/cloudmux/pkg/apis"
)

type SPublisherDriver struct {
	OsType       string
	GetOffers    func() []string
	GetSkus      func(offer string) []string
	GetOsDist    func(offser, sku, version string) string
	GetOsVersion func(offser, sku, version string) string
	GetOsArch    func(offser, sku, version string) string
	GetName      func(offser, sku, version string) string
}

var publisherDrivers = map[string]SPublisherDriver{
	// Microsoft Windows Server
	"microsoftwindowsserver": {
		OsType: "Windows",
		GetOffers: func() []string {
			return []string{"WindowsServer", "2019-Datacenter"}
		},
		GetSkus: func(offer string) []string {
			switch offer {
			case "WindowsServer":
				return []string{"2016-Datacenter", "2019-Datacenter"}
			default:
				return []string{}
			}
		},
		GetOsDist: func(offer, sku, version string) string {
			parts := strings.Split(sku, "-")
			return fmt.Sprintf("Windows Server %s", strings.Join(parts, " "))
		},
		GetOsVersion: func(offer, sku, version string) string {
			parts := strings.Split(sku, "-")
			return parts[0]
		},
		GetOsArch: func(offer, sku, version string) string {
			return apis.OS_ARCH_X86_64
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s-%s", offer, sku, version)
		},
	},
	// RHEL
	"redhat": {
		OsType: "Linux",
		GetOffers: func() []string {
			return []string{"rhel-75"}
		},
		GetSkus: func(offer string) []string {
			switch offer {
			case "rhel-75":
				return []string{"standard"}
			default:
				return []string{}
			}
		},
		GetOsDist: func(offer, sku, version string) string {
			return "RHEL"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return apis.OS_ARCH_X86_64
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s", offer, version)
		},
	},
	// Ubuntu
	"canonical": {
		OsType: "Linux",
		GetOffers: func() []string {
			return []string{"UbuntuServer"}
		},
		GetSkus: func(offer string) []string {
			switch offer {
			case "UbuntuServer":
				return []string{"14.04.5-LTS", "16.04-LTS", "17.10", "18.04-LTS", "18_04-lts-gen2", "19.04", "19_04-gen2"}
			default:
				return []string{}
			}
		},
		GetOsDist: func(offer, sku, version string) string {
			return "Ubuntu"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return apis.OS_ARCH_X86_64
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s", offer, version)
		},
	},
	// CentOS
	"openlogic": {
		OsType: "Linux",
		GetOffers: func() []string {
			return []string{"CentOS"}
		},
		GetSkus: func(offer string) []string {
			switch offer {
			case "CentOS":
				return []string{"6.9", "7.3", "7.4", "7.5", "7.6", "7.7", "8.0", "7_4-gen2", "7_5-gen2", "7_6-gen2", "7_7-gen2", "8_0-gen2", "8_1-gen2"}
			default:
				return []string{}
			}
		},
		GetOsDist: func(offer, sku, version string) string {
			return "CentOS"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return apis.OS_ARCH_X86_64
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s", offer, version)
		},
	},
	// SUSE
	"suse": {
		OsType: "Linux",
		GetOffers: func() []string {
			return []string{"SLES"}
		},
		GetSkus: func(offer string) []string {
			switch offer {
			case "SLES":
				return []string{"12-SP4", "12-SP4-gen2"}
			default:
				return []string{}
			}
		},
		GetOsDist: func(offer, sku, version string) string {
			return "SUSE"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return apis.OS_ARCH_X86_64
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s-%s", offer, sku, version)
		},
	},
	// CoreOS
	"coreos": {
		OsType: "Linux",
		GetOffers: func() []string {
			return []string{"CoreOS"}
		},
		GetSkus: func(offer string) []string {
			switch offer {
			case "CoreOS":
				return []string{"Alpha", "Beta", "Stable"}
			default:
				return []string{}
			}
		},
		GetOsDist: func(offer, sku, version string) string {
			return "CoreOS"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return version
		},
		GetOsArch: func(offer, sku, version string) string {
			return apis.OS_ARCH_X86_64
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s-%s", offer, sku, version)
		},
	},
	// Debian
	"credativ": {
		OsType: "Linux",
		GetOffers: func() []string {
			return []string{"Debian"}
		},
		GetSkus: func(offer string) []string {
			switch offer {
			case "Debian":
				return []string{"8", "9"}
			default:
				return []string{}
			}
		},
		GetOsDist: func(offer, sku, version string) string {
			return "Debian"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return apis.OS_ARCH_X86_64
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s", offer, version)
		},
	},
	// FreeBSD
	"microsoftostc": {
		OsType: "FreeBSD",
		GetOffers: func() []string {
			return []string{"FreeBSD"}
		},
		GetSkus: func(offer string) []string {
			switch offer {
			case "FreeBSD":
				return []string{"10.4", "11.2", "12.0"}
			default:
				return []string{}
			}
		},
		GetOsDist: func(offer, sku, version string) string {
			return "FreeBSD"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return apis.OS_ARCH_X86_64
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s", offer, version)
		},
	},
}

var knownPublishers []string

func init() {
	knownPublishers = make([]string, len(publisherDrivers))
	i := 0
	for k := range publisherDrivers {
		knownPublishers[i] = strings.ToLower(k)
		i += 1
	}
}

func publisherGetName(publisher, offer, sku, version string) string {
	driver, ok := publisherDrivers[publisher]
	if ok {
		return driver.GetName(offer, sku, version)
	}
	return fmt.Sprintf("%s-%s-%s", offer, sku, version)
}

func publisherGetOsType(publisher string) string {
	for _publisher, driver := range publisherDrivers {
		if strings.ToLower(_publisher) == strings.ToLower(publisher) {
			return driver.OsType
		}
	}
	return "Linux"
}

func publisherGetOsDist(publisher, offer, sku, version string) string {
	driver, ok := publisherDrivers[publisher]
	if ok {
		return driver.GetOsDist(offer, sku, version)
	}
	return offer
}

func publisherGetOsVersion(publisher, offer, sku, version string) string {
	driver, ok := publisherDrivers[publisher]
	if ok {
		return driver.GetOsVersion(offer, sku, version)
	}
	return sku
}

func publisherGetOsArch(publisher, offer, sku, version string) string {
	driver, ok := publisherDrivers[publisher]
	if ok {
		return driver.GetOsArch(offer, sku, version)
	}
	return apis.OS_ARCH_X86_64
}
