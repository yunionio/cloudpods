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

package sas2iru

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
)

type Mpt2SASRaidPhyDev struct {
	*raid.RaidBasePhyDev

	enclosure int
	slot      int
	sector    int
	block     int
}

func newMpt2SASRaidPhyDev(adapter int) *Mpt2SASRaidPhyDev {
	b := raid.NewRaidBasePhyDev(baremetal.DISK_DRIVER_MPT2SAS)
	b.Adapter = adapter
	return &Mpt2SASRaidPhyDev{
		RaidBasePhyDev: b,
		slot:           -1,
		enclosure:      -1,
		block:          -1,
		sector:         -1,
	}
}

func (dev *Mpt2SASRaidPhyDev) parseLine(line string) bool {
	key, val := stringutils.SplitKeyValue(line)
	if key == "" {
		return false
	}
	switch key {
	case "Drive Type":
		if strings.HasSuffix(val, "_HDD") {
			dev.Rotate = tristate.True
		} else {
			dev.Rotate = tristate.False
		}
	case "Enclosure #":
		dev.enclosure, _ = strconv.Atoi(val)
	case "Slot #":
		dev.slot, _ = strconv.Atoi(val)
	case "Size (in MB)/(in sectors)":
		dat := strings.Split(val, "/")
		sz, _ := strconv.Atoi(dat[0])
		dev.Size = int64(sz)
		dev.sector, _ = strconv.Atoi(dat[1])
		dev.block = int(dev.Size * 1024 * 1024 / 7814037167)
		if dev.block > 4000 {
			dev.block = 4096
		} else {
			dev.block = 512
		}
		dev.Size = int64(dev.block * dev.sector / 1024 / 1024)
	case "Manufacturer", "Model Number", "Firmware Revision", "Serial No":
		if dev.Model == "" {
			dev.Model = val
		} else {
			dev.Model = fmt.Sprintf("%s %s", dev.Model, val)
		}
	case "State":
		if strings.Contains(strings.ToLower(val), "ready") {
			dev.Status = "online"
		} else {
			dev.Status = strings.ToLower(val)
		}
	default:
		return false
	}
	return true
}

func (dev *Mpt2SASRaidPhyDev) isComplete() bool {
	if !dev.RaidBasePhyDev.IsComplete() {
		return false
	}
	if dev.Size < 0 {
		return false
	}
	if dev.slot < 0 {
		return false
	}
	if dev.sector < 0 {
		return false
	}
	if dev.block < 0 {
		return false
	}
	return true
}

func (dev *Mpt2SASRaidPhyDev) ToBaremetalStorage(idx int) *baremetal.BaremetalStorage {
	s := dev.RaidBasePhyDev.ToBaremetalStorage(idx)
	s.Index = int64(idx)
	s.Slot = dev.slot
	s.Enclosure = dev.enclosure
	s.Block = int64(dev.block)
	s.Sector = int64(dev.sector)
	return s
}

func GetSpecString(dev *baremetal.BaremetalStorage) string {
	if dev.Enclosure < 0 {
		return fmt.Sprintf(":%d", dev.Slot)
	}
	return fmt.Sprintf("%d:%d", dev.Enclosure, dev.Slot)
}

type Mpt2SASRaidAdaptor struct {
	index int
	raid  *Mpt2SASRaid
	devs  []*Mpt2SASRaidPhyDev
}

func newMpt2SASRaidAdaptor(index int, raid *Mpt2SASRaid) *Mpt2SASRaidAdaptor {
	return &Mpt2SASRaidAdaptor{
		index: index,
		raid:  raid,
		devs:  make([]*Mpt2SASRaidPhyDev, 0),
	}
}

func (adapter *Mpt2SASRaidAdaptor) GetIndex() int {
	return adapter.index
}

func (adapter *Mpt2SASRaidAdaptor) ParsePhyDevs() error {
	cmd := adapter.raid.GetCommand(fmt.Sprintf("%d", adapter.index), "DISPLAY")
	ret, err := adapter.raid.term.Run(cmd)
	if err != nil {
		return fmt.Errorf("get physical device: %v", err)
	}
	for _, l := range ret {
		if strings.Contains(l, "RAID Support") && strings.ToLower(strings.TrimSpace(l[strings.Index(l, ":")+1:])) == "no" {
			return fmt.Errorf("No raid support")
		}
	}
	return adapter.parsePhyDevs(ret)
}

