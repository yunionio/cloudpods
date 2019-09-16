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

package hpssactl

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/regutils2"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type HPSARaidPhyDev struct {
	*raid.RaidBasePhyDev
	addr string
}

func newHPSARaidPhyDev(addr string, adapter int, rotate bool) *HPSARaidPhyDev {
	b := raid.NewRaidBasePhyDev(baremetal.DISK_DRIVER_HPSARAID)
	b.Adapter = adapter
	if rotate {
		b.Rotate = tristate.True
	} else {
		b.Rotate = tristate.False
	}
	return &HPSARaidPhyDev{
		RaidBasePhyDev: b,
		addr:           addr,
	}
}

func (dev *HPSARaidPhyDev) ToBaremetalStorage(index int) *baremetal.BaremetalStorage {
	s := dev.RaidBasePhyDev.ToBaremetalStorage(index)
	s.Addr = dev.addr

	return s
}

func (dev *HPSARaidPhyDev) parseLine(line string) bool {
	key, val := stringutils.SplitKeyValue(line)
	if key == "" {
		return false
	}
	switch key {
	case "Size":
		dat := strings.Split(val, " ")
		szStr, unitStr := dat[0], dat[1]
		var sz int64
		szF, err := strconv.ParseFloat(szStr, 64)
		if err != nil {
			log.Errorf("Parse size string %s: %v", szStr, err)
			return false
		}
		switch unitStr {
		case "GB":
			sz = int64(szF * 1000 * 1000 * 1000)
		case "TB":
			sz = int64(szF * 1000 * 1000 * 1000 * 1000)
		case "MB":
			sz = int64(szF * 1000 * 1000)
		default:
			log.Errorf("Unsupported unit: %s", unitStr)
			return false
		}
		dev.Size = sz / 1024 / 1024
	case "Model":
		dev.Model = strings.Join(regexp.MustCompile(`\s+`).Split(val, -1), " ")
	case "Status":
		dev.Status = val
	default:
		return false
	}
	return true
}

func (dev *HPSARaidPhyDev) isComplete() bool {
	if !dev.RaidBasePhyDev.IsComplete() {
		return false
	}
	if dev.Size < 0 {
		return false
	}
	return true
}

func GetSpecString(dev *baremetal.BaremetalStorage) string {
	return dev.Addr
}

type HPSARaidAdaptor struct {
	index int
	raid  *HPSARaid
	devs  []*HPSARaidPhyDev
}

func newHPSARaidAdaptor(index int, raid *HPSARaid) *HPSARaidAdaptor {
	return &HPSARaidAdaptor{
		index: index,
		raid:  raid,
	}
}

func (adapter *HPSARaidAdaptor) GetIndex() int {
	return adapter.index
}

func (adapter *HPSARaidAdaptor) PreBuildRaid(confs []*api.BaremetalDiskConfig) error {
	return nil
}

func (adapter *HPSARaidAdaptor) ParsePhyDevs() error {
	parseByCmd := func(cmd string, isRotate bool) error {
		ret, err := adapter.raid.term.Run(cmd)
		if err != nil {
			return err
		}
		adapter.parsePhyDevs(ret, isRotate)
		return nil
	}
	cmd1 := GetCommand("controller", fmt.Sprintf("slot=%d", adapter.index), "ssdphysicaldrive", "all", "show", "detail")
	cmd2 := GetCommand("controller", fmt.Sprintf("slot=%d", adapter.index), "physicaldrive", "all", "show", "detail")
	var err1 error
	var err2 error
	if err1 = parseByCmd(cmd1, false); err1 != nil {
		err1 = errors.Errorf("parsePhyDevs by cmd %q: %v", cmd1, err1)
	}
	if err2 = parseByCmd(cmd2, true); err2 != nil {
		err2 = errors.Errorf("parsePhyDevs by cmd %q: %v", cmd1, err2)
	}
	if err1 != nil && err2 != nil {
		return errors.Errorf("ssd: %v, hdd: %v", err1, err2)
	}
	return nil
}

