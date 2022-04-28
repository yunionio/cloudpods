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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/options"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
)

func init() {
	raid.RegisterDriver(baremetal.DISK_DRIVER_MEGARAID, newRaid)
}

type iDriver interface {
	GetName() string
	GetAdapterConstructCmd() string
	NewAdaptor(term raid.IExecTerm) iAdaptor
	ClearForeignState(term raid.IExecTerm) error
}

type iAdaptor interface {
	ParseLine(line string)
	IsComplete() bool
	Key() string
	GetPhyDevs() ([]*MegaRaidPhyDev, error)
	ClearJBODDisks(devs []*MegaRaidPhyDev)

	GetIndex() int
	GetLogicVolumes() ([]*raid.RaidLogicalVolume, error)
	RemoveLogicVolumes() error
	BuildRaid0(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error
	BuildRaid1(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error
	BuildRaid5(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error
	BuildRaid10(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error
	BuildNoneRaid(devs []*baremetal.BaremetalStorage) error
}

type sRaid struct {
	driver     iDriver
	term       raid.IExecTerm
	adaptors   []*sRaidAdaptor
	phyDevsCnt int
}

func newRaid(term raid.IExecTerm) raid.IRaidDriver {
	if options.Options.UseMegaRaidPerccli {
		perccliDrv := &sRaid{
			driver:   newPerccliDriver(),
			term:     term,
			adaptors: make([]*sRaidAdaptor, 0),
		}
		if err := perccliDrv.ParsePhyDevs(); err == nil && perccliDrv.phyDevsCnt > 0 {
			log.Infof("Use perccli driver, found %d pds", perccliDrv.phyDevsCnt)
			return perccliDrv
		} else {
			log.Warningf("perccli driver parse physical devices error: %v, %d pds, fallback to old megactl and storcli driver", err, perccliDrv.phyDevsCnt)
			return NewMegaRaid(term)
		}
	}
	log.Infof("Not use perccli, use legacy megaraid driver")
	return NewMegaRaid(term)
}

func (r *sRaid) GetName() string {
	return baremetal.DISK_DRIVER_MEGARAID
}

type sRaidAdaptor struct {
	iAdaptor

	raid         *sRaid
	devs         []*MegaRaidPhyDev
	sn           string
	name         string
	busNumber    string
	deviceNumber string
	funcNumber   string
	// used by sg_map
	hostNum int
	//channelNum int
}

func newRaidAdaptor(baseAda iAdaptor, raid *sRaid) *sRaidAdaptor {
	return &sRaidAdaptor{
		iAdaptor: baseAda,
		raid:     raid,
		devs:     make([]*MegaRaidPhyDev, 0),
	}
}

func (a *sRaidAdaptor) AddPhyDev(dev *MegaRaidPhyDev) {
	a.devs = append(a.devs, dev)
}

func (r *sRaid) ParsePhyDevs() error {
	adaptors, _, err := r.getAdaptor()
	r.phyDevsCnt = 0
	if err != nil {
		return errors.Wrapf(err, "Get %q adaptor", r.driver.GetName())
	}
	r.adaptors = make([]*sRaidAdaptor, 0)
	for _, ada := range adaptors {
		devs, err := ada.GetPhyDevs()
		if err != nil {
			return errors.Wrap(err, "Get %q adaptor physical devices")
		}
		nAda := newRaidAdaptor(ada, r)
		for i := range devs {
			nAda.AddPhyDev(devs[i])
			r.phyDevsCnt++
		}
		r.adaptors = append(r.adaptors, nAda)
	}
	return nil
}

func (r *sRaid) getAdaptor() ([]iAdaptor, map[string]iAdaptor, error) {
	ret := make(map[string]iAdaptor)
	cmd := r.driver.GetAdapterConstructCmd()
	lines, err := r.term.Run(cmd)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Get storcli adapter")
	}
	adaptor := r.driver.NewAdaptor(r.term)
	list := make([]iAdaptor, 0)
	for _, l := range lines {
		adaptor.ParseLine(l)
		if adaptor.IsComplete() {
			ret[adaptor.Key()] = adaptor
			list = append(list, adaptor)
			adaptor = r.driver.NewAdaptor(r.term)
		}
	}
	return list, ret, nil
}

func (r *sRaid) PreBuildRaid(_ []*compute.BaremetalDiskConfig, _ int) error {
	return r.driver.ClearForeignState(r.term)
}

func (r *sRaid) GetAdapters() []raid.IRaidAdapter {
	ret := make([]raid.IRaidAdapter, len(r.adaptors))
	for i := range r.adaptors {
		ret[i] = r.adaptors[i]
	}
	return ret
}

func (r *sRaidAdaptor) PreBuildRaid(_ []*compute.BaremetalDiskConfig) error {
	r.ClearJBODDisks()
	return nil
}

func (r *sRaidAdaptor) ClearJBODDisks() {
	r.iAdaptor.ClearJBODDisks(r.devs)
}

func (r *sRaidAdaptor) GetDevices() []*baremetal.BaremetalStorage {
	ret := []*baremetal.BaremetalStorage{}
	for idx, dev := range r.devs {
		ret = append(ret, dev.ToBaremetalStorage(idx))
	}
	return ret
}

func (r *sRaid) CleanRaid() error {
	for _, ada := range r.adaptors {
		ada.ClearJBODDisks()
		ada.RemoveLogicVolumes()
	}
	return nil
}