func (adapter *Mpt2SASRaidAdaptor) parsePhyDevs(lines []string) error {
	dev := newMpt2SASRaidPhyDev(adapter.index)
	for _, l := range lines {
		if dev.parseLine(l) && dev.isComplete() {
			adapter.devs = append(adapter.devs, dev)
			dev = newMpt2SASRaidPhyDev(adapter.index)
		}
	}
	return nil
}

func (adapter *Mpt2SASRaidAdaptor) GetDevices() []*baremetal.BaremetalStorage {
	ret := []*baremetal.BaremetalStorage{}
	for idx, dev := range adapter.devs {
		ret = append(ret, dev.ToBaremetalStorage(idx))
	}
	return ret
}

func (adapter *Mpt2SASRaidAdaptor) GetLogicVolumes() ([]*raid.RaidLogicalVolume, error) {
	cmd := adapter.raid.GetCommand(fmt.Sprintf("%d", adapter.index), "DISPLAY")
	ret, err := adapter.raid.term.Run(cmd)
	if err != nil {
		return nil, fmt.Errorf("GetLogicVolumes error: %v", err)
	}
	return adapter.parseLogicVolumes(ret)
}

func (adapter *Mpt2SASRaidAdaptor) parseLogicVolumes(lines []string) ([]*raid.RaidLogicalVolume, error) {
	lvIdx := []*raid.RaidLogicalVolume{}
	usedDevs := []*raid.RaidLogicalVolume{}
	for _, line := range lines {
		key, val := stringutils.SplitKeyValue(line)
		if key != "" && key == "Volume ID" {
			idx, _ := strconv.Atoi(val)
			lvIdx = append(lvIdx, &raid.RaidLogicalVolume{
				Index:   idx,
				Adapter: adapter.index,
			})
		} else if regexp.MustCompile(`PHY\[\d+\] Enclosure#/Slot#`).MatchString(key) {
			idx, _ := strconv.Atoi(val)
			usedDevs = append(usedDevs, &raid.RaidLogicalVolume{
				Index:   idx,
				Adapter: adapter.index,
			})
		}
	}
	if len(adapter.devs) < len(usedDevs) {
		return nil, fmt.Errorf("adapter current dev %d < usedDevs %d", len(adapter.devs), len(usedDevs))
	}
	for i := 0; i < len(adapter.devs)-len(usedDevs); i++ {
		lvIdx = append(lvIdx, &raid.RaidLogicalVolume{
			Index:   -1,
			Adapter: adapter.index,
		})
	}
	return lvIdx, nil
}

func (adapter *Mpt2SASRaidAdaptor) rescanLV() error {
	var cmd string
	if adapter.raid.utility == "/opt/lsi/sas2ircu" {
		cmd = "/opt/lsi/rescan.sh mpt2sas"
	} else if adapter.raid.utility == "/opt/lsi/sas3ircu" {
		cmd = "/opt/lsi/rescan.sh mpt3sas"
	} else {
		return fmt.Errorf("Unsupport raid utility: %v", adapter.raid.utility)
	}
	_, err := adapter.raid.term.Run(cmd)
	return err
}

func (adapter *Mpt2SASRaidAdaptor) setBootIR() error {
	lvs, err := adapter.GetLogicVolumes()
	if err != nil {
		return err
	}
	if len(lvs) > 0 && lvs[0].Index > 0 {
		args := []string{fmt.Sprintf("%d", adapter.index), "BOOTIR", fmt.Sprintf("%d", lvs[0].Index)}
		cmd := adapter.raid.GetCommand(args...)
		_, err := adapter.raid.term.Run(cmd)
		return err
	}
	return nil
}

func (adapter *Mpt2SASRaidAdaptor) PreBuildRaid(confs []*api.BaremetalDiskConfig) error {
	return nil
}

func (adapter *Mpt2SASRaidAdaptor) PostBuildRaid() error {
	return nil
}

func (adapter *Mpt2SASRaidAdaptor) buildRaid(level string, devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	if len(conf.Size) > 1 {
		return fmt.Errorf("Subdivide sub-size not supported")
	}
	args := []string{fmt.Sprintf("%d", adapter.index), "CREATE", level, "MAX"}
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, GetSpecString(dev))
	}
	args = append(args, labels...)
	args = append(args, "noprompt")
	_, err := adapter.raid.term.Run(adapter.raid.GetCommand(args...))
	if err != nil {
		return fmt.Errorf("Build raid error: %v", err)
	}
	if err := adapter.setBootIR(); err != nil {
		return fmt.Errorf("setBootIR: %v", err)
	}
	if err := adapter.rescanLV(); err != nil {
		return fmt.Errorf("rescanLV: %v", err)
	}
	return nil
}

