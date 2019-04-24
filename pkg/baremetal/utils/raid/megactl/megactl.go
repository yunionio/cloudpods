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
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	raiddrivers "yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/util/regutils2"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

var (
	sizePattern   = regexp.MustCompile(`(?P<sector>0x[0-9a-fA-F]+)`)
	adapterPatter = regexp.MustCompile(`^Adapter #(?P<idx>[0-9]+)`)
)

type MegaRaidPhyDev struct {
	*raiddrivers.RaidBasePhyDev

	enclosure    int
	slot         int
	minStripSize int
	maxStripSize int
	sector       int64
	block        int64
}

func NewMegaRaidPhyDev() *MegaRaidPhyDev {
	return &MegaRaidPhyDev{
		RaidBasePhyDev: raiddrivers.NewRaidBasePhyDev(baremetal.DISK_DRIVER_MEGARAID),
		enclosure:      -1,
		slot:           -1,
		minStripSize:   -1,
		maxStripSize:   -1,
		sector:         -1,
		block:          512,
	}
}

func (dev *MegaRaidPhyDev) ToBaremetalStorage() *baremetal.BaremetalStorage {
	s := dev.RaidBasePhyDev.ToBaremetalStorage()
	s.Enclosure = dev.enclosure
	s.Slot = dev.slot
	s.Size = dev.GetSize()
	s.MinStripSize = int64(dev.minStripSize)
	s.MaxStripSize = int64(dev.maxStripSize)
	s.Block = dev.block
	s.Sector = dev.sector
	return s
}

func (dev *MegaRaidPhyDev) GetSize() int64 {
	return dev.sector * int64(dev.block) / 1024 / 1024 // MB
}

func (dev *MegaRaidPhyDev) parseLine(line string) bool {
	key, val := stringutils.SplitKeyValue(line)
	if key == "" {
		return false
	}
	switch key {
	case "Media Type":
		if val == "Hard Disk Device" {
			dev.Rotate = tristate.True
		} else {
			dev.Rotate = tristate.False
		}
	case "Enclosure Device ID":
		enclosure, err := strconv.Atoi(val)
		if err == nil {
			dev.enclosure = enclosure
		}
	case "Slot Number":
		dev.slot, _ = strconv.Atoi(val)
	case "Coerced Size":
		sizeStr := regutils2.GetParams(sizePattern, val)["sector"]
		if len(sizeStr) != 0 {
			sizeStr = strings.Replace(sizeStr, "0x", "", -1)
			sector, err := strconv.ParseInt(sizeStr, 16, 64)
			if err != nil {
				log.Errorf("Parse sector %q to Int error: %v", sizeStr, err)
			}
			dev.sector = sector
		} else {
			dev.sector = 0
		}
	case "Inquiry Data":
		dev.Model = strings.Join(regexp.MustCompile(`\s+`).Split(val, -1), " ")
	case "Firmware state":
		if val == "JBOD" {
			dev.Status = "jbod"
		} else if strings.Contains(strings.ToLower(val), "online") {
			dev.Status = "online"
		} else {
			dev.Status = "offline"
		}
	case "Logical Sector Size":
		block, err := strconv.Atoi(val)
		if err != nil {
			log.Errorf("parse logical sector size error: %v", err)
			dev.block = 512
		} else {
			dev.block = int64(block)
		}
	default:
		return false
	}
	return true
}

func (dev *MegaRaidPhyDev) parseStripSize(lines []string) error {
	size2Int := func(sizeStr string) int {
		sz, _ := strconv.ParseFloat(strings.Fields(sizeStr)[0], 32)
		szInt := int(sz)
		if strings.Contains(sizeStr, "KB") {
			return szInt
		}
		if strings.Contains(sizeStr, "MB") {
			return szInt * 1024
		}
		return -1
	}
	for _, line := range lines {
		if strings.Contains(line, "Min") {
			dev.minStripSize = size2Int(strings.Split(line, ": ")[1])
		}
		if strings.Contains(line, "Max") {
			dev.maxStripSize = size2Int(strings.Split(line, ": ")[1])
		}
	}
	return nil
}

