package raid

import (
	"yunion.io/x/onecloud/pkg/compute/baremetal"
)

type IRaidDriver interface {
	ParsePhyDevs() error
	GetName() string
	GetAdapters() []IRaidAdapter
	PreBuildRaid(confs []*baremetal.BaremetalDiskConfig, adapterIdx int) error

	CleanRaid() error
}

type IRaidAdapter interface {
	GetIndex() int
	PreBuildRaid(confs []*baremetal.BaremetalDiskConfig) error
	GetLogicVolumes() ([]int, error)
	RemoveLogicVolumes() error
	GetDevices() []*baremetal.BaremetalStorage

	BuildRaid0(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error
	BuildRaid1(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error
	BuildRaid5(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error
	BuildRaid10(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error
	BuildNoneRaid(devs []*baremetal.BaremetalStorage, conf *baremetal.BaremetalDiskConfig) error
}
