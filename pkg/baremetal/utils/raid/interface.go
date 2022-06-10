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

package raid

import (
	"io"

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
	GetLogicVolumes() ([]*RaidLogicalVolume, error)
	RemoveLogicVolumes() error
	GetDevices() []*baremetal.BaremetalStorage

	BuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error
	BuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error
	BuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error
	BuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error
	BuildNoneRaid(devs []*baremetal.BaremetalStorage) error
	PostBuildRaid() error
}

type IExecTerm interface {
	Run(cmds ...string) ([]string, error)
	RunWithInput(input io.Reader, cmds ...string) ([]string, error)
}