func (dev *MegaRaidPhyDev) isComplete() bool {
	if !dev.RaidBasePhyDev.IsComplete() {
		return false
	}
	if dev.sector < 0 {
		return false
	}
	if dev.block < 0 {
		return false
	}
	if dev.slot < 0 {
		return false
	}
	return true
}

func (dev *MegaRaidPhyDev) isJBOD() bool {
	return dev.Status == "jbod"
}

func GetSpecString(dev *baremetal.BaremetalStorage) string {
	if dev.Enclosure < 0 {
		return fmt.Sprintf(":%d", dev.Slot)
	}
	return fmt.Sprintf("%d:%d", dev.Enclosure, dev.Slot)
}

type MegaRaidAdaptor struct {
	index int
	raid  *MegaRaid
	devs  []*MegaRaidPhyDev
}

func NewMegaRaidAdaptor(index int, raid *MegaRaid) *MegaRaidAdaptor {
	return &MegaRaidAdaptor{
		index: index,
		raid:  raid,
	}
}

func (adapter *MegaRaidAdaptor) GetIndex() int {
	return adapter.index
}

func (adapter *MegaRaidAdaptor) getTerm() *ssh.Client {
	return adapter.raid.term
}

func (adapter *MegaRaidAdaptor) remoteRun(cmds ...string) ([]string, error) {
	return adapter.getTerm().Run(cmds...)
}

func (adapter *MegaRaidAdaptor) AddPhyDev(dev *MegaRaidPhyDev) {
	dev.Adapter = adapter.index
	adapter.devs = append(adapter.devs, dev)
}

func (adapter *MegaRaidAdaptor) GetDevices() []*baremetal.BaremetalStorage {
	ret := []*baremetal.BaremetalStorage{}
	for _, dev := range adapter.devs {
		ret = append(ret, dev.ToBaremetalStorage())
	}
	return ret
}

func (adapter *MegaRaidAdaptor) GetLogicVolumes() ([]int, error) {
	cmd := GetCommand("-LDInfo", "-Lall", fmt.Sprintf("-a%d", adapter.index))
	ret, err := adapter.remoteRun(cmd)
	if err != nil {
		return nil, fmt.Errorf("GetLogicVolumes error: %v", err)
	}
	return adapter.parseLogicVolumes(ret), nil
}

func (adapter *MegaRaidAdaptor) parseLogicVolumes(lines []string) []int {
	lvIdx := []int{}
	for _, line := range lines {
		key, val := stringutils.SplitKeyValue(line)
		if key != "" && key == "Virtual Drive" {
			idx, _ := strconv.Atoi(strings.Split(val, " ")[0])
			lvIdx = append(lvIdx, idx)
		}
	}
	return lvIdx
}

func (adapter *MegaRaidAdaptor) PreBuildRaid(confs []*api.BaremetalDiskConfig) error {
	adapter.clearJBODDisks()
	return nil
}

func (adapter *MegaRaidAdaptor) conf2ParamsStorcliSize(conf *api.BaremetalDiskConfig) []string {
	params := []string{}
	szStr := []string{}
	if len(conf.Size) > 0 {
		for _, sz := range conf.Size {
			szStr = append(szStr, fmt.Sprintf("%dMB", sz))
		}
		params = append(params, fmt.Sprintf("Size=%s", strings.Join(szStr, ",")))
	}
	return params
}

func (adapter *MegaRaidAdaptor) conf2ParamsStorcli(conf *api.BaremetalDiskConfig) []string {
	params := []string{}
	if conf.WT != nil {
		if *conf.WT {
			params = append(params, "wt")
		} else {
			params = append(params, "wb")
		}
	}
	if conf.RA != nil {
		if *conf.RA {
			params = append(params, "ra")
		} else {
			params = append(params, "nora")
		}
	}
	if conf.Direct != nil {
		if *conf.Direct {
			params = append(params, "direct")
		} else {
			params = append(params, "cached")
		}
	}
	if conf.Cachedbadbbu != nil {
		if *conf.Cachedbadbbu {
			params = append(params, "CachedBadBBU")
		} else {
			params = append(params, "NoCachedBadBBU")
		}
	}
	if conf.Strip != nil {
		params = append(params, fmt.Sprintf("Strip=%d", *conf.Strip))
	}
	return params
}

