package api

import (
	"fmt"

	"yunion.io/x/pkg/util/sets"
)

const (
	HostTypeHost         = "host"
	HostTypeBaremetal    = "baremetal"
	SchedTypeGuest       = "guest"
	SchedTypeBaremetal   = "baremetal"
	SchedTypeContainer   = "container"
	SchedTypeEsxi        = "esxi"
	SchedTypeHyperV      = "hyperv"
	SchedTypeKvm         = "kvm"
	HostHypervisorForKvm = "hypervisor"
	HostTypeAliyun       = "aliyun"
	HostTypeAzure        = "azure"
	HostTypeKubelet      = "kubelet"

	AggregateStrategyRequire = "require"
	AggregateStrategyExclude = "exclude"
	AggregateStrategyPrefer  = "prefer"
	AggregateStrategyAvoid   = "avoid"

	// passthrough device type
	DIRECT_PCI_TYPE = "PCI"
	GPU_HPC_TYPE    = "GPU-HPC"
	GPU_VGA_TYPE    = "GPU-VGA"
	USB_TYPE        = "USB"
	NIC_TYPE        = "NIC"

	// Hard code vendor const
	NVIDIA           = "NVIDIA"
	AMD              = "AMD"
	NVIDIA_VENDOR_ID = "10de"
	AMD_VENDOR_ID    = "1002"
)

var (
	AggregateStrategySets = sets.NewString(
		AggregateStrategyRequire,
		AggregateStrategyExclude,
		AggregateStrategyPrefer,
		AggregateStrategyAvoid,
	)

	PublicCloudProviders = sets.NewString(
		HostTypeAliyun,
		HostTypeAzure,
	)

	ValidGpuTypes = sets.NewString(
		GPU_HPC_TYPE,
		GPU_VGA_TYPE,
	)

	ValidPassthroughTypes = sets.NewString(
		DIRECT_PCI_TYPE,
		USB_TYPE,
		NIC_TYPE,
	).Union(ValidGpuTypes)

	IsolatedVendorIDMap = map[string]string{
		NVIDIA: NVIDIA_VENDOR_ID,
		AMD:    AMD_VENDOR_ID,
	}

	IsolatedIDVendorMap = map[string]string{}
)

func init() {
	for k, v := range IsolatedVendorIDMap {
		IsolatedIDVendorMap[v] = k
	}
}

func AggregateStrategyCheck(strategy string) (err error) {
	if !AggregateStrategySets.Has(strategy) {
		err = fmt.Errorf("Strategy %q must in set %v", strategy, AggregateStrategySets.List())
	}
	return
}
