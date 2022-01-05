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

package megactl

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
)

var (
	_ iDriver = new(perccliDriver)
)

type perccliDriver struct{}

func newPerccliDriver() iDriver {
	return new(perccliDriver)
}

func (_ *perccliDriver) GetName() string {
	return "MegaRAID_Perccli"
}

func (_ *perccliDriver) GetAdapterConstructCmd() string {
	return getPerccliCmd("/call", "show", "|", "grep", "-iE", `'^(Controller|Product Name|Serial Number|Bus Number|Device Number|Function Number)\s='`)
}

func (_ *perccliDriver) NewAdaptor(term raid.IExecTerm) iAdaptor {
	return newPerccliAdaptor(term)
}

func (_ *perccliDriver) ClearForeignState(term raid.IExecTerm) error {
	cmd := getPerccliCmd("/call/fall", "delete")
	if _, err := term.Run(cmd); err != nil {
		return err
	}
	return nil
}

func getPerccliCmd(args ...string) string {
	bin := "/opt/MegaRAID/perccli/perccli"
	return raid.GetCommand(bin, args...)
}

type PerccliAdaptor struct {
	*StorcliAdaptor
	term raid.IExecTerm
}

func newPerccliAdaptor(term raid.IExecTerm) *PerccliAdaptor {
	return &PerccliAdaptor{
		StorcliAdaptor: newStorcliAdaptor(),
		term:           term,
	}
}

func (a *PerccliAdaptor) ParseLine(line string) {
	a.StorcliAdaptor.parseLine(line)
}

func (a *PerccliAdaptor) IsComplete() bool {
	return a.isComplete()
}

func (a *PerccliAdaptor) Key() string {
	return a.key()
}

func (a *PerccliAdaptor) GetPhyDevs() ([]*MegaRaidPhyDev, error) {
	return a.getMegaPhyDevs(a.getCmd, a.term)
}

func (a *PerccliAdaptor) ClearJBODDisks(devs []*MegaRaidPhyDev) {
	storcliClearJBODDisks(a.getAdaptorCmd, a.term, devs)
}

func (a *PerccliAdaptor) GetIndex() int {
	return a.Controller
}

func (a *PerccliAdaptor) getCmd(args ...string) string {
	return getPerccliCmd(args...)
}

func (a *PerccliAdaptor) getAdaptorCmd(args ...string) (string, error) {
	nargs := []string{fmt.Sprintf("/c%d", a.GetIndex())}
	nargs = append(nargs, args...)
	return a.getCmd(nargs...), nil
}

func (a *PerccliAdaptor) GetLogicVolumes() ([]*raid.RaidLogicalVolume, error) {
	cmd := a.getCmd(fmt.Sprintf("/c%d/vall", a.GetIndex()), "show")
	ret, err := a.term.Run(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "Get perccli logical volumes")
	}
	return parseStorcliLogicalVolumes(a.GetIndex(), ret)
}

func (a *PerccliAdaptor) RemoveLogicVolumes() error {
	lvIdx, err := a.GetLogicVolumes()
	if err != nil {
		return errors.Wrap(err, "GetLogicVolumes")
	}
	for _, i := range raid.ReverseLogicalArray(lvIdx) {
		cmd := a.getCmd(fmt.Sprintf("/c%d/v%d", a.GetIndex(), i.Index), "delete", "force")
		if _, err := a.term.Run(cmd); err != nil {
			return errors.Wrapf(err, "remove adaptor %d lv %d", a.GetIndex(), i.Index)
		}
	}
	return nil
}

func (a *PerccliAdaptor) BuildRaid0(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, func(bs []*baremetal.BaremetalStorage, bdc *compute.BaremetalDiskConfig) error {
		return storcliBuildRaid(a.getAdaptorCmd, a.term, devs, conf, 0)
	})
}

func (a *PerccliAdaptor) BuildRaid1(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, func(bs []*baremetal.BaremetalStorage, bdc *compute.BaremetalDiskConfig) error {
		return storcliBuildRaid(a.getAdaptorCmd, a.term, devs, conf, 1)
	})
}

func (a *PerccliAdaptor) BuildRaid5(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, func(bs []*baremetal.BaremetalStorage, bdc *compute.BaremetalDiskConfig) error {
		return storcliBuildRaid(a.getAdaptorCmd, a.term, devs, conf, 5)
	})
}

func (a *PerccliAdaptor) BuildRaid10(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, func(bs []*baremetal.BaremetalStorage, bdc *compute.BaremetalDiskConfig) error {
		return storcliBuildRaid(a.getAdaptorCmd, a.term, devs, conf, 10)
	})
}

func (a *PerccliAdaptor) BuildNoneRaid(devs []*baremetal.BaremetalStorage) error {
	return cliBuildRaid(devs, nil, func(bs []*baremetal.BaremetalStorage, bdc *compute.BaremetalDiskConfig) error {
		return storcliBuildNoRaid(a.getAdaptorCmd, a.term, devs)
	})
}

type PerccliPhyDev struct {
	*StorcliPhysicalDrive
}

func NewPerccliPhyDev(base *StorcliPhysicalDrive) *PerccliPhyDev {
	return &PerccliPhyDev{
		StorcliPhysicalDrive: base,
	}
}