func conf2Params(conf *api.BaremetalDiskConfig) []string {
	params := []string{}
	if conf.WT != nil {
		if *conf.WT {
			params = append(params, "WT")
		} else {
			params = append(params, "WB")
		}
	}
	if conf.RA != nil {
		if *conf.RA {
			params = append(params, "RA")
		} else {
			params = append(params, "NORA")
		}
	}
	if conf.Direct != nil {
		if *conf.Direct {
			params = append(params, "Direct")
		} else {
			params = append(params, "Cached")
		}
	}
	if conf.Cachedbadbbu != nil {
		if *conf.Cachedbadbbu {
			params = append(params, "CachedBadBBU")
		} else {
			params = append(params, "NoCachedBadBBU")
		}
	}
	if conf.Strip != nil {
		params = append(params, fmt.Sprintf("-strpsz%d", *conf.Strip))
	}
	if len(conf.Size) > 0 {
		for _, sz := range conf.Size {
			params = append(params, fmt.Sprintf("-sz%d", sz))
		}
	}
	return params
}

func (adapter *MegaRaidAdaptor) storcliBuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.storcliBuildRaid(devs, conf, 0)
}

func (adapter *MegaRaidAdaptor) megacliBuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.megacliBuildRaid(devs, conf, 0)
}

func (adapter *MegaRaidAdaptor) storcliBuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.storcliBuildRaid(devs, conf, 1)
}

func (adapter *MegaRaidAdaptor) megacliBuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.megacliBuildRaid(devs, conf, 1)
}

func (adapter *MegaRaidAdaptor) storcliBuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.storcliBuildRaid(devs, conf, 5)
}

func (adapter *MegaRaidAdaptor) megacliBuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.megacliBuildRaid(devs, conf, 5)
}

func (adapter *MegaRaidAdaptor) storcliBuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return adapter.storcliBuildRaid(devs, conf, 10)
}

func (adapter *MegaRaidAdaptor) megacliBuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	if len(devs)%2 != 0 {
		return fmt.Errorf("Odd number of %d devs", len(devs))
	}
	devCnt := len(devs) / 2
	params := []string{}
	for i := 0; i < devCnt; i++ {
		d1 := devs[i]
		d2 := devs[i+devCnt]
		params = append(params, fmt.Sprintf("-Array%d[%s,%s]", i, GetSpecString(d1), GetSpecString(d2)))
	}
	args := []string{"-CfgSpanAdd", "-r10"}
	args = append(args, params...)
	args = append(args, conf2Params(conf)...)
	args = append(args, fmt.Sprintf("-a%d", adapter.index))
	cmd := GetCommand(args...)
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) storcliBuildRaid(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig, level uint) error {
	args := []string{}
	args = append(args, fmt.Sprintf("/c%d", adapter.index))
	args = append(args, "add", "vd", fmt.Sprintf("type=r%d", level))
	args = append(args, adapter.conf2ParamsStorcliSize(conf)...)
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, GetSpecString(dev))
	}
	args = append(args, fmt.Sprintf("drives=%s", strings.Join(labels, ",")))
	if level == 10 {
		args = append(args, "PDperArray=2")
	}
	args = append(args, adapter.conf2ParamsStorcli(conf)...)
	cmd := GetCommand2(args...)
	log.Infof("_storcliBuildRaid command: %s", cmd)
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) megacliBuildRaid(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig, level uint) error {
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, GetSpecString(dev))
	}
	args := []string{"-CfgLdAdd", fmt.Sprintf("-r%d", level), fmt.Sprintf("[%s]", strings.Join(labels, ","))}
	args = append(args, conf2Params(conf)...)
	args = append(args, fmt.Sprintf("-a%d", adapter.index))
	cmd := GetCommand(args...)
	log.Infof("_megacliBuildRaid command: %s", cmd)
	_, err := adapter.remoteRun(cmd)
	return err
}

func cliBuildRaid(
	devs []*baremetal.BaremetalStorage,
	conf *api.BaremetalDiskConfig,
	funcs ...func([]*baremetal.BaremetalStorage, *api.BaremetalDiskConfig) error,
) error {
	var err error
	for _, f := range funcs {
		err = f(devs, conf)
		if err == nil {
			return nil
		}
	}
	return err
}