func (adapter *HPSARaidAdaptor) parsePhyDevs(lines []string, isRotate bool) {
	var phydev *HPSARaidPhyDev
	for _, line := range lines {
		m := regutils2.SubGroupMatch(`physicaldrive\s+(?P<addr>\w+:\w+:\w)`, line)
		if len(m) != 0 {
			phydev = newHPSARaidPhyDev(m["addr"], adapter.index, isRotate)
		} else if phydev != nil && phydev.parseLine(line) && phydev.isComplete() {
			oldDev := adapter.getPhyDevByAddr(phydev.addr)
			if oldDev == nil {
				adapter.devs = append(adapter.devs, phydev)
			}
			phydev = nil
		}
	}
}

func (adapter *HPSARaidAdaptor) getPhyDevByAddr(addr string) *HPSARaidPhyDev {
	for _, dev := range adapter.devs {
		if addr == dev.addr {
			return dev
		}
	}
	return nil
}

func (adapter *HPSARaidAdaptor) GetDevices() []*baremetal.BaremetalStorage {
	ret := []*baremetal.BaremetalStorage{}
	for idx, dev := range adapter.devs {
		ret = append(ret, dev.ToBaremetalStorage(idx))
	}
	return ret
}

func (adapter *HPSARaidAdaptor) conf2Params(conf *api.BaremetalDiskConfig) []string {
	params := []string{}
	if conf.Direct != nil {
		if *(conf.Direct) {
			params = append(params, "caching=disable")
		} else {
			params = append(params, "caching=enable")
		}
	}
	if conf.Strip != nil {
		params = append(params, fmt.Sprintf("stripsize=%d", *(conf.Strip)))
	}
	return params
}

func (adapter *HPSARaidAdaptor) getLastArray() (string, error) {
	cmd := GetCommand("controller", fmt.Sprintf("slot=%d", adapter.index), "logicaldrive", "all", "show")
	ret, _ := adapter.raid.term.Run(cmd)
	// ignore errors
	// if err != nil {
	// 	return "", err
	// }
	var lastArray string
	for _, line := range ret {
		m := regutils2.SubGroupMatch(`array\s+(?P<idx>\w+)`, line)
		if len(m) > 0 {
			lastArray = m["idx"]
			return lastArray, nil
		}
	}
	return "", nil
}

func (adapter *HPSARaidAdaptor) buildRaid(level string, devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, fmt.Sprintf("%s", GetSpecString(dev)))
	}
	args := []string{
		"controller", fmt.Sprintf("slot=%d", adapter.GetIndex()),
		"create", "type=ld", fmt.Sprintf("drives=%s", strings.Join(labels, ",")),
		fmt.Sprintf("raid=%s", level),
	}
	if len(conf.Size) > 0 {
		args = append(args, fmt.Sprintf("size=%d", conf.Size[0]))
	}
	params := adapter.conf2Params(conf)
	args = append(args, params...)
	cmd := GetCommand(args...)
	_, err := adapter.raid.term.RunWithInput(strings.NewReader("y\n"), cmd)
	if err != nil {
		return err
	}
	if len(conf.Size) > 0 {
		array, err := adapter.getLastArray()
		if err != nil {
			return fmt.Errorf("getLastArray: %v", err)
		}
		cmds := []string{}
		for _, sz := range conf.Size[1:] {
			args = []string{"controller", fmt.Sprintf("slot=%d", adapter.index),
				"array", array, "create", "type=ld",
				fmt.Sprintf("raid=%s", level),
				fmt.Sprintf("size=%d", sz),
			}
			args = append(args, params...)
			cmds = append(cmds, GetCommand(args...))
		}
		_, err = adapter.raid.term.RunWithInput(strings.NewReader("y\n"), cmds...)
	}
	return err
}

func (adapter *HPSARaidAdaptor) BuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.buildRaid("0", devs, conf)
}

func (adapter *HPSARaidAdaptor) BuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.buildRaid("1", devs, conf)
}

func (adapter *HPSARaidAdaptor) BuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.buildRaid("5", devs, conf)
}

func (adapter *HPSARaidAdaptor) BuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.buildRaid("1+0", devs, conf)
}

func (adapter *HPSARaidAdaptor) BuildNoneRaid(devs []*baremetal.BaremetalStorage) error {
	for _, d := range devs {
		// WT|WB] [NORA|RA] [Direct|Cached] [CachedBadBBU|NoCachedBadBBU]
		useWT := true
		useDirect := true
		if err := adapter.buildRaid("0", []*baremetal.BaremetalStorage{d}, &api.BaremetalDiskConfig{WT: &useWT, Direct: &useDirect}); err != nil {
			return err
		}
	}
	return nil
}

