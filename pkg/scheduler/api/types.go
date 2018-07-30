package api

import (
	"fmt"

	"github.com/yunionio/pkg/util/sets"
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

	AggregateStrategyRequire = "require"
	AggregateStrategyExclude = "exclude"
	AggregateStrategyPrefer  = "prefer"
	AggregateStrategyAvoid   = "avoid"

	// Baremetal related const
	DISK_CONF_RAID0  = "raid0"
	DISK_CONF_RAID1  = "raid1"
	DISK_CONF_RAID5  = "raid5"
	DISK_CONF_RAID10 = "raid10"
	DISK_CONF_NONE   = "none"

	DEFAULT_DISK_CONF = DISK_CONF_NONE

	DISK_TYPE_ROTATE = "rotate"
	DISK_TYPE_SSD    = "ssd"
	DISK_TYPE_HYBRID = "hybrid"

	DEFAULT_DISK_TYPE = DISK_TYPE_ROTATE

	DISK_DRIVER_MEGARAID   = "MegaRaid"
	DISK_DRIVER_LINUX      = "Linux"
	DISK_DRIVER_HPSARAID   = "HPSARaid"
	DISK_DRIVER_MPT2SAS    = "Mpt2SAS"
	DISK_DRIVER_MARVELRAID = "MarvelRaid"
	DISK_DRIVER_PCIE       = "PCIE"

	HDD_DISK_SPEC_TYPE = "HDD"
	SSD_DISK_SPEC_TYPE = "SSD"

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
	)

	BaremetalDefaultDiskConfig = BaremetalDiskConfig{
		Type:  DISK_TYPE_HYBRID,
		Conf:  DISK_CONF_NONE,
		Count: 0,
	}

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

type BaremetalStorage struct {
	Slot         int    `json:"slot"`
	Status       string `json:"status"`
	Rotate       bool   `json:"rotate"`
	Adapter      int    `json:"adapter"`
	Driver       string `json:"driver"`
	Model        string `json:"model"`
	Enclosure    int    `json:"enclousure"`
	Size         int64  `json:"size"`
	MinStripSize int64  `json:"min_strip_size,omitempty"`
	MaxStripSize int64  `json:"max_strip_size,omitempty"`
	Index        int64  `json:"index"`
}

type BaremetalDiskConfig struct {
	// disk type
	Type string `json:"type"`
	// raid config
	Conf         string  `json:"conf"`
	Count        int64   `json:"count"`
	Range        []int64 `json:"range"`
	Splits       string  `json:"splits"`
	Adapter      *int    `json:"adapter"`
	Driver       string  `json:"driver"`
	Cachedbadbbu bool    `json:"cachedbadbbu"`
	Strip        int64   `json:"strip"`
	RA           bool    `json:"ra"`
	WT           bool    `json:"wt"`
	Direct       bool    `json:"direct"`
}