func (adapter *MegaRaidAdaptor) BuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, adapter.megacliBuildRaid0, adapter.storcliBuildRaid0)
}

func (adapter *MegaRaidAdaptor) BuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, adapter.megacliBuildRaid1, adapter.storcliBuildRaid1)
}

func (adapter *MegaRaidAdaptor) BuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, adapter.megacliBuildRaid5, adapter.storcliBuildRaid5)
}

func (adapter *MegaRaidAdaptor) BuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return cliBuildRaid(devs, conf, adapter.megacliBuildRaid10, adapter.storcliBuildRaid10)
}

func (adapter *MegaRaidAdaptor) BuildNoneRaid(devs []*baremetal.BaremetalStorage) error {
	return cliBuildRaid(devs, nil, adapter.megacliBuildNoRaid, adapter.storcliBuildNoRaid)
}

func (adapter *MegaRaidAdaptor) storcliIsJBODEnabled() bool {
	cmd := GetCommand2(fmt.Sprintf("/c%d", adapter.index), "show", "jbod")
	lines, err := adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("storcliIsJBODEnabled error: %s", err)
		return false
	}
	for _, line := range lines {
		line = strings.ToLower(line)
		if strings.HasPrefix(line, "jbod") {
			data := strings.Split(line, " ")
			if strings.TrimSpace(data[1]) == "on" {
				return true
			}
			return false
		}
	}
	return false
}

func (adapter *MegaRaidAdaptor) storcliEnableJBOD(enable bool) bool {
	val := "off"
	if enable {
		val = "on"
	}
	cmd := GetCommand2(fmt.Sprintf("/c%d", adapter.index), "set", fmt.Sprintf("jbod=%s", val))
	_, err := adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("EnableJBOD %v fail: %v", enable, err)
		return false
	}
	return true
}

func (adapter *MegaRaidAdaptor) storcliBuildJBOD(devs []*baremetal.BaremetalStorage) error {
	if !adapter.storcliIsJBODEnabled() {
		adapter.storcliEnableJBOD(true)
		adapter.storcliEnableJBOD(false)
		adapter.storcliEnableJBOD(true)
	}
	if !adapter.storcliIsJBODEnabled() {
		return fmt.Errorf("JBOD not supported")
	}
	cmds := []string{}
	for _, d := range devs {
		cmd := GetCommand2(fmt.Sprintf("/c%d/e%d/s%d", adapter.index, d.Enclosure, d.Slot))
		cmds = append(cmds, cmd)
	}
	log.Infof("storcliBuildJBOD cmds: %v", cmds)
	_, err := adapter.remoteRun(cmds...)
	if err != nil {
		return err
	}
	return nil
}

func (adapter *MegaRaidAdaptor) storcliBuildNoRaid(devs []*baremetal.BaremetalStorage, _ *api.BaremetalDiskConfig) error {
	err := adapter.storcliBuildJBOD(devs)
	if err == nil {
		return nil
	}
	log.Errorf("Try build JBOD fail: %v", err)
	labels := []string{}
	for _, dev := range devs {
		labels = append(labels, GetSpecString(dev))
	}
	args := []string{
		fmt.Sprintf("/c%d", adapter.index),
		"add", "vd", "each", "type=raid0",
		fmt.Sprintf("drives=%s", strings.Join(labels, ",")),
		"wt", "nora", "direct",
	}
	cmd := GetCommand2(args...)
	_, err = adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) megacliBuildNoRaid(devs []*baremetal.BaremetalStorage, _ *api.BaremetalDiskConfig) error {
	err := adapter.megacliBuildJBOD(devs)
	if err == nil {
		return nil
	}
	log.Errorf("Try build jbod fail: %v", err)
	cmds := []string{}
	for _, dev := range devs {
		cmd := GetCommand("-CfgLdAdd", "-r0", fmt.Sprintf("[%s]", GetSpecString(dev)),
			"WT", "NORA", "Direct", "NoCachedBadBBU", fmt.Sprintf("-a%d", adapter.index))
		cmds = append(cmds, cmd)
	}
	_, err = adapter.remoteRun(cmds...)
	return err
}