func (adapter *Mpt2SASRaidAdaptor) BuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.buildRaid("RAID0", devs, conf)
}

func (adapter *Mpt2SASRaidAdaptor) BuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.buildRaid("RAID1", devs, conf)
}

func (adapter *Mpt2SASRaidAdaptor) BuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return fmt.Errorf("Not impl")
}

func (adapter *Mpt2SASRaidAdaptor) BuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	if len(devs) > 10 {
		return fmt.Errorf("RAID10 supports no more than 10 disks")
	}
	return adapter.buildRaid("RAID10", devs, conf)
}

func (adapter *Mpt2SASRaidAdaptor) BuildNoneRaid(devs []*baremetal.BaremetalStorage) error {
	// TODO: not impl
	// return fmt.Errorf("Not impl")
	return nil
}

func (adapter *Mpt2SASRaidAdaptor) RemoveLogicVolumes() error {
	cmd := adapter.raid.GetCommand(fmt.Sprintf("%d", adapter.index), "DELETE", "noprompt")
	_, err := adapter.raid.term.Run(cmd)
	return err
}

type Mpt2SASRaid struct {
	term     raid.IExecTerm
	utility  string
	adapters []*Mpt2SASRaidAdaptor
}

func NewMpt2SASRaid(term raid.IExecTerm) raid.IRaidDriver {
	return &Mpt2SASRaid{
		term:     term,
		adapters: make([]*Mpt2SASRaidAdaptor, 0),
	}
}

func (r *Mpt2SASRaid) GetName() string {
	return baremetal.DISK_DRIVER_MPT2SAS
}

func (r *Mpt2SASRaid) ParsePhyDevs() error {
	if r.modulePCIProbed(raid.MODULE_MPT2SAS) {
		r.utility = "/opt/lsi/sas2ircu"
	} else if r.modulePCIProbed(raid.MODULE_MPT3SAS) {
		r.utility = "/opt/lsi/sas3ircu"
	} else {
		return fmt.Errorf("Not probe mpt2sas or mpt3sas kernel module")
	}
	cmd := r.GetCommand("LIST")
	ret, err := r.term.Run(cmd)
	if err != nil {
		return err
	}
	return r.parseAdapters(ret)
}

func getLineAdapterIndex(line string) int {
	dat := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)
	if len(dat) == 0 {
		return -1
	}
	if regexp.MustCompile(`\d+`).MatchString(dat[0]) {
		if !regexp.MustCompile(`^\d+`).MatchString(dat[0]) {
			return -1
		}
		idx, _ := strconv.Atoi(dat[0])
		return idx
	}
	return -1
}

func (r *Mpt2SASRaid) parseAdapters(lines []string) error {
	for _, line := range lines {
		idx := getLineAdapterIndex(line)
		if idx >= 0 {
			adapter := newMpt2SASRaidAdaptor(idx, r)
			r.adapters = append(r.adapters, adapter)
		}
	}
	for _, adapter := range r.adapters {
		if err := adapter.ParsePhyDevs(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Mpt2SASRaid) modulePCIProbed(mod string) bool {
	cmd := fmt.Sprintf("lspci -k | grep %s", mod)
	_, err := r.term.Run(cmd)
	return err == nil
}

func (r *Mpt2SASRaid) GetCommand(args ...string) string {
	return raid.GetCommand(r.utility, args...)
}

func (r *Mpt2SASRaid) PreBuildRaid(_ []*api.BaremetalDiskConfig, _ int) error {
	return nil
}

func (r *Mpt2SASRaid) GetAdapters() []raid.IRaidAdapter {
	ret := make([]raid.IRaidAdapter, 0)
	for _, a := range r.adapters {
		ret = append(ret, a)
	}
	return ret
}

func (r *Mpt2SASRaid) CleanRaid() error {
	return nil
}

func init() {
	raid.RegisterDriver(baremetal.DISK_DRIVER_MPT2SAS, NewMpt2SASRaid)
}
