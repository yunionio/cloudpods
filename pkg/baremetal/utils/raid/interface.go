package raid

import (
	"yunion.io/x/onecloud/pkg/compute/baremetal"
)

type IRaidDriver interface {
	ParsePhyDevs() bool
	GetPhyDevs() []*baremetal.BaremetalStorage
	GetLogicVolumes() []int
	GetName() string
}