func (adapter *MegaRaidAdaptor) megacliIsJBODEnabled() bool {
	cmd := GetCommand("-AdpGetProp", "-EnableJBOD", fmt.Sprintf("-a%d", adapter.index))
	pref := fmt.Sprintf("Adapter %d: JBOD: ", adapter.index)
	lines, err := adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("megacliIsJBODEnabled error: %v", err)
		return false
	}
	for _, line := range lines {
		if strings.HasPrefix(line, pref) {
			val := strings.ToLower(strings.TrimSpace(line[len(pref):]))
			if val == "disabled" {
				return false
			}
			return true
		}
	}
	return false
}

func (adapter *MegaRaidAdaptor) megacliEnableJBOD(enable bool) bool {
	var val string = "0"
	if enable {
		val = "1"
	}
	cmd := GetCommand("-AdpSetProp", "-EnableJBOD", fmt.Sprintf("-%s", val), fmt.Sprintf("-a%d", adapter.index))
	_, err := adapter.remoteRun(cmd)
	if err != nil {
		log.Errorf("enable jbod %v fail: %v", enable, err)
		return false
	}
	return true
}

func (adapter *MegaRaidAdaptor) megacliBuildJBOD(devs []*baremetal.BaremetalStorage) error {
	if !adapter.megacliIsJBODEnabled() {
		adapter.megacliEnableJBOD(true)
		adapter.megacliEnableJBOD(false)
		adapter.megacliEnableJBOD(true)
	}
	if !adapter.megacliIsJBODEnabled() {
		return fmt.Errorf("JBOD not supported")
	}
	devIds := []string{}
	for _, d := range devs {
		devIds = append(devIds, GetSpecString(d))
	}
	cmd := GetCommand("-PDMakeJBOD", fmt.Sprintf("-PhysDrv[%s]", strings.Join(devIds, ",")), fmt.Sprintf("-a%d", adapter.index))
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) RemoveLogicVolumes() error {
	cmds := []string{}
	lvIdx, err := adapter.GetLogicVolumes()
	if err != nil {
		return err
	}
	for i := len(lvIdx) - 1; i >= 0; i-- {
		cmd := GetCommand("-CfgLdDel", fmt.Sprintf("-L%d", i), "-Force", fmt.Sprintf("-a%d", adapter.index))
		cmds = append(cmds, cmd)
	}
	if len(cmds) > 0 {
		_, err := adapter.remoteRun(cmds...)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

/*
def _storcli_clear_jbod_disks(self):
    cmds = []
    for dev in self.devs:
        cmd = self.raid.get_command2(
                    '/c%d/e%d/s%d' % (self.index, dev.enclosure, dev.slot),
                    'set', 'good', 'force')
        cmds.append(cmd)
    logging.info('%s', cmds)
    self.raid.term.exec_remote_commands(cmds)
*/

func (adapter *MegaRaidAdaptor) megacliClearJBODDisks() error {
	devIds := []string{}
	for _, dev := range adapter.devs {
		devIds = append(devIds, GetSpecString(dev.ToBaremetalStorage()))
	}
	cmd := GetCommand("-PDMakeGood", fmt.Sprintf("-PhysDrv[%s]", strings.Join(devIds, ",")), "-Force", fmt.Sprintf("-a%d", adapter.index))
	_, err := adapter.remoteRun(cmd)
	return err
}

func (adapter *MegaRaidAdaptor) clearJBODDisks() {
	adapter.megacliEnableJBOD(true)
	adapter.megacliEnableJBOD(false)
	adapter.megacliEnableJBOD(true)
	adapter.megacliEnableJBOD(false)
}

type MegaRaid struct {
	term       *ssh.Client
	adapters   []*MegaRaidAdaptor
	PhyDevsCnt int
	Capacity   int64
}

func NewMegaRaid(term *ssh.Client) raiddrivers.IRaidDriver {
	return &MegaRaid{
		term:     term,
		adapters: make([]*MegaRaidAdaptor, 0),
	}
}

func GetCommand(args ...string) string {
	bin := "/opt/MegaRAID/MegaCli/MegaCli64"
	return raiddrivers.GetCommand(bin, args...)
}

func GetCommand2(args ...string) string {
	bin := "/opt/MegaRAID/storcli/storcli64"
	return raiddrivers.GetCommand(bin, args...)
}

func (raid *MegaRaid) GetName() string {
	return baremetal.DISK_DRIVER_MEGARAID
}

func (raid *MegaRaid) ParsePhyDevs() error {
	if !utils.IsInStringArray("megaraid_sas", raiddrivers.GetModules(raid.term)) {
		return fmt.Errorf("Not found megaraid_sas module")
	}
	cmd := GetCommand("-PDList", "-aALL")
	ret, err := raid.term.Run(cmd)
	if err != nil {
		return fmt.Errorf("List raid disk error: %v", err)
	}
	err = raid.parsePhyDevs(ret)
	if err != nil {
		return fmt.Errorf("parse physical disk device error: %v", err)
	}
	return nil
}

func (raid *MegaRaid) parsePhyDevs(lines []string) error {
	phyDev := NewMegaRaidPhyDev()
	var adapter *MegaRaidAdaptor
	for _, line := range lines {
		matches := adapterPatter.FindStringSubmatch(line)
		if len(matches) != 0 {
			paramsMap := make(map[string]string)
			for i, name := range sizePattern.SubexpNames() {
				if i > 0 && i <= len(matches) {
					paramsMap[name] = matches[i]
				}
			}
			adapterStr := paramsMap["idx"]
			adapterInt, _ := strconv.Atoi(adapterStr)
			adapter = NewMegaRaidAdaptor(adapterInt, raid)
			raid.adapters = append(raid.adapters, adapter)
		} else if phyDev.parseLine(line) && phyDev.isComplete() {
			if adapter == nil {
				return fmt.Errorf("Adapter is nil")
			}
			adapter.AddPhyDev(phyDev)
			raid.PhyDevsCnt += 1
			raid.Capacity += phyDev.GetSize()
			phyDev = NewMegaRaidPhyDev()
		}
	}
	for _, dev := range adapter.devs {
		if err := raid.addPhyDevStripSize(dev); err != nil {
			return fmt.Errorf("addPhyDevStripSize: %v", err)
		}
	}
	return nil
}

func (raid *MegaRaid) addPhyDevStripSize(phyDev *MegaRaidPhyDev) error {
	grepCmd := []string{"grep", "-iE", "'^(Min|Max) Strip Size'"}
	args := []string{"-adpallinfo", "-aall", "|"}
	args = append(args, grepCmd...)
	cmd := GetCommand(args...)
	ret, err := raid.term.Run(cmd)
	if err != nil {
		return fmt.Errorf("addPhyDevStripSize error: %v", err)
	}
	return phyDev.parseStripSize(ret)
}

/*func (raid *MegaRaid) GetPhyDevs() []*MegaRaidPhyDev {
	devs := make([]*MegaRaidPhyDev, 0)
	for _, ada := range raid.adapters {
		devs = append(devs, ada.GetPhyDevs()...)
	}
	return devs
}*/

//func (raid *MegaRaid) ParseLogicVolumes() bool {
//cmd := GetCommand("-LDInfo", "-Lall", "-aALL")
//ret, err := raid.term.Run(cmd)
//if err != nil {
//return false
//}
//raid.parseLogicVolumes(ret)
//return true
//}

func (raid *MegaRaid) CleanRaid() error {
	for _, adapter := range raid.adapters {
		adapter.clearJBODDisks()
		adapter.RemoveLogicVolumes()
	}
	return nil
}

func (raid *MegaRaid) PreBuildRaid(_ []*api.BaremetalDiskConfig, _ int) error {
	return raid.clearForeignState()
}

func (raid *MegaRaid) GetAdapters() []raiddrivers.IRaidAdapter {
	ret := make([]raiddrivers.IRaidAdapter, 0)
	for _, a := range raid.adapters {
		ret = append(ret, a)
	}
	return ret
}

func (raid *MegaRaid) clearForeignState() error {
	cmd := GetCommand("-CfgForeign", "-Clear", "-aALL")
	_, err := raid.term.Run(cmd)
	return err
}

func (raid *MegaRaid) RemoveLogicVolumes() {
	for _, adapter := range raid.adapters {
		adapter.RemoveLogicVolumes()
	}
}

func init() {
	raiddrivers.RegisterDriver(baremetal.DISK_DRIVER_MEGARAID, NewMegaRaid)
}
