package consts

/// Global virtual resource namespace

var (
	globalVirtualResourceNamespace = false
)

func EnableGlobalVirtualResourceNamespace() {
	globalVirtualResourceNamespace = true
}

func IsGlobalVirtualResourceNamespace() bool {
	return globalVirtualResourceNamespace
}
