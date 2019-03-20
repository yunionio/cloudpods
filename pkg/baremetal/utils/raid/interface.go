package raid

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
)

type IRaidDriver interface {
	ParsePhyDevs() error
	GetName() string
	GetAdapters() []IRaidAdapter
	PreBuildRaid(confs []*api.BaremetalDiskConfig, adapterIdx int) error

	CleanRaid() error
}

type IRaidAdapter interface {
	GetIndex() int
	PreBuildRaid(confs []*api.BaremetalDiskConfig) error
	GetLogicVolumes() ([]int, error)
	RemoveLogicVolumes() error
	GetDevices() []*baremetal.BaremetalStorage

	BuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error
	BuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error
	BuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error
	BuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error
	BuildNoneRaid(devs []*baremetal.BaremetalStorage) error
}