func (adapter *HPSARaidAdaptor) removeLogicVolume(idx int) error {
	cmd := GetCommand("controller", fmt.Sprintf("slot=%d", adapter.index), "logicaldrive",
		fmt.Sprintf("%d", idx), "delete", "forced")
	_, err := adapter.raid.term.Run(cmd)
	return err
}

func (adapter *HPSARaidAdaptor) GetLogicVolumes() ([]*raid.RaidLogicalVolume, error) {
	cmd := GetCommand("controller", fmt.Sprintf("slot=%d", adapter.index), "logicaldrive", "all", "show")
	ret, _ := adapter.raid.term.Run(cmd)
	// ignore error
	// if err != nil {
	// 	return nil, err
	// }
	return adapter.parseLogicalVolumes(ret)
}

func (adapter *HPSARaidAdaptor) parseLogicalVolumes(lines []string) ([]*raid.RaidLogicalVolume, error) {
	lvs := []*raid.RaidLogicalVolume{}
	for _, line := range lines {
		m := regutils2.SubGroupMatch(`logicaldrive\s+(?P<addr>\w+)`, line)
		if len(m) > 0 {
			idxStr := m["addr"]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return nil, fmt.Errorf("%s not int: %v", idxStr, err)
			}
			lvs = append(lvs, &raid.RaidLogicalVolume{
				Index:   idx,
				Adapter: adapter.index,
			})
		}
	}
	return lvs, nil
}

func (adapter *HPSARaidAdaptor) RemoveLogicVolumes() error {
	lvs, err := adapter.GetLogicVolumes()
	if err != nil {
		return fmt.Errorf("Failed to get logic volumes: %v", err)
	}
	for _, i := range raid.ReverseLogicalArray(lvs) {
		if err := adapter.removeLogicVolume(i.Index); err != nil {
			return fmt.Errorf("Remove %#v logical volume: %v", i, err)
		}
	}
	return nil
}

type HPSARaid struct {
	term     *ssh.Client
	adapters []*HPSARaidAdaptor
}

func NewHPSARaid(term *ssh.Client) raid.IRaidDriver {
	return &HPSARaid{
		term:     term,
		adapters: make([]*HPSARaidAdaptor, 0),
	}
}

func (r *HPSARaid) ParsePhyDevs() error {
	if !utils.IsInStringArray(raid.MODULE_HPSA, raid.GetModules(r.term)) {
		return fmt.Errorf("Not found hpsa module")
	}
	cmd := GetCommand("controller", "all", "show")
	ret, err := r.term.Run(cmd)
	if err != nil {
		return err
	}
	return r.parsePhyDevs(ret)
}

func (r *HPSARaid) parsePhyDevs(lines []string) error {
	for _, line := range lines {
		m := regutils2.SubGroupMatch(`\s+Slot\s+(?P<idx>[0-9]+)\s+`, line)
		if len(m) > 0 {
			idxStr := m["idx"]
			idx, err := strconv.Atoi(idxStr)
			if err != nil {
				return err
			}
			adapter := newHPSARaidAdaptor(idx, r)
			r.adapters = append(r.adapters, adapter)
		}
	}
	for _, a := range r.adapters {
		if err := a.ParsePhyDevs(); err != nil {
			return err
		}
	}
	return nil
}

func (r *HPSARaid) PreBuildRaid(_ []*api.BaremetalDiskConfig, _ int) error {
	return nil
}

func (r *HPSARaid) GetAdapters() []raid.IRaidAdapter {
	ret := make([]raid.IRaidAdapter, 0)
	for _, a := range r.adapters {
		ret = append(ret, a)
	}
	return ret
}

func (r *HPSARaid) GetName() string {
	return baremetal.DISK_DRIVER_HPSARAID
}

func (r *HPSARaid) CleanRaid() error {
	// pass
	return nil
}

func GetCommand(args ...string) string {
	bin := "/opt/hp/hpssacli/bld/hpssacli"
	return raid.GetCommand(bin, args...)
}

func init() {
	raid.RegisterDriver(baremetal.DISK_DRIVER_HPSARAID, NewHPSARaid)
}
