package azure

import (
	"fmt"
	"strings"
)

type SPublisherDriver struct {
	OsType       string
	GetOsDist    func(offser, sku, version string) string
	GetOsVersion func(offser, sku, version string) string
	GetOsArch    func(offser, sku, version string) string
	GetName      func(offser, sku, version string) string
}

var publisherDrivers = map[string]SPublisherDriver{
	// Microsoft Windows Server
	"MicrosoftWindowsServer": {
		OsType: "Windows",
		GetOsDist: func(offer, sku, version string) string {
			parts := strings.Split(sku, "-")
			return fmt.Sprintf("Windows Server %s", strings.Join(parts, " "))
		},
		GetOsVersion: func(offer, sku, version string) string {
			parts := strings.Split(sku, "-")
			return parts[0]
		},
		GetOsArch: func(offer, sku, version string) string {
			return "x86_64"
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s-%s", offer, sku, version)
		},
	},
	// RHEL
	"RedHat": {
		OsType: "Linux",
		GetOsDist: func(offer, sku, version string) string {
			return "RHEL"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return "x86_64"
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s", offer, version)
		},
	},
	// Ubuntu
	"Canonical": {
		OsType: "Linux",
		GetOsDist: func(offer, sku, version string) string {
			return "Ubuntu"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return "x86_64"
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s", offer, version)
		},
	},
	// CentOS
	"OpenLogic": {
		OsType: "Linux",
		GetOsDist: func(offer, sku, version string) string {
			return "CentOS"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return "x86_64"
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s", offer, version)
		},
	},
	// SUSE
	"SUSE": {
		OsType: "Linux",
		GetOsDist: func(offer, sku, version string) string {
			return "SUSE"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return "x86_64"
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s-%s", offer, sku, version)
		},
	},
	// CoreOS
	"CoreOS": {
		OsType: "Linux",
		GetOsDist: func(offer, sku, version string) string {
			return "CoreOS"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return version
		},
		GetOsArch: func(offer, sku, version string) string {
			return "x86_64"
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s-%s", offer, sku, version)
		},
	},
	// Debian
	"credativ": {
		OsType: "Linux",
		GetOsDist: func(offer, sku, version string) string {
			return "Debian"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return "x86_64"
		},
		GetName: func(offer, sku, version string) string {
			return fmt.Sprintf("%s-%s", offer, version)
		},
	},
	// FreeBSD
	"MicrosoftOSTC": {
		OsType: "FreeBSD",
		GetOsDist: func(offer, sku, version string) string {
			return "FreeBSD"
		},
		GetOsVersion: func(offer, sku, version string) string {
			return sku
		},
		GetOsArch: func(offer, sku, version string) string {
			return "x86_64"
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
	driver, ok := publisherDrivers[publisher]
	if ok {
		return driver.OsType
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
	return "x86_64"
}
