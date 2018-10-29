package db

import "time"

/// Global virtual resource namespace

var (
	globalVirtualResourceNamespace = false

	globalRegion = ""

	globalServiceType = ""

	globalsRbacEnabled = false
)

func EnableGlobalVirtualResourceNamespace() {
	globalVirtualResourceNamespace = true
}

func SetGlobalRegion(region string) {
	globalRegion = region
}

func GetGlobalRegion() string {
	return globalRegion
}

func SetGlobalServiceType(srvType string) {
	globalServiceType = srvType
}

func GetGlobalServiceType() string {
	return globalServiceType
}

func EnableGlobalRbac(refreshInterval time.Duration, retryInterval time.Duration) {
	globalsRbacEnabled = true
	PolicyManager.start(refreshInterval, retryInterval)
}

func IsGlobalRbacEnabled() bool {
	return globalsRbacEnabled
}
